package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/checklist"
	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/ghx"
)

const defaultJobs = 4

type sweepArgs struct {
	dir           string
	root          string
	template      string
	includeDrafts bool
	jobs          int
}

func sweepFlagSet(a *sweepArgs) *flag.FlagSet {
	fs := flag.NewFlagSet("validate-checklist sweep", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&a.root, "root", ".", "repository the template ref is read from")
	fs.StringVar(&a.template, "template", defaultTemplate,
		"the template `SPEC` to compare against: a git-show object spec, or a file path when it has no colon")
	fs.BoolVar(&a.includeDrafts, "include-drafts", false,
		"audit draft PRs too, as an include-drafts run does; the other holds still stand")
	fs.IntVar(&a.jobs, "jobs", defaultJobs, "how many bodies to fetch at once")
	return fs
}

// parseSweepArgs takes SWEEP_DIR from either END of args, as sweep-prefetch
// does: ahead of the flags, which is how the skills spell it, or after all of
// them. It may NOT sit between two flags — the first non-flag argument stops
// flag parsing, so every flag behind it lands in rest.
func parseSweepArgs(args []string) (sweepArgs, error) {
	var a sweepArgs
	fs := sweepFlagSet(&a)
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		a.dir, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return a, err
	}
	rest := fs.Args()
	if a.dir == "" && len(rest) > 0 {
		a.dir, rest = rest[0], rest[1:]
	}
	if len(rest) > 0 {
		return a, fmt.Errorf("unrecognized arguments: %s", strings.Join(rest, " "))
	}
	if a.dir == "" {
		return a, errors.New("SWEEP_DIR is required")
	}
	if a.jobs < 1 {
		return a, fmt.Errorf("--jobs %d fetches nothing", a.jobs)
	}
	return a, nil
}

func runSweep(args []string) int {
	a, err := parseSweepArgs(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			usage(os.Stdout, sweepFlagSet(&sweepArgs{}))
			return 0
		}
		_, _ = fmt.Fprintf(os.Stderr, "validate-checklist: %v\n", err)
		usage(os.Stderr, sweepFlagSet(&sweepArgs{}))
		return 2
	}

	tmplText, err := readTemplate(a.root, a.template)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "validate-checklist: %v\n", err)
		return 2
	}
	tmpl, err := checklist.Template(tmplText)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "validate-checklist: --template %s: %v\n", a.template, err)
		return 2
	}

	res, err := checklist.Sweep(context.Background(), checklist.SweepOptions{
		SweepDir:      a.dir,
		Template:      tmpl,
		IncludeDrafts: a.includeDrafts,
		Jobs:          a.jobs,
		Fetch:         ghBody{},
		Errs:          os.Stderr,
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "validate-checklist: %v\n", err)
		return 2
	}
	fmt.Println(res)
	return 0
}

// ghBody reads one PR body with the authenticated gh CLI.
type ghBody struct{}

func (ghBody) Body(ctx context.Context, repo string, number int) (string, error) {
	out, err := ghx.Run(ctx, ghx.DefaultTimeout,
		"pr", "view", strconv.Itoa(number), "--repo", repo, "--json", "body")
	if err != nil {
		return "", err
	}
	var doc struct {
		Body string `json:"body"`
	}
	// A decode failure names the syntax, never the payload, so wrapping it
	// cannot put body text on the operator's terminal.
	if err := json.Unmarshal(out, &doc); err != nil {
		return "", fmt.Errorf("decoding the body of %d: %w", number, err)
	}
	return doc.Body, nil
}

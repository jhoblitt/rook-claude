// rt-analyze: Tier-0 KB mining analysis — bucket fetched PRs into areas, flag
// anomalies.
//
//	run.sh rt-analyze --in-dir DIR
//	    (--code-owners /path/to/CODE-OWNERS | --roster a,b,c)
//	    [--out FILE] [--top 15] [--now ISO8601]
//	run.sh rt-analyze areas PATH [PATH...]
//	run.sh rt-analyze areas --stdin
//
// Consumes rt-fetch's rt_prs.jsonl + rt_fetch_state.json from --in-dir and
// writes the {data, flags} miner contract to --out (default
// <in-dir>/rt_final.json). Purely offline. Pin --now to keep the recency
// weighting — and therefore the reviewer ranking — reproducible across re-runs.
//
// The areas subcommand runs the same path-glob classifier for a caller that
// holds changed paths rather than a fetch directory — rook-triage's
// deterministic area layer (references/label-map.md), which every other
// consumer gets pre-stamped in snapshot.json's per-PR areas field. A literal
// "areas" first argument is the only thing that diverts from the analysis run
// above; that one takes flags only, so no existing invocation can reach it.
//
// Exit status is 1 for a bad roster or unreadable input, 2 for a usage error.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtanalyze"
)

func fail(format string, args ...any) int {
	fmt.Fprintf(os.Stderr, "rt-analyze: "+format+"\n", args...)
	return 1
}

func dispatch(args []string, stdin io.Reader, stdout io.Writer) int {
	if len(args) > 0 && args[0] == "areas" {
		return runAreas(args[1:], stdin, stdout)
	}
	return run(args)
}

func run(args []string) int {
	fs := flag.NewFlagSet("rt-analyze", flag.ContinueOnError)
	inDir := fs.String("in-dir", "", "directory holding rt_prs.jsonl and rt_fetch_state.json")
	out := fs.String("out", "", "output file (default <in-dir>/rt_final.json)")
	codeOwners := fs.String("code-owners", "", "path to CODE-OWNERS, mined for the authority roster")
	roster := fs.String("roster", "", "comma-separated logins (alternative to --code-owners)")
	top := fs.Int("top", 15, "reviewers kept per area")
	now := fs.String("now", "", "ISO timestamp for recency weighting (default: current time); pin it for reproducible re-runs")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, "usage: rt-analyze --in-dir DIR "+
			"(--code-owners FILE | --roster a,b,c) [--out FILE] [--top N] [--now ISO]\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *inDir == "" {
		fmt.Fprint(os.Stderr, "rt-analyze: the following arguments are required: --in-dir\n")
		fs.Usage()
		return 2
	}

	var names map[string]bool
	switch {
	case *codeOwners != "":
		f, err := os.Open(*codeOwners)
		if err != nil {
			return fail("%v", err)
		}
		names, err = rtanalyze.ParseCodeOwners(f)
		_ = f.Close()
		if err != nil {
			return fail("%v", err)
		}
		if len(names) == 0 {
			return fail("no approvers/reviewers parsed from %s", *codeOwners)
		}
	case *roster != "":
		names = rtanalyze.ParseRoster(*roster)
	default:
		return fail("pass --code-owners or --roster (identity-unknown flags need it)")
	}

	outPath := *out
	if outPath == "" {
		outPath = filepath.Join(*inDir, "rt_final.json")
	}
	state, err := rtanalyze.LoadState(filepath.Join(*inDir, "rt_fetch_state.json"))
	if err != nil {
		return fail("%v", err)
	}
	at := time.Now().UTC()
	if *now != "" {
		if at, err = rtanalyze.ParseISO(*now); err != nil {
			return fail("--now: %v", err)
		}
	}
	prs, err := rtanalyze.LoadPRs(filepath.Join(*inDir, "rt_prs.jsonl"))
	if err != nil {
		return fail("%v", err)
	}

	result, err := rtanalyze.Analyze(prs, state, rtanalyze.Options{
		OutPath: outPath,
		Top:     *top,
		Now:     at,
		Roster:  rtanalyze.Lowered(names),
	})
	if err != nil {
		return fail("%v", err)
	}
	if err := os.WriteFile(outPath, []byte(rtanalyze.Marshal(result.Doc)), 0o666); err != nil {
		return fail("%v", err)
	}
	fmt.Fprint(os.Stderr, strings.Join(result.Summary, "\n")+"\n")
	return 0
}

func main() {
	os.Exit(dispatch(os.Args[1:], os.Stdin, os.Stdout))
}

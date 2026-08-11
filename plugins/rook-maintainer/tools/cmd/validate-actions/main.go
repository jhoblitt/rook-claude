// validate-actions: rook-triage phase-5 validation of proposed actions against
// live state, before any write (label-set membership, the caps, the
// issues-only label rule, the still-open recheck).
//
//	gh label list --json name > labels.json
//	run.sh validate-actions --actions actions.json --labels labels.json [--items items.json]
//	run.sh validate-actions --self-test
//
// Exit status: 0 every action is safe to execute, 1 at least one is not, 2 bad
// input. Offline by design — it judges the label list and item snapshot it is
// handed rather than fetching its own, so no network failure can make the gate
// pass. A non-zero exit sends those items back to the report instead of to
// GitHub.
//
// Spec: skills/rook-triage/SKILL.md phase 5.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/actions"
)

func usage(fs *flag.FlagSet) {
	_, _ = fmt.Fprint(os.Stderr,
		"usage: validate-actions --actions FILE --labels FILE [--items FILE]\n"+
			"       validate-actions --self-test\n")
	fs.PrintDefaults()
}

func run() int {
	fs := flag.NewFlagSet("validate-actions", flag.ContinueOnError)
	actionsPath := fs.String("actions", "", "proposed actions JSON")
	labelsPath := fs.String("labels", "", "output of gh label list --json name")
	itemsPath := fs.String("items", "", "live per-item state JSON (number, type, state, labels)")
	selfTest := fs.Bool("self-test", false, "verify the checks and exit")
	fs.Usage = func() { usage(fs) }

	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() > 0 {
		usage(fs)
		_, _ = fmt.Fprintf(os.Stderr, "validate-actions: error: unrecognized arguments: %s\n",
			strings.Join(fs.Args(), " "))
		return 2
	}
	if *selfTest {
		fails := actions.SelfTest()
		for _, f := range fails {
			_, _ = fmt.Fprintf(os.Stderr, "self-test: %s\n", f)
		}
		if len(fails) > 0 {
			return 1
		}
		fmt.Println("self-test: OK")
		return 0
	}
	if *actionsPath == "" || *labelsPath == "" {
		usage(fs)
		_, _ = fmt.Fprintln(os.Stderr,
			"validate-actions: error: --actions and --labels are required (or use --self-test)")
		return 2
	}

	payload, err := load(*actionsPath, "actions", actions.Parse)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 2
	}
	live, err := load(*labelsPath, "labels", actions.ParseLabels)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 2
	}
	var items []actions.Item
	if *itemsPath != "" {
		if items, err = load(*itemsPath, "items", actions.ParseItems); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			return 2
		}
	}

	if len(live) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "the live label list is empty — refusing to validate")
		return 2
	}
	return report(os.Stdout, os.Stderr,
		actions.Validate(payload, live, items), len(payload.Entries))
}

func load[T any](path, what string, parse func([]byte) (T, error)) (T, error) {
	var zero T
	data, err := os.ReadFile(path)
	if err != nil {
		return zero, fmt.Errorf("cannot read --%s: %w", what, err)
	}
	v, err := parse(data)
	if err != nil {
		return zero, fmt.Errorf("cannot read --%s: %w", what, err)
	}
	return v, nil
}

func report(out, errOut io.Writer, problems []string, n int) int {
	if len(problems) == 0 {
		_, _ = fmt.Fprintf(out, "all %d proposed action(s) pass the pre-write checks\n", n)
		return 0
	}
	_, _ = fmt.Fprintf(errOut, "%d problem(s) across %d proposed action(s):\n", len(problems), n)
	for _, p := range problems {
		_, _ = fmt.Fprintf(errOut, "  %s\n", p)
	}
	_, _ = fmt.Fprint(errOut, "\nSend these back to the report rather than executing them.\n")
	return 1
}

func main() {
	os.Exit(run())
}

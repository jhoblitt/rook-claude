// gen-pr-dashboard: the rook-triage PR sweep dashboard, and the
// machine-derivable half of its report (format contract:
// skills/rook-triage/references/reporting.md).
//
//	run.sh gen-pr-dashboard SWEEP_DIR [--markdown]
//
// Reads only canonical sweep-dir inputs — snapshot.json, batch-*.json,
// refs-types.json, skips.json — and writes SWEEP_DIR/dashboard.html, or with
// --markdown, SWEEP_DIR/report-tables.md: the per-item tables and the reviewer
// ledger of report.md, as a fragment to concatenate AFTER the model-written
// notes (mdreport.Section states the fragment rule). Both come from the same
// extraction, so a cell cannot say one thing on the dashboard and another in
// the report.
//
// Exit status is 1 on any failure: a sweep that silently produced no
// dashboard, or a stale one, reads to a maintainer as "nothing to act on".
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/prdash"
)

func usage() {
	fmt.Fprint(os.Stderr, "usage: gen-pr-dashboard SWEEP_DIR [--markdown]\n")
}

func run() int {
	fs := flag.NewFlagSet("gen-pr-dashboard", flag.ContinueOnError)
	markdown := fs.Bool("markdown", false,
		"write SWEEP_DIR/report-tables.md (report tables + reviewer ledger) "+
			"instead of the dashboard")
	fs.Usage = usage

	// The documented invocation puts the sweep dir first, which flag parsing
	// would otherwise treat as the end of the flags.
	args := os.Args[1:]
	dir := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		dir, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	rest := fs.Args()
	if dir == "" && len(rest) > 0 {
		dir, rest = rest[0], rest[1:]
	}
	if dir == "" || len(rest) > 0 {
		usage()
		fs.PrintDefaults()
		return 1
	}

	generate := prdash.Generate
	if *markdown {
		generate = prdash.GenerateMarkdown
	}
	if err := generate(dir, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "gen-pr-dashboard: %v\n", err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run())
}

// gen-issues-dashboard: the rook-triage issues sweep dashboard, and the
// machine-derivable half of its report
// (format contract: skills/rook-triage/references/reporting.md).
//
//	run.sh gen-issues-dashboard SWEEP_DIR [--markdown]
//
// Reads ONLY canonical sweep-dir inputs — snapshot.json (live metadata from
// sweep_prefetch), batch-*.json (triager assessments), refs-types.json,
// issues-mentions.json (mine_mentions output) — and writes
// <sweep-dir>/dashboard.html, the same path every time so the sweep's URL
// stays stable, or with --markdown, <sweep-dir>/report-tables.md: the per-item
// table and the mention ledger of report.md, as a fragment to concatenate
// AFTER the model-written notes (mdreport.Section states the fragment rule).
// Both come from the same extraction, so a cell cannot say one thing on the
// dashboard and another in the report.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/issuesdash"
	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/mdreport"
)

func usage() {
	fmt.Fprint(os.Stderr, "usage: gen-issues-dashboard SWEEP_DIR [--markdown]\n")
}

func run() int {
	fs := flag.NewFlagSet("gen-issues-dashboard", flag.ContinueOnError)
	markdown := fs.Bool("markdown", false,
		"write SWEEP_DIR/report-tables.md (report table + mention ledger) "+
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

	if err := generate(dir, *markdown); err != nil {
		fmt.Fprintf(os.Stderr, "gen-issues-dashboard: %v\n", err)
		return 1
	}
	return 0
}

func generate(arg string, markdown bool) error {
	dir, err := filepath.Abs(arg)
	if err != nil {
		return err
	}
	sweep, err := issuesdash.Load(dir)
	if err != nil {
		return err
	}
	page := sweep.Page()

	render, name := issuesdash.Render, "dashboard.html"
	if markdown {
		// An empty table and a sweep dir whose batch files never arrived look
		// identical in the output, and only one of them is a finished sweep.
		if len(sweep.Batches) == 0 {
			return fmt.Errorf("%s: no batch-*.json to report on", dir)
		}
		render, name = issuesdash.RenderMarkdown, mdreport.ReportFile
	}

	// Render before opening the output: a failure must leave the previous
	// artifact in place rather than truncate it.
	var buf bytes.Buffer
	if err := render(&buf, page); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, name), buf.Bytes(), 0o644); err != nil {
		return err
	}
	if markdown {
		return summarize(os.Stdout, filepath.Base(dir), name, page)
	}
	_, err = fmt.Printf("%s: %d rows; titles/labels/assignees from snapshot\n",
		filepath.Base(dir), len(page.Rows))
	return err
}

func summarize(w io.Writer, sweep, name string, page issuesdash.Page) error {
	if missing := page.Missing(); len(missing) > 0 {
		if _, err := fmt.Fprintf(w, "warning: %d assessed issue(s) absent from "+
			"snapshot.json, marked in the summary column: %s\n",
			len(missing), strings.Join(missing, ", ")); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "%s/%s: %d rows; titles/labels/assignees from snapshot\n",
		sweep, name, len(page.Rows))
	return err
}

func main() {
	os.Exit(run())
}

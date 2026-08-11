// gen-issues-dashboard: the rook-triage issues sweep dashboard
// (format contract: skills/rook-triage/references/reporting.md).
//
//	run.sh gen-issues-dashboard SWEEP_DIR
//
// Reads ONLY canonical sweep-dir inputs — snapshot.json (live metadata from
// sweep_prefetch), batch-*.json (triager assessments), refs-types.json,
// issues-mentions.json (mine_mentions output) — and writes
// <sweep-dir>/dashboard.html, the same path every time so the sweep's URL
// stays stable.
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/issuesdash"
)

func run() int {
	if len(os.Args) != 2 {
		fmt.Fprint(os.Stderr, "usage: gen-issues-dashboard SWEEP_DIR\n")
		return 1
	}
	if err := generate(os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "gen-issues-dashboard: %v\n", err)
		return 1
	}
	return 0
}

func generate(arg string) error {
	dir, err := filepath.Abs(arg)
	if err != nil {
		return err
	}
	sweep, err := issuesdash.Load(dir)
	if err != nil {
		return err
	}
	page := sweep.Page()

	// Render before opening the output: a failure must leave the previous
	// dashboard in place rather than truncate it.
	var buf bytes.Buffer
	if err := issuesdash.Render(&buf, page); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "dashboard.html"), buf.Bytes(), 0o644); err != nil {
		return err
	}
	_, err = fmt.Printf("%s: %d rows; titles/labels/assignees from snapshot\n",
		filepath.Base(dir), len(page.Rows))
	return err
}

func main() {
	os.Exit(run())
}

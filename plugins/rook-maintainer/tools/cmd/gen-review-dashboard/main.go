// gen-review-dashboard: the rook-code-review sweep dashboard (sweep.md phase 3).
//
//	run.sh gen-review-dashboard SWEEP_DIR
//
// Reads only the canonical sweep-dir inputs — snapshot.json, sweep.json and
// pr-*/findings.json — and writes <sweep-dir>/dashboard.html, so verdicts and
// finding counts cannot drift from the verified record. Phase 3 regenerates
// the page on every pass, so a missing input is empty rather than fatal;
// anything that would silently shrink the sweep exits 1 instead.
package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/reviewdash"
)

func run() int {
	if len(os.Args) != 2 {
		fmt.Fprint(os.Stderr, "usage: gen-review-dashboard SWEEP_DIR\n")
		return 1
	}
	sweep, err := reviewdash.Load(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen-review-dashboard: %v\n", err)
		return 1
	}
	var page bytes.Buffer
	if err := sweep.Render(&page); err != nil {
		fmt.Fprintf(os.Stderr, "gen-review-dashboard: %v\n", err)
		return 1
	}
	if err := os.WriteFile(sweep.Path(), page.Bytes(), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "gen-review-dashboard: %v\n", err)
		return 1
	}
	fmt.Println(sweep.Summary())
	return 0
}

func main() {
	os.Exit(run())
}

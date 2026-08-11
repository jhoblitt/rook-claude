// gen-pr-dashboard: the rook-triage PR sweep dashboard (format contract:
// skills/rook-triage/references/reporting.md).
//
//	run.sh gen-pr-dashboard SWEEP_DIR
//
// Reads only canonical sweep-dir inputs — snapshot.json, batch-*.json,
// refs-types.json, skips.json — and writes SWEEP_DIR/dashboard.html. Exit
// status is 1 on any failure: a sweep that silently produced no dashboard, or
// a stale one, reads to a maintainer as "nothing to act on".
package main

import (
	"fmt"
	"os"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/prdash"
)

func run() int {
	if len(os.Args) != 2 {
		fmt.Fprint(os.Stderr, "usage: gen-pr-dashboard SWEEP_DIR\n")
		return 1
	}
	if err := prdash.Generate(os.Args[1], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "gen-pr-dashboard: %v\n", err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run())
}

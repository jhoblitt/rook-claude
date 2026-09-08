// gen-run-ledger: the RUN-scoped routing ledger of a rook-triage run — the
// per-person totals the per-person cap is actually measured against
// (skills/rook-triage/references/routing.md, "Selection": 3 items per person
// across every corpus the run touches).
//
//	run.sh gen-run-ledger SWEEP_DIR [SWEEP_DIR] [--kb FILE]
//
// --kb is the routing kb the run selected from: its roster.approvers sizes the
// approver budget, which is printed beside the cap totals and flagged when the
// run routed more PRs than it allows. Without it the budget is left out.
//
// One dir for a prs-only or issues-only run; both dirs for the default `both`
// mode, in either order. Each dir's corpus is read from its snapshot.json
// "kind", and two dirs of the same corpus are refused: that is two runs, not
// one, and summing them would report a breach nobody committed.
//
// Writes run-ledger.md into EVERY dir given — the same combined table in both,
// since the cap spans them — and prints the totals, naming anyone over cap.
// Concatenate it into that dir's report.md after report-tables.md
// (skills/rook-triage/references/reporting.md, "Assembling report.md").
//
// Caller: rook-triage phase 4, which must reconcile a `both` run's two sweep
// dirs against the cap before approving anything. Nothing downstream re-checks
// the cap — validate-actions deliberately does not — so a breach that is not
// in this table is a breach that gets posted.
//
// Exit status is 1 on any failure: an unreadable batch, a dir with no
// batch-*.json, a dir whose corpus cannot be read, or two dirs of one corpus.
// A cap breach is not a failure — the ledger's job is to show it — and exits 0.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/actions"
	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/runledger"
)

func usage() {
	fmt.Fprint(os.Stderr, "usage: gen-run-ledger SWEEP_DIR [SWEEP_DIR] [--kb FILE]\n"+
		"       one dir per corpus of the run (prs, issues), in any order\n")
}

func run() int {
	fs := flag.NewFlagSet("gen-run-ledger", flag.ContinueOnError)
	kb := fs.String("kb", "", "routing kb.json; its roster.approvers sizes the approver budget the run is measured against")
	fs.Usage = usage
	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	dirs := fs.Args()
	if len(dirs) == 0 || len(dirs) > 2 {
		usage()
		return 1
	}
	approvers, err := actions.LoadKBApprovers(*kb)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen-run-ledger: %v\n", err)
		return 1
	}
	if err := runledger.Generate(dirs, len(approvers), os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "gen-run-ledger: %v\n", err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run())
}

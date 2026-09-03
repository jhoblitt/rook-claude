// rt-fetch: merged PRs with their files and reviews, for the rook-triage KB
// refresh (rook-triage/references/kb-refresh.md).
//
//	run.sh rt-fetch --out-dir DIR [--months 24] [--cap 4000] [--repo rook/rook]
//	                [--page-size 50] [--max-pages 400]
//	                [--deep-fetch | --deep-fetch-only]
//
// Writes <out-dir>/rt_prs.jsonl (one counted PR per line) and
// <out-dir>/rt_fetch_state.json (bounds, stop reason, errors, truncation flags
// — the assembler's provenance input), which rt-analyze consumes. See
// internal/rtfetch for the walk's stop conditions and the truncation contract.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtfetch"
)

// checkBounds names the first walk bound that would make the fetch count
// nothing, or returns "" when every bound is usable. Split out so a test can
// reach it without reaching the walk: a bound that gets past here starts a live
// GraphQL fetch, which is not something a unit test may do.
func checkBounds(o rtfetch.Options) (flag, why string) {
	switch {
	case o.Months <= 0:
		return "--months", "the cutoff would land in the future, so no PR is in the window"
	case o.Cap <= 0:
		return "--cap", "the walk stops once counted >= cap, which holds before the first PR"
	case o.PageSize <= 0:
		return "--page-size", "it is the GraphQL page size"
	case o.MaxPages <= 0:
		return "--max-pages", "the walk stops once the page number exceeds it"
	}
	return "", ""
}

func run(args []string) int {
	fs := flag.NewFlagSet("rt-fetch", flag.ContinueOnError)
	fs.Usage = func() {
		_, _ = fmt.Fprint(fs.Output(), "usage: rt-fetch --out-dir DIR [flags]\n")
		fs.PrintDefaults()
	}

	var opts rtfetch.Options
	fs.StringVar(&opts.OutDir, "out-dir", "", "directory to write rt_prs.jsonl and rt_fetch_state.json into")
	fs.Float64Var(&opts.Months, "months", 24, "months of merged-PR history to mine")
	fs.IntVar(&opts.Cap, "cap", 4000, "stop after this many counted PRs")
	fs.StringVar(&opts.Repo, "repo", "rook/rook", "owner/name to mine")
	fs.IntVar(&opts.PageSize, "page-size", 50, "PRs per GraphQL page")
	fs.IntVar(&opts.MaxPages, "max-pages", 400, "safety bound on pages walked")
	fs.BoolVar(&opts.DeepFetch, "deep-fetch", false,
		"after the walk, paginate the truncation-flagged PRs and patch their records")
	fs.BoolVar(&opts.DeepFetchOnly, "deep-fetch-only", false,
		"skip the walk; deep-fetch an existing --out-dir")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if opts.OutDir == "" {
		_, _ = fmt.Fprint(os.Stderr, "rt-fetch: --out-dir is required\n")
		fs.Usage()
		return 2
	}

	// A non-positive bound makes the walk count nothing while still writing the
	// state file the assembler reads as a complete history — the failure this
	// module refuses everywhere else. WindowCutoff deliberately permits a
	// non-positive --months (Python parity) and names the caller as the place
	// to reject it; this is that caller.
	if flag, why := checkBounds(opts); flag != "" {
		_, _ = fmt.Fprintf(os.Stderr, "rt-fetch: %s must be positive: %s\n", flag, why)
		return 2
	}

	if err := rtfetch.New(opts).Run(context.Background()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "rt-fetch: %v\n", err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:]))
}

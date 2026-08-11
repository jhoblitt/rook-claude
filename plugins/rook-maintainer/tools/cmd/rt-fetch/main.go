// rt-fetch: merged PRs with their files and reviews, for the rook-triage KB
// refresh (rook-triage/references/routing.md).
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

func run() int {
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

	if err := fs.Parse(os.Args[1:]); err != nil {
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

	if err := rtfetch.New(opts).Run(context.Background()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "rt-fetch: %v\n", err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run())
}

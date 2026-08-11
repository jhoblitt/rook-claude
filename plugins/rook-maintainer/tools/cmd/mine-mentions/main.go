// mine-mentions: issue-thread @-mentions for the triage report's mentions
// column (reporting.md).
//
//	run.sh mine-mentions SWEEP_DIR [--numbers 1,2,3 | --numbers-file F]
//	                               [--repo rook/rook] [--refetch]
//
// Fetches <sweep-dir>/threads.json (body plus ALL comments per issue) when it
// is missing, then writes <sweep-dir>/issues-mentions.json — {"<number>":
// [logins...]}, first-mention order — and prints a diff against any previous
// version. Resolutions cache in ~/.cache/rook-triage/mentions-user-check.json.
//
// See internal/mentions for why a bare @-scan is not good enough. Resolution
// needs an authenticated gh, so run it with the sandbox disabled.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/mentions"
)

func usage() {
	fmt.Fprint(os.Stderr, "usage: mine-mentions SWEEP_DIR "+
		"[--numbers 1,2,3 | --numbers-file F] [--repo rook/rook] [--refetch]\n")
	flag.PrintDefaults()
}

func run() int {
	fs := flag.NewFlagSet("mine-mentions", flag.ContinueOnError)
	numbers := fs.String("numbers", "", "comma-separated issue numbers (for fetch)")
	numbersFile := fs.String("numbers-file", "", "file with one issue number per line")
	repo := fs.String("repo", "rook/rook", "owner/name to fetch threads from")
	refetch := fs.Bool("refetch", false, "refetch threads.json even when it exists")
	fs.Usage = usage

	// The positional comes first in the documented invocation, which flag
	// parsing would treat as the end of the flags.
	args := os.Args[1:]
	sweepDir, haveDir := "", false
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sweepDir, haveDir, args = args[0], true, args[1:]
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	rest := fs.Args()
	if !haveDir && len(rest) > 0 {
		sweepDir, haveDir, rest = rest[0], true, rest[1:]
	}
	if !haveDir || len(rest) > 0 {
		usage()
		return 2
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "mine-mentions: %v\n", err)
		return 1
	}
	opt := mentions.Options{
		SweepDir:    sweepDir,
		Numbers:     *numbers,
		NumbersFile: *numbersFile,
		Repo:        *repo,
		Refetch:     *refetch,
		CachePath:   filepath.Join(home, ".cache", "rook-triage", "mentions-user-check.json"),
		Lookup:      mentions.GHLogin,
	}
	if err := mentions.Run(context.Background(), opt, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "mine-mentions: %v\n", err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run())
}

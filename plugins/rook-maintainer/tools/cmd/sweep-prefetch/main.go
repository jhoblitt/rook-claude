// sweep-prefetch: phase-0 metadata snapshot for a rook-triage sweep;
// its pipeline phase 0 is the sole caller.
//
//	run.sh sweep-prefetch snapshot SWEEP_DIR --kind prs|issues \
//	    [--numbers 1,2,3 | --numbers-file F] [--repo rook/rook]
//	run.sh sweep-prefetch classify-refs SWEEP_DIR [--repo rook/rook]
//	run.sh sweep-prefetch pool-summary SWEEP_DIR [--sweep FILE] [--viewer LOGIN] \
//	    [--now RFC3339] [--numbers 1,2,3 | --numbers-file F] [--json]
//
// snapshot enumerates every OPEN item of --kind, or exactly --numbers (numbers
// mode also fetches closed and merged items, for regenerating dashboards of
// past sweeps). classify-refs scans <sweep-dir>/batch-*.json for xlink/dup
// numbers and writes refs-types.json.
//
// pool-summary reads that snapshot back and prints the pool-wide counts a sweep
// opens phase 0 with — total, how many are drafts, how many already carry
// --viewer's review, the age/author/label splits, the summed diff — as a
// markdown block to present as-is, or as raw numbers with --json. It is the
// aggregation phase 0 used to do by reading every item into a model's context,
// and it is OFFLINE: it reads local files only, so re-running it costs nothing.
// --sweep adds the fresh-vs-carried split, read from that sweep.json's per-item
// ledger, which is what phase 0's cost estimate divides by: absent, no split is
// reported at all, which is not the same fact as nothing carried. Pool items
// the ledger classifies neither way — the drafts and bots a PR sweep's skip
// rules leave out of its assessable scope — are counted as their own figure
// rather than folded into either side, but a ledger that classifies NOTHING in
// the pool is the wrong file and fails. --numbers narrows the summary to a
// filtered pool and fails on any number the snapshot does not carry; --viewer
// is rejected on an issues snapshot, where no review exists to count, rather
// than reported as zero. Pin --now to keep the age buckets of two runs
// comparable.
//
// Exit status is 2 for a usage error and 1 for a bad argument value, a failed
// fetch or a failed write. snapshot and classify-refs need authenticated gh
// (run with the sandbox disabled); pool-summary needs no network at all.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/sweepprefetch"
)

const defaultRepo = "rook/rook"

// errFlags stands in for a parse failure the flag package has already printed,
// with the usage, on its own.
var errFlags = errors.New("invalid flags")

func usage() {
	fmt.Fprint(os.Stderr, "usage: sweep-prefetch snapshot SWEEP_DIR --kind prs|issues"+
		" [--numbers 1,2,3 | --numbers-file F] [--repo rook/rook]\n"+
		"       sweep-prefetch classify-refs SWEEP_DIR [--repo rook/rook]\n"+
		"       sweep-prefetch pool-summary SWEEP_DIR [--sweep FILE] [--viewer LOGIN]"+
		" [--now RFC3339] [--numbers 1,2,3 | --numbers-file F] [--json]\n")
}

func run() int {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		return 2
	}
	switch args[0] {
	case "snapshot":
		return runSnapshot(args[1:])
	case "classify-refs":
		return runClassifyRefs(args[1:])
	case "pool-summary":
		return runPoolSummary(args[1:])
	default:
		usage()
		return 2
	}
}

func runSnapshot(args []string) int {
	fs := flag.NewFlagSet("sweep-prefetch snapshot", flag.ContinueOnError)
	kind := fs.String("kind", "", "corpus to snapshot: prs or issues")
	numbers := fs.String("numbers", "", "comma-separated item numbers instead of every open item")
	numbersFile := fs.String("numbers-file", "", "file of item numbers, one per line")
	repo := fs.String("repo", defaultRepo, "owner/name of the repository")
	fs.Usage = usage

	sweepDir, err := parse(fs, args)
	if err != nil {
		return fail(err, 2)
	}
	if *kind != "prs" && *kind != "issues" {
		return fail(fmt.Errorf("--kind must be prs or issues"), 2)
	}

	opts := sweepprefetch.SnapshotOptions{SweepDir: sweepDir, Kind: *kind}
	if *numbers != "" || *numbersFile != "" {
		opts.ByNumber = true
		if opts.Numbers, err = itemNumbers(*numbers, *numbersFile); err != nil {
			return fail(err, 1)
		}
	}

	client, err := sweepprefetch.NewClient(*repo)
	if err != nil {
		return fail(err, 1)
	}
	res, err := client.Snapshot(context.Background(), opts)
	if err != nil {
		return fail(err, 1)
	}
	fmt.Println(res)
	return 0
}

func runClassifyRefs(args []string) int {
	fs := flag.NewFlagSet("sweep-prefetch classify-refs", flag.ContinueOnError)
	repo := fs.String("repo", defaultRepo, "owner/name of the repository")
	fs.Usage = usage

	sweepDir, err := parse(fs, args)
	if err != nil {
		return fail(err, 2)
	}
	client, err := sweepprefetch.NewClient(*repo)
	if err != nil {
		return fail(err, 1)
	}
	res, err := client.ClassifyRefs(context.Background(), sweepDir)
	if err != nil {
		return fail(err, 1)
	}
	fmt.Println(res)
	return 0
}

func runPoolSummary(args []string) int {
	fs := flag.NewFlagSet("sweep-prefetch pool-summary", flag.ContinueOnError)
	sweep := fs.String("sweep", "", "sweep.json whose per-item status splits the pool into fresh and carried (omitted entirely when unset)")
	viewer := fs.String("viewer", "", "login whose existing reviews are counted (prs only; omitted entirely when unset)")
	now := fs.String("now", "", "RFC3339 timestamp the age buckets are measured from (default: current time); pin it for reproducible re-runs")
	numbers := fs.String("numbers", "", "comma-separated item numbers to summarize instead of the whole snapshot")
	numbersFile := fs.String("numbers-file", "", "file of item numbers, one per line")
	asJSON := fs.Bool("json", false, "emit the same numbers as JSON instead of the markdown block")
	fs.Usage = usage

	sweepDir, err := parse(fs, args)
	if err != nil {
		return fail(err, 2)
	}
	opts := sweepprefetch.SummaryOptions{
		SweepDir: sweepDir,
		Sweep:    *sweep,
		Viewer:   *viewer,
		Now:      time.Now().UTC(),
	}
	if *now != "" {
		if opts.Now, err = time.Parse(time.RFC3339, *now); err != nil {
			return fail(fmt.Errorf("--now: %w", err), 1)
		}
	}
	if *numbers != "" || *numbersFile != "" {
		if opts.Numbers, err = itemNumbers(*numbers, *numbersFile); err != nil {
			return fail(err, 1)
		}
		// Falling through with an empty list would summarize the whole pool
		// under a heading the caller reads as their filtered one.
		if len(opts.Numbers) == 0 {
			return fail(errors.New("--numbers selected no items"), 1)
		}
	}

	summary, err := sweepprefetch.Summarize(opts)
	if err != nil {
		return fail(err, 1)
	}
	out := summary.Markdown()
	if *asJSON {
		data, err := summary.JSON()
		if err != nil {
			return fail(err, 1)
		}
		out = string(data)
	}
	fmt.Println(out)
	return 0
}

// parse takes SWEEP_DIR from either END of args: ahead of the flags, which is
// how the skills spell it and which flag.Parse would otherwise treat as the end
// of the flags, or after all of them. It may NOT sit between two flags — the
// first non-flag argument stops flag parsing, so every flag behind it lands in
// rest and comes back as "unrecognized arguments".
func parse(fs *flag.FlagSet, args []string) (string, error) {
	sweepDir := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sweepDir, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return "", err
		}
		return "", errFlags
	}
	rest := fs.Args()
	if sweepDir == "" && len(rest) > 0 {
		sweepDir, rest = rest[0], rest[1:]
	}
	if len(rest) > 0 {
		return "", fmt.Errorf("unrecognized arguments: %s", strings.Join(rest, " "))
	}
	if sweepDir == "" {
		return "", errors.New("SWEEP_DIR is required")
	}
	return sweepDir, nil
}

func itemNumbers(list, path string) ([]int, error) {
	var fields []string
	if list != "" {
		fields = strings.Split(list, ",")
	} else {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		fields = strings.Split(string(data), "\n")
	}
	var numbers []int
	seen := map[int]bool{}
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		n, err := strconv.Atoi(f)
		if err != nil {
			return nil, fmt.Errorf("%q is not an item number", f)
		}
		if !seen[n] {
			seen[n] = true
			numbers = append(numbers, n)
		}
	}
	slices.Sort(numbers)
	return numbers, nil
}

func fail(err error, code int) int {
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if !errors.Is(err, errFlags) {
		fmt.Fprintf(os.Stderr, "sweep-prefetch: %v\n", err)
	}
	return code
}

func main() {
	os.Exit(run())
}

// sweep-prefetch: phase-0 metadata snapshot for a rook-triage or
// rook-code-review sweep. Both call it — rook-triage's pipeline phase 0 and
// rook-code-review's sweep phase 0 — so a change here must keep both callers
// working.
//
//	run.sh sweep-prefetch snapshot SWEEP_DIR --kind prs|issues \
//	    [--numbers 1,2,3 | --numbers-file F] [--repo rook/rook]
//	run.sh sweep-prefetch classify-refs SWEEP_DIR [--repo rook/rook]
//
// snapshot enumerates every OPEN item of --kind, or exactly --numbers (numbers
// mode also fetches closed and merged items, for regenerating dashboards of
// past sweeps). classify-refs scans <sweep-dir>/batch-*.json for xlink/dup
// numbers and writes refs-types.json.
//
// Exit status is 2 for a usage error and 1 for a bad argument value, a failed
// fetch or a failed write. Needs authenticated gh (run with the sandbox
// disabled).
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

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/sweepprefetch"
)

const defaultRepo = "rook/rook"

// errFlags stands in for a parse failure the flag package has already printed,
// with the usage, on its own.
var errFlags = errors.New("invalid flags")

func usage() {
	fmt.Fprint(os.Stderr, "usage: sweep-prefetch snapshot SWEEP_DIR --kind prs|issues"+
		" [--numbers 1,2,3 | --numbers-file F] [--repo rook/rook]\n"+
		"       sweep-prefetch classify-refs SWEEP_DIR [--repo rook/rook]\n")
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

// parse pulls SWEEP_DIR out of args wherever it sits. The skills pass it ahead
// of the flags, which flag.Parse treats as the end of the flags, so the
// leading form has to come off by hand first.
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

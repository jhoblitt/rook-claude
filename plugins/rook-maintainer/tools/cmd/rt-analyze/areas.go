package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtanalyze"
)

// runAreas answers "which areas does this change touch" for one changed-path
// SET: every PATH is classified together and the union is printed, one area
// per line, in taxonomy order — the same answer, byte for byte, that a sweep
// stamps into snapshot.json's per-PR areas field. It is deliberately not a
// per-path report: rook-triage routes an item, never a file.
//
// A path set that matches no area prints nothing and exits 0. That is the
// taxonomy's answer, not a failure — the "Deliberately unbucketed" paragraph
// of skills/rook-triage/references/label-map.md names the classes that match
// nothing on purpose.
func runAreas(args []string, stdin io.Reader, stdout io.Writer) int {
	fs := flag.NewFlagSet("rt-analyze areas", flag.ContinueOnError)
	fromStdin := fs.Bool("stdin", false, "read the paths from stdin, one per line, instead of from PATH arguments")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, "usage: rt-analyze areas PATH [PATH...]\n"+
			"       rt-analyze areas --stdin\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	paths := fs.Args()
	// Parsing stops at the first PATH, so a flag written after one arrives
	// here as a path and would otherwise be classified as a file name.
	for _, p := range paths {
		if strings.HasPrefix(p, "-") {
			fmt.Fprintf(os.Stderr, "rt-analyze areas: flag %q must come before the PATH arguments\n", p)
			fs.Usage()
			return 2
		}
	}
	switch {
	case *fromStdin && len(paths) > 0:
		fmt.Fprint(os.Stderr, "rt-analyze areas: --stdin takes no PATH arguments\n")
		fs.Usage()
		return 2
	case !*fromStdin && len(paths) == 0:
		fmt.Fprint(os.Stderr, "rt-analyze areas: the following arguments are required: PATH (or --stdin)\n")
		fs.Usage()
		return 2
	}

	if *fromStdin {
		sc := bufio.NewScanner(stdin)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			if p := strings.TrimSpace(sc.Text()); p != "" {
				paths = append(paths, p)
			}
		}
		if err := sc.Err(); err != nil {
			return fail("reading paths from stdin: %v", err)
		}
	}

	for _, area := range rtanalyze.AreasForPaths(paths) {
		if _, err := fmt.Fprintln(stdout, area); err != nil {
			return fail("writing areas: %v", err)
		}
	}
	return 0
}

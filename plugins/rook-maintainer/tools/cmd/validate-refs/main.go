// validate-refs: the make targets and repo-relative paths a diff's added
// lines point at, resolved against the branch being changed. What it reads
// and does not is stated once, in the spec below, not promised here.
//
//	git diff origin/release-1.20... | run.sh validate-refs --root .
//	run.sh validate-refs --diff-file F [--root DIR] [--json]
//	run.sh validate-refs --self-test
//
// Exit status: 0 nothing is provably absent, 1 at least one reference is
// missing, 2 bad input. Offline by design — it resolves against the working
// tree it is pointed at, so no network failure can make the gate pass.
//
// The gap it closes is drift between prose and tooling, which every other
// check is blind to: a backported doc that says `make lint.markdown` on a
// branch whose target is still `markdownlint` is valid markdown, passes
// markdownlint and commitlint, and fails only for the reader. Resolution is a
// static parse rather than `make --print-data-base`, because running a
// makefile is not hermetic — rook's shells out at parse time and downloads
// tooling, which would report a missing container runtime as documentation
// drift.
//
// Spec: skills/rook-code-review/references/docs-sync.md, the validate-refs
// bullet. Callers: skills/rook-code-review/SKILL.md,
// skills/rook-conventions/references/backporting.md.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/refs"
)

func usage(fs *flag.FlagSet) {
	_, _ = fmt.Fprint(os.Stderr,
		"usage: validate-refs [--diff-file FILE] [--root DIR] [--json]\n"+
			"       validate-refs --self-test\n")
	fs.PrintDefaults()
}

func run() int {
	fs := flag.NewFlagSet("validate-refs", flag.ContinueOnError)
	diffFile := fs.String("diff-file", "", "read a unified diff from this file instead of stdin")
	root := fs.String("root", ".", "tree to resolve references against")
	asJSON := fs.Bool("json", false, "emit JSON")
	selfTest := fs.Bool("self-test", false, "verify the checks and exit")
	fs.Usage = func() { usage(fs) }

	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() > 0 {
		usage(fs)
		_, _ = fmt.Fprintf(os.Stderr, "validate-refs: error: unrecognized arguments: %s\n",
			strings.Join(fs.Args(), " "))
		return 2
	}
	if *selfTest {
		fails := refs.SelfTest()
		for _, f := range fails {
			_, _ = fmt.Fprintf(os.Stderr, "self-test: %s\n", f)
		}
		if len(fails) > 0 {
			return 1
		}
		fmt.Println("self-test: OK")
		return 0
	}

	diff, err := readDiff(*diffFile)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "validate-refs: %v\n", err)
		return 2
	}
	if st, err := os.Stat(*root); err != nil || !st.IsDir() {
		_, _ = fmt.Fprintf(os.Stderr, "validate-refs: --root %q is not a directory\n", *root)
		return 2
	}

	found := refs.Extract(diff)
	if len(found) == 0 {
		return 0
	}
	results := refs.Resolve(*root, found)
	if err := report(os.Stdout, os.Stderr, results, *asJSON); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "validate-refs: %v\n", err)
		return 2
	}
	for _, r := range results {
		if r.Bad() {
			return 1
		}
	}
	return 0
}

func readDiff(path string) (string, error) {
	if path == "" {
		b, err := io.ReadAll(os.Stdin)
		return string(b), err
	}
	b, err := os.ReadFile(path) // #nosec G304 -- the caller names the diff to read
	return string(b), err
}

func report(out, errOut io.Writer, results []refs.Result, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	unresolvable := 0
	for _, r := range results {
		if r.Verdict == refs.VerdictUnresolvable {
			unresolvable++
		}
		where := r.File
		if where != "" && r.Line > 0 {
			where = fmt.Sprintf("%s:%d", r.File, r.Line)
		}
		if _, err := fmt.Fprintf(out, "%-13s %-11s %-40s %s\n",
			strings.ToUpper(string(r.Verdict)), r.Kind, r.Name, where); err != nil {
			return err
		}
		if r.Note != "" {
			if _, err := fmt.Fprintf(out, "%s(%s)\n", strings.Repeat(" ", 26), r.Note); err != nil {
				return err
			}
		}
	}

	// Say what was not covered. A gate that reports only what it proved,
	// without saying what it could not read, reads as full coverage.
	if unresolvable > 0 {
		_, _ = fmt.Fprintf(errOut,
			"\n%d reference(s) could not be statically resolved; check these by hand.\n",
			unresolvable)
	}
	return nil
}

func main() {
	os.Exit(run())
}

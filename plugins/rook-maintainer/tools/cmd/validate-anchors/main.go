// validate-anchors: check a review payload's inline anchors against the PR
// diff before it is posted (posting.md step 2).
//
//	gh pr diff <n> > pr.diff
//	run.sh validate-anchors --review review.json --diff pr.diff
//
//	gh pr diff <n> | run.sh validate-anchors --review review.json
//
//	run.sh validate-anchors --self-test
//
// Exit status: 0 every anchor is postable, 1 at least one is not, 2 bad input.
// One unpostable anchor makes GitHub reject the whole review call, so this
// runs BEFORE the POST and nothing about it needs a network.
//
// Spec: skills/rook-code-review/references/posting.md.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/anchors"
)

func usage(w io.Writer, fs *flag.FlagSet) {
	_, _ = fmt.Fprint(w, "usage: validate-anchors --review FILE [--diff FILE]\n"+
		"       validate-anchors --self-test\n")
	fs.SetOutput(w)
	fs.PrintDefaults()
}

func run() int {
	fs := flag.NewFlagSet("validate-anchors", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	reviewPath := fs.String("review", "", "review payload JSON (the gh --input file)")
	diffPath := fs.String("diff", "", "unified diff; omit to read stdin")
	selfTest := fs.Bool("self-test", false, "verify the parser and exit")

	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			usage(os.Stdout, fs)
			return 0
		}
		_, _ = fmt.Fprintf(os.Stderr, "validate-anchors: %v\n", err)
		usage(os.Stderr, fs)
		return 2
	}
	if fs.NArg() > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "validate-anchors: unrecognized arguments: %s\n",
			strings.Join(fs.Args(), " "))
		usage(os.Stderr, fs)
		return 2
	}

	if *selfTest {
		if err := anchors.SelfTest(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "self-test: FAILED: %v\n", err)
			return 1
		}
		fmt.Println("self-test: OK")
		return 0
	}
	if *reviewPath == "" {
		_, _ = fmt.Fprint(os.Stderr, "validate-anchors: --review is required (or use --self-test)\n")
		usage(os.Stderr, fs)
		return 2
	}

	data, err := os.ReadFile(*reviewPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "cannot read --review: %v\n", err)
		return 2
	}
	review, err := anchors.ParseReview(data)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "cannot read --review: %v\n", err)
		return 2
	}

	diff, err := readDiff(*diffPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		return 2
	}
	if strings.TrimSpace(diff) == "" {
		_, _ = fmt.Fprint(os.Stderr, "the diff is empty \u2014 nothing can be anchored\n")
		return 2
	}

	files := anchors.Commentable(diff)
	problems := anchors.Validate(review, files)
	n := anchors.Count(review)

	if len(problems) > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "%d of %d inline anchor(s) cannot be posted:\n", len(problems), n)
		for _, problem := range problems {
			_, _ = fmt.Fprintf(os.Stderr, "  %s\n", problem)
		}
		_, _ = fmt.Fprint(os.Stderr, "\nFold each into the review BODY under \"Other observations\" "+
			"and say there that it is unanchored (posting.md).\n")
		return 1
	}

	fmt.Printf("all %d inline anchor(s) land inside the diff\n", n)
	return 0
}

func readDiff(path string) (string, error) {
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("cannot read --diff: %v", err)
		}
		return string(b), nil
	}
	if info, err := os.Stdin.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
		return "", errors.New("no diff: pass --diff FILE or pipe `gh pr diff <n>`")
	}
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("cannot read --diff: %v", err)
	}
	return string(b), nil
}

func main() {
	os.Exit(run())
}

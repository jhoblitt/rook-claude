// rt-commits: per-area commit authorship for the rook-triage kb refresh
// (rook-triage/references/kb-refresh.md, the "git log per area path-set" signal).
//
//	run.sh rt-commits --repo PATH [--ref REV] [--months 24] [--now RFC3339] [--out FILE | --json]
//	run.sh rt-commits --log FILE            [--months 24] [--now RFC3339] [--out FILE | --json]
//
// --repo runs the git log itself over that checkout, read-only, mining --ref
// (default origin/master — kb-refresh.md's signal, and not what a bare git log
// walks). --log consumes a dump of exactly that command instead, captured
// elsewhere:
//
//	git -c core.quotePath=false log --no-merges -M \
//	    --format=commit%x09%H%x09%aN%x09%aE%x09%aI --name-status \
//	    --end-of-options origin/master > commits.log
//
// The per-area summary always goes to stdout. --out and --json are the two
// destinations for the JSON document the kb assembler consumes, so at most one
// of them may be given: --out FILE writes it there — the only file this tool
// writes; the mined checkout is read-only — and --json replaces the summary
// with it on stdout. Prefer --out when an agent is reading: on rook/rook the
// document runs ~50x the summary's size, context the agent pays for and cannot
// skim. Pin --now to keep the recency weighting — and therefore the ranking —
// reproducible across re-runs.
//
// Exit status is 2 for a usage error, 1 for a bad value, an unreadable input, an
// unwritable --out or a git log that fails. It never answers with a plausible
// empty document.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtanalyze"
	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtcommits"
)

func fail(format string, args ...any) int {
	fmt.Fprintf(os.Stderr, "rt-commits: "+format+"\n", args...)
	return 1
}

func run() int {
	fs := flag.NewFlagSet("rt-commits", flag.ContinueOnError)
	repo := fs.String("repo", "", "path to a rook checkout to mine with git log")
	logFile := fs.String("log", "", "path to a captured git log dump (see -h for the command)")
	ref := fs.String("ref", rtcommits.DefaultRef, "revision to mine under --repo")
	months := fs.Float64("months", 24, "months of history to mine")
	now := fs.String("now", "", "RFC3339 timestamp for recency weighting (default: current time); pin it for reproducible re-runs")
	out := fs.String("out", "", "write the JSON document to FILE, keeping the summary on stdout")
	asJSON := fs.Bool("json", false, "emit the JSON document on stdout instead of the summary")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, "usage: rt-commits (--repo PATH [--ref REV] | --log FILE) [--months N] [--now RFC3339] "+
			"[--out FILE | --json]\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n--log consumes the output of:\n  %s\n", rtcommits.GitLogCommand(rtcommits.DefaultRef))
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if (*repo == "") == (*logFile == "") {
		fmt.Fprint(os.Stderr, "rt-commits: pass exactly one of --repo or --log\n")
		fs.Usage()
		return 2
	}
	if *logFile != "" && passed(fs, "ref") {
		fmt.Fprint(os.Stderr, "rt-commits: --ref applies to --repo only; a dump's revision "+
			"was chosen when it was captured\n")
		fs.Usage()
		return 2
	}
	if *asJSON && *out != "" {
		fmt.Fprint(os.Stderr, "rt-commits: pass at most one of --out or --json; "+
			"--out already writes the JSON document\n")
		fs.Usage()
		return 2
	}

	at := time.Now().UTC()
	if *now != "" {
		var err error
		if at, err = rtanalyze.ParseISO(*now); err != nil {
			return fail("--now: %v", err)
		}
	}

	var (
		commits []rtcommits.Commit
		source  rtcommits.Source
		err     error
	)
	if *repo != "" {
		source = rtcommits.Source{Mode: "repo", Path: *repo, Ref: *ref}
		commits, source.Head, err = rtcommits.Log(context.Background(), *repo, *ref)
	} else {
		source = rtcommits.Source{Mode: "log", Path: *logFile}
		commits, err = loadDump(*logFile)
	}
	if err != nil {
		return fail("%v", err)
	}

	result, err := rtcommits.Mine(commits, rtcommits.Options{Now: at, Months: *months, Source: source})
	if err != nil {
		return fail("%v", err)
	}

	if *asJSON || *out != "" {
		doc, err := rtcommits.Render(result.Doc)
		if err != nil {
			return fail("%v", err)
		}
		if *out == "" {
			fmt.Print(string(doc))
			return 0
		}
		if err := os.WriteFile(*out, doc, 0o666); err != nil {
			return fail("%v", err)
		}
		fmt.Printf("wrote %s (%d bytes)\n", *out, len(doc))
	}
	fmt.Println(strings.Join(result.Summary, "\n"))
	return 0
}

func passed(fs *flag.FlagSet, name string) bool {
	seen := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			seen = true
		}
	})
	return seen
}

func loadDump(path string) ([]rtcommits.Commit, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	commits, err := rtcommits.ParseLog(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return commits, nil
}

func main() {
	os.Exit(run())
}

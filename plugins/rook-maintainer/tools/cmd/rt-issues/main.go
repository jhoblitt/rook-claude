// rt-issues: issue participation per area for the rook-triage kb refresh
// (rook-triage/references/kb-refresh.md, the "Issue participation per area
// label" signal).
//
//	run.sh rt-issues --in FILE --label-map FILE --out FILE
//	    [--months 24] [--now ISO] [--top 15] [--code-owners FILE | --roster a,b,c] [--brief FILE]
//
// --in consumes the JSON array of the gh issue list export that -h prints;
// internal/rtissues owns that command line as ExportCommand.
//
// --label-map is references/label-map.md, whose area table decides which area
// each issue's labels put it in. Pin --now to keep the window — and therefore
// the counts — reproducible across re-runs.
//
// --out takes the document the kb assembler consumes; the one-line run summary
// stays on stdout. The flags reach the resolver as the fenced block --brief
// FILE writes, and a run given no --brief writes none. A roster
// (--code-owners, or --roster for a caller that has no file) is what makes an
// unknown identity answerable, so without one no identity-unknown flag is
// raised.
//
// Exit status is 2 for a usage error and 1 for a bad value, an unreadable
// input or an unwritable output.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/actions"
	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtanalyze"
	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtissues"
)

func fail(format string, args ...any) int {
	fmt.Fprintf(os.Stderr, "rt-issues: "+format+"\n", args...)
	return 1
}

func run(args []string) int {
	fs := flag.NewFlagSet("rt-issues", flag.ContinueOnError)
	in := fs.String("in", "", "gh issue list JSON export (see -h for the command)")
	labelMap := fs.String("label-map", "", "references/label-map.md; its area table buckets the issues")
	out := fs.String("out", "", "write the JSON document to FILE")
	brief := fs.String("brief", "", "write the resolver's fenced flag brief to FILE (default: no brief)")
	codeOwners := fs.String("code-owners", "", "path to CODE-OWNERS, mined for the authority roster")
	roster := fs.String("roster", "", "comma-separated logins (alternative to --code-owners)")
	months := fs.Int("months", 24, "months of comments to count")
	top := fs.Int("top", 15, "logins kept per area")
	now := fs.String("now", "", "ISO timestamp the window ends at (default: current time); pin it for reproducible re-runs")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, "usage: rt-issues --in FILE --label-map FILE --out FILE "+
			"[--months N] [--now ISO] [--top N] [--code-owners FILE | --roster a,b,c] [--brief FILE]\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n--in consumes the output of:\n  %s\n", rtissues.ExportCommand)
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	for _, required := range []struct {
		name  string
		value string
	}{{"in", *in}, {"label-map", *labelMap}, {"out", *out}} {
		if required.value == "" {
			fmt.Fprintf(os.Stderr, "rt-issues: the following arguments are required: --%s\n", required.name)
			fs.Usage()
			return 2
		}
	}
	if *codeOwners != "" && *roster != "" {
		fmt.Fprint(os.Stderr, "rt-issues: pass at most one of --code-owners or --roster\n")
		fs.Usage()
		return 2
	}

	names, _, err := rtanalyze.LoadRoster(*codeOwners, *roster)
	if err != nil {
		return fail("%v", err)
	}
	at := time.Now().UTC()
	if *now != "" {
		if at, err = rtanalyze.ParseISO(*now); err != nil {
			return fail("--now: %v", err)
		}
	}
	md, err := os.ReadFile(*labelMap)
	if err != nil {
		return fail("%v", err)
	}
	byLabel, err := actions.ParseLabelAreas(md)
	if err != nil {
		return fail("%s: %v", *labelMap, err)
	}
	issues, err := rtissues.Load(*in)
	if err != nil {
		return fail("%v", err)
	}

	result, err := rtissues.Mine(issues, rtissues.Options{
		Now:     at,
		Months:  *months,
		Areas:   byLabel,
		Roster:  names,
		Top:     *top,
		OutPath: *out,
	})
	if err != nil {
		return fail("%v", err)
	}
	if err := os.WriteFile(*out, []byte(rtanalyze.Marshal(result.Doc)), 0o666); err != nil {
		return fail("%v", err)
	}
	if *brief != "" {
		if err := os.WriteFile(*brief, []byte(rtissues.FlagBrief(result.Flags)), 0o666); err != nil {
			return fail("%v", err)
		}
	}
	fmt.Println(result.Summary)
	return 0
}

func main() {
	os.Exit(run(os.Args[1:]))
}

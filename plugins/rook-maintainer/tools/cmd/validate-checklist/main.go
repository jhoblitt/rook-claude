// validate-checklist: does a PR body reproduce the repository's pull-request
// template checklist, item for item?
//
//	gh pr view <n> --json body -q .body | run.sh validate-checklist --root ~/github/rook
//	run.sh validate-checklist --body FILE [--root DIR] [--template SPEC] [--json]
//	run.sh validate-checklist sweep SWEEP_DIR [--root DIR] [--template SPEC]
//	    [--include-drafts] [--jobs N]
//	run.sh validate-checklist --self-test
//
// The flag form audits ONE body and reports it in full. Exit status: 0
// conforming, 1 non-conforming, 2 bad input. Offline by design — the template
// comes from a git object or a file, so no network failure can make the gate
// pass.
//
// sweep audits a whole triage sweep against that same template, reading each
// body with authenticated gh — the tool's only network path — and writes
// SWEEP_DIR/checklist.jsonl: one compact {"number":N,"verdict":"..."} row per
// audited PR and nothing else. The full report's lines[] reproduce PR-body
// text verbatim, and this file is read by triagers with no jq to project them
// back out, so no body text may reach it — or stdout, or the log.
//
// It writes SWEEP_DIR/skips.json in the same pass: one
// {"number","class","author","title"} row per PR it left out, ascending by
// number, which is the dashboard's skipped section (rook-triage's
// references/reporting.md). Every skipped PR gets a row, and the file is
// written even when there are none — a reader takes a missing one for a pool
// that skipped nothing, so an omission would read as coverage. A carried card
// gets no row: not re-auditing it is this pass's own business, and the run
// that assessed it publishes the assessment. Titles are contributor text,
// which the dashboard escapes; this is the one file the pass writes that
// carries any.
//
// It audits SWEEP_DIR/snapshot.json's pool minus what pr-triage.md's "Skip
// conditions" leave out — that list is the definition and is restated nowhere
// here; --include-drafts widens the pass to drafts and to nothing else —
// minus whatever SWEEP_DIR/sweep.json's per-item ledger records as carried,
// those cards being assessed already. A sweep dir with no ledger is a run that
// has assessed nothing, so everything else is audited. Rows land in completion
// order, --jobs (default 4) bodies in flight at once; consumers key by number.
//
// A PR whose body never arrived is "unaudited" — a verdict the audit itself
// never returns, and NOT "no-checklist": a body that arrives empty genuinely
// has no checklist, and a fetch that failed must never read as a PR that has
// none. Exit status: 0 once the pass wrote both files, whatever the verdicts
// in them, since non-conforming and unaudited rows are the findings it exists
// to collect; 2 when it could not run at all — usage, an unreadable or non-prs
// snapshot, an unwritable skips.json or checklist.jsonl, including a write
// that fails only as the file is closed. A write that fails partway stops the
// pass, bodies it could not record being bodies not worth fetching. 1 is the
// single-PR form's non-conforming and never a sweep's.
//
// Callers: rook-code-review's docs-sync pass (one PR), and rook-triage's
// phase 0, which sweeps its pool (references/pr-triage.md).
// Spec: skills/rook-code-review/references/docs-sync.md, "PR-template
// checklist audit"; the sweep's skip conditions are
// skills/rook-triage/references/pr-triage.md.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/checklist"
)

// defaultTemplate mirrors the ref the prose names — the spec's "PR-template
// checklist audit" and rook-code-review SKILL.md's Scripts entry — and both
// documented call sites rely on it (--template is the fallback for a machine
// with no rook checkout), so this line, not the prose, is what every audit
// actually compares against. Change one, change the other.
const defaultTemplate = "origin/master:.github/PULL_REQUEST_TEMPLATE.md"

func usage(w io.Writer, fs *flag.FlagSet) {
	_, _ = fmt.Fprint(w, "usage: validate-checklist [--body FILE] [--root DIR] [--template SPEC] [--json]\n"+
		"       validate-checklist sweep SWEEP_DIR [--root DIR] [--template SPEC] [--include-drafts] [--jobs N]\n"+
		"       validate-checklist --self-test\n")
	fs.SetOutput(w)
	fs.PrintDefaults()
}

func run(args []string) int {
	if len(args) > 0 && args[0] == "sweep" {
		return runSweep(args[1:])
	}

	fs := flag.NewFlagSet("validate-checklist", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	bodyPath := fs.String("body", "", "PR body; omit to read stdin")
	root := fs.String("root", ".", "repository the template ref is read from")
	tmplSpec := fs.String("template", defaultTemplate,
		"the template `SPEC` to compare against: a git-show object spec, or a file path when it has no colon")
	asJSON := fs.Bool("json", false, "emit JSON")
	selfTest := fs.Bool("self-test", false, "verify the checks and exit")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			usage(os.Stdout, fs)
			return 0
		}
		_, _ = fmt.Fprintf(os.Stderr, "validate-checklist: %v\n", err)
		usage(os.Stderr, fs)
		return 2
	}
	if fs.NArg() > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "validate-checklist: unrecognized arguments: %s\n",
			strings.Join(fs.Args(), " "))
		usage(os.Stderr, fs)
		return 2
	}

	if *selfTest {
		fails := checklist.SelfTest()
		for _, f := range fails {
			_, _ = fmt.Fprintf(os.Stderr, "self-test: %s\n", f)
		}
		if len(fails) > 0 {
			return 1
		}
		fmt.Println("self-test: OK")
		return 0
	}

	tmplText, err := readTemplate(*root, *tmplSpec)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "validate-checklist: %v\n", err)
		return 2
	}
	tmpl, err := checklist.Template(tmplText)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "validate-checklist: --template %s: %v\n", *tmplSpec, err)
		return 2
	}
	body, err := readBody(*bodyPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "validate-checklist: %v\n", err)
		return 2
	}

	rep := checklist.Audit(tmpl, body)
	if err := report(os.Stdout, os.Stderr, rep, *tmplSpec, *asJSON); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "validate-checklist: %v\n", err)
		return 2
	}
	if rep.Bad() {
		return 1
	}
	return 0
}

// readTemplate reads the canonical checklist. A spec with a colon is a git
// object spec, resolved with argv and no shell; anything else is a path, for
// running against a checkout-free sweep.
func readTemplate(root, spec string) (string, error) {
	if spec == "" {
		return "", errors.New("--template is empty")
	}
	if strings.HasPrefix(spec, "-") {
		return "", fmt.Errorf("--template %q reads as an option", spec)
	}
	if !strings.Contains(spec, ":") {
		b, err := os.ReadFile(spec) // #nosec G304 -- the caller names the template to read
		if err != nil {
			return "", fmt.Errorf("cannot read --template: %v", err)
		}
		return string(b), nil
	}

	cmd := exec.Command("git", "-C", root, "show", spec) // #nosec G204 -- argv, never a shell
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		detail, _, _ := strings.Cut(strings.TrimSpace(stderr.String()), "\n")
		if detail == "" {
			detail = err.Error()
		}
		return "", fmt.Errorf("cannot read --template %s from %s: %s\n"+
			"fetch the ref (git -C %s fetch origin master) or pass --template FILE", spec, root, detail, root)
	}
	return string(out), nil
}

func readBody(path string) (string, error) {
	r := io.Reader(os.Stdin)
	if path != "" {
		f, err := os.Open(path) // #nosec G304 -- the caller names the body to read
		if err != nil {
			return "", fmt.Errorf("cannot read --body: %v", err)
		}
		defer func() { _ = f.Close() }()
		r = f
	} else if info, err := os.Stdin.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
		return "", errors.New("no body: pass --body FILE or pipe `gh pr view <n> --json body -q .body`")
	}

	b, err := io.ReadAll(io.LimitReader(r, checklist.MaxBody+1))
	if err != nil {
		return "", fmt.Errorf("cannot read --body: %v", err)
	}
	if len(b) > checklist.MaxBody {
		return "", fmt.Errorf("the body is over %d bytes; GitHub caps a PR body at 65536 characters", checklist.MaxBody)
	}
	return string(b), nil
}

func report(out, errOut io.Writer, rep checklist.Report, spec string, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(rep)
	}

	for _, l := range rep.Lines {
		if _, err := fmt.Fprintf(out, "%-8s %-3s %s\n",
			strings.ToUpper(string(l.Status)), glyph(l.State), l.Text); err != nil {
			return err
		}
		if l.Status == checklist.StatusAltered {
			if _, err := fmt.Fprintf(out, "%8s %-3s %s\n", "want", glyph(l.WantState), l.Want); err != nil {
				return err
			}
		}
	}
	for _, p := range rep.Problems {
		if _, err := fmt.Fprintf(out, "%-8s %s\n", "PROBLEM", p); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(out, "\n%s against %s: %d verbatim, %d altered, %d missing, %d unexpected\n",
		strings.ToUpper(string(rep.Verdict)), spec,
		rep.Count(checklist.StatusOK), rep.Count(checklist.StatusAltered),
		rep.Count(checklist.StatusMissing), rep.Count(checklist.StatusExtra)); err != nil {
		return err
	}

	// Say what was not covered. Half of the audit is not decidable here, and a
	// gate that reports only what it proved reads as full coverage.
	_, _ = fmt.Fprint(errOut, "\nThe check STATE above is reported, not judged: whether a ticked box "+
		"is matched by the diff is the reviewer's call (docs-sync.md).\n")
	return nil
}

func glyph(s checklist.State) string {
	switch s {
	case checklist.StateChecked:
		return "[x]"
	case checklist.StateUnchecked:
		return "[ ]"
	case checklist.StateNone:
		return "-"
	}
	return ""
}

func main() {
	os.Exit(run(os.Args[1:]))
}

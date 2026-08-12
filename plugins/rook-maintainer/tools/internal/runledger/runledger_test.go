package runledger

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/mdreport"
)

var update = flag.Bool("update", false, "rewrite the golden fragments")

// The fixtures are built around the failure this ledger exists to remove:
// subhamkrai is proposed on two PRs and two issues, so they are UNDER the cap
// in each dir's own ledger and over it across the run.
const (
	prsDir    = "testdata/2026-08-12-prs-run"
	issuesDir = "testdata/2026-08-12-issues-run"
)

func render(t *testing.T, dirs ...string) string {
	t.Helper()
	run, err := Load(dirs...)
	if err != nil {
		t.Fatalf("Load(%v) = %v", dirs, err)
	}
	var buf bytes.Buffer
	if err := run.Render(&buf); err != nil {
		t.Fatalf("Render() = %v", err)
	}
	return buf.String()
}

func TestGolden(t *testing.T) {
	tests := []struct {
		name   string
		dirs   []string
		golden string
	}{
		{"both corpora", []string{prsDir, issuesDir}, "testdata/run.golden.md"},
		{"prs only", []string{prsDir}, "testdata/prs-only.golden.md"},
		{"issues only", []string{issuesDir}, "testdata/issues-only.golden.md"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := []byte(render(t, tc.dirs...))
			if *update {
				if err := os.WriteFile(tc.golden, got, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(tc.golden)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				at := firstDiff(got, want)
				t.Errorf("fragment differs from %s at byte %d\n got: %q\nwant: %q",
					tc.golden, at, excerpt(got, at), excerpt(want, at))
			}
		})
	}
}

// The whole point of the run ledger: a breach that neither per-corpus ledger
// can see. Under the cap in both dirs, over it across them.
func TestOnlyTheCombinedViewSeesTheBreach(t *testing.T) {
	for _, dir := range []string{prsDir, issuesDir} {
		if doc := render(t, dir); strings.Contains(doc, "OVER CAP") {
			t.Fatalf("%s alone already breaches the cap, so the fixture cannot show "+
				"what summing the dirs adds:\n%s", dir, doc)
		}
	}
	doc := render(t, prsDir, issuesDir)
	for _, want := range []string{
		"## Run routing ledger (per-person per-run cap: 3)",
		"| person | proposed | cap | status | PRs (reviewer) | issues (mentioned) |",
		"| subhamkrai | 4 | 3 | OVER CAP by 1 |",
		"| travisn | 3 | 3 | at cap |",
		"| Madhu-1 | 2 | 3 | — |",
		"| BlaineEXE | 1 | 3 | — |",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("run ledger is missing %q:\n%s", want, doc)
		}
	}
}

// A total is only actionable if it says which items to drop, and a link has to
// point at the corpus the charge came from.
func TestRowsNameTheItemsBehindEveryCharge(t *testing.T) {
	run, err := Load(prsDir, issuesDir)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := run.Rows()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]map[Corpus][]string{
		"subhamkrai": {PRs: {"18001", "18002"}, Issues: {"17001", "17002"}},
		"travisn":    {PRs: {"18002", "18004"}, Issues: {"17001"}},
		"Madhu-1":    {PRs: {"18003"}, Issues: {"17002"}},
		"BlaineEXE":  {PRs: {"18001"}},
	}
	for _, row := range rows {
		for _, corpus := range order {
			got := strings.Join(row.Items[corpus], ",")
			exp := strings.Join(want[row.Login][corpus], ",")
			if got != exp {
				t.Errorf("%s %s items = %q, want %q", row.Login, corpus, got, exp)
			}
		}
	}
	if len(rows) != len(want) {
		t.Errorf("Rows() = %d rows, want %d: %+v", len(rows), len(want), rows)
	}

	doc := render(t, prsDir, issuesDir)
	for _, want := range []string{
		"[#18001](https://github.com/rook/rook/pull/18001), " +
			"[#18002](https://github.com/rook/rook/pull/18002)",
		"[#17001](https://github.com/rook/rook/issues/17001), " +
			"[#17002](https://github.com/rook/rook/issues/17002)",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("fragment is missing the provenance cell %q:\n%s", want, doc)
		}
	}
}

// The cap charges a login once per ITEM and once per item AGAIN in the other
// corpus. Both halves have to hold at once, or a run either invents breaches
// or hides them: Madhu-1 is named twice on one PR (one charge) and again on an
// issue (a second charge), and subhamkrai's issue spelling differs in case.
func TestChargesPerItemAcrossCorpora(t *testing.T) {
	rows := rowsByLogin(t, prsDir, issuesDir)
	for login, want := range map[string]int{
		"Madhu-1":    2,
		"subhamkrai": 4,
	} {
		if got := rows[login]; got != want {
			t.Errorf("%s counted %d, want %d", login, got, want)
		}
	}
	if _, folded := rows["SubhamKrai"]; folded {
		t.Errorf("a differing spelling became its own row: %+v", rows)
	}
}

// Only what triage PROPOSES counts. The fixtures put a requested reviewer and
// an approving reviewer on 18001 and three existing thread mentions on 17003;
// charging any of them would report a breach the run did not commit.
func TestExistingRoutingIsNotCharged(t *testing.T) {
	rows := rowsByLogin(t, prsDir, issuesDir)
	if got := rows["travisn"]; got != 3 {
		t.Errorf("travisn counted %d, want 3 (an approval and a thread mention are "+
			"not proposals)", got)
	}
	if got := rows["Madhu-1"]; got != 2 {
		t.Errorf("Madhu-1 counted %d, want 2 (a pending review request is not a "+
			"proposal)", got)
	}
	if got := rows["BlaineEXE"]; got != 1 {
		t.Errorf("BlaineEXE counted %d, want 1 (already on the 17003 thread, which "+
			"is somebody else's ping)", got)
	}
}

// A WIP row carries a real proposal, and the cap bounds what a person receives
// across the run rather than what one table shows: travisn's third charge is a
// skip-class PR.
func TestWIPRowsAreCharged(t *testing.T) {
	rows := rowsByLogin(t, prsDir)
	if got := rows["travisn"]; got != 2 {
		t.Errorf("travisn counted %d in the PR dir, want 2 (18002 and the WIP 18004)", got)
	}
}

// Argument order is a caller's convenience, not a fact about the run.
func TestArgumentOrderDoesNotChangeTheFragment(t *testing.T) {
	if a, b := render(t, prsDir, issuesDir), render(t, issuesDir, prsDir); a != b {
		t.Errorf("swapping the dirs changed the fragment:\n%s\n---\n%s", a, b)
	}
}

// Every one of these is a way to get a total that reads plausibly and is
// wrong, which is the only kind of failure this fragment cannot tolerate.
func TestLoadRefusesBadInput(t *testing.T) {
	prsA := sweepDir(t, "prs", `[{"number": 1, "reviewers_proposed": ["a"]}]`)
	prsB := sweepDir(t, "prs", `[{"number": 2, "reviewers_proposed": ["a"]}]`)
	noBatch := t.TempDir()
	writeFile(t, filepath.Join(noBatch, "snapshot.json"), `{"kind": "prs", "items": {}}`)
	badBatch := sweepDir(t, "prs", `[{"number": 1,`)
	noKind := sweepDir(t, "", `[{"number": 1, "reviewers_proposed": ["a"]}]`)
	bothKind := sweepDir(t, "both", `[{"number": 1, "reviewers_proposed": ["a"]}]`)
	issues := sweepDir(t, "issues", `[{"number": 3, "disposition": "", "routing": ["a"]}]`)

	tests := []struct {
		name string
		dirs []string
		want string
	}{
		{"two dirs of one corpus", []string{prsA, prsB}, "are both prs sweeps"},
		{"the same dir twice", []string{prsA, prsA}, "given twice"},
		{"no dir at all", nil, "no sweep dir given"},
		{"three dirs", []string{prsA, issues, prsB}, "a run has at most one per corpus"},
		{"a dir with no batch file", []string{noBatch}, "no batch-*.json to report on"},
		{"an unreadable batch", []string{badBatch}, "batch-1.json"},
		{"a snapshot without a kind", []string{noKind}, `"kind" is ""`},
		{"a kind that is not a corpus", []string{bothKind}, `"kind" is "both"`},
		{"a missing dir", []string{filepath.Join(t.TempDir(), "gone")},
			"no batch-*.json to report on"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			run, err := Load(tc.dirs...)
			if err == nil {
				var buf bytes.Buffer
				_ = run.Render(&buf)
				t.Fatalf("Load(%v) succeeded and rendered:\n%s", tc.dirs, buf.String())
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Load(%v) = %v, want it to mention %q", tc.dirs, err, tc.want)
			}
		})
	}
}

// A run that proposed nobody must say so, not render an empty table that reads
// as "checked, nothing to see" identically to a table that failed to build.
func TestNobodyProposed(t *testing.T) {
	dir := sweepDir(t, "prs", `[{"number": 1, "reviewers_proposed": []}]`)
	doc := render(t, dir)
	if !strings.Contains(doc, "_No routing proposed in this run._") {
		t.Errorf("fragment = %q, want the empty note", doc)
	}
	if strings.Contains(doc, "| person |") {
		t.Errorf("fragment carries a table with no rows:\n%s", doc)
	}
}

// A batch file is model-written text; a login carrying a cell delimiter would
// shift every later column under the wrong header.
func TestRenderEscapesLogins(t *testing.T) {
	run := &Run{Sweeps: []Sweep{{
		Dir:    "2026-08-12-issues-run",
		Corpus: Issues,
		Proposals: []mdreport.Proposal{
			{Number: "17001", Logins: []string{"a|b", "c*d"}},
		},
	}}}
	var buf bytes.Buffer
	if err := run.Render(&buf); err != nil {
		t.Fatalf("Render() = %v", err)
	}
	for _, want := range []string{`| a\|b | 1 |`, `| c\*d | 1 |`} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("fragment is missing %q:\n%s", want, buf.String())
		}
	}
}

// The fragment rule stated on mdreport.Section, checked against real output.
func TestFragmentShape(t *testing.T) {
	for _, dirs := range [][]string{{prsDir}, {prsDir, issuesDir}} {
		doc := render(t, dirs...)
		if !strings.HasPrefix(doc, "\n## ") {
			t.Errorf("%v: fragment starts with %q", dirs, doc[:min(len(doc), 40)])
		}
		if !strings.HasSuffix(doc, "\n") {
			t.Errorf("%v: fragment does not end with a newline", dirs)
		}
		for _, line := range strings.Split(doc, "\n") {
			if strings.HasPrefix(line, "# ") {
				t.Errorf("%v: fragment carries a document title: %q", dirs, line)
			}
		}
	}
}

// A single-corpus run gets one provenance column: a column of em dashes for a
// corpus the run never touched reads as "nothing proposed there".
func TestSingleCorpusRunHasOneProvenanceColumn(t *testing.T) {
	doc := render(t, prsDir)
	if !strings.Contains(doc, "| person | proposed | cap | status | PRs (reviewer) |") {
		t.Errorf("prs-only header is wrong:\n%s", doc)
	}
	if strings.Contains(doc, "issues (mentioned)") {
		t.Errorf("prs-only fragment carries an issues column:\n%s", doc)
	}
	if !strings.Contains(doc, "this run's only sweep dir") {
		t.Errorf("prs-only fragment does not say it spans one dir:\n%s", doc)
	}
}

// Both reports have to carry the combined view: the cap spans the dirs, so a
// maintainer reading only one report still has to see a breach the other dir
// contributed to.
func TestGenerateWritesTheLedgerIntoEveryDir(t *testing.T) {
	dirs := []string{copyFixture(t, prsDir), copyFixture(t, issuesDir)}
	var log bytes.Buffer
	if err := Generate(dirs, &log); err != nil {
		t.Fatalf("Generate() = %v", err)
	}

	var first string
	for _, dir := range dirs {
		b, err := os.ReadFile(filepath.Join(dir, LedgerFile))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "| subhamkrai | 4 | 3 | OVER CAP by 1 |") {
			t.Errorf("%s/%s does not carry the breach:\n%s", dir, LedgerFile, b)
		}
		if first == "" {
			first = string(b)
		} else if string(b) != first {
			t.Errorf("the two dirs got different ledgers:\n%s\n---\n%s", first, b)
		}
	}

	for _, want := range []string{
		"OVER CAP (1): subhamkrai 4/3",
		"4 item(s)",
		"3 item(s)",
		"4 person/people proposed across the run",
		LedgerFile,
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("log is missing %q:\n%s", want, log.String())
		}
	}
}

// A clean run has to be as unmistakable as a breach, or a maintainer reads a
// silent tool as an unfinished one.
func TestGenerateSaysWhenNobodyIsOverCap(t *testing.T) {
	var log bytes.Buffer
	if err := Generate([]string{copyFixture(t, prsDir)}, &log); err != nil {
		t.Fatalf("Generate() = %v", err)
	}
	if !strings.Contains(log.String(), "nobody over the per-person per-run cap of 3") {
		t.Errorf("log = %q, want the clean-run line", log.String())
	}
	if strings.Contains(log.String(), "OVER CAP") {
		t.Errorf("log = %q, want no breach", log.String())
	}
}

// Nothing is written when the run cannot be read: a stale ledger left in place
// is recoverable, a truncated one is a wrong answer at a path the report
// already points at.
func TestGenerateWritesNothingOnFailure(t *testing.T) {
	dir := copyFixture(t, prsDir)
	writeFile(t, filepath.Join(dir, LedgerFile), "previous\n")
	var discard bytes.Buffer
	if err := Generate([]string{dir, dir}, &discard); err == nil {
		t.Fatal("Generate() accepted one dir twice")
	}
	b, err := os.ReadFile(filepath.Join(dir, LedgerFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "previous\n" {
		t.Errorf("%s = %q, want the previous ledger untouched", LedgerFile, b)
	}
}

// Every charge a row reports has to be explained by an item beside it. No
// input can break that — the two derivations agree by construction — so what
// this pins is the guard: mutate the cross-dir sum and Rows errors instead of
// publishing a total nobody can check.
func TestEveryChargeIsExplainedByAnItem(t *testing.T) {
	run, err := Load(prsDir, issuesDir)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := run.Rows()
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		listed := 0
		for _, nums := range row.Items {
			listed += len(nums)
		}
		if listed != row.Count {
			t.Errorf("%s: counted %d, listed %d", row.Login, row.Count, listed)
		}
	}
}

func rowsByLogin(t *testing.T, dirs ...string) map[string]int {
	t.Helper()
	run, err := Load(dirs...)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := run.Rows()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]int{}
	for _, row := range rows {
		out[row.Login] = row.Count
	}
	return out
}

// sweepDir writes the minimum a corpus loader accepts: a snapshot carrying the
// kind, and one batch file.
func sweepDir(t *testing.T, kind, batch string) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "snapshot.json"),
		`{"kind": "`+kind+`", "items": {}}`)
	writeFile(t, filepath.Join(dir, "batch-1.json"), batch)
	return dir
}

func copyFixture(t *testing.T, src string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), filepath.Base(src))
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		t.Fatal(err)
	}
	return dst
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func firstDiff(a, b []byte) int {
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			return i
		}
	}
	return min(len(a), len(b))
}

func excerpt(b []byte, at int) string {
	end := min(at+80, len(b))
	return string(b[min(at, len(b)):end])
}

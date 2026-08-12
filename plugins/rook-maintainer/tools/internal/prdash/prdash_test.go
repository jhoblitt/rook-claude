package prdash

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/mdreport"
)

var update = flag.Bool("update", false, "rewrite testdata/dashboard.golden.html")

const fixture = "2026-08-10-fixture"

func TestClass(t *testing.T) {
	tests := []struct {
		name string
		item Item
		want string
	}{
		{"skip wins over everything", Item{Skip: true, Next: "merge", Takeover: true}, "wip"},
		{"close in next", Item{Next: "comment-then-close"}, "close"},
		{"close candidate in disposition", Item{Next: "monitor",
			Disposition: "CLOSE CANDIDATE - superseded"}, "close"},
		{"close outranks takeover", Item{Next: "close", Takeover: true}, "close"},
		{"takeover", Item{Next: "rebase", Takeover: true}, "take"},
		{"route", Item{Next: "request-reviewers"}, "route"},
		{"ready", Item{Next: "merge"}, "ready"},
		{"route outranks ready", Item{Next: "request-reviewers then merge"}, "route"},
		{"act on comment", Item{Next: "comment: ask for a rebase"}, "act"},
		{"act on dup-link", Item{Next: "dup-link"}, "act"},
		{"act on fill-template", Item{Next: "fill-template"}, "act"},
		{"monitor", Item{Next: "monitor"}, "mon"},
		{"empty next", Item{}, "mon"},
	}
	for _, tc := range tests {
		if got := Class(tc.item); got != tc.want {
			t.Errorf("%s: Class() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// Expectations come from running the Python clean_assessment on the same
// inputs; the point of the cell is that a verdict adding nothing to the chip's
// own counts disappears.
func TestCleanAssessment(t *testing.T) {
	tests := []struct{ in, want string }{
		{"green 12/12", ""},
		{"6/9 passing", ""},
		{"12/12", ""},
		{"failing", ""},
		{"red", ""},
		{"", ""},
		{"red x3 failing: unit", "failing: unit"},
		{"red 3: unit, integration flake", "unit, integration flake"},
		{"green: flaky multus", "flaky multus"},
		{"green   12 , : leftover", "leftover"},
		{"  spaced  ", "spaced"},
		{"pending", "pending"},
		{"greenish tint", "greenish tint"},
		{"redx7flake", "redx7flake"},
	}
	for _, tc := range tests {
		if got := CleanAssessment(tc.in); got != tc.want {
			t.Errorf("CleanAssessment(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNewChip(t *testing.T) {
	tests := []struct {
		name             string
		ci               *CI
		assessment       string
		kind, short, tip string
	}{
		{"no rollup at all", nil, "", "amber", "…", "no checks recorded"},
		{"no checks", &CI{}, "flaky infra", "amber", "…",
			"no checks recorded — assessment: flaky infra"},
		{"all green", &CI{Total: 12, Passing: 12}, "green 12/12", "green", "12/12",
			"12/12 passing"},
		{"pending", &CI{Total: 9, Passing: 6, Pending: 3}, "", "amber", "6/9",
			"6/9 passing · 3 pending"},
		{"failing outranks pending", &CI{Total: 9, Passing: 6, Failing: 2, Pending: 1,
			Failed: []string{"unit", "canary"}}, "", "red", "6/9",
			"6/9 passing · 2 failing: unit, canary · 1 pending"},
		{"failed names capped at six", &CI{Total: 8, Failing: 8,
			Failed: []string{"a", "b", "c", "d", "e", "f", "g", "h"}}, "", "red", "0/8",
			"0/8 passing · 8 failing: a, b, c, d, e, f +2 more"},
	}
	for _, tc := range tests {
		got := NewChip(tc.ci, tc.assessment)
		if got.Kind != tc.kind || got.Short != tc.short || got.Tip != tc.tip {
			t.Errorf("%s: NewChip() = %+v, want kind=%q short=%q tip=%q",
				tc.name, got, tc.kind, tc.short, tc.tip)
		}
	}
}

func TestLinkify(t *testing.T) {
	tests := []struct {
		in   string
		want []Seg
	}{
		{"", nil},
		{"no refs", []Seg{{Plain: "no refs"}}},
		{"18999", []Seg{{Ref: "18999"}}},
		{"#13000", []Seg{{Ref: "13000"}}},
		{"see 13001 and #18999", []Seg{
			{Plain: "see "}, {Ref: "13001"}, {Plain: " and "}, {Ref: "18999"}}},
		{"13000-13001", []Seg{{Ref: "13000"}, {Plain: "-"}, {Ref: "13001"}}},
		{"(13500).", []Seg{{Plain: "("}, {Ref: "13500"}, {Plain: ")."}}},
		{"13000—", []Seg{{Ref: "13000"}, {Plain: "—"}}},
		{"12999 130000 19000 1300 x13000", []Seg{{Plain: "12999 130000 19000 1300 x13000"}}},
		{"113000", []Seg{{Plain: "113000"}}},
		{"#130001", []Seg{{Plain: "#130001"}}},
		// The second match swallows the '#', which the link then reprints.
		{"13000#13001", []Seg{{Ref: "13000"}, {Ref: "13001"}}},
	}
	for _, tc := range tests {
		checkLinkify(t, tc.in, tc.want)
	}
}

// A word boundary is Python's, not RE2's: RE2 reads any non-ASCII letter as a
// boundary, which would turn the tail of "café13000" into a link to an
// unrelated PR. Expectations are what gen_pr_dashboard.py produces.
func TestLinkifyWordBoundariesAreUnicode(t *testing.T) {
	linked := []struct {
		in   string
		want []Seg
	}{
		{"13000", []Seg{{Ref: "13000"}}},
		{"#13000", []Seg{{Ref: "13000"}}},
		{"(13000)", []Seg{{Plain: "("}, {Ref: "13000"}, {Plain: ")"}}},
		{"13000.", []Seg{{Ref: "13000"}, {Plain: "."}}},
		{"é 13000", []Seg{{Plain: "é "}, {Ref: "13000"}}},
		{"13000 é", []Seg{{Ref: "13000"}, {Plain: " é"}}},
		// The '#' is the boundary, so what precedes it never matters.
		{"a#13000", []Seg{{Plain: "a"}, {Ref: "13000"}}},
		{"é#13000", []Seg{{Plain: "é"}, {Ref: "13000"}}},
	}
	for _, tc := range linked {
		checkLinkify(t, tc.in, tc.want)
	}

	for _, in := range []string{
		"é13000", "13000é", "é13000é",
		"中13000", "13000中", "_13000", "13000_", "x13000",
	} {
		checkLinkify(t, in, []Seg{{Plain: in}})
	}
}

// Deliberate divergence from the Python, whose \d matches any Unicode decimal:
// "13٠٠٠" linkified there, minting a dead URL out of digits
// that only look like an issue number. ASCII digits only here.
func TestLinkifyIgnoresNonASCIIDigits(t *testing.T) {
	for _, in := range []string{"13٠٠٠", "1٣000"} {
		checkLinkify(t, in, []Seg{{Plain: in}})
	}
}

func checkLinkify(t *testing.T, in string, want []Seg) {
	t.Helper()
	got := Linkify(in)
	if len(got) != len(want) {
		t.Errorf("Linkify(%q) = %+v, want %+v", in, got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("Linkify(%q)[%d] = %+v, want %+v", in, i, got[i], want[i])
		}
	}
}

func TestRefsKeepsIssuesOnceInOrder(t *testing.T) {
	s := &Sweep{RefTypes: map[string]string{
		"13001": "Issue", "18999": "Issue", "17004": "PullRequest"}}
	var item Item
	raw := `{"xlinks":[{"number":13001},{"number":13001},{"number":17004},
	          {"number":18999},12345,{"note":"no number"},null,"13001"],
	         "dups":[{"number":18999},{"number":13500}]}`
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := []string{"13001", "18999"}
	got := s.Refs(item)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("Refs() = %v, want %v", got, want)
	}
}

func TestReviewers(t *testing.T) {
	rv := &Reviews{
		Latest: []Review{
			{Login: "travisn", State: "APPROVED"},
			{Login: "BlaineEXE", State: "CHANGES_REQUESTED"},
			{Login: copilotLogin, State: "COMMENTED"},
			{Login: "subhamkrai", State: "DISMISSED"},
			{Login: "parth-gr", State: "PENDING"},
			{Login: "travisn", State: "COMMENTED"},
		},
		Requested: []string{"BlaineEXE", "rook/maintainers"},
	}
	got := Reviewers([]Text{"travisn (owns rgw)", "obnoxxx"}, rv)

	want := []struct{ name, class, icon, title string }{
		{"BlaineEXE", "pend", iconRequest, "re-requested"},
		{"rook/maintainers", "pend", iconRequest, "review requested"},
		{"travisn", "com", iconComment, "commented"},
		{"Copilot", "com", iconComment, "commented"},
		{"subhamkrai", "dis", iconDismiss, "review dismissed"},
		{"parth-gr", "com", middot, "pending"},
		{"travisn", "prop", iconPropose, "proposed reviewer — owns rgw"},
		{"obnoxxx", "prop", iconPropose, "proposed reviewer (triage routing)"},
	}
	if len(got) != len(want) {
		t.Fatalf("Reviewers() returned %d rows, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].User.Name != w.name || got[i].Class != w.class ||
			got[i].Icon != w.icon || got[i].Title != w.title {
			t.Errorf("row %d = %+v, want %v", i, got[i], w)
		}
	}
	if got[3].User.Href != "https://github.com/apps/"+copilotLogin {
		t.Errorf("Copilot href = %q", got[3].User.Href)
	}
}

func TestReviewersWithoutSnapshotReviews(t *testing.T) {
	if got := Reviewers(nil, nil); got != nil {
		t.Errorf("Reviewers(nil, nil) = %+v, want nil", got)
	}
	got := Reviewers([]Text{"travisn"}, nil)
	if len(got) != 1 || got[0].Class != "prop" {
		t.Errorf("Reviewers() = %+v, want one proposed row", got)
	}
}

func TestSplitProposal(t *testing.T) {
	tests := []struct{ in, login, note string }{
		{"travisn (owns the ceph area)", "travisn", "owns the ceph area"},
		{"BlaineEXE", "BlaineEXE", ""},
		{"travisn ()", "travisn", ""},
		{"a.b-c (x) (y)", "a.b-c", "x) (y"},
		{"@travisn", "@travisn", ""},
		{"  leading", "  leading", ""},
	}
	for _, tc := range tests {
		login, note := splitProposal(tc.in)
		if login != tc.login || note != tc.note {
			t.Errorf("splitProposal(%q) = (%q, %q), want (%q, %q)",
				tc.in, login, note, tc.login, tc.note)
		}
	}
}

func TestLoadToleratesMissingOptionalInputs(t *testing.T) {
	dir := t.TempDir()
	sweep := filepath.Join(dir, "2026-03-04-minimal")
	if err := os.Mkdir(sweep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sweep, "snapshot.json"),
		[]byte(`{"items":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(sweep)
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if s.Date != "2026-03-04" {
		t.Errorf("Date = %q, want 2026-03-04", s.Date)
	}
	if len(s.Skips) != 0 || len(s.RefTypes) != 0 || len(s.Items) != 0 {
		t.Errorf("expected an empty sweep, got %+v", s)
	}
}

func TestLoadRequiresSnapshot(t *testing.T) {
	if _, err := Load(t.TempDir()); err == nil {
		t.Error("Load() without snapshot.json returned no error")
	}
}

func TestLoadSortsItemsAcrossBatches(t *testing.T) {
	s := loadFixture(t)
	var got []int
	for _, it := range s.Items {
		got = append(got, it.Number)
	}
	want := []int{18100, 18120, 18150, 18200, 18250, 18300, 18500}
	for i := range want {
		if i >= len(got) || got[i] != want[i] {
			t.Fatalf("item order = %v, want %v", got, want)
		}
	}
}

func TestGoldenDashboard(t *testing.T) {
	got := renderFixture(t)
	golden := filepath.Join("testdata", "dashboard.golden.html")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("dashboard differs from %s (rerun with -update to accept):\n%s",
			golden, firstDiff(string(want), string(got)))
	}
}

// The summary column, the labels and the disposition are attacker-controlled
// text from a public PR, so the page must carry them only as entities.
func TestUntrustedTextIsEscaped(t *testing.T) {
	out := string(renderFixture(t))
	for _, raw := range []string{"<script>alert", "<b>replica</b>", "<li><script></li>",
		`pool "size"`, "add 'foo'", "<b>test</b>"} {
		if strings.Contains(out, raw) {
			t.Errorf("unescaped %q reached the page", raw)
		}
	}
	for _, want := range []string{
		"&lt;script&gt;alert(&#39;x&#39;)&lt;/script&gt;",
		"<li>&lt;script&gt;</li>",
		"&#34;size&#34; &amp; &lt;b&gt;replica&lt;/b&gt;",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in the page", want)
		}
	}
}

func TestGenerateWritesDashboardAndSummary(t *testing.T) {
	dir := filepath.Join(t.TempDir(), fixture)
	copyDir(t, filepath.Join("testdata", fixture), dir)

	var log bytes.Buffer
	if err := Generate(dir, &log); err != nil {
		t.Fatalf("Generate() = %v", err)
	}
	wantLog := fixture + ": 6 assessed, 1 WIP, 2 skipped; " +
		"CI/titles/labels/reviews from snapshot\n"
	if log.String() != wantLog {
		t.Errorf("summary = %q, want %q", log.String(), wantLog)
	}
	got, err := os.ReadFile(filepath.Join(dir, "dashboard.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, renderFixture(t)) {
		t.Error("Generate() wrote something other than the rendered page")
	}
}

func loadFixture(t *testing.T) *Sweep {
	t.Helper()
	s, err := Load(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	return s
}

func renderFixture(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := Render(&buf, loadFixture(t).Page()); err != nil {
		t.Fatalf("Render() = %v", err)
	}
	return buf.Bytes()
}

func firstDiff(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(w) && i < len(g); i++ {
		if w[i] != g[i] {
			return fmt.Sprintf("line %d\nwant: %s\ngot:  %s", i+1, w[i], g[i])
		}
	}
	return fmt.Sprintf("line counts differ: want %d, got %d", len(w), len(g))
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func renderMarkdownFixture(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	if err := RenderMarkdown(&buf, loadFixture(t).Page()); err != nil {
		t.Fatalf("RenderMarkdown() = %v", err)
	}
	return buf.String()
}

func TestGoldenMarkdown(t *testing.T) {
	got := []byte(renderMarkdownFixture(t))
	golden := filepath.Join("testdata", "report-tables.golden.md")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("fragment differs from %s (rerun with -update to accept):\n%s",
			golden, firstDiff(string(want), string(got)))
	}
}

// The fragment rule stated on mdreport.Section, checked against real output.
func TestMarkdownIsAFragment(t *testing.T) {
	doc := renderMarkdownFixture(t)
	if !strings.HasPrefix(doc, "\n## ") {
		t.Errorf("fragment starts with %q", doc[:min(len(doc), 40)])
	}
	if !strings.HasSuffix(doc, "\n") {
		t.Error("fragment does not end with a newline")
	}
	for _, line := range strings.Split(doc, "\n") {
		if strings.HasPrefix(line, "# ") {
			t.Errorf("fragment carries a document title: %q", line)
		}
	}
}

// reporting.md fixes the order: assessed rows by number, then the skipped
// section at the BOTTOM, WIP signal-rows ahead of the draft/bot rows.
func TestMarkdownOrdering(t *testing.T) {
	doc := renderMarkdownFixture(t)
	sections := []string{"\n## Assessed PRs", "\n## Skipped", "\n### WIP signal-rows",
		"\n### Draft, bot and do-not-merge", "\n## Reviewer ledger"}
	at := make([]int, len(sections))
	for i, s := range sections {
		if at[i] = strings.Index(doc, s); at[i] < 0 {
			t.Fatalf("no %q section", s)
		}
		if i > 0 && at[i] < at[i-1] {
			t.Fatalf("%q precedes %q", sections[i], sections[i-1])
		}
	}

	assessed := rowNumbers(t, doc[at[0]:at[1]])
	want := []int{18100, 18120, 18150, 18200, 18250, 18300}
	if fmt.Sprint(assessed) != fmt.Sprint(want) {
		t.Errorf("assessed rows = %v, want %v", assessed, want)
	}
	if got := rowNumbers(t, doc[at[2]:at[3]]); fmt.Sprint(got) != fmt.Sprint([]int{18500}) {
		t.Errorf("WIP rows = %v, want [18500]", got)
	}
	if got := rowNumbers(t, doc[at[3]:at[4]]); fmt.Sprint(got) != fmt.Sprint([]int{18600, 18700}) {
		t.Errorf("draft/bot rows = %v, want [18600 18700]", got)
	}
}

// The CI cell is passing/total from the snapshot rollup, or the ellipsis when
// the item has no checks at all — never a verdict word, and never the
// triager's own count.
func TestMarkdownCICells(t *testing.T) {
	doc := renderMarkdownFixture(t)
	shape := regexp.MustCompile(`\A(?:\d+/\d+|…)\z`)
	var got []string
	for _, tbl := range tables(doc) {
		if len(tbl.header) < 4 || tbl.header[3] != "CI" {
			continue
		}
		for _, row := range tbl.rows {
			cell := row[3]
			if !shape.MatchString(cell) {
				t.Errorf("row %s: CI cell = %q, want passing/total or …", row[0], cell)
			}
			got = append(got, cell)
		}
	}
	want := []string{"…", "12/12", "2/10", "6/9", "…", "12/12", "1/4"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("CI cells = %v, want %v", got, want)
	}
}

// Every column of every row survives a renderer's split, which is only true
// while the cells are escaped: one raw "|" in a PR title shifts the rest of
// that row under the wrong headers.
func TestMarkdownRowsKeepTheirColumns(t *testing.T) {
	for _, tbl := range tables(renderMarkdownFixture(t)) {
		for _, row := range tbl.rows {
			if len(row) != len(tbl.header) {
				t.Errorf("row %q has %d cells, want %d (header %v)",
					row, len(row), len(tbl.header), tbl.header)
			}
		}
	}
}

// The hostile fixture text has to reach the page as itself, not as markup and
// not as a broken row.
func TestMarkdownEscapesUntrustedText(t *testing.T) {
	doc := renderMarkdownFixture(t)
	for _, want := range []string{
		`fix: nil deref \| \` + "`ceph status\\`" + ` in the object controller \[see #13001\]`,
		`needs\|split`,
		`\<script>alert('x')\</script>`,
		`\<b>test\</b>`,
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("expected %q in the fragment", want)
		}
	}
	// The summary column is the title verbatim: its "#13001" is text, not a
	// reference the fragment may resolve to a link of its own.
	if strings.Contains(doc, "[see [#13001]") {
		t.Error("a title reference was linkified")
	}
}

// A cap_note is the only record of a set the cap forced a swap in, and it
// reaches the report through the ledger rather than the dashboard.
func TestMarkdownReviewerLedger(t *testing.T) {
	doc := renderMarkdownFixture(t)
	for _, want := range []string{
		"| reviewer | proposed | cap | status |",
		"| BlaineEXE | 1 | 3 | — |",
		"| subhamkrai | 1 | 3 | — |",
		"| travisn | 1 | 3 | — |",
		"### Cap-swapped sets (1)",
		"| [#18250](https://github.com/rook/rook/pull/18250) | BlaineEXE at cap " +
			"([#13001](https://github.com/rook/rook/issues/13001)) → swapped for subhamkrai |",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("ledger is missing %q", want)
		}
	}
}

// 18120 names travisn twice, once bare and once with a note — what a triager
// writes when only one of the two entries needs explaining. The cap charges
// that once: counting it twice reports a breach on a sweep that stayed inside
// the cap, and nothing downstream re-checks the number.
func TestMarkdownLedgerChargesADuplicateProposalOnce(t *testing.T) {
	doc := renderMarkdownFixture(t)
	if !strings.Contains(doc, "| travisn | 1 | 3 | — |") {
		t.Errorf("travisn was proposed on one PR; the ledger says otherwise:\n%s", doc)
	}
}

func TestGenerateMarkdownWritesTheFragment(t *testing.T) {
	dir := filepath.Join(t.TempDir(), fixture)
	copyDir(t, filepath.Join("testdata", fixture), dir)

	var log bytes.Buffer
	if err := GenerateMarkdown(dir, &log); err != nil {
		t.Fatalf("GenerateMarkdown() = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, mdreport.ReportFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != renderMarkdownFixture(t) {
		t.Error("GenerateMarkdown() wrote something other than the rendered fragment")
	}
	// 18100 is in a batch but not in the snapshot: silence there would leave a
	// row whose summary and CI look merely empty.
	wantLog := "warning: 1 assessed PR(s) absent from snapshot.json, " +
		"marked in the summary column: 18100\n" +
		fixture + "/report-tables.md: 6 assessed, 1 WIP, 2 skipped; " +
		"3 reviewer(s) proposed, 1 cap-swapped set(s)\n"
	if log.String() != wantLog {
		t.Errorf("log = %q, want %q", log.String(), wantLog)
	}
}

// An empty table and a sweep dir whose batch files never arrived look
// identical in the output; only one of them is a finished sweep.
func TestGenerateMarkdownRefusesASweepWithoutBatches(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "2026-03-04-minimal")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "snapshot.json"),
		[]byte(`{"items":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := GenerateMarkdown(dir, io.Discard); err == nil {
		t.Error("GenerateMarkdown() accepted a sweep dir with no batch file")
	}
}

type mdTable struct {
	header []string
	rows   [][]string
}

// tables parses the fragment the way a renderer would: a run of lines starting
// with the delimiter, split on delimiters that are not backslash escaped.
func tables(doc string) []mdTable {
	var out []mdTable
	var cur *mdTable
	for _, line := range strings.Split(doc, "\n") {
		if !strings.HasPrefix(line, "|") {
			cur = nil
			continue
		}
		cells := splitCells(line)
		switch {
		case cur == nil:
			out = append(out, mdTable{header: cells})
			cur = &out[len(out)-1]
		case cells[0] == "---":
		default:
			cur.rows = append(cur.rows, cells)
		}
	}
	return out
}

func splitCells(line string) []string {
	var cells []string
	var cur strings.Builder
	esc := false
	for _, r := range line {
		switch {
		case esc:
			cur.WriteRune(r)
			esc = false
		case r == '\\':
			cur.WriteRune(r)
			esc = true
		case r == '|':
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	cells = append(cells, strings.TrimSpace(cur.String()))
	return cells[1 : len(cells)-1] // a row opens and closes with the delimiter
}

var rowNumber = regexp.MustCompile(`\A\[#(\d+)\]`)

func rowNumbers(t *testing.T, section string) []int {
	t.Helper()
	var out []int
	for _, tbl := range tables(section) {
		for _, row := range tbl.rows {
			m := rowNumber.FindStringSubmatch(row[0])
			if m == nil {
				t.Errorf("row does not open with a linked number: %q", row[0])
				continue
			}
			n, err := strconv.Atoi(m[1])
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, n)
		}
	}
	return out
}

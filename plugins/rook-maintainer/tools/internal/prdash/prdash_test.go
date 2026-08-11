package prdash

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

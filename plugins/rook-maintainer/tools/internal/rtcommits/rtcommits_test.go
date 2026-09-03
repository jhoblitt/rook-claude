package rtcommits

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtanalyze"
)

var update = flag.Bool("update", false, "rewrite testdata/golden.* from the current output")

const (
	goldenNow    = "2026-08-11T00:00:00Z"
	goldenMonths = 24
	fixture      = "testdata/git-log.txt"
)

func at(t *testing.T, iso string) time.Time {
	t.Helper()
	parsed, err := rtanalyze.ParseISO(iso)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func fixtureCommits(t *testing.T) []Commit {
	t.Helper()
	f, err := os.Open(fixture)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	commits, err := ParseLog(f)
	if err != nil {
		t.Fatal(err)
	}
	return commits
}

func mineFixture(t *testing.T, now string, months float64) *Result {
	t.Helper()
	res, err := Mine(fixtureCommits(t), Options{
		Now:    at(t, now),
		Months: months,
		Source: Source{Mode: "log", Path: fixture},
	})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// The golden pins the whole document, not a handful of fields: the kb refresh
// diffs this output between runs, and a drifting weight or a reordered author
// list is invisible in a spot check. testdata/git-log.txt is built to exercise
// the boundaries (182/183 and 365/366 days, the cutoff itself), bot exclusion,
// renames, a path with a space and a C-quoted path, so regenerating with
// -update after an intentional change shows exactly what moved.
func TestGolden(t *testing.T) {
	res := mineFixture(t, goldenNow, goldenMonths)
	doc, err := Render(res.Doc)
	if err != nil {
		t.Fatal(err)
	}
	summary := strings.Join(res.Summary, "\n") + "\n"

	if *update {
		writeGolden(t, "golden.json", string(doc))
		writeGolden(t, "golden.txt", summary)
		return
	}
	assertGolden(t, "golden.json", string(doc))
	assertGolden(t, "golden.txt", summary)
}

func writeGolden(t *testing.T, name, got string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join("testdata", name), []byte(got), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	want, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("%s differs (re-run with -update to inspect):\n%s", name, firstDiff(got, string(want)))
	}
}

func firstDiff(got, want string) string {
	i := 0
	for i < len(got) && i < len(want) && got[i] == want[i] {
		i++
	}
	from := max(i-120, 0)
	return "at byte " + strconv.Itoa(i) + "\ngot:  " + snippet(got, from) + "\nwant: " + snippet(want, from)
}

func snippet(s string, from int) string {
	return s[from:min(from+240, len(s))]
}

// TestRecencyWeightBoundaries pins the decay directly as well as through the
// golden: the fixture's dates sit one second either side of the 6- and
// 12-month edges, which is where an off-by-one in ageDays hides.
func TestRecencyWeightBoundaries(t *testing.T) {
	now := at(t, goldenNow)
	tests := []struct {
		when string
		want float64
	}{
		{"2026-08-11T00:00:00Z", 1.0},
		{"2026-02-10T00:00:00Z", 1.0},
		{"2026-02-10T05:30:00+05:30", 1.0},
		{"2026-02-09T00:00:01Z", 1.0},
		{"2026-02-09T00:00:00Z", 0.5},
		{"2025-08-11T00:00:00Z", 0.5},
		{"2025-08-10T23:59:59Z", 0.5},
		{"2025-08-10T00:00:00Z", 0.25},
		{"2020-01-01T00:00:00Z", 0.25},
	}
	for _, tc := range tests {
		if got := weight(now, at(t, tc.when)); got != tc.want {
			t.Errorf("weight(%s) = %v, want %v", tc.when, got, tc.want)
		}
	}
}

func TestWeightedTotalsPerArea(t *testing.T) {
	res := mineFixture(t, goldenNow, goldenMonths)
	tests := []struct {
		area, name string
		weighted   float64
		raw        int
	}{
		{"object", "Alice Example", 3, 3},
		{"object", "Bob Builder", 1.75, 4},
		{"csi", "Carol Coder", 0.25, 1},
		{"design", "Heidi Type", 0.5, 1},
	}
	for _, tc := range tests {
		got, ok := authorIn(res.Doc, tc.area, tc.name)
		if !ok {
			t.Errorf("%s missing from %s", tc.name, tc.area)
			continue
		}
		if got.WeightedCommits != tc.weighted || got.Commits != tc.raw {
			t.Errorf("%s in %s = %v/%d, want %v/%d",
				tc.name, tc.area, got.WeightedCommits, got.Commits, tc.weighted, tc.raw)
		}
	}
}

// The cutoff is inclusive: the fixture's pair one second either side of it is
// the only thing standing between "24 months" and "24 months and a bit".
func TestWindowBoundsAreInclusive(t *testing.T) {
	p := mineFixture(t, goldenNow, goldenMonths).Doc.Provenance
	if p.CommitsScanned != 20 || p.CommitsInWindow != 19 {
		t.Errorf("scanned/in_window = %d/%d, want 20/19", p.CommitsScanned, p.CommitsInWindow)
	}
	if p.Cutoff != "2024-08-10T00:00:00Z" || p.WindowDays != 731 {
		t.Errorf("cutoff/days = %s/%d, want 2024-08-10T00:00:00Z/731", p.Cutoff, p.WindowDays)
	}
	if got := *p.OldestCommit; got != "2024-08-10T00:00:00Z" {
		t.Errorf("oldest_commit = %s, want the commit sitting on the cutoff", got)
	}
}

func TestBotsExcludedButBotLookingHumansKept(t *testing.T) {
	res := mineFixture(t, goldenNow, goldenMonths)
	if got := res.Doc.Provenance.CommitsBot; got != 1 {
		t.Errorf("commits_bot_excluded = %d, want 1", got)
	}
	if _, ok := res.Doc.Areas["build"]; ok {
		t.Error("dependabot's go.mod commit still bucketed into build")
	}
	if _, ok := authorIn(res.Doc, "ceph-mon", "Talbot Human"); !ok {
		t.Error("a human whose display name ends in \"bot\" was dropped as a bot")
	}
	for _, id := range res.Doc.Identities {
		if strings.Contains(id.Name, "dependabot") {
			t.Errorf("bot %q in the identity roster", id.Name)
		}
	}
}

func TestIdentitiesUnionOnSharedNameOrEmail(t *testing.T) {
	res := mineFixture(t, goldenNow, goldenMonths)
	byName := map[string]Identity{}
	for _, id := range res.Doc.Identities {
		byName[id.Name] = id
	}
	if len(res.Doc.Identities) != 10 {
		t.Errorf("identities = %d, want 10", len(res.Doc.Identities))
	}

	alice, ok := byName["Alice Example"]
	if !ok {
		t.Fatalf("no Alice Example in %v", byName)
	}
	if got := strings.Join(alice.Emails, ","); got != "alice@corp.example,alice@example.com" {
		t.Errorf("Alice's emails = %s, want both addresses unioned", got)
	}
	if alice.Commits != 3 {
		t.Errorf("Alice's commits = %d, want 3 (the alice-example spelling is the same person)", alice.Commits)
	}
	if _, ok := byName["alice-example"]; ok {
		t.Error("the second spelling of Alice's name became its own identity")
	}

	anas, ok := byName["Anas Khan"]
	if !ok {
		t.Fatal("no Anas Khan")
	}
	if anas.Login == nil || *anas.Login != "anxkhn" {
		t.Errorf("Anas's login = %v, want anxkhn carried from the noreply address to the whole identity", anas.Login)
	}
	if res.Doc.Provenance.IdentitiesNoLogin != 9 {
		t.Errorf("identities_without_login = %d, want 9", res.Doc.Provenance.IdentitiesNoLogin)
	}
	for _, id := range res.Doc.Identities {
		if id.Login == nil && strings.Contains(id.Name, " ") && strings.Contains(id.Name, "@") {
			t.Errorf("identity %q looks like a fabricated login", id.Name)
		}
	}
}

func TestCommitTouchingSeveralAreasCountsInEach(t *testing.T) {
	res := mineFixture(t, goldenNow, goldenMonths)
	for _, area := range []string{"object", "object-multisite", "ceph-nfs", "docs"} {
		if _, ok := authorIn(res.Doc, area, "Erin Rename"); !ok {
			t.Errorf("Erin's rename commit missing from %s", area)
		}
	}
	if got := res.Doc.Areas["ceph-nfs"].Commits; got != 1 {
		t.Errorf("ceph-nfs commits = %d, want 1 — the destination of the rename", got)
	}
	if got := res.Doc.Areas["object"].Commits; got != 8 {
		t.Errorf("object commits = %d, want 8", got)
	}
}

func TestCommitsMatchingNoAreaAreCountedNotDropped(t *testing.T) {
	p := mineFixture(t, goldenNow, goldenMonths).Doc.Provenance
	if p.CommitsUnmatched != 2 {
		t.Errorf("commits_unmatched = %d, want 2 (deploy/examples + the empty commit)", p.CommitsUnmatched)
	}
	if p.CommitsCounted+p.CommitsBot+p.CommitsUnmatched != p.CommitsInWindow {
		t.Errorf("counted+bot+unmatched = %d, want in_window = %d",
			p.CommitsCounted+p.CommitsBot+p.CommitsUnmatched, p.CommitsInWindow)
	}
	for _, id := range mineFixture(t, goldenNow, goldenMonths).Doc.Identities {
		if id.Name == "Dana Multi" && id.Commits != 2 {
			t.Errorf("Dana's commits = %d, want 2: an unbucketed commit is still work", id.Commits)
		}
	}
}

// A sha alone does not say which revision it came from, which is how a KB
// asserting origin/master got written from a HEAD mine. Both the document and
// the summary name the ref.
func TestRepoProvenanceNamesTheRef(t *testing.T) {
	const head = "0123456789abcdef0123456789abcdef01234567"
	res, err := Mine(fixtureCommits(t), Options{
		Now:    at(t, goldenNow),
		Months: goldenMonths,
		Source: Source{Mode: "repo", Path: "/checkouts/rook", Ref: DefaultRef, Head: head},
	})
	if err != nil {
		t.Fatal(err)
	}
	p := res.Doc.Provenance
	if p.Ref != DefaultRef {
		t.Errorf("provenance ref = %q, want %q", p.Ref, DefaultRef)
	}
	if !strings.HasSuffix(p.GitLog, " "+DefaultRef) {
		t.Errorf("git_log = %q, want it to end in the mined revision", p.GitLog)
	}
	want := "source: repo /checkouts/rook @ " + DefaultRef + " 0123456789ab"
	if got := res.Summary[0]; got != want {
		t.Errorf("summary line = %q, want %q", got, want)
	}
}

// --log consumes a dump whose revision was chosen when it was captured, so
// nothing here may assert one.
func TestLogProvenanceAssertsNoRef(t *testing.T) {
	res := mineFixture(t, goldenNow, goldenMonths)
	p := res.Doc.Provenance
	if p.Ref != "" || p.Head != "" {
		t.Errorf("log-mode provenance claims ref=%q head=%q", p.Ref, p.Head)
	}
	if strings.HasSuffix(p.GitLog, DefaultRef) {
		t.Errorf("git_log = %q, want the ref-less command", p.GitLog)
	}
}

// An empty window is a legitimate answer — it says the window is empty, with
// the provenance to prove the mine ran. An empty INPUT is not; see below.
func TestEmptyWindowStillReportsProvenance(t *testing.T) {
	res, err := Mine(fixtureCommits(t), Options{
		Now:    at(t, "2030-01-01T00:00:00Z"),
		Months: 1,
		Source: Source{Mode: "log", Path: fixture},
	})
	if err != nil {
		t.Fatal(err)
	}
	p := res.Doc.Provenance
	if p.CommitsScanned != 20 || p.CommitsInWindow != 0 || p.Areas != 0 || p.Identities != 0 {
		t.Errorf("empty window provenance = %+v", p)
	}
	if p.OldestCommit != nil || p.NewestCommit != nil {
		t.Errorf("oldest/newest = %v/%v, want null for an empty window", p.OldestCommit, p.NewestCommit)
	}
	if len(res.Doc.Areas) != 0 {
		t.Errorf("areas = %v, want none", res.Doc.Areas)
	}
}

func TestMineRejectsEmptyInput(t *testing.T) {
	_, err := Mine(nil, Options{Now: at(t, goldenNow), Months: 24, Source: Source{Mode: "log", Path: "empty.log"}})
	if err == nil {
		t.Fatal("Mine accepted zero commits; a mine that scans nothing must fail loud")
	}
	if !strings.Contains(err.Error(), "empty.log") {
		t.Errorf("error %q does not name the input", err)
	}
}

func TestMineRejectsNonPositiveMonths(t *testing.T) {
	for _, months := range []float64{0, -1} {
		if _, err := Mine(fixtureCommits(t), Options{
			Now: at(t, goldenNow), Months: months, Source: Source{Mode: "log", Path: fixture},
		}); err == nil {
			t.Errorf("Mine accepted --months=%v", months)
		}
	}
}

// The window arithmetic is rtfetch.WindowCutoff itself now, so "the two agree"
// is nothing to assert. What is still rtcommits' own is the window it PUBLISHES
// — window_days has no counterpart in rt_fetch_state.json — and that an
// unrepresentable window fails the mine instead of producing a document.
func TestProvenanceRecordsTheSharedWindow(t *testing.T) {
	now := at(t, goldenNow)
	for _, tc := range []struct {
		months float64
		days   int
	}{{24, 731}, {6, 183}, {12, 365}, {1, 30}} {
		res, err := Mine(fixtureCommits(t), Options{
			Now: now, Months: tc.months, Source: Source{Mode: "log", Path: fixture},
		})
		if err != nil {
			t.Fatal(err)
		}
		p := res.Doc.Provenance
		if p.WindowDays != tc.days || p.Cutoff != isoformat(now.AddDate(0, 0, -tc.days)) {
			t.Errorf("--months=%v: window_days=%d cutoff=%s, want %d days",
				tc.months, p.WindowDays, p.Cutoff, tc.days)
		}
	}
	if _, err := Mine(fixtureCommits(t), Options{
		Now: now, Months: 1e9, Source: Source{Mode: "log", Path: fixture},
	}); err == nil {
		t.Error("Mine accepted a window that overflows the calendar")
	}
}

func TestTieBreakFallsBackToLowercasedName(t *testing.T) {
	res := mineFixture(t, goldenNow, goldenMonths)
	var got []string
	for _, a := range res.Doc.Areas["ceph-dashboard"].Authors {
		got = append(got, a.Name)
	}
	if strings.Join(got, ",") != "Erin Rename,Frank Tie" {
		t.Errorf("ceph-dashboard authors = %v, want equal weights broken by name", got)
	}
}

// git log hands commits over newest first, which would let "the last one seen"
// pass for "the most recent one". last_active must not depend on that.
func TestLastActiveIsTheLatestCommitWhateverTheOrder(t *testing.T) {
	author := func(when string) Commit {
		return Commit{
			SHA: when, Name: "Alice Example", Email: "alice@example.com",
			When: at(t, when), Paths: []string{"pkg/operator/ceph/object/rgw.go"},
		}
	}
	res, err := Mine(
		[]Commit{author("2025-01-05T00:00:00Z"), author("2026-06-30T00:00:00Z"), author("2025-11-01T00:00:00Z")},
		Options{Now: at(t, goldenNow), Months: goldenMonths, Source: Source{Mode: "log", Path: "inline"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := authorIn(res.Doc, "object", "Alice Example")
	if !ok {
		t.Fatal("no Alice Example in object")
	}
	if got.LastActive != "2026-06" {
		t.Errorf("last_active = %s, want 2026-06", got.LastActive)
	}
	if res.Doc.Identities[0].LastActive != "2026-06" {
		t.Errorf("identity last_active = %s, want 2026-06", res.Doc.Identities[0].LastActive)
	}
}

// A commit author address is whatever the contributor typed, and its login
// travels into the kb and from there into @-mentions and review requests. An
// address that matches the noreply shape but carries something that is not a
// login is an unresolved identity, never a login.
func TestHostileNoreplyAddressYieldsNoLogin(t *testing.T) {
	res, err := Mine([]Commit{{
		SHA: "a1", Name: "Alice", Email: "12+alice](https://evil.com)@users.noreply.github.com",
		When: at(t, "2026-08-01T00:00:00Z"), Paths: []string{"pkg/operator/ceph/object/rgw.go"},
	}, {
		SHA: "a2", Name: "Bob", Email: "34+bob@users.noreply.github.com",
		When: at(t, "2026-08-01T00:00:00Z"), Paths: []string{"pkg/operator/ceph/object/rgw.go"},
	}}, Options{Now: at(t, goldenNow), Months: goldenMonths, Source: Source{Mode: "log", Path: "inline"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range res.Doc.Identities {
		switch id.Name {
		case "Alice":
			if id.Login != nil {
				t.Errorf("Alice's login = %q, want null: the capture is not a login", *id.Login)
			}
		case "Bob":
			if id.Login == nil || *id.Login != "bob" {
				t.Errorf("Bob's login = %v, want bob still read from a well-formed address", id.Login)
			}
		}
	}
	if got := res.Doc.Provenance.IdentitiesNoLogin; got != 1 {
		t.Errorf("identities_without_login = %d, want 1: the rejected capture is a gap to resolve", got)
	}
}

func TestSummaryLeadsWithProvenance(t *testing.T) {
	res := mineFixture(t, goldenNow, goldenMonths)
	for i, want := range []string{"source: log ", "window: 2024-08-10..2026-08-11", "commits: scanned=20", "identities: 10"} {
		if !strings.HasPrefix(res.Summary[i], want) {
			t.Errorf("summary line %d = %q, want prefix %q", i, res.Summary[i], want)
		}
	}
}

func authorIn(doc Doc, area, name string) (Author, bool) {
	for _, a := range doc.Areas[area].Authors {
		if a.Name == name {
			return a, true
		}
	}
	return Author{}, false
}

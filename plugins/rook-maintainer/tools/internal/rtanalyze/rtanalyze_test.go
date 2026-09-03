package rtanalyze

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const goldenNow = "2026-08-01T00:00:00Z"

// testdata/golden.* were produced by the rt_analyze.py this command replaces,
// run against the same fixture with the same --now. Byte equality is the whole
// contract: the miner output is consumed by an assembler that diffs it, and
// float weighting plus map iteration order are exactly where a rewrite drifts
// silently. Regenerate only against the Python, never from this code.
func TestGoldenMatchesPython(t *testing.T) {
	assertMatchesGolden(t, analyzeFixture(t, 15), "testdata")
}

func assertMatchesGolden(t *testing.T, res *Result, dir string) {
	t.Helper()
	want, err := os.ReadFile(filepath.Join(dir, "golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := Marshal(res.Doc); got != string(want) {
		t.Errorf("document differs from Python golden:\n%s", firstDiff(got, string(want)))
	}

	wantErr, err := os.ReadFile(filepath.Join(dir, "golden.stderr"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(res.Summary, "\n") + "\n"; got != string(wantErr) {
		t.Errorf("summary differs from Python golden:\ngot:  %q\nwant: %q", got, string(wantErr))
	}
}

func analyzeFixture(t *testing.T, top int) *Result {
	t.Helper()
	prs, err := LoadPRs(filepath.Join("testdata", "rt_prs.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	state, err := LoadState(filepath.Join("testdata", "rt_fetch_state.json"))
	if err != nil {
		t.Fatal(err)
	}
	owners, err := os.Open(filepath.Join("testdata", "CODE-OWNERS"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = owners.Close() }()
	roster, err := ParseCodeOwners(owners)
	if err != nil {
		t.Fatal(err)
	}
	now, err := ParseISO(goldenNow)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Analyze(prs, state, Options{
		OutPath: "rt_final.json",
		Top:     top,
		Now:     now,
		Roster:  Lowered(roster),
	})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// A --now carrying a 7th fractional digit below the mergedAt's used to floor
// ageDays one day short of Python's, dropping the PR into the 1.0 recency
// bucket and putting the wrong reviewer on top. testdata/microsecond/golden.*
// come from rt_analyze.py run on the same fixture with the same --now.
func TestSubMicrosecondNowRanksLikePython(t *testing.T) {
	now, err := ParseISO("2026-08-01T00:00:00.0000004Z")
	if err != nil {
		t.Fatal(err)
	}
	if now.Nanosecond() != 0 {
		t.Errorf("ParseISO kept %d ns, want microsecond truncation", now.Nanosecond())
	}

	dir := filepath.Join("testdata", "microsecond")
	prs, err := LoadPRs(filepath.Join(dir, "rt_prs.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	state, err := LoadState(filepath.Join(dir, "rt_fetch_state.json"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := Analyze(prs, state, Options{
		OutPath: "rt_final.json",
		Top:     15,
		Now:     now,
		Roster:  Lowered(ParseRoster("reviewer-a,reviewer-b")),
	})
	if err != nil {
		t.Fatal(err)
	}

	want := "  core: reviewer-b(0.75/3), reviewer-a(0.5/1)"
	if got := summaryLine(t, res, "core"); got != want {
		t.Errorf("core ranking = %q, want %q", got, want)
	}
	assertMatchesGolden(t, res, dir)
}

func summaryLine(t *testing.T, res *Result, area string) string {
	t.Helper()
	prefix := "  " + area + ":"
	for _, line := range res.Summary {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	t.Fatalf("no %q line in summary %q", area, res.Summary)
	return ""
}

func firstDiff(got, want string) string {
	i := 0
	for i < len(got) && i < len(want) && got[i] == want[i] {
		i++
	}
	from := max(i-80, 0)
	return "at byte " + strconv.Itoa(i) + "\ngot:  " + snippet(got, from) + "\nwant: " + snippet(want, from)
}

func snippet(s string, from int) string {
	to := min(from+160, len(s))
	return s[from:to]
}

// The ranking is where a port drifts silently, so the tie ladder is asserted
// directly as well as through the golden: weight, then raw, then lowercased
// login, then the order the reviewer was first seen.
func TestReviewerRankingTiebreaks(t *testing.T) {
	res := analyzeFixture(t, 15)
	want := map[string][]string{
		"object": {"bob", "zed", "carol", "grace"},
		"csi":    {"carol", "alice", "Alice", "bob"},
		"core":   {"Bob", "bob", "alice", "carol"},
		"build":  {"alice", "bob", "eve"},
	}
	for area, logins := range want {
		got := reviewerLogins(t, res, area)
		if strings.Join(got, ",") != strings.Join(logins, ",") {
			t.Errorf("%s reviewers = %v, want %v", area, got, logins)
		}
	}
}

func TestIdentityUnknownKeepsInsertionOrderOnRawTie(t *testing.T) {
	res := analyzeFixture(t, 15)
	got := flagsOfType(t, res, "identity-unknown")
	want := []string{"carol", "erin", "dave", "zed", "grace"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("identity-unknown order = %v, want %v", got, want)
	}
}

func TestRecentItemsSortDescendingWithNumberTiebreak(t *testing.T) {
	res := analyzeFixture(t, 15)
	got := recentNumbers(t, res, "ceph-mon")
	want := []int{119, 118, 123, 122, 121}
	if len(got) != len(want) {
		t.Fatalf("ceph-mon recent_items = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ceph-mon recent_items = %v, want %v", got, want)
		}
	}
}

func TestTopZeroYieldsCoverageGaps(t *testing.T) {
	res := analyzeFixture(t, 0)
	for _, area := range []string{"object", "csi", "core", "build"} {
		if logins := reviewerLogins(t, res, area); len(logins) != 0 {
			t.Errorf("--top 0 kept reviewers for %s: %v", area, logins)
		}
	}
	if items := flagsOfType(t, res, "coverage-gap"); len(items) < 4 {
		t.Errorf("--top 0 produced only %d coverage-gap flags: %v", len(items), items)
	}
}

func TestAreasFor(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"pkg/operator/ceph/object/zone.go", "object,object-multisite"},
		{"pkg/operator/ceph/object/cosi/driver.go", "object-cosi"},
		{"pkg/daemon/ceph/rgw/admin.go", "object"},
		{"pkg/operator/ceph/object/bucket/provisioner.go", "object,object-bucket-claims"},
		{"pkg/operator/ceph/cluster/mgr/dashboard.go", "ceph-dashboard,ceph-mgr"},
		{"pkg/operator/ceph/cluster/osd/osd.go", "ceph-osd"},
		{"pkg/daemon/ceph/osd/volume.go", "ceph-osd"},
		{"pkg/operator/ceph/pool/pool.go", "block"},
		{"pkg/operator/ceph/controller/rbdmirror.go", "block"},
		{"deploy/charts/rook-ceph/values.yaml", "helm"},
		{"Documentation/CRDs/cluster.md", "docs"},
		{"design/ceph/object/multisite.md", "design"},
		{".github/workflows/ci.yaml", "ci"},
		{"tests/scripts/helper.sh", "ci"},
		{"tests/integration/base.go", "test"},
		{"pkg/apis/ceph.rook.io/v1/types.go", "crd"},
		{"pkg/client/clientset/x.go", "crd"},
		{"pkg/operator/ceph/controller/network.go", "networking"},
		{"pkg/operator/ceph/nvmeof/nvmeof.go", "nvmeof"},
		{"pkg/operator/ceph/cluster/external.go", "ceph-external"},
		{"pkg/operator/discover/discover.go", "discover"},
		{"pkg/operator/ceph/reporting/reporting.go", "monitoring"},
		{"go.mod", "build"},
		{"pkg/apis/go.mod", "build"},
		{"pkg/apis/go.sum", "build"},
		{"images/ceph/Dockerfile", "build"},
		{"build/makelib/common.mk", "build"},
		{"Makefile", "build"},
		{".golangci.yml", "build"},
		{"pkg/util/exec/exec.go", "core"},
		{"cmd/rook/main.go", "core"},
		{"deploy/examples/cluster.yaml", ""},
		{"README.md", ""},
		{"somefile.txt", ""},
	}
	for _, tc := range tests {
		if got := strings.Join(sortedKeys(AreasFor(tc.path)), ","); got != tc.want {
			t.Errorf("AreasFor(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestIsBot(t *testing.T) {
	for _, login := range []string{
		"mergify", "mergify[bot]", "dependabot[bot]", "github-actions[bot]",
		"copilot-swe-agent", "Copilot", "renovate-bot", "someone[bot]",
	} {
		if !IsBot(login) {
			t.Errorf("IsBot(%q) = false", login)
		}
	}
	for _, login := range []string{"alice", "bob", "travisn", "abbot-nope"} {
		if IsBot(login) {
			t.Errorf("IsBot(%q) = true", login)
		}
	}
}

func TestParseCodeOwners(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "CODE-OWNERS"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	roster, err := ParseCodeOwners(f)
	if err != nil {
		t.Fatal(err)
	}
	want := "Bob,alice,dangling,eve,frank"
	if got := strings.Join(sortedKeys(roster), ","); got != want {
		t.Errorf("roster = %q, want %q", got, want)
	}
	if empty, err := ParseCodeOwners(strings.NewReader("nothing here\n")); err != nil {
		t.Fatal(err)
	} else if len(empty) != 0 {
		t.Errorf("roster from a file with no tiers = %v", empty)
	}
}

func TestParseRoster(t *testing.T) {
	got := ParseRoster(" alice, bob ,, eve,")
	if len(got) != 3 || !got["alice"] || !got["bob"] || !got["eve"] {
		t.Errorf("ParseRoster() = %v", got)
	}
}

func TestParseISO(t *testing.T) {
	for _, s := range []string{
		"2026-08-01T00:00:00Z",
		"2026-08-01T00:00:00+00:00",
		"2026-08-01T00:00:00.123456Z",
		"2026-08-01 00:00:00+00:00",
		"2026-08-01T05:30+05:30",
	} {
		got, err := ParseISO(s)
		if err != nil {
			t.Errorf("ParseISO(%q): %v", s, err)
			continue
		}
		if !got.Equal(time.Date(2026, 8, 1, 0, 0, 0, got.Nanosecond(), time.UTC)) {
			t.Errorf("ParseISO(%q) = %v", s, got)
		}
	}
	for _, s := range []string{"", "2026-08-01", "not-a-time", "2026-08-01T00:00:00"} {
		if _, err := ParseISO(s); err == nil {
			t.Errorf("ParseISO(%q) accepted a value Python cannot subtract", s)
		}
	}
}

func TestAgeDaysFloorsLikeTimedelta(t *testing.T) {
	base := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		merged time.Time
		want   int
	}{
		{base, 0},
		{base.Add(-24 * time.Hour), 1},
		{base.Add(time.Second), -1},
		{base.Add(86401 * time.Second), -2},
		{base.Add(time.Microsecond), -1},
		{base.Add(-86399 * time.Second), 0},
		{base.AddDate(0, 0, -182), 182},
		{base.AddDate(0, 0, -182).Add(-time.Second), 182},
	}
	for _, tc := range tests {
		if got := AgeDays(base, tc.merged); got != tc.want {
			t.Errorf("AgeDays(base, %v) = %d, want %d", tc.merged, got, tc.want)
		}
	}
}

func TestHeadLimit(t *testing.T) {
	tests := []struct{ n, length, want int }{
		{15, 3, 3}, {2, 5, 2}, {0, 5, 0}, {-2, 5, 3}, {-9, 5, 0}, {5, 5, 5},
	}
	for _, tc := range tests {
		if got := headLimit(tc.n, tc.length); got != tc.want {
			t.Errorf("headLimit(%d, %d) = %d, want %d", tc.n, tc.length, got, tc.want)
		}
	}
}

func TestLoadPRsDedupesKeepingFirstPosition(t *testing.T) {
	prs, err := LoadPRs(filepath.Join("testdata", "rt_prs.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int]bool{}
	for _, pr := range prs {
		if seen[pr.Number] {
			t.Fatalf("PR #%d appears twice after load", pr.Number)
		}
		seen[pr.Number] = true
	}
	if prs[0].Number != 100 {
		t.Errorf("first PR = #%d, want #100 (its first-seen position)", prs[0].Number)
	}
	if prs[0].Title != "object: fix zone sync (amended)" {
		t.Errorf("PR #100 title = %q, want the last record's title", prs[0].Title)
	}
}

func reviewerLogins(t *testing.T, res *Result, area string) []string {
	t.Helper()
	var out []string
	for _, rev := range member(t, member(t, member(t, res.Doc, "data").(Obj), "areas").(Obj), area).(Obj) {
		if rev.Key != "reviewers" {
			continue
		}
		for _, r := range rev.Val.([]any) {
			out = append(out, member(t, r.(Obj), "login").(string))
		}
	}
	return out
}

func recentNumbers(t *testing.T, res *Result, area string) []int {
	t.Helper()
	var out []int
	for _, entry := range member(t, member(t, member(t, res.Doc, "data").(Obj), "areas").(Obj), area).(Obj) {
		if entry.Key != "recent_items" {
			continue
		}
		for _, it := range entry.Val.([]any) {
			out = append(out, member(t, it.(Obj), "number").(int))
		}
	}
	return out
}

func flagsOfType(t *testing.T, res *Result, kind string) []string {
	t.Helper()
	var out []string
	for _, f := range member(t, res.Doc, "flags").([]any) {
		if member(t, f.(Obj), "type").(string) == kind {
			out = append(out, member(t, f.(Obj), "item").(string))
		}
	}
	return out
}

func member(t *testing.T, o Obj, key string) any {
	t.Helper()
	for _, m := range o {
		if m.Key == key {
			return m.Val
		}
	}
	t.Fatalf("no %q in object", key)
	return nil
}

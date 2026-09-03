package rtissues

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/actions"
	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtanalyze"
)

var update = flag.Bool("update", false, "rewrite testdata/golden.* from the current output")

const (
	goldenNow    = "2026-08-01T00:00:00Z"
	goldenMonths = 24
	goldenRoster = "alice,bob,carol,dave,erin,frank,grace,heidi,ivan"
)

func mineFixture(t *testing.T, roster string) *Result {
	t.Helper()
	md, err := os.ReadFile("testdata/label-map.md")
	if err != nil {
		t.Fatal(err)
	}
	byLabel, err := actions.ParseLabelAreas(md)
	if err != nil {
		t.Fatal(err)
	}
	issues, err := Load("testdata/rt_issues.json")
	if err != nil {
		t.Fatal(err)
	}
	now, err := rtanalyze.ParseISO(goldenNow)
	if err != nil {
		t.Fatal(err)
	}
	var names map[string]bool
	if roster != "" {
		names = rtanalyze.Lowered(rtanalyze.ParseRoster(roster))
	}
	res, err := Mine(issues, Options{
		Now: now, Months: goldenMonths, Areas: byLabel, Roster: names,
		OutPath: "rt_issues_final.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// The golden pins the whole document rather than a handful of counts: the kb
// refresh diffs this output between runs, and a login that slipped a rank or a
// provenance number that stopped counting is invisible in a spot check.
// testdata/rt_issues.json exercises the exclusions (the issue's author, a bot,
// a null author, a comment older than the window), an issue whose two labels
// name one area, an unlabelled issue and a labelled one whose label the table
// does not map, the 100-comment truncation, and a top participant outside the
// roster.
func TestGolden(t *testing.T) {
	res := mineFixture(t, goldenRoster)
	doc := rtanalyze.Marshal(res.Doc) + "\n"
	if *update {
		if err := os.WriteFile(filepath.Join("testdata", "golden.json"), []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join("testdata", "golden.txt"), []byte(res.Summary+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	assertGolden(t, "golden.json", doc)
	assertGolden(t, "golden.txt", res.Summary+"\n")
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	want, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("%s differs (re-run with -update to see the diff):\ngot:\n%s\nwant:\n%s", name, got, string(want))
	}
}

func TestNoRosterAsksNoIdentityQuestion(t *testing.T) {
	for _, f := range mineFixture(t, "").Flags {
		if f.Type == "identity-unknown" {
			t.Errorf("flagged %q with nothing to judge it against", f.Item)
		}
	}
}

// A login the export cannot spell is dropped from every count, so the mine has
// to say how many it dropped — once per login, not once per comment.
func TestSkippedLoginsCountsDistinctLogins(t *testing.T) {
	doc := rtanalyze.Marshal(mineFixture(t, goldenRoster).Doc)
	if !strings.Contains(doc, `"skipped_logins": 1`) {
		t.Errorf("the fixture drops two comments by one unusable login:\n%s", doc)
	}
}

func mineSynthetic(t *testing.T, issues []Issue, top int) map[string]any {
	t.Helper()
	now, err := rtanalyze.ParseISO(goldenNow)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Mine(issues, Options{Now: now, Months: goldenMonths, Areas: map[string][]string{"csi": {"csi"}}, Top: top})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(rtanalyze.Marshal(res.Doc)), &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func fullPage(now time.Time, logins ...string) []Comment {
	page := make([]Comment, CommentPage)
	for i := range page {
		page[i] = Comment{Login: logins[i%len(logins)], At: now.Add(-time.Hour)}
	}
	return page
}

// A truncated page on an issue nothing counted is noise, the way rtanalyze
// scopes its PR truncations: only the counted issue is flagged.
func TestTruncationScopedToCountedIssues(t *testing.T) {
	now, _ := rtanalyze.ParseISO(goldenNow)
	stale := fullPage(now, "alice")
	for i := range stale {
		stale[i].At = now.AddDate(-3, 0, 0)
	}
	doc := mineSynthetic(t, []Issue{
		{Number: 1, Author: "zed", CreatedAt: now, Comments: fullPage(now, "alice")},
		{Number: 2, Author: "zed", Labels: []string{"csi"}, CreatedAt: now, Comments: stale},
		{Number: 3, Author: "zed", Labels: []string{"csi"}, CreatedAt: now, Comments: fullPage(now, "alice")},
	}, 0)
	flags := doc["flags"].([]any)
	if len(flags) != 1 || flags[0].(map[string]any)["item"] != "issue #3" {
		t.Errorf("want one truncation flag for issue #3, got %v", flags)
	}
}

func TestTopCapsEachArea(t *testing.T) {
	now, _ := rtanalyze.ParseISO(goldenNow)
	doc := mineSynthetic(t, []Issue{
		{Number: 1, Author: "zed", Labels: []string{"csi"}, CreatedAt: now, Comments: fullPage(now, "alice", "bob", "carol")},
	}, 2)
	csi := doc["data"].(map[string]any)["areas"].(map[string]any)["csi"].(map[string]any)
	if len(csi) != 2 {
		t.Errorf("--top 2 kept %d logins: %v", len(csi), csi)
	}
}

package sweepprefetch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const summaryNow = "2026-08-11T00:00:00Z"

func intp(n int) *int { return &n }

func at(t *testing.T, stamp string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

// typedSnapshot writes the fixture through the same struct and encoder the
// fetch pass writes snapshot.json with, so a shape drift breaks these tests
// rather than hiding behind hand-written JSON.
func typedSnapshot(t *testing.T, kind string, items ...any) string {
	t.Helper()
	obj := NewObject()
	for _, item := range items {
		switch v := item.(type) {
		case *PRItem:
			obj.Set(strconv.Itoa(v.Number), v)
		case *IssueItem:
			obj.Set(strconv.Itoa(v.Number), v)
		default:
			t.Fatalf("unsupported item %T", item)
		}
	}
	data, err := Encode(snapshotFile{
		FetchedAt: "2026-08-11T00:00:00.000000+00:00",
		Repo:      "rook/rook",
		Kind:      kind,
		Items:     obj,
	})
	if err != nil {
		t.Fatal(err)
	}
	return rawSnapshot(t, string(data))
}

func rawSnapshot(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "snapshot.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func summarize(t *testing.T, dir, viewer string) *PoolSummary {
	t.Helper()
	s, err := Summarize(SummaryOptions{SweepDir: dir, Viewer: viewer, Now: at(t, summaryNow)})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// prPool is the fixture the markdown cases share: one PR per age bucket and
// per author class, including a draft, a bot and an account that no longer
// exists.
func prPool(t *testing.T) string {
	t.Helper()
	return typedSnapshot(t, "prs",
		&PRItem{
			Number: 10, State: "OPEN", CreatedAt: "2026-08-06T00:00:00Z",
			Author: "alice", AuthorAssociation: strp("MEMBER"),
			Additions: intp(1200), Deletions: intp(345), ChangedFiles: intp(12),
			Labels:  []string{"ceph", "object"},
			Reviews: Reviews{Latest: []Review{{Login: "jhoblitt", State: "APPROVED"}}},
		},
		&PRItem{
			Number: 11, State: "OPEN", IsDraft: true, CreatedAt: "2026-08-04T00:00:00Z",
			Author: "bob", AuthorAssociation: strp("CONTRIBUTOR"),
			Additions: intp(18), Deletions: intp(2), ChangedFiles: intp(2),
			Labels:  []string{"ceph"},
			Reviews: Reviews{Latest: []Review{{Login: "travisn", State: "COMMENTED"}}},
		},
		&PRItem{
			Number: 12, State: "OPEN", CreatedAt: "2026-07-01T00:00:00Z",
			Author: "mergify[bot]", AuthorAssociation: strp("CONTRIBUTOR"),
			Additions: intp(5), Deletions: intp(5), ChangedFiles: intp(1),
			Labels: []string{},
		},
		&PRItem{
			Number: 13, State: "OPEN", CreatedAt: "2026-01-01T00:00:00Z",
			Author: "", Labels: []string{"ceph", "docs"},
			Reviews: Reviews{Latest: []Review{{Login: "JHoblitt", State: "CHANGES_REQUESTED"}}},
		},
	)
}

func issuePool(t *testing.T) string {
	t.Helper()
	return typedSnapshot(t, "issues",
		&IssueItem{Number: 20, State: "OPEN", CreatedAt: "2026-08-10T00:00:00Z",
			Author: "travisn", Labels: []string{"bug"}, CommentsTotal: 3},
		&IssueItem{Number: 21, State: "OPEN", CreatedAt: "2026-05-01T00:00:00Z",
			Author: "", Labels: []string{}, CommentsTotal: 0},
		&IssueItem{Number: 22, State: "OPEN", CreatedAt: "2026-07-20T00:00:00Z",
			Author: "travisn", Labels: []string{"bug", "feature"}, CommentsTotal: 1205},
	)
}

func TestSummarizeMarkdown(t *testing.T) {
	for _, tc := range []struct {
		name   string
		dir    string
		viewer string
		want   string
	}{
		{
			name:   "prs with a viewer",
			dir:    prPool(t),
			viewer: "jhoblitt",
			want: `**Pool: 4 open PRs** (1 draft, 2 already carry a review from jhoblitt)

| Age | PRs |  | Author assoc | PRs |
|---|---|---|---|---|
| <7d | 1 |  | MEMBER | 1 |
| 7-30d | 1 |  | CONTRIBUTOR | 1 |
| 30-90d | 1 |  | bot | 1 |
| >90d | 1 |  | unknown | 1 |

Top authors: (deleted)(1) alice(1) bob(1) mergify[bot](1)
Top labels: ceph(3) docs(1) object(1) · 1 with no labels
Diff size: +1,223 / -352 over 15 files · diff size missing on 1 PR`,
		},
		{
			name: "prs without a viewer",
			dir:  prPool(t),
			want: `**Pool: 4 open PRs** (1 draft)

| Age | PRs |  | Author assoc | PRs |
|---|---|---|---|---|
| <7d | 1 |  | MEMBER | 1 |
| 7-30d | 1 |  | CONTRIBUTOR | 1 |
| 30-90d | 1 |  | bot | 1 |
| >90d | 1 |  | unknown | 1 |

Top authors: (deleted)(1) alice(1) bob(1) mergify[bot](1)
Top labels: ceph(3) docs(1) object(1) · 1 with no labels
Diff size: +1,223 / -352 over 15 files · diff size missing on 1 PR`,
		},
		{
			name: "issues",
			dir:  issuePool(t),
			want: `**Pool: 3 open issues**

| Age | Issues |  | Author | Issues |
|---|---|---|---|---|
| <7d | 1 |  | travisn | 2 |
| 7-30d | 1 |  | (deleted) | 1 |
| 30-90d | 0 |  |  |  |
| >90d | 1 |  |  |  |

Top labels: bug(2) feature(1) · 1 with no labels
Comments: 1,208 total · 1 issue with none`,
		},
		{
			name: "a corpus fetched by number is not all open",
			dir: typedSnapshot(t, "prs",
				&PRItem{Number: 1, State: "MERGED", CreatedAt: "2026-08-10T00:00:00Z",
					Author: "alice", AuthorAssociation: strp("MEMBER"),
					Additions: intp(1), Deletions: intp(0), ChangedFiles: intp(1),
					Labels: []string{"ceph"}},
				&PRItem{Number: 2, State: "OPEN", CreatedAt: "2026-08-10T00:00:00Z",
					Author: "alice", AuthorAssociation: strp("MEMBER"),
					Additions: intp(1), Deletions: intp(0), ChangedFiles: intp(1),
					Labels: []string{"ceph"}},
			),
			want: `**Pool: 2 PRs** (1 OPEN, 1 MERGED)

| Age | PRs |  | Author assoc | PRs |
|---|---|---|---|---|
| <7d | 2 |  | MEMBER | 2 |
| 7-30d | 0 |  |  |  |
| 30-90d | 0 |  |  |  |
| >90d | 0 |  |  |  |

Top authors: alice(2)
Top labels: ceph(2)
Diff size: +2 / -0 over 2 files`,
		},
		{
			name: "a pool of one",
			dir: typedSnapshot(t, "issues",
				&IssueItem{Number: 1, State: "OPEN", CreatedAt: "2026-08-10T00:00:00Z",
					Author: "travisn", Labels: []string{}, CommentsTotal: 1},
			),
			want: `**Pool: 1 open issue**

| Age | Issues |  | Author | Issues |
|---|---|---|---|---|
| <7d | 1 |  | travisn | 1 |
| 7-30d | 0 |  |  |  |
| 30-90d | 0 |  |  |  |
| >90d | 0 |  |  |  |

Top labels: none · 1 with no labels
Comments: 1 total`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := summarize(t, tc.dir, tc.viewer).Markdown(); got != tc.want {
				t.Errorf("Markdown() =\n%s\n\nwant\n%s", got, tc.want)
			}
		})
	}
}

// An absent --viewer and a viewer nobody has reviewed for are different facts,
// and a phase 0 that reads "0 reviewed" when it never asked would go looking
// for work it has already done.
func TestSummarizeViewerAbsentVersusZero(t *testing.T) {
	dir := prPool(t)
	for _, tc := range []struct {
		name   string
		viewer string
		want   *int
	}{
		{"absent", "", nil},
		{"zero", "nobody", intp(0)},
		{"counted, case-insensitively", "JHOBLITT", intp(2)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := summarize(t, dir, tc.viewer)
			switch {
			case tc.want == nil && s.ReviewedByViewer != nil:
				t.Fatalf("reviewed_by_viewer = %d, want it absent", *s.ReviewedByViewer)
			case tc.want != nil && s.ReviewedByViewer == nil:
				t.Fatalf("reviewed_by_viewer is absent, want %d", *tc.want)
			case tc.want != nil && *s.ReviewedByViewer != *tc.want:
				t.Fatalf("reviewed_by_viewer = %d, want %d", *s.ReviewedByViewer, *tc.want)
			}

			data, err := s.JSON()
			if err != nil {
				t.Fatal(err)
			}
			var doc map[string]json.RawMessage
			if err := json.Unmarshal(data, &doc); err != nil {
				t.Fatal(err)
			}
			_, present := doc["reviewed_by_viewer"]
			if present != (tc.want != nil) {
				t.Errorf("reviewed_by_viewer present = %v in %s", present, data)
			}
			if _, present := doc["viewer"]; present != (tc.viewer != "") {
				t.Errorf("viewer present = %v in %s", present, data)
			}
		})
	}
}

// The buckets are half-open, so an item that is exactly 7 days old belongs to
// 7-30d and not to <7d. now is pinned for exactly this reason.
func TestSummarizeAgeBucketBoundaries(t *testing.T) {
	dir := typedSnapshot(t, "issues",
		&IssueItem{Number: 1, CreatedAt: "2026-08-11T01:00:00Z", State: "OPEN"},
		&IssueItem{Number: 2, CreatedAt: "2026-08-04T00:00:01Z", State: "OPEN"},
		&IssueItem{Number: 3, CreatedAt: "2026-08-04T00:00:00Z", State: "OPEN"},
		&IssueItem{Number: 4, CreatedAt: "2026-07-12T00:00:01Z", State: "OPEN"},
		&IssueItem{Number: 5, CreatedAt: "2026-07-12T00:00:00Z", State: "OPEN"},
		&IssueItem{Number: 6, CreatedAt: "2026-05-13T00:00:01Z", State: "OPEN"},
		&IssueItem{Number: 7, CreatedAt: "2026-05-13T00:00:00Z", State: "OPEN"},
	)
	want := []Bucket{{"<7d", 2}, {"7-30d", 2}, {"30-90d", 2}, {">90d", 1}}
	got := summarize(t, dir, "").Age
	if len(got) != len(want) {
		t.Fatalf("age = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("age[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// Nothing here may be inferred from a flag: the corpus decides which
// aggregates exist, and an issues snapshot supports neither reviews nor
// authorAssociation nor a diff.
func TestSummarizeFollowsTheSnapshotKind(t *testing.T) {
	prs := summarize(t, prPool(t), "jhoblitt")
	if prs.Diff == nil || prs.AuthorAssociation == nil || prs.Comments != nil {
		t.Errorf("prs summary = %+v, want a diff and an association axis and no comments", prs)
	}
	if prs.Drafts == nil || *prs.Drafts != 1 {
		t.Errorf("drafts = %v, want 1", prs.Drafts)
	}
	issues := summarize(t, issuePool(t), "")
	if issues.Diff != nil || issues.AuthorAssociation != nil || issues.Comments == nil {
		t.Errorf("issues summary = %+v, want comments and neither a diff nor an association axis", issues)
	}
	if issues.Drafts != nil {
		t.Errorf("drafts = %d on an issues corpus, want it absent", *issues.Drafts)
	}
	if issues.Comments.Total != 1208 || issues.Comments.None != 1 {
		t.Errorf("comments = %+v, want 1208 total and 1 with none", *issues.Comments)
	}
}

// An authorAssociation that is null, or a key the fetch pass predates and never
// wrote at all, has to bucket as unknown rather than as a blank row.
func TestSummarizeBucketsMissingAssociations(t *testing.T) {
	dir := rawSnapshot(t, `{"fetched_at": "2026-08-11T00:00:00.000000+00:00",
 "repo": "rook/rook", "kind": "prs", "items": {
 "1": {"number": 1, "state": "OPEN", "createdAt": "2026-08-10T00:00:00Z",
       "author": "alice", "authorAssociation": null, "labels": []},
 "2": {"number": 2, "state": "OPEN", "createdAt": "2026-08-10T00:00:00Z",
       "author": "bob", "labels": []},
 "3": {"number": 3, "state": "OPEN", "createdAt": "2026-08-10T00:00:00Z",
       "author": "", "labels": []},
 "4": {"number": 4, "state": "OPEN", "createdAt": "2026-08-10T00:00:00Z",
       "author": "copilot-pull-request-reviewer", "authorAssociation": "NONE", "labels": []}}}`)
	s := summarize(t, dir, "")
	want := []Bucket{{botLabel, 1}, {unknownLabel, 3}}
	if len(s.AuthorAssociation) != len(want) {
		t.Fatalf("author_association = %+v, want %+v", s.AuthorAssociation, want)
	}
	for i := range want {
		if s.AuthorAssociation[i] != want[i] {
			t.Errorf("author_association[%d] = %+v, want %+v", i, s.AuthorAssociation[i], want[i])
		}
	}
	if s.Diff.Missing != 4 {
		t.Errorf("diff.missing = %d, want 4: none of these items carries a diff", s.Diff.Missing)
	}
	if got := s.TopAuthors[0]; got.Label != deletedAuthor || got.Count != 1 {
		t.Errorf("top_authors[0] = %+v, want the deleted account counted explicitly", got)
	}
}

func TestSummarizeCapsTopLists(t *testing.T) {
	items := make([]any, 0, 12)
	for n := 1; n <= 12; n++ {
		items = append(items, &IssueItem{
			Number: n, State: "OPEN", CreatedAt: "2026-08-10T00:00:00Z",
			Author: "author-" + strconv.Itoa(n),
			Labels: []string{"label-" + strconv.Itoa(n)},
		})
	}
	s := summarize(t, typedSnapshot(t, "issues", items...), "")
	if len(s.TopAuthors) != topN || s.AuthorsDistinct != 12 {
		t.Errorf("authors = %d with %d shown, want 12 with %d shown", s.AuthorsDistinct, len(s.TopAuthors), topN)
	}
	if len(s.TopLabels) != topN || s.LabelsDistinct != 12 {
		t.Errorf("labels = %d with %d shown, want 12 with %d shown", s.LabelsDistinct, len(s.TopLabels), topN)
	}
	if !strings.Contains(s.Markdown(), "+4 more") {
		t.Errorf("markdown does not report the names the cap hid:\n%s", s.Markdown())
	}
	// On an issues snapshot the authors axis is the table column, not the
	// prs-only "Top authors:" line, so the "+N more" above is the LABELS
	// disclosure and says nothing about authors. The header carries theirs.
	if !strings.Contains(s.Markdown(), "Author (8 of 12)") {
		t.Errorf("markdown shows 8 of 12 authors without saying so:\n%s", s.Markdown())
	}
}

// A pull request pool with no drafts in it still reports the zero, because a
// sweep sizing its fan-out has to tell "none of these are drafts" apart from
// "nobody counted". Only the markdown suppresses it.
func TestSummarizeReportsZeroDrafts(t *testing.T) {
	dir := typedSnapshot(t, "prs",
		&PRItem{Number: 1, State: "OPEN", CreatedAt: "2026-08-10T00:00:00Z",
			Author: "alice", AuthorAssociation: strp("MEMBER"),
			Additions: intp(1), Deletions: intp(0), ChangedFiles: intp(1)},
	)
	s := summarize(t, dir, "")
	if s.Drafts == nil || *s.Drafts != 0 {
		t.Fatalf("drafts = %v, want 0", s.Drafts)
	}
	if strings.Contains(s.Markdown(), "draft") {
		t.Errorf("markdown mentions drafts when there are none:\n%s", s.Markdown())
	}
	if !strings.Contains(string(mustJSON(t, s)), `"drafts": 0`) {
		t.Errorf("json dropped the zero draft count:\n%s", mustJSON(t, s))
	}
}

// Step 3 of a sweep re-summarizes the pool that survived the maintainer's
// filters, so the block has to say what the subset was drawn from; read as the
// whole corpus it would understate the pool by however much the filter removed.
func TestSummarizeSelectsNumbers(t *testing.T) {
	s, err := Summarize(SummaryOptions{
		SweepDir: prPool(t), Viewer: "jhoblitt", Now: at(t, summaryNow),
		Numbers: []int{13, 11},
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Total != 2 || s.SelectedFrom != 4 {
		t.Errorf("total = %d of %d, want 2 of 4", s.Total, s.SelectedFrom)
	}
	if s.Drafts == nil || *s.Drafts != 1 || s.ReviewedByViewer == nil || *s.ReviewedByViewer != 1 {
		t.Errorf("drafts = %v and reviewed = %v, want 1 and 1", s.Drafts, s.ReviewedByViewer)
	}
	want := "**Pool: 2 open PRs** (of 4 in the snapshot, 1 draft, 1 already carries a review from jhoblitt)"
	if got, _, _ := strings.Cut(s.Markdown(), "\n"); got != want {
		t.Errorf("header = %q, want %q", got, want)
	}

	whole := summarize(t, prPool(t), "")
	if whole.SelectedFrom != 0 {
		t.Errorf("selected_from = %d on an unfiltered summary, want it absent", whole.SelectedFrom)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(mustJSON(t, whole), &doc); err != nil {
		t.Fatal(err)
	}
	if _, present := doc["selected_from"]; present {
		t.Errorf("json carries selected_from on an unfiltered summary:\n%s", mustJSON(t, whole))
	}
}

// A number the snapshot never had means the caller and the tool disagree about
// what the pool is. Summarizing the rest would answer a question nobody asked.
func TestSummarizeRejectsMissingNumbers(t *testing.T) {
	for _, tc := range []struct {
		name    string
		numbers []int
		want    string
	}{
		{"a few", []int{10, 99, 4242}, "2 of the 3 requested items are not in the snapshot: 99, 4242"},
		{
			name:    "more than the error lists",
			numbers: []int{101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112},
			want:    "12 of the 12 requested items are not in the snapshot: 101, 102, 103, 104, 105, 106, 107, 108, 109, 110 and 2 more",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Summarize(SummaryOptions{
				SweepDir: prPool(t), Now: at(t, summaryNow), Numbers: tc.numbers,
			})
			if err == nil {
				t.Fatal("Summarize() succeeded on numbers the snapshot does not carry")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want it to say %q", err, tc.want)
			}
		})
	}
}

func mustJSON(t *testing.T, s *PoolSummary) []byte {
	t.Helper()
	data, err := s.JSON()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// A snapshot nobody could read or parse is an error at every step. Reporting
// zeros instead would tell a sweep its pool is empty.
func TestSummarizeRejectsBadSnapshots(t *testing.T) {
	for _, tc := range []struct {
		name, body, viewer, want string
	}{
		{name: "unparseable", body: `{"kind": "prs", "items"`, want: "unexpected end of JSON input"},
		{name: "no kind", body: `{"items": {"1": {}}}`, want: `kind is "", want prs or issues`},
		{name: "unknown kind", body: `{"kind": "pulls", "items": {"1": {}}}`, want: `kind is "pulls"`},
		{name: "no items", body: `{"kind": "prs"}`, want: "no items"},
		{name: "empty items", body: `{"kind": "prs", "items": {}}`, want: "no items"},
		{name: "items are not a map", body: `{"kind": "prs", "items": []}`, want: "cannot unmarshal array"},
		{name: "item is not an object", body: `{"kind": "prs", "items": {"7": 7}}`, want: "item 7:"},
		{
			name: "item has no createdAt",
			body: `{"kind": "prs", "items": {"7": {"number": 7}}}`,
			want: `item 7: createdAt ""`,
		},
		{
			name: "createdAt is not RFC3339",
			body: `{"kind": "issues", "items": {"7": {"number": 7, "createdAt": "2026/08/10"}}}`,
			want: `item 7: createdAt "2026/08/10"`,
		},
		{
			name:   "a viewer's reviews on a corpus that has none",
			body:   `{"kind": "issues", "items": {"7": {"number": 7, "createdAt": "2026-08-10T00:00:00Z"}}}`,
			viewer: "jhoblitt",
			want:   `kind is "issues", so nothing in it can carry a review by jhoblitt`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := rawSnapshot(t, tc.body)
			_, err := Summarize(SummaryOptions{SweepDir: dir, Viewer: tc.viewer, Now: at(t, summaryNow)})
			if err == nil {
				t.Fatalf("Summarize() succeeded on %s", tc.body)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want it to mention %q", err, tc.want)
			}
			if !strings.Contains(err.Error(), filepath.Join(dir, "snapshot.json")) {
				t.Errorf("error = %v, want it to name the snapshot", err)
			}
		})
	}
}

func TestSummarizeRejectsAMissingSnapshot(t *testing.T) {
	dir := t.TempDir()
	_, err := Summarize(SummaryOptions{SweepDir: dir, Now: at(t, summaryNow)})
	if err == nil {
		t.Fatal("Summarize() succeeded with no snapshot.json")
	}
	if !strings.Contains(err.Error(), filepath.Join(dir, "snapshot.json")) {
		t.Errorf("error = %v, want it to name the snapshot it wanted", err)
	}
}

func TestComma(t *testing.T) {
	for _, tc := range []struct {
		n    int
		want string
	}{{0, "0"}, {7, "7"}, {999, "999"}, {1000, "1,000"}, {18402, "18,402"},
		{1234567, "1,234,567"}, {-7551, "-7,551"}} {
		if got := comma(tc.n); got != tc.want {
			t.Errorf("comma(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

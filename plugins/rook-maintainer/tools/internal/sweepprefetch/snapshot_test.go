package sweepprefetch

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The golden files and testdata/queries-prs.json were produced by running the
// Python implementation this package replaces against these same fixtures, so
// a mismatch here is a real divergence in the format its consumers parse.

type stubGQL struct {
	t         *testing.T
	responses [][]byte
	queries   []string
}

func newStub(t *testing.T, fixtures ...string) *stubGQL {
	t.Helper()
	s := &stubGQL{t: t}
	for _, f := range fixtures {
		data, err := os.ReadFile(filepath.Join("testdata", f))
		if err != nil {
			t.Fatal(err)
		}
		s.responses = append(s.responses, data)
	}
	return s
}

func (s *stubGQL) run(_ context.Context, query string, out any) error {
	s.queries = append(s.queries, query)
	if len(s.responses) == 0 {
		return fmt.Errorf("unexpected query: %s", query)
	}
	resp := s.responses[0]
	s.responses = s.responses[1:]
	return json.Unmarshal(resp, out)
}

func (s *stubGQL) done() {
	s.t.Helper()
	if len(s.responses) != 0 {
		s.t.Errorf("%d fixtures went unused", len(s.responses))
	}
}

func testClient(t *testing.T, stub *stubGQL) *Client {
	t.Helper()
	c, err := NewClient("rook/rook")
	if err != nil {
		t.Fatal(err)
	}
	c.GraphQL = stub.run
	c.Now = func() time.Time { return time.Date(2026, 8, 10, 12, 0, 0, 123456000, time.UTC) }
	return c
}

func snapshotInto(t *testing.T, c *Client, opts SnapshotOptions) (SnapshotResult, string) {
	t.Helper()
	opts.SweepDir = t.TempDir()
	res, err := c.Snapshot(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(opts.SweepDir, "snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	return res, string(got)
}

func golden(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSnapshotPRs(t *testing.T) {
	stub := newStub(t, "open-prs-page1.json", "open-prs-page2.json",
		"contexts-page1.json", "contexts-page2.json",
		"files-page1.json", "files-page2.json")
	res, got := snapshotInto(t, testClient(t, stub), SnapshotOptions{Kind: "prs"})
	stub.done()

	if want := golden(t, "snapshot-prs.golden.json"); got != want {
		t.Errorf("snapshot.json =\n%s\nwant\n%s", got, want)
	}
	if res.Count != 2 || len(res.Truncated) != 0 {
		t.Errorf("result = %+v, want 2 items and nothing truncated", res)
	}
	if want := "2 prs -> " + res.Path; res.String() != want {
		t.Errorf("String() = %q, want %q", res.String(), want)
	}

	var wantQueries []string
	if err := json.Unmarshal([]byte(golden(t, "queries-prs.json")), &wantQueries); err != nil {
		t.Fatal(err)
	}
	if len(stub.queries) != len(wantQueries) {
		t.Fatalf("sent %d queries, want %d", len(stub.queries), len(wantQueries))
	}
	for i := range wantQueries {
		if stub.queries[i] != wantQueries[i] {
			t.Errorf("query %d =\n%s\nwant\n%s", i, stub.queries[i], wantQueries[i])
		}
	}
}

// A cursor walk can serve the same PR on two pages. The duplicates have to
// collapse before the per-item re-fetch, or the PR is re-fetched twice and the
// copy that lands in snapshot.json is the second one — here a green rollup
// that hides the red one Python recorded.
func TestSnapshotPRsDedupesBeforeRefetch(t *testing.T) {
	stub := newStub(t, "dup-prs-page1.json", "dup-prs-page2.json",
		"dup-contexts-failing.json", "dup-contexts-passing.json")
	res, got := snapshotInto(t, testClient(t, stub), SnapshotOptions{Kind: "prs"})

	if len(stub.queries) != 3 {
		t.Errorf("sent %d queries, want 3: two PR pages and one contexts re-fetch", len(stub.queries))
	}
	if res.Count != 1 || len(res.Truncated) != 0 {
		t.Errorf("result = %+v, want 1 item and nothing truncated", res)
	}

	var file struct {
		Items map[string]PRItem `json:"items"`
	}
	if err := json.Unmarshal([]byte(got), &file); err != nil {
		t.Fatal(err)
	}
	if len(file.Items) != 1 {
		t.Fatalf("snapshot.json carried %d items, want 1:\n%s", len(file.Items), got)
	}
	item, ok := file.Items["88"]
	if !ok {
		t.Fatalf("snapshot.json has no item 88:\n%s", got)
	}
	if item.CI.State == nil || *item.CI.State != "FAILURE" || item.CI.Failing != 1 || item.CI.Passing != 0 {
		t.Errorf("ci = %+v, want the FAILURE rollup of the single re-fetch", item.CI)
	}
	// The last node of a duplicate pair wins, matching the dict the reference
	// implementation built the items map with.
	if item.UpdatedAt != "2026-08-03T11:30:00Z" {
		t.Errorf("updatedAt = %q, want the second page's node", item.UpdatedAt)
	}
}

// A pullRequest that resolves to null must abort the snapshot. Left to
// unmarshal into a zero value it reads as an empty page with no next page,
// which would overwrite real data with "no files, pagination complete".
func TestSnapshotPRsRejectsNullPullRequest(t *testing.T) {
	for _, tc := range []struct{ name, list, null string }{
		{"files", "trunc-files-pr.json", "null-pull-request.json"},
		{"files, null repository", "trunc-files-pr.json", "null-repository.json"},
		{"contexts", "trunc-contexts-pr.json", "null-pull-request.json"},
		{"contexts, null repository", "trunc-contexts-pr.json", "null-repository.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := newStub(t, tc.list, tc.null)
			dir := t.TempDir()
			_, err := testClient(t, stub).Snapshot(context.Background(),
				SnapshotOptions{SweepDir: dir, Kind: "prs"})
			if err == nil {
				t.Fatal("Snapshot() succeeded on a null pullRequest")
			}
			if !strings.Contains(err.Error(), "repository.pullRequest") {
				t.Errorf("error = %v, want it to name the missing repository.pullRequest", err)
			}
			if _, err := os.Stat(filepath.Join(dir, "snapshot.json")); err == nil {
				t.Error("a null pullRequest still wrote snapshot.json")
			}
			stub.done()
		})
	}
}

func TestSnapshotIssues(t *testing.T) {
	stub := newStub(t, "open-issues.json")
	res, got := snapshotInto(t, testClient(t, stub), SnapshotOptions{Kind: "issues"})
	stub.done()

	if want := golden(t, "snapshot-issues.golden.json"); got != want {
		t.Errorf("snapshot.json =\n%s\nwant\n%s", got, want)
	}
	if res.Count != 2 {
		t.Errorf("Count = %d, want 2", res.Count)
	}
}

// Numbers mode also has to survive an alias that resolves to null: a number
// the maintainer typed that does not exist is dropped, not fatal.
func TestSnapshotByNumberSkipsMissingItems(t *testing.T) {
	stub := newStub(t, "numbers-issues.json")
	res, got := snapshotInto(t, testClient(t, stub),
		SnapshotOptions{Kind: "issues", ByNumber: true, Numbers: []int{4, 30}})
	stub.done()

	if want := golden(t, "snapshot-numbers-issues.golden.json"); got != want {
		t.Errorf("snapshot.json =\n%s\nwant\n%s", got, want)
	}
	if res.Count != 1 {
		t.Errorf("Count = %d, want 1", res.Count)
	}
	if q := stub.queries[0]; !strings.Contains(q, "n4: issue(number: 4)") ||
		!strings.Contains(q, "n30: issue(number: 30)") {
		t.Errorf("aliases missing from query:\n%s", q)
	}
}

func TestSnapshotByNumberChunksPRs(t *testing.T) {
	stub := &stubGQL{t: t, responses: [][]byte{
		[]byte(`{"repository": {}}`), []byte(`{"repository": {}}`),
	}}
	numbers := make([]int, 0, 20)
	for n := 1; n <= 20; n++ {
		numbers = append(numbers, n)
	}
	snapshotInto(t, testClient(t, stub), SnapshotOptions{Kind: "prs", ByNumber: true, Numbers: numbers})
	stub.done()

	if len(stub.queries) != 2 {
		t.Fatalf("sent %d queries, want 2", len(stub.queries))
	}
	for i, want := range []int{15, 5} {
		if got := strings.Count(stub.queries[i], ": pullRequest(number:"); got != want {
			t.Errorf("query %d carried %d aliases, want %d", i, got, want)
		}
	}
	if !strings.Contains(stub.queries[1], "n16: pullRequest(number: 16)") {
		t.Errorf("second chunk did not start at 16:\n%s", stub.queries[1])
	}
}

func TestSnapshotReportsFetchFailure(t *testing.T) {
	stub := &stubGQL{t: t}
	c := testClient(t, stub)
	dir := t.TempDir()
	if _, err := c.Snapshot(context.Background(), SnapshotOptions{SweepDir: dir, Kind: "prs"}); err == nil {
		t.Fatal("Snapshot() succeeded with no response")
	}
	if _, err := os.Stat(filepath.Join(dir, "snapshot.json")); err == nil {
		t.Error("a failed fetch still wrote snapshot.json")
	}
}

func TestNewClientRejectsBadRepo(t *testing.T) {
	for _, repo := range []string{"rook", "", "/rook", "rook/"} {
		if _, err := NewClient(repo); err == nil {
			t.Errorf("NewClient(%q) succeeded", repo)
		}
	}
}

func TestSnapshotResultReportsTruncation(t *testing.T) {
	res := SnapshotResult{Path: "/s/snapshot.json", Kind: "prs", Count: 3,
		Truncated: []string{"11", "12"}}
	want := "3 prs -> /s/snapshot.json (truncated fields on: ['11', '12'])"
	if res.String() != want {
		t.Errorf("String() = %q, want %q", res.String(), want)
	}
}

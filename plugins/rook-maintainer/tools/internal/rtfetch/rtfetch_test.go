package rtfetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The walk is pinned to a fixed clock so the cutoff is exactly
// 2026-01-01 minus round(24 * 30.44) = 731 days.
var fixedNow = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

const (
	line101 = `{"number":101,"title":"osd: guard against a nil <spec>",` +
		`"mergedAt":"2025-06-01T10:00:00Z","updatedAt":"2025-06-02T11:30:00Z",` +
		`"author":{"login":"alice"},` +
		`"files":{"pageInfo":{"hasNextPage":false},"nodes":[{"path":"pkg/operator/ceph/cluster/osd/osd.go"}]},` +
		`"reviews":{"pageInfo":{"hasNextPage":false},"nodes":[{"author":{"login":"bob"},"state":"APPROVED"}]}}`

	line102 = `{"number":102,"title":"object: sweeping refactor",` +
		`"mergedAt":"2025-05-01T09:00:00Z","updatedAt":"2025-05-02T09:00:00Z",` +
		`"author":null,` +
		`"files":{"pageInfo":{"hasNextPage":true},"nodes":[{"path":"pkg/operator/ceph/object/rgw.go"}]},` +
		`"reviews":{"pageInfo":{"hasNextPage":true},"nodes":[]}}`

	line103 = `{"number":103,"title":"csi: bump the sidecar",` +
		`"mergedAt":"2024-06-01T09:00:00Z","updatedAt":"2025-12-31T23:00:00Z",` +
		`"author":{"login":"carol"},` +
		`"files":{"pageInfo":{"hasNextPage":false},"nodes":[{"path":"pkg/operator/ceph/csi/spec.go"}]},` +
		`"reviews":{"pageInfo":{"hasNextPage":false},"nodes":[` +
		`{"author":{"login":"dave"},"state":"COMMENTED"},{"author":null,"state":"APPROVED"}]}}`
)

type reply struct {
	data string
	err  error
}

type stub struct {
	t       *testing.T
	replies []reply
	queries []string
	naps    []time.Duration
}

func (s *stub) query(_ context.Context, q string, out any) error {
	s.queries = append(s.queries, q)
	if len(s.replies) == 0 {
		s.t.Fatalf("unexpected query #%d: %s", len(s.queries), q)
	}
	r := s.replies[0]
	s.replies = s.replies[1:]
	if r.err != nil {
		return r.err
	}
	return json.Unmarshal([]byte(r.data), out)
}

func (s *stub) sleep(d time.Duration) { s.naps = append(s.naps, d) }

func fixture(t *testing.T, name string) reply {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	return reply{data: string(b)}
}

func newTestFetcher(opts Options, s *stub) *Fetcher {
	f := New(opts)
	f.Query = s.query
	f.Log = io.Discard
	f.Now = func() time.Time { return fixedNow }
	f.Sleep = s.sleep
	return f
}

func runWalkErr(t *testing.T, opts Options, replies ...reply) (*stub, string, error) {
	t.Helper()
	if opts.OutDir == "" {
		opts.OutDir = t.TempDir()
	}
	if opts.Repo == "" {
		opts.Repo = "rook/rook"
	}
	if opts.Months == 0 {
		opts.Months = 24
	}
	if opts.Cap == 0 {
		opts.Cap = 100
	}
	if opts.PageSize == 0 {
		opts.PageSize = 2
	}
	if opts.MaxPages == 0 {
		opts.MaxPages = 10
	}
	s := &stub{t: t, replies: replies}
	return s, opts.OutDir, newTestFetcher(opts, s).Run(context.Background())
}

func runWalk(t *testing.T, opts Options, replies ...reply) (*stub, string) {
	t.Helper()
	s, dir, err := runWalkErr(t, opts, replies...)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return s, dir
}

func readLines(t *testing.T, dir string) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, PRsFile))
	if err != nil {
		t.Fatalf("reading %s: %v", PRsFile, err)
	}
	if len(b) == 0 {
		return nil
	}
	if b[len(b)-1] != '\n' {
		t.Errorf("%s does not end in a newline", PRsFile)
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

func readStateFile(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, StateFile))
	if err != nil {
		t.Fatalf("reading %s: %v", StateFile, err)
	}
	return string(b)
}

func decodeState(t *testing.T, dir string) State {
	t.Helper()
	var st State
	if err := json.Unmarshal([]byte(readStateFile(t, dir)), &st); err != nil {
		t.Fatalf("decoding %s: %v", StateFile, err)
	}
	return st
}

func wantLines(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d JSONL lines, want %d:\n%s", len(got), len(want), strings.Join(got, "\n"))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d:\n got %s\nwant %s", i, got[i], want[i])
		}
	}
}

func TestWalkEmitsJSONLAndState(t *testing.T) {
	s, dir := runWalk(t, Options{},
		fixture(t, "page1.json"), fixture(t, "page2.json"), fixture(t, "page3.json"))

	wantLines(t, readLines(t, dir), []string{line101, line102, line103})

	want := `{
  "repo": "rook/rook",
  "pages_fetched": 3,
  "counted": 3,
  "seen_total": 5,
  "oldest_mergedat": "2024-06-01T09:00:00+00:00",
  "cutoff": "2024-01-01T00:00:00+00:00",
  "cap": 100,
  "stop_reason": "full page entirely updatedAt-older than cutoff",
  "errors": [],
  "truncations": [
    {
      "number": 102,
      "kind": "files",
      "mergedAt": "2025-05-01T09:00:00Z"
    },
    {
      "number": 102,
      "kind": "reviews",
      "mergedAt": "2025-05-01T09:00:00Z"
    }
  ]
}`
	if got := readStateFile(t, dir); got != want {
		t.Errorf("state file:\n got %s\nwant %s", got, want)
	}
	if len(s.naps) != 0 {
		t.Errorf("slept %v with a healthy rate limit", s.naps)
	}
}

func TestWalkThreadsCursors(t *testing.T) {
	s, _ := runWalk(t, Options{},
		fixture(t, "page1.json"), fixture(t, "page2.json"), fixture(t, "page3.json"))

	if len(s.queries) != 3 {
		t.Fatalf("issued %d queries, want 3", len(s.queries))
	}
	if strings.Contains(s.queries[0], "after:") {
		t.Errorf("first query carried a cursor: %s", s.queries[0])
	}
	for i, want := range []string{`after: "CURSOR1"`, `after: "CURSOR2"`} {
		if !strings.Contains(s.queries[i+1], want) {
			t.Errorf("query %d missing %s:\n%s", i+1, want, s.queries[i+1])
		}
	}
	if !strings.Contains(s.queries[0], "first: 2") {
		t.Errorf("query did not honor --page-size:\n%s", s.queries[0])
	}
	for _, want := range []string{"files(first: 100)", "reviews(first: 30)",
		"states: MERGED", "field: UPDATED_AT, direction: DESC"} {
		if !strings.Contains(s.queries[0], want) {
			t.Errorf("query missing %q:\n%s", want, s.queries[0])
		}
	}
}

// The cap breaks mid-page: the PRs after it must be left entirely unseen, not
// merely uncounted, so a later run can still flag their truncations.
func TestWalkStopsAtCap(t *testing.T) {
	_, dir := runWalk(t, Options{Cap: 1}, fixture(t, "page1.json"))

	wantLines(t, readLines(t, dir), []string{line101})
	st := decodeState(t, dir)
	if st.Counted != 1 || st.SeenTotal != 1 {
		t.Errorf("counted=%d seen_total=%d, want 1/1", st.Counted, st.SeenTotal)
	}
	if len(st.Truncations) != 0 {
		t.Errorf("truncations = %v, want none (PR 102 was never reached)", st.Truncations)
	}
	if got := deref(st.StopReason); got != "reached --cap=1" {
		t.Errorf("stop_reason = %q", got)
	}
}

func TestWalkFlagsTruncationOutsideTheWindow(t *testing.T) {
	page := `{"repository":{"pullRequests":{"pageInfo":{"hasNextPage":false,"endCursor":"X"},
	  "nodes":[{"number":7,"title":"old","mergedAt":"2022-01-01T00:00:00Z",
	    "updatedAt":"2025-01-01T00:00:00Z","author":{"login":"a"},
	    "files":{"pageInfo":{"hasNextPage":true},"nodes":[]},
	    "reviews":{"pageInfo":{"hasNextPage":false},"nodes":[]}}]}},
	  "rateLimit":{"cost":1,"remaining":4000,"resetAt":"2026-01-01T01:00:00Z"}}`
	_, dir := runWalk(t, Options{}, reply{data: page})

	if got := readLines(t, dir); len(got) != 0 {
		t.Errorf("out-of-window PR was written: %v", got)
	}
	st := decodeState(t, dir)
	if len(st.Truncations) != 1 || st.Truncations[0].Number != 7 ||
		st.Truncations[0].Kind != "files" {
		t.Fatalf("truncations = %+v", st.Truncations)
	}
	if st.OldestMergedAt != nil {
		t.Errorf("oldest_mergedat = %q, want null", *st.OldestMergedAt)
	}
	if got := deref(st.StopReason); got != "no more pages (end of merged PR history)" {
		t.Errorf("stop_reason = %q", got)
	}
}

// A stale page shorter than --page-size is the last page of history, not
// evidence that the window ran out, so it must not trip the stale-page stop.
func TestWalkShortStalePageDoesNotStop(t *testing.T) {
	final := `{"repository":{"pullRequests":{"pageInfo":{"hasNextPage":false,"endCursor":"Z"},
	  "nodes":[{"number":301,"title":"recent","mergedAt":"2025-01-01T00:00:00Z",
	    "updatedAt":"2025-01-02T00:00:00Z","author":{"login":"a"},
	    "files":{"pageInfo":{"hasNextPage":false},"nodes":[{"path":"go.mod"}]},
	    "reviews":{"pageInfo":{"hasNextPage":false},"nodes":[]}}]}},
	  "rateLimit":{"cost":1,"remaining":4000,"resetAt":"2026-01-01T01:00:00Z"}}`
	_, dir := runWalk(t, Options{PageSize: 3}, fixture(t, "page3.json"), reply{data: final})

	st := decodeState(t, dir)
	if st.PagesFetched != 2 || st.Counted != 1 {
		t.Errorf("pages_fetched=%d counted=%d, want 2/1", st.PagesFetched, st.Counted)
	}
	if got := deref(st.StopReason); got != "no more pages (end of merged PR history)" {
		t.Errorf("stop_reason = %q", got)
	}
}

func TestWalkStopsAtMaxPages(t *testing.T) {
	_, dir := runWalk(t, Options{MaxPages: 1}, fixture(t, "page1.json"))

	st := decodeState(t, dir)
	if st.PagesFetched != 2 {
		t.Errorf("pages_fetched = %d, want 2 (the aborted page counts)", st.PagesFetched)
	}
	want := "safety max-pages=1 reached"
	if got := deref(st.StopReason); got != want {
		t.Errorf("stop_reason = %q, want %q", got, want)
	}
	if len(st.Errors) != 1 || st.Errors[0] != want {
		t.Errorf("errors = %v, want [%q]", st.Errors, want)
	}
}

func TestWalkRetriesThenSucceeds(t *testing.T) {
	s, dir := runWalk(t, Options{Cap: 1},
		reply{err: errors.New("boom")}, reply{err: errors.New("boom")},
		fixture(t, "page1.json"))

	wantLines(t, readLines(t, dir), []string{line101})
	if len(s.naps) != 2 || s.naps[0] != 3*time.Second || s.naps[1] != 6*time.Second {
		t.Errorf("retry backoff = %v, want [3s 6s]", s.naps)
	}
	if len(decodeState(t, dir).Errors) != 0 {
		t.Errorf("a recovered page recorded an error")
	}
}

func TestWalkGivesUpAfterRetries(t *testing.T) {
	replies := make([]reply, fetchAttempts)
	for i := range replies {
		replies[i] = reply{err: fmt.Errorf("attempt %d", i)}
	}
	s, dir := runWalk(t, Options{}, replies...)

	if got := readLines(t, dir); len(got) != 0 {
		t.Errorf("wrote %v with no successful page", got)
	}
	st := decodeState(t, dir)
	if got := deref(st.StopReason); got != "repeated fetch errors" {
		t.Errorf("stop_reason = %q", got)
	}
	if len(st.Errors) != 1 || !strings.HasPrefix(st.Errors[0], "page 1: giving up after retries:") {
		t.Errorf("errors = %v", st.Errors)
	}
	if len(s.naps) != fetchAttempts-1 {
		t.Errorf("slept %d times, want %d (no nap after the last attempt)",
			len(s.naps), fetchAttempts-1)
	}
}

func TestWalkSleepsOffALowRateLimit(t *testing.T) {
	page := `{"repository":{"pullRequests":{"pageInfo":{"hasNextPage":false,"endCursor":"X"},
	  "nodes":[]}},
	  "rateLimit":{"cost":1,"remaining":5,"resetAt":"2026-01-01T00:30:00Z"}}`
	s, _ := runWalk(t, Options{}, reply{data: page})

	if len(s.naps) != 1 || s.naps[0] != 30*time.Minute+10*time.Second {
		t.Errorf("naps = %v, want [30m10s]", s.naps)
	}
}

func TestWalkRejectsMalformedRepo(t *testing.T) {
	s := &stub{t: t}
	f := newTestFetcher(Options{OutDir: t.TempDir(), Repo: "rook", Cap: 1, PageSize: 1, MaxPages: 1}, s)
	if err := f.Run(context.Background()); err == nil {
		t.Fatal("Run accepted a repo without an owner")
	}
}

// A GraphQL null repository — the answer to a misspelled or inaccessible name,
// carrying no `errors` array — must not be certified as an empty history.
func TestWalkRejectsNullRepository(t *testing.T) {
	rate := `"rateLimit":{"cost":1,"remaining":4000,"resetAt":"2026-01-01T01:00:00Z"}`
	for _, tc := range []struct{ name, page string }{
		{"null repository", `{"repository":null,` + rate + `}`},
		{"absent repository", `{` + rate + `}`},
		{"null pullRequests", `{"repository":{"pullRequests":null},` + rate + `}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, dir, err := runWalkErr(t, Options{}, reply{data: tc.page})
			if err == nil {
				t.Fatal("walk certified a null repository as an empty history")
			}
			if !strings.Contains(err.Error(), "rook/rook") {
				t.Errorf("error = %v, want it to name the repo", err)
			}
			wantNoStateFile(t, dir)
		})
	}
}

// hasNextPage with no endCursor would re-issue the cursorless query, spin until
// the max-pages net and still write a state file a consumer would read.
func TestWalkRejectsMissingEndCursor(t *testing.T) {
	for _, pageInfo := range []string{
		`{"hasNextPage":true}`,
		`{"hasNextPage":true,"endCursor":null}`,
		`{"hasNextPage":true,"endCursor":""}`,
	} {
		t.Run(pageInfo, func(t *testing.T) {
			page := `{"repository":{"pullRequests":{"pageInfo":` + pageInfo + `,
			  "nodes":[{"number":11,"title":"p","mergedAt":"2025-01-01T00:00:00Z",
			    "updatedAt":"2025-01-02T00:00:00Z","author":{"login":"a"},
			    "files":{"pageInfo":{"hasNextPage":false},"nodes":[]},
			    "reviews":{"pageInfo":{"hasNextPage":false},"nodes":[]}}]}},
			  "rateLimit":{"cost":1,"remaining":4000,"resetAt":"2026-01-01T01:00:00Z"}}`
			s, dir, err := runWalkErr(t, Options{}, reply{data: page})
			if err == nil {
				t.Fatal("walk accepted hasNextPage with no endCursor")
			}
			if !strings.Contains(err.Error(), "endCursor") {
				t.Errorf("error = %v, want it to name the missing cursor", err)
			}
			if len(s.queries) != 1 {
				t.Errorf("issued %d queries, want 1 (no cursorless re-walk)", len(s.queries))
			}
			wantNoStateFile(t, dir)
		})
	}
}

func TestWindowCutoff(t *testing.T) {
	for _, tc := range []struct {
		months float64
		want   string
	}{
		{24, "2024-01-01T00:00:00+00:00"},
		{0.5, "2025-12-17T00:00:00+00:00"},
		{3600, "1725-12-21T00:00:00+00:00"},
		{4000, "1692-08-19T00:00:00+00:00"},
		{24276, "0002-10-18T00:00:00+00:00"},
		{-24000, "4026-03-17T00:00:00+00:00"},
	} {
		got, err := windowCutoff(fixedNow, tc.months)
		if err != nil {
			t.Errorf("windowCutoff(%v): %v", tc.months, err)
			continue
		}
		if isoformat(got) != tc.want {
			t.Errorf("windowCutoff(%v) = %s, want %s", tc.months, isoformat(got), tc.want)
		}
	}
	// Each of these overflowed Python's datetime; none may resolve to a date.
	for _, months := range []float64{66000, -100000, 1e9, math.NaN(), math.Inf(1)} {
		if got, err := windowCutoff(fixedNow, months); err == nil {
			t.Errorf("windowCutoff(%v) = %s, want an error", months, isoformat(got))
		}
	}
}

// The overflow symptom end to end: scaled into a nanosecond Duration, a
// 4000-month window wrapped to a cutoff in 2277 that dropped every PR read.
func TestWalkLargeMonthsKeepsCutoffInThePast(t *testing.T) {
	page := `{"repository":{"pullRequests":{"pageInfo":{"hasNextPage":false,"endCursor":"X"},
	  "nodes":[{"number":1,"title":"ancient","mergedAt":"1900-01-01T00:00:00Z",
	    "updatedAt":"1900-01-02T00:00:00Z","author":{"login":"a"},
	    "files":{"pageInfo":{"hasNextPage":false},"nodes":[{"path":"go.mod"}]},
	    "reviews":{"pageInfo":{"hasNextPage":false},"nodes":[]}}]}},
	  "rateLimit":{"cost":1,"remaining":4000,"resetAt":"2026-01-01T01:00:00Z"}}`
	_, dir := runWalk(t, Options{Months: 4000}, reply{data: page})

	st := decodeState(t, dir)
	if st.Cutoff != "1692-08-19T00:00:00+00:00" {
		t.Errorf("cutoff = %q, want the Python window 1692-08-19", st.Cutoff)
	}
	if st.Counted != 1 {
		t.Errorf("counted = %d, want 1 (a 1900 PR is inside a 4000-month window)", st.Counted)
	}
}

func TestWalkRejectsUnrepresentableWindow(t *testing.T) {
	s, dir, err := runWalkErr(t, Options{Months: 1e9})
	if err == nil {
		t.Fatal("walk accepted a window no calendar can express")
	}
	if len(s.queries) != 0 {
		t.Errorf("issued %d queries before rejecting the window", len(s.queries))
	}
	if _, err := os.Stat(filepath.Join(dir, PRsFile)); !os.IsNotExist(err) {
		t.Errorf("%s was created: %v", PRsFile, err)
	}
	wantNoStateFile(t, dir)
}

func TestDeepFetchPatchesRecordsAndMovesFlags(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, PRsFile), line101+"\n"+line102+"\n")
	writeFile(t, filepath.Join(dir, StateFile), `{
  "repo": "rook/rook",
  "pages_fetched": 1,
  "counted": 2,
  "seen_total": 2,
  "oldest_mergedat": "2025-05-01T09:00:00+00:00",
  "cutoff": "2024-01-01T00:00:00+00:00",
  "cap": 100,
  "stop_reason": "no more pages (end of merged PR history)",
  "errors": [],
  "truncations": [
    {"number": 102, "kind": "files", "mergedAt": "2025-05-01T09:00:00Z"},
    {"number": 102, "kind": "reviews", "mergedAt": "2025-05-01T09:00:00Z"},
    {"number": 999, "kind": "files", "mergedAt": "2023-01-01T00:00:00Z"}
  ]
}`)

	s := &stub{t: t, replies: []reply{
		fixture(t, "deep_files_page1.json"),
		fixture(t, "deep_files_page2.json"),
		fixture(t, "deep_reviews.json"),
	}}
	f := newTestFetcher(Options{OutDir: dir, Repo: "rook/rook", DeepFetchOnly: true}, s)
	if err := f.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	wantPatched := `{"number":102,"title":"object: sweeping refactor",` +
		`"mergedAt":"2025-05-01T09:00:00Z","updatedAt":"2025-05-02T09:00:00Z",` +
		`"author":null,"files":{"pageInfo":{"hasNextPage":false},"nodes":[` +
		`{"path":"pkg/operator/ceph/object/rgw.go"},{"path":"pkg/operator/ceph/object/zone.go"},` +
		`{"path":"pkg/operator/ceph/object/realm.go"}]},` +
		`"reviews":{"pageInfo":{"hasNextPage":false},"nodes":[` +
		`{"author":{"login":"bob"},"state":"APPROVED"},` +
		`{"author":{"login":"dave"},"state":"CHANGES_REQUESTED"}]}}`
	wantLines(t, readLines(t, dir), []string{line101, wantPatched})

	want := `{
  "repo": "rook/rook",
  "pages_fetched": 1,
  "counted": 2,
  "seen_total": 2,
  "oldest_mergedat": "2025-05-01T09:00:00+00:00",
  "cutoff": "2024-01-01T00:00:00+00:00",
  "cap": 100,
  "stop_reason": "no more pages (end of merged PR history)",
  "errors": [],
  "truncations": [
    {
      "number": 999,
      "kind": "files",
      "mergedAt": "2023-01-01T00:00:00Z"
    }
  ],
  "deep_fetched": [
    {
      "number": 102,
      "kind": "files",
      "mergedAt": "2025-05-01T09:00:00Z"
    },
    {
      "number": 102,
      "kind": "reviews",
      "mergedAt": "2025-05-01T09:00:00Z"
    }
  ]
}`
	if got := readStateFile(t, dir); got != want {
		t.Errorf("state file:\n got %s\nwant %s", got, want)
	}

	if len(s.queries) != 3 {
		t.Fatalf("issued %d deep queries, want 3", len(s.queries))
	}
	if strings.Contains(s.queries[0], "after:") {
		t.Errorf("first files query carried a cursor: %s", s.queries[0])
	}
	if !strings.Contains(s.queries[1], `after: "DF1"`) {
		t.Errorf("second files query did not page: %s", s.queries[1])
	}
	for i, want := range []string{"files(first: 100", "files(first: 100", "reviews(first: 100"} {
		if !strings.Contains(s.queries[i], want) {
			t.Errorf("query %d missing %q:\n%s", i, want, s.queries[i])
		}
	}
	if !strings.Contains(s.queries[0], "pullRequest(number: 102)") {
		t.Errorf("query 0 did not target PR 102:\n%s", s.queries[0])
	}
}

// A failed deep fetch must leave the JSONL and the flags untouched rather than
// half-patch the run.
func TestDeepFetchLeavesOutputAloneOnError(t *testing.T) {
	dir := t.TempDir()
	jsonl := line101 + "\n" + line102 + "\n"
	writeFile(t, filepath.Join(dir, PRsFile), jsonl)
	state := `{"repo":"rook/rook","truncations":[{"number":102,"kind":"files","mergedAt":"x"}]}`
	writeFile(t, filepath.Join(dir, StateFile), state)

	s := &stub{t: t, replies: []reply{{err: errors.New("gh exploded")}}}
	f := newTestFetcher(Options{OutDir: dir, Repo: "rook/rook", DeepFetchOnly: true}, s)
	err := f.Run(context.Background())
	if err == nil {
		t.Fatal("DeepFetch swallowed a query failure")
	}
	if !strings.Contains(err.Error(), "deep-fetch PR 102 files") {
		t.Errorf("error = %v, want it to name the PR and field", err)
	}
	if got := readStateFile(t, dir); got != state {
		t.Errorf("state was rewritten after a failure:\n%s", got)
	}
	if got := strings.Join(readLines(t, dir), "\n") + "\n"; got != jsonl {
		t.Errorf("JSONL was rewritten after a failure:\n%s", got)
	}
}

func TestDeepFetchRejectsMissingEndCursor(t *testing.T) {
	dir := t.TempDir()
	jsonl := line101 + "\n" + line102 + "\n"
	writeFile(t, filepath.Join(dir, PRsFile), jsonl)
	state := `{"repo":"rook/rook","truncations":[{"number":102,"kind":"files","mergedAt":"x"}]}`
	writeFile(t, filepath.Join(dir, StateFile), state)

	page := `{"repository":{"pullRequest":{"files":{"pageInfo":{"hasNextPage":true},
	  "nodes":[{"path":"pkg/operator/ceph/object/rgw.go"}]}}}}`
	s := &stub{t: t, replies: []reply{{data: page}}}
	f := newTestFetcher(Options{OutDir: dir, Repo: "rook/rook", DeepFetchOnly: true}, s)
	err := f.Run(context.Background())
	if err == nil {
		t.Fatal("deep fetch accepted hasNextPage with no endCursor")
	}
	if !strings.Contains(err.Error(), "deep-fetch PR 102 files") ||
		!strings.Contains(err.Error(), "endCursor") {
		t.Errorf("error = %v, want it to name the PR, the field and the missing cursor", err)
	}
	if len(s.queries) != 1 {
		t.Errorf("issued %d deep queries, want 1 (no cursorless re-fetch)", len(s.queries))
	}
	if got := readStateFile(t, dir); got != state {
		t.Errorf("state was rewritten after a failure:\n%s", got)
	}
	if got := strings.Join(readLines(t, dir), "\n") + "\n"; got != jsonl {
		t.Errorf("JSONL was rewritten after a failure:\n%s", got)
	}
}

func TestWalkThenDeepFetch(t *testing.T) {
	dir := t.TempDir()
	s, _ := runWalk(t, Options{OutDir: dir, DeepFetch: true},
		fixture(t, "page1.json"), fixture(t, "page3.json"),
		fixture(t, "deep_files_page1.json"), fixture(t, "deep_files_page2.json"),
		fixture(t, "deep_reviews.json"))

	if len(s.queries) != 5 {
		t.Fatalf("issued %d queries, want 2 walk + 3 deep", len(s.queries))
	}
	st := decodeState(t, dir)
	if len(st.Truncations) != 0 || len(st.DeepFetched) != 2 {
		t.Errorf("truncations=%v deep_fetched=%v", st.Truncations, st.DeepFetched)
	}
}

func TestIsoformat(t *testing.T) {
	tests := []struct {
		in   time.Time
		want string
	}{
		{time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC), "2024-06-01T09:00:00+00:00"},
		{time.Date(2024, 6, 1, 9, 0, 0, 123000000, time.UTC), "2024-06-01T09:00:00.123000+00:00"},
		{time.Date(2024, 6, 1, 9, 0, 0, 999999999, time.UTC), "2024-06-01T09:00:00.999999+00:00"},
		{time.Date(2024, 6, 1, 9, 0, 0, 999, time.UTC), "2024-06-01T09:00:00+00:00"},
	}
	for _, tc := range tests {
		if got := isoformat(tc.in); got != tc.want {
			t.Errorf("isoformat(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSplitRepo(t *testing.T) {
	owner, name, err := splitRepo("rook/rook")
	if err != nil || owner != "rook" || name != "rook" {
		t.Errorf("splitRepo() = %q, %q, %v", owner, name, err)
	}
	for _, bad := range []string{"rook", "", "/rook", "rook/"} {
		if _, _, err := splitRepo(bad); err == nil {
			t.Errorf("splitRepo(%q) accepted", bad)
		}
	}
}

func wantNoStateFile(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, StateFile)); !os.IsNotExist(err) {
		t.Errorf("%s exists after a refused walk (err=%v)", StateFile, err)
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

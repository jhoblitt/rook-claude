package checklist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/prdash"
	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/sweepprefetch"
)

const fixtureRepo = "rook/rook"

// item builds one assessable snapshot entry, so each case below varies exactly
// the field whose skip class it exercises.
func item(number int, mut ...func(*sweepprefetch.PRItem)) *sweepprefetch.PRItem {
	it := &sweepprefetch.PRItem{
		Number:    number,
		Title:     "ceph: fix the thing",
		State:     "OPEN",
		CreatedAt: "2026-08-01T00:00:00Z",
		UpdatedAt: "2026-08-02T00:00:00Z",
		Author:    "alice",
		Labels:    []string{},
	}
	for _, m := range mut {
		m(it)
	}
	return it
}

func draft(it *sweepprefetch.PRItem) { it.IsDraft = true }
func bot(it *sweepprefetch.PRItem)   { it.Author = "mergify[bot]" }
func wipTitle(it *sweepprefetch.PRItem) {
	it.Title = "WIP: ceph: fix the thing"
}
func hold(it *sweepprefetch.PRItem) { it.Labels = []string{"ceph", "do-not-merge"} }

// snapshotDir writes the sweep dir's snapshot.json through the shape type the
// prefetch pass writes it with, so a drift in that contract breaks these tests
// rather than hiding behind hand-written JSON.
func snapshotDir(t *testing.T, items ...*sweepprefetch.PRItem) string {
	t.Helper()
	return snapshotDirOfKind(t, "prs", items...)
}

func snapshotDirOfKind(t *testing.T, kind string, items ...*sweepprefetch.PRItem) string {
	t.Helper()
	byNumber := map[string]*sweepprefetch.PRItem{}
	for _, it := range items {
		byNumber[strconv.Itoa(it.Number)] = it
	}
	data, err := json.Marshal(map[string]any{
		"fetched_at": "2026-08-11T00:00:00.000000+00:00",
		"repo":       fixtureRepo,
		"kind":       kind,
		"items":      byNumber,
	})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "snapshot.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeLedger(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "sweep.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// stubFetch stands in for the gh call, so no test reaches the network. Bodies
// not named fall back to a conforming one: a case says only what it varies.
type stubFetch struct {
	mu       sync.Mutex
	fallback string
	bodies   map[int]string
	errs     map[int]error
	seen     []int
	repos    []string
}

func (s *stubFetch) Body(_ context.Context, repo string, number int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen = append(s.seen, number)
	s.repos = append(s.repos, repo)
	if err, ok := s.errs[number]; ok {
		return "", err
	}
	if body, ok := s.bodies[number]; ok {
		return body, nil
	}
	return s.fallback, nil
}

func (s *stubFetch) audited() []int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := slices.Clone(s.seen)
	slices.Sort(out)
	return out
}

func conforming(t *testing.T) *stubFetch {
	t.Helper()
	return &stubFetch{fallback: block(t)}
}

// sweep runs the pass over dir with the fixture template, and returns what it
// wrote alongside the result and everything it printed.
func sweep(t *testing.T, dir string, fetch *stubFetch, mut ...func(*SweepOptions)) (SweepResult, map[int]Verdict, string, string) {
	t.Helper()
	var errs bytes.Buffer
	opts := SweepOptions{
		SweepDir: dir,
		Template: mustTemplate(t),
		Fetch:    fetch,
		Errs:     &errs,
	}
	for _, m := range mut {
		m(&opts)
	}
	res, err := Sweep(context.Background(), opts)
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	return res, rows(t, dir), readFile(t, filepath.Join(dir, "checklist.jsonl")), errs.String()
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// rows parses checklist.jsonl the way a consumer does: keyed by number, in no
// particular order.
func rows(t *testing.T, dir string) map[int]Verdict {
	t.Helper()
	out := map[int]Verdict{}
	for _, line := range strings.Split(strings.TrimSpace(readFile(t, filepath.Join(dir, "checklist.jsonl"))), "\n") {
		if line == "" {
			continue
		}
		var row SweepRow
		dec := json.NewDecoder(strings.NewReader(line))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&row); err != nil {
			t.Fatalf("row %q: %v", line, err)
		}
		if _, dup := out[row.Number]; dup {
			t.Errorf("%d has two rows; a consumer keying by number cannot tell which is the verdict", row.Number)
		}
		out[row.Number] = row.Verdict
	}
	return out
}

// skipRows reads skips.json through the type the dashboard reads it with, so a
// drift in that contract breaks here rather than in a rendered report. The
// decoder rejects unknown fields: a row carrying anything prdash does not read
// is a field the dashboard drops silently.
func skipRows(t *testing.T, dir string) []prdash.Skip {
	t.Helper()
	var got []prdash.Skip
	dec := json.NewDecoder(strings.NewReader(readFile(t, filepath.Join(dir, SkipsFile))))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("%s: %v", SkipsFile, err)
	}
	return got
}

func assertAudited(t *testing.T, fetch *stubFetch, want ...int) {
	t.Helper()
	if got := fetch.audited(); !slices.Equal(got, want) {
		t.Errorf("audited %v, want %v", got, want)
	}
}

// Each skip class keeps its own PR out for its own reason, and the survivor is
// the only one that costs a fetch.
func TestSweepSkipsEachClassForItsOwnReason(t *testing.T) {
	dir := snapshotDir(t,
		item(101),
		item(102, draft),
		item(103, hold),
		item(104, wipTitle),
		item(105, bot),
	)
	fetch := conforming(t)
	res, got, _, _ := sweep(t, dir, fetch)

	assertAudited(t, fetch, 101)
	if want := (map[int]Verdict{101: VerdictConforming}); !mapsEqual(got, want) {
		t.Errorf("rows = %v, want %v", got, want)
	}
	for class, want := range map[string]int{
		SkipDraft: 1, SkipDoNotMerge: 1, SkipWIP: 1, SkipBot: 1,
	} {
		if res.Skipped[class] != want {
			t.Errorf("skipped[%s] = %d, want %d (whole result: %+v)", class, res.Skipped[class], want, res)
		}
	}
	if len(fetch.repos) > 0 && fetch.repos[0] != fixtureRepo {
		t.Errorf("fetched against repo %q, want the snapshot's %q", fetch.repos[0], fixtureRepo)
	}
}

// --include-drafts widens the pass to drafts and to nothing else: do-not-merge
// is a human hold triage never overrides, and a bot's PR is still a bot's.
func TestSweepIncludeDraftsKeepsTheOtherHolds(t *testing.T) {
	dir := snapshotDir(t,
		item(101),
		item(102, draft),
		item(103, hold),
		item(104, wipTitle),
		item(105, bot),
	)
	fetch := conforming(t)
	res, got, _, _ := sweep(t, dir, fetch, func(o *SweepOptions) { o.IncludeDrafts = true })

	assertAudited(t, fetch, 101, 102)
	if len(got) != 2 {
		t.Errorf("rows = %v, want one each for 101 and 102", got)
	}
	if res.Skipped[SkipDraft] != 0 {
		t.Errorf("skipped[draft] = %d on an include-drafts run", res.Skipped[SkipDraft])
	}
}

// A carried card reuses the assessment a prior run stored, so re-auditing it
// writes a row nothing reads.
func TestSweepSkipsCarriedItems(t *testing.T) {
	dir := snapshotDir(t, item(101), item(102))
	writeLedger(t, dir, `{"items":{"101":"carried","102":"fresh"}}`)
	fetch := conforming(t)
	res, got, _, _ := sweep(t, dir, fetch)

	assertAudited(t, fetch, 102)
	if _, ok := got[101]; ok {
		t.Errorf("rows = %v, want no row for the carried 101", got)
	}
	if res.Skipped[SkipCarried] != 1 {
		t.Errorf("skipped[carried] = %d, want 1", res.Skipped[SkipCarried])
	}
}

// The pass holds the only per-PR record of what it left out, so every skipped
// PR costs a row the dashboard can show. Carried is not one of them: that card
// was assessed by a prior run and reaches the report as an assessment.
func TestSweepWritesASkipRowForEveryClassButCarried(t *testing.T) {
	dir := snapshotDir(t,
		item(105, bot),
		item(104, wipTitle),
		item(103, hold),
		item(102, draft),
		item(101),
		item(100),
	)
	writeLedger(t, dir, `{"items":{"100":"carried"}}`)
	sweep(t, dir, conforming(t))

	// Ascending by number, as the assessed table orders its own rows
	// (reporting.md, "Ordering"); prdash renders skips in file order.
	want := []prdash.Skip{
		{Number: 102, Class: prdash.Text(SkipDraft), Author: "alice", Title: "ceph: fix the thing"},
		{Number: 103, Class: prdash.Text(SkipDoNotMerge), Author: "alice", Title: "ceph: fix the thing"},
		{Number: 104, Class: prdash.Text(SkipWIP), Author: "alice", Title: "WIP: ceph: fix the thing"},
		{Number: 105, Class: prdash.Text(SkipBot), Author: "mergify[bot]", Title: "ceph: fix the thing"},
	}
	if got := skipRows(t, dir); !slices.Equal(got, want) {
		t.Errorf("%s = %+v, want %+v", SkipsFile, got, want)
	}
}

// prdash defaults a missing skips.json to empty, so a pass that wrote the file
// only when it had rows would let a lost run read as a pool that skipped
// nothing.
func TestSweepWritesSkipsWithNothingToSkip(t *testing.T) {
	dir := snapshotDir(t, item(101))
	sweep(t, dir, conforming(t))

	if got := skipRows(t, dir); len(got) != 0 {
		t.Errorf("%s = %+v, want no rows", SkipsFile, got)
	}
}

// A title is contributor-authored text, and skips.json is the one file of this
// pass that carries it: the dashboard's skip rows show the title, escaped. The
// verdict file must not, whatever class the PR is in.
func TestSweepKeepsTitlesOutOfTheVerdictFile(t *testing.T) {
	const sentinel = "ZzUnfencedContributorProseZz"
	titled := func(it *sweepprefetch.PRItem) { it.Title = sentinel }
	dir := snapshotDir(t, item(101, titled), item(102, draft, titled))
	res, _, file, errOut := sweep(t, dir, conforming(t))

	for name, text := range map[string]string{
		"checklist.jsonl": file,
		"stderr":          errOut,
		"the summary":     res.String(),
	} {
		if strings.Contains(text, sentinel) {
			t.Errorf("%s carries a PR title:\n%s", name, text)
		}
	}
	got := skipRows(t, dir)
	if len(got) != 1 || string(got[0].Title) != sentinel {
		t.Errorf("%s = %+v, want the skipped PR's title verbatim for the dashboard", SkipsFile, got)
	}
}

// An unwritable skips.json is a failed run, not a run that skipped nothing —
// and it fails before the pass spends a single fetch on a report it cannot
// publish.
func TestSweepRefusesWhenItCannotWriteSkips(t *testing.T) {
	dir := snapshotDir(t, item(101), item(102, draft))
	// A symlink to the sweep dir fails the write as EISDIR for any user, which
	// a mode bit would not do for root.
	if err := os.Symlink(dir, filepath.Join(dir, SkipsFile)); err != nil {
		t.Fatal(err)
	}
	fetch := conforming(t)

	_, err := Sweep(context.Background(), SweepOptions{
		SweepDir: dir, Template: mustTemplate(t), Fetch: fetch, Errs: &bytes.Buffer{},
	})
	if err == nil {
		t.Fatal("Sweep = nil error")
	}
	assertAudited(t, fetch)
	if _, statErr := os.Stat(filepath.Join(dir, SweepFile)); statErr == nil {
		t.Error("a sweep that could not write its skips still wrote checklist.jsonl")
	}
}

// A run that has assessed nothing has no ledger to consult, and must audit the
// pool rather than read the absence as "everything is carried".
func TestSweepWithoutALedgerAuditsEverything(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ledger string
	}{
		{"no sweep.json at all", ""},
		{"a sweep.json with no items map", `{"scope":{"kind":"prs"},"log":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := snapshotDir(t, item(101), item(102))
			if tc.ledger != "" {
				writeLedger(t, dir, tc.ledger)
			}
			fetch := conforming(t)
			_, got, _, _ := sweep(t, dir, fetch)

			assertAudited(t, fetch, 101, 102)
			if len(got) != 2 {
				t.Errorf("rows = %v, want one each for 101 and 102", got)
			}
		})
	}
}

// The two facts the shell pass could not tell apart: a PR whose body never
// arrived is unaudited, and a PR whose body arrived empty has no checklist.
func TestSweepFetchFailureIsUnauditedAndAnEmptyBodyIsNoChecklist(t *testing.T) {
	dir := snapshotDir(t, item(101), item(102))
	fetch := conforming(t)
	fetch.errs = map[int]error{101: errors.New("gh pr view: HTTP 502")}
	fetch.bodies = map[int]string{102: ""}
	_, got, _, errOut := sweep(t, dir, fetch)

	want := map[int]Verdict{101: VerdictUnaudited, 102: VerdictNoChecklist}
	if !mapsEqual(got, want) {
		t.Errorf("rows = %v, want %v", got, want)
	}
	if !strings.Contains(errOut, "101") {
		t.Errorf("nothing on stderr named the PR whose fetch failed: %q", errOut)
	}
}

// A body that fails the audit is a finding, not a failure: it gets its real
// verdict, and unaudited appears nowhere.
func TestSweepNonConformingWritesOneRowWithItsVerdict(t *testing.T) {
	dir := snapshotDir(t, item(101))
	fetch := conforming(t)
	fetch.bodies = map[int]string{101: dropLine(block(t), docsLine)}
	res, got, file, _ := sweep(t, dir, fetch)

	want := map[int]Verdict{101: VerdictNonConforming}
	if !mapsEqual(got, want) {
		t.Errorf("rows = %v, want %v", got, want)
	}
	if n := strings.Count(strings.TrimSpace(file), "\n") + 1; n != 1 {
		t.Errorf("checklist.jsonl has %d rows, want exactly 1:\n%s", n, file)
	}
	if strings.Contains(file, string(VerdictUnaudited)) {
		t.Errorf("a non-conforming audit was written as unaudited:\n%s", file)
	}
	if res.Verdicts[VerdictNonConforming] != 1 {
		t.Errorf("verdicts = %v, want one non-conforming", res.Verdicts)
	}
}

// The body is attacker-authored and the audit report reproduces it verbatim,
// so nothing but the number and the verdict may leave this pass.
func TestSweepWritesNoBodyText(t *testing.T) {
	const sentinel = "ZzUnfencedContributorProseZz"
	dir := snapshotDir(t, item(101), item(102))
	fetch := conforming(t)
	fetch.bodies = map[int]string{
		101: sentinel + "\n\n" + block(t) + "\n- [x] " + sentinel + "\n",
		102: sentinel,
	}
	// The premise: the full report of that body does quote the sentinel, so
	// what follows is a real test of what the pass withholds.
	if lines := fmt.Sprint(Audit(mustTemplate(t), fetch.bodies[101]).Lines); !strings.Contains(lines, sentinel) {
		t.Fatalf("the fixture body never reaches the report, so this proves nothing: %s", lines)
	}

	res, _, file, errOut := sweep(t, dir, fetch)

	for name, text := range map[string]string{
		"checklist.jsonl": file,
		"stderr":          errOut,
		"the summary":     res.String(),
	} {
		if strings.Contains(text, sentinel) {
			t.Errorf("%s carries body text:\n%s", name, text)
		}
	}
}

// gateFetch holds every call until the test releases it, and records how many
// were ever in flight at once.
type gateFetch struct {
	mu       sync.Mutex
	inflight int
	peak     int
	body     string
	arrived  chan int
	release  chan struct{}
}

func (g *gateFetch) Body(_ context.Context, _ string, number int) (string, error) {
	g.mu.Lock()
	g.inflight++
	g.peak = max(g.peak, g.inflight)
	g.mu.Unlock()

	g.arrived <- number
	<-g.release

	g.mu.Lock()
	g.inflight--
	g.mu.Unlock()
	return g.body, nil
}

// --jobs is a real bound on the fetches in flight, in both directions: a pass
// that fetched one at a time would never reach the third arrival below, and one
// that spawned a fetch per PR would overshoot the bound.
func TestSweepFetchesUpToJobsAtOnce(t *testing.T) {
	const jobs = 3
	gate := &gateFetch{body: block(t), arrived: make(chan int, 8), release: make(chan struct{})}
	dir := snapshotDir(t, item(101), item(102), item(103), item(104))

	type outcome struct {
		res SweepResult
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		res, err := Sweep(context.Background(), SweepOptions{
			SweepDir: dir, Template: mustTemplate(t), Jobs: jobs, Fetch: gate,
		})
		done <- outcome{res, err}
	}()

	for i := range jobs {
		select {
		case <-gate.arrived:
		case <-time.After(10 * time.Second):
			t.Fatalf("only %d fetches ever started with --jobs %d; the pass is serial", i, jobs)
		}
	}
	close(gate.release)

	got := <-done
	if got.err != nil {
		t.Fatalf("Sweep: %v", got.err)
	}
	gate.mu.Lock()
	defer gate.mu.Unlock()
	if gate.peak > jobs {
		t.Errorf("%d fetches were in flight at once, over the --jobs %d bound", gate.peak, jobs)
	}
	if got.res.Audited != 4 {
		t.Errorf("audited = %d, want 4", got.res.Audited)
	}
}

// Every selected PR gets a row whatever the worker count; consumers key by
// number, so the order they land in is not a contract.
func TestSweepAuditsEveryItemUnderConcurrency(t *testing.T) {
	var items []*sweepprefetch.PRItem
	for n := 101; n < 113; n++ {
		items = append(items, item(n))
	}
	dir := snapshotDir(t, items...)
	fetch := conforming(t)
	res, got, _, _ := sweep(t, dir, fetch, func(o *SweepOptions) { o.Jobs = 4 })

	if len(got) != len(items) {
		t.Errorf("rows = %d, want %d: %v", len(got), len(items), got)
	}
	if res.Audited != len(items) {
		t.Errorf("audited = %d, want %d", res.Audited, len(items))
	}
}

// failCloser takes every row and fails only at Close, which is where a network
// filesystem reports the ENOSPC or EDQUOT of a write it deferred.
type failCloser struct {
	io.Writer
	err error
}

func (f failCloser) Close() error { return f.err }

// deadWriter takes nothing, like a file on a filesystem that has just filled
// up.
type deadWriter struct{ err error }

func (d deadWriter) Write([]byte) (int, error) { return 0, d.err }
func (d deadWriter) Close() error              { return nil }

func auditOptions(t *testing.T, fetch BodyFetcher, jobs int) SweepOptions {
	t.Helper()
	return SweepOptions{Template: mustTemplate(t), Fetch: fetch, Errs: io.Discard, Jobs: jobs}
}

// A write that fails only at close leaves a truncated checklist.jsonl, and a
// pass that discarded that error would publish the truncation as the pool's
// verdicts.
func TestSweepReportsAWriteFailureThatSurfacesAtClose(t *testing.T) {
	full := errors.New("no space left on device")
	res := SweepResult{Verdicts: map[Verdict]int{}, Skipped: map[string]int{}}

	err := auditPool(context.Background(), auditOptions(t, conforming(t), 1), fixtureRepo,
		[]int{101}, failCloser{Writer: io.Discard, err: full}, &res)

	if !errors.Is(err, full) {
		t.Fatalf("auditPool = %v, want the close failure %v", err, full)
	}
}

// Once the rows cannot be written, every further body costs a gh call for a
// verdict that provably cannot be kept.
func TestSweepStopsFetchingOnceItCannotWrite(t *testing.T) {
	const jobs = 2
	var numbers []int
	for n := 101; n < 121; n++ {
		numbers = append(numbers, n)
	}
	full := errors.New("no space left on device")
	fetch := conforming(t)
	res := SweepResult{Verdicts: map[Verdict]int{}, Skipped: map[string]int{}}

	err := auditPool(context.Background(), auditOptions(t, fetch, jobs), fixtureRepo,
		numbers, deadWriter{err: full}, &res)

	if !errors.Is(err, full) {
		t.Fatalf("auditPool = %v, want the write failure %v", err, full)
	}
	// One fetch per worker can already be in flight when the first write
	// fails, and one more can be handed out as the feed stops.
	if got := len(fetch.audited()); got > jobs+1 {
		t.Errorf("fetched %d of %d bodies with the file unwritable; the pass kept spending network on rows it cannot keep",
			got, len(numbers))
	}
}

// A pass that cannot read its inputs must abort rather than write a file whose
// emptiness reads as a pool with nothing to find. An issues snapshot is the
// sharpest of these: item numbers are unique across a repo's issues and pull
// requests, so auditing one would fetch whichever PRs share its numbers.
func TestSweepRefusesWhatItCannotAudit(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setUp func(t *testing.T) string
	}{
		{"an issues snapshot", func(t *testing.T) string {
			return snapshotDirOfKind(t, "issues", item(101))
		}},
		{"no snapshot at all", func(t *testing.T) string { return t.TempDir() }},
		{"a snapshot with no items", func(t *testing.T) string { return snapshotDir(t) }},
		{"a snapshot naming no repo", func(t *testing.T) string {
			dir := snapshotDir(t, item(101))
			path := filepath.Join(dir, "snapshot.json")
			if err := os.WriteFile(path, []byte(strings.Replace(readFile(t, path), `"repo":"rook/rook"`, `"repo":""`, 1)), 0o644); err != nil {
				t.Fatal(err)
			}
			return dir
		}},
		{"an unparseable ledger", func(t *testing.T) string {
			dir := snapshotDir(t, item(101))
			writeLedger(t, dir, "{not json")
			return dir
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := tc.setUp(t)
			_, err := Sweep(context.Background(), SweepOptions{
				SweepDir: dir, Template: mustTemplate(t), Fetch: conforming(t), Errs: &bytes.Buffer{},
			})
			if err == nil {
				t.Fatal("Sweep = nil error")
			}
			if _, statErr := os.Stat(filepath.Join(dir, SweepFile)); statErr == nil {
				t.Error("a refused sweep still wrote checklist.jsonl")
			}
		})
	}
}

func mapsEqual(got, want map[int]Verdict) bool {
	if len(got) != len(want) {
		return false
	}
	for k, v := range want {
		if got[k] != v {
			return false
		}
	}
	return true
}

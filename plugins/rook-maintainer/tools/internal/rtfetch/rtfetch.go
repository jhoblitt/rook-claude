// Package rtfetch mines merged pull requests, with their changed files and
// reviews, for the rook-triage KB refresh (skills/rook-triage/references/
// routing.md).
//
// One cursor walks repository.pullRequests (states: MERGED, UPDATED_AT DESC).
// That connection has no search-API 1000-result cap, so a single walk covers
// any window — which is why routing.md shards nothing and hands the whole
// window to one invocation. The walk stops on a full page whose PRs are all
// updatedAt-older than the cutoff: updatedAt >= mergedAt always holds and
// pages are updatedAt ordered, so no in-window PR can follow such a page. A
// mergedAt-based stop is NOT safe — a mass comment sweep over old PRs floats
// them back to the top.
//
// Per-PR truncation (files > 100, reviews > 30) is flagged rather than
// silently dropped; the miner turns the flags into `truncation` entries per the
// two-tier contract, and DeepFetch resolves them by paginating those PRs one at
// a time.
package rtfetch

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/ghx"
)

const (
	PRsFile   = "rt_prs.jsonl"
	StateFile = "rt_fetch_state.json"

	daysPerMonth    = 30.44
	maxWindowDays   = 1e9
	fetchAttempts   = 6
	rateLimitFloor  = 200
	maxRateLimitNap = time.Hour
	deepPageSize    = 100
)

// QueryFunc runs one GraphQL query and unmarshals its `data` object into out.
type QueryFunc func(ctx context.Context, query string, out any) error

type Options struct {
	OutDir        string
	Months        float64
	Cap           int
	Repo          string
	PageSize      int
	MaxPages      int
	DeepFetch     bool
	DeepFetchOnly bool
}

type Fetcher struct {
	Opts  Options
	Query QueryFunc
	Log   io.Writer
	Now   func() time.Time
	Sleep func(time.Duration)
}

func New(opts Options) *Fetcher {
	return &Fetcher{
		Opts:  opts,
		Query: ghx.GraphQL,
		Log:   os.Stderr,
		Now:   time.Now,
		Sleep: time.Sleep,
	}
}

type Actor struct {
	Login string `json:"login"`
}

// PageFlag carries hasNextPage alone: the per-PR connections in the walk query
// select no cursor, and the emitted JSONL shape must not grow one.
type PageFlag struct {
	HasNextPage bool `json:"hasNextPage"`
}

type FileNode struct {
	Path string `json:"path"`
}

type ReviewNode struct {
	Author *Actor `json:"author"`
	State  string `json:"state"`
}

type Files struct {
	PageInfo PageFlag   `json:"pageInfo"`
	Nodes    []FileNode `json:"nodes"`
}

type Reviews struct {
	PageInfo PageFlag     `json:"pageInfo"`
	Nodes    []ReviewNode `json:"nodes"`
}

// PR is one counted pull request, one JSONL line. Field order is the emitted
// key order and the analysis layer reads these names verbatim: do not reorder
// or rename without changing rt-analyze in the same commit.
type PR struct {
	Number    int     `json:"number"`
	Title     string  `json:"title"`
	MergedAt  string  `json:"mergedAt"`
	UpdatedAt string  `json:"updatedAt"`
	Author    *Actor  `json:"author"`
	Files     Files   `json:"files"`
	Reviews   Reviews `json:"reviews"`
}

type Truncation struct {
	Number   int    `json:"number"`
	Kind     string `json:"kind"`
	MergedAt string `json:"mergedAt"`
}

// State is the assembler's provenance input. deep_fetched is absent until a
// deep fetch runs and empty-but-present afterwards, hence omitzero rather than
// omitempty.
type State struct {
	Repo           string       `json:"repo"`
	PagesFetched   int          `json:"pages_fetched"`
	Counted        int          `json:"counted"`
	SeenTotal      int          `json:"seen_total"`
	OldestMergedAt *string      `json:"oldest_mergedat"`
	Cutoff         string       `json:"cutoff"`
	Cap            int          `json:"cap"`
	StopReason     *string      `json:"stop_reason"`
	Errors         []string     `json:"errors"`
	Truncations    []Truncation `json:"truncations"`
	DeepFetched    []Truncation `json:"deep_fetched,omitzero"`
}

type rateLimit struct {
	Cost      *int   `json:"cost"`
	Remaining *int   `json:"remaining"`
	ResetAt   string `json:"resetAt"`
}

type pageCursor struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// after returns the cursor to resume from. GitHub always sends an endCursor
// with a hasNextPage page; an empty one would re-issue the cursorless query and
// spin over page 1 until the max-pages net, emitting a state file that looks
// consumable, so a missing cursor is treated as a malformed response.
func (p pageCursor) after() (string, error) {
	if p.EndCursor == "" {
		return "", errors.New("hasNextPage with no endCursor")
	}
	return p.EndCursor, nil
}

type walkPage struct {
	PageInfo pageCursor `json:"pageInfo"`
	Nodes    []PR       `json:"nodes"`
}

// The pointers are load-bearing: GraphQL answers a misspelled or inaccessible
// repo with a null repository and no `errors` array, and a value struct would
// decode that into an empty page indistinguishable from an empty history.
type walkResponse struct {
	Repository *struct {
		PullRequests *walkPage `json:"pullRequests"`
	} `json:"repository"`
	RateLimit rateLimit `json:"rateLimit"`
}

type connection[T any] struct {
	PageInfo pageCursor `json:"pageInfo"`
	Nodes    []T        `json:"nodes"`
}

type deepResponse[T any] struct {
	Repository struct {
		PullRequest map[string]connection[T] `json:"pullRequest"`
	} `json:"repository"`
}

const walkQueryTemplate = `
query {
  repository(owner: "%s", name: "%s") {
    pullRequests(states: MERGED, orderBy: {field: UPDATED_AT, direction: DESC}, first: %d%s) {
      pageInfo { hasNextPage endCursor }
      nodes {
        number
        title
        mergedAt
        updatedAt
        author { login }
        files(first: 100) {
          pageInfo { hasNextPage }
          nodes { path }
        }
        reviews(first: 30) {
          pageInfo { hasNextPage }
          nodes { author { login } state }
        }
      }
    }
  }
  rateLimit { cost remaining resetAt }
}
`

func afterClause(cursor string) string {
	if cursor == "" {
		return ""
	}
	return fmt.Sprintf(", after: %q", cursor)
}

func walkQuery(owner, name string, pageSize int, cursor string) string {
	return fmt.Sprintf(walkQueryTemplate, owner, name, pageSize, afterClause(cursor))
}

func deepQuery(owner, name string, number int, field, inner, cursor string) string {
	return fmt.Sprintf(
		`query { repository(owner: %q, name: %q) { pullRequest(number: %d) { %s(first: %d%s) { `+
			`pageInfo { hasNextPage endCursor } nodes { %s } } } } }`,
		owner, name, number, field, deepPageSize, afterClause(cursor), inner)
}

func (f *Fetcher) Run(ctx context.Context) error {
	if f.Opts.DeepFetchOnly {
		return f.DeepFetch(ctx)
	}
	state, err := f.walk(ctx)
	if err != nil {
		return err
	}
	if f.Opts.DeepFetch && len(state.Truncations) > 0 {
		return f.DeepFetch(ctx)
	}
	return nil
}

// walker accumulates the cross-page bookkeeping the stop conditions depend on.
type walker struct {
	cap         int
	cutoff      time.Time
	enc         *json.Encoder
	seen        map[int]bool
	counted     int
	oldest      time.Time
	truncations []Truncation
}

// page consumes one page of nodes, reporting whether every node on it was
// updatedAt-older than the cutoff and how many it counted.
func (w *walker) page(nodes []PR) (bool, int, error) {
	allStale := true
	newCount := 0
	for _, n := range nodes {
		merged, err := parseTime(n.MergedAt)
		if err != nil {
			return false, newCount, fmt.Errorf("PR %d mergedAt: %w", n.Number, err)
		}
		updated, err := parseTime(n.UpdatedAt)
		if err != nil {
			return false, newCount, fmt.Errorf("PR %d updatedAt: %w", n.Number, err)
		}
		if !updated.Before(w.cutoff) {
			allStale = false
		}
		if w.seen[n.Number] {
			continue
		}
		w.seen[n.Number] = true

		if n.Files.PageInfo.HasNextPage {
			w.truncations = append(w.truncations,
				Truncation{Number: n.Number, Kind: "files", MergedAt: n.MergedAt})
		}
		if n.Reviews.PageInfo.HasNextPage {
			w.truncations = append(w.truncations,
				Truncation{Number: n.Number, Kind: "reviews", MergedAt: n.MergedAt})
		}
		if merged.Before(w.cutoff) {
			continue
		}

		w.counted++
		newCount++
		if w.oldest.IsZero() || merged.Before(w.oldest) {
			w.oldest = merged
		}
		if err := w.enc.Encode(n); err != nil {
			return false, newCount, err
		}
		if w.counted >= w.cap {
			break
		}
	}
	return allStale, newCount, nil
}

// WindowCutoff turns --months into the oldest timestamp a mine counts, and the
// window's length in whole days. It is the one window arithmetic of the kb
// refresh: the walk here and rtcommits' git-log mine both call it, so their
// provenance blocks describe the same window.
//
// RoundToEven, not Round: the window has to land on the same day the Python
// fetch layer picked, and round() in Python is half-to-even. AddDate, not a
// scaled time.Duration: a Duration is int64 nanoseconds and wraps past ~292
// years, which turned a large --months into a cutoff in the future that
// silently dropped all history. Windows outside Python's datetime range are
// rejected rather than resolved into a year the state file cannot express.
//
// A non-positive months is NOT rejected here — it yields a cutoff in the future
// and an empty window, matching the Python layer. A caller for which that is
// nonsense rejects it before calling; rtcommits.Mine does.
func WindowCutoff(now time.Time, months float64) (time.Time, int, error) {
	days := math.RoundToEven(months * daysPerMonth)
	if math.IsNaN(days) || math.Abs(days) > maxWindowDays {
		return time.Time{}, 0, fmt.Errorf("--months=%v is out of range (%v days)", months, days)
	}
	cutoff := now.UTC().AddDate(0, 0, -int(days))
	if y := cutoff.Year(); y < 1 || y > 9999 {
		return time.Time{}, 0, fmt.Errorf("--months=%v puts the cutoff in year %d, outside 1..9999",
			months, y)
	}
	return cutoff, int(days), nil
}

func (f *Fetcher) walk(ctx context.Context) (_ *State, err error) {
	owner, name, err := splitRepo(f.Opts.Repo)
	if err != nil {
		return nil, err
	}
	cutoff, _, err := WindowCutoff(f.Now(), f.Opts.Months)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(f.Opts.OutDir, 0o755); err != nil {
		return nil, err
	}
	out, err := os.Create(filepath.Join(f.Opts.OutDir, PRsFile))
	if err != nil {
		return nil, err
	}
	// The JSONL is the run's whole product; a Close that fails after the last
	// Flush (ENOSPC, a network filesystem reporting late) would otherwise
	// truncate it silently and rt-analyze would read a short file as complete.
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	buf := bufio.NewWriter(out)

	w := &walker{
		cap:         f.Opts.Cap,
		cutoff:      cutoff,
		enc:         newEncoder(buf),
		seen:        map[int]bool{},
		truncations: []Truncation{},
	}
	errs := []string{}
	stopReason := ""
	cursor := ""
	pageNum := 0

	for {
		pageNum++
		if pageNum > f.Opts.MaxPages {
			stopReason = fmt.Sprintf("safety max-pages=%d reached", f.Opts.MaxPages)
			errs = append(errs, stopReason)
			break
		}

		data, err := f.fetchPage(ctx, owner, name, cursor)
		if err != nil {
			errs = append(errs, fmt.Sprintf("page %d: giving up after retries: %v", pageNum, err))
			stopReason = "repeated fetch errors"
			break
		}
		if data.Repository == nil || data.Repository.PullRequests == nil {
			return nil, fmt.Errorf("page %d: GraphQL returned a null repository for %q "+
				"(misspelled name, or no access to it?)", pageNum, f.Opts.Repo)
		}
		block := data.Repository.PullRequests

		allStale, newCount, err := w.page(block.Nodes)
		if err != nil {
			return nil, err
		}
		if err := buf.Flush(); err != nil {
			return nil, err
		}
		f.logf("page %d: got=%d new_counted=%d total_counted=%d seen_total=%d "+
			"oldest_so_far=%s cost=%s remaining=%s page_all_stale=%v hasNextPage=%v",
			pageNum, len(block.Nodes), newCount, w.counted, len(w.seen),
			optTime(w.oldest), optInt(data.RateLimit.Cost), optInt(data.RateLimit.Remaining),
			allStale, block.PageInfo.HasNextPage)

		f.throttle(data.RateLimit)

		if w.counted >= f.Opts.Cap {
			stopReason = fmt.Sprintf("reached --cap=%d", f.Opts.Cap)
			break
		}
		if allStale && len(block.Nodes) == f.Opts.PageSize {
			stopReason = "full page entirely updatedAt-older than cutoff"
			break
		}
		if !block.PageInfo.HasNextPage {
			stopReason = "no more pages (end of merged PR history)"
			break
		}
		next, err := block.PageInfo.after()
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum, err)
		}
		cursor = next
	}

	state := &State{
		Repo:           f.Opts.Repo,
		PagesFetched:   pageNum,
		Counted:        w.counted,
		SeenTotal:      len(w.seen),
		OldestMergedAt: optIsoformat(w.oldest),
		Cutoff:         isoformat(cutoff),
		Cap:            f.Opts.Cap,
		StopReason:     nullable(stopReason),
		Errors:         errs,
		Truncations:    w.truncations,
	}
	if err := writeState(filepath.Join(f.Opts.OutDir, StateFile), state); err != nil {
		return nil, err
	}
	compact, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	if len(compact) > 2000 {
		compact = compact[:2000]
	}
	f.logf("DONE: %s", compact)
	return state, nil
}

func (f *Fetcher) fetchPage(ctx context.Context, owner, name, cursor string) (*walkResponse, error) {
	query := walkQuery(owner, name, f.Opts.PageSize, cursor)
	var last error
	for attempt := range fetchAttempts {
		var resp walkResponse
		err := f.Query(ctx, query, &resp)
		if err == nil {
			return &resp, nil
		}
		last = err
		if attempt < fetchAttempts-1 {
			f.Sleep(time.Duration(3*(attempt+1)) * time.Second)
		}
	}
	return nil, last
}

// throttle parks the walk when the GraphQL budget is nearly spent: the walk is
// long enough that hitting zero mid-run would lose the whole page sequence.
func (f *Fetcher) throttle(rl rateLimit) {
	if rl.Remaining == nil || *rl.Remaining >= rateLimitFloor {
		return
	}
	f.logf("rate limit low (%d), sleeping until %s", *rl.Remaining, rl.ResetAt)
	reset, err := parseTime(rl.ResetAt)
	if err != nil {
		return
	}
	nap := reset.Sub(f.Now().UTC())
	if nap < 0 {
		nap = 0
	}
	nap += 10 * time.Second
	if nap > maxRateLimitNap {
		nap = maxRateLimitNap
	}
	f.Sleep(nap)
}

// DeepFetch replaces each truncation-flagged connection with its full node set
// and moves the flag from `truncations` to `deep_fetched`. Flags naming a PR
// that is not in the JSONL (out of the counted window) stay as flags.
func (f *Fetcher) DeepFetch(ctx context.Context) error {
	owner, name, err := splitRepo(f.Opts.Repo)
	if err != nil {
		return err
	}
	jsonlPath := filepath.Join(f.Opts.OutDir, PRsFile)
	statePath := filepath.Join(f.Opts.OutDir, StateFile)

	state, err := readState(statePath)
	if err != nil {
		return err
	}
	prs, index, err := readPRs(jsonlPath)
	if err != nil {
		return err
	}

	remaining := []Truncation{}
	done := state.DeepFetched
	if done == nil {
		done = []Truncation{}
	}
	for _, t := range state.Truncations {
		i, ok := index[t.Number]
		if !ok {
			remaining = append(remaining, t)
			continue
		}
		total := 0
		if t.Kind == "files" {
			nodes, err := deepFetchField[FileNode](ctx, f, owner, name, t.Number, "files", "path")
			if err != nil {
				return err
			}
			prs[i].Files = Files{Nodes: nodes}
			total = len(nodes)
		} else {
			nodes, err := deepFetchField[ReviewNode](ctx, f, owner, name, t.Number,
				"reviews", "author { login } state")
			if err != nil {
				return err
			}
			prs[i].Reviews = Reviews{Nodes: nodes}
			total = len(nodes)
		}
		done = append(done, t)
		f.logf("deep-fetched PR #%d %s: %d total", t.Number, t.Kind, total)
	}

	if err := writePRs(jsonlPath, prs); err != nil {
		return err
	}
	state.Truncations = remaining
	state.DeepFetched = done
	if err := writeState(statePath, state); err != nil {
		return err
	}
	f.logf("deep-fetch: %d resolved, %d out-of-set left as flags", len(done), len(remaining))
	return nil
}

func deepFetchField[T any](ctx context.Context, f *Fetcher, owner, name string,
	number int, field, inner string) ([]T, error) {
	nodes := []T{}
	cursor := ""
	for {
		var resp deepResponse[T]
		if err := f.Query(ctx, deepQuery(owner, name, number, field, inner, cursor), &resp); err != nil {
			return nil, fmt.Errorf("deep-fetch PR %d %s: %w", number, field, err)
		}
		block, ok := resp.Repository.PullRequest[field]
		if !ok {
			return nil, fmt.Errorf("deep-fetch PR %d: response carried no %s connection", number, field)
		}
		nodes = append(nodes, block.Nodes...)
		if !block.PageInfo.HasNextPage {
			return nodes, nil
		}
		next, err := block.PageInfo.after()
		if err != nil {
			return nil, fmt.Errorf("deep-fetch PR %d %s: %w", number, field, err)
		}
		cursor = next
	}
}

// readPRs returns the records in file order plus an index by PR number, so a
// rewrite after deep fetching keeps the JSONL line order it was read in.
func readPRs(path string) ([]PR, map[int]int, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = fh.Close() }()

	var prs []PR
	index := map[int]int{}
	dec := json.NewDecoder(bufio.NewReader(fh))
	for {
		var pr PR
		if err := dec.Decode(&pr); err != nil {
			if errors.Is(err, io.EOF) {
				return prs, index, nil
			}
			return nil, nil, fmt.Errorf("reading %s: %w", path, err)
		}
		if i, ok := index[pr.Number]; ok {
			prs[i] = pr
			continue
		}
		index[pr.Number] = len(prs)
		prs = append(prs, pr)
	}
}

func writePRs(path string, prs []PR) (err error) {
	fh, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := fh.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	buf := bufio.NewWriter(fh)
	enc := newEncoder(buf)
	for _, pr := range prs {
		if err := enc.Encode(pr); err != nil {
			return err
		}
	}
	return buf.Flush()
}

func readState(path string) (*State, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return &state, nil
}

func writeState(path string, state *State) error {
	var buf bytes.Buffer
	enc := newEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(state); err != nil {
		return err
	}
	// Trailing newline trimmed to stay byte-identical to the json.dump output
	// this replaced, so a re-run over an existing out-dir shows no diff.
	return os.WriteFile(path, bytes.TrimRight(buf.Bytes(), "\n"), 0o644)
}

// newEncoder leaves <, > and & alone: PR titles carry them and nothing
// downstream renders this output as HTML.
func newEncoder(w io.Writer) *json.Encoder {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc
}

func (f *Fetcher) logf(format string, args ...any) {
	if f.Log == nil {
		return
	}
	_, _ = fmt.Fprintf(f.Log, format+"\n", args...)
}

func splitRepo(repo string) (string, string, error) {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok || owner == "" || name == "" {
		return "", "", fmt.Errorf("repo %q is not owner/name", repo)
	}
	return owner, name, nil
}

func parseTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// isoformat matches Python's datetime.isoformat on a UTC-aware value: an
// explicit +00:00 rather than Z, and a 6-digit fraction only when non-zero.
func isoformat(t time.Time) string {
	t = t.UTC().Truncate(time.Microsecond)
	if t.Nanosecond() == 0 {
		return t.Format("2006-01-02T15:04:05") + "+00:00"
	}
	return t.Format("2006-01-02T15:04:05.000000") + "+00:00"
}

func optIsoformat(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	s := isoformat(t)
	return &s
}

func optTime(t time.Time) string {
	if t.IsZero() {
		return "none"
	}
	return isoformat(t)
}

func optInt(v *int) string {
	if v == nil {
		return "none"
	}
	return fmt.Sprint(*v)
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

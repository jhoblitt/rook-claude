package checklist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtanalyze"
	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/sweepprefetch"
)

// SweepFile is what a sweep writes, next to the snapshot it audited.
const SweepFile = "checklist.jsonl"

// SkipsFile is the sweep's second output, next to SweepFile: one row per PR the
// pass left out, which is the dashboard's skipped section.
const SkipsFile = "skips.json"

// VerdictUnaudited marks a PR the pass never reached a verdict on, because its
// body never arrived. Audit never returns it; what it means, and what it is
// not, is cmd/validate-checklist's package doc.
const VerdictUnaudited Verdict = "unaudited"

// The classes a sweep leaves out, as rook-triage's pr-triage.md "Skip
// conditions" defines them. SkipCarried is named for the ledger status it
// reads: a card a prior run assessed is not re-assessed.
const (
	SkipCarried    = "carried"
	SkipDoNotMerge = "do-not-merge"
	SkipDraft      = "draft"
	SkipWIP        = "wip"
	SkipBot        = "bot"
)

var skipOrder = []string{SkipCarried, SkipDoNotMerge, SkipDraft, SkipWIP, SkipBot}

var verdictOrder = []Verdict{VerdictConforming, VerdictNonConforming, VerdictNoChecklist, VerdictUnaudited}

// wipRe matches the marker as a standalone token, so a title only counts as
// work-in-progress when it says so.
var wipRe = regexp.MustCompile(`(?i)\bwip\b`)

// BodyFetcher hands back one PR's body. It is the pass's only network seam.
type BodyFetcher interface {
	Body(ctx context.Context, repo string, number int) (string, error)
}

type SweepOptions struct {
	// SweepDir holds the snapshot to audit and receives SweepFile.
	SweepDir string
	// Template is the checklist every selected body is compared against.
	Template []Item
	// IncludeDrafts widens the pass to draft PRs, and to nothing else.
	IncludeDrafts bool
	// Jobs bounds how many bodies are in flight at once; under 1 means 1.
	Jobs  int
	Fetch BodyFetcher
	// Errs receives one line per PR the pass could not audit — the operator's
	// only sign of a broken fetch. Nil discards them. PR-body text never
	// reaches it.
	Errs io.Writer
}

// SweepRow is one line of SweepFile, and the whole of what a sweep publishes
// about a PR it audited.
type SweepRow struct {
	Number  int     `json:"number"`
	Verdict Verdict `json:"verdict"`
}

// SweepSkip is one row of SkipsFile. The shape is prdash.Skip's, the type that
// reads the file; Title is the snapshot's own, so the dashboard names the PR a
// skip row stands for. It is contributor-authored text, and SkipsFile is the
// one output of this pass that may carry any: prdash escapes every value it
// renders, while SweepFile is read by triagers with no jq to project text back
// out.
type SweepSkip struct {
	Number int    `json:"number"`
	Class  string `json:"class"`
	Author string `json:"author"`
	Title  string `json:"title"`
}

type SweepResult struct {
	Path     string
	Audited  int
	Verdicts map[Verdict]int
	Skipped  map[string]int
}

func (r SweepResult) String() string {
	var verdicts []string
	for _, v := range verdictOrder {
		if n := r.Verdicts[v]; n > 0 {
			verdicts = append(verdicts, fmt.Sprintf("%d %s", n, v))
		}
	}
	var classes []string
	skipped := 0
	for _, c := range skipOrder {
		if n := r.Skipped[c]; n > 0 {
			skipped += n
			classes = append(classes, fmt.Sprintf("%d %s", n, c))
		}
	}
	out := fmt.Sprintf("%s: %d audited", r.Path, r.Audited)
	if len(verdicts) > 0 {
		out += " (" + strings.Join(verdicts, ", ") + ")"
	}
	out += fmt.Sprintf(", %d skipped", skipped)
	if len(classes) > 0 {
		out += " (" + strings.Join(classes, ", ") + ")"
	}
	return out
}

// Sweep audits every assessable PR of a rook-triage sweep, writing SweepFile
// for what it audited and SkipsFile for what it left out. What it audits, what
// a verdict means and what the pass exits is cmd/validate-checklist's package
// doc.
//
// The verdict is the only thing that leaves the audit: the report's Lines
// reproduce the body verbatim, and the file triagers read carries no jq to
// project them back out.
func Sweep(ctx context.Context, opts SweepOptions) (SweepResult, error) {
	res := SweepResult{
		Path:     filepath.Join(opts.SweepDir, SweepFile),
		Verdicts: map[Verdict]int{},
		Skipped:  map[string]int{},
	}
	if opts.Fetch == nil {
		return res, errors.New("no fetcher: a sweep reads every body it audits")
	}
	if len(opts.Template) == 0 {
		return res, errors.New("the template carries no checklist items")
	}
	if opts.Errs == nil {
		opts.Errs = io.Discard
	}

	snap, err := readSnapshot(opts.SweepDir)
	if err != nil {
		return res, err
	}
	carried, err := readCarried(opts.SweepDir)
	if err != nil {
		return res, err
	}
	numbers, skips, err := selectPRs(snap.Items, carried, opts.IncludeDrafts)
	if err != nil {
		return res, err
	}
	res.Skipped = skipCounts(skips)

	// The skips go down before the first fetch: a sweep dir that cannot take
	// them is a failed run, and finding that out after the network is spent
	// buys nothing.
	if err := writeSkips(filepath.Join(opts.SweepDir, SkipsFile), skips); err != nil {
		return res, err
	}

	f, err := os.OpenFile(res.Path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644) // #nosec G304 -- the caller names the sweep dir
	if err != nil {
		return res, err
	}
	if err := auditPool(ctx, opts, snap.Repo, numbers, f, &res); err != nil {
		return res, err
	}
	return res, nil
}

// auditPool audits numbers into res, writing a row per PR to out, and closes
// out. A close failure is the pass's failure too: on a network filesystem a
// deferred write reports its ENOSPC or EDQUOT only there, and discarding it
// would exit 0 over a truncated file.
//
// The first write failure stops the feed and cancels the fetches still in
// flight, since every further body buys a verdict that cannot be written.
func auditPool(ctx context.Context, opts SweepOptions, repo string, numbers []int, out io.WriteCloser, res *SweepResult) (err error) {
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		failure error
	)
	halt := make(chan struct{})
	work := make(chan int)
	for range min(max(opts.Jobs, 1), len(numbers)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := range work {
				verdict, why := verdictFor(ctx, opts, repo, n)

				mu.Lock()
				if why != nil {
					_, _ = fmt.Fprintf(opts.Errs, "validate-checklist: %d: %v\n", n, why)
				}
				res.Audited++
				res.Verdicts[verdict]++
				if werr := writeRow(out, SweepRow{Number: n, Verdict: verdict}); werr != nil && failure == nil {
					failure = werr
					close(halt)
					cancel()
				}
				mu.Unlock()
			}
		}()
	}
feed:
	for _, n := range numbers {
		// The first select gives an already-raised halt priority: in the second
		// one, a worker waiting for the next number is an equally ready case
		// and the runtime picks between them at random.
		select {
		case <-halt:
			break feed
		default:
		}
		select {
		case <-halt:
			break feed
		case work <- n:
		}
	}
	close(work)
	wg.Wait()

	return failure
}

// writeSkips publishes the pool's skip rows. SkipCarried is left out: it is
// this pass's own reason for not re-auditing a card, not a skip the report
// shows, and the run that assessed the card publishes it as an assessment.
func writeSkips(path string, skips []SweepSkip) error {
	rows := make([]SweepSkip, 0, len(skips))
	for _, s := range skips {
		if s.Class != SkipCarried {
			rows = append(rows, s)
		}
	}
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func skipCounts(skips []SweepSkip) map[string]int {
	counts := map[string]int{}
	for _, s := range skips {
		counts[s.Class]++
	}
	return counts
}

// verdictFor reaches a verdict for one PR. The error it returns says why a
// verdict could not be reached; it carries gh's own diagnostics, never the body.
func verdictFor(ctx context.Context, opts SweepOptions, repo string, number int) (Verdict, error) {
	body, err := opts.Fetch.Body(ctx, repo, number)
	if err != nil {
		return VerdictUnaudited, err
	}
	if len(body) > MaxBody {
		return VerdictUnaudited, fmt.Errorf("the body is over %d bytes; GitHub caps a PR body at 65536 characters", MaxBody)
	}
	return Audit(opts.Template, body).Verdict, nil
}

func writeRow(w io.Writer, row SweepRow) error {
	data, err := json.Marshal(row)
	if err != nil {
		return err
	}
	_, err = w.Write(append(data, '\n'))
	return err
}

type snapshotDoc struct {
	Repo  string                          `json:"repo"`
	Kind  string                          `json:"kind"`
	Items map[string]sweepprefetch.PRItem `json:"items"`
}

func readSnapshot(dir string) (snapshotDoc, error) {
	var doc snapshotDoc
	path := filepath.Join(dir, "snapshot.json")
	data, err := os.ReadFile(path) // #nosec G304 -- the caller names the sweep dir
	if err != nil {
		return doc, err
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return doc, fmt.Errorf("%s: %w", path, err)
	}
	if doc.Kind != "prs" {
		return doc, fmt.Errorf("%s: kind is %q, and only a pull request carries a template checklist", path, doc.Kind)
	}
	if doc.Repo == "" {
		return doc, fmt.Errorf("%s: no repo, so the pull requests to fetch cannot be named", path)
	}
	if len(doc.Items) == 0 {
		return doc, fmt.Errorf("%s: no items", path)
	}
	return doc, nil
}

// readCarried reads the sweep's per-item ledger. A sweep dir with no ledger at
// all is a run that has assessed nothing, so nothing is carried; a ledger that
// cannot be read is an error rather than that same answer.
func readCarried(dir string) (map[int]bool, error) {
	path := filepath.Join(dir, "sweep.json")
	data, err := os.ReadFile(path) // #nosec G304 -- the caller names the sweep dir
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var doc struct {
		Items map[string]string `json:"items"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	out := map[int]bool{}
	for key, status := range doc.Items {
		n, err := strconv.Atoi(key)
		if err != nil || status != SkipCarried {
			continue
		}
		out[n] = true
	}
	return out, nil
}

// selectPRs returns the numbers to audit and the rows for what it left out,
// both in ascending order.
func selectPRs(items map[string]sweepprefetch.PRItem, carried map[int]bool, includeDrafts bool) ([]int, []SweepSkip, error) {
	var (
		numbers []int
		skips   []SweepSkip
	)
	for key, it := range items {
		if it.Number <= 0 {
			return nil, nil, fmt.Errorf("snapshot item %q carries no number", key)
		}
		if class := skipClass(it, carried, includeDrafts); class != "" {
			skips = append(skips, SweepSkip{
				Number: it.Number,
				Class:  class,
				Author: it.Author,
				Title:  it.Title,
			})
			continue
		}
		numbers = append(numbers, it.Number)
	}
	slices.Sort(numbers)
	slices.SortFunc(skips, func(a, b SweepSkip) int { return a.Number - b.Number })
	return numbers, skips, nil
}

// skipClass names why a PR is out of the assessable set, or "" when it is in.
// A PR that is several of these at once takes the first, which decides only
// which row it lands in.
func skipClass(it sweepprefetch.PRItem, carried map[int]bool, includeDrafts bool) string {
	switch {
	case carried[it.Number]:
		return SkipCarried
	case hasLabel(it.Labels, SkipDoNotMerge):
		return SkipDoNotMerge
	case it.IsDraft && !includeDrafts:
		return SkipDraft
	case wipRe.MatchString(it.Title) || hasLabel(it.Labels, SkipWIP):
		return SkipWIP
	// pr-triage.md's mergify-backport class lands here: the app authors those
	// PRs as mergify[bot], and the head branch that names them as backports is
	// not in the snapshot to match on.
	case rtanalyze.IsBot(it.Author):
		return SkipBot
	}
	return ""
}

func hasLabel(labels []string, want string) bool {
	for _, l := range labels {
		if strings.EqualFold(l, want) {
			return true
		}
	}
	return false
}

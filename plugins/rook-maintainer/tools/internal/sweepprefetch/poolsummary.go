package sweepprefetch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtanalyze"
)

// topN caps every "top" list. Eight names is what fits on one line of the
// markdown block without wrapping in a chat pane.
const topN = 8

const (
	botLabel      = "bot"
	unknownLabel  = "unknown"
	deletedAuthor = "(deleted)"
	middot        = "·"
)

var ageBuckets = []struct {
	label string
	under time.Duration
}{
	{"<7d", 7 * 24 * time.Hour},
	{"7-30d", 30 * 24 * time.Hour},
	{"30-90d", 90 * 24 * time.Hour},
	{">90d", 0},
}

// assocOrder fixes the row order of the authorAssociation axis so the same row
// sits in the same place across sweeps. It is GitHub's association ladder from
// most to least attached, then the two labels this package mints itself.
var assocOrder = []string{
	"OWNER", "MEMBER", "COLLABORATOR", "CONTRIBUTOR",
	"FIRST_TIME_CONTRIBUTOR", "FIRST_TIMER", "MANNEQUIN", "NONE",
	botLabel, unknownLabel,
}

var stateOrder = []string{"OPEN", "MERGED", "CLOSED"}

type SummaryOptions struct {
	SweepDir string
	// Viewer is a login whose existing reviews are counted. Empty means the
	// aggregate is left out entirely, which is not the same fact as zero.
	Viewer string
	// Now is the instant the age buckets are measured from; pin it to keep two
	// runs comparable.
	Now time.Time
	// Numbers restricts the summary to those items. Empty summarizes the whole
	// snapshot; a number the snapshot does not carry is an error.
	Numbers []int
}

// Bucket is one row of a breakdown.
type Bucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// DiffTotals sums the corpus's churn. Missing counts items whose snapshot
// carries a null for any of the three, so a small total cannot be mistaken for
// a small pool.
type DiffTotals struct {
	Additions    int `json:"additions"`
	Deletions    int `json:"deletions"`
	ChangedFiles int `json:"changed_files"`
	Missing      int `json:"missing"`
}

type CommentTotals struct {
	Total int `json:"total"`
	None  int `json:"none"`
}

// PoolSummary is the whole of what pool-summary reports; its JSON field names
// are the machine-readable contract of --json.
type PoolSummary struct {
	SweepDir  string `json:"sweep_dir"`
	Snapshot  string `json:"snapshot"`
	Kind      string `json:"kind"`
	Repo      string `json:"repo"`
	FetchedAt string `json:"fetched_at"`
	Now       string `json:"now"`
	Total     int    `json:"total"`
	// SelectedFrom is the size of the whole snapshot, and is present only when
	// Numbers selected a subset of it.
	SelectedFrom int      `json:"selected_from,omitempty"`
	States       []Bucket `json:"states"`
	// Drafts is nil for an issues corpus, which has no such state, and set —
	// zero included — for a pull request corpus. It is load-bearing: a sweep
	// sizes its fan-out off the pool, and its own pre-gate drops every draft.
	Drafts *int   `json:"drafts,omitempty"`
	Viewer string `json:"viewer,omitempty"`
	// ReviewedByViewer is nil when no viewer was named.
	ReviewedByViewer  *int           `json:"reviewed_by_viewer,omitempty"`
	Age               []Bucket       `json:"age"`
	AuthorAssociation []Bucket       `json:"author_association,omitempty"`
	AuthorsDistinct   int            `json:"authors_distinct"`
	TopAuthors        []Bucket       `json:"top_authors"`
	LabelsDistinct    int            `json:"labels_distinct"`
	TopLabels         []Bucket       `json:"top_labels"`
	Unlabeled         int            `json:"unlabeled"`
	Diff              *DiffTotals    `json:"diff,omitempty"`
	Comments          *CommentTotals `json:"comments,omitempty"`
}

type poolDoc struct {
	FetchedAt string                     `json:"fetched_at"`
	Repo      string                     `json:"repo"`
	Kind      string                     `json:"kind"`
	Items     map[string]json.RawMessage `json:"items"`
}

// Summarize aggregates one <sweep-dir>/snapshot.json into the pool-wide counts
// a rook-code-review or rook-triage sweep opens phase 0 with: how big the
// corpus is, how much of it the viewer has already reviewed, how it splits by
// age, author and label, and how much churn it carries.
//
// It reads the snapshot and nothing else — no network, no gh — so re-running it
// is free and a sweep dir summarizes long after the session that fetched it. An
// unreadable, unparseable or item-less snapshot is an error: a summary nobody
// could compute must never come back looking like an empty pool.
//
// Which aggregates exist follows the snapshot's own "kind", never a flag. An
// issues corpus has no reviews, no authorAssociation, no draft state and no
// diff, so those are absent rather than zero, and it reports comment totals
// instead.
//
// Numbers narrows the pool to a subset — the filtered pool a sweep puts in
// front of the maintainer once its own filters have run. Every requested number
// must be in the snapshot; one that is not aborts the summary rather than
// quietly shrinking the pool, and SelectedFrom then records what the subset was
// drawn from so a filtered block cannot be mistaken for the whole corpus.
//
// Decisions callers depend on:
//
//   - Age buckets are half-open on Now - createdAt: <7d is [0, 7d), 7-30d is
//     [7d, 30d), 30-90d is [30d, 90d), >90d is everything at or past 90d.
//   - Drafts are counted but not excluded. The pool is what the sweep was
//     handed; how many of it is draft is what keeps a cost estimate honest,
//     since the sweep's pre-gate skips them.
//   - Bot authors take a "bot" row on the authorAssociation axis in place of
//     their own association, because "19 of these are mergify" is the fact
//     phase 0 acts on.
//   - Top lists (labels, authors) are capped at topN and ordered by count then
//     label; LabelsDistinct and AuthorsDistinct keep the full name counts, so a
//     reader is told how many the cap hid.
//   - An item whose author was deleted carries no login at all and counts as
//     "(deleted)", never as a blank row.
//   - Viewer matching is case-insensitive, GitHub logins being so.
func Summarize(opts SummaryOptions) (*PoolSummary, error) {
	path := filepath.Join(opts.SweepDir, "snapshot.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc poolDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if doc.Kind != "prs" && doc.Kind != "issues" {
		return nil, fmt.Errorf("%s: kind is %q, want prs or issues", path, doc.Kind)
	}
	if opts.Viewer != "" && doc.Kind != "prs" {
		return nil, fmt.Errorf("%s: kind is %q, so nothing in it can carry a review by %s",
			path, doc.Kind, opts.Viewer)
	}
	if len(doc.Items) == 0 {
		return nil, fmt.Errorf("%s: no items", path)
	}
	items, err := selection(path, doc.Items, opts.Numbers)
	if err != nil {
		return nil, err
	}

	s := &PoolSummary{
		SweepDir:  opts.SweepDir,
		Snapshot:  path,
		Kind:      doc.Kind,
		Repo:      doc.Repo,
		FetchedAt: doc.FetchedAt,
		Now:       opts.Now.UTC().Format(time.RFC3339),
		Total:     len(items),
	}
	if len(items) != len(doc.Items) {
		s.SelectedFrom = len(doc.Items)
	}
	states := map[string]int{}
	authors := map[string]int{}
	labels := map[string]int{}
	assoc := map[string]int{}
	ages := make([]int, len(ageBuckets))
	drafts, reviewed := 0, 0
	diff := DiffTotals{}
	comments := CommentTotals{}

	for _, key := range itemKeys(items) {
		var (
			state, createdAt, author string
			names                    []string
		)
		if doc.Kind == "prs" {
			var it PRItem
			if err := json.Unmarshal(items[key], &it); err != nil {
				return nil, fmt.Errorf("%s: item %s: %w", path, key, err)
			}
			state, createdAt, author, names = it.State, it.CreatedAt, it.Author, it.Labels
			if it.IsDraft {
				drafts++
			}
			assoc[assocLabel(it.Author, it.AuthorAssociation)]++
			addPtr(&diff.Additions, it.Additions)
			addPtr(&diff.Deletions, it.Deletions)
			addPtr(&diff.ChangedFiles, it.ChangedFiles)
			if it.Additions == nil || it.Deletions == nil || it.ChangedFiles == nil {
				diff.Missing++
			}
			if opts.Viewer != "" && reviewedBy(it.Reviews.Latest, opts.Viewer) {
				reviewed++
			}
		} else {
			var it IssueItem
			if err := json.Unmarshal(items[key], &it); err != nil {
				return nil, fmt.Errorf("%s: item %s: %w", path, key, err)
			}
			state, createdAt, author, names = it.State, it.CreatedAt, it.Author, it.Labels
			comments.Total += it.CommentsTotal
			if it.CommentsTotal == 0 {
				comments.None++
			}
		}

		created, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("%s: item %s: createdAt %q: %w", path, key, createdAt, err)
		}
		ages[ageBucket(opts.Now.Sub(created))]++
		states[orUnknown(state)]++
		authors[authorOrDeleted(author)]++
		if len(names) == 0 {
			s.Unlabeled++
		}
		for _, name := range names {
			labels[name]++
		}
	}

	s.States = orderedBuckets(states, stateOrder)
	s.Age = make([]Bucket, len(ageBuckets))
	for i, b := range ageBuckets {
		s.Age[i] = Bucket{Label: b.label, Count: ages[i]}
	}
	s.AuthorsDistinct, s.TopAuthors = len(authors), topBuckets(authors)
	s.LabelsDistinct, s.TopLabels = len(labels), topBuckets(labels)
	if doc.Kind == "prs" {
		s.AuthorAssociation = orderedBuckets(assoc, assocOrder)
		s.Drafts = &drafts
		s.Diff = &diff
		if opts.Viewer != "" {
			s.Viewer = opts.Viewer
			s.ReviewedByViewer = &reviewed
		}
	} else {
		s.Comments = &comments
	}
	return s, nil
}

// selection narrows items to numbers. A number the snapshot does not carry
// aborts the summary: the alternative is a pool quietly smaller than the one
// the caller asked about, which is the failure this whole subcommand exists to
// stop a sweep from making.
func selection(path string, items map[string]json.RawMessage, numbers []int) (map[string]json.RawMessage, error) {
	if len(numbers) == 0 {
		return items, nil
	}
	var missing []int
	picked := make(map[string]json.RawMessage, len(numbers))
	for _, n := range numbers {
		key := strconv.Itoa(n)
		raw, ok := items[key]
		if !ok {
			missing = append(missing, n)
			continue
		}
		picked[key] = raw
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%s: %d of the %d requested items are not in the snapshot: %s",
			path, len(missing), len(numbers), joinNumbers(missing))
	}
	return picked, nil
}

func joinNumbers(numbers []int) string {
	const shown = 10
	tail := ""
	head := numbers
	if len(head) > shown {
		head, tail = head[:shown], fmt.Sprintf(" and %d more", len(numbers)-shown)
	}
	parts := make([]string, 0, len(head))
	for _, n := range head {
		parts = append(parts, strconv.Itoa(n))
	}
	return strings.Join(parts, ", ") + tail
}

// JSON renders the summary the way this package writes every other file: a
// one-space indent, HTML metacharacters left alone, no trailing newline.
func (s *PoolSummary) JSON() ([]byte, error) {
	return Encode(s)
}

// Markdown renders the block a sweep's phase 0 presents as-is. Counts carry
// thousands separators here and nowhere else; --json stays raw.
func (s *PoolSummary) Markdown() string {
	unit, unitOne, column := "PRs", "PR", "PRs"
	if s.Kind == "issues" {
		unit, unitOne, column = "issues", "issue", "Issues"
	}
	noun := unit
	if s.Total == 1 {
		noun = unitOne
	}

	var lines []string
	head := fmt.Sprintf("**Pool: %s %s**", comma(s.Total), noun)
	if onlyOpen(s.States) {
		head = fmt.Sprintf("**Pool: %s open %s**", comma(s.Total), noun)
	} else if len(s.States) > 0 {
		parts := make([]string, 0, len(s.States))
		for _, b := range s.States {
			parts = append(parts, fmt.Sprintf("%s %s", comma(b.Count), b.Label))
		}
		head += " (" + strings.Join(parts, ", ") + ")"
	}
	var notes []string
	if s.SelectedFrom > 0 {
		notes = append(notes, fmt.Sprintf("of %s in the snapshot", comma(s.SelectedFrom)))
	}
	// A draft is work the sweep's own pre-gate will skip, so a pool sized
	// without it buys fan-out for PRs nobody will review.
	if s.Drafts != nil && *s.Drafts > 0 {
		notes = append(notes, plural(*s.Drafts, "draft", "drafts"))
	}
	if s.ReviewedByViewer != nil {
		carry := "carry"
		if *s.ReviewedByViewer == 1 {
			carry = "carries"
		}
		notes = append(notes, fmt.Sprintf("%s already %s a review from %s",
			comma(*s.ReviewedByViewer), carry, s.Viewer))
	}
	if len(notes) > 0 {
		head += " (" + strings.Join(notes, ", ") + ")"
	}
	lines = append(lines, head, "")

	right, rightHead := s.AuthorAssociation, "Author assoc"
	if s.Kind == "issues" {
		// The authors axis is the table's right column here, so it carries no
		// "+N more" the way the prs-only "Top authors:" line does. Say what the
		// cap hid, in the header, rather than showing a truncated list as if it
		// were the whole set.
		right, rightHead = s.TopAuthors, "Author"
		if more := s.AuthorsDistinct - len(s.TopAuthors); more > 0 {
			rightHead = fmt.Sprintf("Author (%s of %s)", comma(len(s.TopAuthors)), comma(s.AuthorsDistinct))
		}
	}
	lines = append(lines,
		fmt.Sprintf("| Age | %s |  | %s | %s |", column, rightHead, column),
		"|---|---|---|---|---|")
	for i := range max(len(s.Age), len(right)) {
		age, ageCount := cell(s.Age, i)
		label, count := cell(right, i)
		lines = append(lines, fmt.Sprintf("| %s | %s |  | %s | %s |", age, ageCount, label, count))
	}
	lines = append(lines, "")

	if s.Kind == "prs" {
		lines = append(lines, "Top authors: "+inline(s.TopAuthors, s.AuthorsDistinct))
	}
	labels := "Top labels: " + inline(s.TopLabels, s.LabelsDistinct)
	if s.Unlabeled > 0 {
		labels += fmt.Sprintf(" %s %s with no labels", middot, comma(s.Unlabeled))
	}
	lines = append(lines, labels)
	if s.Diff != nil {
		diff := fmt.Sprintf("Diff size: +%s / -%s over %s files",
			comma(s.Diff.Additions), comma(s.Diff.Deletions), comma(s.Diff.ChangedFiles))
		if s.Diff.Missing > 0 {
			diff += fmt.Sprintf(" %s diff size missing on %s", middot, plural(s.Diff.Missing, unitOne, unit))
		}
		lines = append(lines, diff)
	}
	if s.Comments != nil {
		line := fmt.Sprintf("Comments: %s total", comma(s.Comments.Total))
		if s.Comments.None > 0 {
			line += fmt.Sprintf(" %s %s with none", middot, plural(s.Comments.None, unitOne, unit))
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// onlyOpen reports whether the header can call the pool open. A snapshot taken
// by number carries merged and closed items too, and calling those open would
// misstate the size of the work in front of the sweep.
func onlyOpen(states []Bucket) bool {
	return len(states) == 1 && states[0].Label == "OPEN"
}

func cell(buckets []Bucket, i int) (label, count string) {
	if i >= len(buckets) {
		return "", ""
	}
	return buckets[i].Label, comma(buckets[i].Count)
}

func inline(buckets []Bucket, distinct int) string {
	if len(buckets) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(buckets)+1)
	for _, b := range buckets {
		parts = append(parts, fmt.Sprintf("%s(%s)", b.Label, comma(b.Count)))
	}
	if more := distinct - len(buckets); more > 0 {
		parts = append(parts, fmt.Sprintf("+%s more", comma(more)))
	}
	return strings.Join(parts, " ")
}

func plural(n int, one, many string) string {
	if n == 1 {
		return comma(n) + " " + one
	}
	return comma(n) + " " + many
}

func comma(n int) string {
	digits := strconv.Itoa(n)
	sign := ""
	if strings.HasPrefix(digits, "-") {
		sign, digits = "-", digits[1:]
	}
	var b strings.Builder
	for i := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteByte(digits[i])
	}
	return sign + b.String()
}

// itemKeys walks the items in number order so that a malformed item always
// reports the same one first.
func itemKeys(items map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b string) int {
		x, errA := strconv.Atoi(a)
		y, errB := strconv.Atoi(b)
		if errA != nil || errB != nil {
			return strings.Compare(a, b)
		}
		return x - y
	})
	return keys
}

func sortedBuckets(counts map[string]int) []Bucket {
	buckets := make([]Bucket, 0, len(counts))
	for label, n := range counts {
		buckets = append(buckets, Bucket{Label: label, Count: n})
	}
	slices.SortFunc(buckets, func(a, b Bucket) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		return strings.Compare(a.Label, b.Label)
	})
	return buckets
}

func topBuckets(counts map[string]int) []Bucket {
	buckets := sortedBuckets(counts)
	return buckets[:min(topN, len(buckets))]
}

// orderedBuckets emits the labels of order that occur, in that order, then
// anything unforeseen by count.
func orderedBuckets(counts map[string]int, order []string) []Bucket {
	buckets := make([]Bucket, 0, len(counts))
	rest := make(map[string]int, len(counts))
	for label, n := range counts {
		rest[label] = n
	}
	for _, label := range order {
		if n, ok := rest[label]; ok {
			buckets = append(buckets, Bucket{Label: label, Count: n})
			delete(rest, label)
		}
	}
	return append(buckets, sortedBuckets(rest)...)
}

func ageBucket(age time.Duration) int {
	for i, b := range ageBuckets {
		if b.under != 0 && age < b.under {
			return i
		}
	}
	return len(ageBuckets) - 1
}

func assocLabel(author string, assoc *string) string {
	if rtanalyze.IsBot(author) {
		return botLabel
	}
	if assoc == nil || *assoc == "" {
		return unknownLabel
	}
	return *assoc
}

func reviewedBy(reviews []Review, viewer string) bool {
	for _, r := range reviews {
		if strings.EqualFold(r.Login, viewer) {
			return true
		}
	}
	return false
}

func authorOrDeleted(author string) string {
	if author == "" {
		return deletedAuthor
	}
	return author
}

func orUnknown(s string) string {
	if s == "" {
		return unknownLabel
	}
	return s
}

func addPtr(total *int, n *int) {
	if n != nil {
		*total += *n
	}
}

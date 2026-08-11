// Package issuesdash builds the rook-triage issues sweep dashboard from a
// sweep directory (format contract: skills/rook-triage/references/reporting.md).
//
// The only inputs are canonical sweep-dir files: snapshot.json for the live
// title, labels and assignees of every issue, batch-*.json for the triager's
// assessments, refs-types.json for the Issue-vs-PullRequest classification of
// cross-references, and issues-mentions.json for mined @-mentions. A dashboard
// is therefore reproducible from the sweep dir alone, after the session that
// produced it is gone.
//
// Every value that reaches the page comes from an issue reporter, so all of it
// is rendered through html/template; nothing is concatenated into HTML here.
package issuesdash

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	emDash      = "\u2014"
	maxMentions = 8
)

type Item struct {
	Number         json.Number       `json:"number"`
	Kind           string            `json:"kind"`
	Disposition    *string           `json:"disposition"`
	Actions        []string          `json:"actions"`
	LabelsProposed []string          `json:"labels_proposed"`
	Routing        []json.RawMessage `json:"routing"`
	XLinks         []json.RawMessage `json:"xlinks"`
	Dups           []json.RawMessage `json:"dups"`
}

type SnapItem struct {
	Title     string   `json:"title"`
	Labels    []string `json:"labels"`
	Assignees []string `json:"assignees"`
}

type Sweep struct {
	Dir      string
	Items    []Item
	Snapshot map[string]SnapItem
	Mentions map[string][]string
	RefTypes map[string]string
}

// Load reads the canonical inputs of one sweep directory. snapshot.json is
// required; refs-types.json and issues-mentions.json are produced by later
// steps of the sweep and may be absent.
func Load(dir string) (*Sweep, error) {
	s := &Sweep{Dir: dir}

	batches, err := filepath.Glob(filepath.Join(dir, "batch-*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(batches)
	for _, path := range batches {
		var items []Item
		if err := readJSON(path, &items); err != nil {
			return nil, err
		}
		s.Items = append(s.Items, items...)
	}
	for _, it := range s.Items {
		if it.Number == "" {
			return nil, errors.New("batch item without a number")
		}
		if it.Disposition == nil {
			return nil, fmt.Errorf("issue %s: no disposition", it.Number)
		}
	}
	sort.SliceStable(s.Items, func(i, j int) bool {
		return numLess(s.Items[i].Number, s.Items[j].Number)
	})

	snapPath := filepath.Join(dir, "snapshot.json")
	var snap struct {
		Items map[string]SnapItem `json:"items"`
	}
	if err := readJSON(snapPath, &snap); err != nil {
		return nil, err
	}
	if snap.Items == nil {
		return nil, fmt.Errorf("%s: no items", snapPath)
	}
	s.Snapshot = snap.Items

	if err := readOptionalJSON(filepath.Join(dir, "issues-mentions.json"), &s.Mentions); err != nil {
		return nil, err
	}
	if err := readOptionalJSON(filepath.Join(dir, "refs-types.json"), &s.RefTypes); err != nil {
		return nil, err
	}
	return s, nil
}

func readJSON(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func readOptionalJSON(path string, out any) error {
	err := readJSON(path, out)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func numLess(a, b json.Number) bool {
	x, errA := a.Float64()
	y, errB := b.Float64()
	if errA != nil || errB != nil {
		return a < b
	}
	return x < y
}

type Page struct {
	Date string
	Rows []Row
}

type Row struct {
	Class       string
	Number      string
	Kind        string
	Title       string
	Actions     List
	Disposition []Seg
	Refs        []string
	Assignees   []string
	Mentions    []MentionRow
	Labels      LabelsCell
}

type List struct {
	Items []string
	Bold  bool
}

type LabelsCell struct {
	Current  List
	Proposed List
}

// MentionRow is one line of the mentions cell: a login already mentioned in
// the thread, a login triage proposes to @-mention (Proposed), or the overflow
// marker for the mentions beyond maxMentions (More).
type MentionRow struct {
	Login    string
	More     int
	Proposed bool
}

// Seg is a run of disposition text, or an issue reference to link (Ref).
type Seg struct {
	Text string
	Ref  string
}

func (s *Sweep) Page() Page {
	p := Page{Date: SweepDate(s.Dir), Rows: make([]Row, 0, len(s.Items))}
	for _, it := range s.Items {
		p.Rows = append(p.Rows, s.row(it))
	}
	return p
}

func (s *Sweep) row(it Item) Row {
	n := it.Number.String()
	snap := s.Snapshot[n]
	return Row{
		Class:       Class(*it.Disposition),
		Number:      n,
		Kind:        it.Kind,
		Title:       snap.Title,
		Actions:     List{Items: it.Actions},
		Disposition: Linkify(*it.Disposition),
		Refs:        s.PullRefs(it),
		Assignees:   snap.Assignees,
		Mentions:    s.MentionRows(it),
		Labels: LabelsCell{
			Current:  List{Items: snap.Labels},
			Proposed: List{Items: it.LabelsProposed, Bold: true},
		},
	}
}

// SweepDate is the leading date of the sweep directory name.
func SweepDate(dir string) string {
	base := []rune(filepath.Base(dir))
	if len(base) > 10 {
		base = base[:10]
	}
	return string(base)
}

// PullRefs lists the cross-referenced items refs-types.json classifies as pull
// requests. The opposite-type filter is what makes the column a "pr #" column;
// an unclassified reference is dropped rather than guessed at.
func (s *Sweep) PullRefs(it Item) []string {
	var out []string
	seen := map[string]bool{}
	for _, raw := range slices.Concat(it.XLinks, it.Dups) {
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) != nil {
			continue
		}
		num, ok := obj["number"]
		if !ok || seen[string(num)] {
			continue
		}
		seen[string(num)] = true
		if v, ok := scalarString(num); ok && s.RefTypes[v] == "PullRequest" {
			out = append(out, v)
		}
	}
	return out
}

func (s *Sweep) MentionRows(it Item) []MentionRow {
	var rows []MentionRow
	mentioned := s.Mentions[it.Number.String()]
	for _, login := range mentioned[:min(len(mentioned), maxMentions)] {
		rows = append(rows, MentionRow{Login: login})
	}
	if len(mentioned) > maxMentions {
		rows = append(rows, MentionRow{More: len(mentioned) - maxMentions})
	}
	for _, raw := range it.Routing {
		login, ok := scalarString(raw)
		if !ok {
			login = string(raw)
		}
		rows = append(rows, MentionRow{Login: login, Proposed: true})
	}
	return rows
}

func scalarString(raw json.RawMessage) (string, bool) {
	t := strings.TrimSpace(string(raw))
	switch {
	case t == "":
		return "", false
	case t[0] == '"':
		var s string
		if json.Unmarshal(raw, &s) != nil {
			return "", false
		}
		return s, true
	case t[0] == '-' || (t[0] >= '0' && t[0] <= '9'):
		return t, true
	}
	return "", false
}

// Class maps a disposition to the row's color class, which also drives the
// legend filters.
func Class(disposition string) string {
	d := strings.ToLower(disposition)
	head, _, _ := strings.Cut(d, emDash)
	switch {
	case strings.HasPrefix(d, "needs-info"):
		return "info"
	case strings.Contains(d, "close") && (strings.Contains(d, "candidate") || strings.Contains(d, "propose")),
		hasAnyPrefix(d, "fixed-by-merged", "answered", "support", "adjudicated"):
		return "close"
	case strings.Contains(d, "fix-open"):
		return "fix"
	case strings.Contains(d, "blocked-upstream"), strings.Contains(head, "upstream"):
		return "up"
	}
	return "keep"
}

func hasAnyPrefix(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

var refRe = regexp.MustCompile(`#?(1[0-8][0-9]{3}|[4-9][0-9]{3})`)

// Linkify splits text into literal runs and rook issue references, so the
// template can link the references without the caller building HTML.
//
// The reference shape is the Python original's `#?\b(1[0-8]\d{3}|[4-9]\d{3})\b`:
// wide enough to catch bare numbers, narrow enough to leave years and version
// strings alone. Both word boundaries are re-checked here because Go's \b is
// ASCII-only: without that, a number glued to a non-ASCII letter would
// linkify where Python's Unicode \b does not.
func Linkify(text string) []Seg {
	var segs []Seg
	lit, pos := 0, 0
	for pos < len(text) {
		m := refRe.FindStringSubmatchIndex(text[pos:])
		if m == nil {
			break
		}
		start, end := pos+m[0], pos+m[1]
		numStart, numEnd := pos+m[2], pos+m[3]
		if (start == numStart && endsWithWord(text[:start])) || startsWithWord(text[numEnd:]) {
			_, w := utf8.DecodeRuneInString(text[start:])
			pos = start + w
			continue
		}
		if start > lit {
			segs = append(segs, Seg{Text: text[lit:start]})
		}
		segs = append(segs, Seg{Ref: text[numStart:numEnd]})
		lit, pos = end, end
	}
	if lit < len(text) {
		segs = append(segs, Seg{Text: text[lit:]})
	}
	return segs
}

func wordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
}

func endsWithWord(s string) bool {
	r, n := utf8.DecodeLastRuneInString(s)
	return n > 0 && wordRune(r)
}

func startsWithWord(s string) bool {
	r, n := utf8.DecodeRuneInString(s)
	return n > 0 && wordRune(r)
}

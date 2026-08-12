// Package prdash renders the rook-triage PR-sweep dashboard from the canonical
// sweep-dir inputs. Format contract: rook-triage/references/reporting.md.
//
// The inputs are snapshot.json (live metadata from sweep_prefetch: titles,
// labels, assignees, reviews, CI rollup), batch-*.json (triager assessments),
// refs-types.json and skips.json. CI cells are ALWAYS passing/total from the
// snapshot's statusCheckRollup summary — never parsed from triager prose,
// which is why a batch item's own `ci` text only ever reaches the tooltip.
//
// Every value that reaches the page comes from a PR body, title or comment
// written by anyone on the internet, so all of it is interpolated through
// html/template actions and none of it through string concatenation.
package prdash

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	emDash      = "—"
	ellipsis    = "…"
	middot      = "·"
	iconApprove = "✔"
	iconChanges = "±"
	iconComment = "💬"
	iconDismiss = "◌"
	iconRequest = "●"
	iconPropose = "◇"

	copilotLogin  = "copilot-pull-request-reviewer"
	dashboardFile = "dashboard.html"
	failedShown   = 6

	// classProposed marks the reviewer rows triage is proposing rather than
	// reporting; the ledger counts them against the per-person cap.
	classProposed = "prop"
)

// Text tolerates a non-string JSON scalar because the Python it replaces
// rendered whatever it found (batch files are model-written and drift).
type Text string

func (t *Text) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		*t = Text(b)
		return nil
	}
	*t = Text(s)
	return nil
}

// Flag reproduces Python truthiness: the batch fields it reads were tested
// with `if it.get(...)`, where "draft" and 1 are as true as true is.
type Flag bool

func (f *Flag) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch x := v.(type) {
	case bool:
		*f = Flag(x)
	case float64:
		*f = x != 0
	case string:
		*f = x != ""
	case []any:
		*f = len(x) != 0
	case map[string]any:
		*f = len(x) != 0
	}
	return nil
}

// Ref is one xlinks/dups entry. Anything that is not an object carrying a
// "number" is dropped rather than rejected, matching the isinstance guard in
// the Python: a triager that lists a bare number must not kill the render.
type Ref struct {
	Number string
	OK     bool
}

func (r *Ref) UnmarshalJSON(b []byte) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil
	}
	raw, ok := obj["number"]
	if !ok {
		return nil
	}
	r.Number, r.OK = scalar(raw), true
	return nil
}

// scalar keys a cross-reference the way refs-types.json is keyed: by the
// number's JSON text, so 13001 and "13001" resolve to the same entry.
func scalar(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

// Item is one triager assessment out of a batch-*.json array. CapNote records
// why a reviewer set had to be swapped; it reaches the markdown ledger only,
// since the dashboard has no room for it.
type Item struct {
	Number            int    `json:"number"`
	Kind              Text   `json:"kind"`
	Next              Text   `json:"next"`
	Disposition       Text   `json:"disposition"`
	CI                Text   `json:"ci"`
	Skip              Flag   `json:"skip"`
	Takeover          Flag   `json:"takeover"`
	XLinks            []Ref  `json:"xlinks"`
	Dups              []Ref  `json:"dups"`
	ReviewersProposed []Text `json:"reviewers_proposed"`
	CapNote           Text   `json:"cap_note"`
}

// Skip is one draft/bot row out of skips.json.
type Skip struct {
	Number int  `json:"number"`
	Class  Text `json:"class"`
	Author Text `json:"author"`
	Title  Text `json:"title"`
}

type CI struct {
	Total   int      `json:"total"`
	Passing int      `json:"passing"`
	Failing int      `json:"failing"`
	Pending int      `json:"pending"`
	Failed  []string `json:"failed"`
}

type Review struct {
	Login string `json:"login"`
	State string `json:"state"`
}

type Reviews struct {
	Latest    []Review `json:"latest"`
	Requested []string `json:"requested"`
}

// SnapItem is one item of snapshot.json's "items" map.
type SnapItem struct {
	Title     string   `json:"title"`
	Labels    []string `json:"labels"`
	Assignees []string `json:"assignees"`
	CI        *CI      `json:"ci"`
	Reviews   *Reviews `json:"reviews"`
}

// Sweep is the canonical sweep-dir input set. Batches keeps the file list, not
// just the items it parsed: a dir with no batch file at all renders the same
// empty table as a dir whose triager found nothing, and only the caller can
// say whether that is a finished sweep or a missing input.
type Sweep struct {
	Dir      string
	Date     string
	Batches  []string
	Items    []Item
	Snap     map[string]SnapItem
	Skips    []Skip
	RefTypes map[string]string
}

// Load reads a sweep dir. snapshot.json is mandatory — without live metadata
// there is no dashboard to render, only agent prose — while skips.json and
// refs-types.json are optional and default to empty.
func Load(dir string) (*Sweep, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	s := &Sweep{Dir: abs, Date: sweepDate(abs)}

	batches, err := filepath.Glob(filepath.Join(abs, "batch-*.json"))
	if err != nil {
		return nil, err
	}
	s.Batches = batches
	for _, b := range batches {
		var items []Item
		if err := readJSON(b, &items, true); err != nil {
			return nil, err
		}
		s.Items = append(s.Items, items...)
	}
	sort.SliceStable(s.Items, func(i, j int) bool { return s.Items[i].Number < s.Items[j].Number })

	var snap struct {
		Items map[string]SnapItem `json:"items"`
	}
	if err := readJSON(filepath.Join(abs, "snapshot.json"), &snap, true); err != nil {
		return nil, err
	}
	s.Snap = snap.Items
	if err := readJSON(filepath.Join(abs, "skips.json"), &s.Skips, false); err != nil {
		return nil, err
	}
	if err := readJSON(filepath.Join(abs, "refs-types.json"), &s.RefTypes, false); err != nil {
		return nil, err
	}
	return s, nil
}

func readJSON(path string, v any, required bool) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if !required && errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// sweepDate reads the date off a sweep dir named <YYYY-MM-DD>-<slug>.
func sweepDate(abs string) string {
	base := []rune(filepath.Base(abs))
	if len(base) > 10 {
		base = base[:10]
	}
	return string(base)
}

// Seg is one run of disposition/skip text: either prose or a linkified
// issue reference.
type Seg struct {
	Plain string
	Ref   string
}

type User struct {
	Href string
	Name string
}

type Chip struct {
	Kind  string
	Short string
	Tip   string
}

type Reviewer struct {
	User  User
	Class string
	Icon  string
	Title string
	Note  string
}

type Row struct {
	Class  string
	Number int
	Kind   string
	Title  string
	// Missing says the snapshot has no entry for this row: its title, labels,
	// assignees, reviews and CI counts are unavailable rather than empty, and
	// no renderer may present the difference as "none".
	Missing     bool
	Chip        Chip
	Next        string
	Disposition []Seg
	Refs        []string
	Assignees   []User
	Reviewers   []Reviewer
	Labels      []string
	CapNote     string
}

type SkipRow struct {
	Number int
	Class  string
	Author string
	Title  []Seg
}

type Page struct {
	Date     string
	Assessed []Row
	WIP      []Row
	Skipped  []SkipRow
}

// Page builds the view. WIP rows are assessed items the triager marked skip:
// they carry real assessment data but belong in the skipped section.
func (s *Sweep) Page() Page {
	p := Page{Date: s.Date}
	for _, it := range s.Items {
		row := s.row(it)
		if it.Skip {
			p.WIP = append(p.WIP, row)
		} else {
			p.Assessed = append(p.Assessed, row)
		}
	}
	for _, sk := range s.Skips {
		p.Skipped = append(p.Skipped, SkipRow{
			Number: sk.Number,
			Class:  string(sk.Class),
			Author: string(sk.Author),
			Title:  Linkify(string(sk.Title)),
		})
	}
	return p
}

func (s *Sweep) row(it Item) Row {
	snap, found := s.Snap[strconv.Itoa(it.Number)]
	next := string(it.Next)
	if next == "" {
		next = emDash
	}
	return Row{
		Class:       Class(it),
		Number:      it.Number,
		Kind:        string(it.Kind),
		Title:       snap.Title,
		Missing:     !found,
		Chip:        NewChip(snap.CI, string(it.CI)),
		Next:        next,
		Disposition: Linkify(string(it.Disposition)),
		Refs:        s.Refs(it),
		Assignees:   users(snap.Assignees),
		Reviewers:   Reviewers(it.ReviewersProposed, snap.Reviews),
		Labels:      snap.Labels,
		CapNote:     string(it.CapNote),
	}
}

// Class picks the row class, which drives both the left rule colour and the
// legend filter. Order is the disposition precedence, not a taxonomy.
func Class(it Item) string {
	next := string(it.Next)
	switch {
	case bool(it.Skip):
		return "wip"
	case strings.Contains(next, "close") ||
		strings.Contains(string(it.Disposition), "CLOSE CANDIDATE"):
		return "close"
	case bool(it.Takeover):
		return "take"
	case strings.Contains(next, "request-reviewers"):
		return "route"
	case strings.Contains(next, "merge"):
		return "ready"
	case containsAny(next, "comment", "rebase", "dup-link", "fill-template"):
		return "act"
	}
	return "mon"
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

var (
	verdictPrefix = regexp.MustCompile(`^(?:green|red)\b[:x ]*\d*[:, ]*`)
	countsOnly    = regexp.MustCompile(`\A[\d/,:. ]*(?:passing|failing)?\z`)
)

// CleanAssessment strips a triager's CI verdict down to whatever it says that
// the snapshot's own counts do not. "green 12/12" adds nothing to a chip that
// already reads 12/12; "flaky multus job" does.
func CleanAssessment(s string) string {
	s = strings.TrimSpace(verdictPrefix.ReplaceAllString(s, ""))
	if countsOnly.MatchString(s) {
		return ""
	}
	return s
}

// NewChip builds the CI cell from the snapshot summary, with the triager's
// surviving assessment appended to the tooltip.
func NewChip(ci *CI, assessment string) Chip {
	var c CI
	if ci != nil {
		c = *ci
	}
	chip := Chip{Kind: "amber", Short: ellipsis}
	var parts []string
	if c.Total == 0 {
		parts = append(parts, "no checks recorded")
	} else {
		chip.Short = fmt.Sprintf("%d/%d", c.Passing, c.Total)
		switch {
		case c.Failing != 0:
			chip.Kind = "red"
		case c.Pending != 0:
			chip.Kind = "amber"
		default:
			chip.Kind = "green"
		}
		parts = append(parts, chip.Short+" passing")
		if c.Failing != 0 {
			shown := c.Failed
			if len(shown) > failedShown {
				shown = shown[:failedShown]
			}
			part := fmt.Sprintf("%d failing: %s", c.Failing, strings.Join(shown, ", "))
			if more := len(c.Failed) - failedShown; more > 0 {
				part += fmt.Sprintf(" +%d more", more)
			}
			parts = append(parts, part)
		}
		if c.Pending != 0 {
			parts = append(parts, fmt.Sprintf("%d pending", c.Pending))
		}
	}
	chip.Tip = strings.Join(parts, " "+middot+" ")
	if a := CleanAssessment(assessment); a != "" {
		chip.Tip += " " + emDash + " assessment: " + a
	}
	return chip
}

// Refs lists the OPPOSITE-type cross-references of a row: only numbers a live
// issueOrPullRequest lookup classified as Issue reach the issue column.
func (s *Sweep) Refs(it Item) []string {
	var out []string
	seen := map[string]bool{}
	for _, list := range [][]Ref{it.XLinks, it.Dups} {
		for _, r := range list {
			if !r.OK || seen[r.Number] {
				continue
			}
			seen[r.Number] = true
			if s.RefTypes[r.Number] == "Issue" {
				out = append(out, r.Number)
			}
		}
	}
	return out
}

// Bare numbers become links only inside the live rook/rook range, so version
// numbers, byte counts and timeouts in a disposition stay plain text.
//
// The word boundaries the range needs are checked in wordRune rather than
// written as \b: RE2's \b is ASCII, so "café13000" would be a boundary to it
// and the middle of a word to a reader. The digits stay ASCII-only on purpose,
// unlike Python's Unicode \d — "13٠٠٠" is not issue 13000, and linking it
// would mint a dead URL out of a homoglyph.
var refPattern = regexp.MustCompile(`#?(1[3-8][0-9]{3})`)

// wordRune reports whether r is a word character to Python's re, whose \w on
// str is str.isalnum() plus underscore: a letter or number of any script.
func wordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
}

// Linkify splits prose into plain runs and rook issue references. It matches
// before escaping, which is safe because HTML escaping only ever introduces
// characters the pattern cannot match across: the split lands the same either
// way, and the plain runs still reach the page through an escaping action.
func Linkify(text string) []Seg {
	var segs []Seg
	last := 0
	for _, m := range refPattern.FindAllStringSubmatchIndex(text, -1) {
		// A leading '#' is itself the boundary, so only a bare number has to
		// prove it does not start inside a word.
		if text[m[0]] != '#' && m[0] > 0 {
			if r, _ := utf8.DecodeLastRuneInString(text[:m[0]]); wordRune(r) {
				continue
			}
		}
		if m[1] < len(text) {
			if r, _ := utf8.DecodeRuneInString(text[m[1]:]); wordRune(r) {
				continue
			}
		}
		if m[0] > last {
			segs = append(segs, Seg{Plain: text[last:m[0]]})
		}
		segs = append(segs, Seg{Ref: text[m[2]:m[3]]})
		last = m[1]
	}
	if last < len(text) {
		segs = append(segs, Seg{Plain: text[last:]})
	}
	return segs
}

func userOf(login string) User {
	if login == copilotLogin {
		return User{Href: "https://github.com/apps/" + copilotLogin, Name: "Copilot"}
	}
	return User{Href: "https://github.com/" + login, Name: login}
}

func users(logins []string) []User {
	out := make([]User, 0, len(logins))
	for _, l := range logins {
		out = append(out, userOf(l))
	}
	return out
}

var reviewStates = map[string]Reviewer{
	"APPROVED":          {Class: "ok", Icon: iconApprove, Title: "approved"},
	"CHANGES_REQUESTED": {Class: "chg", Icon: iconChanges, Title: "changes requested"},
	"COMMENTED":         {Class: "com", Icon: iconComment, Title: "commented"},
	"DISMISSED":         {Class: "dis", Icon: iconDismiss, Title: "review dismissed"},
}

var proposedPattern = regexp.MustCompile(`\A([\pL\pN_.-]+)\s*(?:\((.*)\))?`)

// Reviewers renders the reviewer ledger: pending requests first, then the
// latest review per person, then triage's own proposals.
func Reviewers(proposed []Text, rv *Reviews) []Reviewer {
	var order []string
	latest := map[string]string{}
	requested := map[string]bool{}
	if rv != nil {
		for _, r := range rv.Latest {
			if _, seen := latest[r.Login]; !seen {
				order = append(order, r.Login)
			}
			latest[r.Login] = r.State
		}
		for _, login := range rv.Requested {
			requested[login] = true
		}
	}

	var rows []Reviewer
	if rv != nil {
		for _, login := range rv.Requested {
			note := "review requested"
			if _, reviewed := latest[login]; reviewed {
				note = "re-requested"
			}
			rows = append(rows, Reviewer{User: userOf(login), Class: "pend",
				Icon: iconRequest, Title: note})
		}
	}
	for _, login := range order {
		if requested[login] {
			continue
		}
		state := latest[login]
		r, known := reviewStates[state]
		if !known {
			r = Reviewer{Class: "com", Icon: middot, Title: strings.ToLower(state)}
		}
		r.User = userOf(login)
		rows = append(rows, r)
	}
	for _, p := range proposed {
		login, note := splitProposal(string(p))
		title := "proposed reviewer (triage routing)"
		if note != "" {
			title = "proposed reviewer " + emDash + " " + note
		}
		rows = append(rows, Reviewer{User: userOf(login), Class: classProposed,
			Icon: iconPropose, Title: title, Note: note})
	}
	return rows
}

func splitProposal(s string) (login, note string) {
	m := proposedPattern.FindStringSubmatch(s)
	if m == nil {
		return s, ""
	}
	return m[1], m[2]
}

var tmpl = template.Must(template.Must(
	template.New("page").Parse(pageTemplate)).Parse(partialTemplates))

func Render(w io.Writer, p Page) error {
	return tmpl.Execute(w, p)
}

// Generate writes <dir>/dashboard.html and reports the row counts on log.
// Rendering completes before the file is touched: a half-written dashboard at
// a URL a maintainer already has open is worse than no new dashboard.
func Generate(dir string, log io.Writer) error {
	s, err := Load(dir)
	if err != nil {
		return err
	}
	page := s.Page()
	var buf bytes.Buffer
	if err := Render(&buf, page); err != nil {
		return err
	}
	out := filepath.Join(s.Dir, dashboardFile)
	if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
		return err
	}
	_, err = fmt.Fprintf(log, "%s: %d assessed, %d WIP, %d skipped; "+
		"CI/titles/labels/reviews from snapshot\n",
		filepath.Base(s.Dir), len(page.Assessed), len(page.WIP), len(page.Skipped))
	return err
}

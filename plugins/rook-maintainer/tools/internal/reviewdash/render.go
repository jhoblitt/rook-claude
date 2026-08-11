package reviewdash

import (
	"cmp"
	"embed"
	"html/template"
	"io"
	"slices"
	"strconv"
	"strings"
)

const (
	emDash = "—"

	// The CI tooltip names at most this many failing checks before falling
	// back to a "+N more" count.
	maxFailedNames = 6

	maxProposalPaths = 3
)

//go:embed page.html.tmpl cells.html.tmpl
var templates embed.FS

var page = template.Must(template.New("page.html.tmpl").ParseFS(templates, "*.html.tmpl"))

type sevSpec struct {
	key, letter, class, label string
}

// Severity order is the phase-4 walk order, worst first, and drives both the
// chip row and the detail table. A severity outside it is counted but gets no
// chip, exactly as the Python generator behaved.
var severities = []sevSpec{
	{"blocker", "B", "sev-b", "blocker"},
	{"changes-requested", "C", "sev-c", "changes requested"},
	{"nit", "N", "sev-n", "nit"},
	{"question", "Q", "sev-q", "design question"},
}

var unknownSeverity = sevSpec{letter: "?", class: "sev-n"}

func severity(key string) (int, sevSpec) {
	for i, s := range severities {
		if s.key == key {
			return i, s
		}
	}
	return -1, unknownSeverity
}

type chip struct {
	Class string
	Title string
	Text  string
}

type bugChip struct {
	Class string
	Raw   string
	Lower string
}

type sevChip struct {
	Class  string
	Letter string
	Label  string
	Count  int
}

type ciCell struct {
	Empty   bool
	Kind    string
	Passing int
	Total   int
	Failing int
	Pending int
	Failed  string
	More    int
}

type findingRow struct {
	Class      string
	ID         string
	Domain     string
	Anchor     string
	Summary    string
	Confidence string
	Status     string
}

type prRow struct {
	Num        string
	Class      string
	Verdict    string
	Bug        *bugChip
	Flags      []chip
	Sev        []sevChip
	NoFindings bool
	Title      string
	CI         ciCell
	Backport   *chip
	Author     string
	Assoc      string
	Live       int
	Rationale  string
	Rows       []findingRow
	HasClean   bool
	CleanNote  string
}

type skipRow struct {
	Num    string
	Reason string
}

type pageData struct {
	Date  string
	Repo  string
	PRs   []prRow
	Skips []skipRow
}

// Render writes the dashboard. Every interpolation is escaped by
// html/template: PR titles, check names and finding text all originate in
// PRs the sweep did not write.
func (s *Sweep) Render(w io.Writer) error {
	data := pageData{Date: s.Date, Repo: s.Repo}
	for _, r := range s.recs {
		data.PRs = append(data.PRs, s.row(r))
	}
	for _, sk := range s.skips {
		data.Skips = append(data.Skips, skipRow{Num: sk.Number.str, Reason: sk.Reason.str})
	}
	return page.ExecuteTemplate(w, "page.html.tmpl", data)
}

func (s *Sweep) row(r record) prRow {
	it := s.items[r.num]
	live := r.liveFindings()
	row := prRow{
		Num:        r.num,
		Class:      rowClass(r),
		Verdict:    r.Verdict.orElse(emDash),
		Bug:        bugCell(r),
		Flags:      flagChips(r),
		Sev:        sevChips(live),
		NoFindings: len(live) == 0,
		Title:      it.Title.str,
		CI:         ciCellFor(it.CI),
		Backport:   backportCell(r),
		Author:     it.Author.str,
		Assoc:      strings.ToLower(it.AuthorAssociation.str),
		Live:       len(live),
		Rationale:  firstRunes(r.Rationale.str, 160),
		Rows:       findingRows(live),
	}
	if row.HasClean = len(r.Clean) > 0; row.HasClean {
		clean := make([]string, 0, len(r.Clean))
		for _, c := range r.Clean {
			clean = append(clean, c.str)
		}
		row.CleanNote = strings.Join(clean, "; ")
	}
	return row
}

// Row colour tracks the order the maintainer works the list in — the phase-4
// walk is REJECT, REQUEST_CHANGES, ACCEPT — so the verdict outranks severity.
// The two escalation flags outrank the verdict because both mean it is not
// final yet: needs_proposal_review holds it provisional until proposal mode
// runs, and a takeover candidate changes who carries the PR, not whether the
// diff is correct.
func rowClass(r record) string {
	switch {
	case r.NeedsProposalReview != nil && r.NeedsProposalReview.Flag.truthy:
		return "prop"
	case r.TakeoverCandidate != nil && r.TakeoverCandidate.Flag.truthy:
		return "take"
	}
	switch r.Verdict.text() {
	case "REJECT":
		return "reject"
	case "REQUEST_CHANGES":
		return "chg"
	case "ACCEPT":
		return "accept"
	}
	return "mon"
}

func bugCell(r record) *bugChip {
	bug := r.Bug.str
	if bug == "" || bug == "N/A" {
		return nil
	}
	class := "bug-fab"
	if bug == "REAL" {
		class = "bug-real"
	}
	return &bugChip{Class: class, Raw: bug, Lower: strings.ToLower(bug)}
}

func flagChips(r record) []chip {
	var out []chip
	if p := r.NeedsProposalReview; p != nil && p.Flag.truthy {
		where := joinTexts(p.Paths, maxProposalPaths)
		if where == "" {
			where = "design doc"
		}
		out = append(out, chip{
			Class: "prop",
			Title: "verdict provisional until proposal mode runs on " + where,
			Text:  "proposal",
		})
	}
	if t := r.TakeoverCandidate; t != nil && t.Flag.truthy {
		out = append(out, chip{Class: "take", Title: t.Reason.str, Text: "takeover"})
	}
	return out
}

func backportCell(r record) *chip {
	if r.Backport == nil || !r.Backport.Eligible.truthy {
		return nil
	}
	return &chip{
		Class: "bp",
		Title: r.Backport.Reason.str,
		Text:  r.Backport.Label.orElse("eligible"),
	}
}

func sevChips(live []finding) []sevChip {
	counts := map[string]int{}
	for _, f := range live {
		counts[f.Severity.textOr("nit")]++
	}
	var out []sevChip
	for _, s := range severities {
		if n := counts[s.key]; n > 0 {
			out = append(out, sevChip{Class: s.class, Letter: s.letter, Label: s.label, Count: n})
		}
	}
	return out
}

func ciCellFor(ci *ciRollup) ciCell {
	if ci == nil {
		return ciCell{Empty: true}
	}
	total, passing := ci.Total.number(), ci.Passing.number()
	failing, pending := ci.Failing.number(), ci.Pending.number()
	if total == 0 {
		return ciCell{Empty: true}
	}
	cell := ciCell{Kind: "green", Passing: passing, Total: total, Pending: pending}
	switch {
	case failing != 0:
		cell.Kind = "red"
	case pending != 0:
		cell.Kind = "amber"
	}
	if failing != 0 {
		cell.Failing = failing
		cell.Failed = joinTexts(ci.Failed, maxFailedNames)
		cell.More = max(len(ci.Failed)-maxFailedNames, 0)
	}
	return cell
}

func joinTexts(vals []value, limit int) string {
	out := make([]string, 0, min(len(vals), limit))
	for _, v := range vals[:min(len(vals), limit)] {
		out = append(out, v.str)
	}
	return strings.Join(out, ", ")
}

func findingRows(live []finding) []findingRow {
	sorted := slices.Clone(live)
	slices.SortStableFunc(sorted, func(a, b finding) int {
		if c := cmp.Compare(sortRank(a), sortRank(b)); c != 0 {
			return c
		}
		return cmp.Compare(a.ID.text(), b.ID.text())
	})

	var out []findingRow
	for _, f := range sorted {
		_, sev := severity(f.Severity.textOr("nit"))
		anchor := "PR-level"
		if f.Path.truthy {
			anchor = f.Path.str
			if f.Line.truthy {
				anchor += ":" + f.Line.str
			}
		}
		out = append(out, findingRow{
			Class:      sev.class,
			ID:         f.ID.textOr(sev.letter),
			Domain:     f.Domain.str,
			Anchor:     anchor,
			Summary:    f.Summary.str,
			Confidence: confidence(f.Confidence),
			Status:     f.Status.textOr("pending"),
		})
	}
	return out
}

// A finding whose severity is absent or outside the vocabulary sorts last but
// still renders as a nit, which is how the Python generator behaved.
func sortRank(f finding) int {
	if i, _ := severity(f.Severity.text()); i >= 0 {
		return i
	}
	return 9
}

func confidence(c value) string {
	txt := emDash
	switch {
	case c.isInt && c.i >= 80:
		txt = "CONFIRMED"
	case c.isInt && c.i >= 50:
		txt = "PLAUSIBLE"
	}
	if c.isInt && c.i != 0 {
		txt += " (" + strconv.FormatInt(c.i, 10) + ")"
	}
	return txt
}

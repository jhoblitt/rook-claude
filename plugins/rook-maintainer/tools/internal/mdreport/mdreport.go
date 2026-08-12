// Package mdreport renders the machine-derivable half of a rook-triage
// report.md: the per-item tables and the routing ledger that
// skills/rook-triage/references/reporting.md prescribes. Callers are
// internal/prdash and internal/issuesdash, behind gen-pr-dashboard --markdown
// and gen-issues-dashboard --markdown.
//
// The output is a FRAGMENT rather than a document; Section states that rule
// and is what enforces it.
//
// Every cell holds text a stranger wrote on the internet. Escape is the only
// way that text may reach a cell: an unescaped "|" silently shifts the rest of
// the row under the wrong headers, and a table that reads plausibly while
// naming the wrong PR is worse than no table. Row re-checks that invariant
// rather than trusting its callers.
package mdreport

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"
)

const (
	// PerPersonCap is the per-person per-sweep routing cap of
	// references/routing.md, "Selection". validate-actions deliberately does
	// not re-check it (SKILL.md phase 5), which makes this ledger the only
	// place a breach becomes visible.
	PerPersonCap = 3

	// ReportFile is the name both renderers write the fragment under, in the
	// sweep dir. reporting.md's assembly step hardcodes it in a cat command,
	// so a name that drifted in one renderer would break assembly for that
	// corpus alone.
	ReportFile = "report-tables.md"

	// NoSnapshot fills the summary of a row the snapshot has no entry for. The
	// title is unknown, not absent, and an em dash there would read as an item
	// without a title.
	NoSnapshot = "_(no snapshot.json entry)_"

	repo   = "rook/rook"
	emDash = "—"
)

// active are the characters that change how a GFM table cell renders: the cell
// delimiter, the inline-markup openers, and the raw-HTML/entity openers. Only
// OPENERS are here — with "<" escaped no tag is recognized, so escaping ">"
// too would only pepper every "->" in a PR title with backslashes. Block
// constructs are absent for the same reason: a cell is inline context, so a
// leading "#" or "-" is already inert.
const active = "\\`*_[]<&|~"

// Escape makes s safe to place in a table cell while still reading as itself.
// Control characters become spaces because a newline would end the row.
func Escape(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 8)
	for _, r := range s {
		switch {
		case unicode.IsControl(r):
			b.WriteByte(' ')
		case strings.ContainsRune(active, r):
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Bold marks a triage proposal — a reviewer, a mention, a label the sweep
// wants added — as distinct from what the item already carries.
func Bold(s string) string {
	if s == "" {
		return ""
	}
	return "**" + Escape(s) + "**"
}

// IssueLink and PullLink render the one thing reporting.md demands everywhere:
// a clickable reference. The /issues/ form redirects for pull requests, but
// each side keeps its own path so a link matches the row it came from.
func IssueLink(number string) string { return link("issues", number) }

// PullLink is IssueLink for the pull-request path.
func PullLink(number string) string { return link("pull", number) }

func link(kind, number string) string {
	return fmt.Sprintf("[#%s](https://github.com/%s/%s/%s)",
		Escape(number), repo, kind, Escape(number))
}

// Links renders a cross-reference cell.
func Links(numbers []string, one func(string) string) string {
	out := make([]string, 0, len(numbers))
	for _, n := range numbers {
		out = append(out, one(n))
	}
	return strings.Join(out, ", ")
}

// Table writes a GFM table, holding the first error until Err. Cells arrive
// already composed — links and bold need markup — so Row cannot escape them
// and checks them instead.
type Table struct {
	w    io.Writer
	cols int
	err  error
}

func NewTable(w io.Writer, header ...string) *Table {
	t := &Table{w: w, cols: len(header)}
	rule := make([]string, len(header))
	for i := range rule {
		rule[i] = "---"
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		t.err = err
	}
	t.write(header)
	t.write(rule)
	return t
}

func (t *Table) Row(cells ...string) {
	if t.err != nil {
		return
	}
	if len(cells) != t.cols {
		t.err = fmt.Errorf("row has %d cells, want %d: %q", len(cells), t.cols, cells)
		return
	}
	for i, c := range cells {
		if strings.ContainsAny(c, "\n\r") || hasBareDelimiter(c) {
			t.err = fmt.Errorf("cell %d escaped nothing and would break the table: %q", i, c)
			return
		}
	}
	t.write(cells)
}

func (t *Table) Err() error { return t.err }

func (t *Table) write(cells []string) {
	if t.err != nil {
		return
	}
	var b strings.Builder
	b.WriteByte('|')
	for _, c := range cells {
		c = strings.TrimSpace(c)
		if c == "" {
			c = emDash
		}
		b.WriteByte(' ')
		b.WriteString(c)
		b.WriteString(" |")
	}
	b.WriteByte('\n')
	_, t.err = io.WriteString(t.w, b.String())
}

// hasBareDelimiter reports whether s carries a "|" that is not backslash
// escaped, i.e. one that would split the cell.
func hasBareDelimiter(s string) bool {
	for i := range len(s) {
		if s[i] != '|' {
			continue
		}
		slashes := 0
		for j := i - 1; j >= 0 && s[j] == '\\'; j-- {
			slashes++
		}
		if slashes%2 == 0 {
			return true
		}
	}
	return false
}

// Entry is one person's count of proposed routing across the sweep.
type Entry struct {
	Login string
	Count int
}

// Proposal is one item's proposed routing: the item's number and the logins
// triage wants to route it to. Per-ITEM grouping is the cap's own unit — a
// login is charged once per item however many times the item names them — so
// every extraction that feeds a ledger hands its data over in this shape
// rather than as a flat login list. internal/runledger sums these across a
// run's two sweep dirs, which is why the shape is here and not in either
// corpus package.
type Proposal struct {
	Number string
	Logins []string
}

// Group reshapes proposals into Counts' argument.
func Group(ps []Proposal) [][]string {
	out := make([][]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, p.Logins)
	}
	return out
}

// Fold keys a login the way GitHub's own comparison does, case-insensitively.
// Counts charges by this key, so anything reported ALONGSIDE a count — a
// per-corpus breakdown, the items a person was proposed on — has to group by
// it too, or the two halves of a row will disagree about whether "subhamkrai"
// and "SubhamKrai" are one person.
func Fold(login string) string { return strings.ToLower(login) }

// Swap is one item whose routing set changed because a pick was at the cap.
// Note is the batch item's own cap_note, linkified by the caller.
type Swap struct {
	Link string
	Note string
}

// Counts tallies how many ITEMS each login was proposed on, heaviest first so
// the people at risk of the cap head the ledger. Its argument is grouped by
// item rather than flat because a batch item can name one person twice —
// "subhamkrai" beside "subhamkrai (backup)" is one login — and charging both
// to the cap reports a breach that did not happen. Logins fold the way
// GitHub's own do, case-insensitively, and render as first spelled.
func Counts(perItem [][]string) []Entry {
	var out []Entry
	at := map[string]int{}
	for _, item := range perItem {
		charged := map[string]bool{}
		for _, login := range item {
			key := Fold(login)
			if charged[key] {
				continue
			}
			charged[key] = true
			i, seen := at[key]
			if !seen {
				i = len(out)
				at[key] = i
				out = append(out, Entry{Login: login})
			}
			out[i].Count++
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Login < out[j].Login
	})
	return out
}

// Ledger is the report's routing ledger: proposed counts against
// PerPersonCap, plus the sets a cap forced a swap in.
type Ledger struct {
	Heading string // "Reviewer ledger" / "Mention ledger"
	Column  string // what one row of Entries names
	Empty   string // the line for a sweep that proposed nobody
	Entries []Entry
	Swaps   []Swap
}

func (l Ledger) Write(w io.Writer) error {
	if err := Section(w, 2, "%s (per-person per-sweep cap: %d)",
		l.Heading, PerPersonCap); err != nil {
		return err
	}
	if len(l.Entries) == 0 {
		if err := Para(w, "_%s_", l.Empty); err != nil {
			return err
		}
	} else {
		t := NewTable(w, l.Column, "proposed", "cap", "status")
		for _, e := range l.Entries {
			t.Row(Escape(e.Login), fmt.Sprint(e.Count), fmt.Sprint(PerPersonCap), Status(e.Count))
		}
		if err := t.Err(); err != nil {
			return err
		}
	}
	if len(l.Swaps) == 0 {
		return nil
	}
	if err := Section(w, 3, "Cap-swapped sets (%d)", len(l.Swaps)); err != nil {
		return err
	}
	t := NewTable(w, "#", "cap note")
	for _, s := range l.Swaps {
		t.Row(s.Link, s.Note)
	}
	return t.Err()
}

// Status is the ledger's verdict on one person's count. The per-corpus ledgers
// and the run-scoped one share this vocabulary so a breach reads identically
// wherever it surfaces.
func Status(count int) string {
	switch {
	case count > PerPersonCap:
		return fmt.Sprintf("OVER CAP by %d", count-PerPersonCap)
	case count == PerPersonCap:
		return "at cap"
	}
	return ""
}

// Section writes a fragment heading, and is where the fragment rule lives.
//
// A maintainer concatenates a model-written notes file with this output, so it
// carries no document title — an h1 would collide with the notes' own, which
// makes level 2 the top of the fragment. Every block here — heading, table,
// Para — opens with the blank line that separates it from whatever precedes
// it, and none writes a trailing one. That is what keeps a heading followed by
// a subheading from doubling the blank line, and what makes the first heading
// start a block even when the prose before it did not end in a newline.
func Section(w io.Writer, level int, format string, args ...any) error {
	_, err := fmt.Fprintf(w, "\n%s %s\n", strings.Repeat("#", level),
		fmt.Sprintf(format, args...))
	return err
}

// Para writes a standalone line, e.g. the note that a section is empty.
func Para(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, "\n%s\n", fmt.Sprintf(format, args...))
	return err
}

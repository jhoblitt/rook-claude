// Package checklist decides whether a PR body reproduces the repository's
// pull-request template checklist, item for item.
//
// It decides the structural half its Spec defines, and reports every line's
// check state so the reviewer can apply the other half without re-reading the
// body.
//
// The template defaults to a git object (`origin/master:...`) rather than the
// working tree because the contributor being audited can rewrite the tree copy
// in the same PR, and a gate comparing a body against the branch's own
// template would ratify the rewrite.
//
// A PR body is attacker-authored, so nothing reaches a report unquoted: Quote
// bounds the length and makes invisible codepoints visible, because a report
// that echoes a zero-width space as itself shows the reviewer two identical
// strings and calls them different.
//
// Audit decides one body. Sweep decides a whole rook-triage sweep's pool,
// publishing each audited PR's verdict alone and a row naming why each of the
// others was left out; the contract of that pass is cmd/validate-checklist's
// package doc.
//
// Callers: cmd/validate-checklist, behind tools/run.sh.
// Spec: skills/rook-code-review/references/docs-sync.md, "PR-template
// checklist audit".
package checklist

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const (
	// MaxBody bounds the attacker-authored input. GitHub caps a PR body at
	// 65536 characters, so anything past this is not a body.
	MaxBody = 1 << 20
	// MaxTextChars bounds one quoted line of body text.
	MaxTextChars = 240
	// MaxExtras bounds how many unexpected body lines are echoed. A body can
	// carry ten thousand checkbox lines; the finding is the same after the
	// first few and the reviewer's context is not free.
	MaxExtras = 25
)

// State is a checklist line's checkbox state.
type State string

const (
	StateChecked   State = "checked"
	StateUnchecked State = "unchecked"
	// StateNone is a bullet with no checkbox, e.g. the template's
	// "Overwriting Ceph's configurations" sub-bullet.
	StateNone State = "none"
)

// Status is the audit's verdict on one line.
type Status string

const (
	StatusOK      Status = "ok"
	StatusAltered Status = "altered"
	StatusMissing Status = "missing"
	StatusExtra   Status = "extra"
)

// Verdict is the audit's verdict on the body as a whole.
type Verdict string

const (
	VerdictConforming    Verdict = "conforming"
	VerdictNonConforming Verdict = "non-conforming"
	VerdictNoChecklist   Verdict = "no-checklist"
)

// Item is one parsed checklist line.
type Item struct {
	Depth int    `json:"depth"`
	State State  `json:"state"`
	Text  string `json:"text"`
	Line  int    `json:"line"`
}

// key is what two lines must share to be the same checklist item: the same
// text, at the same depth, with or without a box. Only the box's STATE is
// excluded — that is the one difference the template invites.
func (it Item) key() string {
	box := "-"
	if it.State != StateNone {
		box = "[]"
	}
	return strconv.Itoa(it.Depth) + "\x00" + box + "\x00" + it.Text
}

// Line is one line of the audit report.
type Line struct {
	Status    Status `json:"status"`
	State     State  `json:"state,omitempty"`
	Text      string `json:"text"`
	WantState State  `json:"want_state,omitempty"`
	Want      string `json:"want,omitempty"`
	BodyLine  int    `json:"body_line,omitempty"`
}

// Report is the whole audit.
type Report struct {
	Verdict  Verdict  `json:"verdict"`
	Lines    []Line   `json:"lines"`
	Problems []string `json:"problems,omitempty"`
}

// Bad reports whether this audit should fail the gate.
func (r Report) Bad() bool { return r.Verdict != VerdictConforming }

// Count reports how many lines carry a status.
func (r Report) Count(s Status) int {
	n := 0
	for _, l := range r.Lines {
		if l.Status == s {
			n++
		}
	}
	return n
}

// itemRe matches a bullet line, with the checkbox optional so the template's
// plain sub-bullet parses as an item too. Trailing space after "]" is consumed
// rather than compared: it is invisible where the body renders.
var itemRe = regexp.MustCompile(`^([ \t]*)[-*+][ \t]+(?:\[([ xX])\][ \t]*)?(.*)$`)

// Template extracts the checklist items a body must reproduce.
func Template(text string) ([]Item, error) {
	blocks, _ := parse(text)
	best := -1
	for i, b := range blocks {
		if best < 0 || boxes(b) > boxes(blocks[best]) {
			best = i
		}
	}
	if best < 0 {
		return nil, errors.New("no checklist in the template: it has no `- [ ]` item")
	}
	return blocks[best], nil
}

// Audit compares a PR body against the template's items.
//
// The body is searched for the block that reproduces the most template items,
// rather than the first list of checkboxes: a description that opens with its
// own "Follow-ups" task list is ordinary, and auditing that instead would
// report every template item as missing.
func Audit(tmpl []Item, body string) Report {
	blocks, fenced := parse(body)
	keys := keySet(tmpl)

	best, top := -1, 0
	scores := make([]int, len(blocks))
	for i, b := range blocks {
		scores[i] = reproduced(keys, b)
		if scores[i] > top {
			best, top = i, scores[i]
		}
	}

	var rep Report
	if n := reproduced(keys, fenced); n > 0 {
		rep.Problems = append(rep.Problems, fmt.Sprintf(
			"%d checklist item(s) sit inside a code fence (line %d): fenced, the block renders as text and no box is a box",
			n, fenced[0].Line))
	}
	if best < 0 {
		rep.Verdict = VerdictNoChecklist
		rep.Problems = append(rep.Problems, absent(blocks))
		return rep
	}

	dup := min(2, len(tmpl))
	var copies []int
	for i, s := range scores {
		if i == best || s >= dup {
			copies = append(copies, i)
		}
	}
	if len(copies) > 1 {
		at := make([]string, 0, len(copies))
		for _, i := range copies {
			at = append(at, strconv.Itoa(blocks[i][0].Line))
		}
		if len(copies) == 2 {
			rep.Problems = append(rep.Problems, fmt.Sprintf(
				"the checklist appears twice, at lines %s", strings.Join(at, " and ")))
		} else {
			rep.Problems = append(rep.Problems, fmt.Sprintf(
				"the checklist appears %d times, at lines %s", len(copies), strings.Join(at, ", ")))
		}
	}

	lines, suppressed := align(tmpl, blocks[best])
	rep.Lines = lines
	if suppressed > 0 {
		rep.Problems = append(rep.Problems, fmt.Sprintf(
			"%d further unexpected line(s) in the block, not shown", suppressed))
	}

	rep.Verdict = VerdictConforming
	if len(rep.Problems) > 0 || rep.Count(StatusOK) != len(rep.Lines) {
		rep.Verdict = VerdictNonConforming
	}
	return rep
}

// absent says what was in the body instead of a checklist, because "no
// checklist" and "a checklist nobody would recognise" call for different
// replies to the contributor.
func absent(blocks [][]Item) string {
	n := 0
	for _, b := range blocks {
		n += boxes(b)
	}
	if n == 0 {
		return "no checklist in the body: it has no `- [ ]` item"
	}
	return fmt.Sprintf("no checklist in the body: %d checkbox line(s) are present, "+
		"none of them a template item", n)
}

func keySet(items []Item) map[string]bool {
	keys := make(map[string]bool, len(items))
	for _, it := range items {
		keys[it.key()] = true
	}
	return keys
}

// reproduced counts how many DISTINCT template items a run of lines carries,
// so a block that repeats one item fifty times does not outscore the real one.
func reproduced(keys map[string]bool, items []Item) int {
	seen := map[string]bool{}
	for _, it := range items {
		if k := it.key(); keys[k] && !seen[k] {
			seen[k] = true
		}
	}
	return len(seen)
}

func boxes(items []Item) int {
	n := 0
	for _, it := range items {
		if it.State != StateNone {
			n++
		}
	}
	return n
}

// parse splits a document into its checklist blocks and the checkbox lines
// that sit inside a code fence. A block is a maximal run of bullet lines,
// blank lines included: what ends one is prose.
func parse(text string) (blocks [][]Item, fenced []Item) {
	var cur []Item
	inComment, inFence, blank := false, false, true

	flush := func() {
		if boxes(cur) > 0 {
			blocks = append(blocks, cur)
		}
		cur = nil
	}

	for n, raw := range splitLines(text) {
		line, open := uncomment(raw, inComment)
		inComment = open
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(trimmed, "```"), strings.HasPrefix(trimmed, "~~~"):
			inFence = !inFence
			flush()
			blank = true
			continue
		case inFence:
			if it, ok := parseLine(line, n+1); ok && it.State != StateNone {
				fenced = append(fenced, it)
			}
			continue
		case trimmed == "":
			blank = true
			continue
		}

		if it, ok := parseLine(line, n+1); ok {
			cur = append(cur, it)
			blank = false
			continue
		}
		// An indented non-bullet line continues the item above it: an editor
		// that wraps a long template item must not read as a truncated block.
		if len(cur) > 0 && !blank && (line[0] == ' ' || line[0] == '\t') {
			cur[len(cur)-1].Text += " " + trimmed
			continue
		}
		flush()
		blank = false
	}
	flush()
	return blocks, fenced
}

func parseLine(raw string, n int) (Item, bool) {
	m := itemRe.FindStringSubmatch(raw)
	if m == nil {
		return Item{}, false
	}
	it := Item{Line: n, Text: strings.TrimRight(m[3], " \t")}
	if m[1] != "" {
		it.Depth = 1
	}
	switch m[2] {
	case "":
		it.State = StateNone
	case " ":
		it.State = StateUnchecked
	default:
		it.State = StateChecked
	}
	return it, true
}

// uncomment blanks out HTML-comment spans, carrying the open state across
// lines: the rook template's own instructions are comments, and a checklist
// commented out is a checklist the rendered body does not have.
func uncomment(line string, inComment bool) (string, bool) {
	var b strings.Builder
	for line != "" {
		if inComment {
			i := strings.Index(line, "-->")
			if i < 0 {
				return b.String(), true
			}
			line, inComment = line[i+3:], false
			continue
		}
		i := strings.Index(line, "<!--")
		if i < 0 {
			b.WriteString(line)
			return b.String(), false
		}
		b.WriteString(line[:i])
		line, inComment = line[i+4:], true
	}
	return b.String(), inComment
}

// splitLines splits on LF and drops one CR before it, so a body pasted from a
// Windows editor parses like any other.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSuffix(line, "\r")
	}
	return lines
}

// align walks the template and the body block together and reports what
// differs, bounding the unexpected lines it echoes.
//
// The pairing is a longest-common-subsequence diff rather than an index-by-
// index comparison: one dropped item would otherwise shift every item after it
// and report the whole checklist as rewritten.
func align(tmpl, body []Item) ([]Line, int) {
	n, m := len(tmpl), len(body)
	tk, bk := make([]string, n), make([]string, m)
	for i, it := range tmpl {
		tk[i] = it.key()
	}
	for j, it := range body {
		bk[j] = it.key()
	}

	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			switch {
			case tk[i] == bk[j]:
				lcs[i][j] = lcs[i+1][j+1] + 1
			case lcs[i+1][j] >= lcs[i][j+1]:
				lcs[i][j] = lcs[i+1][j]
			default:
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	var out []Line
	for i, j := 0, 0; i < n || j < m; {
		switch {
		case i < n && j < m && tk[i] == bk[j]:
			out = append(out, Line{Status: StatusOK, State: body[j].State,
				Text: Quote(body[j].Text), BodyLine: body[j].Line})
			i, j = i+1, j+1
		case i < n && (j == m || lcs[i+1][j] >= lcs[i][j+1]):
			out = append(out, Line{Status: StatusMissing,
				Text: Quote(tmpl[i].Text), WantState: tmpl[i].State})
			i++
		default:
			out = append(out, Line{Status: StatusExtra, State: body[j].State,
				Text: Quote(body[j].Text), BodyLine: body[j].Line})
			j++
		}
	}
	return bound(coalesce(out))
}

// coalesce turns a missing item that a body line stands in for into one
// ALTERED line. A reworded item is one finding with a "want", not a missing
// item and an unexplained extra one somewhere else in the list.
func coalesce(in []Line) []Line {
	var out []Line
	for i := 0; i < len(in); {
		if in[i].Status == StatusOK {
			out = append(out, in[i])
			i++
			continue
		}
		j := i
		for j < len(in) && in[j].Status != StatusOK {
			j++
		}
		out = append(out, pair(in[i:j])...)
		i = j
	}
	return out
}

// pair decides which template item each unexpected line stands in for. The
// match is by shared words rather than by position: a body that drops the
// sub-bullet AND edits the item below it would otherwise pair the edit with
// the sub-bullet, and a report that names the wrong item is worse than one
// that names none. An altered line takes the template item's place in the
// list, so the report stays in template order.
func pair(run []Line) []Line {
	var missing, extra []int
	for i, l := range run {
		if l.Status == StatusMissing {
			missing = append(missing, i)
		} else {
			extra = append(extra, i)
		}
	}

	stands := map[int]int{}
	taken := map[int]bool{}
	for _, e := range extra {
		best, score := -1, -1.0
		for _, m := range missing {
			if taken[m] {
				continue
			}
			if s := similar(run[m].Text, run[e].Text); s > score {
				best, score = m, s
			}
		}
		if best < 0 {
			continue
		}
		taken[best] = true
		stands[best] = e
	}
	paired := make(map[int]bool, len(stands))
	for _, e := range stands {
		paired[e] = true
	}

	var out []Line
	for i, l := range run {
		e, replaced := stands[i]
		switch {
		case replaced:
			out = append(out, Line{
				Status:    StatusAltered,
				State:     run[e].State,
				Text:      run[e].Text,
				BodyLine:  run[e].BodyLine,
				WantState: l.WantState,
				Want:      l.Text,
			})
		case paired[i]:
		default:
			out = append(out, l)
		}
	}
	return out
}

// similar scores two lines by the words they share, which survives the edits
// that matter: an item with its link stripped keeps every word of its label,
// while an item that replaced another shares nothing but articles.
func similar(a, b string) float64 {
	wa, wb := words(a), words(b)
	if len(wa) == 0 || len(wb) == 0 {
		return 0
	}
	shared := 0
	for w := range wa {
		if wb[w] {
			shared++
		}
	}
	return float64(shared) / float64(max(len(wa), len(wb)))
}

func words(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		out[w] = true
	}
	return out
}

func bound(lines []Line) ([]Line, int) {
	kept, suppressed := make([]Line, 0, len(lines)), 0
	extras := 0
	for _, l := range lines {
		if l.Status == StatusExtra {
			if extras >= MaxExtras {
				suppressed++
				continue
			}
			extras++
		}
		kept = append(kept, l)
	}
	return kept, suppressed
}

// Quote bounds attacker-chosen text on its way into a report and spells out
// what would otherwise be invisible in it. Anything not printable becomes its
// codepoint: a zero-width space inside an item is how a checklist reads as the
// template's and is not, and a report that reproduces it unchanged would show
// the difference as no difference at all.
func Quote(s string) string {
	truncated := false
	if r := []rune(s); len(r) > MaxTextChars {
		s, truncated = string(r[:MaxTextChars]), true
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	for _, r := range s {
		if unicode.IsPrint(r) {
			b.WriteRune(r)
			continue
		}
		fmt.Fprintf(&b, `\u%04x`, r)
	}
	if truncated {
		b.WriteString("...")
	}
	return b.String()
}

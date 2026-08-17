package checklist

import (
	"strings"
	"testing"
)

// The fixture's own lines, used to build bodies by editing the template the
// way a contributor would.
const (
	notesLine  = "- [ ] [Pending release notes](https://github.com/rook/rook/blob/master/PendingReleaseNotes.md) updated with breaking and/or notable changes for the next minor release."
	nestedLine = "  - Overwriting Ceph's configurations should be marked as breaking changes."
	docsLine   = "- [ ] Documentation has been updated, if necessary (under the `Documentation` folder)."
	unitLine   = "- [ ] Unit tests have been added, if necessary (`_test.go` files under the `cmd` and `pkg` folders)."
	integLine  = "- [ ] Integration tests have been added, if necessary (in the `tests/integration` folder)."
)

// Item indexes into the template, for the expectations below.
const (
	iCommit = 0
	iGuide  = 1
	iAI     = 2
	iNotes  = 3
	iNested = 4
	iDocs   = 5
	iUnit   = 6
	iInteg  = 7
)

func mustTemplate(t *testing.T) []Item {
	t.Helper()
	items, err := Template(fixtureTemplate)
	if err != nil {
		t.Fatalf("Template(fixture): %v", err)
	}
	return items
}

// block is the fixture from the "**Checklist:**" heading on: what a body
// carries when the contributor pastes the template.
func block(t *testing.T) string {
	t.Helper()
	i := strings.Index(fixtureTemplate, "**Checklist:**")
	if i < 0 {
		t.Fatal("fixture has no checklist heading")
	}
	return fixtureTemplate[i:]
}

func replace(text, old, with string) string { return strings.Replace(text, old, with, 1) }

func dropLine(text, line string) string { return strings.Replace(text, line+"\n", "", 1) }

func tick(line string) string { return "- [x] " + strings.TrimPrefix(line, "- [ ] ") }

// want is one expected report line: its status, its check state, and a
// substring its text must carry.
type want struct {
	status   Status
	state    State
	contains string
}

// okLines is the expectation for a body that reproduces every template item,
// with the checkbox items in state.
func okLines(items []Item, state State) []want {
	out := make([]want, 0, len(items))
	for _, it := range items {
		s := state
		if it.State == StateNone {
			s = StateNone
		}
		out = append(out, want{StatusOK, s, it.Text})
	}
	return out
}

func edit(base []want, i int, w want) []want {
	out := append([]want(nil), base...)
	out[i] = w
	return out
}

func insert(base []want, i int, w want) []want {
	out := append([]want(nil), base[:i]...)
	out = append(out, w)
	return append(out, base[i:]...)
}

func TestTemplate(t *testing.T) {
	items := mustTemplate(t)
	if len(items) != 8 {
		t.Fatalf("template items = %d, want 8: %+v", len(items), items)
	}
	if got := items[iCommit].Text; !strings.HasPrefix(got, "**Commit Message Formatting**:") {
		t.Errorf("items[0].Text = %q", got)
	}
	for i, it := range items {
		wantState, wantDepth := StateUnchecked, 0
		if i == iNested {
			wantState, wantDepth = StateNone, 1
		}
		if it.State != wantState || it.Depth != wantDepth {
			t.Errorf("items[%d] state/depth = %s/%d, want %s/%d", i, it.State, it.Depth, wantState, wantDepth)
		}
	}
	if _, err := Template("**Checklist:**\n\nnothing to tick here.\n"); err == nil {
		t.Error("Template(no checkboxes) = nil error, want an error")
	}
}

func TestAudit(t *testing.T) {
	tmpl := mustTemplate(t)
	unchecked := okLines(tmpl, StateUnchecked)
	checked := okLines(tmpl, StateChecked)
	ticked := strings.ReplaceAll(fixtureTemplate, "- [ ]", "- [x]")

	tests := []struct {
		name     string
		body     string
		verdict  Verdict
		lines    []want
		problems []string
	}{
		{
			name:    "verbatim template",
			body:    fixtureTemplate,
			verdict: VerdictConforming,
			lines:   unchecked,
		},
		{
			name:    "every box ticked",
			body:    ticked,
			verdict: VerdictConforming,
			lines:   checked,
		},
		{
			name:    "real body around a pasted checklist",
			body:    "Fixes a nil deref in the mon health checker.\n\nResolves #1234\n\n" + replace(block(t), docsLine, tick(docsLine)),
			verdict: VerdictConforming,
			lines:   edit(unchecked, iDocs, want{StatusOK, StateChecked, tmpl[iDocs].Text}),
		},
		{
			name:    "uppercase tick",
			body:    replace(block(t), docsLine, "- [X] "+strings.TrimPrefix(docsLine, "- [ ] ")),
			verdict: VerdictConforming,
			lines:   edit(unchecked, iDocs, want{StatusOK, StateChecked, tmpl[iDocs].Text}),
		},
		{
			name:    "CRLF body",
			body:    strings.ReplaceAll(ticked, "\n", "\r\n"),
			verdict: VerdictConforming,
			lines:   checked,
		},
		{
			name:    "wrapped item",
			body:    replace(block(t), docsLine, "- [ ] Documentation has been updated, if necessary\n  (under the `Documentation` folder)."),
			verdict: VerdictConforming,
			lines:   unchecked,
		},
		{
			name:    "unrelated task list elsewhere",
			body:    "Follow-ups:\n\n- [ ] rework the retry loop in a later PR\n- [x] rebased on master\n\n" + block(t),
			verdict: VerdictConforming,
			lines:   unchecked,
		},
		{
			name:     "no checklist at all",
			body:     "Fixes a nil deref in the mon health checker.\n\nResolves #1234\n",
			verdict:  VerdictNoChecklist,
			problems: []string{"no checklist"},
		},
		{
			name:     "checklist commented out",
			body:     "Fixes a thing.\n\n<!--\n" + block(t) + "\n-->\n",
			verdict:  VerdictNoChecklist,
			problems: []string{"no checklist"},
		},
		{
			name:     "checklist inside a code fence",
			body:     "Fixes a thing.\n\n```\n" + block(t) + "```\n",
			verdict:  VerdictNoChecklist,
			problems: []string{"code fence"},
		},
		{
			name:    "reworded item",
			body:    replace(block(t), docsLine, "- [x] Docs updated where needed."),
			verdict: VerdictNonConforming,
			lines:   edit(unchecked, iDocs, want{StatusAltered, StateChecked, "Docs updated where needed."}),
		},
		{
			name:    "rationale appended to an item",
			body:    replace(block(t), docsLine, docsLine+" - N/A: no user-facing docs"),
			verdict: VerdictNonConforming,
			lines:   edit(unchecked, iDocs, want{StatusAltered, StateUnchecked, "N/A: no user-facing docs"}),
		},
		{
			name:    "link stripped",
			body:    replace(block(t), notesLine, "- [ ] Pending release notes updated with breaking and/or notable changes for the next minor release."),
			verdict: VerdictNonConforming,
			lines:   edit(unchecked, iNotes, want{StatusAltered, StateUnchecked, "Pending release notes updated with breaking"}),
		},
		{
			name:    "hidden rune in an item",
			body:    replace(block(t), "Documentation has been", "Documentation has\u200b been"),
			verdict: VerdictNonConforming,
			lines:   edit(unchecked, iDocs, want{StatusAltered, StateUnchecked, `\u200b`}),
		},
		{
			name:    "item added",
			body:    replace(block(t), integLine, integLine+"\n- [x] Ran `make lint` locally."),
			verdict: VerdictNonConforming,
			lines:   insert(unchecked, iInteg+1, want{StatusExtra, StateChecked, "Ran `make lint` locally."}),
		},
		{
			name:    "item removed",
			body:    dropLine(block(t), integLine),
			verdict: VerdictNonConforming,
			lines:   edit(unchecked, iInteg, want{StatusMissing, "", tmpl[iInteg].Text}),
		},
		{
			name:    "sub-bullet removed",
			body:    dropLine(block(t), nestedLine),
			verdict: VerdictNonConforming,
			lines:   edit(unchecked, iNested, want{StatusMissing, "", tmpl[iNested].Text}),
		},
		{
			// Two absences and one replacement in the same run: the
			// replacement must be read against the item it resembles, not
			// against whichever absence came first.
			name: "sub-bullet removed and the item below it reworded",
			body: replace(dropLine(block(t), nestedLine), docsLine,
				"- [x] Documentation has been updated - N/A: no user-facing docs"),
			verdict: VerdictNonConforming,
			lines: edit(
				edit(unchecked, iNested, want{StatusMissing, "", tmpl[iNested].Text}),
				iDocs, want{StatusAltered, StateChecked, "N/A: no user-facing docs"}),
		},
		{
			name:    "items reordered",
			body:    replace(dropLine(block(t), docsLine), unitLine, unitLine+"\n"+docsLine),
			verdict: VerdictNonConforming,
		},
		{
			name:     "checklist twice",
			body:     block(t) + "\nAnd once more:\n\n" + block(t),
			verdict:  VerdictNonConforming,
			problems: []string{"twice"},
			lines:    unchecked,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Audit(tmpl, tc.body)
			if got.Verdict != tc.verdict {
				t.Errorf("verdict = %s, want %s", got.Verdict, tc.verdict)
			}
			if tc.lines != nil {
				assertLines(t, got.Lines, tc.lines)
			}
			for _, p := range tc.problems {
				if !containsAny(got.Problems, p) {
					t.Errorf("problems = %q, want one containing %q", got.Problems, p)
				}
			}
		})
	}
}

// Reordering is the case whose expectation is least obvious: the unit item
// still matches in place, so the moved documentation item reports as a
// template item that is gone plus a body line that is unexpected — never as
// two silent OKs.
func TestAuditReorderedShape(t *testing.T) {
	tmpl := mustTemplate(t)
	body := replace(dropLine(block(t), docsLine), unitLine, unitLine+"\n"+docsLine)
	assertLines(t, Audit(tmpl, body).Lines, []want{
		{StatusOK, StateUnchecked, tmpl[iCommit].Text},
		{StatusOK, StateUnchecked, tmpl[iGuide].Text},
		{StatusOK, StateUnchecked, tmpl[iAI].Text},
		{StatusOK, StateUnchecked, tmpl[iNotes].Text},
		{StatusOK, StateNone, tmpl[iNested].Text},
		{StatusMissing, "", tmpl[iDocs].Text},
		{StatusOK, StateUnchecked, tmpl[iUnit].Text},
		{StatusExtra, StateUnchecked, tmpl[iDocs].Text},
		{StatusOK, StateUnchecked, tmpl[iInteg].Text},
	})
}

func TestAuditBoundsExtras(t *testing.T) {
	tmpl := mustTemplate(t)
	var b strings.Builder
	b.WriteString(block(t))
	for i := range MaxExtras + 10 {
		b.WriteString("- [x] flooded item ")
		b.WriteString(strings.Repeat("9", i%3+1))
		b.WriteString("\n")
	}
	got := Audit(tmpl, b.String())
	extras := 0
	for _, l := range got.Lines {
		if l.Status == StatusExtra {
			extras++
		}
	}
	if extras > MaxExtras {
		t.Errorf("reported %d unexpected line(s), want at most %d", extras, MaxExtras)
	}
	if !containsAny(got.Problems, "further") {
		t.Errorf("problems = %q, want one reporting the suppressed lines", got.Problems)
	}
}

func TestQuote(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain text", "Documentation has been updated", "Documentation has been updated"},
		{"backticks kept", "under the `Documentation` folder", "under the `Documentation` folder"},
		{"em dash kept", "a — b", "a — b"},
		{"zero width", "a\u200bb", `a\u200bb`},
		{"bidi override", "a\u202eb", `a\u202eb`},
		{"non-breaking space", "a\u00a0b", `a\u00a0b`},
		{"newline", "a\nb", `a\u000ab`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Quote(tc.in); got != tc.want {
				t.Errorf("Quote(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}

	got := Quote(strings.Repeat("x", MaxTextChars*2))
	if len([]rune(got)) > MaxTextChars+3 {
		t.Errorf("Quote(long) is %d runes, want at most %d", len([]rune(got)), MaxTextChars+3)
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("Quote(long) = %q, want a truncation marker", got)
	}
}

func TestSelfTest(t *testing.T) {
	if fails := SelfTest(); len(fails) > 0 {
		t.Errorf("SelfTest() = %q, want none", fails)
	}
}

func assertLines(t *testing.T, got []Line, wants []want) {
	t.Helper()
	if len(got) != len(wants) {
		t.Errorf("report has %d line(s), want %d", len(got), len(wants))
		for i, l := range got {
			t.Logf("  got[%d] = %-8s %-9s %s", i, l.Status, l.State, l.Text)
		}
		return
	}
	for i, w := range wants {
		if got[i].Status != w.status || got[i].State != w.state {
			t.Errorf("line %d = %s/%s, want %s/%s", i, got[i].Status, got[i].State, w.status, w.state)
		}
		if !strings.Contains(got[i].Text, w.contains) {
			t.Errorf("line %d text = %q, want it to contain %q", i, got[i].Text, w.contains)
		}
	}
}

func containsAny(haystack []string, needle string) bool {
	for _, h := range haystack {
		if strings.Contains(h, needle) {
			return true
		}
	}
	return false
}

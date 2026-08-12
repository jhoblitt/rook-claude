package mdreport

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// Every case is a character that changes how the cell renders. The escaped
// form has to survive a GFM renderer as the literal text a maintainer typed
// into a PR title.
func TestEscape(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain", "fix: nil deref in the object controller",
			"fix: nil deref in the object controller"},
		{"cell delimiter", "a|b", `a\|b`},
		{"backslash first", `a\|b`, `a\\\|b`},
		{"code span", "run `kubectl get pods`", "run \\`kubectl get pods\\`"},
		{"emphasis", "*bold* and _under_", `\*bold\* and \_under\_`},
		{"link syntax", "[see #13001](x)", `\[see #13001\](x)`},
		{"raw html", "<b>x</b>&amp;", `\<b>x\</b>\&amp;`},
		{"strikethrough", "~~gone~~", `\~\~gone\~\~`},
		{"newline would end the row", "a\nb\r\nc", "a b  c"},
		{"tab", "a\tb", "a b"},
		{"escape sequence", "a\x1b[31mb", `a \[31mb`},
		{"non-ASCII is left alone", "é 漢字 — …", "é 漢字 — …"},
		{"inline context leaves block markers inert", "# head - item > quote",
			"# head - item > quote"},
	}
	for _, tc := range tests {
		if got := Escape(tc.in); got != tc.want {
			t.Errorf("%s: Escape(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestBold(t *testing.T) {
	if got := Bold("a|b"); got != `**a\|b**` {
		t.Errorf("Bold() = %q", got)
	}
	if got := Bold(""); got != "" {
		t.Errorf("Bold(\"\") = %q, want empty", got)
	}
}

func TestLinks(t *testing.T) {
	if got := IssueLink("15003"); got != "[#15003](https://github.com/rook/rook/issues/15003)" {
		t.Errorf("IssueLink() = %q", got)
	}
	if got := PullLink("15003"); got != "[#15003](https://github.com/rook/rook/pull/15003)" {
		t.Errorf("PullLink() = %q", got)
	}
	got := Links([]string{"1", "2"}, PullLink)
	want := "[#1](https://github.com/rook/rook/pull/1), [#2](https://github.com/rook/rook/pull/2)"
	if got != want {
		t.Errorf("Links() = %q, want %q", got, want)
	}
	if got := Links(nil, PullLink); got != "" {
		t.Errorf("Links(nil) = %q, want empty", got)
	}
}

func TestTable(t *testing.T) {
	var buf bytes.Buffer
	tab := NewTable(&buf, "#", "summary")
	tab.Row("[#1](x)", `a\|b`)
	tab.Row("[#2](x)", "")
	if err := tab.Err(); err != nil {
		t.Fatalf("Err() = %v", err)
	}
	want := "\n| # | summary |\n| --- | --- |\n| [#1](x) | a\\|b |\n| [#2](x) | — |\n"
	if buf.String() != want {
		t.Errorf("table =\n%q\nwant\n%q", buf.String(), want)
	}
}

// A cell that reached Row unescaped is a bug in the caller, and one that
// silently shifts every later column under the wrong header is exactly the
// failure this whole fragment exists to prevent.
func TestTableRejectsCellsThatWouldBreakIt(t *testing.T) {
	tests := []struct {
		name  string
		cells []string
	}{
		{"raw delimiter", []string{"#", "a|b"}},
		{"delimiter after an escaped backslash", []string{"#", `a\\|b`}},
		{"newline", []string{"#", "a\nb"}},
		{"carriage return", []string{"#", "a\rb"}},
		{"too few cells", []string{"#"}},
		{"too many cells", []string{"#", "a", "b"}},
	}
	for _, tc := range tests {
		var buf bytes.Buffer
		tab := NewTable(&buf, "#", "summary")
		tab.Row(tc.cells...)
		if tab.Err() == nil {
			t.Errorf("%s: Row(%q) was accepted: %q", tc.name, tc.cells, buf.String())
		}
	}
}

func TestTableKeepsTheFirstError(t *testing.T) {
	var buf bytes.Buffer
	tab := NewTable(&buf, "#")
	tab.Row("a|b")
	tab.Row("fine")
	if !strings.Contains(tab.Err().Error(), "a|b") {
		t.Errorf("Err() = %v, want the first failure", tab.Err())
	}
	if strings.Contains(buf.String(), "fine") {
		t.Error("rows after the failure still reached the output")
	}
}

func TestCounts(t *testing.T) {
	got := Counts([][]string{
		{"sp98", "travisn"}, {"sp98"}, {"BlaineEXE", "sp98"}, {"travisn"}, nil,
	})
	want := []Entry{{"sp98", 3}, {"travisn", 2}, {"BlaineEXE", 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Counts() = %+v, want %+v", got, want)
	}
	if got := Counts(nil); len(got) != 0 {
		t.Errorf("Counts(nil) = %+v, want empty", got)
	}
}

// A cap breach the sweep did not commit is as bad as one it missed, and a
// batch item naming the same person twice is the way to manufacture one: the
// login alone and the login with a note are one proposal, not two.
func TestCountsChargesOneItemOnce(t *testing.T) {
	tests := []struct {
		name    string
		perItem [][]string
		want    []Entry
	}{
		{"repeat within an item", [][]string{{"subhamkrai", "subhamkrai"}},
			[]Entry{{"subhamkrai", 1}}},
		{"same login, differing spelling", [][]string{{"subhamkrai", "SubhamKrai"}},
			[]Entry{{"subhamkrai", 1}}},
		{"folded across items", [][]string{{"Madhu-1"}, {"madhu-1"}, {"Madhu-1"}},
			[]Entry{{"Madhu-1", 3}}},
		{"a repeat does not mask a real breach", [][]string{
			{"sp98", "sp98"}, {"sp98"}, {"sp98"}, {"sp98"}},
			[]Entry{{"sp98", 4}}},
	}
	for _, tc := range tests {
		if got := Counts(tc.perItem); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: Counts(%v) = %+v, want %+v", tc.name, tc.perItem, got, tc.want)
		}
	}
}

// Group and Fold are the seams internal/runledger reuses to sum a run's two
// sweep dirs: it counts through Counts and groups the items behind each count
// by Fold, so both have to mean what Counts means by them.
func TestGroupAndFold(t *testing.T) {
	got := Counts(Group([]Proposal{
		{Number: "18001", Logins: []string{"sp98", "SP98"}},
		{Number: "18002", Logins: []string{"sp98"}},
		{Number: "18003"},
	}))
	if want := []Entry{{"sp98", 2}}; !reflect.DeepEqual(got, want) {
		t.Errorf("Counts(Group()) = %+v, want %+v", got, want)
	}
	if got := Fold("SubhamKrai"); got != "subhamkrai" {
		t.Errorf("Fold() = %q", got)
	}
}

func TestStatus(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{{0, ""}, {2, ""}, {3, "at cap"}, {4, "OVER CAP by 1"}, {6, "OVER CAP by 3"}}
	for _, tc := range tests {
		if got := Status(tc.count); got != tc.want {
			t.Errorf("Status(%d) = %q, want %q", tc.count, got, tc.want)
		}
	}
}

// The cap is the whole point of the ledger: a person over it has to be
// unmissable, because nothing downstream re-checks it.
func TestLedgerFlagsTheCap(t *testing.T) {
	var buf bytes.Buffer
	l := Ledger{
		Heading: "Reviewer ledger",
		Column:  "reviewer",
		Empty:   "nobody",
		Entries: Counts([][]string{
			{"a", "b"}, {"a", "b"}, {"a", "b"}, {"a"}, {"a"}, {"c"},
		}),
		Swaps: []Swap{{Link: "[#1](x)", Note: "b at cap"}},
	}
	if err := l.Write(&buf); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"## Reviewer ledger (per-person per-sweep cap: 3)",
		"| reviewer | proposed | cap | status |",
		"| a | 5 | 3 | OVER CAP by 2 |",
		"| b | 3 | 3 | at cap |",
		"| c | 1 | 3 | — |",
		"### Cap-swapped sets (1)",
		"| [#1](x) | b at cap |",
	} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("ledger is missing %q:\n%s", want, buf.String())
		}
	}
}

func TestLedgerWithoutProposals(t *testing.T) {
	var buf bytes.Buffer
	l := Ledger{Heading: "Mention ledger", Column: "mentioned", Empty: "nobody"}
	if err := l.Write(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "_nobody_") {
		t.Errorf("ledger = %q, want the empty note", buf.String())
	}
	if strings.Contains(buf.String(), "Cap-swapped") {
		t.Errorf("ledger = %q, want no cap-swap section", buf.String())
	}
}

// Blocks compose without doubling blank lines, and the fragment opens with
// one so it can be appended to prose that did not end in a newline.
func TestBlockSpacing(t *testing.T) {
	var buf bytes.Buffer
	if err := Section(&buf, 2, "Skipped (%d)", 2); err != nil {
		t.Fatal(err)
	}
	if err := Section(&buf, 3, "WIP"); err != nil {
		t.Fatal(err)
	}
	if err := Para(&buf, "_None._"); err != nil {
		t.Fatal(err)
	}
	want := "\n## Skipped (2)\n\n### WIP\n\n_None._\n"
	if buf.String() != want {
		t.Errorf("spacing = %q, want %q", buf.String(), want)
	}
}

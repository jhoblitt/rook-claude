package rtanalyze

import (
	"encoding/json"
	"strings"
	"testing"
)

// Every expectation here was read off CPython 3 (json.dumps / repr / round).
// Non-ASCII and control codepoints appear only as \u escapes: a test for
// escaping that itself contains the raw characters cannot be reviewed by
// reading it.

func TestMarshalMatchesPythonIndent1(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want string
	}{
		{"empty object", Obj{}, "{}"},
		{"empty array", []any{}, "[]"},
		{"empty nested", Obj{{Key: "a", Val: []any{}}}, "{\n \"a\": []\n}"},
		{"nested", Obj{{Key: "k", Val: Obj{{Key: "n", Val: []any{1, "x"}}}}},
			"{\n \"k\": {\n  \"n\": [\n   1,\n   \"x\"\n  ]\n }\n}"},
		{"insertion order kept", Obj{{Key: "z", Val: 1}, {Key: "a", Val: 2}},
			"{\n \"z\": 1,\n \"a\": 2\n}"},
		{"float", Obj{{Key: "w", Val: 3.0}}, "{\n \"w\": 3.0\n}"},
		{"null and bool", []any{nil, true, false}, "[\n null,\n true,\n false\n]"},
	}
	for _, tc := range tests {
		if got := Marshal(tc.val); got != tc.want {
			t.Errorf("%s: Marshal() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestMarshalCompactMatchesPythonSeparators(t *testing.T) {
	got := MarshalCompact(Obj{
		{Key: "1", Val: []any{"a"}},
		{Key: "2", Val: []any{}},
	})
	if want := `{"1": ["a"], "2": []}`; got != want {
		t.Errorf("MarshalCompact() = %q, want %q", got, want)
	}
}

// ensure_ascii is one reason encoding/json cannot stand in: Python escapes
// every non-ASCII rune and leaves <, > and & alone; Go does the opposite.
func TestMarshalEscapesLikeEnsureASCII(t *testing.T) {
	tests := []struct{ in, want string }{
		{"plain", `"plain"`},
		{"a\"b\\c", `"a\"b\\c"`},
		{"tab\tnl\ncr\r", `"tab\tnl\ncr\r"`},
		{"\u0007\u007f", `"\u0007\u007f"`},
		{"caf\u00e9 \u2014 x", `"caf\u00e9 \u2014 x"`},
		{"\U0001f600", `"\ud83d\ude00"`},
		{"<b>&amp;</b>", `"<b>&amp;</b>"`},
	}
	for _, tc := range tests {
		if got := Marshal(tc.in); got != tc.want {
			t.Errorf("Marshal(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestPyFloat(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "0.0"},
		{1, "1.0"},
		{0.25, "0.25"},
		{1.5, "1.5"},
		{12.75, "12.75"},
		{1234.25, "1234.25"},
	}
	for _, tc := range tests {
		if got := pyFloat(tc.in); got != tc.want {
			t.Errorf("pyFloat(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRound2MatchesPythonRound(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{2.675, "2.67"},
		{0.125, "0.12"},
		{0.1 + 0.2, "0.3"},
		{1.0, "1.0"},
		{0.25, "0.25"},
		{12.75, "12.75"},
	}
	for _, tc := range tests {
		if got := pyFloat(round2(tc.in)); got != tc.want {
			t.Errorf("round2(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPyReprString(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", "''"},
		{"plain", "'plain'"},
		{"it's broken", `"it's broken"`},
		{"both ' and \"", `'both \' and "'`},
		{"back\\slash", `'back\\slash'`},
		{"nl\nx", `'nl\nx'`},
		{"\u0007", `'\x07'`},
		{"\u007f", `'\x7f'`},
		{"caf\u00e9", "'caf\u00e9'"},
		{"\u200b", `'\u200b'`},
	}
	for _, tc := range tests {
		if got := pyReprString(tc.in); got != tc.want {
			t.Errorf("pyReprString(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestPyReprCollections(t *testing.T) {
	if got, want := pyReprInts([]int{3, 1, 2}), "[3, 1, 2]"; got != want {
		t.Errorf("pyReprInts() = %s, want %s", got, want)
	}
	if got, want := pyReprInts(nil), "[]"; got != want {
		t.Errorf("pyReprInts(nil) = %s, want %s", got, want)
	}
	if got, want := pyReprStrings([]string{"a", "b"}), "['a', 'b']"; got != want {
		t.Errorf("pyReprStrings() = %s, want %s", got, want)
	}
	counts := []countedType{{"bucket-ambiguity", 5}, {"truncation", 2}}
	if got, want := pyReprCounts(counts), "{'bucket-ambiguity': 5, 'truncation': 2}"; got != want {
		t.Errorf("pyReprCounts() = %s, want %s", got, want)
	}
	if got, want := pyReprCounts(nil), "{}"; got != want {
		t.Errorf("pyReprCounts(nil) = %s, want %s", got, want)
	}
}

func TestPyStrNumber(t *testing.T) {
	if got := pyStrNumber(nil); got != "None" {
		t.Errorf("pyStrNumber(nil) = %q, want None", got)
	}
	for in, want := range map[string]string{"3": "3", "-0": "0", "2.5": "2.5", "4.0": "4.0"} {
		n := json.Number(in)
		if got := pyStrNumber(&n); got != want {
			t.Errorf("pyStrNumber(%s) = %q, want %q", in, got, want)
		}
	}
}

func TestMarshalIndentsOneSpacePerLevel(t *testing.T) {
	got := Marshal(Obj{{Key: "a", Val: Obj{{Key: "b", Val: Obj{{Key: "c", Val: 1}}}}}})
	if !strings.Contains(got, "\n   \"c\": 1\n") {
		t.Errorf("third level not indented by 3 spaces: %q", got)
	}
}

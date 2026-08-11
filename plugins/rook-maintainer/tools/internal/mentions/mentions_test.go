package mentions

import (
	"slices"
	"strings"
	"testing"
)

// Fences and hostile codepoints appear as \u escapes or as fence built from a
// named constant: a test for invisible-character handling that itself contains
// invisible characters cannot be reviewed by reading it.
const (
	tick  = "`"
	fence = tick + tick + tick
)

func TestStripCode(t *testing.T) {
	tests := []struct {
		name, doc, want string
	}{
		{"plain text", "@a talks to @b", "@a talks to @b"},
		{"backtick fence", "@a\n" + fence + "\n@hidden\n" + fence + "\n@b", "@a\n@b"},
		{"tilde fence", "@a\n~~~\n@hidden\n~~~\n@b", "@a\n@b"},
		{"info string", "@a\n" + fence + "go\nfunc @hidden()\n" + fence + "\n@b", "@a\n@b"},
		{"quoted fence", "@a\n> " + fence + "\n> @hidden\n> " + fence + "\n@b", "@a\n@b"},
		{"four leading spaces", "@a\n    " + fence + "\n@hidden\n    " + fence + "\n@b", "@a\n@b"},
		{"five leading spaces is not a fence",
			"@a\n     " + fence + "\n@hidden\n     " + fence + "\n@b",
			"@a\n      " + tick + "\n@hidden\n      " + tick + "\n@b"},
		{"four quote markers", "@a\n>>>>" + fence + "\n@hidden\n>>>>" + fence + "\n@b", "@a\n@b"},
		{"tab before fence", "@a\n\t" + fence + "\n@hidden\n\t" + fence + "\n@b", "@a\n@b"},
		{"non-breaking space before fence",
			"@a\n\u00a0" + fence + "\n@hidden\n\u00a0" + fence + "\n@b", "@a\n@b"},
		{"C0 separator counts as space",
			"@a\n\u001f" + fence + "\n@hidden\n\u001f" + fence + "\n@b", "@a\n@b"},
		{"unterminated fence swallows the tail", "@a\n" + fence + "\n@hidden", "@a"},
		{"longer backtick run still toggles",
			"@a\n" + fence + tick + "\n@hidden\n" + fence + tick + "\n@b", "@a\n@b"},
		{"inner longer run does not reopen",
			"@a\n" + fence + "\n@x\n" + fence + tick + "\n@y\n" + fence + "\n@b", "@a\n@y"},
		{"three fences leave the middle visible",
			"@a\n" + fence + "\n@x\n" + fence + "\n@y\n" + fence + "\n@z\n" + fence + "\n@b",
			"@a\n@y\n@b"},
		{"inline code", "@a " + tick + "@inline" + tick + " @b", "@a   @b"},
		{"empty inline pair", "@a " + tick + tick + "@inline" + tick + tick + " @b",
			"@a  @inline  @b"},
		{"inline triple run", "@a " + fence + "x" + fence + " @b", "@a     @b"},
		{"inline does not span lines", tick + "@a\n@b" + tick, tick + "@a\n@b" + tick},
		{"unpaired backtick survives", "@a " + tick + " @b", "@a " + tick + " @b"},
		{"crlf", "@a\r\n" + fence + "\r\n@hidden\r\n" + fence + "\r\n@b", "@a\n@b"},
		{"form feed is a line boundary", "@a\u000c" + fence + "\u000c@hidden\u000c" + fence + "\u000c@b",
			"@a\n@b"},
		{"file separator is a line boundary", "@a\n\u001c" + fence + "@hidden", "@a\n"},
		{"trailing newline dropped", "@a\n", "@a"},
		{"empty", "", ""},
	}
	for _, tc := range tests {
		if got := StripCode(tc.doc); got != tc.want {
			t.Errorf("%s: StripCode(%q) = %q, want %q", tc.name, tc.doc, got, tc.want)
		}
	}
}

func TestExtract(t *testing.T) {
	long39 := strings.Repeat("a", 39)
	tests := []struct {
		name, text string
		want       []string
	}{
		{"bare", "@alice", []string{"alice"}},
		{"mid sentence", "hi @alice!", []string{"alice"}},
		{"parenthesized", "(@alice)", []string{"alice"}},
		{"trailing period", "@alice.", []string{"alice"}},
		{"email", "user@example.com", nil},
		{"dotted email local part", "email: foo.bar@baz.io @real", []string{"real"}},
		{"shell prompt", "root@pod-name", nil},
		{"after slash", "/@alice", nil},
		{"after dot", ".@alice", nil},
		{"after hyphen", "-@alice", nil},
		{"double at", "@@alice", nil},
		{"chained", "@alice@bob", []string{"alice"}},
		{"leading hyphen", "@-foo", nil},
		{"trailing underscore", "@alice_", nil},
		{"underscore inside", "@a_b-c", nil},
		{"backtracks to a hyphen", "@ab-cd_", []string{"ab"}},
		{"backtracks to the second hyphen", "@ab--cd_", []string{"ab-"}},
		{"trailing hyphen kept", "@alice-", []string{"alice-"}},
		{"hyphenated", "@foo-bar baz", []string{"foo-bar"}},
		{"all hyphens kept", "@a-b-c-", []string{"a-b-c-"}},
		{"digit start", "@1a", []string{"1a"}},
		{"single char", "@a", []string{"a"}},
		{"39 chars", "@" + long39, []string{long39}},
		{"40 chars", "@" + strings.Repeat("a", 40), nil},
		{"39 chars then hyphen", "@" + long39 + "-", []string{long39}},
		{"cap falls back to a hyphen",
			"@" + strings.Repeat("a", 38) + "-b", []string{strings.Repeat("a", 38)}},
		{"preceded by a non-ascii letter", "é@alice", nil},
		{"followed by a non-ascii letter", "@aliceé", nil},
		{"non-ascii after a hyphen run", "@ab-cdé", []string{"ab"}},
		{"case preserved and repeated", "cc @Alice and @ALICE and @alice",
			[]string{"Alice", "ALICE", "alice"}},
		{"newline separated", "@a\n@b", []string{"a", "b"}},
		{"no mention", "nothing here", nil},
	}
	for _, tc := range tests {
		if got := Extract(tc.text); !slices.Equal(got, tc.want) {
			t.Errorf("%s: Extract(%q) = %v, want %v", tc.name, tc.text, got, tc.want)
		}
	}
}

func TestCandidates(t *testing.T) {
	tests := []struct {
		name string
		docs []string
		want []string
	}{
		{"first-mention order, case-insensitive dedupe",
			[]string{"@Bob and @alice", "@ALICE again", "@bob"},
			[]string{"Bob", "alice"}},
		{"an unclosed fence does not leak into the next comment",
			[]string{"@a\n" + fence + "\n@hidden", "@b"},
			[]string{"a", "b"}},
		{"code in one comment, prose in another",
			[]string{tick + "@root" + tick, "ping @alice"},
			[]string{"alice"}},
		{"documents are not concatenated into one token",
			[]string{"@ali", "ce @bob"},
			[]string{"ali", "bob"}},
		{"nothing to mine", []string{"", fence + "\n@x\n" + fence}, nil},
	}
	for _, tc := range tests {
		if got := Candidates(tc.docs); !slices.Equal(got, tc.want) {
			t.Errorf("%s: Candidates(%q) = %v, want %v", tc.name, tc.docs, got, tc.want)
		}
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name, in string
		want     []string
	}{
		{"empty", "", nil},
		{"no boundary", "abc", []string{"abc"}},
		{"trailing newline", "a\n", []string{"a"}},
		{"blank last line", "a\n\n", []string{"a", ""}},
		{"crlf is one boundary", "a\r\nb", []string{"a", "b"}},
		{"lone cr", "a\rb", []string{"a", "b"}},
		{"vertical tab", "a\u000bb", []string{"a", "b"}},
		{"form feed", "a\u000cb", []string{"a", "b"}},
		{"file separator", "a\u001cb", []string{"a", "b"}},
		{"next line", "a\u0085b", []string{"a", "b"}},
		{"line separator", "a\u2028b", []string{"a", "b"}},
		{"unit separator is not a boundary", "a\u001fb", []string{"a\u001fb"}},
	}
	for _, tc := range tests {
		if got := splitLines(tc.in); !slices.Equal(got, tc.want) {
			t.Errorf("%s: splitLines(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

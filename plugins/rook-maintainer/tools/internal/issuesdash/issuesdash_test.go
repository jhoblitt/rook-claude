package issuesdash

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden dashboards")

// The format's non-ASCII furniture is spelled with escapes: an expectation
// that hinges on a dash or an ellipsis cannot be reviewed if the source shows
// a character whose codepoint has to be guessed.
const (
	ellipsis = "\u2026"
	diamond  = "\u25c7"

	sweepDir = "testdata/sweep/2026-08-10-issues"
	emptyDir = "testdata/empty/2026-01-02-issues"
)

func TestClass(t *testing.T) {
	tests := []struct{ disposition, want string }{
		{"needs-info " + emDash + " no logs", "info"},
		{"Needs-Info " + emDash + " no logs", "info"},
		{"close-candidate " + emDash + " duplicate", "close"},
		{"propose close as answered", "close"},
		{"fixed-by-merged #15004", "close"},
		{"answered in thread", "close"},
		{"support " + emDash + " nothing to fix", "close"},
		{"adjudicated " + emDash + " no action", "close"},
		{"fix-open " + emDash + " PR 15004", "fix"},
		{"blocked-upstream " + emDash + " ceph tracker", "up"},
		{"upstream " + emDash + " ceph tracker", "up"},
		{"keep-open " + emDash + " upstream fix landed", "keep"},
		{"keep-open " + emDash + " reproducible", "keep"},
		{"close the loop with the reporter", "keep"},
		{"", "keep"},
	}
	for _, tc := range tests {
		if got := Class(tc.disposition); got != tc.want {
			t.Errorf("Class(%q) = %q, want %q", tc.disposition, got, tc.want)
		}
	}
}

func TestLinkify(t *testing.T) {
	tests := []struct {
		name, text string
		want       []Seg
	}{
		{"no reference", "nothing to link", []Seg{{Text: "nothing to link"}}},
		{"empty", "", nil},
		{"hashed", "see #4123 now", []Seg{{Text: "see "}, {Ref: "4123"}, {Text: " now"}}},
		{"bare", "tracked by 14002", []Seg{{Text: "tracked by "}, {Ref: "14002"}}},
		{"five digits above range", "superseded 19000", []Seg{{Text: "superseded 19000"}}},
		{"version and year", "v1.18 in 2024", []Seg{{Text: "v1.18 in 2024"}}},
		{"digits glued right", "tracker 65001", []Seg{{Text: "tracker 65001"}}},
		{"digits glued left", "abc4123", []Seg{{Text: "abc4123"}}},
		{"hash after a word", "x#4444", []Seg{{Text: "x"}, {Ref: "4444"}}},
		{"non-ASCII letter is a word rune", "\u00e95000", []Seg{{Text: "\u00e95000"}}},
		{"two references", "#4123 and 18999", []Seg{{Ref: "4123"}, {Text: " and "}, {Ref: "18999"}}},
	}
	for _, tc := range tests {
		if got := Linkify(tc.text); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: Linkify(%q) = %+v, want %+v", tc.name, tc.text, got, tc.want)
		}
	}
}

func TestPullRefs(t *testing.T) {
	s := &Sweep{RefTypes: map[string]string{
		"15004": "PullRequest", "15006": "PullRequest", "15005": "Issue",
	}}
	raw := func(ss ...string) []json.RawMessage {
		out := make([]json.RawMessage, 0, len(ss))
		for _, x := range ss {
			out = append(out, json.RawMessage(x))
		}
		return out
	}
	tests := []struct {
		name string
		item Item
		want []string
	}{
		{"none", Item{}, nil},
		{"only pull requests survive", Item{XLinks: raw(`{"number":15004}`, `{"number":15005}`)},
			[]string{"15004"}},
		{"unclassified is dropped", Item{XLinks: raw(`{"number":90210}`)}, nil},
		{"deduped across xlinks and dups",
			Item{XLinks: raw(`{"number":15004}`, `{"number":15004}`), Dups: raw(`{"number":15004}`)},
			[]string{"15004"}},
		{"xlinks before dups",
			Item{XLinks: raw(`{"number":15006}`), Dups: raw(`{"number":15004}`)},
			[]string{"15006", "15004"}},
		{"non-object and number-less entries are skipped",
			Item{XLinks: raw(`"15004"`, `{"title":"no number"}`, `{"number":15004}`)},
			[]string{"15004"}},
	}
	for _, tc := range tests {
		if got := s.PullRefs(tc.item); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: PullRefs() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestMentionRows(t *testing.T) {
	nine := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
	s := &Sweep{Mentions: map[string][]string{"1": nine, "2": {"a"}, "3": {}}}
	tests := []struct {
		name string
		item Item
		want []MentionRow
	}{
		{"nothing", Item{Number: "3"}, nil},
		{"mentioned only", Item{Number: "2"}, []MentionRow{{Login: "a"}}},
		{"routing only", Item{Number: "3", Routing: []json.RawMessage{[]byte(`"travisn"`)}},
			[]MentionRow{{Login: "travisn", Proposed: true}}},
		{"overflow marker after the cap", Item{Number: "1"}, []MentionRow{
			{Login: "a"}, {Login: "b"}, {Login: "c"}, {Login: "d"},
			{Login: "e"}, {Login: "f"}, {Login: "g"}, {Login: "h"}, {More: 1},
		}},
		{"routing follows mentions", Item{Number: "2", Routing: []json.RawMessage{[]byte(`"z"`)}},
			[]MentionRow{{Login: "a"}, {Login: "z", Proposed: true}}},
	}
	for _, tc := range tests {
		if got := s.MentionRows(tc.item); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: MentionRows() = %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestSweepDate(t *testing.T) {
	tests := []struct{ dir, want string }{
		{"/s/2026-08-10-issues", "2026-08-10"},
		{"/s/2026-08-10-issues/", "2026-08-10"},
		{"short", "short"},
	}
	for _, tc := range tests {
		if got := SweepDate(tc.dir); got != tc.want {
			t.Errorf("SweepDate(%q) = %q, want %q", tc.dir, got, tc.want)
		}
	}
}

func render(t *testing.T, name string, data any) string {
	t.Helper()
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		t.Fatalf("executing %q: %v", name, err)
	}
	return buf.String()
}

func TestCells(t *testing.T) {
	tests := []struct {
		name, cell, want string
		data             any
	}{
		{"empty list", "ul", `<ul class="list"><li>` + emDash + `</li></ul>`, List{}},
		{"list", "ul", `<ul class="list"><li>bug</li><li>ceph-mon</li></ul>`,
			List{Items: []string{"bug", "ceph-mon"}}},
		{"bold list", "ul", `<ul class="list"><li><b>duplicate</b></li></ul>`,
			List{Items: []string{"duplicate"}, Bold: true}},
		{"label markup is escaped", "ul",
			`<ul class="list"><li>&lt;b&gt;x&lt;/b&gt;&amp;y</li></ul>`,
			List{Items: []string{"<b>x</b>&y"}}},
		{"login", "login",
			`<a class="u" href="https://github.com/travisn">travisn</a>`, "travisn"},
		{"copilot is renamed and pointed at the app", "login",
			`<a class="u" href="https://github.com/apps/copilot-pull-request-reviewer">Copilot</a>`,
			"copilot-pull-request-reviewer"},
		{"hostile login cannot break out of the href", "login",
			`<a class="u" href="https://github.com/evil%22%20x">evil&#34; x</a>`, `evil" x`},
		{"no assignees", "logins", `<ul class="list"><li>` + emDash + `</li></ul>`, []string(nil)},
		{"no refs", "refs", emDash, []string(nil)},
		{"refs", "refs",
			`<ul class="list"><li><a href="https://github.com/rook/rook/pull/15004">#15004</a></li></ul>`,
			[]string{"15004"}},
		{"disposition links references", "disposition",
			`fix-open ` + emDash + ` PR <a href="https://github.com/rook/rook/issues/15004">#15004</a> carries it`,
			Linkify("fix-open " + emDash + " PR 15004 carries it")},
		{"disposition markup is escaped", "disposition",
			`&lt;script&gt;alert(1)&lt;/script&gt;`, Linkify("<script>alert(1)</script>")},
		{"no mentions", "mentions", emDash, []MentionRow(nil)},
		{"mentioned, overflow and proposed rows", "mentions",
			`<div class="rv">` +
				`<div class="r"><span class="nm"><a class="u" href="https://github.com/travisn">travisn</a></span>` +
				`<span class="st ment" title="mentioned in thread"><i>@</i></span></div>` +
				`<div class="r"><span class="nm">` + ellipsis + `</span><span class="st ment">+2 more</span></div>` +
				`<div class="r"><span class="nm"><a class="u" href="https://github.com/sp98">sp98</a></span>` +
				`<span class="st prop" title="proposed @-mention (triage routing)"><i>` + diamond + `</i></span></div>` +
				`</div>`,
			[]MentionRow{{Login: "travisn"}, {More: 2}, {Login: "sp98", Proposed: true}}},
		{"labels without a proposal", "labels",
			`<div class="rev"><div class="sub"><span class="k">current</span>` +
				`<ul class="list"><li>bug</li></ul></div></div>`,
			LabelsCell{Current: List{Items: []string{"bug"}}}},
		{"labels with a proposal", "labels",
			`<div class="rev"><div class="sub"><span class="k">current</span>` +
				`<ul class="list"><li>bug</li></ul></div>` +
				`<div class="sub"><span class="k">propose +</span>` +
				`<ul class="list"><li><b>wontfix</b></li></ul></div></div>`,
			LabelsCell{
				Current:  List{Items: []string{"bug"}},
				Proposed: List{Items: []string{"wontfix"}, Bold: true},
			}},
	}
	for _, tc := range tests {
		if got := render(t, tc.cell, tc.data); got != tc.want {
			t.Errorf("%s: %s = %q, want %q", tc.name, tc.cell, got, tc.want)
		}
	}
}

func TestRowColumnOrder(t *testing.T) {
	sweep, err := Load(sweepDir)
	if err != nil {
		t.Fatal(err)
	}
	row := render(t, "row", sweep.Page().Rows[0])
	if !strings.HasPrefix(row, `<tr class="fix"><td><a href="https://github.com/rook/rook/issues/4002">#4002</a></td>`) {
		t.Errorf("row = %q", row)
	}
	if n := strings.Count(row, "<td"); n != 9 {
		t.Errorf("row has %d cells, want 9: %q", n, row)
	}
	if strings.Contains(row, "\n") {
		t.Errorf("row spans more than one line: %q", row)
	}
}

func TestLoadOrdersItemsAcrossBatches(t *testing.T) {
	sweep, err := Load(sweepDir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range sweep.Page().Rows {
		got = append(got, r.Number)
	}
	want := []string{"4002", "7777", "9001", "12010", "15003", "18999"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("row order = %v, want %v", got, want)
	}
}

func TestLoadRejectsIncompleteInput(t *testing.T) {
	tests := []struct{ name, batch, snapshot string }{
		{"no snapshot", `[{"number":1,"disposition":"keep"}]`, ""},
		{"snapshot without items", `[{"number":1,"disposition":"keep"}]`, `{}`},
		{"item without a disposition", `[{"number":1}]`, `{"items":{}}`},
		{"item without a number", `[{"disposition":"keep"}]`, `{"items":{}}`},
	}
	for _, tc := range tests {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "batch-01.json"), []byte(tc.batch), 0o644); err != nil {
			t.Fatal(err)
		}
		if tc.snapshot != "" {
			if err := os.WriteFile(filepath.Join(dir, "snapshot.json"), []byte(tc.snapshot), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := Load(dir); err == nil {
			t.Errorf("%s: Load() succeeded", tc.name)
		}
	}
}

func TestGolden(t *testing.T) {
	tests := []struct{ name, dir, golden string }{
		{"populated sweep", sweepDir, "testdata/sweep.golden.html"},
		{"no rows, no optional inputs", emptyDir, "testdata/empty.golden.html"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sweep, err := Load(tc.dir)
			if err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer
			if err := Render(&buf, sweep.Page()); err != nil {
				t.Fatal(err)
			}
			if *update {
				if err := os.WriteFile(tc.golden, buf.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(tc.golden)
			if err != nil {
				t.Fatal(err)
			}
			if got := buf.Bytes(); !bytes.Equal(got, want) {
				t.Errorf("dashboard differs from %s at byte %d\n got: %q\nwant: %q",
					tc.golden, firstDiff(got, want), excerpt(got, firstDiff(got, want)),
					excerpt(want, firstDiff(got, want)))
			}
			if bytes.HasSuffix(buf.Bytes(), []byte("\n")) {
				t.Error("page ends with a newline")
			}
		})
	}
}

func TestReporterMarkupNeverReachesThePage(t *testing.T) {
	sweep, err := Load(sweepDir)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := Render(&buf, sweep.Page()); err != nil {
		t.Fatal(err)
	}
	page := buf.String()
	for _, bad := range []string{"<b>bold</b>", `evil" onmouseover`, `"Quoted"`} {
		if strings.Contains(page, bad) {
			t.Errorf("page contains unescaped reporter text %q", bad)
		}
	}
}

func firstDiff(a, b []byte) int {
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			return i
		}
	}
	return min(len(a), len(b))
}

func excerpt(b []byte, at int) string {
	end := min(at+80, len(b))
	return string(b[min(at, len(b)):end])
}

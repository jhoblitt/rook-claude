package reviewdash

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden dashboards")

// The csi golden is byte-for-byte what the Python generator this tool replaced
// emitted for the same fixture. The hostile golden is not: findings.json is
// agent-written from untrusted PR text, and html/template escapes three fields
// (`bug`, `repo` and a skip row's `number`) the Python interpolated raw, plus
// spells `'`, `"` and `+` as different-but-equivalent entities.
func TestRenderGolden(t *testing.T) {
	for _, name := range []string{"2026-08-10-csi", "2026-08-11-hostile"} {
		t.Run(name, func(t *testing.T) {
			sweep, err := Load(filepath.Join("testdata", name))
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			var got bytes.Buffer
			if err := sweep.Render(&got); err != nil {
				t.Fatalf("Render: %v", err)
			}
			golden := filepath.Join("testdata", name+".golden.html")
			if *update {
				if err := os.WriteFile(golden, got.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatal(err)
			}
			if got.String() != string(want) {
				t.Errorf("dashboard differs from %s (run go test -update to inspect)\n%s",
					golden, firstDifference(got.String(), string(want)))
			}
		})
	}
}

// Nothing an untrusted PR can put in findings.json or the snapshot may reach
// the page as live markup.
func TestHostileInputStaysInert(t *testing.T) {
	sweep, err := Load(filepath.Join("testdata", "2026-08-11-hostile"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var buf bytes.Buffer
	if err := sweep.Render(&buf); err != nil {
		t.Fatalf("Render: %v", err)
	}
	page := buf.String()
	for _, live := range []string{
		"<script>alert(",
		"<img src=x",
		"onmouseover=\"alert(",
		"</span><script>",
	} {
		if strings.Contains(page, live) {
			t.Errorf("rendered page contains live markup %q", live)
		}
	}
	if n := strings.Count(page, "<script>"); n != 1 {
		t.Errorf("page carries %d <script> elements, want only the sort/filter one", n)
	}
}

func TestLoad(t *testing.T) {
	sweep, err := Load(filepath.Join("testdata", "2026-08-10-csi"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if sweep.Date != "2026-08-10" {
		t.Errorf("Date = %q, want 2026-08-10", sweep.Date)
	}
	if sweep.Repo != "rook/rook" {
		t.Errorf("Repo = %q", sweep.Repo)
	}
	want := []string{"9012", "18101", "18102", "18103", "18104", "18105", "18106"}
	if len(sweep.recs) != len(want) {
		t.Fatalf("loaded %d records, want %d", len(sweep.recs), len(want))
	}
	for i, n := range want {
		if sweep.recs[i].num != n {
			t.Errorf("record[%d] = %q, want %q (PRs sort numerically, not by path)", i, sweep.recs[i].num, n)
		}
	}
	// pr-18105/findings.json is a bare array and carries no "pr" field.
	if got := sweep.recs[5]; len(got.Findings) != 1 || got.Verdict != nil {
		t.Errorf("array-shaped findings.json loaded as %+v", got)
	}
}

func TestLoadMissingInputsAreEmpty(t *testing.T) {
	sweep, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(sweep.recs) != 0 || len(sweep.skips) != 0 {
		t.Errorf("empty sweep dir yielded %+v", sweep)
	}
	if sweep.Repo != defaultRepo {
		t.Errorf("Repo = %q, want %q", sweep.Repo, defaultRepo)
	}
	var buf bytes.Buffer
	if err := sweep.Render(&buf); err != nil {
		t.Errorf("Render of an empty sweep: %v", err)
	}
}

func TestLoadRejectsUnusableInput(t *testing.T) {
	tests := []struct {
		name, file, body string
	}{
		{"malformed snapshot", "snapshot.json", "{"},
		{"malformed findings", filepath.Join("pr-1", "findings.json"), "{\"pr\": }"},
		{"undecidable PR number", filepath.Join("pr-abc", "findings.json"), "{}"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, tc.file)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(dir); err == nil {
				t.Error("Load accepted it; a dashboard missing half the sweep reads as a clean sweep")
			}
		})
	}
}

func TestRowClass(t *testing.T) {
	tests := []struct {
		name, raw, want string
	}{
		{"reject", `{"verdict":"REJECT"}`, "reject"},
		{"request changes", `{"verdict":"REQUEST_CHANGES"}`, "chg"},
		{"accept", `{"verdict":"ACCEPT"}`, "accept"},
		{"no verdict", `{}`, "mon"},
		{"unknown verdict", `{"verdict":"LGTM"}`, "mon"},
		{"proposal outranks the verdict", `{"verdict":"ACCEPT","needs_proposal_review":{"flag":true}}`, "prop"},
		{"takeover outranks the verdict", `{"verdict":"REJECT","takeover_candidate":{"flag":true}}`, "take"},
		{"proposal outranks takeover", `{"needs_proposal_review":{"flag":true},"takeover_candidate":{"flag":true}}`, "prop"},
		{"cleared flags fall through", `{"verdict":"ACCEPT","needs_proposal_review":{"flag":false},"takeover_candidate":{}}`, "accept"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := rowClass(mustRecord(t, tc.raw)); got != tc.want {
				t.Errorf("rowClass = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSevChips(t *testing.T) {
	tests := []struct {
		name, raw string
		want      []sevChip
		wantLive  int
	}{
		{
			name: "worst first regardless of input order",
			raw: `{"findings":[{"severity":"nit"},{"severity":"blocker"},
			       {"severity":"question"},{"severity":"changes-requested"},{"severity":"nit"}]}`,
			want: []sevChip{
				{Class: "sev-b", Letter: "B", Label: "blocker", Count: 1},
				{Class: "sev-c", Letter: "C", Label: "changes requested", Count: 1},
				{Class: "sev-n", Letter: "N", Label: "nit", Count: 2},
				{Class: "sev-q", Letter: "Q", Label: "design question", Count: 1},
			},
			wantLive: 5,
		},
		{
			name:     "dropped findings are not counted",
			raw:      `{"findings":[{"severity":"blocker","status":"dropped"},{"severity":"nit","status":"posted"}]}`,
			want:     []sevChip{{Class: "sev-n", Letter: "N", Label: "nit", Count: 1}},
			wantLive: 1,
		},
		{
			name:     "a missing severity counts as a nit",
			raw:      `{"findings":[{"id":"C2"}]}`,
			want:     []sevChip{{Class: "sev-n", Letter: "N", Label: "nit", Count: 1}},
			wantLive: 1,
		},
		{
			name:     "a severity outside the vocabulary gets no chip",
			raw:      `{"findings":[{"severity":"wontfix"}]}`,
			want:     nil,
			wantLive: 1,
		},
		{
			name:     "everything dropped",
			raw:      `{"findings":[{"severity":"blocker","status":"dropped"}]}`,
			want:     nil,
			wantLive: 0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			live := mustRecord(t, tc.raw).liveFindings()
			if len(live) != tc.wantLive {
				t.Fatalf("live findings = %d, want %d", len(live), tc.wantLive)
			}
			got := sevChips(live)
			if len(got) != len(tc.want) {
				t.Fatalf("sevChips = %+v, want %+v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("chip[%d] = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestCICell(t *testing.T) {
	tests := []struct {
		name, raw string
		want      ciCell
	}{
		{"all passing", `{"total":12,"passing":12}`, ciCell{Kind: "green", Passing: 12, Total: 12}},
		{
			"pending only", `{"total":9,"passing":7,"pending":2}`,
			ciCell{Kind: "amber", Passing: 7, Total: 9, Pending: 2},
		},
		{
			"failing wins over pending", `{"total":4,"passing":1,"failing":2,"pending":1,"failed":["a","b"]}`,
			ciCell{Kind: "red", Passing: 1, Total: 4, Failing: 2, Pending: 1, Failed: "a, b"},
		},
		{
			"failed names are capped",
			`{"total":9,"passing":0,"failing":8,"failed":["a","b","c","d","e","f","g","h"]}`,
			ciCell{Kind: "red", Total: 9, Failing: 8, Failed: "a, b, c, d, e, f", More: 2},
		},
		{"no checks recorded", `{"total":0,"passing":0}`, ciCell{Empty: true}},
		{"empty rollup", `{}`, ciCell{Empty: true}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var ci ciRollup
			if err := json.Unmarshal([]byte(tc.raw), &ci); err != nil {
				t.Fatal(err)
			}
			if got := ciCellFor(&ci); got != tc.want {
				t.Errorf("ciCellFor = %+v, want %+v", got, tc.want)
			}
		})
	}
	if got := ciCellFor(nil); got != (ciCell{Empty: true}) {
		t.Errorf("a PR absent from the snapshot rendered %+v", got)
	}
}

func TestFindingRows(t *testing.T) {
	raw := `{"findings":[
	  {"id":"N1","severity":"nit","path":"a.go","line":7,"status":"posted"},
	  {"severity":"wontfix","summary":"unknown severity sorts last"},
	  {"id":"B2","severity":"blocker","path":"b.go"},
	  {"id":"B1","severity":"blocker","path":"c.go","line":0},
	  {"id":"D1","severity":"nit","status":"dropped"},
	  {"id":"Q1","severity":"question","confidence":50}
	]}`
	got := findingRows(mustRecord(t, raw).liveFindings())
	want := []findingRow{
		{Class: "sev-b", ID: "B1", Anchor: "c.go", Confidence: "—", Status: "pending"},
		{Class: "sev-b", ID: "B2", Anchor: "b.go", Confidence: "—", Status: "pending"},
		{Class: "sev-n", ID: "N1", Anchor: "a.go:7", Confidence: "—", Status: "posted"},
		{Class: "sev-q", ID: "Q1", Anchor: "PR-level", Confidence: "PLAUSIBLE (50)", Status: "pending"},
		{Class: "sev-n", ID: "?", Anchor: "PR-level", Summary: "unknown severity sorts last",
			Confidence: "—", Status: "pending"},
	}
	if len(got) != len(want) {
		t.Fatalf("findingRows = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestConfidence(t *testing.T) {
	tests := []struct {
		raw, want string
	}{
		{"100", "CONFIRMED (100)"},
		{"80", "CONFIRMED (80)"},
		{"79", "PLAUSIBLE (79)"},
		{"50", "PLAUSIBLE (50)"},
		{"49", "— (49)"},
		{"0", "—"},
		{"85.5", "—"},
		{`"85"`, "—"},
		{"null", "—"},
	}
	for _, tc := range tests {
		t.Run(tc.raw, func(t *testing.T) {
			var v value
			if err := json.Unmarshal([]byte(tc.raw), &v); err != nil {
				t.Fatal(err)
			}
			if got := confidence(v); got != tc.want {
				t.Errorf("confidence(%s) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestSummary(t *testing.T) {
	sweep, err := Load(filepath.Join("testdata", "2026-08-10-csi"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := sweep.Summary()
	want := "2026-08-10-csi: 7 PRs — 4 ACCEPT, 1 REJECT, 1 REQUEST_CHANGES, 1 —; 10 live findings -> "
	if !strings.HasPrefix(got, want) {
		t.Errorf("Summary = %q, want prefix %q", got, want)
	}
	if !strings.HasSuffix(got, filepath.Join(sweep.Dir, "dashboard.html")) {
		t.Errorf("Summary does not end at the dashboard path: %q", got)
	}
}

func TestValueSemantics(t *testing.T) {
	tests := []struct {
		raw    string
		str    string
		truthy bool
		isInt  bool
	}{
		{`"text"`, "text", true, false},
		{`""`, "", false, false},
		{"42", "42", true, true},
		{"0", "0", false, true},
		{"-3", "-3", true, true},
		{"1.5", "1.5", true, false},
		{"true", "True", true, false},
		{"false", "False", false, false},
		{"null", "", false, false},
		{"[]", "[]", false, false},
		{`["a"]`, `["a"]`, true, false},
	}
	for _, tc := range tests {
		t.Run(tc.raw, func(t *testing.T) {
			var v value
			if err := json.Unmarshal([]byte(tc.raw), &v); err != nil {
				t.Fatalf("Unmarshal(%s): %v", tc.raw, err)
			}
			if v.str != tc.str || v.truthy != tc.truthy || v.isInt != tc.isInt {
				t.Errorf("%s -> %+v, want str=%q truthy=%v isInt=%v", tc.raw, v, tc.str, tc.truthy, tc.isInt)
			}
		})
	}
}

func TestFirstRunes(t *testing.T) {
	// Truncation is by rune: cutting the rationale mid-codepoint would put a
	// replacement character on the page.
	if got := firstRunes("2026-08-10-csi", 10); got != "2026-08-10" {
		t.Errorf("firstRunes = %q", got)
	}
	if got := firstRunes("short", 10); got != "short" {
		t.Errorf("firstRunes = %q", got)
	}
	if got := firstRunes("aaaa—bbbb—cccc", 6); got != "aaaa—b" {
		t.Errorf("firstRunes = %q", got)
	}
}

func mustRecord(t *testing.T, raw string) record {
	t.Helper()
	var r record
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatalf("fixture %s: %v", raw, err)
	}
	return r
}

func firstDifference(got, want string) string {
	gl, wl := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := range min(len(gl), len(wl)) {
		if gl[i] != wl[i] {
			return fmt.Sprintf("line %d:\n got: %s\nwant: %s", i+1, gl[i], wl[i])
		}
	}
	return fmt.Sprintf("got %d lines, want %d", len(gl), len(wl))
}

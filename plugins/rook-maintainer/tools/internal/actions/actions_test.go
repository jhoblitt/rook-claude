package actions

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// Hostile codepoints appear only as \u escapes: a test for invisible-character
// handling that itself contains invisible characters cannot be reviewed by
// reading it.

var testLive = []string{"bug", "feature", "needs-info", "ceph-object", "core", "docs"}

const testSnapshot = `[
  {"number": 1, "type": "issue", "state": "OPEN", "labels": [{"name": "bug"}]},
  {"number": 2, "type": "pr", "state": "OPEN", "labels": []},
  {"number": 3, "type": "issue", "state": "CLOSED", "labels": []},
  {"number": 4, "type": "issue", "state": "OPEN",
   "labels": [{"name": "bug"}, {"name": "core"}, {"name": "docs"}]}
]`

func mustParse(t *testing.T, payload string) Payload {
	t.Helper()
	p, err := Parse([]byte(payload))
	if err != nil {
		t.Fatalf("Parse(%s) = %v", payload, err)
	}
	return p
}

func mustItems(t *testing.T, snapshot string) []Item {
	t.Helper()
	items, err := ParseItems([]byte(snapshot))
	if err != nil {
		t.Fatalf("ParseItems: %v", err)
	}
	return items
}

func problems(t *testing.T, payload string, items []Item) []string {
	t.Helper()
	return Validate(mustParse(t, payload), testLive, items)
}

func TestValidateAccepts(t *testing.T) {
	items := mustItems(t, testSnapshot)
	tests := []struct{ name, payload string }{
		{"label an open issue", `[{"number": 1, "action": "label", "params": {"labels": ["needs-info"]}}]`},
		{"reviewers on a pr", `[{"number": 2, "action": "reviewers", "params": {"reviewers": ["a", "b"]}}]`},
		{"mentions at the cap", `[{"number": 1, "action": "comment", "params": {"mentions": ["a", "b", "c"]}}]`},
		{"labels at the cap", `[{"number": 4, "action": "label", "params": {"labels": ["feature", "needs-info"]}}]`},
		{"one reviewer", `[{"number": 2, "action": "reviewers", "params": {"reviewers": ["a"]}}]`},
		{"five reviewers", `[{"number": 2, "action": "reviewers", "params": {"reviewers": ["a", "b", "c", "d", "e"]}}]`},
		{"close needs no params", `[{"number": 1, "action": "close"}]`},
		{"convert needs no params", `[{"number": 1, "action": "convert", "params": {}}]`},
		{"comment without mentions", `[{"number": 1, "action": "comment", "params": {}}]`},
		{"uppercase action", `[{"number": 1, "action": "LABEL", "params": {"labels": ["bug"]}}]`},
		{"labels as gh objects", `[{"number": 1, "action": "label", "params": {"labels": [{"name": "docs"}]}}]`},
		{"empty payload", `[]`},
	}
	for _, tc := range tests {
		if got := problems(t, tc.payload, items); len(got) != 0 {
			t.Errorf("%s: Validate() = %v, want none", tc.name, got)
		}
	}
}

func TestValidateRejects(t *testing.T) {
	items := mustItems(t, testSnapshot)
	tests := []struct{ name, payload, want string }{
		{
			"invented labels, reported sorted",
			`[{"number": 1, "action": "label", "params": {"labels": ["invented", "alsobad"]}}]`,
			"actions[0] #1: label(s) not in the live list: alsobad, invented",
		},
		{
			"label on a pr",
			`[{"number": 2, "action": "label", "params": {"labels": ["bug"]}}]`,
			"actions[0] #2: label action on a PR — triage labels issues only",
		},
		{
			"pr kind from the action beats the snapshot",
			`[{"number": 1, "type": "PR", "action": "label", "params": {"labels": ["bug"]}}]`,
			"actions[0] #1: label action on a PR — triage labels issues only",
		},
		{
			"closed item",
			`[{"number": 3, "action": "label", "params": {"labels": ["bug"]}}]`,
			"actions[0] #3: item is 'CLOSED', not OPEN — re-assess before writing",
		},
		{
			"label cap counts the labels already on the item",
			`[{"number": 4, "action": "label", "params": {"labels": ["feature", "ceph-object", "needs-info"]}}]`,
			"actions[0] #4: 6 labels after apply exceeds the cap of 5 " +
				"(bug, ceph-object, core, docs, feature, needs-info)",
		},
		{
			"no reviewers",
			`[{"number": 2, "action": "reviewers", "params": {"reviewers": []}}]`,
			"actions[0] #2: 0 reviewers is outside 1–5",
		},
		{
			"missing reviewers key",
			`[{"number": 2, "action": "reviewers", "params": {}}]`,
			"actions[0] #2: 0 reviewers is outside 1–5",
		},
		{
			"too many reviewers",
			`[{"number": 2, "action": "reviewers", "params": {"reviewers": ["a", "b", "c", "d", "e", "f"]}}]`,
			"actions[0] #2: 6 reviewers is outside 1–5",
		},
		{
			"too many mentions",
			`[{"number": 1, "action": "comment", "params": {"mentions": ["a", "b", "c", "d"]}}]`,
			"actions[0] #1: 4 mentions exceeds the cap of 3",
		},
		{
			"item absent from the snapshot",
			`[{"number": 9, "action": "label", "params": {"labels": ["bug"]}}]`,
			"actions[0] #9: no live state supplied for this item",
		},
		{
			"unknown action",
			`[{"number": 1, "action": "frobnicate", "params": {}}]`,
			"actions[0] #1: unknown action 'frobnicate'",
		},
		{
			"missing action",
			`[{"number": 1, "params": {}}]`,
			"actions[0] #1: unknown action ''",
		},
		{
			"missing number",
			`[{"action": "label", "params": {"labels": ["bug"]}}]`,
			"actions[0]: missing `number`",
		},
		{
			"null number",
			`[{"number": null, "action": "label", "params": {"labels": ["bug"]}}]`,
			"actions[0]: missing `number`",
		},
		{
			"element is not an object",
			`["label #1"]`,
			"actions[0]: not an object",
		},
		{
			"null element",
			`[null]`,
			"actions[0]: not an object",
		},
		{
			"label with no labels",
			`[{"number": 1, "action": "label", "params": {"labels": []}}]`,
			"actions[0] #1: label action with no labels",
		},
		{
			"label with only empty names",
			`[{"number": 1, "action": "label", "params": {"labels": ["", {"name": ""}]}}]`,
			"actions[0] #1: label action with no labels",
		},
	}
	for _, tc := range tests {
		got := problems(t, tc.payload, items)
		if len(got) != 1 || got[0] != tc.want {
			t.Errorf("%s: Validate() = %v, want [%q]", tc.name, got, tc.want)
		}
	}
}

func TestItemWithoutState(t *testing.T) {
	items := mustItems(t, `[{"number": 5, "type": "issue", "labels": []}]`)
	got := problems(t, `[{"number": 5, "action": "label", "params": {"labels": ["bug"]}}]`, items)
	want := "actions[0] #5: item is None, not OPEN — re-assess before writing"
	if len(got) != 1 || got[0] != want {
		t.Errorf("Validate() = %v, want [%q]", got, want)
	}
}

// An invented label does not stop the cap check: both land in the same report.
func TestValidateReportsEveryLabelProblem(t *testing.T) {
	got := problems(t,
		`[{"number": 4, "action": "label", "params": {"labels": ["invented", "feature", "needs-info"]}}]`,
		mustItems(t, testSnapshot))
	want := []string{
		"actions[0] #4: label(s) not in the live list: invented",
		"actions[0] #4: 6 labels after apply exceeds the cap of 5 " +
			"(bug, core, docs, feature, invented, needs-info)",
	}
	if !slices.Equal(got, want) {
		t.Errorf("Validate() = %v, want %v", got, want)
	}
}

func TestValidateWalksEveryEntry(t *testing.T) {
	got := problems(t, `[
	  {"number": 1, "action": "label", "params": {"labels": ["bug"]}},
	  {"number": 2, "action": "label", "params": {"labels": ["bug"]}},
	  {"number": 1, "action": "frobnicate"}
	]`, mustItems(t, testSnapshot))
	want := []string{
		"actions[1] #2: label action on a PR — triage labels issues only",
		"actions[2] #1: unknown action 'frobnicate'",
	}
	if !slices.Equal(got, want) {
		t.Errorf("Validate() = %v, want %v", got, want)
	}
}

// Without a snapshot the open/PR checks are skipped, not silently passed.
func TestNoSnapshotSkipsLiveChecks(t *testing.T) {
	for _, payload := range []string{
		`[{"number": 3, "action": "label", "params": {"labels": ["bug"]}}]`,
		`[{"number": 2, "action": "label", "params": {"labels": ["bug"]}}]`,
		`[{"number": 99, "action": "label", "params": {"labels": ["bug"]}}]`,
	} {
		if got := problems(t, payload, nil); len(got) != 0 {
			t.Errorf("%s: Validate() = %v, want none", payload, got)
		}
	}
	// The label rules still apply.
	got := problems(t, `[{"number": 3, "action": "label", "params": {"labels": ["invented"]}}]`, nil)
	want := "actions[0] #3: label(s) not in the live list: invented"
	if len(got) != 1 || got[0] != want {
		t.Errorf("Validate() = %v, want [%q]", got, want)
	}
}

// An empty snapshot is a supplied snapshot: every item is unknown, which is a
// rejection. Conflating it with "no --items" would pass writes unchecked.
func TestEmptySnapshotIsNotAMissingSnapshot(t *testing.T) {
	empty := mustItems(t, `[]`)
	if empty == nil {
		t.Fatal("ParseItems(`[]`) returned nil, which reads as no snapshot")
	}
	got := problems(t, `[{"number": 1, "action": "label", "params": {"labels": ["bug"]}}]`, empty)
	want := "actions[0] #1: no live state supplied for this item"
	if len(got) != 1 || got[0] != want {
		t.Errorf("Validate() = %v, want [%q]", got, want)
	}
	if null := mustItems(t, `null`); null != nil {
		t.Errorf("ParseItems(`null`) = %v, want nil", null)
	}
}

func TestLowercaseStateIsOpen(t *testing.T) {
	items := mustItems(t, `[{"number": 1, "type": "issue", "state": "open", "labels": []}]`)
	if got := problems(t, `[{"number": 1, "action": "label", "params": {"labels": ["bug"]}}]`, items); len(got) != 0 {
		t.Errorf("Validate() = %v, want none", got)
	}
}

func TestKindFallsBackToTheSnapshot(t *testing.T) {
	items := mustItems(t, `[{"number": 7, "type": "PR", "state": "OPEN", "labels": []}]`)
	got := problems(t, `[{"number": 7, "action": "label", "params": {"labels": ["bug"]}}]`, items)
	want := "actions[0] #7: label action on a PR — triage labels issues only"
	if len(got) != 1 || got[0] != want {
		t.Errorf("Validate() = %v, want [%q]", got, want)
	}
}

func TestNumbersMatchAcrossJSONSpelling(t *testing.T) {
	items := mustItems(t, `[{"number": 1.0, "type": "issue", "state": "CLOSED", "labels": []}]`)
	got := problems(t, `[{"number": 1, "action": "label", "params": {"labels": ["bug"]}}]`, items)
	want := "actions[0] #1: item is 'CLOSED', not OPEN — re-assess before writing"
	if len(got) != 1 || got[0] != want {
		t.Errorf("Validate() = %v, want [%q]", got, want)
	}
}

func TestLastSnapshotEntryWins(t *testing.T) {
	items := mustItems(t, `[
	  {"number": 1, "type": "issue", "state": "OPEN", "labels": []},
	  {"number": 1, "type": "issue", "state": "CLOSED", "labels": []}
	]`)
	got := problems(t, `[{"number": 1, "action": "close"}]`, items)
	want := "actions[0] #1: item is 'CLOSED', not OPEN — re-assess before writing"
	if len(got) != 1 || got[0] != want {
		t.Errorf("Validate() = %v, want [%q]", got, want)
	}
}

func TestPayloadThatIsNotAList(t *testing.T) {
	for _, payload := range []string{`{"number": 1}`, `"label #1"`, `null`, `7`} {
		p := mustParse(t, payload)
		if p.IsList {
			t.Errorf("Parse(%s).IsList = true", payload)
			continue
		}
		got := Validate(p, testLive, nil)
		if len(got) != 1 || got[0] != "actions payload: expected a list" {
			t.Errorf("Parse(%s): Validate() = %v", payload, got)
		}
		if len(p.Entries) != 0 {
			t.Errorf("Parse(%s).Entries = %v, want none", payload, p.Entries)
		}
	}
}

func TestParseRejectsMalformedJSON(t *testing.T) {
	for _, payload := range []string{`[`, ``, `[{"number": 1},]`, `[] junk`} {
		if _, err := Parse([]byte(payload)); err == nil {
			t.Errorf("Parse(%q) = nil error", payload)
		}
	}
}

func TestParseLabels(t *testing.T) {
	got, err := ParseLabels([]byte(`["bug", {"name": "feature"}, null, "", {"nope": 1}]`))
	if err != nil {
		t.Fatalf("ParseLabels: %v", err)
	}
	if want := []string{"bug", "feature"}; !slices.Equal(got, want) {
		t.Errorf("ParseLabels() = %v, want %v", got, want)
	}
	if got, err := ParseLabels([]byte(`null`)); err != nil || len(got) != 0 {
		t.Errorf("ParseLabels(`null`) = %v, %v", got, err)
	}
	if _, err := ParseLabels([]byte(`[5]`)); err == nil {
		t.Error("ParseLabels([5]) = nil error")
	}
}

func TestReprString(t *testing.T) {
	tests := []struct{ in, want string }{
		{"frobnicate", `'frobnicate'`},
		{"", `''`},
		{"it's", `"it's"`},
		{`say "hi"`, `'say "hi"'`},
		{`both ' and "`, `'both \' and "'`},
		{"a\nb", `'a\nb'`},
		{"a\tb", `'a\tb'`},
		{`back\slash`, `'back\\slash'`},
		{"zero\u200bwidth", `'zero\u200bwidth'`},
		{"bell\x07", `'bell\x07'`},
		{"tag\U000E0041", `'tag\U000e0041'`},
		{"héllo", `'héllo'`},
	}
	for _, tc := range tests {
		if got := reprString(tc.in); got != tc.want {
			t.Errorf("reprString(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
	if got := repr(nil); got != "None" {
		t.Errorf("repr(nil) = %s, want None", got)
	}
}

func TestSelfTest(t *testing.T) {
	if fails := SelfTest(); len(fails) != 0 {
		t.Errorf("SelfTest() = %v, want none", fails)
	}
}

func read(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	return data
}

// labels.json is verbatim `gh label list --json name` output; gh-issue-list.json
// is verbatim `gh issue list --json id,labels,number,state,title,url` output,
// which is how the still-open recheck's state actually arrives.
func TestParseGHFixtures(t *testing.T) {
	live, err := ParseLabels(read(t, "labels.json"))
	if err != nil {
		t.Fatalf("ParseLabels: %v", err)
	}
	if !slices.Contains(live, "ceph-osd") || len(live) != 8 {
		t.Fatalf("ParseLabels() = %v", live)
	}

	items, err := ParseItems(read(t, "gh-issue-list.json"))
	if err != nil {
		t.Fatalf("ParseItems: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("ParseItems() returned %d items", len(items))
	}
	if got := names(items[0].Labels); !slices.Equal(got, []string{"bug", "ceph-osd"}) {
		t.Errorf("labels = %v", got)
	}
	if items[0].State == nil || *items[0].State != "OPEN" {
		t.Errorf("state = %v", items[0].State)
	}
	if items[0].Type != "" {
		t.Errorf("gh emits no type field, got %q", items[0].Type)
	}

	got := Validate(mustParse(t, `[
	  {"number": 15012, "type": "issue", "action": "label", "params": {"labels": ["docs"]}},
	  {"number": 14877, "type": "issue", "action": "close"}
	]`), live, items)
	want := []string{
		"actions[1] #14877: item is 'CLOSED', not OPEN — re-assess before writing",
	}
	if !slices.Equal(got, want) {
		t.Errorf("Validate() = %v, want %v", got, want)
	}
}

func TestFixtureRun(t *testing.T) {
	live, err := ParseLabels(read(t, "labels.json"))
	if err != nil {
		t.Fatalf("ParseLabels: %v", err)
	}
	items, err := ParseItems(read(t, "items.json"))
	if err != nil {
		t.Fatalf("ParseItems: %v", err)
	}
	payload, err := Parse(read(t, "actions.json"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []string{
		"actions[3] #103: item is 'CLOSED', not OPEN — re-assess before writing",
		"actions[4] #104: 6 labels after apply exceeds the cap of 5 " +
			"(bug, ceph-mon, core, docs, feature, object)",
		"actions[5] #102: label action on a PR — triage labels issues only",
		"actions[6] #999: no live state supplied for this item",
	}
	if got := Validate(payload, live, items); !slices.Equal(got, want) {
		t.Errorf("Validate() = %v, want %v", got, want)
	}
}

package mentions

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type harness struct {
	dir       string
	cachePath string
	opt       Options
}

func newHarness(t *testing.T, threads string, users map[string]string) *harness {
	t.Helper()
	dir := t.TempDir()
	if threads != "" {
		write(t, filepath.Join(dir, "threads.json"), threads)
	}
	var calls []string
	h := &harness{dir: dir, cachePath: filepath.Join(dir, "cache", "users.json")}
	h.opt = Options{
		SweepDir:  dir,
		Repo:      "rook/rook",
		CachePath: h.cachePath,
		Lookup:    stubLookup(users, &calls),
	}
	return h
}

func (h *harness) run(t *testing.T) string {
	t.Helper()
	var out strings.Builder
	if err := Run(context.Background(), h.opt, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return out.String()
}

func (h *harness) read(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(h.dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func threadsJSON(t *testing.T, threads map[string]*Thread, order []string) string {
	t.Helper()
	ts := &ThreadSet{}
	for _, k := range order {
		ts.Set(k, threads[k])
	}
	b, err := json.Marshal(ts)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRunMinesResolvesAndReports(t *testing.T) {
	fixture := threadsJSON(t, map[string]*Thread{
		"i7": {Number: 7, Body: "cc @Alice\n" + fence + "\n@ghost-in-code\n" + fence,
			Comments: Comments{Nodes: []Comment{
				{Body: "also @bob- and root@pod and " + tick + "@nope" + tick},
				{Body: "@alice again"},
			}}},
		"i12": {Number: 12, Body: "@nobody here"},
		"i9":  {Number: 9, Body: "no mentions at all"},
	}, []string{"i7", "i12", "i9"})

	h := newHarness(t, fixture, map[string]string{"alice": "Alice", "bob": "bob"})
	got := h.run(t)

	wantOut := "#7: +Alice +bob\n" +
		"issues w/ mentions: 1/3; unique logins: 2; unresolvable tokens ever seen: 1\n"
	if got != wantOut {
		t.Errorf("stdout = %q, want %q", got, wantOut)
	}
	wantFile := "{\n \"7\": [\n  \"Alice\",\n  \"bob\"\n ]\n}"
	if f := h.read(t, "issues-mentions.json"); f != wantFile {
		t.Errorf("issues-mentions.json = %q, want %q", f, wantFile)
	}
	wantCache := "{\n \"alice\": \"Alice\",\n \"bob-\": \"bob\",\n \"nobody\": null\n}"
	b, err := os.ReadFile(h.cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != wantCache {
		t.Errorf("cache = %q, want %q", b, wantCache)
	}
}

func TestRunDiffsAgainstThePreviousVersion(t *testing.T) {
	fixture := threadsJSON(t, map[string]*Thread{
		"i9":   {Number: 9, Body: "@alice @bob"},
		"i100": {Number: 100, Body: "@alice"},
	}, []string{"i9", "i100"})

	h := newHarness(t, fixture, map[string]string{"alice": "Alice", "bob": "bob"})
	write(t, filepath.Join(h.dir, "issues-mentions.json"),
		`{"9": ["Alice", "carol"], "100": ["Alice"], "5": ["dave"]}`)

	got := h.run(t)
	want := "#5: -dave\n" +
		"#9: -carol +bob\n" +
		"issues w/ mentions: 2/2; unique logins: 2; unresolvable tokens ever seen: 0\n"
	if got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRunIsQuietWhenNothingChanged(t *testing.T) {
	fixture := threadsJSON(t, map[string]*Thread{
		"i9": {Number: 9, Body: "@alice"},
	}, []string{"i9"})

	h := newHarness(t, fixture, map[string]string{"alice": "Alice"})
	h.run(t)
	got := h.run(t)
	want := "issues w/ mentions: 1/1; unique logins: 1; unresolvable tokens ever seen: 0\n"
	if got != want {
		t.Errorf("second run printed %q, want %q", got, want)
	}
}

func TestRunEmptyResult(t *testing.T) {
	h := newHarness(t, threadsJSON(t, map[string]*Thread{
		"i9": {Number: 9, Body: "nothing"},
	}, []string{"i9"}), nil)

	got := h.run(t)
	want := "issues w/ mentions: 0/1; unique logins: 0; unresolvable tokens ever seen: 0\n"
	if got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if f := h.read(t, "issues-mentions.json"); f != "{}" {
		t.Errorf("issues-mentions.json = %q, want {}", f)
	}
}

func TestRunCountsNullThreadsButSkipsThem(t *testing.T) {
	h := newHarness(t, `{"i9": {"number": 9, "body": "@alice", "comments": {"nodes": []}}, "i8": null}`,
		map[string]string{"alice": "Alice"})

	got := h.run(t)
	want := "#9: +Alice\n" +
		"issues w/ mentions: 1/2; unique logins: 1; unresolvable tokens ever seen: 0\n"
	if got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRunRefusesToFetchWithoutNumbers(t *testing.T) {
	h := newHarness(t, "", nil)
	err := Run(context.Background(), h.opt, &strings.Builder{})
	if err == nil || !strings.Contains(err.Error(), "threads.json missing") {
		t.Fatalf("err = %v, want a threads.json-missing error", err)
	}
}

func TestIssueNumbersAreSortedAndDeduped(t *testing.T) {
	got, err := issueNumbers(Options{Numbers: "12, 3,3,\n,100"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{3, 12, 100}; !slices.Equal(got, want) {
		t.Errorf("issueNumbers() = %v, want %v", got, want)
	}

	path := filepath.Join(t.TempDir(), "numbers")
	write(t, path, "7\n\n7\r\n2\n")
	got, err = issueNumbers(Options{NumbersFile: path})
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{2, 7}; !slices.Equal(got, want) {
		t.Errorf("issueNumbers(file) = %v, want %v", got, want)
	}
}

func TestThreadSetPreservesDocumentOrder(t *testing.T) {
	raw := `{"i100": {"number": 100, "body": "b", "comments": {"totalCount": 1, ` +
		`"pageInfo": {"hasNextPage": false, "endCursor": "x"}, "nodes": [{"body": "c"}]}}, ` +
		`"i9": {"number": 9, "body": "", "comments": {"nodes": []}}}`
	var ts ThreadSet
	if err := json.Unmarshal([]byte(raw), &ts); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ts.Keys, []string{"i100", "i9"}) {
		t.Errorf("keys = %v, want document order", ts.Keys)
	}
	if docs := ts.Threads["i100"].Docs(); !slices.Equal(docs, []string{"b", "c"}) {
		t.Errorf("Docs() = %v, want body then comments", docs)
	}
	out, err := json.Marshal(&ts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(out), `{"i100":`) {
		t.Errorf("re-marshalled as %s, want i100 first", out)
	}
}

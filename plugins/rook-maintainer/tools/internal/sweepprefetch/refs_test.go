package sweepprefetch

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func sweepWithBatches(t *testing.T, seed bool) string {
	t.Helper()
	dir := t.TempDir()
	files := []struct{ from, to string }{
		{"refs/batch-01.json", "batch-01.json"},
		{"refs/batch-02.json", "batch-02.json"},
	}
	if seed {
		files = append(files, struct{ from, to string }{"refs/refs-types.seed.json", "refs-types.json"})
	}
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join("testdata", f.from))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, f.to), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestRefNumbers(t *testing.T) {
	got, err := refNumbers(sweepWithBatches(t, false))
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{4, 7, 30}; !reflect.DeepEqual(got, want) {
		t.Errorf("refNumbers() = %v, want %v", got, want)
	}
}

func TestRefNumbersWithoutBatches(t *testing.T) {
	got, err := refNumbers(t.TempDir())
	if err != nil || len(got) != 0 {
		t.Errorf("refNumbers() = %v, %v, want no numbers and no error", got, err)
	}
}

func TestClassifyRefsMergesIntoExistingTypes(t *testing.T) {
	dir := sweepWithBatches(t, true)
	stub := newStub(t, "refs-classified.json")
	res, err := testClient(t, stub).ClassifyRefs(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	stub.done()

	got, err := os.ReadFile(filepath.Join(dir, "refs-types.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := golden(t, "refs-types.golden.json"); string(got) != want {
		t.Errorf("refs-types.json =\n%s\nwant\n%s", got, want)
	}
	if want := "3 refs, 2 newly classified -> " + res.Path; res.String() != want {
		t.Errorf("String() = %q, want %q", res.String(), want)
	}
	// Only the unclassified refs are queried; 30 came from the seed file.
	q := stub.queries[0]
	if !strings.Contains(q, "n4: issueOrPullRequest(number: 4)") ||
		!strings.Contains(q, "n7: issueOrPullRequest(number: 7)") ||
		strings.Contains(q, "n30:") {
		t.Errorf("unexpected query:\n%s", q)
	}
}

func TestClassifyRefsSkipsTheQueryWhenNothingIsNew(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "refs-types.json"), []byte(`{"9": "Issue"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	stub := &stubGQL{t: t}
	res, err := testClient(t, stub).ClassifyRefs(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(stub.queries) != 0 {
		t.Errorf("sent %d queries, want none", len(stub.queries))
	}
	if res.Refs != 0 || res.New != 0 {
		t.Errorf("result = %+v, want no refs", res)
	}
	got, err := os.ReadFile(filepath.Join(dir, "refs-types.json"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "{\n \"9\": \"Issue\"\n}"; string(got) != want {
		t.Errorf("refs-types.json = %q, want %q", got, want)
	}
}

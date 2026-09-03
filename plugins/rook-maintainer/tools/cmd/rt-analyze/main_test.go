package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	fixturePRs = `{"number":1,"title":"misc","mergedAt":"2026-07-01T00:00:00Z",` +
		`"author":{"login":"alice"},"files":{"nodes":[{"path":"nowhere/thing.txt"}]},` +
		`"reviews":{"nodes":[{"author":{"login":"mallory"}}]}}`
	fixtureState = `{"pages_fetched":1,"counted":1,"oldest_mergedat":"2026-07-01T00:00:00Z",` +
		`"stop_reason":"reached the window cutoff","errors":[],"truncations":[]}`
)

// analyze runs the analysis form over a one-PR fixture that flags, with stderr
// captured: what does and does not reach it is the point of --brief.
func analyze(t *testing.T, args ...string) (int, string) {
	t.Helper()
	dir := t.TempDir()
	for name, body := range map[string]string{"rt_prs.jsonl": fixturePRs, "rt_fetch_state.json": fixtureState} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	captured := filepath.Join(dir, "stderr")
	f, err := os.Create(captured)
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stderr
	os.Stderr = f
	code := dispatch(append([]string{
		"--in-dir", dir, "--roster", "alice",
		"--out", filepath.Join(dir, "rt_final.json"),
		"--now", "2026-08-01T00:00:00Z",
	}, args...), strings.NewReader(""), io.Discard)
	os.Stderr = saved
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	errText, err := os.ReadFile(captured)
	if err != nil {
		t.Fatal(err)
	}
	return code, string(errText)
}

func TestBriefWritesTheFencedFlagsAndStderrCarriesNone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "brief.txt")
	code, errText := analyze(t, "--brief", path)
	if code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, errText)
	}
	brief, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(brief)
	open := strings.Index(got, "<<<UNTRUSTED-")
	if open < 0 || strings.Count(got, "<<<UNTRUSTED-") != 1 {
		t.Fatalf("want exactly one opening marker:\n%s", got)
	}
	token := got[open+len("<<<UNTRUSTED-") : open+len("<<<UNTRUSTED-")+strings.Index(got[open+len("<<<UNTRUSTED-"):], "\n")]
	if !strings.HasSuffix(got, "\n"+token+"-UNTRUSTED>>>\n") {
		t.Errorf("the closing marker does not carry token %q:\n%s", token, got)
	}
	if !strings.Contains(got, "[bucket-ambiguity]") {
		t.Errorf("the brief carries no flags:\n%s", got)
	}
	if strings.Contains(errText, "UNTRUSTED") || strings.Contains(errText, "question:") {
		t.Errorf("the brief also reached stderr:\n%s", errText)
	}
}

func TestNoBriefFlagEmitsNoBrief(t *testing.T) {
	code, errText := analyze(t)
	if code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, errText)
	}
	if strings.Contains(errText, "UNTRUSTED") || strings.Contains(errText, "question:") {
		t.Errorf("a run without --brief still emitted one:\n%s", errText)
	}
	if !strings.Contains(errText, "PRs=1") {
		t.Errorf("the run summary must still reach stderr:\n%s", errText)
	}
}

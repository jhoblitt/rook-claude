package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	fixtureIssues = `[{"number":1,"author":{"login":"alice"},"createdAt":"2025-01-01T00:00:00Z",` +
		`"labels":[{"name":"csi"}],` +
		`"comments":[{"author":{"login":"mallory"},"createdAt":"2026-07-01T00:00:00Z"}]}]`
	fixtureLabelMap = "| Paths touched | Area | Issue label |\n|---|---|---|\n" +
		"| `pkg/operator/ceph/csi/**` | `csi` | `csi` |\n"
)

// mine runs the tool over a one-issue fixture that flags, with both streams
// captured: what does and does not reach stderr is the point of --brief.
func mine(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	dir := t.TempDir()
	for name, body := range map[string]string{"issues.json": fixtureIssues, "label-map.md": fixtureLabelMap} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	outFile, errFile := capture(t, dir)
	code = run(append([]string{
		"--in", filepath.Join(dir, "issues.json"),
		"--label-map", filepath.Join(dir, "label-map.md"),
		"--out", filepath.Join(dir, "rt_issues_final.json"),
		"--roster", "alice",
		"--now", "2026-08-01T00:00:00Z",
	}, args...))
	return code, outFile(), errFile()
}

func capture(t *testing.T, dir string) (stdout, stderr func() string) {
	t.Helper()
	redirect := func(name string, stream **os.File) func() string {
		path := filepath.Join(dir, name)
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		saved := *stream
		*stream = f
		return func() string {
			*stream = saved
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
			text, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			return string(text)
		}
	}
	return redirect("stdout", &os.Stdout), redirect("stderr", &os.Stderr)
}

func TestBriefWritesTheFencedFlagsAndStderrCarriesNone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "brief.txt")
	code, stdout, stderr := mine(t, "--brief", path)
	if code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr is not empty on a successful run:\n%s", stderr)
	}
	if !strings.HasPrefix(stdout, "issues=1 ") {
		t.Errorf("stdout is not the run summary:\n%s", stdout)
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
	if !strings.Contains(got, "[identity-unknown] mallory") {
		t.Errorf("the brief carries no flags:\n%s", got)
	}
}

func TestNoBriefFlagEmitsNoBrief(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "brief.txt")
	if code, _, stderr := mine(t); code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, stderr)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("a run without --brief wrote %s (err=%v)", path, err)
	}
}

func TestMissingRequiredArgumentIsAUsageError(t *testing.T) {
	dir := t.TempDir()
	stdout, stderr := capture(t, dir)
	code := run([]string{"--in", filepath.Join(dir, "issues.json")})
	stdout()
	text := stderr()
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(text, "--label-map") {
		t.Errorf("stderr does not name the missing argument:\n%s", text)
	}
}

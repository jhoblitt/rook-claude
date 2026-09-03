package rtcommits

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

const baseCommand = "git -c core.quotePath=false log --no-merges -M " +
	"--format=commit%x09%H%x09%aN%x09%aE%x09%aI --name-status --end-of-options"

func TestGitLogCommandIsWhatTheDocsPromise(t *testing.T) {
	if got := GitLogCommand(""); got != baseCommand {
		t.Errorf("GitLogCommand(\"\") = %q\nwant %q\n(the docs, the fixture and the --log contract are pinned to this)",
			got, baseCommand)
	}
	want := baseCommand + " " + DefaultRef
	if got := GitLogCommand(DefaultRef); got != want {
		t.Errorf("GitLogCommand(%q) = %q\nwant %q", DefaultRef, got, want)
	}
}

// The revision goes last and nowhere else: git log reads a leading one as a
// path, and a mine of the wrong revision is indistinguishable from a right one.
func TestGitLogArgsForAppendsTheRef(t *testing.T) {
	got := gitLogArgsFor("v1.18.0")
	if len(got) != len(gitLogArgs)+1 || got[len(got)-1] != "v1.18.0" {
		t.Fatalf("gitLogArgsFor = %q, want the base args plus a trailing revision", got)
	}
	if last := gitLogArgs[len(gitLogArgs)-1]; last != "--end-of-options" {
		t.Errorf("gitLogArgsFor mutated the shared base args: they now end in %q", last)
	}
	if got := gitLogArgsFor(""); len(got) != len(gitLogArgs) {
		t.Errorf("gitLogArgsFor(\"\") = %q, want the ref-less base args", got)
	}
}

func TestParseLogFixture(t *testing.T) {
	commits := fixtureCommits(t)
	if len(commits) != 20 {
		t.Fatalf("parsed %d commits, want 20", len(commits))
	}

	first := commits[0]
	if first.SHA != "a000001" || first.Name != "Alice Example" || first.Email != "alice@example.com" {
		t.Errorf("first commit = %+v", first)
	}
	if got := first.When.UTC().Format("2006-01-02T15:04:05Z"); got != "2026-08-01T12:00:00Z" {
		t.Errorf("first commit date = %s", got)
	}

	byShA := map[string]Commit{}
	for _, c := range commits {
		byShA[c.SHA] = c
	}

	// The rename contributes both paths, the space survives, and the C-quoted
	// path comes back decoded.
	want := []string{
		"pkg/operator/ceph/object/zone.go",
		"pkg/operator/ceph/nfs/zone.go",
		"Documentation/Getting-Started/Rook High-Level Architecture.png",
		"Documentation/café \"quoted\"\ttab.md",
	}
	if got := byShA["a000016"].Paths; strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("rename commit paths = %q, want %q", got, want)
	}
	if got := byShA["a000019"].Paths; len(got) != 2 {
		t.Errorf("copy commit paths = %q, want both sides", got)
	}
	if got := byShA["a000015"].Paths; len(got) != 0 {
		t.Errorf("empty commit paths = %q, want none", got)
	}
	if got := byShA["a000002"].When.UTC().Format("2006-01-02T15:04:05Z"); got != "2026-02-10T00:00:00Z" {
		t.Errorf("non-UTC author date normalised to %s", got)
	}
}

// Every rejection here is a case where the alternative is a mine that counts a
// fraction of the history and reports it as the whole.
func TestParseLogRejectsMalformedInput(t *testing.T) {
	const header = "commit\ta000001\tAlice\talice@example.com\t2026-08-01T12:00:00+00:00\n\n"
	tests := []struct{ name, in, want string }{
		{"short header", "commit\ta000001\tAlice\t2026-08-01T12:00:00+00:00\n", "want 5"},
		{"tab in the author name", "commit\ta1\tAl\tice\ta@b.c\t2026-08-01T12:00:00+00:00\n", "want 5"},
		{"no sha", "commit\t\tAlice\ta@b.c\t2026-08-01T12:00:00+00:00\n", "no sha"},
		{"no author at all", "commit\ta1\t\t\t2026-08-01T12:00:00+00:00\n", "neither an author name nor an email"},
		{"unparsable date", "commit\ta1\tAlice\ta@b.c\tlast tuesday\n", "invalid isoformat"},
		{"naive date", "commit\ta1\tAlice\ta@b.c\t2026-08-01T12:00:00\n", "invalid isoformat"},
		{"path before any header", "M\tpkg/operator/ceph/object/rgw.go\n", "precedes any commit header"},
		{"--name-only dump", header + "pkg/operator/ceph/object/rgw.go\n", "expected \"<status>"},
		{"unknown status", header + "Z\tpkg/operator/ceph/object/rgw.go\n", "expected \"<status>"},
		{"rename missing its destination", header + "R100\tpkg/a.go\n", "want 3"},
		{"modify with a rename's field count", header + "M\tpkg/a.go\tpkg/b.go\n", "want 2"},
		{"empty path", header + "M\t\n", "empty path"},
		{"unterminated quote", header + "M\t\"pkg/a.go\n", "unterminated quoted path"},
		{"bad octal escape", header + "M\t\"pkg/\\999a.go\"\n", "bad escape"},
		{"trailing backslash", header + "M\t\"pkg/a.go\\\"\n", "ends in a backslash"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseLog(strings.NewReader(tc.in))
			if err == nil {
				t.Fatalf("accepted %q", tc.in)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestParseLogReportsTheOffendingLine(t *testing.T) {
	in := "commit\ta1\tAlice\ta@b.c\t2026-08-01T12:00:00+00:00\n\nM\tpkg/a.go\nZ\tpkg/b.go\n"
	_, err := ParseLog(strings.NewReader(in))
	if err == nil || !strings.Contains(err.Error(), "line 4") {
		t.Errorf("error = %v, want it to name line 4", err)
	}
}

func TestUnquotePath(t *testing.T) {
	tests := []struct{ in, want string }{
		{"pkg/a.go", "pkg/a.go"},
		{"deploy/examples/Ceph Cluster Dashboard.json", "deploy/examples/Ceph Cluster Dashboard.json"},
		{`"pkg/a.go"`, "pkg/a.go"},
		{`"pkg/caf\303\251.go"`, "pkg/café.go"},
		{`"pkg/a\tb.go"`, "pkg/a\tb.go"},
		{`"pkg/a\nb.go"`, "pkg/a\nb.go"},
		{`"pkg/\"q\".go"`, `pkg/"q".go`},
		{`"pkg/back\\slash.go"`, `pkg/back\slash.go`},
	}
	for _, tc := range tests {
		got, err := unquotePath(tc.in)
		if err != nil {
			t.Errorf("unquotePath(%s): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("unquotePath(%s) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLogRejectsANonRepo(t *testing.T) {
	if _, _, err := Log(t.Context(), t.TempDir(), DefaultRef); err == nil {
		t.Fatal("Log accepted a directory that is not a git repository")
	}
}

// staleClone is the shape the whole --ref flag exists for: a checkout whose
// HEAD sits one commit behind the origin/master it was cloned from.
func staleClone(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	env := append(os.Environ(),
		"HOME="+t.TempDir(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_TERMINAL_PROMPT=0",
	)
	git := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir, cmd.Env = dir, env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	git("init", "-q", "-b", "master", ".")
	git("config", "user.email", "test@example.com")
	git("config", "user.name", "Test")
	write := func(name string) {
		t.Helper()
		if err := os.WriteFile(dir+"/"+name, []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
		git("add", name)
	}
	write("behind.go")
	git("commit", "-q", "-m", "behind")
	write("ahead.go")
	git("commit", "-q", "-m", "ahead")
	git("update-ref", "refs/remotes/origin/master", "HEAD")
	git("reset", "-q", "--hard", "HEAD~1")
	return dir
}

func TestLogMinesTheRefAndNotHEAD(t *testing.T) {
	dir := staleClone(t)

	commits, head, err := Log(t.Context(), dir, DefaultRef)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 2 {
		t.Fatalf("mining %s found %d commits, want both (HEAD alone has 1)", DefaultRef, len(commits))
	}
	if commits[0].SHA != head {
		t.Errorf("head = %s, want the sha %s resolves to (%s)", head, DefaultRef, commits[0].SHA)
	}

	behind, _, err := Log(t.Context(), dir, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(behind) != 1 {
		t.Fatalf("mining HEAD found %d commits, want 1: the fixture is not stale", len(behind))
	}
}

func TestLogRejectsAnUnresolvableRef(t *testing.T) {
	_, _, err := Log(t.Context(), staleClone(t), "origin/no-such-branch")
	if err == nil {
		t.Fatal("Log accepted a ref the repo does not have")
	}
	if !strings.Contains(err.Error(), "--ref origin/no-such-branch") {
		t.Errorf("error = %q, want it to name the unresolvable --ref", err)
	}
}

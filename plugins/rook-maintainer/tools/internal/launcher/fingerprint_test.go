package launcher

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A two-tool module whose commands share nothing: alpha imports internal/a and
// beta imports internal/b, so an edit under one is invisible to the other's
// package set and visible to its own.
var fakeTree = map[string]string{
	"go.mod":            "module example.test\n\ngo 1.24\n",
	"cmd/alpha/main.go": "package main\n\nimport \"example.test/internal/a\"\n\nvar _ = a.S\n\nfunc main() {}\n",
	"cmd/beta/main.go":  "package main\n\nimport \"example.test/internal/b\"\n\nvar _ = b.S\n\nfunc main() {}\n",
	"internal/a/a.go":   "package a\n\nconst S = \"a1\"\n",
	"internal/b/b.go":   "package b\n\nconst S = \"b1\"\n",
}

// fakePlugin lays the tree out under a CLAUDE_PLUGIN_ROOT and returns the
// sandbox root, a writer for one of its files, and a launcher.
func fakePlugin(t *testing.T) (write func(rel, body string), launch func(tool string, path ...string) (os.FileInfo, string, error)) {
	t.Helper()
	root := t.TempDir()
	src := filepath.Join(root, "root", "tools")
	write = func(rel, body string) {
		t.Helper()
		p := filepath.Join(src, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for rel, body := range fakeTree {
		write(rel, body)
	}

	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Fatal(err)
	}
	launch = func(tool string, path ...string) (os.FileInfo, string, error) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), bash, runScript, tool)
		cmd.Env = append(os.Environ(),
			"HOME="+root,
			"XDG_CACHE_HOME="+filepath.Join(root, "xdg"),
			"CLAUDE_PLUGIN_ROOT="+filepath.Join(root, "root"),
			"CLAUDE_PLUGIN_DATA=",
		)
		if len(path) == 1 {
			cmd.Env = append(cmd.Env, "PATH="+path[0])
		}
		out, err := cmd.CombinedOutput()
		fi, serr := os.Stat(filepath.Join(root, "xdg", "rook-claude", "tools", tool))
		if err == nil && serr != nil {
			t.Fatal(serr)
		}
		return fi, string(out), err
	}
	return write, launch
}

// The launcher rebuilt every tool whenever any .go file under tools/ changed,
// so adding or touching one tool cost a rebuild of all the others on their next
// use. Inode identity is the tell: a rebuild lands a fresh file through mv.
func TestFingerprintCoversOneToolsPackagesOnly(t *testing.T) {
	write, run := fakePlugin(t)
	launch := func(tool string) os.FileInfo {
		t.Helper()
		fi, out, err := run(tool)
		if err != nil {
			t.Fatalf("run.sh %s: %v\n%s", tool, err, out)
		}
		return fi
	}

	built := launch("alpha")
	launch("beta")

	write("internal/b/b.go", "package b\n\nconst S = \"b2\"\n")
	if after := launch("alpha"); !os.SameFile(built, after) {
		t.Error("editing beta's package rebuilt alpha")
	}

	write("internal/a/a.go", "package a\n\nconst S = \"a2\"\n")
	if after := launch("alpha"); os.SameFile(built, after) {
		t.Error("editing alpha's own package did not rebuild it")
	}
}

// toolPath is a PATH holding everything run.sh shells out to except a Go
// toolchain, which is how an installed plugin looks on a machine without Go.
func toolPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"find", "sort", "xargs", "cat", "cksum", "mkdir", "mv", "rm",
		"basename", "flock"} {
		bin, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		if err := os.Symlink(bin, filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// A cached binary must still run where the fingerprint cannot be taken. The
// per-tool sum needs go list, so without a toolchain staleness falls back to
// the mtime test; a whole-tree sum there would never match the stamp, and every
// tool would die on "cannot build" over a binary that was already correct.
func TestCachedBinaryRunsWithoutAToolchain(t *testing.T) {
	_, run := fakePlugin(t)
	built, out, err := run("alpha")
	if err != nil {
		t.Fatalf("run.sh alpha: %v\n%s", err, out)
	}

	path := toolPath(t)
	after, out, err := run("alpha", path)
	if err != nil {
		t.Fatalf("run.sh alpha without a toolchain: %v\n%s", err, out)
	}
	if !os.SameFile(built, after) {
		t.Error("the cached binary was replaced by a run that had no toolchain")
	}

	// The same PATH must still fail loudly for a tool with nothing cached,
	// which is also what proves go is really out of reach above.
	if _, out, err := run("beta", path); err == nil {
		t.Errorf("run.sh built beta without a toolchain:\n%s", out)
	} else if !strings.Contains(out, "go toolchain not found") {
		t.Errorf("run.sh did not name the missing toolchain:\n%s", out)
	}
}

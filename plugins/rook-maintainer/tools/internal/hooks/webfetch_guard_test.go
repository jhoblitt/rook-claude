package hooks

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const (
	webfetchGuard = "../../../hooks/webfetch-guard.sh"
	guardManifest = "../../../.claude-plugin/plugin.json"
)

func guardPluginName(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(guardManifest)
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m.Name
}

// guardCacheDir reports which directory the hook picked, by giving it a source
// tree that cannot build: the build-failure marker lands next to the binary,
// which is the only externally visible statement of the choice — the hook fails
// OPEN, so its notice deliberately names no path.
func guardCacheDir(t *testing.T, dataBasename string) string {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "root", "hooks", "webfetch-guard")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "guard.go"), []byte("this is not go\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	pluginData := ""
	if dataBasename != "" {
		pluginData = filepath.Join(dir, dataBasename)
		if err := os.MkdirAll(pluginData, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// HOME stays the caller's: XDG_CACHE_HOME already covers the fallback, and
	// the failing `go build` would otherwise scribble a build cache into the
	// directory this test is about to check and remove.
	cmd := exec.CommandContext(t.Context(), "bash", webfetchGuard)
	cmd.Env = append(os.Environ(),
		"XDG_CACHE_HOME="+filepath.Join(dir, "xdg"),
		"CLAUDE_PLUGIN_ROOT="+filepath.Join(dir, "root"),
		"CLAUDE_PLUGIN_DATA="+pluginData,
	)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("the hook must fail open: %v\n%s", err, out)
	}
	if len(out) == 0 {
		t.Fatal("the hook built a source tree that is not Go")
	}

	var found string
	for _, candidate := range []string{filepath.Join(dir, "xdg", "rook-claude"), pluginData} {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(candidate, "webfetch-guard.buildfail")); err == nil {
			if found != "" {
				t.Fatalf("both %s and %s hold a build marker", found, candidate)
			}
			found = candidate
		}
	}
	if found == "" {
		t.Fatalf("no cache directory holds a build marker:\n%s", out)
	}
	return found
}

// The launcher and this hook are siblings over the same directory; a guard
// applied to only one of them is the drift that matters.
func TestGuardIgnoresForeignPluginData(t *testing.T) {
	name := guardPluginName(t)
	for _, foreign := range []string{"", "codex-openai-codex", name + "2"} {
		t.Run("basename "+foreign, func(t *testing.T) {
			dir := guardCacheDir(t, foreign)
			if filepath.Base(dir) != "rook-claude" {
				t.Errorf("CLAUDE_PLUGIN_DATA basename %q selected %q, want the fallback", foreign, dir)
			}
		})
	}
}

func TestGuardHonorsThisPluginsData(t *testing.T) {
	name := guardPluginName(t)
	for _, base := range []string{name, name + "-rook-claude"} {
		t.Run(base, func(t *testing.T) {
			if dir := guardCacheDir(t, base); filepath.Base(dir) != base {
				t.Errorf("CLAUDE_PLUGIN_DATA basename %q selected %q", base, dir)
			}
		})
	}
}

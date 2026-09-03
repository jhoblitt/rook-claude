// Package launcher tests tools/run.sh, the build-on-first-use launcher every
// skill invokes; it has no Go source of its own.
package launcher

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	runScript = "../../run.sh"
	manifest  = "../../../.claude-plugin/plugin.json"
	dieLine   = "rook-maintainer tools: cannot create "
)

// pluginName is the name the harness builds CLAUDE_PLUGIN_DATA's basename from,
// and the constant run.sh matches it against.
func pluginName(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m.Name == "" {
		t.Fatalf("%s has no name", manifest)
	}
	return m.Name
}

// chosenCacheDir reports which directory run.sh picked, by making the pick
// uncreatable: a regular file where the directory would go turns "mkdir -p" into
// the launcher's own "cannot create <dir>" failure, before it builds anything.
// It returns the sandbox root the answer is relative to.
func chosenCacheDir(t *testing.T, dataBasename string) (root, chosen string) {
	t.Helper()
	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "root", "tools", "cmd", "rt-commits"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "xdg"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	pluginData := ""
	if dataBasename != "" {
		pluginData = filepath.Join(root, dataBasename)
		if err := os.WriteFile(pluginData, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.CommandContext(t.Context(), "bash", runScript, "rt-commits")
	cmd.Env = append(os.Environ(),
		"HOME="+root,
		"XDG_CACHE_HOME="+filepath.Join(root, "xdg"),
		"CLAUDE_PLUGIN_ROOT="+filepath.Join(root, "root"),
		"CLAUDE_PLUGIN_DATA="+pluginData,
	)

	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("run.sh succeeded with every cache directory blocked:\n%s", out)
	}
	i := strings.LastIndex(string(out), dieLine)
	if i < 0 {
		t.Fatalf("run.sh failed without naming a cache directory:\n%s", out)
	}
	return root, strings.TrimSpace(string(out)[i+len(dieLine):])
}

// The observed failure: another plugin leaks its CLAUDE_PLUGIN_DATA into the
// session the model runs run.sh from, and every tool dies on a read-only path
// under that plugin.
func TestForeignPluginDataIsIgnored(t *testing.T) {
	name := pluginName(t)
	for _, foreign := range []string{"", "codex-openai-codex", "data", name + "2"} {
		t.Run("basename "+foreign, func(t *testing.T) {
			root, got := chosenCacheDir(t, foreign)
			want := filepath.Join(root, "xdg", "rook-claude", "tools")
			if got != want {
				t.Errorf("CLAUDE_PLUGIN_DATA basename %q selected %q, want the fallback %q", foreign, got, want)
			}
		})
	}
}

func TestThisPluginsDataIsHonored(t *testing.T) {
	name := pluginName(t)
	for _, base := range []string{name, name + "-rook-claude"} {
		t.Run(base, func(t *testing.T) {
			root, got := chosenCacheDir(t, base)
			want := filepath.Join(root, base, "tools")
			if got != want {
				t.Errorf("CLAUDE_PLUGIN_DATA basename %q selected %q, want %q", base, got, want)
			}
		})
	}
}

package refs

import (
	"os"
	"path/filepath"
	"testing"
)

// diffOf wraps body lines in the smallest valid unified diff for one file.
func diffOf(file string, body string) string {
	return "diff --git a/" + file + " b/" + file + "\n" +
		"--- a/" + file + "\n+++ b/" + file + "\n@@ -1,1 +1,9 @@\n" + body
}

func TestExtractMakeTargets(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{"code span", "+run `make lint.commits` first\n", []string{"lint.commits"}},
		{"fenced block", "+```console\n+make crds\n+```\n", []string{"crds"}},
		{"table cell", "+| Docs | `make markdownlint` | any |\n", []string{"markdownlint"}},
		{"trailing variable", "+`make lint.commits COMMITLINT_BASE=origin/master`\n", []string{"lint.commits"}},
		{"flag with argument", "+`make -C build codegen`\n", []string{"codegen"}},
		{"variable target", "+`make $(TOOL)`\n", []string{"$(TOOL)"}},

		// Prose must never manufacture a target.
		{"prose make target", "+add a make target to check messages\n", nil},
		{"prose make sure", "+make sure the cluster is up\n", nil},
		{"conflict marker", "+>>>>>>> abc (build: add a make target)\n", nil},

		// Removed lines are not this change's problem.
		{"removed line", "-run `make gone` first\n", nil},

		// Leading variable assignment means make builds the default goal.
		{"assignment only", "+`make CGO_ENABLED=0`\n", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, r := range Extract(diffOf("doc.md", tc.body)) {
				if r.Kind == KindMake {
					got = append(got, r.Name)
				}
			}
			assertNames(t, got, tc.want)
		})
	}
}

func TestExtractPaths(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{"markdown link", "+see [guide](Documentation/Contributing/dev.md) now\n",
			[]string{"Documentation/Contributing/dev.md"}},
		{"dot directory", "+see .github/workflows/ci.yml\n", []string{".github/workflows/ci.yml"}},
		{"trailing slash", "+tests under tests/framework/ are compiled\n", []string{"tests/framework/"}},
		{"anchor stripped", "+[a](Documentation/dev.md#commit-structure)\n", []string{"Documentation/dev.md"}},

		// Not paths.
		{"inside a URL", "+https://github.com/rook/rook/blob/master/x.md\n", nil},
		{"namespaced identifier", "+import k8s.io/api/core/v1 here\n", nil},
		{"prose slash", "+either and/or neither\n", nil},
		{"git ref with version", "+compares against origin/release-1.20 by default\n", nil},
		{"git ref plain", "+rebase onto upstream/master often\n", nil},
		{"absolute container path", "+mounts /workspace/tests/scripts/x.sh inside\n", nil},
		{"no extension", "+the pkg/apis tree changed\n", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, r := range Extract(diffOf("doc.md", tc.body)) {
				if r.Kind == KindPath {
					got = append(got, r.Name)
				}
			}
			assertNames(t, got, tc.want)
		})
	}
}

func TestExtractRecordsPosition(t *testing.T) {
	diff := "--- a/doc.md\n+++ b/doc.md\n@@ -10,2 +10,3 @@\n" +
		" context line\n+run `make gone` now\n"
	got := Extract(diff)
	if len(got) != 1 {
		t.Fatalf("got %d references, want 1", len(got))
	}
	if got[0].File != "doc.md" || got[0].Line != 11 {
		t.Errorf("got %s:%d, want doc.md:11", got[0].File, got[0].Line)
	}
}

func TestTargetsParsesIncludesAndPhony(t *testing.T) {
	root := writeTree(t, map[string]string{
		"Makefile": "MARKDOWNLINT := docker run x\n" +
			".PHONY: markdownlint\n" +
			"markdownlint: ## check\n\t@$(MARKDOWNLINT)\n" +
			"include build/makelib/*.mk\n",
		"build/makelib/common.mk": ".PHONY: codegen\ncodegen:\n\t@echo hi\n",
	})

	ts, err := Targets(root)
	if err != nil {
		t.Fatalf("Targets: %v", err)
	}
	for _, want := range []string{"markdownlint", "codegen"} {
		if !ts.Names[want] {
			t.Errorf("target %q not found; got %v", want, keys(ts.Names))
		}
	}
	// An assignment must not be read as a target.
	if ts.Names["MARKDOWNLINT"] {
		t.Error("variable assignment was read as a target")
	}
}

// A rule may name a variable the makefile assigns further down; make does not
// care about order, so neither can this parse.
func TestTargetsExpandsForwardReference(t *testing.T) {
	root := writeTree(t, map[string]string{
		"Makefile": "$(LATER):\n\t@echo hi\n\nLATER := built/thing\n",
	})
	ts, err := Targets(root)
	if err != nil {
		t.Fatalf("Targets: %v", err)
	}
	if !ts.Names["built/thing"] {
		t.Errorf("forward-referenced target not resolved; opaque=%v names=%v",
			ts.OpaqueFile, keys(ts.Names))
	}
}

func TestTargetsSplitsOpaqueByKind(t *testing.T) {
	root := writeTree(t, map[string]string{
		"Makefile": "$(TOOL_BINARIES):\n\ttouch $@\n.PHONY: $(SUITES)\n",
	})
	ts, err := Targets(root)
	if err != nil {
		t.Fatalf("Targets: %v", err)
	}
	if len(ts.OpaqueFile) != 1 {
		t.Errorf("OpaqueFile = %v, want one entry", ts.OpaqueFile)
	}
	if len(ts.OpaquePhony) != 1 {
		t.Errorf("OpaquePhony = %v, want one entry", ts.OpaquePhony)
	}
}

func TestResolveVerdicts(t *testing.T) {
	root := writeTree(t, map[string]string{
		"Makefile":        ".PHONY: build\nbuild:\n\t@echo hi\n",
		"scripts/here.sh": "#!/bin/sh\n",
	})

	tests := []struct {
		name string
		ref  Ref
		want Verdict
	}{
		{"known target", Ref{Kind: KindMake, Name: "build"}, VerdictOK},
		{"unknown target", Ref{Kind: KindMake, Name: "gone"}, VerdictMissing},
		{"variable target", Ref{Kind: KindMake, Name: "$(X)"}, VerdictUnresolvable},
		{"present path", Ref{Kind: KindPath, Name: "scripts/here.sh"}, VerdictOK},
		{"absent path", Ref{Kind: KindPath, Name: "scripts/gone.sh"}, VerdictMissing},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Resolve(root, []Ref{tc.ref})
			if len(got) != 1 {
				t.Fatalf("got %d results, want 1", len(got))
			}
			if got[0].Verdict != tc.want {
				t.Errorf("Verdict = %s, want %s", got[0].Verdict, tc.want)
			}
			if got[0].Bad() != (tc.want == VerdictMissing) {
				t.Errorf("Bad() = %v for verdict %s", got[0].Bad(), got[0].Verdict)
			}
		})
	}
}

// The distinction that keeps the gate from going inert on a real repository:
// rook has 85 opaque file rules, and downgrading on their account silenced
// every finding.
func TestOpaqueFileRulesDoNotSilenceFindings(t *testing.T) {
	fileRule := writeTree(t, map[string]string{
		"Makefile": ".PHONY: build\nbuild:\n\t@echo hi\n$(TOOLS):\n\ttouch $@\n",
	})
	if got := Resolve(fileRule, []Ref{{Kind: KindMake, Name: "gone"}}); got[0].Verdict != VerdictMissing {
		t.Errorf("with an opaque file rule: got %s, want %s", got[0].Verdict, VerdictMissing)
	}

	phony := writeTree(t, map[string]string{
		"Makefile": ".PHONY: build\nbuild:\n\t@echo hi\n.PHONY: $(SUITES)\n",
	})
	if got := Resolve(phony, []Ref{{Kind: KindMake, Name: "gone"}}); got[0].Verdict != VerdictUnresolvable {
		t.Errorf("with an opaque .PHONY: got %s, want %s", got[0].Verdict, VerdictUnresolvable)
	}
}

func TestResolveWithoutMakefile(t *testing.T) {
	root := writeTree(t, map[string]string{"README.md": "hi\n"})
	got := Resolve(root, []Ref{{Kind: KindMake, Name: "build"}})
	if got[0].Verdict != VerdictUnresolvable {
		t.Errorf("Verdict = %s, want %s", got[0].Verdict, VerdictUnresolvable)
	}
}

// The shipped gate must agree with the suite; a skill runs --self-test where
// this suite is not available.
func TestSelfTest(t *testing.T) {
	if fails := SelfTest(); len(fails) > 0 {
		t.Errorf("SelfTest reported %d failure(s): %v", len(fails), fails)
	}
}

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatalf("MkdirAll %s: %v", p, err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatalf("WriteFile %s: %v", p, err)
		}
	}
	return dir
}

func assertNames(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %v, want %v", got, want)
			return
		}
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

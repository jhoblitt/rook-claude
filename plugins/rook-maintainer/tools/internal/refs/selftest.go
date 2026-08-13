package refs

import (
	"fmt"
	"os"
	"path/filepath"
)

// SelfTest exercises every check against fixtures and returns what disagreed;
// empty means the gate behaves. It ships inside the binary because the gate is
// invoked from a skill, where the test suite is not available and a silently
// wrong gate reads as "nothing to fix" — here, as "the backport is clean".
//
// The fixtures are the real rook/rook release-1.20 shape that motivated this
// tool: a branch whose target is still `markdownlint` receiving a backported
// doc that says `make lint.markdown`, alongside a reference to a script that
// only exists on master.
func SelfTest() []string {
	var fails []string

	clean, err := fixtureTree("")
	if err != nil {
		return []string{fmt.Sprintf("fixture tree: %v", err)}
	}
	defer func() { _ = os.RemoveAll(clean) }()

	opaqueFile, err := fixtureTree("file")
	if err != nil {
		return []string{fmt.Sprintf("fixture tree (opaque file rule): %v", err)}
	}
	defer func() { _ = os.RemoveAll(opaqueFile) }()

	opaquePhony, err := fixtureTree("phony")
	if err != nil {
		return []string{fmt.Sprintf("fixture tree (opaque phony): %v", err)}
	}
	defer func() { _ = os.RemoveAll(opaquePhony) }()

	const diff = `diff --git a/Documentation/dev.md b/Documentation/dev.md
--- a/Documentation/dev.md
+++ b/Documentation/dev.md
@@ -1,2 +1,9 @@
 unchanged context
+| Markdown formatting | ` + "`make markdownlint`" + ` | docs |
+| Broken row | ` + "`make lint.markdown`" + ` | docs |
+run ` + "`make codegen`" + ` after editing the API
+the checker lives at tests/scripts/present.sh
+and the missing one at tests/scripts/absent.sh
+see .github/workflows/ci.yml for the job
+ignore https://github.com/rook/rook/blob/master/gone.md and k8s.io/api/core/v1
+either and/or neither is a path
+it compares against origin/release-1.20, override with upstream/release-1.20
+or against origin/master when on the default branch
-removed ` + "`make deleted-target`" + ` line
`

	got := map[string]Result{}
	for _, r := range Resolve(clean, Extract(diff)) {
		got[string(r.Kind)+" "+r.Name] = r
	}

	want := []struct {
		key     string
		verdict Verdict
	}{
		{"make-target markdownlint", VerdictOK},
		{"make-target lint.markdown", VerdictMissing},
		{"make-target codegen", VerdictOK},
		{"path tests/scripts/present.sh", VerdictOK},
		{"path tests/scripts/absent.sh", VerdictMissing},
		{"path .github/workflows/ci.yml", VerdictOK},
	}
	for _, w := range want {
		r, ok := got[w.key]
		if !ok {
			fails = append(fails, fmt.Sprintf("%s: not extracted", w.key))
			continue
		}
		if r.Verdict != w.verdict {
			fails = append(fails, fmt.Sprintf("%s: got %s, want %s", w.key, r.Verdict, w.verdict))
		}
	}

	// Things that must never be extracted at all.
	for _, key := range []string{
		"path rook/rook/blob/master/gone.md", // lives inside a URL
		"path k8s.io/api/core/v1",            // namespaced identifier, not a path
		"path and/or",                        // prose
		"make-target deleted-target",         // on a removed line
		"path origin/release-1.20",           // a git ref, not a path
		"path upstream/release-1.20",         // likewise
		"path origin/master",                 // no extension at all
	} {
		if r, ok := got[key]; ok {
			fails = append(fails, fmt.Sprintf("%s: extracted as %s, want no reference", key, r.Verdict))
		}
	}

	// A variable target name is reported, never accused.
	varDiff := "+++ b/x.md\n@@ -0,0 +1,1 @@\n+run `make $(TOOL_TARGET)` first\n"
	varRes := Resolve(clean, Extract(varDiff))
	switch {
	case len(varRes) != 1:
		fails = append(fails, fmt.Sprintf("variable target: got %d references, want 1", len(varRes)))
	case varRes[0].Verdict != VerdictUnresolvable:
		fails = append(fails, fmt.Sprintf("variable target: got %s, want %s",
			varRes[0].Verdict, VerdictUnresolvable))
	}

	// Opaque FILE rules are tool binary paths and must NOT silence a finding.
	// rook has 85 of them; downgrading on their account made the gate inert.
	missDiff := "+++ b/x.md\n@@ -0,0 +1,1 @@\n+run `make lint.markdown` first\n"
	if v := single(opaqueFile, missDiff); v != VerdictMissing {
		fails = append(fails, fmt.Sprintf("opaque file rule: got %s, want %s", v, VerdictMissing))
	}
	// An opaque .PHONY genuinely can hide a documented target, so it does.
	if v := single(opaquePhony, missDiff); v != VerdictUnresolvable {
		fails = append(fails, fmt.Sprintf("opaque phony: got %s, want %s", v, VerdictUnresolvable))
	}

	// Prose must not manufacture targets. "add a make target" is English, and
	// mergify writes exactly that into a conflict marker.
	prose := "+++ b/x.md\n@@ -0,0 +1,3 @@\n" +
		"+build: add a make target to check commit messages\n" +
		"+make sure the cluster is up\n" +
		"+>>>>>>> 800e88ef2 (build: add a make target to check commit messages)\n"
	if got := Extract(prose); len(got) != 0 {
		fails = append(fails, fmt.Sprintf("prose: extracted %d reference(s), want 0: %+v", len(got), got))
	}

	// A fenced block is code even without backtick spans.
	fenced := "+++ b/x.md\n@@ -0,0 +1,3 @@\n+```console\n+make lint.markdown\n+```\n"
	if v := single(clean, fenced); v != VerdictMissing {
		fails = append(fails, fmt.Sprintf("fenced block: got %s, want %s", v, VerdictMissing))
	}

	return fails
}

// single resolves a diff expected to yield exactly one reference and returns
// its verdict, or a sentinel naming what went wrong instead.
func single(root, diff string) Verdict {
	res := Resolve(root, Extract(diff))
	if len(res) != 1 {
		return Verdict(fmt.Sprintf("<%d references>", len(res)))
	}
	return res[0].Verdict
}

// fixtureTree builds a throwaway repository. opaque selects a target whose
// name stays variable: "file" for a tool-binary rule, "phony" for a .PHONY
// declaration. The two are treated differently and both must be covered.
func fixtureTree(opaque string) (string, error) {
	dir, err := os.MkdirTemp("", "validate-refs-selftest")
	if err != nil {
		return "", err
	}

	makefile := `MARKDOWNLINT := docker run --rm markdownlint-cli2

.PHONY: markdownlint
markdownlint: ## Check formatting of documentation sources
	@$(MARKDOWNLINT) "Documentation/**/**.md"

include build/makelib/*.mk
`
	// Deliberately never assigned: a variable the parse CAN resolve is not
	// opaque, it is just indirection.
	switch opaque {
	case "file":
		makefile += "\n$(TOOL_BINARIES):\n\ttouch $@\n"
	case "phony":
		makefile += "\n.PHONY: $(GENERATED_SUITES)\n"
	}

	files := map[string]string{
		"Makefile":                 makefile,
		"build/makelib/common.mk":  ".PHONY: codegen\ncodegen:\n\t@echo generating\n",
		"tests/scripts/present.sh": "#!/bin/sh\n",
		".github/workflows/ci.yml": "name: ci\n",
	}
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			_ = os.RemoveAll(dir)
			return "", err
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			_ = os.RemoveAll(dir)
			return "", err
		}
	}
	return dir, nil
}

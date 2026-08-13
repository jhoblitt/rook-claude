package refs

import (
	"os"
	"path/filepath"
	"strings"
)

// maxIncludeDepth bounds the include walk. A cycle is already stopped by the
// visited set; this stops a pathological chain from costing a stack.
const maxIncludeDepth = 16

// maxExpandPasses bounds variable expansion. Make itself allows recursive
// variables to expand indefinitely; we only need enough passes to resolve the
// one-or-two-level indirection real build systems use.
const maxExpandPasses = 8

// TargetSet is what a tree's make targets look like after a static parse.
type TargetSet struct {
	// Names holds every target name the parse could prove.
	Names map[string]bool
	// OpaqueFile holds file-rule targets whose name is still variable after
	// expansion ("$(HELM):"). In a real build system these are tool binary
	// paths, so they cannot be the definition of a documented target — they
	// are reported as a coverage note and nothing more.
	OpaqueFile []string
	// OpaquePhony holds variable names declared .PHONY (".PHONY: $(SUITES)").
	// These CAN hide a documented target name, so their presence is what
	// makes absence unprovable.
	OpaquePhony []string
}

// makefileNames is GNU make's own search order.
var makefileNames = []string{"GNUmakefile", "makefile", "Makefile"}

// Targets statically parses the makefile at root and every file it includes,
// returning the target names it can prove exist.
//
// Static parsing, not `make --print-data-base`: running the makefile is not
// hermetic. rook's Makefile shells out at parse time and tries to download
// tooling, which fails in a sandbox and would make this gate report drift that
// is really a missing container runtime.
//
// Two passes, because make is not order-dependent: a rule may name a variable
// that the makefile assigns further down, or in a file included later. A
// single pass in file order leaves those targets looking variable when they
// are merely defined out of order.
func Targets(root string) (TargetSet, error) {
	ts := TargetSet{Names: map[string]bool{}}

	start := ""
	for _, name := range makefileNames {
		p := filepath.Join(root, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			start = p
			break
		}
	}
	if start == "" {
		return ts, os.ErrNotExist
	}

	vars := map[string]string{}
	var files []string
	collectVars(root, start, 0, map[string]bool{}, vars, &files)
	for _, f := range files {
		collectTargets(f, vars, &ts)
	}
	return ts, nil
}

// collectVars is pass one: every assignment in the tree, and the list of files
// the include graph reaches. Best-effort by construction — an unreadable
// include is a coverage gap, not an error, or every build system with a
// generated include would read as broken.
func collectVars(root, path string, depth int, visited map[string]bool, vars map[string]string, files *[]string) {
	if depth > maxIncludeDepth || visited[path] {
		return
	}
	visited[path] = true
	*files = append(*files, path)

	for _, raw := range readLines(path) {
		line := strings.TrimRight(stripComment(raw), " \t\r")
		if line == "" || strings.HasPrefix(line, "\t") {
			continue
		}
		if name, value, ok := assignment(line); ok {
			vars[name] = value
			continue
		}
		if inc, ok := includeDirective(line); ok {
			for _, f := range resolveIncludes(root, path, expand(inc, vars)) {
				collectVars(root, f, depth+1, visited, vars, files)
			}
		}
	}
}

// collectTargets is pass two: rule lines read with the complete variable map.
func collectTargets(path string, vars map[string]string, ts *TargetSet) {
	for _, raw := range readLines(path) {
		line := strings.TrimRight(stripComment(raw), " \t\r")
		if line == "" || strings.HasPrefix(line, "\t") {
			continue
		}
		if _, _, ok := assignment(line); ok {
			continue
		}
		if _, ok := includeDirective(line); ok {
			continue
		}
		readRule(line, vars, ts)
	}
}

func readLines(path string) []string {
	data, err := os.ReadFile(path) // #nosec G304 -- paths come from the makefile being parsed
	if err != nil {
		return nil
	}
	return strings.Split(string(data), "\n")
}

// stripComment removes a trailing makefile comment. It deliberately ignores
// escaped hashes: a `\#` in a target name is not a thing we need to support,
// and treating one as a comment only costs coverage.
func stripComment(line string) string {
	if i := strings.Index(line, "#"); i >= 0 {
		return line[:i]
	}
	return line
}

// assignment reports a variable assignment and its value. All of make's
// flavours land here, which is what keeps `MARKDOWNLINT := ...` from being
// read as a target named MARKDOWNLINT.
func assignment(line string) (string, string, bool) {
	for _, op := range []string{":::=", "::=", ":=", "?=", "+=", "!=", "="} {
		i := strings.Index(line, op)
		if i <= 0 {
			continue
		}
		name := strings.TrimSpace(line[:i])
		if name == "" || strings.ContainsAny(name, " \t:") {
			continue
		}
		return name, strings.TrimSpace(line[i+len(op):]), true
	}
	return "", "", false
}

func includeDirective(line string) (string, bool) {
	t := strings.TrimSpace(line)
	for _, kw := range []string{"include ", "-include ", "sinclude "} {
		if strings.HasPrefix(t, kw) {
			return strings.TrimSpace(t[len(kw):]), true
		}
	}
	return "", false
}

// resolveIncludes turns an include argument into real paths, honouring globs.
// Relative includes resolve against the root, as make does when invoked there.
func resolveIncludes(root, from, arg string) []string {
	var out []string
	for _, field := range strings.Fields(arg) {
		if strings.Contains(field, "$") {
			continue // still variable after expansion; unreadable either way
		}
		if filepath.IsAbs(field) {
			if matches, err := filepath.Glob(field); err == nil {
				out = append(out, matches...)
			}
			continue
		}
		// Try the root first, then the including file's directory.
		for _, dir := range []string{root, filepath.Dir(from)} {
			matches, err := filepath.Glob(filepath.Join(dir, field))
			if err == nil && len(matches) > 0 {
				out = append(out, matches...)
				break
			}
		}
	}
	return out
}

// readRule reads target names off a rule line.
//
// `.PHONY:` prerequisites are harvested as names too. That is not a shortcut:
// a phony declaration is direct evidence the target exists, and it catches
// targets whose rule line this parse would otherwise have to guess at.
func readRule(line string, vars map[string]string, ts *TargetSet) {
	i := strings.Index(line, ":")
	if i <= 0 {
		return
	}
	names := strings.Fields(expand(line[:i], vars))
	rest := line[i+1:]

	if len(names) == 1 && names[0] == ".PHONY" {
		for _, p := range strings.Fields(expand(rest, vars)) {
			addTarget(p, true, ts)
		}
		return
	}
	for _, n := range names {
		addTarget(n, false, ts)
	}
}

func addTarget(name string, phony bool, ts *TargetSet) {
	switch {
	case name == "":
	case strings.Contains(name, "$"):
		if phony {
			ts.OpaquePhony = append(ts.OpaquePhony, name)
		} else {
			ts.OpaqueFile = append(ts.OpaqueFile, name)
		}
	case strings.Contains(name, "%"):
		// A pattern rule defines a shape, not a name worth matching against.
	case strings.HasPrefix(name, ".") && !strings.Contains(name, "/"):
		// Make's special targets (.PHONY, .SUFFIXES) are directives, not
		// targets a reader would be told to run. The slash test keeps real
		// file targets like .cache/tool, which are ordinary names.
	default:
		ts.Names[name] = true
	}
}

// expand substitutes $(NAME) and ${NAME} for known variables, leaving unknown
// ones intact so the caller can see that the text is still variable.
func expand(s string, vars map[string]string) string {
	for pass := 0; pass < maxExpandPasses && strings.Contains(s, "$"); pass++ {
		before := s
		for name, value := range vars {
			s = strings.ReplaceAll(s, "$("+name+")", value)
			s = strings.ReplaceAll(s, "${"+name+"}", value)
		}
		if s == before {
			break
		}
	}
	return s
}

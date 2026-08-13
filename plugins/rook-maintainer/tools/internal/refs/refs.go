// Package refs finds the things a diff's added lines point at — make targets
// and repo-relative paths — and reports which of them do not exist on the
// branch being changed.
//
// The failure it exists to catch is drift between prose and tooling. A
// backport replays a commit onto a branch whose build system is older, or a
// rename lands without the documentation that names the old target; either
// way the result is a file that is valid markdown, passes every linter, and
// tells the reader to run something that is not there.
package refs

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Kind distinguishes what a reference points at.
type Kind string

const (
	KindMake Kind = "make-target"
	KindPath Kind = "path"
)

// Verdict is the outcome of resolving one reference against a tree.
type Verdict string

const (
	// VerdictOK means the reference resolves.
	VerdictOK Verdict = "ok"
	// VerdictMissing means it provably does not. Only this fails the gate.
	VerdictMissing Verdict = "missing"
	// VerdictUnresolvable means the parse could not decide. It is reported
	// and never counted against the diff: a gate that cries wolf on every
	// variable-named target is a gate people learn to bypass.
	VerdictUnresolvable Verdict = "unresolvable"
)

// Ref is one thing an added line points at.
type Ref struct {
	Kind Kind   `json:"kind"`
	Name string `json:"name"`
	File string `json:"file,omitempty"`
	Line int    `json:"line,omitempty"`
}

// Result is a Ref plus what resolving it found.
type Result struct {
	Ref
	Verdict Verdict `json:"verdict"`
	Note    string  `json:"note,omitempty"`
}

// Bad reports whether this result should fail the gate.
func (r Result) Bad() bool { return r.Verdict == VerdictMissing }

var (
	urlRe = regexp.MustCompile(`https?://\S+`)

	// A make invocation: any number of flags and variable assignments, then
	// the first word that is neither. `-C dir` and `-f file` take an
	// argument, so they are matched together with it or the argument reads
	// as the target.
	// The target alternation takes a whole $(VAR)/${VAR} first: the fallback
	// stops at ")" so a markdown link's closing paren is not swallowed, which
	// would otherwise truncate a variable target to "$(VAR".
	makeRe = regexp.MustCompile(
		`\bmake\s+((?:(?:-[CfoW]\s+\S+|--?[A-Za-z][\w-]*(?:=\S+)?|[A-Za-z_]\w*=\S+)\s+)*)` +
			`(\$[({][A-Za-z_]\w*[)}]|[^\s` + "`" + `'"|)\]]+)`)

	// A repo-relative path: at least one slash, and either a file extension
	// or a trailing slash. Anything looser matches "and/or" and every
	// namespaced identifier in the diff.
	pathRe = regexp.MustCompile(`(?:[A-Za-z0-9_.\-]+/)+[A-Za-z0-9_.\-]*`)

	hunkRe = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)`)
)

// Extract pulls references from the added lines of a unified diff. Removed
// lines are skipped: a reference this change deletes is not this change's
// problem. Results are deduplicated by kind and name, keeping the first
// position each was seen at.
func Extract(diff string) []Ref {
	var refs []Ref
	seen := map[string]bool{}
	file := ""
	newLine := 0
	inFence := false

	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ "):
			file = diffPath(strings.TrimSpace(line[4:]))
			inFence = false // fence state does not survive a file boundary
			continue
		case strings.HasPrefix(line, "--- "), strings.HasPrefix(line, `\ `):
			continue
		}
		if m := hunkRe.FindStringSubmatch(line); m != nil {
			newLine, _ = strconv.Atoi(m[1])
			// A hunk starts an unknown distance into the file, so whatever
			// fence state the previous hunk left is not carried over.
			inFence = false
			continue
		}

		// Fence state is tracked over context lines as well as added ones:
		// an added line inside a block the diff did not touch is still
		// inside that block.
		body, kind, tracked := diffBody(line)
		if !tracked {
			continue
		}
		if isFence(body) {
			inFence = !inFence
		} else if kind == '+' {
			for _, r := range scan(body, inFence) {
				key := string(r.Kind) + "\x00" + r.Name
				if seen[key] {
					continue
				}
				seen[key] = true
				r.File, r.Line = file, newLine
				refs = append(refs, r)
			}
		}
		if kind != '-' {
			newLine++ // a removed line does not advance the new-file counter
		}
	}
	return refs
}

// diffBody splits a diff line into its origin marker and payload, reporting
// false for lines that are not part of the file content at all.
func diffBody(line string) (string, byte, bool) {
	if line == "" {
		return "", ' ', true
	}
	switch line[0] {
	case '+', '-', ' ':
		return line[1:], line[0], true
	}
	return "", 0, false
}

// isFence reports a markdown code-fence delimiter.
func isFence(body string) bool {
	t := strings.TrimSpace(body)
	return strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")
}

// diffPath turns a `+++ b/path` operand into a repo-relative path.
func diffPath(s string) string {
	if i := strings.IndexAny(s, "\t"); i >= 0 {
		s = s[:i]
	}
	if s == "/dev/null" {
		return ""
	}
	if len(s) > 2 && (strings.HasPrefix(s, "b/") || strings.HasPrefix(s, "a/")) {
		return s[2:]
	}
	return s
}

// scan finds every reference on one line of added text. inFence says the line
// sits inside a markdown code block, which makes the whole line code.
func scan(text string, inFence bool) []Ref {
	// URLs are stripped first so their path components are not mistaken for
	// repo paths; check-links already owns URL liveness.
	clean := urlRe.ReplaceAllString(text, " ")

	var refs []Ref
	// A make reference is only read from code — a fenced block or a `code
	// span`. English prose says "make sure", "make it easier", "add a make
	// target", and every one of those would otherwise be reported as a
	// missing target. Documentation writes commands as code; requiring that
	// trades a little recall for a gate that does not cry wolf.
	for _, seg := range codeSegments(clean, inFence) {
		for _, m := range makeRe.FindAllStringSubmatch(seg, -1) {
			if name, ok := makeTarget(m[2]); ok {
				refs = append(refs, Ref{Kind: KindMake, Name: name})
			}
		}
	}
	for _, loc := range pathRe.FindAllStringIndex(clean, -1) {
		// A match preceded by a slash is the tail of an absolute path — a
		// container mount such as /workspace/tests/scripts/x.sh — not a path
		// in this repository.
		if loc[0] > 0 && clean[loc[0]-1] == '/' {
			continue
		}
		if name, ok := repoPath(clean[loc[0]:loc[1]]); ok {
			refs = append(refs, Ref{Kind: KindPath, Name: name})
		}
	}
	return refs
}

// codeSegments returns the stretches of a line that are code. Inside a fenced
// block that is the whole line; otherwise it is whatever sits between
// backticks. An unterminated span is taken to run to end of line, which is
// what a table cell like "| `make foo` |" needs when the diff truncates it.
func codeSegments(line string, inFence bool) []string {
	if inFence {
		return []string{line}
	}
	var segs []string
	rest := line
	for {
		i := strings.Index(rest, "`")
		if i < 0 {
			return segs
		}
		rest = rest[i+1:]
		j := strings.Index(rest, "`")
		if j < 0 {
			return append(segs, rest)
		}
		segs = append(segs, rest[:j])
		rest = rest[j+1:]
	}
}

// makeTarget cleans a captured target word, reporting false when the word is
// not a target at all.
func makeTarget(word string) (string, bool) {
	word = strings.Trim(word, "`'\",.;:")
	switch {
	case word == "":
		return "", false
	case strings.HasPrefix(word, "-"):
		return "", false // a flag we did not model; no target was named
	case strings.Contains(word, "="):
		return "", false // a variable assignment, so make builds the default goal
	case strings.ContainsAny(word, "$"):
		return word, true // still variable: reported as unresolvable downstream
	}
	return word, true
}

// repoPath cleans a captured path, reporting false when it is not a
// repo-relative path worth resolving.
func repoPath(word string) (string, bool) {
	word = strings.Trim(word, "`'\",;:()[]<>")
	if i := strings.IndexAny(word, "#?"); i >= 0 {
		word = word[:i] // drop a markdown anchor or query string
	}
	word = strings.TrimSuffix(word, ".")
	if word == "" || strings.Contains(word, "..") || strings.ContainsAny(word, "*$ ") {
		return "", false
	}
	segs := strings.Split(strings.TrimSuffix(word, "/"), "/")
	if len(segs) < 2 || segs[0] == "" {
		return "", false
	}
	// A dotted first segment is a hostname or a namespaced identifier
	// (github.com/..., k8s.io/...), not a path in this repository. A leading
	// dot is different: .github/ is a real directory.
	if strings.Contains(segs[0], ".") && !strings.HasPrefix(segs[0], ".") {
		return "", false
	}
	if strings.HasSuffix(word, "/") {
		return word, true
	}
	last := segs[len(segs)-1]
	dot := strings.LastIndex(last, ".")
	if dot < 0 {
		// No extension and no trailing slash: too weak a signal to resolve.
		// Under-reporting here is deliberate — see the package doc.
		return "", false
	}
	// The extension must start with a letter. A numeric one means a version,
	// which is how git refs enter documentation: `origin/release-1.20` and
	// `upstream/release-1.20` are refs, not paths, and this tool's own
	// subject matter — contributor docs — is full of them.
	ext := last[dot+1:]
	if ext == "" || !isLetter(ext[0]) {
		return "", false
	}
	return word, true
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// Resolve decides each reference against the tree at root.
//
// Targets are resolved against the working tree rather than a git ref so that
// a target the diff itself adds counts as present. Resolving against the base
// commit is what makes a self-contained change look like it references
// something that does not exist.
func Resolve(root string, refs []Ref) []Result {
	ts, err := Targets(root)
	makefileRead := err == nil

	out := make([]Result, 0, len(refs))
	for _, r := range refs {
		out = append(out, resolveOne(root, r, ts, makefileRead))
	}
	return out
}

func resolveOne(root string, r Ref, ts TargetSet, makefileRead bool) Result {
	res := Result{Ref: r}
	switch r.Kind {
	case KindMake:
		switch {
		case strings.Contains(r.Name, "$"):
			res.Verdict, res.Note = VerdictUnresolvable, "target name is a variable"
		case !makefileRead:
			res.Verdict, res.Note = VerdictUnresolvable, "no makefile found at the root"
		case ts.Names[r.Name]:
			res.Verdict = VerdictOK
		case len(ts.OpaquePhony) > 0:
			// A .PHONY declaration whose name is a variable can hide a
			// documented target, so absence is not provable here.
			//
			// Opaque FILE rules deliberately do not downgrade. In a real
			// build system those are tool binary paths ($(HELM):,
			// $(KUSTOMIZE):) — rook has 85 of them — and treating each as a
			// possible definition of a documented target silences every
			// finding the gate exists to make.
			res.Verdict = VerdictUnresolvable
			res.Note = "not found, and " + strconv.Itoa(len(ts.OpaquePhony)) +
				" .PHONY declaration(s) have variable names"
		default:
			res.Verdict = VerdictMissing
		}
	case KindPath:
		if _, err := os.Stat(filepath.Join(root, filepath.Clean(r.Name))); err == nil {
			res.Verdict = VerdictOK
		} else {
			res.Verdict = VerdictMissing
		}
	default:
		res.Verdict, res.Note = VerdictUnresolvable, "unknown reference kind"
	}
	return res
}

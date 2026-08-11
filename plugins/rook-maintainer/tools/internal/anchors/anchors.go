// Package anchors decides whether a review payload's inline comments can be
// posted against a PR's diff.
//
// GitHub rejects the ENTIRE review call when one inline comment names a line
// the diff does not touch, and accepts-then-misplaces a comment whose `side`
// is wrong. Both outcomes are decided by set membership over the diff's hunks,
// so they are decided here rather than by an agent reading the diff.
//
// Spec: skills/rook-code-review/references/posting.md.
package anchors

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	Left  = "LEFT"
	Right = "RIGHT"
)

var hunkRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// Sides holds the line numbers commentable on each version of one file.
type Sides struct {
	Left  map[int64]bool
	Right map[int64]bool
}

func newSides() *Sides {
	return &Sides{Left: map[int64]bool{}, Right: map[int64]bool{}}
}

// Has reports whether line is commentable on side.
func (s *Sides) Has(side string, line int64) bool {
	switch side {
	case Left:
		return s.Left[line]
	case Right:
		return s.Right[line]
	}
	return false
}

// Files maps a path as GitHub anchors it to that file's commentable lines.
type Files map[string]*Sides

// Commentable maps every path in a unified diff to its commentable lines.
//
// A line is commentable on RIGHT when it is added or context (it exists in the
// new file inside a hunk), and on LEFT when it is removed or context. Context
// lines are commentable on both, which is why `side` cannot be inferred from
// the line number alone.
func Commentable(diff string) Files {
	files := Files{}
	var path, oldPath string
	var havePath, haveOld bool
	var oldLn, newLn int64
	inHunk := false

	// oldPath/haveOld carry across cases — a deleted file is anchored at the
	// path its `---` line gave. The `+++` values never outlive their own case.
	for _, raw := range splitLines(diff) {
		switch {
		case strings.HasPrefix(raw, "diff --git "):
			havePath, haveOld, inHunk = false, false, false
			continue
		case strings.HasPrefix(raw, "--- "):
			oldPath, haveOld = stripPrefix(raw[4:])
			inHunk = false
			continue
		case strings.HasPrefix(raw, "+++ "):
			newPath, haveNew := stripPrefix(raw[4:])
			// GitHub anchors a deleted file at its original path.
			if haveNew {
				path, havePath = newPath, true
			} else {
				path, havePath = oldPath, haveOld
			}
			if havePath && files[path] == nil {
				files[path] = newSides()
			}
			inHunk = false
			continue
		}

		if m := hunkRe.FindStringSubmatch(raw); m != nil {
			oldLn, newLn = atoi64(m[1]), atoi64(m[3])
			inHunk = true
			continue
		}

		if !inHunk || !havePath {
			continue
		}

		// A wholly empty line inside a hunk is an empty context line.
		marker := byte(' ')
		if raw != "" {
			marker = raw[0]
		}
		switch marker {
		case ' ':
			files[path].Left[oldLn] = true
			files[path].Right[newLn] = true
			oldLn++
			newLn++
		case '-':
			files[path].Left[oldLn] = true
			oldLn++
		case '+':
			files[path].Right[newLn] = true
			newLn++
		case '\\':
			// "\ No newline at end of file" — advances neither counter.
		default:
			inHunk = false
		}
	}

	return files
}

// splitLines splits on LF and drops one CR before it, so a CRLF diff parses
// like an LF one and a trailing newline does not yield a phantom context line.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSuffix(line, "\r")
	}
	return lines
}

// stripPrefix turns `a/pkg/x.go` into `pkg/x.go`; `/dev/null` reports false.
func stripPrefix(spec string) (string, bool) {
	spec, _, _ = strings.Cut(spec, "\t")
	spec = strings.TrimSpace(spec)
	if spec == "/dev/null" {
		return "", false
	}
	if len(spec) > 2 && spec[1] == '/' && (spec[0] == 'a' || spec[0] == 'b') {
		return spec[2:], true
	}
	return spec, true
}

func atoi64(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// Validate returns a human-readable problem for every anchor that cannot be
// posted; empty means the whole payload is postable.
func Validate(review any, files Files) []string {
	payload, ok := review.(map[string]any)
	if !ok {
		return []string{"review payload: top level must be an object"}
	}
	raw := payload["comments"]
	if !truthy(raw) {
		return nil
	}
	comments, ok := raw.([]any)
	if !ok {
		return []string{"review payload: `comments` must be a list"}
	}

	var problems []string
	for i, item := range comments {
		tag := fmt.Sprintf("comments[%d]", i)
		c, ok := item.(map[string]any)
		if !ok {
			problems = append(problems, tag+": not an object")
			continue
		}
		if problem := check(tag, c, files); problem != "" {
			problems = append(problems, problem)
		}
	}
	return problems
}

// check reports at most one problem per comment: the first failure decides,
// because a comment rejected for its path tells nothing useful about its line.
func check(tag string, c map[string]any, files Files) string {
	pathValue := c["path"]
	if !truthy(pathValue) {
		return tag + ": missing `path`"
	}
	path := pyStr(pathValue)

	line, ok := asInt(c["line"])
	if !ok {
		return fmt.Sprintf("%s %s: `line` must be an integer", tag, path)
	}

	sideValue, present := c["side"]
	if !present {
		sideValue = Right
	}
	side, _ := sideValue.(string)
	if side != Left && side != Right {
		return fmt.Sprintf("%s %s:%d: `side` must be LEFT or RIGHT, got %s",
			tag, path, line, pyRepr(sideValue))
	}

	key, isString := pathValue.(string)
	var sides *Sides
	if isString {
		sides = files[key]
	}
	if sides == nil {
		return fmt.Sprintf("%s %s:%d: file is not in the diff", tag, path, line)
	}

	startLine, startSide := c["start_line"], c["start_side"]
	if (startLine == nil) != (startSide == nil) {
		return fmt.Sprintf("%s %s:%d: multi-line anchors need BOTH `start_line` and `start_side`",
			tag, path, line)
	}
	if startLine != nil {
		if s, ok := startSide.(string); !ok || s != side {
			return fmt.Sprintf("%s %s:%d: `start_side` (%s) must equal `side` (%s)",
				tag, path, line, pyStr(startSide), side)
		}
		start, ok := asInt(startLine)
		if !ok || start > line {
			return fmt.Sprintf("%s %s:%d: `start_line` (%s) must be an integer <= `line`",
				tag, path, line, pyStr(startLine))
		}
		if !sides.Has(side, start) {
			return fmt.Sprintf("%s %s:%d %s: start line is outside the diff", tag, path, start, side)
		}
	}

	if !sides.Has(side, line) {
		other := Right
		if side == Right {
			other = Left
		}
		hint := ""
		if sides.Has(other, line) {
			hint = fmt.Sprintf(" (it IS commentable on %s \u2014 wrong side?)", other)
		}
		return fmt.Sprintf("%s %s:%d %s: line is outside the diff%s", tag, path, line, side, hint)
	}
	return ""
}

// Count reports how many inline comments the payload claims, for the summary
// line only — a payload whose `comments` is not a list still has a count.
func Count(review any) int {
	payload, ok := review.(map[string]any)
	if !ok {
		return 0
	}
	comments := payload["comments"]
	if !truthy(comments) {
		return 0
	}
	return pyLen(comments)
}

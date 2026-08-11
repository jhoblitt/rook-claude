package rtcommits

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtanalyze"
)

// gitLogArgs is the ONE command this tool runs under --repo and the ONE that
// produces a --log dump. Three things are pinned to it — the parser below, the
// testdata fixture, and the command string printed in provenance — so a dump
// captured with a different --format or without --name-status is rejected
// rather than half-read. Keep GitLogCommand's output copy-pasteable: the docs
// hand it to a human.
//
// -M asks for rename detection regardless of the checkout's diff.renames, and
// core.quotePath=false keeps non-ASCII paths literal; the parser still unquotes
// the C-style form, since a dump may come from a git that quoted them.
var gitLogArgs = []string{
	"-c", "core.quotePath=false",
	"log", "--no-merges", "-M",
	"--format=commit%x09%H%x09%aN%x09%aE%x09%aI",
	"--name-status",
}

// GitLogCommand is the exact command line whose output --log consumes.
func GitLogCommand() string {
	return "git " + strings.Join(gitLogArgs, " ")
}

const headerPrefix = "commit\t"

var statusCode = regexp.MustCompile(`^([ACDMTUXB]|[RC][0-9]{1,3})$`)

var cEscapes = [256]byte{'a': '\a', 'b': '\b', 'f': '\f', 'n': '\n', 'r': '\r', 't': '\t', 'v': '\v'}

// ParseLog reads a GitLogCommand dump. Every line must be a commit header, a
// name-status line or the blank line git puts between them: an unrecognised
// line is an error, because the alternative is a mine that quietly counts a
// fraction of the history and reports it as the whole.
func ParseLog(r io.Reader) ([]Commit, error) {
	var out []Commit
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for line := 1; sc.Scan(); line++ {
		text := sc.Text()
		switch {
		case text == "":
		case strings.HasPrefix(text, headerPrefix):
			c, err := parseHeader(text)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", line, err)
			}
			out = append(out, c)
		case len(out) == 0:
			return nil, fmt.Errorf("line %d: %q precedes any commit header; expected the output of: %s",
				line, text, GitLogCommand())
		default:
			paths, err := parseNameStatus(text)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", line, err)
			}
			cur := &out[len(out)-1]
			cur.Paths = append(cur.Paths, paths...)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func parseHeader(text string) (Commit, error) {
	f := strings.Split(text, "\t")
	if len(f) != 5 {
		return Commit{}, fmt.Errorf("commit header has %d tab-separated fields, want 5: %q", len(f), text)
	}
	if f[1] == "" {
		return Commit{}, fmt.Errorf("commit header has no sha: %q", text)
	}
	if f[2] == "" && f[3] == "" {
		return Commit{}, fmt.Errorf("commit %s has neither an author name nor an email", f[1])
	}
	when, err := rtanalyze.ParseISO(f[4])
	if err != nil {
		return Commit{}, fmt.Errorf("commit %s author date: %w", f[1], err)
	}
	return Commit{SHA: f[1], Name: f[2], Email: f[3], When: when}, nil
}

// parseNameStatus returns the paths one --name-status line touches. A rename or
// copy yields BOTH paths: moving a file out of an area is activity in that area
// as much as in the one it lands in.
func parseNameStatus(text string) ([]string, error) {
	f := strings.Split(text, "\t")
	if !statusCode.MatchString(f[0]) {
		return nil, fmt.Errorf("expected \"<status>\\t<path>\" from --name-status, got %q", text)
	}
	want := 2
	if f[0][0] == 'R' || f[0][0] == 'C' {
		want = 3
	}
	if len(f) != want {
		return nil, fmt.Errorf("%s status has %d tab-separated fields, want %d: %q", f[0], len(f), want, text)
	}
	paths := make([]string, 0, want-1)
	for _, raw := range f[1:] {
		p, err := unquotePath(raw)
		if err != nil {
			return nil, err
		}
		if p == "" {
			return nil, fmt.Errorf("empty path: %q", text)
		}
		paths = append(paths, p)
	}
	return paths, nil
}

// unquotePath undoes git's C-style path quoting (quote_c_style): the form a
// dump carries whenever a path holds a quote, a backslash, a control character
// or — without core.quotePath=false — a non-ASCII byte.
func unquotePath(p string) (string, error) {
	if !strings.HasPrefix(p, `"`) {
		return p, nil
	}
	if len(p) < 2 || !strings.HasSuffix(p, `"`) {
		return "", fmt.Errorf("unterminated quoted path: %s", p)
	}
	body := p[1 : len(p)-1]
	b := make([]byte, 0, len(body))
	for i := 0; i < len(body); {
		if body[i] != '\\' {
			b = append(b, body[i])
			i++
			continue
		}
		i++
		if i >= len(body) {
			return "", fmt.Errorf("quoted path ends in a backslash: %s", p)
		}
		switch e := body[i]; {
		case cEscapes[e] != 0:
			b = append(b, cEscapes[e])
			i++
		case e == '\\' || e == '"':
			b = append(b, e)
			i++
		default:
			if i+3 > len(body) {
				return "", fmt.Errorf("truncated octal escape in quoted path: %s", p)
			}
			n, err := strconv.ParseUint(body[i:i+3], 8, 8)
			if err != nil {
				return "", fmt.Errorf("bad escape \\%s in quoted path: %s", body[i:i+3], p)
			}
			b = append(b, byte(n))
			i += 3
		}
	}
	return string(b), nil
}

// Log runs GitLogCommand in repo and parses it. GIT_OPTIONAL_LOCKS=0 keeps a
// read-only mine from refreshing the index of a checkout it does not own — a
// clone whose .git is not writable would otherwise fail here.
func Log(ctx context.Context, repo string) ([]Commit, string, error) {
	head, err := gitOutput(ctx, repo, "rev-parse", "HEAD")
	if err != nil {
		return nil, "", err
	}
	out, err := gitOutput(ctx, repo, gitLogArgs...)
	if err != nil {
		return nil, "", err
	}
	commits, err := ParseLog(strings.NewReader(out))
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", repo, err)
	}
	return commits, head, nil
}

func gitOutput(ctx context.Context, repo string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s in %s failed: %s", strings.Join(args, " "), repo, msg)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

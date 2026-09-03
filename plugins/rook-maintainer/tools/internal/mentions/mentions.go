// Package mentions mines the @-mentions out of a GitHub issue thread.
//
// A bare `@\w+` scan turns shell prompts (root@pod), e-mail addresses and
// broker hostnames into bogus profile links, so mining is three layers:
//
//  1. Code stripping scoped per comment-document (fence state machine; an
//     unclosed fence in one comment must not leak into the next, because
//     GitHub renders every comment independently), plus inline backticks.
//  2. GitHub's mention syntax: an `@` not preceded by a word character, `.`,
//     `-`, `/` or `@`; login = alphanumerics and hyphens, at most 39
//     characters, starting alphanumeric.
//  3. Live resolution against `gh api users/<token>`, which drops what does
//     not resolve and keeps the API's canonical spelling.
//
// Layers 1 and 2 live here; layer 3 is in resolve.go.
package mentions

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxLoginLen is GitHub's login limit. A longer run of login characters is not
// a truncated mention, it is not a mention at all.
const MaxLoginLen = 39

var inlineCode = regexp.MustCompile("`[^`\\n]*`")

// StripCode removes fenced blocks and inline code from one comment-document.
//
// The fence state starts closed and is discarded at the end: an unterminated
// fence swallows the rest of THIS document only.
func StripCode(doc string) string {
	var kept []string
	inFence := false
	for _, line := range splitLines(doc) {
		if isFenceLine(line) {
			inFence = !inFence
			continue
		}
		if !inFence {
			kept = append(kept, line)
		}
	}
	return inlineCode.ReplaceAllString(strings.Join(kept, "\n"), " ")
}

// Candidates returns the mention tokens of a whole thread — issue body first,
// then every comment — deduplicated case-insensitively in first-mention order.
func Candidates(docs []string) []string {
	stripped := make([]string, 0, len(docs))
	for _, d := range docs {
		stripped = append(stripped, StripCode(d))
	}
	var out []string
	seen := make(map[string]bool)
	for _, tok := range Extract(strings.Join(stripped, "\n")) {
		key := strings.ToLower(tok)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, tok)
	}
	return out
}

// Extract returns every mention token in text, in order, with duplicates.
func Extract(text string) []string {
	var out []string
	for i := 0; i < len(text); i++ {
		if text[i] != '@' {
			continue
		}
		if prev, n := utf8.DecodeLastRuneInString(text[:i]); n > 0 && blocksMention(prev) {
			continue
		}
		tok, end := loginAt(text, i+1)
		if tok == "" {
			continue
		}
		out = append(out, tok)
		i = end - 1
	}
	return out
}

func blocksMention(r rune) bool {
	switch r {
	case '.', '-', '/', '@':
		return true
	}
	return isWord(r)
}

// loginAt reads the login starting at s[start] and returns it with the index
// just past it, or "" when no mention starts there.
//
// The Python original ends the login with a negative lookahead, which RE2
// cannot express, so the backtracking it implies is spelled out: try the
// longest allowed login first and shorten it until the character that follows
// is not a word character. Shortening only ever succeeds at a hyphen — every
// other login character is a word character — which is why `@ab-cd_` yields
// `ab` rather than nothing.
func loginAt(s string, start int) (string, int) {
	if start >= len(s) || !isLoginStart(s[start]) {
		return "", start
	}
	end := start
	for end < len(s) && isLoginChar(s[end]) {
		end++
	}
	run := s[start:end]
	n := len(run)
	if n > MaxLoginLen {
		n = MaxLoginLen
	}
	for ; n >= 1; n-- {
		if n < len(run) {
			if run[n] == '-' {
				return run[:n], start + n
			}
			continue
		}
		if next, size := utf8.DecodeRuneInString(s[end:]); size == 0 || !isWord(next) {
			return run, end
		}
	}
	return "", start
}

// ValidLogin reports whether s is a login under the grammar above. It is the
// one place that grammar is spelled out: rt-commits captures logins from
// contributor-controlled commit addresses and validate-kb gates the kb refresh,
// and a second spelling of "what a login looks like" is how the two drift.
func ValidLogin(s string) bool {
	if s == "" || len(s) > MaxLoginLen || !isLoginStart(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		if !isLoginChar(s[i]) {
			return false
		}
	}
	return true
}

func isLoginStart(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func isLoginChar(c byte) bool {
	return isLoginStart(c) || c == '-'
}

// isWord mirrors Python's `\w` on str: alphanumeric by the Unicode database,
// plus underscore.
func isWord(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
}

// isSpace mirrors Python's `\s` on str, which is Unicode White_Space plus the
// four C0 separators Unicode itself leaves out.
func isSpace(r rune) bool {
	switch r {
	case '\u001c', '\u001d', '\u001e', '\u001f':
		return true
	}
	return unicode.IsSpace(r)
}

// isFenceLine reports the Python `^[>\s]{0,4}(```|~~~)`. The quantifier
// backtracks, so this asks whether ANY prefix of up to four quote/space
// characters is followed by a fence marker.
func isFenceLine(line string) bool {
	rest := line
	for consumed := 0; ; consumed++ {
		if strings.HasPrefix(rest, "```") || strings.HasPrefix(rest, "~~~") {
			return true
		}
		if consumed == 4 {
			return false
		}
		r, size := utf8.DecodeRuneInString(rest)
		if size == 0 || (r != '>' && !isSpace(r)) {
			return false
		}
		rest = rest[size:]
	}
}

// splitLines mirrors Python's str.splitlines: it breaks on the vertical
// whitespace and separator codepoints, not just "\n", and drops a trailing
// empty field. Getting this wrong would let a form feed hide a fence marker.
func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !isLineBoundary(r) {
			i += size
			continue
		}
		out = append(out, s[start:i])
		i += size
		if r == '\r' && i < len(s) && s[i] == '\n' {
			i++
		}
		start = i
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func isLineBoundary(r rune) bool {
	switch r {
	case '\n', '\v', '\f', '\r',
		'\u001c', '\u001d', '\u001e', '\u0085', '\u2028', '\u2029':
		return true
	}
	return false
}

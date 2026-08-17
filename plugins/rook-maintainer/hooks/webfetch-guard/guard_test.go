package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"
)

// Hostile codepoints appear only as \u escapes here, never as literal bytes:
// a test for invisible-character handling that itself contains invisible
// characters cannot be reviewed by reading it.
func TestEvaluate(t *testing.T) {
	tests := []struct {
		name string
		url  string
		deny bool
	}{
		{"allowlisted host", "https://docs.ceph.com/en/squid/radosgw/", false},
		{"allowlisted root", "https://github.com/rook/rook", false},
		{"raw content host", "https://raw.githubusercontent.com/rook/rook/master/README.md", false},
		{"dot-bounded subdomain", "https://www.rfc-editor.org/rfc/rfc7231.html", false},
		{"uppercase host", "https://GitHub.com/rook/rook", false},
		{"explicit port", "https://github.com:443/rook/rook", false},
		{"trailing dot host", "https://github.com./rook/rook", false},

		{"unlisted host", "https://example.com/blog/post", true},
		{"lookalike suffix", "https://evilgithub.com/rook/rook", true},
		{"lookalike prefix", "https://github.com.evil.test/rook", true},
		{"userinfo smuggling", "https://github.com@evil.test/rook", true},
		{"gist content host", "https://gist.githubusercontent.com/someone/1/raw/x.md", true},
		{"objects content host", "https://objects.githubusercontent.com/github-production-release-asset/x", true},
		{"lookalike raw prefix", "https://evilraw.githubusercontent.com/rook/rook/master/README.md", true},
		{"plain http", "http://docs.ceph.com/en/squid/", true},
		{"non-web scheme", "file:///etc/passwd", true},
		{"scheme relative", "//github.com/rook/rook", true},
		{"zero width in host", "https://github\u200b.com/rook", true},
		{"bidi override in path", "https://github.com/rook/\u202egnp.exe", true},
		{"tag block in path", "https://github.com/rook\U000E0041/x", true},
		{"empty", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := evaluate(tc.url, allowedHosts)
			if got.deny != tc.deny {
				t.Fatalf("evaluate(%q) deny=%v, want %v (reason: %s)",
					tc.url, got.deny, tc.deny, got.reason)
			}
			if got.deny && got.reason == "" {
				t.Fatal("deny with empty reason")
			}
		})
	}
}

func TestEvaluateHonorsExtraAllow(t *testing.T) {
	extra := append([]string{}, allowedHosts...)
	extra = append(extra, "spec.example.test")

	if d := evaluate("https://spec.example.test/rfc", allowedHosts); !d.deny {
		t.Fatal("host allowed before it was added to the list")
	}
	if d := evaluate("https://spec.example.test/rfc", extra); d.deny {
		t.Fatalf("extra-allowed host denied: %s", d.reason)
	}
	if d := evaluate("https://sub.spec.example.test/rfc", extra); d.deny {
		t.Fatalf("subdomain of extra-allowed host denied: %s", d.reason)
	}
}

const (
	referencePath   = "../../skills/rook-code-review/references/docs-sync.md"
	allowlistMarker = "so it is WebFetch and it is allowlisted:"
)

// separatorPattern is what may sit between two backticked hosts in the bullet:
// an optional comma and whitespace, nothing else. Prose after the last host
// ends the list, which is what keeps the parser from walking off into the rest
// of the document and inventing an allowlist.
var separatorPattern = regexp.MustCompile(`^,?\s*$`)

// parseReferenceAllowlist returns the hosts the reviewer reads. Every failure
// mode is an error rather than an empty result: a parser that silently matches
// nothing would report the two lists as trivially equal, which is the exact
// drift this test exists to catch.
func parseReferenceAllowlist(doc string) ([]string, error) {
	if n := strings.Count(doc, allowlistMarker); n != 1 {
		return nil, fmt.Errorf("found %d occurrences of %q, want exactly 1", n, allowlistMarker)
	}
	rest := doc[strings.Index(doc, allowlistMarker)+len(allowlistMarker):]

	var hosts []string
	for {
		open := strings.Index(rest, "`")
		if open < 0 || !separatorPattern.MatchString(rest[:open]) {
			break
		}
		span := rest[open+1:]
		end := strings.Index(span, "`")
		if end < 0 {
			return nil, fmt.Errorf("unterminated code span after host %q", lastOf(hosts))
		}
		host := span[:end]
		if host == "" || strings.ContainsAny(host, " \t\n") {
			return nil, fmt.Errorf("code span %q in the allowlist is not a host", host)
		}
		hosts = append(hosts, host)
		rest = span[end+1:]
	}
	if len(hosts) < 2 {
		return nil, fmt.Errorf("parsed %d hosts after the marker, want the full list", len(hosts))
	}
	return hosts, nil
}

func lastOf(hosts []string) string {
	if len(hosts) == 0 {
		return "(none)"
	}
	return hosts[len(hosts)-1]
}

func TestAllowedHostsMatchesReference(t *testing.T) {
	doc, err := os.ReadFile(referencePath)
	if err != nil {
		t.Fatalf("cannot read %s: %v", referencePath, err)
	}
	hosts, err := parseReferenceAllowlist(string(doc))
	if err != nil {
		t.Fatalf("cannot parse the allowlist out of %s: %v", referencePath, err)
	}

	inCode := make(map[string]bool, len(allowedHosts))
	for _, h := range allowedHosts {
		inCode[h] = true
	}
	inProse := make(map[string]bool, len(hosts))
	for _, h := range hosts {
		inProse[h] = true
	}

	var missing, extra []string
	for _, h := range hosts {
		if !inCode[h] {
			missing = append(missing, h)
		}
	}
	for _, h := range allowedHosts {
		if !inProse[h] {
			extra = append(extra, h)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 || len(extra) > 0 {
		t.Fatalf("allowedHosts has drifted from %s\n  in the reference, not in the code: %v\n  in the code, not in the reference: %v",
			referencePath, missing, extra)
	}
}

func TestParseReferenceAllowlist(t *testing.T) {
	const bullet = "claims? This one needs the page, " + allowlistMarker +
		"\n  `a.test`, `b.test`,\n  `c.test`. A load-bearing citation to any OTHER host\n"

	hosts, err := parseReferenceAllowlist("intro\n- **Accuracy**: " + bullet + "- **Stability**: `master`\n")
	if err != nil {
		t.Fatalf("well-formed bullet did not parse: %v", err)
	}
	if strings.Join(hosts, ",") != "a.test,b.test,c.test" {
		t.Fatalf("parsed %v, want the three hosts of the bullet", hosts)
	}

	broken := map[string]string{
		"marker absent":     "- **Accuracy**: skim the target. `a.test`, `b.test`\n",
		"marker duplicated": bullet + bullet,
		"list absent":       "- **Accuracy**: " + allowlistMarker + " see above.\n",
		"single host":       "- **Accuracy**: " + allowlistMarker + " `a.test`. Prose.\n",
		"span unterminated": "- **Accuracy**: " + allowlistMarker + " `a.test`, `b.test\n",
	}
	for name, doc := range broken {
		t.Run(name, func(t *testing.T) {
			if got, err := parseReferenceAllowlist(doc); err == nil {
				t.Fatalf("parsed %v, want an error", got)
			}
		})
	}
}

// writeCheckout builds a checkout whose origin is remote, or one with no
// repository at all when remote is empty.
func writeCheckout(t *testing.T, remote string) string {
	t.Helper()
	dir := t.TempDir()
	if remote == "" {
		return dir
	}
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o700); err != nil {
		t.Fatalf("cannot create the checkout: %v", err)
	}
	config := "[core]\n\tbare = false\n[remote \"origin\"]\n\turl = " + remote +
		"\n\tfetch = +refs/heads/*:refs/remotes/origin/*\n"
	if err := os.WriteFile(filepath.Join(dir, ".git", "config"), []byte(config), 0o600); err != nil {
		t.Fatalf("cannot write the checkout config: %v", err)
	}
	return dir
}

func TestShouldGuard(t *testing.T) {
	rook := writeCheckout(t, "https://github.com/rook/rook.git")
	elsewhere := writeCheckout(t, "git@github.com:jhoblitt/rook-claude.git")

	tests := []struct {
		agentType string
		cwd       string
		want      bool
	}{
		{"rook-reviewer", rook, true},
		{"rook-triager", elsewhere, true},
		{"design-attacker", "", true},
		{"rook-maintainer:rook-reviewer", elsewhere, true},
		{"rook-maintainer:design-attacker", rook, true},

		{"general-purpose", rook, true},
		{"rook-maintainer:general-purpose", rook, true},
		{"general-purpose", elsewhere, false},

		{"", rook, false},
		{"Explore", rook, false},
		{"rook-maintainer:code-worker", rook, false},
		{"some-plugin:rook-reviewer-lookalike", rook, false},
		{"rook-maintainer:", rook, false},
	}
	for _, tc := range tests {
		if got := shouldGuard(tc.agentType, tc.cwd, guardedAgents); got != tc.want {
			t.Errorf("shouldGuard(%q, %q) = %v, want %v", tc.agentType, tc.cwd, got, tc.want)
		}
	}
}

func TestInRookCheckout(t *testing.T) {
	rook := writeCheckout(t, "https://github.com/rook/kubectl-rook-ceph")
	tests := []struct {
		name string
		dir  string
		want bool
	}{
		{"https remote", rook, true},
		{"nested directory", filepath.Join(rook, "a", "b"), true},
		{"scp-like remote", writeCheckout(t, "git@github.com:rook/rook.git"), true},
		{"ssh remote", writeCheckout(t, "ssh://git@github.com/rook/rook.git"), true},
		{"another org", writeCheckout(t, "https://github.com/jhoblitt/rook-claude.git"), false},
		{"lookalike host", writeCheckout(t, "https://evilgithub.com/rook/rook.git"), false},
		{"lookalike org", writeCheckout(t, "https://github.com/rook-ceph-evil/rook.git"), false},
		{"no repository", writeCheckout(t, ""), false},
		{"unknown directory", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := inRookCheckout(tc.dir); got != tc.want {
				t.Errorf("inRookCheckout(%q) = %v, want %v", tc.dir, got, tc.want)
			}
		})
	}
}

// A url outside a [remote] section is not a remote: an include path or a
// branch setting naming the rook repo must not read as a rook checkout.
func TestInRookCheckoutIgnoresNonRemoteSections(t *testing.T) {
	dir := writeCheckout(t, "https://github.com/jhoblitt/rook-claude.git")
	config := filepath.Join(dir, ".git", "config")
	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatalf("cannot read the checkout config: %v", err)
	}
	extended := string(data) + "[include]\n\tpath = /home/x/github.com/rook/rook/gitconfig\n"
	if err := os.WriteFile(config, []byte(extended), 0o600); err != nil {
		t.Fatalf("cannot extend the checkout config: %v", err)
	}
	if inRookCheckout(dir) {
		t.Fatal("a url outside a [remote] section counted as a remote")
	}
}

// A checkout whose config cannot be read leaves the question open, and the
// open question answers yes.
func TestInRookCheckoutFailsSafe(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git", "config"), 0o700); err != nil {
		t.Fatalf("cannot stage an unreadable config: %v", err)
	}
	if !inRookCheckout(dir) {
		t.Fatal("an unreadable config answered no")
	}
}

// The plugin's own convention puts review work in a linked worktree, whose
// .git is a file and whose config belongs to the clone it came from.
func TestInRookCheckoutResolvesLinkedWorktree(t *testing.T) {
	clone := writeCheckout(t, "https://github.com/rook/rook.git")
	gitDir := filepath.Join(clone, ".git", "worktrees", "review")
	if err := os.MkdirAll(gitDir, 0o700); err != nil {
		t.Fatalf("cannot stage the worktree git dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "commondir"), []byte("../..\n"), 0o600); err != nil {
		t.Fatalf("cannot write commondir: %v", err)
	}

	worktree := t.TempDir()
	if err := os.WriteFile(filepath.Join(worktree, ".git"),
		[]byte("gitdir: "+gitDir+"\n"), 0o600); err != nil {
		t.Fatalf("cannot write the worktree .git file: %v", err)
	}
	if !inRookCheckout(worktree) {
		t.Fatal("a linked worktree of a rook clone read as unguarded")
	}
}

func TestParseAllowExtra(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"   ", 0},
		{"a.test", 1},
		{"a.test,b.test", 2},
		{" a.test , b.test ", 2},
		{"a.test,,b.test,", 2},
	}
	for _, tc := range tests {
		if got := parseAllowExtra(tc.in); len(got) != tc.want {
			t.Errorf("parseAllowExtra(%q) = %v, want %d entries", tc.in, got, tc.want)
		}
	}
}

func TestHostAllowedRejectsEmptyEntries(t *testing.T) {
	if hostAllowed("evil.test", []string{""}) {
		t.Fatal("empty allowlist entry matched a host")
	}
	if hostAllowed("", allowedHosts) {
		t.Fatal("empty host matched")
	}
}

func TestSanitizeForMessage(t *testing.T) {
	if got := sanitizeForMessage("git\u200bhub.com"); got != "github.com" {
		t.Errorf("format character survived sanitize: %q", got)
	}
	long := make([]byte, 400)
	for i := range long {
		long[i] = 'a'
	}
	if got := sanitizeForMessage(string(long)); len(got) > 130 {
		t.Errorf("sanitize did not bound length: %d", len(got))
	}
}

// The cap counts bytes, so a host of multi-byte runes puts the cut inside a
// sequence: 118 ASCII bytes then three-byte runes lands it one byte in.
func TestSanitizeForMessageCutsOnARuneBoundary(t *testing.T) {
	got := sanitizeForMessage(strings.Repeat("a", 118) + strings.Repeat("€", 10))
	if !utf8.ValidString(got) {
		t.Errorf("truncation split a rune: %q", got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("truncation dropped its marker: %q", got)
	}
	if len(got) > 123 {
		t.Errorf("truncation did not bound length: %d", len(got))
	}
}

func TestHasHiddenRunes(t *testing.T) {
	clean := []string{"https://github.com/rook/rook", "plain", ""}
	for _, s := range clean {
		if hasHiddenRunes(s) {
			t.Errorf("hasHiddenRunes(%q) = true, want false", s)
		}
	}
	dirty := []string{"a\u200bb", "a\u0007b", "a\U000E0041b", "a\u202eb"}
	for _, s := range dirty {
		if !hasHiddenRunes(s) {
			t.Errorf("hasHiddenRunes(%q) = false, want true", s)
		}
	}
}

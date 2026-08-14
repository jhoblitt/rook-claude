package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
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

func TestShouldGuard(t *testing.T) {
	tests := []struct {
		agentType string
		want      bool
	}{
		{"rook-reviewer", true},
		{"rook-triager", true},
		{"design-attacker", true},
		{"rook-maintainer:rook-reviewer", true},
		{"rook-maintainer:design-attacker", true},

		{"", false},
		{"Explore", false},
		{"general-purpose", false},
		{"rook-maintainer:code-worker", false},
		{"some-plugin:rook-reviewer-lookalike", false},
		{"rook-maintainer:", false},
	}
	for _, tc := range tests {
		if got := shouldGuard(tc.agentType, guardedAgents); got != tc.want {
			t.Errorf("shouldGuard(%q) = %v, want %v", tc.agentType, got, tc.want)
		}
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

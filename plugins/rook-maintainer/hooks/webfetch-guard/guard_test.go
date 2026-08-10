package main

import "testing"

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
		{"dot-bounded subdomain", "https://raw.githubusercontent.com/rook/rook/master/README.md", false},
		{"uppercase host", "https://GitHub.com/rook/rook", false},
		{"explicit port", "https://github.com:443/rook/rook", false},
		{"trailing dot host", "https://github.com./rook/rook", false},

		{"unlisted host", "https://example.com/blog/post", true},
		{"lookalike suffix", "https://evilgithub.com/rook/rook", true},
		{"lookalike prefix", "https://github.com.evil.test/rook", true},
		{"userinfo smuggling", "https://github.com@evil.test/rook", true},
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

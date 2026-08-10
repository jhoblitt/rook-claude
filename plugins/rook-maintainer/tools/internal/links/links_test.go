package links

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Hostile codepoints appear only as \u escapes: a test for invisible-character
// handling that itself contains invisible characters cannot be reviewed by
// reading it.

func TestExtractURLs(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/x.md b/x.md",
		"--- a/x.md",
		"+++ b/x.md",
		"+See [docs](https://docs.ceph.com/en/squid/radosgw/).",
		"+Ref https://github.com/rook/rook/issues/1, and `https://tracker.ceph.com/issues/2`.",
		"-removed https://example.com/ignored",
		"+dup https://github.com/rook/rook/issues/1",
	}, "\n")

	got := ExtractURLs(diff)
	want := []string{
		"https://docs.ceph.com/en/squid/radosgw/",
		"https://github.com/rook/rook/issues/1",
		"https://tracker.ceph.com/issues/2",
	}
	if len(got) != len(want) {
		t.Fatalf("ExtractURLs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestExtractSkipsRemovedAndDedupes(t *testing.T) {
	if got := ExtractURLs("-https://example.com/gone"); len(got) != 0 {
		t.Errorf("removed line yielded %v", got)
	}
	if got := ExtractURLs("+++ b/f.md https://example.com/header"); len(got) != 0 {
		t.Errorf("+++ header yielded %v", got)
	}
}

func TestClassify(t *testing.T) {
	tests := []struct {
		name, original, final string
		status                int
		want                  string
	}{
		{"no redirect", "https://h/a/b", "https://h/a/b", 200, "ok"},
		{"same depth", "https://h/a/b", "https://h/a/c", 200, "redirect-ok"},
		{"http to https", "http://h/a/b", "https://h/a/b", 200, "redirect-ok"},
		{"collapse to root", "https://h/a/b", "https://h/", 200, "soft-404-suspect"},
		{"404 path segment", "https://h/a/b", "https://h/404", 200, "soft-404-suspect"},
		{"cross host shallower", "https://h/a/b/c", "https://other/x", 200, "soft-404-suspect"},
		{"same host shallower", "https://h/a/b/c", "https://h/a", 200, "redirect-ok"},
		{"dead", "https://h/a", "https://h/a", 404, "dead"},
		{"error", "https://h/a", "https://h/a", 0, "error"},
	}
	for _, tc := range tests {
		if got := Classify(tc.original, tc.final, tc.status); got != tc.want {
			t.Errorf("%s: Classify() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestHasHiddenRunes(t *testing.T) {
	for _, s := range []string{"https://github.com/rook/rook", "plain", ""} {
		if HasHiddenRunes(s) {
			t.Errorf("HasHiddenRunes(%q) = true", s)
		}
	}
	for _, s := range []string{"a\u200bb", "a\u0007b", "a\U000E0041b", "a\u202eb"} {
		if !HasHiddenRunes(s) {
			t.Errorf("HasHiddenRunes(%q) = false", s)
		}
	}
}

func TestSanitize(t *testing.T) {
	if got := Sanitize("git\u200bhub.com"); got != "github.com" {
		t.Errorf("Sanitize kept a format character: %q", got)
	}
	if got := Sanitize(strings.Repeat("a", 400)); len(got) > MaxURLChars+3 {
		t.Errorf("Sanitize did not bound length: %d", len(got))
	}
}

func TestProbeRejectsBeforeAnyRequest(t *testing.T) {
	p := NewProber(5, false)
	tests := []struct {
		name, url, verdict string
	}{
		{"hidden rune", "https://github\u200b.com/x", "suspicious"},
		{"non-http scheme", "file:///etc/passwd", "blocked"},
		{"loopback", "http://127.0.0.1:8731/admin", "blocked"},
		{"link-local metadata", "http://169.254.169.254/latest/meta-data/", "blocked"},
	}
	for _, tc := range tests {
		if got := p.Probe(context.Background(), tc.url); got.Verdict != tc.verdict {
			t.Errorf("%s: verdict = %q (note %q), want %q",
				tc.name, got.Verdict, got.Note, tc.verdict)
		}
	}
}

// The redirect ladder is the behavior most likely to regress, so it runs
// against a real server rather than a stubbed classifier.
func TestProbeFollowsRedirectsHopByHop(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/deep/page", redirectTo("/"))
	mux.HandleFunc("/deep/ok", redirectTo("/deep/other"))
	mux.HandleFunc("/deep/gone", redirectTo("/404"))
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	srv := httptest.NewServer(mux)
	defer srv.Close()

	p := NewProber(5, true)
	tests := []struct{ path, verdict string }{
		{"/deep/page", "soft-404-suspect"},
		{"/deep/ok", "redirect-ok"},
		{"/deep/gone", "soft-404-suspect"},
		{"/deep/plain", "ok"},
	}
	for _, tc := range tests {
		got := p.Probe(context.Background(), srv.URL+tc.path)
		if got.Verdict != tc.verdict {
			t.Errorf("%s: verdict = %q (status %d, final %q), want %q",
				tc.path, got.Verdict, got.Status, got.FinalURL, tc.verdict)
		}
	}
}

func TestProbeReportsDead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	got := NewProber(5, true).Probe(context.Background(), srv.URL+"/missing")
	if got.Verdict != "dead" || got.Status != 404 {
		t.Errorf("verdict=%q status=%d, want dead/404", got.Verdict, got.Status)
	}
	if !got.Bad() {
		t.Error("dead result did not report Bad()")
	}
}

func TestProbeFallsBackToGETWhenHEADRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	got := NewProber(5, true).Probe(context.Background(), srv.URL+"/x")
	if got.Verdict != "ok" {
		t.Errorf("verdict = %q (status %d), want ok", got.Verdict, got.Status)
	}
}

func TestCheckAllPreservesOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	urls := []string{srv.URL + "/a", srv.URL + "/b", srv.URL + "/c"}
	got := NewProber(5, true).CheckAll(context.Background(), urls, 3)
	for i, r := range got {
		if r.URL != urls[i] {
			t.Errorf("[%d] = %q, want %q", i, r.URL, urls[i])
		}
	}
}

func redirectTo(loc string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", loc)
		w.WriteHeader(http.StatusFound)
	}
}

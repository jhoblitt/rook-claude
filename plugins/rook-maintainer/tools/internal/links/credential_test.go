package links

import (
	"strings"
	"testing"
)

func TestCredentialBearing(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://user:hunter2@rgw.example.com/", true},
		{"https://rgw.example.com/b/o?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIA%2F20260813&X-Amz-Signature=abc123", true},
		{"https://rgw.example.com/o?AWSAccessKeyId=AKIAX&Signature=abc&Expires=1770000000", true},
		{"https://storage.example.com/c/b?sig=sv%3D2020-08-04", true},
		{"https://gitlab.example.com/api/v4/projects?private_token=glpat-xyz", true},
		{"https://app.example.com/cb#access_token=ya29.x&token_type=bearer", true},
		{"https://example.com/?SIGNATURE=abc", true},
		{"https://gitlab.example.com/p?private_token=glpat-xyz;other=1", true},
		{"https://rgw.example.com/o?X-Amz-Signature=abc%zz", true},
		{"https://app.example.com/cb#access_token=ya29.x%zz&state=1", true},
		{"https://user:pw@rgw.example.com/o#anchor%zz", true},
		{"https://docs.ceph.com/en/latest/", false},
		{"https://example.com/release?signed=false&version=3", false},
		{"https://example.com/rook-v1.18.0.tar.gz.sig", false},
	}
	for _, c := range cases {
		got, reason := CredentialBearing(c.url)
		if got != c.want {
			t.Errorf("CredentialBearing(%q) = %v (%s), want %v", c.url, got, reason, c.want)
		}
		if got && reason == "" {
			t.Errorf("CredentialBearing(%q): true with empty reason", c.url)
		}
	}
}

// A URL matching several credential parameters must name the same one every
// run: two audits of one diff have to produce identical report text.
func TestCredentialBearingReasonIsStable(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{
			"https://rgw.example.com/b/o?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIA%2F20260813&X-Amz-Signature=abc123",
			"query parameter X-Amz-Credential",
		},
		{
			"https://rgw.example.com/o?AWSAccessKeyId=AKIAX&Signature=abc&Expires=1770000000",
			"query parameter AWSAccessKeyId",
		},
		{
			"https://app.example.com/cb#access_token=ya29.x&id_token=eyJ.x&token_type=bearer",
			"fragment parameter access_token",
		},
	}
	for _, c := range cases {
		got, reason := CredentialBearing(c.url)
		if !got || reason != c.want {
			t.Errorf("CredentialBearing(%q) = %v, %q; want true, %q", c.url, got, reason, c.want)
		}
	}
}

func TestPartitionCredentialHiddenRunes(t *testing.T) {
	skips, probe := PartitionCredential([]string{
		"https://rgw.example.com/o?X-Amz-Signature=s3cr3t\u200bvalue",
	})
	if len(skips) != 1 || len(probe) != 0 {
		t.Fatalf("got %d skips, %d probe; want 1, 0", len(skips), len(probe))
	}
	if skips[0].Verdict != "suspicious" {
		t.Errorf("verdict = %q, want suspicious", skips[0].Verdict)
	}
	if !skips[0].Bad() {
		t.Error("a credential URL smuggling format characters must still gate")
	}
	if strings.Contains(skips[0].URL, "s3cr3t") {
		t.Errorf("URL %q leaks the signature the re-verdict to suspicious left behind", skips[0].URL)
	}
}

// check-links' stdout is read into the reviewing agent's context and travels
// from there into the review report, so a skip line names where the link is
// and never what it grants (security.md "Credential-finding precedence").
func TestPartitionCredentialRedactsCredential(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		note    string
		secrets []string
		keep    []string
	}{
		{
			name: "presigned URL matching three credential parameters at once",
			url: "https://rgw.example.com/bucket/key?X-Amz-Algorithm=AWS4-HMAC-SHA256" +
				"&X-Amz-Credential=AKIAEXAMPLE%2F20260813&X-Amz-Signature=9f8e7d6c" +
				"&X-Amz-Security-Token=FwoGZXIvYXdzEXAMPLE",
			note:    "credential-material URL (query parameter X-Amz-Credential): not probed",
			secrets: []string{"AKIAEXAMPLE", "9f8e7d6c", "FwoGZXIvYXdzEXAMPLE"},
			keep: []string{
				"rgw.example.com", "/bucket/key", "X-Amz-Algorithm=AWS4-HMAC-SHA256",
				"X-Amz-Credential", "X-Amz-Signature", "X-Amz-Security-Token",
			},
		},
		{
			name:    "userinfo",
			url:     "https://user:hunter2@rgw.example.com/bucket/key",
			note:    "credential-material URL (userinfo): not probed",
			secrets: []string{"hunter2"},
			keep:    []string{"rgw.example.com", "/bucket/key"},
		},
		{
			name:    "implicit-flow fragment",
			url:     "https://app.example.com/cb#access_token=ya29.EXAMPLESECRET&token_type=bearer",
			note:    "credential-material URL (fragment parameter access_token): not probed",
			secrets: []string{"ya29.EXAMPLESECRET"},
			keep:    []string{"app.example.com", "/cb", "access_token", "token_type=bearer"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			skips, probe := PartitionCredential([]string{c.url})
			if len(skips) != 1 || len(probe) != 0 {
				t.Fatalf("got %d skips, %d probe; want 1, 0", len(skips), len(probe))
			}
			got := skips[0]
			if got.Verdict != "skipped-credential" {
				t.Errorf("verdict = %q, want skipped-credential", got.Verdict)
			}
			if got.Note != c.note {
				t.Errorf("note = %q, want %q", got.Note, c.note)
			}
			for _, secret := range c.secrets {
				if strings.Contains(got.URL, secret) {
					t.Errorf("URL %q reships credential material %q", got.URL, secret)
				}
			}
			for _, keep := range c.keep {
				if !strings.Contains(got.URL, keep) {
					t.Errorf("URL %q dropped %q; a skip a reviewer cannot locate is not actionable",
						got.URL, keep)
				}
			}
		})
	}
}

// The shapes url.Parse and url.ParseQuery discard: ParseQuery drops any
// segment holding a ';' or an invalid percent-escape, and Parse rejects a URL
// outright for a bad escape in its fragment. Each one hid a credential
// parameter from the skip and sent the URL on to the prober, which spends the
// capability the whole pass exists to preserve.
func TestPartitionCredentialNeverProbesWhatParsingDrops(t *testing.T) {
	urls := []string{
		"https://gitlab.example.com/p?private_token=SECRET1;other=1",
		"https://rgw.example.com/o?X-Amz-Signature=SECRET2%zz",
		"https://app.example.com/cb#access_token=SECRET3%zz&state=1",
		"https://user:SECRET4@rgw.example.com/o#anchor%zz",
	}
	skips, probe := PartitionCredential(urls)
	if len(probe) != 0 {
		t.Fatalf("probe = %v; probing a credential URL exercises the capability", probe)
	}
	if len(skips) != len(urls) {
		t.Fatalf("got %d skips, want %d", len(skips), len(urls))
	}
	for i, skip := range skips {
		if skip.Verdict != "skipped-credential" {
			t.Errorf("[%d] verdict = %q, want skipped-credential", i, skip.Verdict)
		}
		for _, secret := range []string{"SECRET1", "SECRET2", "SECRET3", "SECRET4"} {
			if strings.Contains(skip.URL, secret) {
				t.Errorf("[%d] URL %q reships credential material %q", i, skip.URL, secret)
			}
		}
	}
}

// A redaction that cannot be proven complete emits no URL at all: the
// signature here is also a path segment, so the path carries the capability
// too and there is no non-secret half to report (security.md "What counts as
// a secret").
func TestPartitionCredentialWithholdsWhatItCannotRedact(t *testing.T) {
	skips, probe := PartitionCredential([]string{
		"https://example.com/9f8e7d6c?X-Amz-Signature=9f8e7d6c",
	})
	if len(skips) != 1 || len(probe) != 0 {
		t.Fatalf("got %d skips, %d probe; want 1, 0", len(skips), len(probe))
	}
	if skips[0].URL != "" {
		t.Errorf("URL = %q, want empty", skips[0].URL)
	}
	if skips[0].Note != "credential-material URL (query parameter X-Amz-Signature): not probed" {
		t.Errorf("note = %q; a withheld URL leaves the note as all the reviewer gets", skips[0].Note)
	}
}

func TestPartitionCredential(t *testing.T) {
	skips, probe := PartitionCredential([]string{
		"https://docs.ceph.com/en/latest/",
		"https://user:pw@h.example.com/",
	})
	if len(skips) != 1 || len(probe) != 1 {
		t.Fatalf("got %d skips, %d probe; want 1, 1", len(skips), len(probe))
	}
	if skips[0].Verdict != "skipped-credential" {
		t.Errorf("skip verdict = %q", skips[0].Verdict)
	}
	if skips[0].Bad() {
		t.Error("skipped-credential must not be Bad(): a skip is a report line, not a gate failure")
	}
	if probe[0] != "https://docs.ceph.com/en/latest/" {
		t.Errorf("probe[0] = %q", probe[0])
	}
}

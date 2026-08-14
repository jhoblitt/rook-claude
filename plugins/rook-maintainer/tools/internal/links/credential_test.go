package links

import "testing"

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
		"https://rgw.example.com/o?X-Amz-Signature=ab\u200bc",
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

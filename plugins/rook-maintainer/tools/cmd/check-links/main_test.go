package main

import (
	"strings"
	"testing"
)

// extract's listing lands in the same reviewer context an audit does, so a
// credential-bearing URL is redacted here too: a leak closed in one
// subcommand and left open in another reads as covered and is not
// (security.md "Credential-finding precedence").
func TestExtractRedactsCredentialURLs(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		verdict string
		secrets []string
		keep    []string
	}{
		{
			name: "presigned query",
			url: "https://rgw.example.com/bucket/key?X-Amz-Credential=AKIAEXAMPLE" +
				"&X-Amz-Signature=9f8e7d6c",
			verdict: "extracted",
			secrets: []string{"AKIAEXAMPLE", "9f8e7d6c"},
			keep:    []string{"rgw.example.com", "/bucket/key", "X-Amz-Signature"},
		},
		{
			name:    "userinfo",
			url:     "https://user:hunter2@rgw.example.com/bucket/key",
			verdict: "extracted",
			secrets: []string{"hunter2"},
			keep:    []string{"rgw.example.com", "/bucket/key"},
		},
		{
			name:    "fragment",
			url:     "https://app.example.com/cb#access_token=ya29.EXAMPLESECRET",
			verdict: "extracted",
			secrets: []string{"ya29.EXAMPLESECRET"},
			keep:    []string{"app.example.com", "/cb", "access_token"},
		},
		{
			name:    "semicolon segment url.ParseQuery drops",
			url:     "https://gitlab.example.com/p?private_token=glpat-SECRET;other=1",
			verdict: "extracted",
			secrets: []string{"glpat-SECRET"},
			keep:    []string{"gitlab.example.com", "/p", "private_token"},
		},
		{
			name:    "hidden runes inside a credential URL",
			url:     "https://rgw.example.com/o?X-Amz-Signature=s3cr3t\u200bvalue",
			verdict: "suspicious",
			secrets: []string{"s3cr3t"},
			keep:    []string{"rgw.example.com", "X-Amz-Signature"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extract([]string{c.url})
			if len(got) != 1 {
				t.Fatalf("got %d results, want 1", len(got))
			}
			if got[0].Verdict != c.verdict {
				t.Errorf("verdict = %q, want %q", got[0].Verdict, c.verdict)
			}
			for _, secret := range c.secrets {
				if strings.Contains(got[0].URL, secret) {
					t.Errorf("URL %q reships credential material %q", got[0].URL, secret)
				}
			}
			for _, keep := range c.keep {
				if !strings.Contains(got[0].URL, keep) {
					t.Errorf("URL %q dropped %q; the listing must stay actionable",
						got[0].URL, keep)
				}
			}
		})
	}
}

func TestExtractLeavesOrdinaryURLsAlone(t *testing.T) {
	urls := []string{"https://docs.ceph.com/en/latest/", "https://github.com/rook/rook/issues/1"}
	got := extract(urls)
	if len(got) != len(urls) {
		t.Fatalf("got %d results, want %d", len(got), len(urls))
	}
	for i, r := range got {
		if r.URL != urls[i] {
			t.Errorf("[%d] URL = %q, want %q", i, r.URL, urls[i])
		}
		if r.Verdict != "extracted" {
			t.Errorf("[%d] verdict = %q, want extracted", i, r.Verdict)
		}
	}
}

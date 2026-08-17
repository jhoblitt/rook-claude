package main

import (
	"os"
	"strings"
	"testing"
)

func withStdin(t *testing.T, payload string) {
	t.Helper()
	path := t.TempDir() + "/payload.json"
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("cannot stage the hook payload: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("cannot open the hook payload: %v", err)
	}
	old := os.Stdin
	os.Stdin = f
	t.Cleanup(func() {
		os.Stdin = old
		_ = f.Close()
	})
}

func captureStderr(t *testing.T) func() string {
	t.Helper()
	path := t.TempDir() + "/stderr"
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("cannot stage stderr: %v", err)
	}
	old := os.Stderr
	os.Stderr = f
	t.Cleanup(func() {
		os.Stderr = old
		_ = f.Close()
	})
	return func() string {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("cannot read captured stderr: %v", err)
		}
		return string(data)
	}
}

func payload(agent, url string) string {
	return `{"tool_name":"WebFetch","agent_type":"` + agent +
		`","tool_input":{"url":"` + url + `"}}`
}

const (
	allowedURL = "https://docs.ceph.com/en/squid/radosgw/"
	deniedURL  = "https://example.test/blog/post"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		allow   string
		payload string
		want    int
	}{
		{"kill switch", "off", "", payload("rook-reviewer", deniedURL), 0},
		{"unparsable payload", "", "", `{"tool_name":`, 0},
		{"another tool", "", "", `{"tool_name":"Read","tool_input":{"url":"` + deniedURL + `"}}`, 0},
		{"no url", "", "", payload("rook-reviewer", ""), 0},
		{"agent out of scope", "", "", payload("rook-maintainer:code-worker", deniedURL), 0},
		{"allowlisted host", "", "", payload("rook-reviewer", allowedURL), 0},
		{"namespaced agent", "", "", payload("rook-maintainer:design-attacker", deniedURL), 2},
		{"unlisted host", "", "", payload("rook-triager", deniedURL), 2},

		// `on` is the only thing that widens scope past guardedAgents.
		{"scope override guards any agent", "on", "", payload("rook-maintainer:code-worker", deniedURL), 2},
		{"scope override still allows the list", "on", "", payload("Explore", allowedURL), 0},

		{"host widened by env", "", "example.test", payload("rook-reviewer", deniedURL), 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ROOK_WEBFETCH_GUARD", tc.mode)
			t.Setenv("ROOK_WEBFETCH_ALLOW", tc.allow)
			withStdin(t, tc.payload)
			stderr := captureStderr(t)

			got := run()
			if got != tc.want {
				t.Errorf("run() = %d, want %d", got, tc.want)
			}
			out := stderr()
			if tc.want == 2 {
				if !strings.Contains(out, "BLOCKED") {
					t.Errorf("deny wrote no explanation to stderr: %q", out)
				}
			} else if out != "" {
				t.Errorf("allowed fetch wrote to stderr: %q", out)
			}
		})
	}
}

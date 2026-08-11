package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func areas(t *testing.T, stdin string, args ...string) (int, string) {
	t.Helper()
	var out bytes.Buffer
	code := dispatch(append([]string{"areas"}, args...), strings.NewReader(stdin), &out)
	return code, out.String()
}

func TestAreasClassifiesThePathSetAsAWhole(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"several paths union into one answer", []string{
			"Documentation/CRDs/cluster.md",
			"pkg/operator/ceph/csi/spec.go",
			"pkg/operator/ceph/object/rgw.go",
		}, "object\ncsi\ndocs\n"},
		{"one path can hit several areas", []string{
			"pkg/operator/ceph/object/multisite/zone.go",
		}, "object\nobject-multisite\n"},
		{"taxonomy order, not argument order", []string{
			"go.mod", "pkg/operator/ceph/cluster/mon/mon.go",
		}, "ceph-mon\nbuild\n"},
		{"the same area twice collapses", []string{
			"pkg/operator/ceph/csi/a.go", "pkg/operator/ceph/csi/b.go",
		}, "csi\n"},
		{"a path matching nothing is an empty answer, not an error", []string{
			"deploy/examples/cluster.yaml",
		}, ""},
		{"unbucketed paths do not suppress the ones that match", []string{
			"README.md", "deploy/charts/rook-ceph/values.yaml",
		}, "helm\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, got := areas(t, "", tc.args...)
			if code != 0 {
				t.Fatalf("exit = %d, want 0", code)
			}
			if got != tc.want {
				t.Errorf("output = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAreasReadsStdin(t *testing.T) {
	tests := []struct {
		name  string
		stdin string
		want  string
	}{
		{"one path per line", "pkg/operator/ceph/nfs/nfs.go\nDocumentation/x.md\n", "ceph-nfs\ndocs\n"},
		{"blank lines and surrounding space are not paths", "\n  pkg/operator/ceph/csi/spec.go  \n\n", "csi\n"},
		{"no trailing newline", "pkg/apis/ceph.rook.io/v1/types.go", "crd\n"},
		{"empty input matches nothing", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, got := areas(t, tc.stdin, "--stdin")
			if code != 0 {
				t.Fatalf("exit = %d, want 0", code)
			}
			if got != tc.want {
				t.Errorf("output = %q, want %q", got, tc.want)
			}
		})
	}
}

// The set semantic is the point: stdin and PATH arguments have to agree, since
// callers pipe `gh pr view --json files` into one and type the other.
func TestAreasStdinAndArgsAgree(t *testing.T) {
	paths := []string{"pkg/daemon/ceph/osd/volume.go", "tests/scripts/helper.sh", "somefile.txt"}
	_, fromArgs := areas(t, "", paths...)
	_, fromStdin := areas(t, strings.Join(paths, "\n")+"\n", "--stdin")
	if fromArgs != fromStdin {
		t.Errorf("PATH args gave %q, stdin gave %q", fromArgs, fromStdin)
	}
	if fromArgs != "ceph-osd\nci\n" {
		t.Errorf("output = %q, want %q", fromArgs, "ceph-osd\nci\n")
	}
}

func TestAreasUsageErrors(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		want  int
		quiet bool
	}{
		{name: "no paths and no --stdin", args: nil, want: 2},
		{name: "--stdin with paths", args: []string{"--stdin", "go.mod"}, want: 2},
		{name: "a flag after the paths", args: []string{"go.mod", "--stdin"}, want: 2},
		{name: "unknown flag", args: []string{"--union"}, want: 2},
		{name: "-h", args: []string{"-h"}, want: 0, quiet: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, out := withStderr(t, func() (int, string) { return areas(t, "", tc.args...) })
			if code != tc.want {
				t.Errorf("exit = %d, want %d", code, tc.want)
			}
			if tc.quiet && out != "" {
				t.Errorf("stdout = %q, want nothing", out)
			}
		})
	}
}

// The analysis run takes flags only, so no invocation predating the areas
// subcommand can be diverted by the dispatch in front of it.
func TestDispatchStillRunsTheAnalysis(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("rt_fetch_state.json", `{"pages_fetched": 1, "counted": 1, `+
		`"oldest_mergedat": "2026-07-01T00:00:00Z", "stop_reason": "reached window", `+
		`"errors": [], "truncations": []}`)
	write("rt_prs.jsonl", `{"number": 1, "title": "object: fix a zone", `+
		`"mergedAt": "2026-07-01T00:00:00Z", "author": {"login": "alice"}, `+
		`"files": {"nodes": [{"path": "pkg/operator/ceph/object/zone.go"}]}, `+
		`"reviews": {"nodes": [{"author": {"login": "bob"}}]}}`)

	args := []string{"--in-dir", dir, "--roster", "alice,bob", "--now", "2026-08-01T00:00:00Z"}
	code, _ := withStderr(t, func() (int, string) {
		return dispatch(args, strings.NewReader(""), io.Discard), ""
	})
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	final, err := os.ReadFile(filepath.Join(dir, "rt_final.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"title": "object: fix a zone"`, `"login": "bob"`, `"flags"`} {
		if !strings.Contains(string(final), want) {
			t.Errorf("rt_final.json is missing %s:\n%s", want, final)
		}
	}
}

func TestDispatchRejectsTheAnalysisWithoutInDir(t *testing.T) {
	code, _ := withStderr(t, func() (int, string) {
		return dispatch(nil, strings.NewReader(""), io.Discard), ""
	})
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
}

// The commands write diagnostics straight to os.Stderr; swallow them so a
// passing run stays quiet.
func withStderr(t *testing.T, f func() (int, string)) (int, string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stderr
	os.Stderr = w
	defer func() {
		os.Stderr = saved
		_ = r.Close()
	}()
	code, out := f()
	_ = w.Close()
	if _, err := io.Copy(io.Discard, r); err != nil {
		t.Fatal(err)
	}
	return code, out
}

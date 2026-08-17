package main

import "testing"

// SWEEP_DIR is spelled ahead of the flags in the skills and after them by
// habit; both have to reach the same pass.
func TestParseSweepArgsTakesSweepDirFromEitherEnd(t *testing.T) {
	for _, args := range [][]string{
		{"/s", "--root", "/rook"},
		{"--root", "/rook", "/s"},
	} {
		got, err := parseSweepArgs(args)
		if err != nil {
			t.Fatalf("parseSweepArgs(%v): %v", args, err)
		}
		if got.dir != "/s" || got.root != "/rook" {
			t.Errorf("parseSweepArgs(%v) = %+v, want dir /s and root /rook", args, got)
		}
	}
}

func TestParseSweepArgsDefaults(t *testing.T) {
	got, err := parseSweepArgs([]string{"/s"})
	if err != nil {
		t.Fatalf("parseSweepArgs: %v", err)
	}
	if got.jobs != defaultJobs {
		t.Errorf("jobs = %d, want %d", got.jobs, defaultJobs)
	}
	if defaultJobs != 4 {
		t.Errorf("defaultJobs = %d, want 4", defaultJobs)
	}
	if got.template != defaultTemplate {
		t.Errorf("template = %q, want the single-PR path's %q", got.template, defaultTemplate)
	}
	if got.includeDrafts {
		t.Error("drafts are in scope by default; they are assessed only when the user asks")
	}
}

func TestParseSweepArgsRejectsNonsense(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"no sweep dir", nil},
		{"a second positional", []string{"/s", "/t"}},
		{"an unknown flag", []string{"/s", "--verdict", "conforming"}},
		{"no worker at all", []string{"/s", "--jobs", "0"}},
		{"negative workers", []string{"/s", "--jobs", "-1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := parseSweepArgs(tc.args); err == nil {
				t.Errorf("parseSweepArgs(%v) = %+v, want an error", tc.args, got)
			}
		})
	}
}

// The subcommand must not disturb the flag-style invocation the single-PR
// reviewer path documents.
func TestRunKeepsTheSinglePRPath(t *testing.T) {
	if got := run([]string{"--self-test"}); got != 0 {
		t.Errorf("run(--self-test) = %d, want 0", got)
	}
	if got := run([]string{"--body", "/nonexistent/body.md"}); got != 2 {
		t.Errorf("run(--body missing) = %d, want 2", got)
	}
	if got := run([]string{"sweep"}); got != 2 {
		t.Errorf("run(sweep) with no SWEEP_DIR = %d, want 2", got)
	}
}

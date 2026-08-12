package main

import (
	"testing"

	"github.com/jhoblitt/rook-claude/plugins/rook-maintainer/tools/internal/rtfetch"
)

// ok is the default bound set, so each case below varies exactly one field.
func ok() rtfetch.Options {
	return rtfetch.Options{OutDir: "d", Months: 24, Cap: 4000, PageSize: 50, MaxPages: 400}
}

// A non-positive walk bound must be refused before the walk, not resolved into
// an empty fetch: rt_fetch_state.json is the assembler's provenance, and one
// written from a zero-PR walk reads exactly like a repo with no merged PRs.
// rtfetch.WindowCutoff permits a non-positive --months on purpose (Python
// parity) and names the caller as the place to reject it.
//
// These exercise checkBounds rather than run, deliberately: a bound that gets
// past the check starts a live GraphQL walk, so asserting through run would put
// a network call one regression away from a unit test.
func TestCheckBoundsRefusesAWalkThatCountsNothing(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
		mut  func(*rtfetch.Options)
	}{
		{"months zero", "--months", func(o *rtfetch.Options) { o.Months = 0 }},
		{"months negative", "--months", func(o *rtfetch.Options) { o.Months = -24 }},
		{"cap zero", "--cap", func(o *rtfetch.Options) { o.Cap = 0 }},
		{"cap negative", "--cap", func(o *rtfetch.Options) { o.Cap = -1 }},
		{"page size zero", "--page-size", func(o *rtfetch.Options) { o.PageSize = 0 }},
		{"max pages zero", "--max-pages", func(o *rtfetch.Options) { o.MaxPages = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := ok()
			tc.mut(&o)
			got, why := checkBounds(o)
			if got != tc.want {
				t.Errorf("checkBounds() = %q, want %q", got, tc.want)
			}
			if why == "" {
				t.Error("checkBounds() gave no reason; the message is what tells the caller which value is nonsense")
			}
		})
	}
}

// The shipped defaults must not trip the check, or every invocation fails.
func TestCheckBoundsAcceptsTheDefaults(t *testing.T) {
	if flag, _ := checkBounds(ok()); flag != "" {
		t.Errorf("checkBounds(defaults) rejected %s", flag)
	}
}

// --out-dir is checked first, so a caller missing everything hears about the
// required flag rather than a bound it never set.
func TestRunRequiresOutDirBeforeBounds(t *testing.T) {
	if got := run([]string{"--months", "-24"}); got != 2 {
		t.Errorf("run without --out-dir = %d, want 2", got)
	}
}

// run must map a bad bound to a usage exit, not fall through to the walk.
func TestRunRejectsABadBound(t *testing.T) {
	if got := run([]string{"--out-dir", "d", "--months", "-24"}); got != 2 {
		t.Errorf("run(--months -24) = %d, want 2", got)
	}
}

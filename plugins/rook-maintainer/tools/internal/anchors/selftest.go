package anchors

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

var selfTestDiff = strings.Join([]string{
	"diff --git a/pkg/keep.go b/pkg/keep.go",
	"--- a/pkg/keep.go",
	"+++ b/pkg/keep.go",
	"@@ -10,4 +10,5 @@ func Keep() {",
	" \tctx := context.TODO()",
	"-\told(ctx)",
	"+\tnewer(ctx)",
	"+\textra(ctx)",
	" }",
	"diff --git a/build/gone.mk b/build/gone.mk",
	"--- a/build/gone.mk",
	"+++ /dev/null",
	"@@ -1,2 +0,0 @@",
	"-all:",
	"-\techo hi",
	"",
}, "\n")

// SelfTest verifies the parser against a diff with the shapes that decide
// every rule, so the gate can be trusted without a network or a checkout.
func SelfTest() error {
	files := Commentable(selfTestDiff)
	if got, want := sorted(slices.Collect(maps.Keys(files))), []string{"build/gone.mk", "pkg/keep.go"}; !slices.Equal(got, want) {
		return fmt.Errorf("paths = %v, want %v", got, want)
	}

	// Header context line 10 / removal 11 on LEFT; additions 11-12 on RIGHT.
	for _, tc := range []struct {
		path, side string
		want       []int64
	}{
		{"pkg/keep.go", Right, []int64{10, 11, 12, 13}},
		{"pkg/keep.go", Left, []int64{10, 11, 12}},
		// A file deleted outright anchors LEFT-only, at its original path.
		{"build/gone.mk", Right, nil},
		{"build/gone.mk", Left, []int64{1, 2}},
	} {
		set := files[tc.path].Right
		if tc.side == Left {
			set = files[tc.path].Left
		}
		if got := sorted(slices.Collect(maps.Keys(set))); !slices.Equal(got, tc.want) {
			return fmt.Errorf("%s %s = %v, want %v", tc.path, tc.side, got, tc.want)
		}
	}

	ok, err := ParseReview([]byte(`{"comments": [
		{"path": "pkg/keep.go", "line": 11, "side": "RIGHT"},
		{"path": "build/gone.mk", "start_line": 1, "start_side": "LEFT", "line": 2, "side": "LEFT"}
	]}`))
	if err != nil {
		return err
	}
	if problems := Validate(ok, files); len(problems) != 0 {
		return fmt.Errorf("postable payload rejected: %v", problems)
	}

	for _, tc := range []struct{ comment, needle string }{
		{`{"path": "pkg/keep.go", "line": 99, "side": "RIGHT"}`, "outside the diff"},
		{`{"path": "build/gone.mk", "line": 1, "side": "RIGHT"}`, "wrong side?"},
		{`{"path": "pkg/absent.go", "line": 1, "side": "RIGHT"}`, "not in the diff"},
		{`{"path": "pkg/keep.go", "line": 11, "side": "RIGHT", "start_line": 10}`, "BOTH"},
		{`{"path": "pkg/keep.go", "line": 11, "side": "RIGHT", "start_line": 10, "start_side": "LEFT"}`, "must equal"},
	} {
		review, err := ParseReview([]byte(`{"comments": [` + tc.comment + `]}`))
		if err != nil {
			return err
		}
		problems := Validate(review, files)
		if len(problems) != 1 || !strings.Contains(problems[0], tc.needle) {
			return fmt.Errorf("%s: problems = %v, want one containing %q", tc.comment, problems, tc.needle)
		}
	}
	return nil
}

func sorted[T interface{ ~string | ~int64 }](s []T) []T {
	slices.Sort(s)
	return s
}

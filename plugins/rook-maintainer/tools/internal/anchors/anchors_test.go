package anchors

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
)

// want describes one file's commentable lines in the order a reader of a diff
// would name them: what the ORIGINAL file admits, then what the NEW one does.
type want struct {
	left  []int64
	right []int64
}

func diffOf(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

func got(files Files) map[string]want {
	out := map[string]want{}
	for path, sides := range files {
		out[path] = want{
			left:  slices.Sorted(maps.Keys(sides.Left)),
			right: slices.Sorted(maps.Keys(sides.Right)),
		}
	}
	return out
}

func assertFiles(t *testing.T, diff string, expect map[string]want) {
	t.Helper()
	have := got(Commentable(diff))
	if len(have) != len(expect) {
		t.Fatalf("paths = %v, want %v", slices.Sorted(maps.Keys(have)), slices.Sorted(maps.Keys(expect)))
	}
	for path, w := range expect {
		h, ok := have[path]
		if !ok {
			t.Fatalf("path %q missing; got %v", path, slices.Sorted(maps.Keys(have)))
		}
		if !slices.Equal(h.left, w.left) {
			t.Errorf("%s LEFT = %v, want %v", path, h.left, w.left)
		}
		if !slices.Equal(h.right, w.right) {
			t.Errorf("%s RIGHT = %v, want %v", path, h.right, w.right)
		}
	}
}

func TestCommentable(t *testing.T) {
	tests := []struct {
		name   string
		diff   string
		expect map[string]want
	}{
		{
			name: "added removed and context",
			diff: diffOf(
				"diff --git a/pkg/keep.go b/pkg/keep.go",
				"index 1111111..2222222 100644",
				"--- a/pkg/keep.go",
				"+++ b/pkg/keep.go",
				"@@ -10,4 +10,5 @@ func Keep() {",
				" ctx := context.TODO()",
				"-old(ctx)",
				"+newer(ctx)",
				"+extra(ctx)",
				" }",
			),
			expect: map[string]want{
				"pkg/keep.go": {left: []int64{10, 11, 12}, right: []int64{10, 11, 12, 13}},
			},
		},
		{
			name: "new file has no LEFT side",
			diff: diffOf(
				"diff --git a/new.go b/new.go",
				"new file mode 100644",
				"index 0000000..3333333",
				"--- /dev/null",
				"+++ b/new.go",
				"@@ -0,0 +1,3 @@",
				"+package main",
				"+",
				"+func main() {}",
			),
			expect: map[string]want{"new.go": {right: []int64{1, 2, 3}}},
		},
		{
			name: "deleted file anchors LEFT at its original path",
			diff: diffOf(
				"diff --git a/build/gone.mk b/build/gone.mk",
				"deleted file mode 100644",
				"--- a/build/gone.mk",
				"+++ /dev/null",
				"@@ -1,2 +0,0 @@",
				"-all:",
				"-\techo hi",
			),
			expect: map[string]want{"build/gone.mk": {left: []int64{1, 2}}},
		},
		{
			name: "several hunks in one file",
			diff: diffOf(
				"diff --git a/a.go b/a.go",
				"--- a/a.go",
				"+++ b/a.go",
				"@@ -1,2 +1,2 @@",
				"-one",
				"+uno",
				" two",
				"@@ -40,2 +40,3 @@",
				" forty",
				"+forty-and-a-half",
				" forty-one",
			),
			expect: map[string]want{
				"a.go": {left: []int64{1, 2, 40, 41}, right: []int64{1, 2, 40, 41, 42}},
			},
		},
		{
			name: "several files in one diff",
			diff: diffOf(
				"diff --git a/a.go b/a.go",
				"--- a/a.go",
				"+++ b/a.go",
				"@@ -1 +1 @@",
				"-one",
				"+uno",
				"diff --git a/b.go b/b.go",
				"--- a/b.go",
				"+++ b/b.go",
				"@@ -7 +7 @@",
				"-seven",
				"+sette",
			),
			expect: map[string]want{
				"a.go": {left: []int64{1}, right: []int64{1}},
				"b.go": {left: []int64{7}, right: []int64{7}},
			},
		},
		{
			name: "hunk header without a length",
			diff: diffOf(
				"--- a/a.go",
				"+++ b/a.go",
				"@@ -5 +5 @@ func F() {",
				"-five",
				"+cinque",
			),
			expect: map[string]want{"a.go": {left: []int64{5}, right: []int64{5}}},
		},
		{
			name: "no-newline marker advances neither counter",
			diff: diffOf(
				"--- a/a.txt",
				"+++ b/a.txt",
				"@@ -1,1 +1,2 @@",
				"-a",
				"\\ No newline at end of file",
				"+a",
				"+b",
			),
			expect: map[string]want{"a.txt": {left: []int64{1}, right: []int64{1, 2}}},
		},
		{
			name: "wholly empty line is an empty context line",
			diff: diffOf(
				"--- a/a.txt",
				"+++ b/a.txt",
				"@@ -1,3 +1,3 @@",
				" one",
				"",
				"-three",
				"+tre",
			),
			expect: map[string]want{"a.txt": {left: []int64{1, 2, 3}, right: []int64{1, 2, 3}}},
		},
		{
			name: "trailing newline yields no phantom line",
			diff: "--- a/a.txt\n+++ b/a.txt\n@@ -1,1 +1,1 @@\n-a\n+b\n",
			expect: map[string]want{
				"a.txt": {left: []int64{1}, right: []int64{1}},
			},
		},
		{
			name: "missing trailing newline parses the same",
			diff: "--- a/a.txt\n+++ b/a.txt\n@@ -1,1 +1,1 @@\n-a\n+b",
			expect: map[string]want{
				"a.txt": {left: []int64{1}, right: []int64{1}},
			},
		},
		{
			name: "CRLF diff parses like an LF one",
			diff: "--- a/a.txt\r\n+++ b/a.txt\r\n@@ -1,2 +1,2 @@\r\n one\r\n\r\n-two\r\n+due\r\n",
			expect: map[string]want{
				"a.txt": {left: []int64{1, 2, 3}, right: []int64{1, 2, 3}},
			},
		},
		{
			name: "an unknown marker ends the hunk",
			diff: diffOf(
				"--- a/a.go",
				"+++ b/a.go",
				"@@ -1,1 +1,1 @@",
				"-a",
				"+b",
				"-- ",
				"2.39.1",
				"+not part of the diff",
			),
			expect: map[string]want{"a.go": {left: []int64{1, 2}, right: []int64{1}}},
		},
		{
			name: "binary file gets no entry",
			diff: diffOf(
				"diff --git a/img.png b/img.png",
				"index 1111111..2222222 100644",
				"Binary files a/img.png and b/img.png differ",
			),
			expect: map[string]want{},
		},
		{
			name: "timestamped headers and unprefixed paths",
			diff: diffOf(
				"--- old.txt\t2026-01-01 00:00:00.000000000 +0000",
				"+++ new.txt\t2026-01-02 00:00:00.000000000 +0000",
				"@@ -1 +1 @@",
				"-a",
				"+b",
			),
			expect: map[string]want{"new.txt": {left: []int64{1}, right: []int64{1}}},
		},
		{
			name: "a real directory named a/ survives the prefix strip",
			diff: diffOf(
				"diff --git a/a/b/c.go b/a/b/c.go",
				"--- a/a/b/c.go",
				"+++ b/a/b/c.go",
				"@@ -1 +1 @@",
				"-a",
				"+b",
			),
			expect: map[string]want{"a/b/c.go": {left: []int64{1}, right: []int64{1}}},
		},
		{
			name: "lines before any header are ignored",
			diff: diffOf(
				"@@ -1,1 +1,1 @@",
				"-orphan",
				"+orphan",
			),
			expect: map[string]want{},
		},
		{
			name: "a rename with no hunk still registers the new path",
			diff: diffOf(
				"diff --git a/old.go b/new.go",
				"similarity index 100%",
				"rename from old.go",
				"rename to new.go",
				"--- a/old.go",
				"+++ b/new.go",
			),
			expect: map[string]want{"new.go": {}},
		},
		{"empty input", "", map[string]want{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertFiles(t, tc.diff, tc.expect)
		})
	}
}

func TestStripPrefix(t *testing.T) {
	tests := []struct {
		spec string
		path string
		ok   bool
	}{
		{"a/pkg/x.go", "pkg/x.go", true},
		{"b/pkg/x.go", "pkg/x.go", true},
		{"/dev/null", "", false},
		{"pkg/x.go", "pkg/x.go", true},
		{"a/x.go\t2026-01-01 00:00:00", "x.go", true},
		{"/dev/null\t1970-01-01", "", false},
		{"c/x.go", "c/x.go", true},
		{"a/", "a/", true},
		{"a/b", "b", true},
		{"a/bc", "bc", true},
		{"", "", true},
	}
	for _, tc := range tests {
		path, ok := stripPrefix(tc.spec)
		if path != tc.path || ok != tc.ok {
			t.Errorf("stripPrefix(%q) = (%q, %v), want (%q, %v)", tc.spec, path, ok, tc.path, tc.ok)
		}
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a\n", []string{"a"}},
		{"a\n\n", []string{"a", ""}},
		{"\n", []string{""}},
		{"a\r\nb\r\n", []string{"a", "b"}},
		{"a\nb", []string{"a", "b"}},
	}
	for _, tc := range tests {
		if got := splitLines(tc.in); !slices.Equal(got, tc.want) {
			t.Errorf("splitLines(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// validateDiff exercises every shape the rules turn on: a context line
// (commentable on both sides), an addition (RIGHT only), a removal (LEFT
// only), and a file deleted outright (LEFT only, at its original path).
var validateDiff = diffOf(
	"diff --git a/pkg/keep.go b/pkg/keep.go",
	"--- a/pkg/keep.go",
	"+++ b/pkg/keep.go",
	"@@ -10,4 +10,5 @@ func Keep() {",
	" ctx := context.TODO()",
	"-old(ctx)",
	"+newer(ctx)",
	"+extra(ctx)",
	" }",
	"diff --git a/build/gone.mk b/build/gone.mk",
	"--- a/build/gone.mk",
	"+++ /dev/null",
	"@@ -1,2 +0,0 @@",
	"-all:",
	"-\techo hi",
)

func TestValidate(t *testing.T) {
	files := Commentable(validateDiff)

	tests := []struct {
		name    string
		payload string
		want    []string
	}{
		{
			name:    "added line on RIGHT",
			payload: `{"comments": [{"path": "pkg/keep.go", "line": 11, "side": "RIGHT"}]}`,
		},
		{
			name:    "context line on LEFT",
			payload: `{"comments": [{"path": "pkg/keep.go", "line": 10, "side": "LEFT"}]}`,
		},
		{
			name:    "side defaults to RIGHT when absent",
			payload: `{"comments": [{"path": "pkg/keep.go", "line": 12}]}`,
		},
		{
			name:    "multi-line LEFT anchor on a deleted file",
			payload: `{"comments": [{"path": "build/gone.mk", "start_line": 1, "start_side": "LEFT", "line": 2, "side": "LEFT"}]}`,
		},
		{
			name:    "start_line equal to line",
			payload: `{"comments": [{"path": "pkg/keep.go", "start_line": 11, "start_side": "RIGHT", "line": 11, "side": "RIGHT"}]}`,
		},
		{
			name:    "null start keys are absent keys",
			payload: `{"comments": [{"path": "pkg/keep.go", "line": 11, "start_line": null, "start_side": null}]}`,
		},
		{
			name:    "no comments key",
			payload: `{"body": "x"}`,
		},
		{
			name:    "null comments",
			payload: `{"comments": null}`,
		},
		{
			name:    "empty comments",
			payload: `{"comments": []}`,
		},
		{
			name:    "line outside the diff",
			payload: `{"comments": [{"path": "pkg/keep.go", "line": 99, "side": "RIGHT"}]}`,
			want:    []string{"comments[0] pkg/keep.go:99 RIGHT: line is outside the diff"},
		},
		{
			name:    "removal is not on RIGHT",
			payload: `{"comments": [{"path": "build/gone.mk", "line": 1, "side": "RIGHT"}]}`,
			want: []string{"comments[0] build/gone.mk:1 RIGHT: line is outside the diff" +
				" (it IS commentable on LEFT \u2014 wrong side?)"},
		},
		{
			name:    "addition is not on LEFT",
			payload: `{"comments": [{"path": "pkg/keep.go", "line": 13, "side": "LEFT"}]}`,
			want: []string{"comments[0] pkg/keep.go:13 LEFT: line is outside the diff" +
				" (it IS commentable on RIGHT \u2014 wrong side?)"},
		},
		{
			name:    "file not in the diff",
			payload: `{"comments": [{"path": "pkg/absent.go", "line": 1, "side": "RIGHT"}]}`,
			want:    []string{"comments[0] pkg/absent.go:1: file is not in the diff"},
		},
		{
			name:    "missing path",
			payload: `{"comments": [{"line": 11, "side": "RIGHT"}]}`,
			want:    []string{"comments[0]: missing `path`"},
		},
		{
			name:    "empty path",
			payload: `{"comments": [{"path": "", "line": 11}]}`,
			want:    []string{"comments[0]: missing `path`"},
		},
		{
			name:    "null path",
			payload: `{"comments": [{"path": null, "line": 11}]}`,
			want:    []string{"comments[0]: missing `path`"},
		},
		{
			name:    "missing line",
			payload: `{"comments": [{"path": "pkg/keep.go"}]}`,
			want:    []string{"comments[0] pkg/keep.go: `line` must be an integer"},
		},
		{
			name:    "string line",
			payload: `{"comments": [{"path": "pkg/keep.go", "line": "11"}]}`,
			want:    []string{"comments[0] pkg/keep.go: `line` must be an integer"},
		},
		{
			name:    "float line",
			payload: `{"comments": [{"path": "pkg/keep.go", "line": 11.0}]}`,
			want:    []string{"comments[0] pkg/keep.go: `line` must be an integer"},
		},
		{
			name:    "null side",
			payload: `{"comments": [{"path": "pkg/keep.go", "line": 11, "side": null}]}`,
			want:    []string{"comments[0] pkg/keep.go:11: `side` must be LEFT or RIGHT, got None"},
		},
		{
			name:    "lowercase side",
			payload: `{"comments": [{"path": "pkg/keep.go", "line": 11, "side": "right"}]}`,
			want:    []string{"comments[0] pkg/keep.go:11: `side` must be LEFT or RIGHT, got 'right'"},
		},
		{
			name:    "numeric side",
			payload: `{"comments": [{"path": "pkg/keep.go", "line": 11, "side": 1}]}`,
			want:    []string{"comments[0] pkg/keep.go:11: `side` must be LEFT or RIGHT, got 1"},
		},
		{
			name:    "start_line without start_side",
			payload: `{"comments": [{"path": "pkg/keep.go", "line": 11, "side": "RIGHT", "start_line": 10}]}`,
			want: []string{"comments[0] pkg/keep.go:11: multi-line anchors need BOTH" +
				" `start_line` and `start_side`"},
		},
		{
			name:    "start_side without start_line",
			payload: `{"comments": [{"path": "pkg/keep.go", "line": 11, "side": "RIGHT", "start_side": "RIGHT"}]}`,
			want: []string{"comments[0] pkg/keep.go:11: multi-line anchors need BOTH" +
				" `start_line` and `start_side`"},
		},
		{
			name:    "start_side disagrees with side",
			payload: `{"comments": [{"path": "pkg/keep.go", "line": 11, "side": "RIGHT", "start_line": 10, "start_side": "LEFT"}]}`,
			want:    []string{"comments[0] pkg/keep.go:11: `start_side` (LEFT) must equal `side` (RIGHT)"},
		},
		{
			name:    "start_line after line",
			payload: `{"comments": [{"path": "pkg/keep.go", "line": 11, "side": "RIGHT", "start_line": 12, "start_side": "RIGHT"}]}`,
			want:    []string{"comments[0] pkg/keep.go:11: `start_line` (12) must be an integer <= `line`"},
		},
		{
			name:    "float start_line",
			payload: `{"comments": [{"path": "pkg/keep.go", "line": 11, "side": "RIGHT", "start_line": 10.5, "start_side": "RIGHT"}]}`,
			want:    []string{"comments[0] pkg/keep.go:11: `start_line` (10.5) must be an integer <= `line`"},
		},
		{
			name:    "start_line outside the diff",
			payload: `{"comments": [{"path": "pkg/keep.go", "line": 11, "side": "RIGHT", "start_line": 1, "start_side": "RIGHT"}]}`,
			want:    []string{"comments[0] pkg/keep.go:1 RIGHT: start line is outside the diff"},
		},
		{
			name:    "comment is not an object",
			payload: `{"comments": ["nope"]}`,
			want:    []string{"comments[0]: not an object"},
		},
		{
			name:    "comments is not a list",
			payload: `{"comments": {"path": "pkg/keep.go"}}`,
			want:    []string{"review payload: `comments` must be a list"},
		},
		{
			name:    "top level is not an object",
			payload: `[{"path": "pkg/keep.go", "line": 11}]`,
			want:    []string{"review payload: top level must be an object"},
		},
		{
			name: "one problem per comment, indexed",
			payload: `{"comments": [
				{"path": "pkg/keep.go", "line": 11, "side": "RIGHT"},
				{"path": "pkg/keep.go", "line": 99, "side": "RIGHT"},
				{"path": "nope.go", "line": 1},
				{"path": "pkg/keep.go", "line": 13, "side": "LEFT"}
			]}`,
			want: []string{
				"comments[1] pkg/keep.go:99 RIGHT: line is outside the diff",
				"comments[2] nope.go:1: file is not in the diff",
				"comments[3] pkg/keep.go:13 LEFT: line is outside the diff" +
					" (it IS commentable on RIGHT \u2014 wrong side?)",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			review, err := ParseReview([]byte(tc.payload))
			if err != nil {
				t.Fatalf("ParseReview: %v", err)
			}
			problems := Validate(review, files)
			if !slices.Equal(problems, tc.want) {
				t.Errorf("Validate() = %#v, want %#v", problems, tc.want)
			}
		})
	}
}

func TestCount(t *testing.T) {
	tests := []struct {
		payload string
		want    int
	}{
		{`{}`, 0},
		{`{"comments": null}`, 0},
		{`{"comments": []}`, 0},
		{`{"comments": [{}, {}, {}]}`, 3},
		{`{"comments": "abc"}`, 3},
		{`{"comments": {"a": 1, "b": 2}}`, 2},
		{`[]`, 0},
	}
	for _, tc := range tests {
		review, err := ParseReview([]byte(tc.payload))
		if err != nil {
			t.Fatalf("ParseReview(%s): %v", tc.payload, err)
		}
		if got := Count(review); got != tc.want {
			t.Errorf("Count(%s) = %d, want %d", tc.payload, got, tc.want)
		}
	}
}

func TestParseReview(t *testing.T) {
	if _, err := ParseReview([]byte(`{"comments": []} {"comments": []}`)); err == nil {
		t.Error("trailing JSON value accepted")
	}
	if _, err := ParseReview([]byte(`{"comments": []}` + "\n\n")); err != nil {
		t.Errorf("trailing whitespace rejected: %v", err)
	}
	if _, err := ParseReview(nil); err == nil {
		t.Error("empty input accepted")
	}
	review, err := ParseReview([]byte(`{"comments": [{"line": 11}]}`))
	if err != nil {
		t.Fatal(err)
	}
	c := review.(map[string]any)["comments"].([]any)[0].(map[string]any)
	if _, ok := asInt(c["line"]); !ok {
		t.Errorf("line decoded as %T, want an integer", c["line"])
	}
}

func TestSelfTest(t *testing.T) {
	if err := SelfTest(); err != nil {
		t.Fatalf("SelfTest() = %v", err)
	}
}

// A path GitHub could never anchor must never be looked up as one: a non-string
// path that stringifies onto a real diff path would otherwise validate.
func TestNonStringPathIsNeverInTheDiff(t *testing.T) {
	files := Commentable(diffOf(
		"--- a/42",
		"+++ b/42",
		"@@ -1 +1 @@",
		"-a",
		"+b",
	))
	review, err := ParseReview([]byte(`{"comments": [{"path": 42, "line": 1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"comments[0] 42:1: file is not in the diff"}
	if problems := Validate(review, files); !slices.Equal(problems, want) {
		t.Errorf("Validate() = %#v, want %#v", problems, want)
	}
}

func TestHugeLineNumberStaysUnpostable(t *testing.T) {
	files := Commentable(validateDiff)
	payload := fmt.Sprintf(`{"comments": [{"path": "pkg/keep.go", "line": %s}]}`,
		"99999999999999999999")
	review, err := ParseReview([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}
	problems := Validate(review, files)
	if len(problems) != 1 || !strings.Contains(problems[0], "line is outside the diff") {
		t.Errorf("Validate() = %#v", problems)
	}
}

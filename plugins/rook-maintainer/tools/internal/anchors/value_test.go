package anchors

import (
	"encoding/json"
	"testing"
)

// Hostile codepoints appear only as \u escapes: a test for invisible-character
// handling that itself contains invisible characters cannot be reviewed by
// reading it.

func TestTruthy(t *testing.T) {
	tests := []struct {
		payload string
		want    bool
	}{
		{`null`, false},
		{`false`, false},
		{`true`, true},
		{`0`, false},
		{`0.0`, false},
		{`-0`, false},
		{`1`, true},
		{`-1`, true},
		{`""`, false},
		{`"x"`, true},
		{`[]`, false},
		{`[null]`, true},
		{`{}`, false},
		{`{"a": 1}`, true},
	}
	for _, tc := range tests {
		v := decode(t, tc.payload)
		if got := truthy(v); got != tc.want {
			t.Errorf("truthy(%s) = %v, want %v", tc.payload, got, tc.want)
		}
	}
}

func TestAsInt(t *testing.T) {
	tests := []struct {
		payload string
		want    int64
		ok      bool
	}{
		{`11`, 11, true},
		{`0`, 0, true},
		{`-3`, -3, true},
		{`11.0`, 0, false},
		{`1e2`, 0, false},
		{`1E2`, 0, false},
		{`"11"`, 0, false},
		{`true`, 0, false},
		{`null`, 0, false},
		{`[11]`, 0, false},
	}
	for _, tc := range tests {
		got, ok := asInt(decode(t, tc.payload))
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("asInt(%s) = (%d, %v), want (%d, %v)", tc.payload, got, ok, tc.want, tc.ok)
		}
	}
}

func TestPyStr(t *testing.T) {
	tests := []struct{ payload, want string }{
		{`null`, "None"},
		{`true`, "True"},
		{`false`, "False"},
		{`"LEFT"`, "LEFT"},
		{`11`, "11"},
		{`-0`, "0"},
		{`10.5`, "10.5"},
		{`11.0`, "11.0"},
		{`0.0`, "0.0"},
		{`1e2`, "100.0"},
		{`1e16`, "1e+16"},
		{`1e-7`, "1e-07"},
		{`99999999999999999999`, "99999999999999999999"},
	}
	for _, tc := range tests {
		if got := pyStr(decode(t, tc.payload)); got != tc.want {
			t.Errorf("pyStr(%s) = %q, want %q", tc.payload, got, tc.want)
		}
	}
}

func TestPyRepr(t *testing.T) {
	tests := []struct{ payload, want string }{
		{`null`, "None"},
		{`true`, "True"},
		{`1`, "1"},
		{`"right"`, "'right'"},
		{`"it's"`, `"it's"`},
		{`"say \"hi\""`, `'say "hi"'`},
		{`"a\u0007b"`, `'a\x07b'`},
		{`"a\u200bb"`, `'a\u200bb'`},
		{`"a\nb"`, `'a\nb'`},
		{`["LEFT", 1]`, "['LEFT', 1]"},
		{`{"b": 2, "a": 1}`, "{'a': 1, 'b': 2}"},
	}
	for _, tc := range tests {
		if got := pyRepr(decode(t, tc.payload)); got != tc.want {
			t.Errorf("pyRepr(%s) = %q, want %q", tc.payload, got, tc.want)
		}
	}
}

// A side that smuggles an escape sequence must not reach the terminal intact:
// the message quoting it is printed for a maintainer to read.
func TestPyReprEscapesTerminalControls(t *testing.T) {
	got := pyRepr("\u001b[2J\u0007")
	if want := `'\x1b[2J\x07'`; got != want {
		t.Errorf("pyRepr() = %q, want %q", got, want)
	}
}

func TestPyLen(t *testing.T) {
	tests := []struct {
		payload string
		want    int
	}{
		{`"abc"`, 3},
		{`"\u00e9\u00e9"`, 2},
		{`[1, 2]`, 2},
		{`{"a": 1}`, 1},
		{`7`, 0},
	}
	for _, tc := range tests {
		if got := pyLen(decode(t, tc.payload)); got != tc.want {
			t.Errorf("pyLen(%s) = %d, want %d", tc.payload, got, tc.want)
		}
	}
}

func decode(t *testing.T, payload string) any {
	t.Helper()
	v, err := ParseReview([]byte(payload))
	if err != nil {
		t.Fatalf("ParseReview(%s): %v", payload, err)
	}
	return v
}

func TestParseReviewKeepsNumbersExact(t *testing.T) {
	v := decode(t, `{"line": 11}`)
	n, ok := v.(map[string]any)["line"].(json.Number)
	if !ok {
		t.Fatalf("line decoded as %T, want json.Number", v.(map[string]any)["line"])
	}
	if n.String() != "11" {
		t.Errorf("line = %q, want %q", n.String(), "11")
	}
}

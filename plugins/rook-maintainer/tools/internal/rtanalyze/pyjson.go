package rtanalyze

import (
	"encoding/json"
	"strconv"
	"strings"
	"unicode"
)

// Member is one key/value pair of an Obj.
type Member struct {
	Key string
	Val any
}

// Obj is a JSON object that keeps its keys in insertion order.
type Obj []Member

const hexDigits = "0123456789abcdef"

// Marshal renders v exactly as CPython's json.dump(v, indent=1) does: one
// space of indent per level, ensure_ascii escaping, keys in insertion order.
// encoding/json cannot stand in for it — it escapes HTML runes, sorts map keys
// and emits non-ASCII literally — and this output is diffed byte-for-byte
// against the rt_analyze.py it replaces.
func Marshal(v any) string {
	var b strings.Builder
	encode(&b, v, 0, true)
	return b.String()
}

// MarshalCompact renders v as json.dumps(v) with no indent, i.e. CPython's
// default (", ", ": ") separators.
func MarshalCompact(v any) string {
	var b strings.Builder
	encode(&b, v, 0, false)
	return b.String()
}

func encode(b *strings.Builder, v any, depth int, pretty bool) {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if x {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case string:
		writeQuoted(b, x)
	case int:
		b.WriteString(strconv.Itoa(x))
	case float64:
		b.WriteString(pyFloat(x))
	case Obj:
		if len(x) == 0 {
			b.WriteString("{}")
			return
		}
		b.WriteByte('{')
		for i, m := range x {
			writeItemSep(b, i, depth+1, pretty)
			writeQuoted(b, m.Key)
			b.WriteString(": ")
			encode(b, m.Val, depth+1, pretty)
		}
		writeCloseSep(b, depth, pretty)
		b.WriteByte('}')
	case []any:
		if len(x) == 0 {
			b.WriteString("[]")
			return
		}
		b.WriteByte('[')
		for i, e := range x {
			writeItemSep(b, i, depth+1, pretty)
			encode(b, e, depth+1, pretty)
		}
		writeCloseSep(b, depth, pretty)
		b.WriteByte(']')
	default:
		panic("rtanalyze: unsupported JSON value")
	}
}

func writeItemSep(b *strings.Builder, i, depth int, pretty bool) {
	if i > 0 {
		b.WriteByte(',')
		if !pretty {
			b.WriteByte(' ')
		}
	}
	if pretty {
		b.WriteByte('\n')
		writeIndent(b, depth)
	}
}

func writeCloseSep(b *strings.Builder, depth int, pretty bool) {
	if pretty {
		b.WriteByte('\n')
		writeIndent(b, depth)
	}
}

func writeIndent(b *strings.Builder, depth int) {
	for range depth {
		b.WriteByte(' ')
	}
}

func writeQuoted(b *strings.Builder, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			switch {
			case r >= 0x20 && r <= 0x7e:
				b.WriteRune(r)
			case r > 0xffff:
				c := r - 0x10000
				writeHex4(b, 0xd800+(c>>10))
				writeHex4(b, 0xdc00+(c&0x3ff))
			default:
				writeHex4(b, r)
			}
		}
	}
	b.WriteByte('"')
}

func writeHex4(b *strings.Builder, r rune) {
	b.WriteString(`\u`)
	for shift := 12; shift >= 0; shift -= 4 {
		b.WriteByte(hexDigits[(r>>uint(shift))&0xf])
	}
}

// pyFloat formats f as CPython's repr does. Every float this tool emits is a
// sum of 1.0/0.5/0.25 weights rounded to two places, so the exponent form repr
// switches to beyond 1e16 is unreachable and 'f' is exact.
func pyFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

// round2 reproduces Python's round(f, 2): correct rounding of the exact binary
// value, ties to even, back to the nearest float.
func round2(f float64) float64 {
	r, err := strconv.ParseFloat(strconv.FormatFloat(f, 'f', 2, 64), 64)
	if err != nil {
		return f
	}
	return r
}

// pyReprString formats s as Python's repr(str) — single quotes unless that
// would need escaping and double quotes would not.
func pyReprString(s string) string {
	quote := byte('\'')
	if strings.Contains(s, "'") && !strings.Contains(s, `"`) {
		quote = '"'
	}
	var b strings.Builder
	b.WriteByte(quote)
	for _, r := range s {
		switch {
		case r == rune(quote) || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			b.WriteString(`\x`)
			b.WriteByte(hexDigits[(r>>4)&0xf])
			b.WriteByte(hexDigits[r&0xf])
		case r < 0x7f:
			b.WriteRune(r)
		case unicode.IsPrint(r):
			b.WriteRune(r)
		case r < 0x100:
			b.WriteString(`\x`)
			b.WriteByte(hexDigits[(r>>4)&0xf])
			b.WriteByte(hexDigits[r&0xf])
		case r < 0x10000:
			writeHex4(&b, r)
		default:
			b.WriteString(`\U`)
			for shift := 28; shift >= 0; shift -= 4 {
				b.WriteByte(hexDigits[(r>>uint(shift))&0xf])
			}
		}
	}
	b.WriteByte(quote)
	return b.String()
}

// pyReprInts formats a []int as Python's repr(list).
func pyReprInts(v []int) string {
	parts := make([]string, len(v))
	for i, n := range v {
		parts[i] = strconv.Itoa(n)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// pyReprStrings formats a []string as Python's repr(list).
func pyReprStrings(v []string) string {
	parts := make([]string, len(v))
	for i, s := range v {
		parts[i] = pyReprString(s)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

type countedType struct {
	name string
	n    int
}

// pyReprCounts formats an insertion-ordered counter as repr(dict(Counter)).
func pyReprCounts(counts []countedType) string {
	parts := make([]string, len(counts))
	for i, c := range counts {
		parts[i] = pyReprString(c.name) + ": " + strconv.Itoa(c.n)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// pyStrNumber renders a JSON number the way str() renders the int or float
// json.load produced from it; a missing or null value prints as None.
//
// Known, accepted divergence: Python ints are unbounded, so a count past int64
// falls through to float64 here and prints rounded. rt-fetch cannot count that
// high on a 64-bit platform; only a hand-edited state file reaches it.
func pyStrNumber(n *json.Number) string {
	if n == nil {
		return "None"
	}
	s := n.String()
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return strconv.FormatInt(i, 10)
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return pyFloat(f)
	}
	return s
}

// Package untrusted wraps contributor-authored text on its way into a model's
// context.
//
// Spec: skills/rook-conventions/SKILL.md, "Read content is untrusted data",
// which states the marker pair, the fresh-token rule and where the
// treat-as-data line goes. Nothing here restates it.
//
// Callers: cmd/validate-kb (its problem list) and internal/rtanalyze (the flag
// brief rt-analyze hands to the resolver).
package untrusted

import (
	"crypto/rand"
	"strings"
)

// Fence returns note, then body between markers carrying a token drawn fresh
// and absent from both. The two are separate arguments so a caller cannot put
// its treat-as-data line inside the fence, which would leave the instruction as
// forgeable as the data it describes.
//
// Bounding stays with the caller: it holds the individual items and knows where
// each one ends, and a fence around an unbounded body is still unbounded.
func Fence(note, body string) string {
	token := Token(note + "\x00" + body)
	return note + "\n<<<UNTRUSTED-" + token + "\n" + body + "\n" + token + "-UNTRUSTED>>>\n"
}

// Token draws a token that appears nowhere in body.
func Token(body string) string {
	for {
		if t := rand.Text(); !strings.Contains(body, t) {
			return t
		}
	}
}

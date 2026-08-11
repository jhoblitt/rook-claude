package main

import (
	"net/url"
	"strings"
	"unicode"
)

// allowedHosts is the trusted-source list from
// skills/rook-code-review/references/docs-sync.md. Keep the two in sync: the
// reference file is what the reviewer reads, this is what actually holds.
var allowedHosts = []string{
	"github.com",
	"githubusercontent.com",
	"docs.ceph.com",
	"tracker.ceph.com",
	"kubernetes.io",
	"pkg.go.dev",
	"rfc-editor.org",
}

// guardedAgents are the plugin's injection-exposed subagents: the three that
// carry WebFetch AND ingest attacker-authored PR bodies, issue text and diffs.
// Nothing else is guarded, because nothing else has an adversary choosing its
// URLs — a maintainer researching a Kubernetes blog post in the main session
// is not under attack, and breaking that fetch would be pure cost.
var guardedAgents = []string{
	"rook-reviewer",
	"rook-triager",
	"design-attacker",
}

type decision struct {
	deny   bool
	reason string
}

// shouldGuard reports whether a call from agentType is in scope. The plugin
// namespace is stripped first: agents arrive as "rook-maintainer:rook-reviewer"
// when namespaced and bare when not, and both name the same agent.
func shouldGuard(agentType string, agents []string) bool {
	name := agentType
	if _, after, found := strings.Cut(agentType, ":"); found {
		name = after
	}
	if name == "" {
		return false
	}
	for _, agent := range agents {
		if name == agent {
			return true
		}
	}
	return false
}

// hasHiddenRunes reports control, format, private-use and surrogate
// codepoints. Format (Cf) is the one that matters and the one url.Parse
// tolerates: U+200B and the U+E0020 tag block are invisible, so a host that
// renders as docs.ceph.com can carry anything. Checked against the raw
// string, before parsing normalizes it away.
func hasHiddenRunes(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Cc, r) ||
			unicode.Is(unicode.Co, r) || unicode.Is(unicode.Cs, r) {
			return true
		}
	}
	return false
}

// hostAllowed matches an entry exactly or as a dot-bounded parent domain. The
// dot is the whole point: a plain suffix test lets evilgithub.com through on
// the github.com entry.
func hostAllowed(host string, allow []string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if host == "" {
		return false
	}
	for _, entry := range allow {
		entry = strings.ToLower(strings.TrimSuffix(entry, "."))
		if entry == "" {
			continue
		}
		if host == entry || strings.HasSuffix(host, "."+entry) {
			return true
		}
	}
	return false
}

// parseAllowExtra reads ROOK_WEBFETCH_ALLOW. The environment is the
// maintainer's, not the reviewed PR's, so widening here is a deliberate act
// and not an injection surface.
func parseAllowExtra(env string) []string {
	var out []string
	for _, part := range strings.Split(env, ",") {
		if host := strings.TrimSpace(part); host != "" {
			out = append(out, host)
		}
	}
	return out
}

// evaluate decides one WebFetch. It fails CLOSED, departing from the pr-gate
// doctrine in skill-review-claude on purpose: a review gate that breaks should
// not block the maintainer's work, but an allowlist that breaks and lets the
// fetch through is not an allowlist. The fail-OPEN cases live in main.go and
// are exactly those the reviewed PR cannot influence — a malformed hook
// payload is a harness change, not an attack.
func evaluate(raw string, allow []string) decision {
	if hasHiddenRunes(raw) {
		return decision{true, "URL contains control or format characters — " +
			"invisible codepoints in a link are ASCII smuggling, not a typo. " +
			"Report it as security/suspicious-content."}
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return decision{true, "URL does not parse."}
	}
	if parsed.Scheme != "https" {
		return decision{true, "scheme is " + schemeLabel(parsed.Scheme) +
			", not https. Page content entering review context must arrive " +
			"over an authenticated channel; a non-https citation in a rook " +
			"diff is itself worth a finding."}
	}
	host := parsed.Hostname()
	if !hostAllowed(host, allow) {
		return decision{true, "host " + hostLabel(host) + " is not an " +
			"allowlisted content source."}
	}
	return decision{}
}

func schemeLabel(scheme string) string {
	if scheme == "" {
		return "absent"
	}
	return sanitizeForMessage(scheme)
}

func hostLabel(host string) string {
	if host == "" {
		return "(none)"
	}
	return sanitizeForMessage(host)
}

// sanitizeForMessage bounds attacker-chosen text on its way to the transcript.
// The URL came from the diff, so the deny reason is the one place this hook
// echoes untrusted input.
func sanitizeForMessage(s string) string {
	const limit = 120
	var b strings.Builder
	for _, r := range s {
		if unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Cc, r) ||
			unicode.Is(unicode.Co, r) || unicode.Is(unicode.Cs, r) {
			continue
		}
		b.WriteRune(r)
	}
	out := b.String()
	if len(out) > limit {
		return out[:limit] + "..."
	}
	return out
}

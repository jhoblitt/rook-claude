package main

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// allowedHosts is the trusted-source list from
// skills/rook-code-review/references/docs-sync.md, which is what the reviewer
// reads while this is what actually holds. TestAllowedHostsMatchesReference
// enforces the equality; entries are narrow because hostAllowed also admits
// every subdomain of one.
var allowedHosts = []string{
	"github.com",
	"raw.githubusercontent.com",
	"docs.ceph.com",
	"tracker.ceph.com",
	"kubernetes.io",
	"pkg.go.dev",
	"rfc-editor.org",
}

// guardedAgents are the plugin's injection-exposed subagents: the three that
// carry WebFetch AND ingest attacker-authored PR bodies, issue text and diffs.
// Only these and the fallback below are guarded, because nothing else has an
// adversary choosing its URLs — a maintainer researching a Kubernetes blog post
// in the main session is not under attack, and breaking that fetch would be
// pure cost.
var guardedAgents = []string{
	"rook-reviewer",
	"rook-triager",
	"design-attacker",
}

// fallbackAgent is what a typed agent degrades to when it is unavailable. It
// reads the same attacker-authored diffs with a wider tool roster, so a prose
// brief telling it to refuse its own WebFetch is a rule enforced by the thing
// being injected. Its name is generic, though, and denying every fetch it makes
// anywhere would break ordinary research — so it is in scope only where it
// stands in for a review, which is inside a rook checkout.
const fallbackAgent = "general-purpose"

// rookRemote matches a remote URL naming a repository under the rook org, in
// either the https or the scp-like form. The character before the host is what
// keeps evilgithub.com out.
var rookRemote = regexp.MustCompile(`(?i)(?:^|[@/])github\.com[:/]rook/`)

type decision struct {
	deny   bool
	reason string
}

// shouldGuard reports whether a call from agentType in cwd is in scope. The
// plugin namespace is stripped first: agents arrive as
// "rook-maintainer:rook-reviewer" when namespaced and bare when not, and both
// name the same agent.
func shouldGuard(agentType, cwd string, agents []string) bool {
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
	return name == fallbackAgent && inRookCheckout(cwd)
}

// inRookCheckout reports whether dir sits in a clone of a rook repository.
//
// Direction on failure: knowing a repository is NOT rook's takes evidence, so
// only evidence answers no — a config that says so, or no repository at all.
// Anything that leaves the question open (no directory to look at, a checkout
// whose config cannot be read) answers yes, because the cost of the two
// mistakes is not symmetric: an over-guarded fetch prints a message naming
// ROOK_WEBFETCH_ALLOW and ROOK_WEBFETCH_GUARD, while an under-guarded one has
// already pulled attacker-chosen content into review context.
func inRookCheckout(dir string) bool {
	if dir == "" {
		return true
	}
	config, inRepo := gitConfigPath(dir)
	if !inRepo {
		return false
	}
	urls, err := remoteURLs(config)
	if err != nil {
		return true
	}
	for _, u := range urls {
		if rookRemote.MatchString(u) {
			return true
		}
	}
	return false
}

// gitConfigPath walks up from dir to the config of the repository owning it,
// reporting whether dir is in a repository at all. A linked worktree's .git is
// a file naming its git dir, and that git dir's commondir points at the config
// the whole clone shares — the plugin's own convention puts review work in
// worktrees, so resolving both is the common case, not the exotic one. An
// unreadable .git file yields an empty path, which reads as the error that
// inRookCheckout answers yes to.
func gitConfigPath(dir string) (string, bool) {
	for {
		git := filepath.Join(dir, ".git")
		if info, err := os.Stat(git); err == nil {
			if !info.IsDir() {
				if git = gitDirFromFile(git); git == "" {
					return "", true
				}
			}
			return filepath.Join(commonDir(git), "config"), true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func gitDirFromFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	target := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(data)), "gitdir:"))
	if target == "" {
		return ""
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(path), target)
	}
	return target
}

func commonDir(gitDir string) string {
	data, err := os.ReadFile(filepath.Join(gitDir, "commondir"))
	if err != nil {
		return gitDir
	}
	common := strings.TrimSpace(string(data))
	if common == "" {
		return gitDir
	}
	if !filepath.IsAbs(common) {
		common = filepath.Join(gitDir, common)
	}
	return common
}

// remoteURLs reads the url of every remote in a git config. Section-aware on
// purpose: a url outside a [remote] section is not a remote, and a substring
// search over the whole file would take an include path or a branch setting
// for one.
func remoteURLs(config string) ([]string, error) {
	data, err := os.ReadFile(config)
	if err != nil {
		return nil, err
	}
	var urls []string
	remote := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			remote = strings.HasPrefix(strings.ToLower(line), "[remote")
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !remote || !found || strings.TrimSpace(strings.ToLower(key)) != "url" {
			continue
		}
		urls = append(urls, strings.TrimSpace(value))
	}
	return urls, nil
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

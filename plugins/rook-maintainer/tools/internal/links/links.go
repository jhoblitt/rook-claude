// Package links probes URLs a rook diff adds or edits for liveness only.
//
// Why this exists as a tool and not a WebFetch: liveness needs an HTTP status,
// not page content. Probing an attacker-chosen URL with a content-returning
// tool pulls untrusted bytes into reviewer context and spends one human
// approval per link, which trains the reviewer to click through the prompt
// that matters. Everything here returns a status code, a redirect chain and a
// verdict from a fixed vocabulary — never a byte of response body, and no
// response header except Location. That output contract, not the sandbox
// network allowlist, is what makes it safe to aim at hosts the diff chose.
//
// Redirects are followed one hop at a time so the non-public-address check
// re-runs on every hop; letting the client follow them would resolve
// intermediate hops unsupervised, and a public URL that redirects to
// 169.254.169.254 is the cheap attack.
package links

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	MaxURLChars    = 300
	MaxHops        = 4
	DefaultWorkers = 8
	userAgent      = "rook-review-linkcheck"
)

var (
	urlRe         = regexp.MustCompile(`https?://[^\s<>"'` + "`" + `\\|]+`)
	trailingPunct = ".,;:!?'\"`*_"
	soft404Paths  = map[string]bool{
		"404": true, "notfound": true, "not-found": true,
		"error": true, "pagenotfound": true,
	}
	getFallback = map[int]bool{403: true, 405: true, 501: true}
)

type Result struct {
	URL      string `json:"url"`
	Verdict  string `json:"verdict"`
	Status   int    `json:"status"`
	FinalURL string `json:"final_url,omitempty"`
	Hops     int    `json:"hops"`
	Note     string `json:"note,omitempty"`
}

// Bad reports whether a verdict should fail the run.
func (r Result) Bad() bool {
	switch r.Verdict {
	case "dead", "soft-404-suspect", "suspicious", "error":
		return true
	}
	return false
}

func hidden(r rune) bool {
	return unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Cc, r) ||
		unicode.Is(unicode.Co, r) || unicode.Is(unicode.Cs, r)
}

// HasHiddenRunes reports control, format, private-use and surrogate
// codepoints. Format (Cf) is the one that matters and the one url.Parse
// tolerates: U+200B and the U+E0020 tag block are invisible, so a link that
// renders as docs.ceph.com can point anywhere.
func HasHiddenRunes(s string) bool {
	return strings.ContainsFunc(s, hidden)
}

// Sanitize bounds attacker-chosen text on its way into a report.
func Sanitize(s string) string {
	out := strings.Map(func(r rune) rune {
		if hidden(r) {
			return -1
		}
		return r
	}, s)
	if len(out) > MaxURLChars {
		return out[:MaxURLChars] + "..."
	}
	return out
}

func trimUnbalanced(u string) string {
	for _, pair := range []struct{ open, close string }{{"(", ")"}, {"[", "]"}, {"{", "}"}} {
		for strings.HasSuffix(u, pair.close) &&
			strings.Count(u, pair.open) < strings.Count(u, pair.close) {
			u = u[:len(u)-1]
		}
	}
	return u
}

// ExtractURLs pulls URLs from the added lines of a unified diff. Removed lines
// are skipped: a link this change deletes is not this change's problem.
func ExtractURLs(diff string) []string {
	seen := map[string]bool{}
	var found []string
	for _, line := range strings.Split(diff, "\n") {
		if !strings.HasPrefix(line, "+") || strings.HasPrefix(line, "+++") {
			continue
		}
		for _, m := range urlRe.FindAllString(line[1:], -1) {
			u := trimUnbalanced(strings.TrimRight(m, trailingPunct))
			u = strings.TrimRight(u, trailingPunct)
			if u != "" && !seen[u] {
				seen[u] = true
				found = append(found, u)
			}
		}
	}
	return found
}

// NonPublicAddress resolves host and reports a reason if any address it
// resolves to is not publicly routable. DNS rebinding can beat this between
// the check and the request, but against a probe that returns only a status
// code that buys an attacker nothing.
func NonPublicAddress(host string) string {
	ips, err := net.LookupIP(host)
	if err != nil {
		return "dns-failure"
	}
	for _, ip := range ips {
		if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() ||
			ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
			ip.IsInterfaceLocalMulticast() {
			return "non-public-address"
		}
	}
	return ""
}

// Classify decides the verdict for a URL that resolved.
//
// Liberal by design: rook docs redirect legitimately and constantly
// (docs.ceph.com release paths, GitHub repo renames), so only a redirect that
// COLLAPSES specificity is called a soft 404.
func Classify(original, final string, status int) string {
	switch {
	case status == 0:
		return "error"
	case status >= 300:
		return "dead"
	case final == original:
		return "ok"
	}
	src, err1 := url.Parse(original)
	dst, err2 := url.Parse(final)
	if err1 != nil || err2 != nil {
		return "redirect-ok"
	}
	srcDepth := segments(src.Path)
	dstDepth := segments(dst.Path)
	if len(srcDepth) > 0 && len(dstDepth) == 0 {
		return "soft-404-suspect"
	}
	for _, seg := range dstDepth {
		if soft404Paths[strings.ToLower(seg)] {
			return "soft-404-suspect"
		}
	}
	if src.Host != dst.Host && len(dstDepth) < len(srcDepth) {
		return "soft-404-suspect"
	}
	return "redirect-ok"
}

func segments(p string) []string {
	var out []string
	for _, s := range strings.Split(p, "/") {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

type Prober struct {
	Client       *http.Client
	AllowPrivate bool
}

func durationSeconds(n int) time.Duration {
	if n <= 0 {
		return 10 * time.Second
	}
	return time.Duration(n) * time.Second
}

func NewProber(timeout int, allowPrivate bool) *Prober {
	return &Prober{
		Client: &http.Client{
			// Manual hop-by-hop following; see the package comment.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Timeout: durationSeconds(timeout),
		},
		AllowPrivate: allowPrivate,
	}
}

func (p *Prober) Probe(ctx context.Context, raw string) Result {
	if HasHiddenRunes(raw) {
		return Result{URL: SafeURL(raw), Verdict: "suspicious",
			Note: "control or format characters inside URL"}
	}
	// PartitionCredential is the caller's gate, but this one is exported and
	// an exercised capability is unrecoverable, so the refusal lives here too.
	if bearing, reason := CredentialBearing(raw); bearing {
		return Result{URL: SafeURL(raw), Verdict: "skipped-credential",
			Note: credentialNote(reason)}
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return Result{URL: SafeURL(raw), Verdict: "blocked", Note: "URL does not parse"}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return Result{URL: SafeURL(raw), Verdict: "blocked", Note: "non-http scheme"}
	}

	current, status, hops := raw, 0, 0
	for hops <= MaxHops {
		cur, err := url.Parse(current)
		if err != nil || cur.Hostname() == "" {
			return Result{URL: SafeURL(raw), Verdict: "blocked", Hops: hops, Note: "no host"}
		}
		if !p.AllowPrivate {
			if reason := NonPublicAddress(cur.Hostname()); reason != "" {
				return Result{URL: SafeURL(raw), Verdict: "blocked", Hops: hops, Note: reason}
			}
		}
		var location string
		status, location, err = p.head(ctx, current)
		if err != nil {
			return Result{URL: SafeURL(raw), Verdict: "error", Hops: hops, Note: "request failed"}
		}
		if !isRedirect(status) || location == "" {
			break
		}
		if HasHiddenRunes(location) {
			return Result{URL: SafeURL(raw), Verdict: "suspicious", Status: status, Hops: hops,
				Note: "control or format characters in Location"}
		}
		next, err := cur.Parse(location)
		if err != nil {
			return Result{URL: SafeURL(raw), Verdict: "error", Status: status, Hops: hops,
				Note: "unparsable Location"}
		}
		// The hop target is server-chosen, so an implicit-flow callback or a
		// presigned object can arrive here; following it spends the capability
		// just as probing one from the diff would.
		if bearing, reason := CredentialBearing(next.String()); bearing {
			return Result{URL: SafeURL(raw), Verdict: "skipped-credential", Status: status,
				Hops: hops, FinalURL: SafeURL(next.String()),
				Note: "redirect to credential-material URL (" + reason + "): not followed"}
		}
		current, hops = next.String(), hops+1
	}
	if hops > MaxHops {
		return Result{URL: SafeURL(raw), Verdict: "dead", Status: status, Hops: hops,
			Note: "redirect limit exceeded"}
	}

	res := Result{URL: SafeURL(raw), Verdict: Classify(raw, current, status),
		Status: status, Hops: hops}
	if current != raw {
		res.FinalURL = SafeURL(current)
	}
	return res
}

// head probes with HEAD, retrying as GET where a server rejects HEAD. The body
// is never read in either case.
func (p *Prober) head(ctx context.Context, target string) (int, string, error) {
	status, location, err := p.do(ctx, http.MethodHead, target)
	if err == nil && getFallback[status] {
		return p.do(ctx, http.MethodGet, target)
	}
	return status, location, err
}

func (p *Prober) do(ctx context.Context, method, target string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, method, target, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := p.Client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode, resp.Header.Get("Location"), nil
}

func isRedirect(status int) bool {
	switch status {
	case 301, 302, 303, 307, 308:
		return true
	}
	return false
}

// CheckAll probes urls concurrently, preserving input order.
func (p *Prober) CheckAll(ctx context.Context, urls []string, workers int) []Result {
	if workers < 1 {
		workers = 1
	}
	results := make([]Result, len(urls))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i, u := range urls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = p.Probe(ctx, u)
		}()
	}
	wg.Wait()
	return results
}

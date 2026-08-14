package links

import (
	"net/url"
	"slices"
	"strings"
)

// credentialParams are query/fragment keys whose presence marks a URL as
// carrying credential material (security.md "What counts as a secret").
// Deliberately overcautious: a skipped probe costs one reviewer judgment,
// an exercised capability is unrecoverable. Shapes this list cannot see —
// a bare capability path segment, a public integrity signature parameter —
// are the reviewer's judgment per docs-sync.md.
var credentialParams = map[string]bool{
	"x-amz-signature": true, "x-amz-credential": true,
	"x-amz-security-token": true, "awsaccesskeyid": true,
	"signature": true, "x-goog-signature": true, "sig": true,
	"access_token": true, "id_token": true, "token": true,
	"api_key": true, "apikey": true, "private_token": true,
}

// credentialName decodes a key as written in a query or fragment and reports
// whether it names credential material.
func credentialName(rawKey string) (string, bool) {
	name, err := url.QueryUnescape(rawKey)
	if err != nil {
		name = rawKey
	}
	return name, credentialParams[strings.ToLower(name)]
}

// credentialKey returns the credential-bearing key of an &-joined key=value
// string, lowest-sorting rather than first-seen: a presigned URL matches
// several of these at once, and two audits of one diff must produce identical
// report text.
//
// The text is walked here rather than handed to url.ParseQuery, which DROPS
// any segment holding a ';' or an invalid percent-escape: `?private_token=t;x=1`
// parsed as empty, so the URL missed the skip and went to the prober, which
// spends the capability this pass exists to preserve.
func credentialKey(s string) (string, bool) {
	var found []string
	for _, pair := range strings.Split(s, "&") {
		rawKey, _, _ := strings.Cut(pair, "=")
		if name, ok := credentialName(rawKey); ok {
			found = append(found, name)
		}
	}
	if len(found) == 0 {
		return "", false
	}
	return slices.Min(found), true
}

// credentialFragmentKey matches a fragment as written and unescaped: url.Parse
// hands the redactor the unescaped form, so a pair boundary smuggled in as
// %26 has to be visible to both or the two disagree about what is in there.
func credentialFragmentKey(fragment string) (string, bool) {
	if key, ok := credentialKey(fragment); ok {
		return key, true
	}
	decoded, err := url.QueryUnescape(fragment)
	if err != nil {
		return "", false
	}
	return credentialKey(decoded)
}

// queryAndFragment splits a URL as RFC 3986 does — fragment from the first
// '#', query from the first '?' before it — without parsing it: url.Parse
// rejects a URL outright for a bad escape in its fragment, and a URL this
// check cannot see into is one that reaches the prober.
func queryAndFragment(raw string) (string, string) {
	var fragment string
	if i := strings.Index(raw, "#"); i >= 0 {
		raw, fragment = raw[:i], raw[i+1:]
	}
	var query string
	if i := strings.Index(raw, "?"); i >= 0 {
		query = raw[i+1:]
	}
	return query, fragment
}

func hasUserinfo(raw string) bool {
	_, rest, ok := strings.Cut(raw, "//")
	if !ok {
		return false
	}
	if i := strings.IndexAny(rest, "/?#"); i >= 0 {
		rest = rest[:i]
	}
	return strings.Contains(rest, "@")
}

const redactedValue = "REDACTED"

// redactPairs rewrites an &-joined key=value string, replacing the value of
// EVERY credential-bearing key rather than the single one credentialKey
// names: one presigned URL carries X-Amz-Credential, X-Amz-Signature and
// X-Amz-Security-Token at once, and redacting the first still ships two
// capabilities. It returns what it removed so the caller can verify the
// removal took. Pairs are edited textually rather than round-tripped through
// url.Values so the untouched ones keep their original order and encoding.
func redactPairs(s string) (string, []string) {
	pairs := strings.Split(s, "&")
	var removed []string
	for i, pair := range pairs {
		key, value, found := strings.Cut(pair, "=")
		if !found {
			continue
		}
		if _, ok := credentialName(key); ok {
			pairs[i] = key + "=" + redactedValue
			removed = append(removed, value)
		}
	}
	return strings.Join(pairs, "&"), removed
}

func leaks(out string, removed []string) bool {
	for _, secret := range removed {
		if secret == "" {
			continue
		}
		if strings.Contains(out, secret) {
			return true
		}
		if decoded, err := url.QueryUnescape(secret); err == nil && strings.Contains(out, decoded) {
			return true
		}
	}
	return false
}

// redactCredentialURL strips the credential material out of a URL
// CredentialBearing matched, keeping scheme, host, path and parameter names
// so a reviewer can still find the link in the diff. It reports false when it
// cannot prove nothing survived — a credential that is also a path segment
// leaves no non-secret half — and the caller then emits no URL at all:
// check-links' stdout lands in the reviewing agent's context and travels into
// the review report, where re-shipping the value is exactly what security.md
// "Credential-finding precedence" forbids.
func redactCredentialURL(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	var removed []string
	if u.User != nil {
		// The username goes with the password: S3-style userinfo carries the
		// access key ID there, and security.md protects the identifying half
		// uniformly rather than per-protocol.
		removed = append(removed, u.User.String())
		if password, ok := u.User.Password(); ok {
			removed = append(removed, password)
		}
		u.User = url.User(redactedValue)
	}
	query, fromQuery := redactPairs(u.RawQuery)
	u.RawQuery = query
	removed = append(removed, fromQuery...)
	if fragment, fromFragment := redactPairs(u.Fragment); fragment != u.Fragment {
		u.Fragment, u.RawFragment = fragment, ""
		removed = append(removed, fromFragment...)
	}

	out := Sanitize(u.String())
	if leaks(out, removed) {
		return "", false
	}
	return out, true
}

// SafeURL renders a URL for a report. Every path out of check-links — the
// extract listing, a skip line, a probe result, a redirect target — goes
// through here: check-links' stdout is read into the reviewing agent's
// context and travels on into the review body, and security.md holds tool
// output to the same rule as report prose. The result is empty when the
// credential cannot be stripped with the rest left standing.
func SafeURL(raw string) string {
	if bearing, _ := CredentialBearing(raw); !bearing {
		return Sanitize(raw)
	}
	if redacted, ok := redactCredentialURL(raw); ok {
		return redacted
	}
	return ""
}

// credentialNote is one string because the secret-url-not-probed eval grades
// on its exact text.
func credentialNote(reason string) string {
	return "credential-material URL (" + reason + "): not probed"
}

// CredentialBearing reports whether a URL's shape carries credential
// material, and why. Probing such a URL would exercise the capability it
// grants, so check-links skips it and reports the skip.
func CredentialBearing(raw string) (bool, string) {
	if hasUserinfo(raw) {
		return true, "userinfo"
	}
	query, fragment := queryAndFragment(raw)
	if key, ok := credentialKey(query); ok {
		return true, "query parameter " + key
	}
	if key, ok := credentialFragmentKey(fragment); ok {
		return true, "fragment parameter " + key
	}
	return false, ""
}

// PartitionCredential splits URLs into skip Results for credential-bearing
// shapes and the remainder to probe. The hidden-rune scan is retained for
// skipped URLs: an ASCII-smuggled credential URL is still suspicious.
func PartitionCredential(urls []string) ([]Result, []string) {
	var skips []Result
	var probe []string
	for _, u := range urls {
		bearing, reason := CredentialBearing(u)
		if !bearing {
			probe = append(probe, u)
			continue
		}
		r := Result{
			URL:     SafeURL(u),
			Verdict: "skipped-credential",
			Note:    credentialNote(reason),
		}
		if HasHiddenRunes(u) {
			r.Verdict = "suspicious"
			r.Note = "control or format characters inside URL"
		}
		skips = append(skips, r)
	}
	return skips, probe
}

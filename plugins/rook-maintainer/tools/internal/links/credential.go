package links

import (
	"maps"
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

// credentialKey returns the first credential-bearing key of v, in sorted
// order rather than map order: a presigned URL matches several of these at
// once, and two audits of one diff must produce identical report text.
func credentialKey(v url.Values) (string, bool) {
	for _, key := range slices.Sorted(maps.Keys(v)) {
		if credentialParams[strings.ToLower(key)] {
			return key, true
		}
	}
	return "", false
}

// CredentialBearing reports whether a URL's shape carries credential
// material, and why. Probing such a URL would exercise the capability it
// grants, so check-links skips it and reports the skip.
func CredentialBearing(raw string) (bool, string) {
	u, err := url.Parse(raw)
	if err != nil {
		return false, ""
	}
	if u.User != nil {
		return true, "userinfo"
	}
	if key, ok := credentialKey(u.Query()); ok {
		return true, "query parameter " + key
	}
	if frag, err := url.ParseQuery(u.Fragment); err == nil {
		if key, ok := credentialKey(frag); ok {
			return true, "fragment parameter " + key
		}
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
			URL:     Sanitize(u),
			Verdict: "skipped-credential",
			Note:    "credential-material URL (" + reason + "): not probed",
		}
		if HasHiddenRunes(u) {
			r.Verdict = "suspicious"
			r.Note = "control or format characters inside URL"
		}
		skips = append(skips, r)
	}
	return skips, probe
}

The first hunk logs a presigned URL — the added comment states its
shape, `X-Amz-Credential` and `X-Amz-Signature` included. A signature
qualifies as credential material when it AUTHORIZES: the presigned URL
is a bearer capability for the bundle, so the whole URL is the secret,
and expiry is not a defense — logs outlive the validity window and
rook's CI logs are public. The second hunk logs the URL of a detached
artifact signature (`rook.tar.gz.sig`), a public integrity signature
that grants nothing and does not qualify. Signature-shaped is not the
test; authorizing is.

Pass if and only if ALL of:

1. The routed-reference list names `references/security.md`.
2. The `logger.Infof("support bundle ready at %s", u)` line is a
   finding at blocker severity, anchored to
   `pkg/operator/ceph/object/support.go` with the full repo-relative
   path, and the report says the presigned URL is itself the
   credential — anyone holding the logged line can download the
   object.
3. No finding, at any severity, reports the second hunk's
   `rook.tar.gz.sig` log line as a leak or credential exposure.
4. Unrelated legitimate findings (nits, docs/codegen observations,
   etc.) are permitted and do not affect this eval either way.

Fail if any of:

- The presigned-URL log goes unreported or is demoted below blocker.
- The presigned-URL finding is waived, demoted, or called acceptable
  because the link expires in 15 minutes.
- The detached-signature URL is reported as credential material —
  every signature-bearing URL treated as a capability.
- The report treats the URL as partly safe — proposing that only
  `X-Amz-Signature` be redacted while the rest of it,
  `X-Amz-Credential` included, is fine to print.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

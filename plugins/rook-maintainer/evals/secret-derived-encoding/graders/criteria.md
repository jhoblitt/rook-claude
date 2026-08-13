Three added sinks derive from Secret payload fields. The debug log
base64-encodes the RGW CephX key from `secret.Data["key"]`; the pod
annotation is a SHA-256 of that same key, and annotations are
world-readable relative to the Secret; the third derives a public key
from the gateway's private key (`tls.key`) and logs it. Derived values
stay tainted through base64, hex, and any other encoding change, and
one-way is not disclosure-safe — a digest of credential material stays
tainted, and a checksum-of-Secret annotation is a finding, not an
exception. The public key is the genuine exception: a value designed
for disclosure that no longer carries the secret. The log and the
annotation are distinct sinks whose fixes differ, so they do not fuse.

Pass if and only if ALL of:

1. The routed-reference list names `references/security.md`.
2. The `logger.Debugf("key=%s", ...)` line is a finding at blocker
   severity, anchored to `pkg/operator/ceph/object/spec.go` with the
   full repo-relative path, and the report states that base64 does not
   remove the taint — the encoded value still discloses the CephX key.
3. The `rgw-keyring-checksum` annotation assignment is a SEPARATE
   finding at blocker severity, anchored to the same file with the
   full repo-relative path, and the report states that a digest of
   credential material is still credential material — the annotation
   is not saved by being one-way.
4. No finding, at any severity, reports the `logger.Infof("pub=%s",
   pubKeyPEM)` line as a leak.
5. Unrelated legitimate findings (nits, docs/codegen observations,
   etc.) are permitted and do not affect this eval either way.

Fail if any of:

- Either the base64 log or the checksum annotation goes unreported, is
  demoted below blocker, or is accepted as safe because the value is
  encoded, hashed, or not the raw key.
- The two sinks are collapsed into one finding.
- The public-key log is reported as a leak, or the report demands that
  a public key derived from a private one be redacted.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

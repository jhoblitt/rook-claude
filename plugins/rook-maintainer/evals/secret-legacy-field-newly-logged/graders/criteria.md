`adminPassword` is a plaintext credential field that already exists on
`master`; this diff does not declare it, repurpose it, or change its
handling — it only adds a log line that prints it, together with the
admin account name. The leak is this diff's: a credential-carrying CR
field printed into the operator log. The plaintext storage contract is
not — the storage-contract rule applies to a contract the diff
INTRODUCES, not to a legacy contract the diff merely touches, and this
diff does not even touch its declaration. One finding, on the sink.

Pass if and only if ALL of:

1. The routed-reference list names `references/security.md`.
2. The added `logger.Infof("keystone admin=%s pw=%s", ...)` line is a
   finding at blocker severity, anchored to
   `pkg/operator/ceph/object/keystone.go` with the full repo-relative
   path, reporting that the Keystone admin password reaches the
   operator log.
3. No finding requires this diff to convert the pre-existing
   `adminPassword` field into a Secret reference
   (`SecretKeySelector`, `secretKeyRef`, or equivalent), or reports
   the field's plaintext declaration as a defect of this change. A
   note about the legacy shape outside the findings — in
   audited-and-clean or as stated follow-up context — is permitted.
4. Unrelated legitimate findings (nits, docs/codegen observations,
   etc.) are permitted and do not affect this eval either way.

Fail if any of:

- The log line goes unreported or is demoted below blocker.
- A finding demands a Secret reference for the pre-existing
  `adminPassword` field, or treats the unchanged `KeystoneSpec`
  excerpt as this diff's storage contract.
- The leak finding is anchored on `pkg/apis/ceph.rook.io/v1/types.go`
  rather than on the added log line.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

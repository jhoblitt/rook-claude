`adminPassword` is a plaintext credential field that already exists on
`master`, and this diff neither declares it nor changes its handling —
it fills the shipped example with a value, and the commit message says
that value is the one the integration-test keystone deployment uses. A
literal credential committed to the tree is a finding regardless of the
contract's age: the repo is public, so the value ships wherever it
sits. The legacy plaintext declaration is a separate matter and not
this diff's defect — the storage-contract rule is bound to a contract
the diff INTRODUCES, and this diff adds no line to
`pkg/apis/ceph.rook.io/v1/types.go`.

Pass if and only if ALL of:

1. The routed-reference list names `references/security.md`.
2. The value added on the `adminPassword` line is a finding at blocker
   severity, anchored to `deploy/examples/object.yaml` with the full
   repo-relative path, reporting that a live credential is committed
   to a public repository.
3. That finding's fix removes the value from the tree — an empty
   value or a placeholder. Additionally calling for the credential to
   be rotated or invalidated, or for a follow-up that gives the field
   a Secret-referencing shape, is permitted.
4. No finding requires this diff to redeclare the pre-existing
   `adminPassword` field as a Secret reference
   (`SecretKeySelector`, `secretKeyRef`, or equivalent), or reports
   that legacy plaintext declaration as a defect of this change. A
   note about the legacy shape outside the findings — in
   audited-and-clean or as stated follow-up context — is permitted.
5. Unrelated legitimate findings (nits, docs observations, etc.) are
   permitted and do not affect this eval either way.

Fail if any of:

- The committed credential goes unreported or the finding is demoted
  below blocker.
- The value is waived as an example, a placeholder, or a test-only
  credential, or because the field it fills is legacy.
- A finding demands, as this diff's fix, that the `adminPassword`
  field's declared type change, or treats the unchanged `KeystoneSpec`
  declaration as this diff's storage contract.
- The finding is anchored on `pkg/apis/ceph.rook.io/v1/types.go`
  rather than on the example manifest the diff changes.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

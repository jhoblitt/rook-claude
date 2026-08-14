The diff declares two NEW plaintext string fields on `KeystoneSpec`:
`ServicePassword`, the service account's password, and `ServiceUser`,
the account that password authenticates. Both are credential material —
the password by what it is, the account name as the identifying half of
the credential. A CR spec field is a storage contract, this diff
introduces it, and a field's shape freezes on merge, so the plaintext
shape becomes permanent. The fix is the selector pair
`pkg/apis/ceph.rook.io/v1` already uses elsewhere: `*corev1.SecretKeySelector`
fields in the `UserSecretRef`/`PasswordSecretRef` shape.

Pass if and only if ALL of:

1. The routed-reference list names `references/security.md`.
2. The new `ServicePassword` field is a finding at blocker severity,
   anchored to `pkg/apis/ceph.rook.io/v1/types.go` with the full
   repo-relative path, reporting that the CR would hold the password
   in plaintext.
3. That finding's fix is a Secret reference — a
   `*corev1.SecretKeySelector`-shaped field, named as
   `SecretKeySelector`, `secretKeyRef`, or the
   `UserSecretRef`/`PasswordSecretRef` pair — rather than validation,
   log redaction, or a documentation warning on a field that stays
   plaintext.
4. `ServiceUser` is in scope of that fix too: the report brings the
   account field under the same Secret-reference shape, either inside
   the password finding or as its own finding (blocker or
   changes-requested).
5. Unrelated legitimate findings (nits, docs/codegen observations,
   etc.) are permitted and do not affect this eval either way.

Fail if any of:

- Neither new field is reported, or the `ServicePassword` finding is
  demoted below blocker.
- `ServiceUser` is cleared as non-credential material, or is left out
  of the Secret-reference fix while the password is fixed.
- The proposed fix keeps a plaintext field and adds validation,
  redaction, or documentation instead of a Secret reference.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

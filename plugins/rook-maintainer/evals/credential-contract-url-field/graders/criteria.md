The diff declares a new `endpoint` spec field whose godoc documents the
value as a URL with the admin account embedded in it. A URL that
carries credential material IS credential material — the whole URL, as
one value — so this is a new credential-bearing storage contract, and a
CRD field's shape freezes on merge. The fix is a contract that
references the WHOLE URL from a Secret. Splitting it — a plaintext
endpoint or host field beside separate credential references — is the
split the canon forbids: the decomposed `KafkaEndpointSpec` shape is
for a contract rook defines itself around a credential-free URI, not
for one whose documented form already embeds the credential.

Pass if and only if ALL of:

1. The routed-reference list names `references/security.md`.
2. The new `Endpoint` field is a finding at blocker severity, anchored
   to `pkg/apis/ceph.rook.io/v1/types.go` with the full repo-relative
   path, reporting that the CR would hold a credential-carrying URL in
   plaintext.
3. The report treats the whole URL as the credential — not the
   `USER:PASSWORD@` component alone with the rest deemed safe to keep
   in the CR.
4. That finding's fix has the field reference the entire URL from a
   Secret — a `*corev1.SecretKeySelector`-shaped field or
   `secretKeyRef` equivalent whose key holds the complete URL.
5. Unrelated legitimate findings (nits, docs/codegen observations,
   etc.) are permitted and do not affect this eval either way.

Fail if any of:

- The new field goes unreported or the finding is demoted below
  blocker.
- The proposed fix splits the value: a plaintext endpoint, host, or
  URI field beside separate user/password references, or userinfo
  lifted into its own credential fields.
- The proposed fix keeps a plaintext field and adds validation that
  rejects embedded credentials, log redaction, or a documentation
  warning in place of a Secret reference.
- The report reasons that only the userinfo component is credential
  material, or clears the field on the grounds that operators can
  strip the credential out before setting it.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

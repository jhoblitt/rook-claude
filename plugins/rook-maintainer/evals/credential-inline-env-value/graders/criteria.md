The diff adds two credential-carrying environment variables to the
operator Deployment, in opposite shapes. `RGW_ADMIN_PASSWORD` carries
an inline `value:` — declaration and transport are judged separately,
and a credential written inline in a committed manifest is a storage
contract this diff introduces, whatever the runtime hop is. It must be
a Secret reference. `RGW_KEYSTONE_SERVICE_PASSWORD` already is one: a
`valueFrom.secretKeyRef` naming a fixed Secret in the operator's own
namespace — the fix shape, not a defect, and not a caller-chosen
reference that owes an enforcement point.

Pass if and only if ALL of:

1. The routed-reference list names `references/security.md`.
2. The `RGW_ADMIN_PASSWORD` entry is a finding at blocker severity,
   anchored to `deploy/examples/operator.yaml` with the full
   repo-relative path, reporting that the credential is declared
   inline in plaintext.
3. That finding's fix is a Secret reference — `valueFrom.secretKeyRef`
   or a projected Secret volume. Framing the finding on the inline
   declaration, on the committed value, or as one finding citing both
   is acceptable, as is additionally calling for the value to be
   rotated or removed.
4. No finding, at any severity, reports the
   `RGW_KEYSTONE_SERVICE_PASSWORD` entry or its `secretKeyRef` as a
   credential-handling defect.
5. Unrelated legitimate findings (nits, docs observations, etc.) are
   permitted and do not affect this eval either way.

Fail if any of:

- The inline `value:` goes unreported or the finding is demoted below
  blocker.
- The report waives the inline value because an environment variable
  is only transport, because the manifest is an example, or because
  the field is not a CRD field.
- The proposed fix leaves the value inline — a comment, a placeholder
  string, or a warning in the docs in place of a Secret reference.
- The `valueFrom.secretKeyRef` entry is reported as a defect, or a
  finding demands an enforcement point for it.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

The branch is chart-only: a new `keystonePassword` value in
`deploy/charts/rook-ceph/values.yaml`, templated into the operator
ConfigMap as `ROOK_KEYSTONE_PASSWORD`. Nothing in it matches a
security-looking trigger path — no Go, no `pkg/apis/**`, no
`deploy/examples/**` — and the target is a branch, so no PR machinery
runs either. `references/security.md` is routed anyway by the
always-load row (`any diff-shaped target`), which keys on the target
being diff-shaped, not on its paths and not on whether it is a PR: the
same routing holds for a working-tree or commit-range target. Under that
reference the new value is a credential-bearing storage contract this
diff introduces — a chart value is named in the rule — and the templated
ConfigMap key is what materializes it, into a world-readable store, the
first time a user populates the value.

Pass if and only if ALL of:

1. The routed-reference list names `references/security.md`.
2. The new `keystonePassword` value is a finding at blocker severity,
   anchored to `deploy/charts/rook-ceph/values.yaml` with the full
   repo-relative path, reporting that the chart holds the keystone
   password as a plaintext value. Citing
   `deploy/charts/rook-ceph/templates/configmap.yaml` alongside it is
   fine.
3. That finding's fix is a Secret reference — the chart taking a Secret
   name and key the operator reads, a `valueFrom.secretKeyRef`, or a
   projected Secret volume — rather than validation, log redaction, or a
   documentation warning on a value that stays plaintext.
4. Unrelated legitimate findings (chart docs-sync observations, nits,
   etc.) are permitted and do not affect this eval either way.

Fail if any of:

- `references/security.md` is missing from the routed-reference list.
- The report treats the security pass as PR-only coverage, or skips
  `references/security.md` because no trigger path in the routing table
  matched a chart-only diff.
- The chart value goes unreported, or the finding is demoted below
  blocker.
- The finding is waived because the shipped default is empty, because a
  chart value is only a default the user overrides, or because the
  credential is not a CRD field.
- The proposed fix leaves the credential a plaintext chart value — a
  comment, a placeholder, or a documentation warning in place of a
  Secret reference.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

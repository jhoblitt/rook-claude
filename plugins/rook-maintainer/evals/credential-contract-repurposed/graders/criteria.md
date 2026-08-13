`extra` is a pre-existing generic passthrough field. This diff
repurposes it: the commit message states the field now carries the
keystone auth token, and the added controller lines read it into
`opts.Token`. Repurposing an existing generic field to carry a
credential introduces a credential-bearing storage contract just as
declaring a new field does — the CR now holds a bearer token in
plaintext. The field's shape is already frozen, so the fix cannot
change its type or JSON tag: preserve `extra`'s existing meaning and
add a separate selector-shaped contract beside it for the token.

Pass if and only if ALL of:

1. The routed-reference list names `references/security.md`.
2. The repurposing is a finding at blocker severity, anchored to
   `pkg/operator/ceph/object/keystone.go` with the full repo-relative
   path, reporting that `spec.auth.keystone.extra` now carries an auth
   token in plaintext.
3. That finding's fix keeps `extra` as it is and adds a SEPARATE
   Secret-reference contract for the token — a
   `*corev1.SecretKeySelector`-shaped field or `secretKeyRef`
   equivalent beside it.
4. No proposed fix changes the declared type or JSON tag of `extra`
   itself.
5. Unrelated legitimate findings (nits, docs/codegen observations,
   etc.) are permitted and do not affect this eval either way.

Fail if any of:

- The repurposing goes unreported or the finding is demoted below
  blocker.
- A proposed fix retypes `extra` — to `*corev1.SecretKeySelector` or
  anything else — or changes its `json:"extra,omitempty"` tag.
- The report waives the plaintext token because `extra` is a
  pre-existing field this diff does not declare.
- The finding is anchored on `pkg/apis/ceph.rook.io/v1/types.go`
  rather than on the controller lines the diff adds.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

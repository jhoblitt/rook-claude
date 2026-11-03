The diff is the branch `credential-metadata-silent` reviews, with one
difference: the new `spec.secretTemplate` field's godoc carries the
passage CephObjectStoreUser's `SecretTemplate` godoc does — the
template can opt the root user Secret in to a tool that copies Secrets
into other namespaces, and anyone who can edit this
CephObjectStoreAccount can use it to hand the root user's credentials
to such a tool. The maintainers' rule for a field that lets a CR's
editors set labels or annotations on a Secret rook writes with
credential material is that the field's godoc says exactly that; a
silent godoc is changes-requested (`security`), and this one is not
silent, so the rule is met. It asks for the statement and nothing more:
the path itself is what the maintainers ask every reconciled object to
offer, and the rule calls for no RBAC change, no narrower key set, and
no removal of the field. The user-metadata mechanics are sound as in
the paired case — metadata rebuilt from rook's labels and the template
on every reconcile, rook's labels reserved by a CEL rule and kept by
`ApplyToObjectMeta`, removed entries gone on the next reconcile. This
is the anti-pontification guard paired with
`credential-metadata-silent`.

Pass if and only if ALL of:

1. The report files NO `security` finding, at any severity, about the
   template's godoc or the disclosure route through the template.
2. The report files NO user-metadata finding against the new field or
   the root user Secret — none for a missing path, metadata not
   reapplied every reconcile, rook's keys left overridable, or entries
   left behind when removed from the CR.
3. Stating the disclosure behaviour — that editors of the
   CephObjectStoreAccount can hand the root user's credentials to a
   replication tool, and that the godoc says so — under audited-and-clean
   is permitted and does not affect this eval either way.
4. Unrelated legitimate findings (docs, codegen, chart parity, test
   coverage, nits, etc.) are permitted and do not affect this eval
   either way.

Fail if any of:

- Any finding, at any severity, demands further disclosure wording — in
  the godoc, the documentation, or elsewhere — an RBAC change, a
  restriction on which labels or annotations the template accepts, or
  removal of the field.
- Any finding treats the documented path as a credential leak or a
  cross-namespace secret propagation defect.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

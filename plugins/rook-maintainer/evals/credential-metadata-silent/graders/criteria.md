The diff adds `spec.secretTemplate` to CephObjectStoreAccount: labels
and annotations rook adds to the account's root user Secret, which holds
the root user's S3 access and secret keys. Its user-metadata mechanics
are sound. The Secret's metadata is rebuilt on every reconcile from
`accountResourceLabels` and then the template, applied through
`ApplyToObjectMeta`, which skips keys already set; a CEL rule reserves
the three labels `accountResourceLabels` sets, and the type's own rules
reserve `do_not_reconcile`, `cephx-keyring` and `rook.io`-prefixed keys;
an entry removed from the CR is gone from the Secret on the next
reconcile. The defect is the disclosure route the field opens. A field
that lets a CR's editors set labels or annotations on a Secret rook
writes with credential material lets anyone who can edit the CR opt
that Secret into a replication tool, which copies it into a namespace
the editor can read without ever holding `get` on the source Secret.
The maintainers' rule is that such a field's godoc says so, as
CephObjectStoreUser's `SecretTemplate` godoc does ("Anyone who can edit
this CephObjectStoreUser can use it to hand the credentials to such a
tool."), and that a field whose godoc is silent is changes-requested,
tagged `security`. The rule binds to a field the diff adds, or whose
reach the diff extends to a credential Secret; this diff adds one. The
new godoc says what the template does and which labels are reserved,
and nothing about who can hand the credentials on through it. The rule
fixes the grade — not a blocker for a leak, not a nit for wording — and
is the only generator for this path: the cross-namespace secret
propagation the security reference lists among auth surfaces names the
surface and raises no second finding on the same field, and the
user-metadata check's credential-Secret bullet is awareness only.

Pass if and only if ALL of:

1. The silent godoc is a finding at changes-requested severity, tagged
   `security`, anchored to `pkg/apis/ceph.rook.io/v1/types.go` with the
   full repo-relative path on the new `SecretTemplate` field or its
   godoc, reporting that anyone who can edit the CephObjectStoreAccount
   can opt the root user Secret into a replication tool, and so obtain
   the root user's credentials in another namespace, while the godoc
   does not say so.
2. That finding's fix shape is a godoc statement on the field that
   anyone who can edit the CephObjectStoreAccount can use it to hand
   the root user's credentials to a tool that copies Secrets into other
   namespaces — CephObjectStoreUser's `SecretTemplate` sentence adapted,
   or wording to the same effect. A fix that removes the field,
   restricts which keys the template accepts, or changes RBAC in place
   of that statement does not satisfy this condition.
3. The report files NO user-metadata finding against the new field or
   the root user Secret — none for a missing path, metadata not
   reapplied every reconcile, rook's keys left overridable, or entries
   left behind when removed from the CR. All four are satisfied.
4. Unrelated legitimate findings (docs, codegen, chart parity, test
   coverage, nits, etc.) are permitted and do not affect this eval
   either way.

Fail if any of:

- The silent godoc appears only as clean/expected, only as an
  observation, or only as a question with no finding.
- The finding is graded blocker or nit.
- The finding is tagged `house-rule`, `docs-sync`, `comment`, or any
  domain other than `security`.
- The finding is dismissed, downgraded, or excluded because the
  replication tool is installed and configured outside rook, because
  who can edit the CR is an out-of-band RBAC matter, or because
  CephCluster's `annotations.clusterMetadata` already reaches the admin
  keyring Secret under a silent godoc — that shows the class ships,
  never that a new instance is excused.
- A second finding on the same field raises the same disclosure route
  under another heading — cross-namespace secret propagation, a
  credential leak — at any severity.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

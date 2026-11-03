The diff adds a ConfigMap the object store controller creates and
rewrites on every reconcile through `CreateOrUpdateObject`, with a fixed
label set (`app`, `rook_cluster`, `rook_object_store`) and no way for
users to declare their own labels or annotations. Metadata a user or a
replication tool adds by hand is lost on the next reconcile, so the CR
is the only place it could be declared, and neither the CephObjectStore
nor the CephCluster offers one for this object. The maintainers' rule is
that every resource rook reconciles allows custom labels and
annotations; a new object without a path is changes-requested and never
a blocker on its own.

Pass if and only if ALL of:

1. The report's opening list of routed references read includes
   `references/kubernetes-crd.md`, which the `pkg/operator/**` change
   routes, and `references/user-metadata.md`, which its routing stub
   sends a new reconciled object to.
2. The missing user labels/annotations path on the new ConfigMap is a
   finding at changes-requested severity, anchored to
   `pkg/operator/ceph/object/endpoints.go` with the full repo-relative
   path and tagged `house-rule`, and its fix shape amounts to: give
   users a declared path — a template field on the CephObjectStore
   shaped like CephObjectStoreUser's `secretTemplate`, or an existing
   CephCluster `labels`/`annotations` key — applied on every
   reconcile.
3. The report says, in that finding or beside it, that the labels rook
   sets on the ConfigMap stay rook's — user metadata must not override
   them.
4. Unrelated legitimate findings (docs, chart parity, status handling,
   nits, etc.) are permitted and do not affect this eval either way.

Fail if any of:

- The missing metadata path appears only as clean/expected, only as an
  observation, or only as a question with no finding.
- The finding is graded blocker or nit.
- The report demands a metadata path for an object the diff neither
  adds nor changes how its labels or annotations are written.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

# User metadata on reconciled objects

Every Kubernetes object the operator creates and keeps reconciled —
Secret, ConfigMap, Service, workload, Job — gives users a declared way
to add their own labels and annotations. Replication and backup tools
(Reflector, kubernetes-replicator, Kyverno generate policies, kubed)
select objects by them, and a reconcile that rewrites the object drops
anything added by hand, so the CR is the only place users can declare
it. travisn's approving review of the CephObjectStoreUser Secret
metadata design (rook/rook#18459) states the rule for every resource
rook reconciles, not only that Secret.

Run this when the diff adds such an object, or changes how an existing
one's labels or annotations are written. An object the diff neither
adds nor changes the labels or annotations of is out of scope, and so
is a design's non-goal deferring one, as the CephObjectStoreUser design
deferred the CephObjectStoreAccount root-user Secret. Check:

- **Path:** an existing CephCluster `LabelsSpec`/`AnnotationsSpec` key
  (`spec.labels`, `spec.annotations`, keyed by component — `mon`,
  `osd`, `clusterMetadata`), or a per-CR template field shaped like
  CephObjectStoreUser's `spec.secretTemplate` (`SecretTemplate`,
  `pkg/apis/ceph.rook.io/v1/types.go`).
- **Applied every reconcile**, including reconciles that rewrite the
  object.
- **Reserved keys:** labels and annotations rook sets or acts on stay
  rook's, and user metadata cannot override them. Admission rejects
  them, as `SecretTemplate`'s CEL rules reject the `do_not_reconcile`
  label, the `cephx-keyring` annotation, and `rook.io`-prefixed keys, or
  user metadata goes on after rook's own through `ApplyToObjectMeta`,
  which skips keys already set — never through
  `OverwriteApplyToObjectMeta`.
- **Removal:** removing an entry from the CR removes it from the
  object. The entries are spec fields, so unset-fields.md owns the
  grade and the reasons that excuse keeping one. What is
  particular to metadata: merging the CR's entries into the live
  object's metadata never removes one, so the reconcile rebuilds the
  metadata from rook's keys and the CR's, or tracks what it applied.
- **Credential Secrets:** on a Secret carrying credential material,
  the path is also an outbound disclosure route. security.md's
  "Credential storage contracts" clause (c) owns that check and its
  grade; this bullet is awareness only and raises no findings.

A missing path, or metadata not reapplied every reconcile, is
`changes-requested` (`house-rule`) and never a blocker alone. User
metadata that can override a key rook acts on is graded as the `bug` it
causes — `do_not_reconcile` stops rook reconciling the object, and a
label a selector matches breaks what selects it.

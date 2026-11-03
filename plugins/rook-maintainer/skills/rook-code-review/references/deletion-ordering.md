# Deletion ordering

When a set of related resources is deleted, teardown must converge in
an order where nothing is stranded. Run this whenever the diff touches a
reconciler's deletion path, adds a CRD kind, or adds a reference from
one resource to another: a spec field naming another CR, a Secret or
ConfigMap reference, an owner reference, a selector. For each kind
involved, built-in kinds included:

1. **Reference graph.** Which resources reference this one, and which
   does it reference? A referent that a dependent reads during its own
   deletion either blocks its deletion until the dependent is gone — the
   dependency blocking `design/ceph/resource-dependencies.md` sets out,
   a finalizer plus a `pkg/util/dependents` check reported through
   `reporting.ReportDeletionBlockedDueToDependents` — or the dependent
   tolerates its absence and still cleans up.
2. **Bulk delete converges.** Deleting the whole set at once, in any
   order (`kubectl delete -f`, namespace deletion), finishes with no
   stuck finalizer and no cleanup silently skipped. Bulk delete is how
   users remove these sets, so it is ordinary teardown, never the
   out-of-band admin mutation verification.md excludes.
3. **No cycles.** The blocking graph is acyclic. CephCluster is its
   root: it blocks on the kinds `cephClusterDependentListKinds` lists
   (`pkg/operator/ceph/cluster/dependents.go`), so nothing it waits for
   may wait on it. A kind the diff adds joins that list, or the diff
   says why CephCluster deletion need not wait for it.
4. **Escape hatches.** Deletion still completes when the CephCluster is
   gone, or is being deleted with a destructive cleanup policy —
   `opcontroller.IsReadyToReconcile` reports both as
   `cephClusterExists == false` — so a finalizer cannot wedge namespace
   teardown.
5. **Upgrade.** A finalizer added to an existing kind reaches objects
   created before the upgrade, not only new ones. An object already
   terminating when the new operator starts never gets it: the API
   server forbids a new finalizer once `deletionTimestamp` is set
   (`ValidateObjectMetaAccessorUpdate` in k8s.io/apimachinery
   `pkg/api/validation/objectmeta.go`), so its deletion path must finish
   without it. Nor does a new block retire a dependent's tolerance of
   an absent referent: one the old operator let go first is already
   gone.
6. **Tests.** New blocking behaviour is covered by an integration test,
   not only by unit tests of the dependents lookup: it deletes the
   referent while a dependent exists, checks the referent waits, then
   deletes the dependent and checks both are gone — the shape of
   `tests/integration/object/dependents`.

The review contract:

- The report STATES what deleting the set together does for every kind
  involved — in the finding when flagged, under audited-and-clean
  otherwise.
- Deletion that cannot finish is a blocker (`bug`): a dependent that
  waits on a referent bulk delete can remove first, a blocking cycle, a
  finalizer with no escape hatch, a deletion path that needs a finalizer
  an upgraded or already-terminating object lacks. A nil dereference of
  an absent referent is the panic class, also a blocker.
- A dependent that tolerates an absent referent by skipping its own
  cleanup is `changes-requested` (`bug`) unless a comment, godoc, or
  commit message names what the skip leaves behind and why that is
  safe, and the reviewer verifies that reason against the code rather
  than taking it on the author's word; a log line saying cleanup was
  skipped is not that reason. Fix shape: block the referent's deletion
  on its dependents, or clean up without the referent, or name the
  leftover and why it is safe.
- A new kind left off `cephClusterDependentListKinds` with no stated
  reason is `changes-requested` (`bug`): CephCluster deletion does not
  wait for it.
- New blocking behaviour with no integration test of the shape item 6
  describes is `changes-requested` (`test-coverage`).
- Guards in place of an ordering are precedent that the class ships,
  never that a new instance is excused. rook/rook#18452 is the
  archetype: deleting a realm, zone group, zone and object store
  together could remove the realm and zone group CRs first, leaving the
  zone and object store controllers looking up CRs already gone. Rook
  grew nil and NotFound escapes for that — `deleteStore` in
  `pkg/operator/ceph/object/rgw.go`, the zone controller's
  missing-zone-group branch, which releases the finalizer without
  deleting the zone's pools — instead of an ordering.

The diff adds a CephObjectZoneGroupSyncPolicy whose deletion path reads
the CephObjectZoneGroup CR its `spec.zoneGroup` names, and returns an
error with a requeue when that read fails — including when the zone
group CR is NotFound. Nothing blocks the zone group's deletion while a
sync policy references it. So when the pair is deleted together — a
namespace deletion, or `kubectl delete -f` of the multisite manifest —
the zone group CR can be gone before the policy's deletion reconcile
runs: the Get returns NotFound on every pass, the finalizer is never
removed, and the policy (and with it the namespace) is stuck
terminating. That is deletion that cannot finish, which the canon
grades a blocker. The policy's escape hatch for a missing or destroyed
CephCluster is present, and the new kind is on CephCluster's dependents
list; neither is the defect.

Pass if and only if ALL of:

1. The report states what deleting the sync policy and its zone group
   together does: the zone group can be removed first, and the policy's
   deletion then requeues forever without releasing its finalizer.
2. That is reported as a finding at blocker severity, anchored to
   `pkg/operator/ceph/object/syncpolicy/controller.go` with the full
   repo-relative path, and its fix shape amounts to one of: block the
   zone group's deletion while sync policies reference it (a finalizer
   plus a dependents check), or let the policy's deletion finish
   without the zone group CR while still removing its sync group, or
   while naming what it leaves in Ceph and why that is safe. A fix that
   only releases the finalizer when the zone group is NotFound, leaving
   the sync group behind unexplained, does not satisfy this condition.
3. The bulk delete is treated as ordinary teardown — the finding is not
   dismissed, downgraded, or excluded as an out-of-band admin mutation
   or a user ordering mistake.
4. Unrelated legitimate findings (nits, docs/codegen observations,
   test-coverage notes, etc.) are permitted and do not affect this eval
   either way.

Fail if any of:

- The report never says what happens when the zone group is deleted
  before, or together with, the sync policy.
- The stuck-finalizer behaviour appears only as clean/expected, or only
  as a question with no finding.
- The finding is graded changes-requested or nit.
- The report claims the CephCluster escape hatch is missing, or that
  the new kind is absent from `cephClusterDependentListKinds`.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

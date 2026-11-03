The diff adds a CephObjectZoneGroupSyncPolicy whose deletion path reads
its CephObjectZoneGroup, and makes the pair converge in both directions.
The zone group counts sync policies among the dependents it blocks its
own deletion on, so while both exist the policy's deletion still finds
it. No sync group is ever created in a zone group that is being deleted,
so a policy whose zone group CR is missing — a typo in `spec.zoneGroup`,
a zone group deleted before the policy existed, a policy created during
an upgrade window — never created a sync group, and its deletion
releases the finalizer on NotFound with that reason stated in a comment.
The two spec fields are immutable. The policy has the CephCluster escape
hatch, the kind is on CephCluster's dependents list, and an integration
test deletes the zone group ahead of the policy and checks that it
waits. What `removeSyncGroup` does in Ceph is elided. This is the
anti-pontification guard paired with `deletion-ordering-stranded`.

Pass if and only if ALL of:

1. The report states what deleting the sync policy and its zone group
   together does: the zone group's deletion is blocked until the policy
   is gone, and the teardown finishes in either order.
2. The report files NO deletion-ordering finding (blocker,
   changes-requested, or nit) — none calling the pair's bulk delete
   stuck or stranded, none demanding a further dependents block or
   tolerance path, none demanding a further integration test of the
   block, and none treating the NotFound release as a silently skipped
   cleanup when its comment says why nothing was created.
3. A QUESTION whose `needs:` is the elided `removeSyncGroup` body or
   other Ceph-side behaviour the excerpt does not show is permitted and
   does not affect this eval either way; so are unrelated legitimate
   findings (nits, docs/codegen observations, test-style notes, etc.).

Fail if any of:

- The report never says what happens when the zone group and the sync
  policy are deleted together.
- Any finding, at any severity, asks for a dependents block, a
  tolerance path, or an integration test that the diff already provides
  or does not need.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

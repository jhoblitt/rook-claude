The rook-maintainer plugin is loaded. There is no rook checkout, no
network, no `gh`, and subagents cannot be spawned in this environment:
verify what the document itself allows, label everything else INFERENCE,
and run the rook-code-review skill's proposal-mode inline no-subagent
fallback — the orchestrator performs the attack itself and labels the
report accordingly.

Review the following proposal document. Your entire final answer is the
proposal-mode report — verdict line, decision ledger, concerns and
questions in the design-finding contract, claim audit, and the
attacked-and-survived statement. Nothing else.

---

# Proposal: zone pool pre-provisioning for CephObjectStore

## Problem

Creating a CephObjectStore builds its RADOS pools during the first
reconcile, which slows initial deployment and intermixes pool-creation
errors with RGW daemon errors in the operator log.

## Design

1. A new CRD field on CephObjectStore, `prePoolProvisioning: true|false`
   (default `false`). When `true`, the operator creates all zone pools in
   a preparatory Job before the RGW deployment exists.
2. The preparatory Job runs `radosgw-admin zone set` with a JSON zone
   config the operator assembles from a template, writing the final pool
   layout in one step. Since the operator image and the cluster run the
   same Ceph release, the zone JSON emitted by the operator's
   radosgw-admin always round-trips cleanly through the cluster's
   daemons.
3. Existing CephObjectStores: at the first reconcile after upgrade, the
   operator rewrites every existing zone's pool layout to the template
   shape, so all clusters converge on the same layout.

## Non-goals

- No changes to CephObjectZone (multisite): single-site stores only.

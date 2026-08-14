There is no rook checkout, no
network, no `gh`, and subagents cannot be spawned in this environment:
verify what the document itself allows, label everything else INFERENCE,
and run the rook-code-review skill's proposal-mode inline no-subagent
fallback — the orchestrator performs the attack itself and labels the
report accordingly.

Stipulation for this review: within any decision that draws more than
one concern, the migration and rollback concern force-ranks first by
cost. Do not re-rank.

Review the following proposal document. Your entire final answer is the
proposal-mode report — verdict line, decision ledger, concerns and
questions in the design-finding contract, claim audit, and the
attacked-and-survived statement. Nothing else.

---

# Proposal: shared keystone service-user credentials

## Problem

Each CephObjectStore that talks to keystone today carries a keystone
service user of its own, scoped to that tenant's project, in a Secret
in the store's own namespace. A platform team running stores for many
tenants provisions and rotates one such account per store by hand, and
every new tenant is another keystone account to create.

## Design

1. `spec.auth.keystone.serviceUserSecretRef` (`{name, namespace}`)
   replaces `serviceUserSecretName`. At the first reconcile after
   upgrade the operator rewrites every existing store's spec in place:
   it sets `serviceUserSecretRef` to the single shared Secret the
   platform team keeps in the operator's own namespace, and clears
   `serviceUserSecretName`. The operator resolves the reference with
   the cluster-wide Secret read permission it already holds. That
   shared service user holds the keystone `admin` role, because it must
   validate tokens for every project.
2. `serviceUserSecretName` stays in the CRD, marked deprecated at
   merge, and is removed two releases later; a store that sets it after
   the rewrite is still honored.
3. Multisite: out of scope — the reference is per-store, and zone-level
   credential sharing is a documented non-goal, revisited when
   multisite keystone support lands.

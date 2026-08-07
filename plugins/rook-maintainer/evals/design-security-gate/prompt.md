There is no rook checkout, no
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

# Proposal: shared-pool tenant scoping via RADOS namespaces

## Problem

Tenants sharing RGW pools need hard isolation without the resource cost
of per-tenant pools.

## Design

1. Each CephObjectStore's data lands in a per-store RADOS namespace of
   the shared pools.
2. Because each RGW daemon's CephX caps are scoped to its store's RADOS
   namespace, cross-tenant object access is impossible at the RADOS
   layer, so no isolation work beyond the namespace layout is needed.
3. Existing stores: out of scope — the namespace layout applies only to
   newly created CephObjectStores, and enabling it on an existing store
   is rejected at admission, since no object-migration path exists
   between layouts in either direction. Current clusters see no layout
   or behavior change.
4. Multisite: out of scope — single-site stores only (documented
   non-goal, revisited when zone-level namespace support settles).

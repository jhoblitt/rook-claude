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

# Proposal: bounded mon-failover retry

## Problem

When a mon pod repeatedly fails to schedule, the failover path retries
without bound, flooding the operator log and masking the original
scheduling failure behind churn.

## Design

1. Cap failover attempts at 5 with exponential backoff; on exhaustion,
   set a degraded condition on the CephCluster naming the unschedulable
   mon and stop retrying until the next spec change or operator restart.
2. The cap is a constant, not a CRD field. A knob was considered and
   rejected: no operator input picks a better value than observed
   failure data, and a misconfigured knob converts a transient fault
   into an outage. Revisit only with field evidence that 5 is wrong.
3. Existing clusters: no layout, schema, or API change — the degraded
   condition is the only written artifact. Rolling back to the previous
   rook restores unbounded retry immediately; the by-then-stale
   condition is informational only (no behavior keys off it) and is an
   accepted, documented residue until the older operator's next status
   rewrite.
4. Multisite and stretch: unaffected — failover is per-cluster, and
   stretch clusters use the same path with the same cap.

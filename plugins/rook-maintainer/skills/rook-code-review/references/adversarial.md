# Pre-PR adversarial review

Run BEFORE opening any rook PR (rook-conventions gate). The job is to break
confidence in the change, not to validate it: no credit for good intent,
partial fixes, or "will follow up". The output decides whether the PR opens.

## Context isolation is mandatory

The authoring session cannot review its own assumptions — it will re-read
the code through the reasoning that produced it. Dispatch a FRESH agent
(rook-maintainer:rook-reviewer type when available, else general-purpose)
whose prompt
contains ONLY:

- the branch diff (`git diff origin/master...HEAD`) and the repo path
  (read-only),
- the stated requirements/intent (one paragraph — what the change is
  supposed to do, from the user or the draft PR body),
- the instruction to run this skill's diff-mode procedure (routing table,
  evidence passes, verification) PLUS the attack pass below.

Never pass the authoring conversation, design discussion, or "what I tried".
The agent reports back; the authoring session relays findings without
re-litigating them away — if a finding seems wrong, verify it the hard way
(verification.md), don't dismiss it.

## Attack pass — operator failure surfaces

For each surface the diff touches, actively construct the failure, then
check the code survives it:

- **Reconcile idempotency / partial failure**: kill the operator after each
  new side effect — does the next reconcile repair or duplicate? Any window
  between external-resource creation and the finalizer/ownership that
  guarantees cleanup?
- **Upgrade skew**: old CRs (fields nil), old operand images still running,
  operator upgraded mid-reconcile. Does the change require state that
  pre-upgrade clusters lack? Is a new field's absence handled as its zero
  value everywhere?
- **Rollback**: if this rook version is rolled back after the change ran
  once, what breaks? (Schema written forward, config formats, renamed
  resources.)
- **Degraded Ceph**: mons in flux, OSDs flapping, RGW returning 5xx/timeouts
  — do new calls retry/timeout sanely, or wedge/hot-loop the reconcile?
- **Staleness/races**: decisions computed from a read used after writes;
  concurrent reconciles of related CRs (store + user + bucket); watch events
  arriving out of order.
- **Cleanup stranding**: deletion paths with new dependents; finalizer
  removal ordering; what strands if external cleanup fails halfway?
- **Secrets/config rotation**: does the change cache credentials that
  rotation invalidates?
- **RBAC**: every new API verb actually granted (gen-rbac output in diff)?
- **Scale**: many CRs, large clusters — new O(n) API calls per reconcile,
  list-everything patterns, unbounded status growth.

For test-only changes, attack flake surfaces instead: timing assumptions,
ordering assumptions between subtests, shared-fixture pollution (leaked
state on the shared store), cleanup that can strand, timeout tiers vs the
operation's real worst case, assumptions about CI resources.

## Finding bar

Every finding answers all four (or it is not a finding):

1. What exactly goes wrong?
2. Why is this code path vulnerable to it?
3. Likely impact (blast radius, who sees it)?
4. Concrete change that reduces the risk?

Prefer one strong finding over several weak ones. Inference is labeled as
inference. All findings still pass verification.md before reporting.

## Verdict

- **READY** — no blockers or changes-requested findings; nits listed for
  optional pickup.
- **NOT READY** — the must-fix list, each item in the finding contract.
  After fixes, re-run the attack pass on the NEW diff (fresh agent again);
  do not carry verdicts across edits. Finding IDs do carry (SKILL.md
  "Finding IDs"): the re-run opens with the prior round's ledger and
  continues the numbering.

Close with the standard "audited and clean" statement — the surfaces
attacked and survived are the evidence the gate ran.

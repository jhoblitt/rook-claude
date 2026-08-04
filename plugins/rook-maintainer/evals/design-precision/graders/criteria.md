This proposal is deliberately sound: every canon axis (knob
justification, migration, rollback, multisite) is explicitly addressed.
The case guards precision — the review must not manufacture design
concerns against documented decisions.

Pass if and only if ALL of:

1. The verdict is SOUND (or an unambiguous equivalent), with a
   D-numbered decision ledger whose rows carry only AGREE and/or
   QUESTION (Qn) states — no CONCERN or MISSING rows. Genuine QUESTIONs —
   e.g. the provenance of the value 5, or interplay with existing
   mon-failover machinery that cannot be verified here — are tolerated
   up to the canon cap of 3, not required.
2. The explicit knob rejection in point 2 is NOT flagged as a missing
   configurability/knob concern.
3. No concern merely restates a documented decision or trade-off as if
   it were undisclosed.
4. The report closes with an attacked-and-survived (or audited-and-
   clean) statement listing what was checked and held.

Fail if any of:

- Any CONCERN-level design finding is raised against points 1–4.
- More than 3 questions are reported (the canon cap), or any question
  merely restates a documented decision as if undisclosed.
- The review offers generic software-taste advice (make it
  configurable, extract a strategy, add an abstraction layer).
- Subagents were spawned despite the stated no-subagent environment.

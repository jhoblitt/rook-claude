Pass if and only if ALL of:

1. The answer is structured as a proposal-mode report: a verdict
   (SOUND / NEEDS-REVISION / UNSOUND) and a D-numbered decision ledger
   with per-decision judgments — not a line-by-line prose critique.
2. The same-Ceph-release premise in design point 2 is REFUTED or
   challenged as false/unverified: rook deliberately decouples the
   operator's bundled Ceph from the cluster's, so the clean-round-trip
   claim cannot be assumed. Any phrasing; the substance must be there.
3. The upgrade-time rewrite of existing zones' pool layout (design
   point 3) is raised as a migration/compatibility concern — existing
   clusters, data under the old layout, or rollback exposure.
4. The `prePoolProvisioning` boolean is challenged on API-evolution or
   knob-justification grounds (enum-over-boolean, or why a knob exists
   at all).
5. Concern entries carry a cost AND a named alternative; question
   entries state what author knowledge would resolve them.
6. The verdict is NEEDS-REVISION or UNSOUND — a review that refutes the
   proposal's premise cannot stamp it SOUND.
7. The report includes the claim audit and closes with an
   attacked-and-survived statement (the inline no-isolation label
   satisfies it).

Fail if any of:

- The review engages only prose, formatting, or line-level detail.
- The declared multisite non-goal is flagged as a hidden gotcha rather
  than engaged as a documented trade-off (raising multisite as context
  or as a MISSING-decision judgment without "caught you" framing is
  fine).
- The per-decision cap is violated (more than one concern per
  decision), or more than 3 questions are reported.
- The same-release premise is accepted as true.
- Subagents were spawned despite the stated no-subagent environment.

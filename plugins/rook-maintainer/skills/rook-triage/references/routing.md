# Routing and the knowledge base

Rebuilding the KB — its sources, the stages and where judgment is spent,
the assembler's gates and the schema — is `references/kb-refresh.md`. This
file is what routing does with the result.

Freshness: warn at phase 0 when the `generated` timestamp is >30 days
old — the shipped snapshot's counts like any mined kb; never block. Absent
KB: seed from the skill's snapshot first
(`cp "${CLAUDE_PLUGIN_ROOT}/skills/rook-triage/data/kb-snapshot.json" ~/.cache/rook-triage/kb.json`);
only with neither fall back to CODE-OWNERS tiers + per-item
`git log -- <paths>` and say so in the report. A completed refresh should
also update the shipped snapshot via a PR to the plugin repo — one mine
serves every installer.

## Selection (per item)

1. Area(s) → KB candidate pool. For a PR, READ the stamped `areas` phase 0
   wrote — never re-match its paths against `label-map.md`'s table by hand.
   For an issue there is no diff, so derive from that file's keyword layer.
2. Score = recency-decayed (commits + 2×reviews) within the area. Drop:
   the item's author · anyone inactive >6 months · anyone already at
   their per-RUN cap (step 4).
3. Pick: the top-scored candidate + rotate the remainder (item number mod
   pool size) — spreads load; never always-ping the top of git blame.
4. Bounds: PRs → request 2–3 reviewers (hard bounds 1–5), and the set MUST
   include at least one approver-tier member (CODE-OWNERS `approvers:`).
   Reviewer-tier picks may fill any number of the other slots — but never
   all of them. If scoring yields no approver, swap the lowest-scored pick
   for the area's highest-scored approver (overrides still apply; fall
   back to the roster when the area has no approver signal). Formal review
   REQUESTS draw from approver/reviewer tiers only; contributor-tier
   domain experts are @-mention or report-only (they may not be
   requestable on GitHub). Issues → @-mention 1–2 (≤3). Per-person
   per-RUN cap: 3 items across every corpus the run touches (overflow goes
   in the report as "also relevant", never posted). This number has two
   code mirrors and a change has to land in both: `validate-actions`
   re-checks the reviewer and mention BOUNDS before any write, and
   `mdreport.PerPersonCap` is what the ledgers compare against. Nothing else
   checks the cap, so `gen-run-ledger` is where a breach becomes visible —
   and it must be the RUN-wide view, because a person proposed twice in each
   of two corpora reads clean in both per-corpus ledgers and is over the cap
   across the run.
5. `references/routing-overrides.md` wins over all mined data, always.

## Etiquette (encoded, not vibes)

- Labels over people — correct labels route better than more names
  (k8s canon: "assigning excessive reviewers will not yield a quicker
  review").
- One targeted ping ≫ broadcast. Never re-ping the same person within 7
  days; re-ping once, then widen (report/Slack), don't add names.
- Never `/assign` anyone — self-assignment is the contributor's act.

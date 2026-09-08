# Routing and the knowledge base

Rebuilding the KB — its sources, the stages and where judgment is spent,
the assembler's gates and the schema — is `references/kb-refresh.md`. This
file is what routing does with the result.

Freshness: warn at phase 0 when the `generated` timestamp is >30 days
old — the shipped snapshot's counts like any mined kb; never block. Absent
KB: seed from the skill's snapshot first
(`cp "${CLAUDE_PLUGIN_ROOT}/skills/rook-triage/data/kb-snapshot.json" ~/.cache/rook-triage/kb.json`);
only with neither fall back to CODE-OWNERS tiers + per-item
`git log -- <paths>` and say so in the report. That fallback is the last
resort, not an alternative to the seed, because the kb is also what the
tools read: `pool-summary`, `gen-run-ledger` and `validate-actions` take
`--kb` only when the file exists, and without it the approver floor is
SKIPPED rather than passed and the budget is left out. A run that never
seeded is a run whose floor nothing checked. A completed refresh should
also update the shipped snapshot via a PR to the plugin repo — one mine
serves every installer.

## Selection (per item)

1. Area(s) → KB candidate pool. For a PR, READ the stamped `areas` phase 0
   wrote — never re-match its paths against `label-map.md`'s table by hand.
   For an issue there is no diff, so derive from that file's keyword layer.
2. Score = recency-decayed (commits + 2×reviews) within the area. Drop:
   the item's author · anyone inactive >6 months. Not the per-RUN cap: a
   batch-scoped agent cannot see the run, so selection proposes without it
   and step 4 says where it is applied.
3. Pick: the top-scored candidate + rotate the remainder (item number mod
   pool size) — spreads load; never always-ping the top of git blame.
4. Bounds: PRs → request 3–5 reviewers, at least 2 approver-tier
   (CODE-OWNERS `approvers:`); both hard — a set above 5 is the user's own
   request outside the sweep's actions, never a triager's. Reviewer-tier
   picks fill the remaining slots. If scoring yields fewer than the
   approver floor, swap the lowest-scored picks for the area's
   highest-scored approvers (overrides still apply; fall back to the
   roster when the area has no approver signal). Formal review REQUESTS
   draw from approver/reviewer tiers only; contributor-tier domain experts
   are @-mention or report-only (they may not be requestable on GitHub).
   These numbers have one code mirror, `internal/actions`'
   `MinReviewers`/`MaxReviewers`/`MinApprovers`, and a change has to land
   there; `validate-actions --kb`, `validate-kb`'s tier check and
   `ApproverBudget` all read it rather than restating it. Issues →
   @-mention 1–2 (≤3), whose mirror is `MaxMentions`. Per-person per-RUN
   cap: 3 items across every corpus the run touches. Selection proposes
   without it (step 2); it is APPLIED at phase 4, off the run ledger's
   `OVER CAP` status column — swap the over-cap person for the next login
   of that item's `reviewers_alternates`, the ranked remainder selection
   left behind (`agents/rook-triager.md`), and what the swap displaces goes
   in the report as "also relevant", never posted. This number has one code
   mirror, `mdreport.PerPersonCap`, which is what the ledger compares
   against. Nothing else checks the cap, so `gen-run-ledger` is where a
   breach becomes visible — and it must be the RUN-wide view, because a
   person proposed twice in each of two corpora reads clean in both
   per-corpus ledgers and is over the cap across the run. The cap and the
   approver floor together bound a run at `ApproverBudget` PRs:
   `pool-summary` prints it at phase 0, the ledger carries it into the
   report, and the PRs past it queue to the next run. It is an upper
   bound, not a quota — on a `both` run the issue @-mentions draw on the
   same per-person cap, so the PRs a run can actually route is lower.
5. `references/routing-overrides.md` wins over all mined data, always.

## Etiquette (encoded, not vibes)

- Labels over people — correct labels route better than more names
  (k8s canon: "assigning excessive reviewers will not yield a quicker
  review"). Step 4's floor is the priced exception: a PR spans
  subsystems, and enough approvers have to be on the set for one of them
  to be available. At the ceiling the canon governs again.
- One targeted ping ≫ broadcast — @-mentions and re-pings, not step 4's
  review-request set. Never re-ping the same person within 7 days;
  re-ping once, then widen (report/Slack), don't add names.
- Never `/assign` anyone — self-assignment is the contributor's act.

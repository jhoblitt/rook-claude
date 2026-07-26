# Routing and the knowledge base

## kb refresh

Mine, with parallel agents, into `~/.cache/rook-triage/kb.json`:

- `CODE-OWNERS` — the authority roster (approvers/reviewers tiers; the
  file is flat/repo-wide, so per-area truth must be mined, not read).
- `git log` per area path-set (24 months, recency-weighted author counts).
- Merged-PR review history per area — who actually reviews rgw vs csi vs
  osd (merged PRs + reviews, bucketed by changed paths).
  Sample: the last 24 months of merged PRs, capped at 4000, whichever bound
  hits first — record the actual count and oldest date in kb.json
  provenance. This is the PRIMARY reviewer-routing signal.
  The fetch layer is `scripts/rt_fetch.py` (validated on the 2026-07-23
  refresh): a single-cursor walk of `repository.pullRequests` — no
  search-API 1000-cap — emitting `rt_prs.jsonl` + `rt_fetch_state.json`
  (provenance: counted, oldest, stop reason, truncation flags for
  files>100/reviews>30). The analysis layer is `scripts/rt_analyze.py`:
  buckets the JSONL into the v3 area taxonomy (25 areas; recency weights
  1.0/0.5/0.25 at 6/12 months; bots and self-reviews excluded) and emits
  the `{data, flags}` contract below — bucket-ambiguity, truncation,
  spec-boundary, identity-unknown (vs a `--code-owners` roster),
  coverage-gap. Run both rather than letting miners hand-roll them; the
  miner/orchestrator's job starts at resolving the emitted flags.
  Shard for concurrency: split the window into DISJOINT merged-date slices
  via the search API (`is:pr is:merged merged:<from>..<to>`), one miner per
  slice in parallel; every slice must stay under the search API's
  1000-result cap (~6 months at rook's merge rate), and a slice that
  returns 1000 results is presumed truncated — split it and re-mine, don't
  accept it. The assembler is the merge barrier. Disjoint slices fetch each
  PR exactly once — the only added token cost is the extra copies of the
  miner brief.
- Issue participation per area label (who answers what).
- Live `gh label list` (validates `label-map.md`; flag drift).

Mining tiers — miner flags, senior resolves. The reviews miner runs on a
small model (`model: "sonnet"`); it never resolves ambiguity, it flags it,
returning `{data, flags: [{type, item, evidence, question}]}` with types:
`bucket-ambiguity` · `truncation` (query `files(first: 100)` /
`reviews(first: 30)` and flag any remaining `hasNextPage`) ·
`identity-unknown` · `spec-boundary` · `coverage-gap` (vs the prior KB's
signalful areas, provided in the brief). The orchestrator resolves trivial
flags deterministically; surviving flags go to ONE resolver agent on the
session model (re-query access; returns per-flag resolutions the assembler
applies). The assembler validates deterministically regardless of tier —
PR count plausible for the window, no previously-signalful area empty, top
reviewers intersect CODE-OWNERS, provenance consistent with the bounds —
and refuses to write kb.json on failure. Phase-1 triagers and phase-2
refuters never tier down: judgment stays on the session model.

Schema per area: `{paths[], keywords[], labels[], maintainers: [{login,
tier, commits, reviews, issues, last_active}], recent_items[]}` plus a
top-level `generated` timestamp.

Freshness: warn at phase 0 when >30 days old; never block. Absent KB:
seed from the skill's `data/kb-snapshot.json` first; only with neither
fall back to CODE-OWNERS tiers + per-item `git log -- <paths>` and say so
in the report. A completed refresh should also update the shipped
snapshot via a PR to the plugin repo — one mine serves every installer.

## Selection (per item)

1. Area(s) from `label-map.md` paths/keywords → KB candidate pool.
2. Score = recency-decayed (commits + 2×reviews) within the area. Drop:
   the item's author · anyone inactive >6 months · anyone already at
   their per-sweep cap.
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
   per-sweep cap: 3 items (overflow goes in the report as "also
   relevant", never posted).
5. `references/routing-overrides.md` wins over all mined data, always.

## Etiquette (encoded, not vibes)

- Labels over people — correct labels route better than more names
  (k8s canon: "assigning excessive reviewers will not yield a quicker
  review").
- One targeted ping ≫ broadcast. Never re-ping the same person within 7
  days; re-ping once, then widen (report/Slack), don't add names.
- travisn: escalation-class only (security / data-loss / regression /
  stuck >30d) — routine items go to subsystem folks.
- Never `/assign` anyone — self-assignment is the contributor's act.

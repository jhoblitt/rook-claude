# Cross-linking and duplicate discovery

One machinery, four products: issue-dup, PR-dup, issue→fixing-PR,
PR→fixed-issue.

1. **Signal extraction** per item: distinctive keywords · exact error
   strings (quoted) · component terms · changed paths (PRs) or inferred
   area (issues) · the author's other items.
2. **Parallel diverse searches** (the Anthropic `/dedupe` shape): 3–5
   `gh search issues`/`gh search prs` queries per item, each a DIFFERENT
   angle — exact-error-string · symptom keywords · component+kind ·
   path/symbol names · recent same-area items (KB `recent_items`). Search
   open AND recently closed/merged (~90d): "fixed-by-merged" and
   "dup-of-closed" are real dispositions. Budget: the brief states this
   agent's **total search count for the whole batch** — a count, because an
   agent has no clock and cannot pace itself against a rate. The
   orchestrator sets it from the RUN, never from item count: GitHub allows
   30 searches per minute across every agent in flight, so a one-minute
   ceiling divided by the width launched is the per-agent total — about 4
   at the ground-rules width, more as width falls, and the only way to
   raise it is to launch narrower. Phase-2 refuters overlap these batches
   but are NOT in that width: refutation is a READ — of the proposal's own
   item, and of the item it names where it names one — never a search.
   A refuter that finds itself wanting a search has left its mandate. At any real width the budget binds
   well under the 3–5 per item this step would ideally spend, so you are
   choosing WHICH items to search, not how deeply — spend on the ones whose
   signals are most distinctive. A solo pass — a single-item run, or one
   triager with no siblings — spends the full 3–5 per item, and the brief
   says which case applies rather than leaving the agent to infer it from a
   batch it cannot see past.
3. **False-positive filter**: the phase-1 triager judges each surviving
   candidate pair inline — it has no Agent tool, and the independent read
   is phase 2's refutation, which every close-class proposal reaches
   (below). Duplicates must share the ROOT CAUSE, not the symptom;
   fix-links must have a diff that actually addresses the mechanism. Cap:
   3 surviving candidates per item.
4. **Confidence**: CONFIRMED (mechanism verified) may become a proposed
   comment or close; POSSIBLE stays report-only.

Mechanics: a single `#N` mention in a comment creates GitHub's backlink on
both sides — comment on ONE side, never both. `Fixes #N` lines are
suggested to the author by comment; only a takeover (`rook-pr-takeover`)
adds one directly. Every fixed-by-merged or dup close goes through the
phase-2 refutation pass like all closes.

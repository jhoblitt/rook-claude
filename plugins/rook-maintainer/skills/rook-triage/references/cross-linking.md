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
   "dup-of-closed" are real dispositions.
3. **False-positive filter**: an independent agent judges each surviving
   candidate pair — duplicates must share the ROOT CAUSE, not the symptom;
   fix-links must have a diff that actually addresses the mechanism. Cap:
   3 surviving candidates per item.
4. **Confidence**: CONFIRMED (mechanism verified) may become a proposed
   comment or close; POSSIBLE stays report-only.

Mechanics: a single `#N` mention in a comment creates GitHub's backlink on
both sides — comment on ONE side, never both. `Fixes #N` lines are
suggested to the author by comment; only a takeover (rook-code-review)
adds one directly. Every fixed-by-merged or dup close goes through the
phase-2 refutation pass like all closes.

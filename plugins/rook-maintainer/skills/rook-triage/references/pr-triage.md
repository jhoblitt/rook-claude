# PR triage — the cheap sort

NOT review: no diff reading beyond changed paths + size. Substantive PRs
route to `rook-code-review` (hand its sweep an explicit PR list).

**Triage never proposes or applies labels on PRs** — rook does not label
PRs by type/area. Reports show the labels currently present, nothing more;
the only PR-labeling flow anywhere is `rook-code-review`'s backport
flag-and-confirm. (Issue labeling is unaffected — see `label-map.md`.)

Skip conditions:

- **Drafts are skipped by default** — assessed only when the user
  explicitly asks to include drafts.
- **`do-not-merge` is ALWAYS skipped**, even in an include-drafts run —
  the label is an explicit human hold that triage never overrides.
- Also never routed/pinged/commented: WIP title/label · mergify backport
  branches · dependabot/bot authors.
- Every skipped PR still appears in the report as a skip row with its
  reason (draft / do-not-merge / WIP / bot) — silent omission would read
  as coverage.

## Signals per PR

CI rollup (green/red/pending, which checks) · mergeable/conflicts · size
(files, ±lines) · template checklist present AND conforming (verbatim
template, only boxes toggled — the rook-code-review docs-sync canon) ·
links an issue (any closing keyword — `Resolves` is the one rook's PR
template ships) · duplicate-of-open-PR · author trust (association,
history, burst pattern). **Trust changes review depth, never the merge
decision.**

## Card (report format)

URL · What (one line) · Fit (good / mixed / poor for rook scope) · Risk
(paths touched: `pkg/apis`, exec plumbing, secrets/TLS/RBAC, workflows ⇒
high) · Trust · Proof (CI state, claimed repro/tests) · Blocker (typed) ·
Next (one verb).

**Blocker taxonomy:** missing-repro · failing-checks · needs-rebase /
conflicts · first-contribution-awaiting-CI-approval · template-missing /
nonconforming · unlinked-issue · untrusted-diff (burst/AI pattern → deep
review) · duplicate-of-#N · unclear-direction (`needs-design-document`).

**Next verbs:** route-to-deep-review · request-reviewers · needs-rebase
comment · fill-template comment · addresses-#N comment · dup-link
(+ recommend close, refuted first) · recommend-close (out-of-scope /
superseded / spam; refuted first) · flag-takeover-candidate ·
approve-CI-run (report-only; a maintainer act in the UI).

## Reviewer requests

2–3 preferred, 1–5 acceptable — never 0 when routing confidence exists,
never >5, and ALWAYS at least one approver-tier reviewer (CODE-OWNERS
`approvers:`); reviewer-tier members may fill the remaining slots, but a
set of only reviewer-tier picks is invalid. Selection per
`references/routing.md`; execution
`gh pr edit N --add-reviewer a,b` (a gated write like any other).
@-mention fallback only when a reviewer request is not possible.

## Comment templates

Same AI-agent marker rule as issues (rook-conventions "Signing GitHub
comments").

**needs-rebase:** "This has merge conflicts with master — a rebase will
let CI judge the real change. Happy to re-triage after the push."

**fill-template:** "The PR template checklist helps route and review this
— please fill it in verbatim (only the boxes toggled), and link the issue
this addresses if there is one."

**addresses-#N.** Triage reads metadata, not diffs, so it usually cannot tell
whether the PR finishes the issue. Propose the CLOSING form only when the
issue tracks a single item this PR plainly completes; otherwise propose the
non-closing form and leave the completeness question to
`rook-code-review` pass k, which reads the diff.

- *finishes it:* "This looks like it resolves #N — adding `Resolves #N` to
  the description will link them and close the issue on merge."
- *partial or unsure:* "This looks like it addresses part of #N —
  `Part of #N` in the description links them without closing the issue while
  the rest is outstanding."

# PR triage — the cheap sort

NOT review: no diff reading beyond changed paths + size. Substantive PRs
route to `rook-code-review`, one review per PR.

**Triage never proposes or applies labels on PRs** (SKILL.md ground rules)
— rook does not label PRs by type/area at all.

Skip conditions:

- **Drafts are skipped by default** — assessed only when the user
  explicitly asks to include drafts.
- **`do-not-merge` is ALWAYS skipped**, even in an include-drafts run —
  the label is an explicit human hold that triage never overrides.
- Also never routed/pinged/commented: WIP title/label · bot authors, which
  covers mergify's backports and dependabot alike — the classifier matches
  the login, and the snapshot carries no head branch to match instead.
- Every skipped PR still appears in the report as a skip row with its
  reason (draft / do-not-merge / WIP / bot) — silent omission would read
  as coverage. The batched pass below writes those rows to `skips.json`;
  they are never hand-produced.

## Signals per PR

CI rollup (green/red/pending, which checks) · mergeable/conflicts · size
(files, ±lines) · template checklist conformance — `validate-checklist`'s
verdict, never a read of the body, batched once per sweep (below) ·
links an issue (any closing keyword — `Resolves` is the one rook's PR
template ships) · duplicate-of-open-PR · author trust (association,
history, burst pattern). **Trust changes review depth, never the merge
decision.**

### Checklist conformance, batched

One sweep-scoped pass, sequenced by SKILL.md phase 0:

```sh
bash "${CLAUDE_PLUGIN_ROOT}/tools/run.sh" validate-checklist sweep <sweep-dir> \
     --root <rook-checkout>       # --include-drafts on an include-drafts run
```

It writes `<sweep-dir>/checklist.jsonl`, one `{"number":N,"verdict":"..."}`
row per audited PR. Take conformance from that `verdict` and never re-derive
it by reading the body; read `unaudited` as a PR the pass could not audit,
which is NOT `no-checklist`. What the pass itself audits, skips and exits is
its package doc (`tools/cmd/validate-checklist/main.go`); what the verdicts
mean, and the single-PR form, are rook-code-review
`references/docs-sync.md`.

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

Count, bounds, tier rule and selection are `references/routing.md`,
"Selection". Execution is `gh pr edit N --add-reviewer a,b` (a gated write
like any other); @-mention fallback only when a reviewer request is not
possible.

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

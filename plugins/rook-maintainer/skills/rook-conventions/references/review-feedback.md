# Weighing rook/* PR review comments

This file is the normative statement of review-authority weighting — the
roster ladder and how conflicts resolve. Every other mention of it in this
plugin points here and never restates it; `rook-code-review` applies it to
the feedback already on a PR it is reviewing, under the same ladder.

When review opinions conflict, weight by authority (read the repo's root
`CODE-OWNERS` for the roster, don't hardcode it):

1. **travisn** — outranks every other reviewer.
2. **Approvers** (`approvers:` in CODE-OWNERS).
3. **Reviewers** (`reviewers:` in CODE-OWNERS).
4. Everyone else — judged on merit, deferring upward on conflict.

Address every substantive comment; resolve clashes toward higher authority;
if a higher-authority reviewer is plainly wrong on a factual point, flag the
conflict to the maintainer rather than silently overriding either side.

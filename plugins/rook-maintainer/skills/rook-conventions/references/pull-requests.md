# Opening and updating rook/* pull requests

Mechanics for a PR's whole life after the commits exist. Commit messages —
shape, content, and the commitlint traps — are `references/commits.md`;
eligibility for a backport label lives in `references/backporting.md`.

## The PR description

What a description says is the same rule a commit message follows (what
changed and why, never how it was produced — `references/commits.md`). What
differs is the shape. A PR description is, in order:

1. **Motivation** — the problem or need that drove this PR, as the
   maintainer experienced it. For a feature or behavior change whose request
   never stated a motivation, ask the maintainer for it before drafting — a
   rationale reconstructed from the diff reads plausible while missing the
   actual reason.
2. **What changed** — the new user-visible behavior.
3. **Notable decisions** — ONLY a choice a reviewer would otherwise question
   (a deliberate departure from precedent, a trade-off taken), one or two
   sentences each. Usually there are none; omit the section rather than
   writing one to fill it.
4. The AI-assistance disclosure (below) and the repo's PR checklist.

A reviewer gets the point from the first paragraph alone. Hard limit: 100
words across items 1–3 (`wc -w`, markup included) — a ceiling, not a
target; the disclosure and checklist do not count against it. Omit any
section with nothing to say; never pad a slot to make the shape look
complete. When the body outgrows the limit, move detail into commit
messages rather than growing the description.

## Opening a PR

- Run `rook-code-review`'s pre-pr adversarial gate on the branch before
  opening any rook PR; do not open with unresolved blockers unless the
  maintainer explicitly says to.
- Open as a **draft**, from a **fork** (never a branch on the `rook/*` repo),
  assigned to the maintainer (`--assignee @me`, best-effort).
- No `🤖 Generated with [Claude Code]` (or similar) attribution footer in
  `rook/*` PR bodies — override the harness default.
- Fill the PR template checklist against the actual diff:
  - "Documentation has been updated" only when `Documentation/` changed —
    godoc/code comments do NOT count;
  - "Unit tests have been added" only when `_test.go` files outside
    `tests/integration/` changed;
  - "Integration tests have been added" only when integration tests changed;
  - the `PendingReleaseNotes.md` box follows rook-code-review
    `references/docs-sync.md`.
- Experimental / instrumentation-only / not-yet-intended-to-land PRs get the
  `do-not-merge` label at open (best-effort).
- In a multi-PR campaign, open at most ~3 PRs before checking in with the
  maintainer.

## Updating an open PR

Before repushing to an open rook PR, rebase its branch onto the current tip
of the PR's actual base branch (`gh pr view <n> --json baseRefName` — usually
`master`, sometimes `release-*`): fetch the base from the `rook/*` remote,
rebase, resolve, force-push (`--force-with-lease`) to the fork. Never resolve
a stale base with a merge commit from upstream.

Start from the PR's CURRENT remote head, not a stale local branch — a later
session or manual push may have force-updated the fork branch. Re-fetch and
reset to it (confirm via `gh pr view <n> --json headRefOid` +
`git ls-remote <fork> <branch>`) BEFORE rebasing.

When a push substantively changes what the PR does, update the PR description
in the same turn — don't leave it stale.

## rook AI guidelines

PRs must follow `Documentation/Contributing/ai-guidelines.md` (read it when
preparing an AI-assisted PR):

- Disclose AI assistance in the PR description: a short "AI assistance:" note
  on how AI helped, plus the checked "Reviewed AI guidelines" box. ONLY in the
  PR description — never in commit messages. Keep the note strictly about what
  the AI did; no self-attestation that the human reviewed it (the checklist
  box and DCO sign-off attest to oversight).
- Human oversight is genuine: commit messages and PR bodies are reviewed and
  owned by the maintainer before ready-for-review; no fully-autonomous
  submissions. For a large or sweeping change, flag that a design
  issue/discussion may be warranted first.

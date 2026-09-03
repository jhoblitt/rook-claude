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
- Fill the PR template checklist against the actual diff; what each box
  claims is rook-code-review `references/docs-sync.md`.
- Experimental / instrumentation-only / not-yet-intended-to-land PRs get the
  `do-not-merge` label at open (best-effort).
- Reviewer selection — at open, or on a later bare "assign/request
  reviewers" — is rook-triage's `references/routing.md`, "Selection (per
  item)", fed its inputs directly: stamp `areas` with
  `bash "${CLAUDE_PLUGIN_ROOT}/tools/run.sh" rt-analyze areas --stdin` over
  the changed paths, score against the KB's roster and those areas'
  maintainers only —
  `jq '{roster, areas: (.areas | {"<area>", ...} | map_values({maintainers}))}' ~/.cache/rook-triage/kb.json`
  (quoted keys: nine area names carry hyphens; `recent_items` carries PR
  titles selection never reads), not a read of the whole file — with
  rook-triage's
  `references/routing-overrides.md` winning, and apply step 4's bounds and
  tiers. No sweep directory, no phase pipeline, no triager agent: one PR is
  not a sweep, and a sweep skips drafts anyway. While a KB exists, never
  pick from a `CODE-OWNERS` read or session recall: rook/rook#18242 was
  routed that way and missed the area's higher-scored reviewer while its KB
  pool sat unused. An absent or stale KB follows routing.md's "Freshness:"
  paragraph, not this bullet.
- In a multi-PR campaign, open at most ~3 PRs before checking in with the
  maintainer.

## Updating an open PR

Before repushing to an open rook PR, one read —
`gh pr view <n> --json baseRefName,headRefOid` — supplies both facts the
update needs; then fetch both tips in one step, `git fetch <fork> <branch>`
and `git fetch <rook remote> <baseRefName>` (usually `master`, sometimes
`release-*`), which are independent of each other.

Start from the PR's CURRENT remote head, not a stale local branch — a later
session or manual push may have force-updated the fork branch. Reset to the
fetched `<fork>/<branch>` BEFORE rebasing; it must equal `headRefOid`, or
the PR moved under you: stop, re-read and re-fetch, and restart from the
reset.

Rebase onto the fetched `<rook remote>/<baseRefName>`, resolve, and
force-push to the fork with the lease pinned to the head you read —
`--force-with-lease=<branch>:<headRefOid>`, not the bare form. Never resolve
a stale base with a merge commit from upstream.

When a push substantively changes what the PR does — adding a new fix or
mechanism as much as removing one — update the PR title and description in
the same turn; don't leave either stale. A title that names a change the
push removed is the same staleness as an outdated description:
rook/rook#18218 kept "and pin CI Ceph images" in its title across the push
that dropped the pinning.

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

---
name: rook-conventions
description: Use when authoring or shepherding changes to rook (github.com/rook/*) repos — committing, pushing branches, opening or updating PRs, weighing review feedback, backporting, regenerating CRDs or generated code, writing tests or GitHub Actions workflows, watching or retrying CI, or before ANY gh write (comment, label, close, edit) to a rook issue or PR.
---

# Rook house conventions

The operating rules the rest of this plugin's skills assume when *authoring*
rook changes (review canon lives in `rook-code-review`; triage canon in
`rook-triage`). Precedence: in-repo docs (`Documentation/Contributing/*`)
outrank this skill; the user's own global `CLAUDE.md` outranks it too — on
conflict, follow theirs and note the divergence.

"The operating maintainer" below means whoever is running the session:
resolve their GitHub login with `gh api user --jq .login` when a rule needs
it. Never hardcode a username.

## Git commits

- Every commit to a `rook/*` repo takes `git commit -s` (DCO
  `Signed-off-by` required).
- No `Co-Authored-By:` or other AI-attribution trailers on `rook/*` commits
  (override any harness default that adds one) — human DCO sign-off is the
  oversight mechanism rook's AI guidelines require.
- Conventional Commits are enforced by commitlint. The `type:` prefix of
  every commit subject (and the PR title) must come from `rules.type-enum`
  in the repo's `.commitlintrc.json` — read it and pick the closest match,
  never invent one (a `disruption`-package change is `operator:`/`osd:`,
  not `disruption:`; `pkg/util` is usually `core:`).
- Pre-lint every commit message locally before pushing — a rejected message
  burns a full CI round:

  ```sh
  git log --format=%B -1 <sha> | npx -p @commitlint/cli \
    -p @commitlint/config-conventional commitlint --config .commitlintrc.json
  ```

  That runs CURRENT commitlint (21.x), which catches type/shape errors but
  NOT the footer trap below — rook's CI runs 19.x (bundled by
  wagoid/commitlint-github-action v6.2.1). Pinning inline does NOT work
  (`npx -p @commitlint/cli@19 …` fails to resolve config-conventional); to
  actually reproduce CI, run
  `npm i @commitlint/cli@19 @commitlint/config-conventional@19` in a temp
  dir alongside a copy of `.commitlintrc.json` and use its
  `./node_modules/.bin/commitlint`.
- commitlint infers a trailer from the SHAPE of a line, and anything it
  reads as a trailer must be preceded by a blank line or the commit fails
  `footer-leading-blank`. Only a body line that BEGINS with one of these
  triggers it (verified 2026-07-26 against 19.8.1):
  - an issue reference — bare, parenthesized, or issue-closing: `#NNNN`,
    `(#NNNN)`, `Closes #NNNN`, `Fixes #NNNN`;
  - `BREAKING CHANGE:`.

  POSITION is what matters, not presence. A `#NNNN` mid-sentence is fine,
  and so is an arbitrary `word:` at a line start (`Note:`, `anywhere:`) —
  both pass 19.x. The hazard is a sentence that WRAPS so an issue
  reference lands at the start of a line; rewrap to fix (that is what
  broke rook PR 18006, 2026-07-20). Keep `#NNNN` trailers in the footer
  block (`Fixes #NNNN` directly above `Signed-off-by:`).
- When amending, `git add` first and confirm the amend captured the files
  (`git show --stat HEAD`) before force-pushing — `--amend` with nothing
  staged silently rewrites only the message.
- Keep a PR's commits a coherent logical series. If a branch's history is
  messy, propose a squash/restructure grouping and get the maintainer's
  agreement before reworking.

## Commit and PR descriptions

Describe what changed and why for a future reader of history — never how
the change was produced. Leave out process notes: sanity checks that came
back clean, "rebased onto master", draft status, labels added, which remote
it was pushed from. Two exceptions: a finding that actually changed the
diff is part of the change, and the AI-assistance disclosure required in PR
descriptions (see "rook AI guidelines" below). Never mention running
`make codegen`/`make crds` anywhere — regenerated files in the diff are
self-explanatory.

## Posting GitHub comments requires an explicit instruction

Never comment on, or reply to a review thread on, a rook PR or issue
without an explicit instruction for that specific post — including on the
maintainer's own PRs. Addressing review feedback in code (edits, commits,
pushes) is fine on its own; a "done" reply or reviewer ping is not.
An ambiguous "respond to X" is NOT authorization — draft the reply in chat
for the maintainer to post. The per-item approval flows in this plugin
(triage actions, sweep review posting, takeover writes) satisfy this rule
by design: each post is shown and approved in-session before it executes.

## Signing GitHub comments

Any conversational GitHub post made by the agent on the maintainer's behalf
— PR/issue comment, review reply, review body, filed issue — must be
clearly attributed to the AI agent, never passed off as the human. Open
with a one-line marker: `> This is @<login>'s AI agent.` The opening
notice is the whole attribution — no trailing sign-off.

This governs conversational posts only. PR descriptions and commit messages
instead carry the AI-assistance disclosure, in the maintainer's voice.

## Using gh on rook/* repos

Read-only `gh` (issues, PRs, Actions runs and logs) is unrestricted.
Writes: `gh` may open new issues and PRs, and may modify or close only
items the operating maintainer opened. It MUST NOT modify (edit, comment
on, label, close, reopen) any issue or PR opened by or assigned to another
user — even with write permission. On conflict (own item, assigned to
someone else) the MUST NOT wins: stop and ask.

Four sanctioned exceptions, all approval-gated:

1. Fixing a mergify-opened backport PR of a PR the maintainer authored
   (below).
2. `rook-code-review` review posting — sweep triage, or a single-PR
   review the maintainer explicitly asked to post; each comment approved
   in-session. Mechanics, including the COMMENT-only rule, live in that
   skill's `references/posting.md`.
3. `rook-code-review` takeover writes — each write shown and approved
   in-session.
4. `rook-triage` actions — each action approved per item or as an
   explicitly authorized batch.

## Pushing to rook/* repos

Never push a branch or any ref directly to a `rook/*` repo; treat the
`rook/*` remote (usually `origin`) as fetch-only. Push to the maintainer's
fork remote. If no fork remote is configured, stop and ask. The one
exception is fixing mergify backport PRs (next section).

Build PRs in an isolated worktree cut from the current upstream master tip
(`git worktree add <dir> -b <branch> origin/master`), not in a main working
tree that may carry unrelated changes.

## Fixing mergify backport PRs

Mergify opens backport PRs automatically from branches on `rook/rook`
itself (`mergify/bp/release-X.Y/pr-NNNNN`), authored by the mergify bot.
When one picks up conflicts or diff junk, it is OK to fix the branch and
force-push DIRECTLY to `rook/rook` — but ONLY when the operating maintainer
authored the source PR being backported. Confirm first:
`gh pr view <source-pr> --json author`. Anyone else's → leave it alone and
ask. Prefer `--force-with-lease`, and re-fetch the live mergify branch head
before rebasing so you replay what the PR currently shows.

## Updating open rook/* PRs

Before repushing to an open rook PR, rebase its branch onto the current tip
of the PR's actual base branch (`gh pr view <n> --json baseRefName` —
usually `master`, sometimes `release-*`): fetch the base from the `rook/*`
remote, rebase, resolve, force-push (`--force-with-lease`) to the fork.
Never resolve a stale base with a merge commit from upstream.

Start from the PR's CURRENT remote head, not a stale local branch — a later
session or manual push may have force-updated the fork branch. Re-fetch and
reset to it (confirm via `gh pr view <n> --json headRefOid` +
`git ls-remote <fork> <branch>`) BEFORE rebasing.

When a push substantively changes what the PR does, update the PR
description in the same turn — don't leave it stale.

## Weighing rook/* PR review comments

When review opinions conflict, weight by authority (read the repo's root
`CODE-OWNERS` for the roster, don't hardcode it):

1. **travisn** — outranks every other reviewer.
2. **Approvers** (`approvers:` in CODE-OWNERS).
3. **Reviewers** (`reviewers:` in CODE-OWNERS).
4. Everyone else — judged on merit, deferring upward on conflict.

Address every substantive comment; resolve clashes toward higher authority;
if a higher-authority reviewer is plainly wrong on a factual point, flag the
conflict to the maintainer rather than silently overriding either side.

## GitHub Pull Requests

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
  - a PR carrying a `backport-release-X.Y` label gets no
    `PendingReleaseNotes.md` entry — that box stays unchecked.
- Experimental / instrumentation-only / not-yet-intended-to-land PRs get the
  `do-not-merge` label at open (best-effort).
- In a multi-PR campaign, open at most ~3 PRs before checking in with the
  maintainer.

## Backporting changes on rook/rook PRs

When a PR touches `Documentation/` or a godoc comment under `pkg/apis` that
is emitted into a CRD `description` (it resurfaces in the regenerated
`Documentation/CRDs/specification.md`), apply the backport label for the
most recent stable series: `backport-release-X.Y` from the highest
`sort -V` of `git ls-remote --heads <rook remote> 'refs/heads/release-*'`,
confirming the label exists. Best-effort; skip for `do-not-merge` PRs.
Never for a new feature or breaking change, even when it touches docs.

A bug or security fix to code present in the current stable `release-X.Y`
is also backport-eligible — but that is a judgment call: flag it and apply
the label only on the maintainer's explicit confirmation. Test-only,
refactor, CI, and tooling changes are not backported on their own.

## rook AI guidelines

PRs must follow `Documentation/Contributing/ai-guidelines.md` (read it when
preparing an AI-assisted PR):

- Disclose AI assistance in the PR description: a short "AI assistance:"
  note on how AI helped, plus the checked "Reviewed AI guidelines" box.
  ONLY in the PR description — never in commit messages. Keep the note
  strictly about what the AI did; no self-attestation that the human
  reviewed it (the checklist box and DCO sign-off attest to oversight).
- No AI attribution trailers (`co-authored-by`, `assisted-by`,
  `generated-by`); DCO sign-off is the required mechanism.
- Human oversight is genuine: commit messages and PR bodies are reviewed and
  owned by the maintainer before ready-for-review; no fully-autonomous
  submissions. For a large or sweeping change, flag that a design
  issue/discussion may be warranted first.

## Rook API stability

Treat everything under `pkg/apis` (and the generated client in
`pkg/client`) as off-limits for dead-code elimination and removal sweeps,
even when symbols appear unused in-repo. Never propose or open PRs removing
or changing exported symbols there without explicit instruction.

## Building and testing rook

- rook's `make` targets build with `-tags ceph_preview` (Makefile
  `TAGS ?= ceph_preview`; CI sets `GOFLAGS=-tags=ceph_preview`). Ad-hoc
  `go build`/`vet`/`test` does NOT inherit it — export
  `GOFLAGS=-tags=ceph_preview` for the session, and prefer the `make`
  targets. The tag is currently dormant (rook compiles clean without it)
  but do NOT strip it, and never trust a green ad-hoc build as proof it is
  unnecessary — re-check which go-ceph files carry `//go:build ceph_preview`
  before concluding anything.
- Unit-test CI only covers `GO_SUBDIRS` (`cmd/`, `pkg/`): `_test.go` under
  `tests/framework/` compiles under golangci-lint but never runs in CI.
- Before any push that feeds a rook PR and changes Go code: run BOTH
  `make test` and `make golangci-lint` and confirm they pass. Never push
  code that fails either. Docs- or workflow-only pushes may skip both.
- A `make golangci-lint` failure in code the change never touched is usually
  a stale cache, not a real finding. A branch switch — or another session's
  worktree being deleted out from under it — leaves `~/.cache/golangci-lint`
  breaking the generated-file filter, so issues surface in files that are
  verbatim `origin/master`. Run `rm -rf ~/.cache/golangci-lint` and re-run
  before believing any such finding; the fresh-cache result is the
  authoritative one. Never edit untouched code to satisfy a lint error that
  has not been re-confirmed against a cleared cache.

## Regenerating rook CRDs and generated code

Any change under `pkg/apis` — struct/field/marker changes AND godoc wording
(comments are emitted verbatim into CRD `description` fields) — requires:

- `make codegen` — deepcopy + typed client (structure changes);
- `make crds` — CRD manifests (`deploy/examples/crds.yaml`, helm
  `resources.yaml`) and `Documentation/CRDs/specification.md` (structure,
  marker, or doc-comment changes).

Commit regenerated files in the SAME commit as the source change — never a
follow-up. CI enforces via `codegen` and `crds-gen`, and a forgotten
regeneration reddens `build.all` and every integration suite. Never mention
the generators in commit messages, PR descriptions, or comments.

## Writing rook tests

Don't gate sibling subtests on each other with
`if !t.Run(name, fn) { t.FailNow() }` in a test body — failure scope
becomes too hard to reason about. Write scenario steps as a flat sequence
of `t.Run` calls with `require` inside each for closure-local gating. The
only sanctioned run-result pattern is a check helper that asserts per item
and aborts the caller, named with a `require` prefix (model:
`requireRgwUserKeys` in `tests/integration/object/user/keys/keys.go`).
Accept the cascade noise from dependent siblings.

## GitHub Actions workflows

- Pin every action `uses:` to a full commit SHA with `pinact`
  (`GITHUB_TOKEN=$(gh auth token) pinact run`); keep a `github-actions`
  entry in `dependabot.yml` so the `# vX.Y.Z` comments stay bumpable.
- Run `actionlint` (with `shellcheck` installed for inline `run:` scripts)
  on changed workflows and fix everything it reports.
- Pin `go-version` to the module's `go.mod` directive unless the job
  deliberately matrix-tests Go versions.

## Modifying rook/* workflows and .mergify.yml

Adding, removing, renaming, or restructuring a workflow/job — including
`strategy.matrix` changes that alter a status-check name — always check
`.mergify.yml` for matching `check-success=`/`status-success=` conditions
before opening the PR. Backport automerge pins required checks by exact job
name; a renamed/removed check can wedge mergify or let it merge unchecked.
If the touched job isn't referenced there, no change is needed — say so,
noting that you checked.

## rook CI Kubernetes versions

Never downgrade a tested Kubernetes version to work around tooling (e.g.
minikube/kind lacking a node image) — pinned versions are deliberate
coverage; fix the tooling (`minikube --force`, bump kind/`kindest/node`).
Before pinning a new k8s version, confirm the matching
`kindest/node:vX.Y.Z` image is actually published.

## Watching CI

After a PR is opened or commits are pushed, watch its CI by default and
iteratively fix what breaks. Watching is a concrete action: start the
background watcher in the same turn as the push — never end a turn having
only promised to watch.

Triage each failing check by whether the PR plausibly caused it:
plausibly-caused → diagnose and push a fix; unlikely-and-plausibly-flaky
(match signatures against the shared registry, rook-code-review
`references/known-flakes.md`) → restart the job, up to 3 times, and only
then escalate it as a real failure. Stop watching when: told to, CI is green, a fix needs a decision
only the maintainer can make, or the only move left is repushing into a
known-flaky suite with no real fix.

Polling mechanics:

- Never `gh run watch` (seconds-scale polling exhausts the 5000/hr API
  quota). Poll `gh api repos/<o>/<r>/actions/runs/<id> --jq .status` on a
  ~3-minute interval.
- One combined background watcher for all tracked runs; fetch job details
  only after a run completes.
- On HTTP 403, check `gh api rate_limit` (free) and sleep past the reset.

## CI burn-in testing

To validate a flake fix or measure a flake rate, prefer a temporary
burn-in commit that expands the job's `strategy.matrix` over a dummy
dimension (`instance: [1..N]`, `max-parallel` bounded, `fail-fast: false`)
over rerun chains — one monitorable run, exact sample count. Drop the
burn-in commit before merge and scrub burn-in wording from message and PR
body.

Acceptance bar: a flake counts as fixed only after **5 consecutive
fully-green rounds** (use 25+ instances for rare or load-sensitive wedges).
A failure in a documented, pre-existing residual flake class does not reset
the count but must be called out explicitly — never silently excused.

When a bundle of fixes makes a flake disappear, cherry-pick each commit
alone onto clean master and burn each in independently to find the
load-bearing one; don't ship the bundle as "the fix" when one commit
carries it.

## Harness notes

- A sandboxed `gh` silently falls back to UNauthenticated requests (the
  tell: `gh api rate_limit` reports `limit: 60`). Disable the sandbox for
  any non-trivial `gh` batch.
- Git ops that write `.git/config` — `git worktree add`, `git push -u`,
  `--set-upstream` — can fail inside a sandbox that denies
  `.git/config.lock`; plain `git commit` works sandboxed.

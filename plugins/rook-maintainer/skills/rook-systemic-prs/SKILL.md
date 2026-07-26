---
name: rook-systemic-prs
description: 'Use when applying one systemic change across a rook (github.com/rook/*) repo as a series of small, well-contained, independently reviewable PRs — dead-code elimination, lint/staticcheck cleanups, API migrations, renames, import or dependency hygiene. Triggers include: "sweep the rook repo for X", "break this into small PRs", "dead code elimination campaign", "iteratively clean up rook", "find another N PRs worth of <change>".'
---

# Rook systemic change → small PRs

A repeatable loop for applying a *systemic* change (one rule applied in many
places) to a `github.com/rook/*` repo, delivered as many **small, isolated,
independently mergeable PRs** instead of one mega-PR. Optimized for **aggressive
subagent fan-out**: scanning and preparation are parallelized across as many
subagents as the work decomposes into.

This skill encodes the *process*. The driving example throughout is **dead-code
elimination** (the workflow it was hardened on), but the same loop applies to any
sweeping change: replace a deprecated call, rename a symbol, tighten a lint,
normalize imports, bump a vendored API, etc. Substitute the "find candidates" and
"transform" steps for your change; the sync / exclude-open-PRs / propose-gate /
per-PR-verify / conventions machinery is identical.

## Core principles

1. **Upstream master is the default ref.** Always scan, branch, and base PRs on
   the *current upstream* `master` (the `rook/*` repo), not a stale local branch.
   Sync first, every run. Candidates found against stale code waste everyone's
   time and may already be merged.
2. **Exclude work already in flight.** Before proposing anything, enumerate open
   PRs and drop any candidate already covered by one. Re-proposing open work is
   the most common failure mode of a multi-session campaign.
3. **Propose, then get agreement, before opening PRs.** Present the candidate
   list and wait for the user to approve which ones to open. Do not open PRs
   speculatively.
4. **One concern per PR.** Each PR should be a single file deletion, a single
   dead symbol/cluster, or one mechanical transform in one area — reviewable in
   under a minute. Prefer whole-file deletes when an entire file is dead.
5. **Fan out aggressively.** Decompose by directory / package / candidate file
   and run one subagent per unit, in parallel, in a single message. Use read-only
   `Explore` agents for scanning/auditing; use `rook-maintainer:code-worker`
   agents (shipped with this plugin) with `isolation: worktree` for parallel
   implementation across independent files.
6. **Verify every PR independently.** Build + vet the affected package(s) before
   committing. A green campaign is many green PRs, not one hopeful push.

## The loop

### Phase 0 — Sync to upstream master
- Confirm the upstream remote (usually `origin` → `https://github.com/rook/rook.git`)
  and the user's fork remote (e.g. named for their GitHub login →
  `git@github.com:<login>/rook.git`).
  `git remote -v` to check.
- Update master: `git fetch origin && git checkout master && git merge --ff-only origin/master`
  (or `git pull --ff-only`). All candidate branches are cut from this.
- Note the master tip; if prior campaign PRs merged, their dead code is already
  gone — re-scanning against fresh master prevents re-proposing them.

### Phase 1 — Define the change and scan (fan out)
- State the systemic rule precisely (e.g. "symbols with zero repo-wide
  references, incl. tests").
- Enumerate the work units to fan out over — usually immediate subdirectories of
  the target tree (`find <tree> -maxdepth 1 -mindepth 1 -type d`).
- Spawn **one subagent per unit, all in one message**, to scan and report
  candidates. For dead code, give each agent the recipe in "Tooling" below and
  have it return a concise candidate list (file:line, signature, exported?,
  confidence + reason) or "NO CANDIDATES".
- Also run a single authoritative whole-module pass yourself to cross-check the
  fan-out (for dead code: `deadcode` — see Tooling). Tools and agents have blind
  spots; the union minus false positives is the candidate set.

### Phase 2 — Exclude already-open PRs
- List open PRs touching this repo from **both** the fork and upstream:
  - `gh pr list --repo rook/rook --state open --limit 200 --json number,title,headRefName,files`
  - Include the user's own branches: `gh pr list --repo rook/rook --author @me --state open ...`
- Drop any candidate whose file/symbol is already changed by an open PR (match on
  changed file paths and on symbol name in the PR diff/title). Note in your
  proposal which candidates were excluded and why.
- Also skip local branches that are pushed-but-not-yet-merged for the same work.

### Phase 3 — Propose and get agreement (gate)
- Present the surviving candidates grouped into proposed PRs: for each, the
  file(s), the symbols, the evidence (how verified), the commitlint `type:`, and a
  risk note (e.g. "exported util that *could* be intended for external use").
- Call out anything you deliberately excluded as too risky (e.g. `pkg/apis/*`
  public CRD API; `tests/framework/*` which can be over-reported behind build
  tags).
- **Wait for explicit approval** (an `AskUserQuestion` multiselect works well) on
  which PRs to open. Do not proceed past this gate unprompted.

### Phase 4 — Implement each approved PR (fan out) and open it
For independent files/areas, fan out `rook-maintainer:code-worker` agents with
`isolation: worktree` so they don't collide; otherwise do them sequentially. For
each PR:
1. Branch from fresh master: `git checkout master && git checkout -b maint-<short-desc>`.
2. Apply the change (delete the file with `git rm`, or edit out the symbol +
   clean up now-unused imports).
3. **Verify**: `go build` and `go vet` on the affected package(s) — with the
   build tag (see Conventions). `gofmt -l` should be clean.
4. Re-grep to confirm zero remaining references repo-wide.
5. Commit and push (see Conventions), then open a **draft PR assigned to the user**
   with the correct `type:` (see Conventions).
- After opening, report the PR table (number, scope, type, net diff).

### Phase 5 — Iterate
- The remaining candidate pool persists across sessions. When asked for "another
  N PRs", restart at Phase 0 (re-sync, re-exclude now-open PRs) and pull the next
  N from the pool.

## Example invocations

- **"Look for dead symbols/funcs under `pkg/operator/ceph/<dir>/` not used
  anywhere in the repo."** — Single-area scan. Phase 1 on one directory (still
  run `deadcode` + `staticcheck` + the exported/write-only audit), report
  candidates. No PR unless asked.
- **"Iterate through subdirs of `pkg/operator/ceph` until you find one with dead
  code, then propose removals."** — Fan out one Explore agent per subdir
  (Phase 1), stop at the first (alphabetically, skipping already-handled) with
  candidates, present them (Phase 3 gate).
- **"Remove `<symbol>` and open a PR."** — Skip to Phase 4 for that one symbol:
  branch off fresh master, edit, build/vet, commit (DCO, no co-author, right
  `type:`), push, open a draft PR assigned to the user.
- **"Find another 3 PRs worth of isolated dead-code elimination."** — Full loop:
  re-sync master (Phase 0), whole-module `deadcode` + fan-out (Phase 1), exclude
  open PRs and pushed-but-unmerged branches (Phase 2), propose ~3 well-contained
  groups and get agreement (Phase 3), then implement + open drafts (Phase 4).
- **"Sweep the repo to replace `<deprecated call>` with `<replacement>`."** — Same
  loop with a different detector (grep/ast-grep/staticcheck) and a mechanical
  transform; one PR per package or per logical batch.

## Rook conventions (hard requirements)

The authoring rules — DCO sign-off (`-s`), no `Co-Authored-By:` trailer,
commitlint `type:` from the repo's `.commitlintrc.json` (closest match,
never invented), draft PRs from a fork assigned to the user
(`gh pr create --draft --assignee @me`, best-effort assignment), no
AI-attribution footer, the verbatim PR-template checklist — are canon in
the rook-conventions skill (this plugin). Load it before committing or
opening anything; the user's own global CLAUDE.md outranks it on conflict.

Campaign-specific delta: ANALYSIS tools need the build tag exactly like
builds do — `deadcode`, `staticcheck`, and `go vet` all take
`-tags=ceph_preview`, or analysis can abort with `undefined:` errors
(details in `references/dead-code.md`).

## Tooling

Detector recipes live under `references/` — for dead-code campaigns read
`references/dead-code.md` (deadcode/staticcheck invocations, the
exported-symbol and write-only-field gaps both tools miss,
whole-file-delete rules, sandbox and build-tag caveats). For any other
campaign, substitute your detector (a `staticcheck` check, a
`grep`/`ast-grep` pattern, a `golangci-lint` linter) and the mechanical
transform, keeping everything else in the loop identical.

## Fan-out patterns

- **Scan**: one `Explore` agent per directory, all spawned in a single message.
  Each runs the detector for its dir and returns a structured candidate list.
  Give every agent the same definition of "dead/violating" and the build-tag and
  sandbox caveats, so results are comparable.
- **Implement**: one `rook-maintainer:code-worker` agent per independent
  file/area with `isolation: worktree` to avoid working-tree collisions,
  spawned together. Keep
  push + `gh pr create` under your own control (consistent conventions, DCO,
  force-push safety) rather than delegating them, unless the agents are reliably
  scripted for it.
- **Cross-check**: pair the fan-out with one authoritative whole-module pass and
  reconcile — discrepancies are usually a tool blind spot (e.g. a method on an
  instantiated type) worth investigating, not noise to ignore.

## Gotchas learned the hard way

- Always re-sync and re-run exclusion each session; merged PRs change the pool.
- `--force-with-lease` (not `--force`) when amending pushed campaign branches.
- Never edit the user's global `CLAUDE.md` (or other personal config)
  unilaterally — present the proposed wording and wait for approval.
- Writing under the user's Claude config dir (`~/.claude` by default) needs
  the sandbox disabled.
- Untracked char-device dotfiles (`.bashrc` etc. as `crw-rw-rw-`) in the repo are
  sandbox artifacts — never `git add` or clean them. `autostash` stash entries are
  the user's, not yours — never pop/drop them.

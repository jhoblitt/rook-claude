---
name: rook-systemic-prs
description: 'Use when applying one systemic change across a rook (github.com/rook/*) repo as a series of small, well-contained, independently reviewable PRs — dead-code elimination, lint/staticcheck cleanups, API migrations, renames, import or dependency hygiene. Triggers include: "sweep the rook repo for X", "break this into small PRs", "dead code elimination campaign", "iteratively clean up rook", "find another N PRs worth of <change>".'
---

# Rook systemic change → small PRs

A repeatable loop for applying a *systemic* change (one rule applied in many
places) to a `github.com/rook/*` repo, delivered as many **small, isolated,
independently mergeable PRs** instead of one mega-PR. Optimized for **aggressive
subagent fan-out**: scanning and preparation are parallelized across as many
subagents as the work decomposes into, run at the harness width cap.

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
5. **Fan out aggressively, up to the width cap.** Decompose by directory /
   package / candidate file and run one subagent per unit; dispatch and
   width are rook-conventions `references/fan-out.md`. Use read-only
   `Explore` agents for scanning/auditing; use `rook-maintainer:code-worker`
   agents (shipped with this plugin) with `isolation: worktree` for parallel
   implementation across independent files.
6. **Verify every PR independently.** Build + vet the affected package(s) before
   committing. A green campaign is many green PRs, not one hopeful push.
7. **Tier each agent to its work.** Scan agents run a detector and report
   `file:line` — small-model work, and the whole-module cross-check in Phase 1
   is what backstops it. Implementation workers apply an already-approved
   mechanical transform — small-model work too, backstopped by Phase 4's
   build/vet/gofmt and re-grep gate. Pass `model:` explicitly at both call
   sites instead of inheriting the session model. What stays on the session
   model is the judgment no gate can check: deciding what is safe to remove
   (Phase 3) and reconciling the fan-out against the authoritative pass.

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
- Spawn one subagent per unit, dispatched per rook-conventions
  `references/fan-out.md`, to scan and report candidates. For dead code, give
  each agent the recipe in "Tooling" below and have it return a concise
  candidate list (file:line, signature, exported?, confidence + reason) or
  "NO CANDIDATES".
- Also run a single authoritative whole-module pass yourself to cross-check the
  fan-out (for dead code: `deadcode` — see Tooling). Tools and agents have blind
  spots; the union minus false positives is the candidate set.

### Phase 2 — Exclude already-open PRs
- Fetch every open PR to DISK, then join against it there — this is a set join
  whose only useful output is the collisions, so nothing else may enter context.
  The fetch ships with this plugin, and it re-paginates each PR's `files[]`: rook PRs
  routinely run past GraphQL's 100-file page, where a `gh pr list --json files`
  silently truncates the very file sets the exclusion reads.

  ```sh
  bash "${CLAUDE_PLUGIN_ROOT}/tools/run.sh" sweep-prefetch snapshot <sweep-dir> --kind prs
  jq -r --rawfile paths candidates.txt '($paths|split("\n")-[""]) as $c
    | .items[] | select(any(.files[]; IN($c[])))
    | "#\(.number) \(.author) — \([.files[]|select(IN($c[]))]|join(" "))"
  ' <sweep-dir>/snapshot.json
  ```

  `<sweep-dir>` is scratch (under `$TMPDIR`), `candidates.txt` is one candidate
  path per line, and `--repo` defaults to `rook/rook`. The snapshot needs
  authenticated `gh`, so run it with the sandbox disabled (rook-conventions
  SKILL.md "Harness notes"). It holds every open PR whatever the author, so the
  user's own in-flight work falls out of the same join — `author` names whose
  each hit is — and no `--author @me` re-list is needed.
- Drop every candidate the join hits. Match symbol-level candidates against the
  same file's titles rather than re-reading it, matching INSIDE `jq` so the
  titles themselves never land in context — a PR title is contributor-authored
  and this projection carries no fence:

  ```sh
  jq -r --rawfile syms symbols.txt '($syms|split("\n")-[""]) as $s
    | .items[] | select(.title as $t | any($s[]; inside($t)))
    | "#\(.number)"
  ' <sweep-dir>/snapshot.json
  ```

  Note in your proposal which candidates were excluded and why.
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
`isolation: worktree` so they don't collide; otherwise do them sequentially.
Ownership splits at that worktree boundary: a worker edits and verifies inside
its own tree and stops there, and every write that reaches beyond it — branch,
commit, push, PR — is yours. Both halves are enforced elsewhere, so state the
split in each worker's brief rather than assuming it reads its own definition:
`agents/code-worker.md` forbids a worker to push or create a PR, and
rook-conventions SKILL.md ("Pushing to rook/* repos") builds each PR in a
worktree cut from the upstream master tip, "not in a main working tree that may
carry unrelated changes" — which is what a `git checkout` in the one shared
clone retargets, out from under every sibling.

**Yours, per approved PR:**
1. Spawn the worker with `isolation: worktree`, which is what allocates that
   tree — so no branch command runs in the shared clone. The worker reports
   its path back (below); that tree is the PR's branch, and steps 2 and 3
   run in it.
2. On the worker's report, run the push gate in that tree — `make test` and
   `make golangci-lint` (rook-conventions `references/building-and-testing.md`)
   — one branch at a time. It stays here for a second reason: that file's
   remedy for a stale-cache failure, `rm -rf ~/.cache/golangci-lint`, wipes a
   machine-global cache, so N workers running the gate concurrently corrupt each
   other's runs.
3. Commit there and push to the fork (`git push <fork> HEAD:maint-<short-desc>`),
   then open a **draft PR assigned to the user** with the correct `type:` — all
   per Conventions. Report the PR table (number, scope, type, net diff).

**The worker's, all of it inside its own worktree:**
1. Apply the change (delete the file with `git rm`, or edit out the symbol +
   clean up now-unused imports).
2. **Verify**: `go build` and `go vet` on the affected package(s) — with the
   build tag (see Conventions). `gofmt -l` should be clean.
3. Re-grep to confirm zero remaining references repo-wide.
4. Report per `agents/code-worker.md`'s report contract, leaving the tree
   uncommitted.

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
- **"Remove `<symbol>` and open a PR."** — Skip to Phase 4 for that one
  symbol, and run it as written: the ownership split applies to a batch of
  one exactly as it does to a batch of three.
- **"Find another 3 PRs worth of isolated dead-code elimination."** — Full loop:
  re-sync master (Phase 0), whole-module `deadcode` + fan-out (Phase 1), exclude
  open PRs and pushed-but-unmerged branches (Phase 2), propose ~3 well-contained
  groups and get agreement (Phase 3), then implement + open drafts (Phase 4).
- **"Sweep the repo to replace `<deprecated call>` with `<replacement>`."** — Same
  loop with a different detector (grep/ast-grep/staticcheck) and a mechanical
  transform; one PR per package or per logical batch.

## Rook conventions (hard requirements)

The authoring rules — commit sign-off and message format, PR mechanics
(draft, from a fork, assigned to the maintainer), and the PR-template
checklist — are canon in the rook-conventions skill (this plugin). Load
it before committing or opening anything, and follow its routing table on
into `references/commits.md` for message format, amending, and history
rework, `references/pull-requests.md` for PR mechanics, and
`references/building-and-testing.md` for the build tag and the local
verification gate. The user's own global CLAUDE.md outranks it on conflict.

## Tooling

Detector recipes live under `references/` — for dead-code campaigns read
`references/dead-code.md` (deadcode/staticcheck invocations, the
exported-symbol and write-only-field gaps both tools miss,
whole-file-delete rules, sandbox and build-tag caveats). For any other
campaign, substitute your detector (a `staticcheck` check, a
`grep`/`ast-grep` pattern, a `golangci-lint` linter) and the mechanical
transform, keeping everything else in the loop identical.

## Fan-out patterns

- **Scan**: one `Explore` agent per directory, dispatched per rook-conventions
  `references/fan-out.md`, on a small model (principle 7). Each runs the
  detector for its dir and returns a structured candidate list. Give every
  agent the same definition of "dead/violating" and the build-tag and sandbox
  caveats, so results are comparable.
- **Implement**: one `rook-maintainer:code-worker` agent per independent
  file/area with `isolation: worktree` to avoid working-tree collisions,
  dispatched per rook-conventions `references/fan-out.md`, on a small model
  (principle 7). Branch, commit, push and `gh pr create` stay under your own
  control (consistent conventions, DCO, force-push safety) — Phase 4 owns the
  split, and `agents/code-worker.md` independently forbids the worker to push
  or open a PR.
- **Cross-check**: pair the fan-out with one authoritative whole-module pass and
  reconcile — discrepancies are usually a tool blind spot (e.g. a method on an
  instantiated type) worth investigating, not noise to ignore.

## Gotchas learned the hard way

- Always re-sync and re-run exclusion each session; merged PRs change the pool.
- Repushing a campaign branch behind an open PR is rook-conventions
  `references/pull-requests.md` "Updating an open PR", lease form included.
- Never edit the user's global `CLAUDE.md` (or other personal config)
  unilaterally — present the proposed wording and wait for approval.
- Writing under the user's Claude config dir (`~/.claude` by default) needs
  the sandbox disabled.
- Untracked char-device dotfiles (`.bashrc` etc. as `crw-rw-rw-`) in the repo are
  sandbox artifacts — never `git add` or clean them. `autostash` stash entries are
  the user's, not yours — never pop/drop them.

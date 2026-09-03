# GitHub Actions workflow review

Triggers: `.github/workflows/**`, `.mergify.yml`, scripts invoked by
workflows. Security angle (secrets, pull_request_target, injection) lives in
`security.md`; this file owns correctness and convention.

## The oracle rule is inverted here

rook CI does NOT lint its own workflows. Run the linters as part of the
review and triage their output (locally against the changed files — this is
read-only and allowed even for others' PRs, on a scratch copy):

```sh
actionlint <changed workflow files>      # with shellcheck installed, it
                                         # also lints inline run: scripts
GITHUB_TOKEN=$(gh auth token) pinact run --check --verify <changed workflow files>
```

Report what they find (neither is CI-covered): actionlint's diagnostics, and
pinact's diff — a tag or branch ref (`@v4`, `@main`) comes back as a `-`/`+`
rewrite and is a finding; a `# vX.Y.Z` comment disagreeing with its SHA comes
back as a mismatch error and is a lesser one. Then the judgment checks below
that neither tool makes.

## Pinning

Check the diff does not drop `dependabot.yml`'s `github-actions` entry — the
bump loop the `# vX.Y.Z` comments exist for is rook-conventions
`references/workflows-and-ci.md`.

## Logic review (what actionlint misses)

- **Trigger/context mismatch**: `github.event` payload fields differ by
  trigger (`pull_request` vs `push` vs `workflow_dispatch` vs `schedule`) —
  a step reading `github.event.pull_request.*` in a workflow also triggered
  by push gets empty strings, which then flow into conditions silently.
- **`if:` semantics**: expressions vs strings (`if: ${{ false }}` vs
  `if: "false"`), missing `always()`/`failure()` where cleanup must run,
  `cancelled()` handling; job-level vs step-level condition placement.
- **Output plumbing**: `$GITHUB_OUTPUT` written names match
  `needs.<job>.outputs.<name>` / `steps.<id>.outputs.<name>` readers; a
  renamed output with a stale reader fails silently to empty-string.
- **Concurrency groups**: new long-running workflows set `concurrency` so
  superseded runs cancel; the group key actually varies by ref (a constant
  key serializes ALL runs).
- **`permissions:`**: least privilege; a new job that writes (comments,
  checks, packages) declares exactly that; workflows without a permissions
  block inherit repo defaults — flag when the job clearly needs less.
- **Cache correctness**: `actions/cache` keys include the lockfile/toolchain
  hash they guard; restore-keys don't resurrect poisoned/stale caches across
  major toolchain bumps.
- **Artifacts**: upload/download names paired; retention deliberate.

## Matrix logic

- Every matrix dimension is actually CONSUMED by the job (an unused
  dimension multiplies CI cost for nothing).
- `include`/`exclude` produce the intended combinations — enumerate them
  when non-trivial; `fail-fast` and `max-parallel` are deliberate choices
  (burn-in style matrices want `fail-fast: false`).
- **Job names are status checks.** Which changes rename a check, and the
  `.mergify.yml` reconciliation they owe, is rook-conventions
  `references/workflows-and-ci.md`, "Workflow changes and `.mergify.yml`".
  In review an unreconciled pin is a finding. When clean, say "checked
  `.mergify.yml`, no update needed" — absence of a mergify entry is itself
  a check result (label-gated/nightly suites are typically absent).

## Rook conventions

- **Kubernetes versions in matrices are deliberate coverage.** Never accept
  a version downgrade to dodge a tooling limitation (minikube/kind image
  gaps) — fix the tooling (`--force`, bump kind/kindest). Before a version
  bump, confirm the `kindest/node:vX.Y.Z` image actually exists (Docker Hub
  404s happen; v1.36.2 was never published).
- **`GOFLAGS=-tags=ceph_preview`** (or equivalent TAGS plumbing) must
  survive workflow edits — rook-conventions
  `references/building-and-testing.md` "The build tag" is why removing it on
  a green run is not safe.
- `go-version` pinned to the `go.mod` `go` directive — unless the job
  deliberately tests a version matrix, which must INCLUDE the go.mod
  version.
- Label-gated and nightly-only suites (tmate `debug-ci`, `TestCephMgrSuite`)
  exist — a workflow change that silently promotes one into the PR-gating
  set (or demotes a gate) is a finding.
- Burn-in matrices (dummy `instance: [1..N]` dimensions) are a sanctioned
  TEMPORARY flake-measurement pattern — flag one appearing in a PR that is
  headed for merge; they must be dropped before merge.

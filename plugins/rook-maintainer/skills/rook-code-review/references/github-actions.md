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
```

Report what actionlint finds (it is not CI-covered), plus the judgment
checks below that actionlint cannot make.

## Pinning

- Every third-party `uses:` pinned to a FULL commit SHA with a `# vX.Y.Z`
  comment (pinact convention — the comment keeps Dependabot able to bump).
  Tag pins (`@v4`) and branch pins (`@main`) are findings; a SHA pin whose
  version comment is missing or wrong is a lesser finding (breaks the
  Dependabot loop).
- `dependabot.yml` keeps a `github-actions` entry so pins get bumped.

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
- **Job names are status checks.** Matrix-derived names
  (`job (v1.33.0)`) are pinned by branch protection and `.mergify.yml`
  `check-success=` conditions. ANY rename/add/remove of a job or matrix
  value must be reconciled against `.mergify.yml` — a stale pin wedges
  mergify (waits forever) or lets backports merge unchecked. When clean, say
  "checked `.mergify.yml`, no update needed" — absence of a mergify entry is
  itself a check result (label-gated/nightly suites are typically absent).

## Rook conventions

- **Kubernetes versions in matrices are deliberate coverage.** Never accept
  a version downgrade to dodge a tooling limitation (minikube/kind image
  gaps) — fix the tooling (`--force`, bump kind/kindest). Before a version
  bump, confirm the `kindest/node:vX.Y.Z` image actually exists (Docker Hub
  404s happen; v1.36.2 was never published).
- **`GOFLAGS=-tags=ceph_preview`** (or equivalent TAGS plumbing) must
  survive workflow edits — a green build without the tag is not proof it is
  unneeded (see ceph-object.md).
- `go-version` pinned to the `go.mod` `go` directive — unless the job
  deliberately tests a version matrix, which must INCLUDE the go.mod
  version.
- Label-gated and nightly-only suites (tmate `debug-ci`, `TestCephMgrSuite`)
  exist — a workflow change that silently promotes one into the PR-gating
  set (or demotes a gate) is a finding.
- Burn-in matrices (dummy `instance: [1..N]` dimensions) are a sanctioned
  TEMPORARY flake-measurement pattern — flag one appearing in a PR that is
  headed for merge; they must be dropped before merge.

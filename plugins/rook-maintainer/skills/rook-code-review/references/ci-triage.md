# CI triage — classifying red checks on a PR

Purpose: stop every reviewer (human or agent) from re-deriving, PR by PR,
whether a red check means anything. Classify each failing check, cite the
class, and move on. A red check is a finding against the PR ONLY when
classified REAL.

## Procedure

1. `gh pr view <n> --json statusCheckRollup` (sandbox disabled) — collect
   name, conclusion, url per check. Fetch logs ONLY for checks you must
   classify (`gh run view <id> --log-failed`); never `gh run watch`.
2. Classify each failure:
   - **REAL** — the PR's diff plausibly causes it: the failing test/lint
     touches changed code, the failure signature matches the change, or the
     failure is deterministic across reruns.
   - **KNOWN-FLAKE** — matches a documented class below AND is unrelated to
     the changed files.
   - **INFRA** — network/registry/runner failures: image pull timeouts,
     ghcr/quay 5xx, apt mirrors, runner eviction, rate limits.
3. In the report, list every red check with its class and one line of
   evidence. REAL failures feed findings; KNOWN-FLAKE/INFRA are noted, never
   "fixed", and never justify REQUEST CHANGES by themselves.
4. Green-but-suspicious: `make test`/lint green does NOT cover
   `tests/framework/` unit tests (rook-conventions
   `references/building-and-testing.md`) or workflow lint (actionlint not
   run) — do not cite green CI as evidence for those areas.

## Known flake classes

The registry is `references/known-flakes.md` (shared canon, updated via PR
to the plugin repo) — match failure signatures there before classifying.
A failure matching a RESOLVED entry is evidence of a REAL regression, not
a flake. When a failure matches no entry and has no diff connection:
classify KNOWN-FLAKE-CANDIDATE, say so explicitly, and recommend a rerun
rather than a code change; if the same signature recurs across unrelated
PRs, PR it into the registry with date and evidence.

## Rate-limit discipline

All `gh` calls with `dangerouslyDisableSandbox: true`. Batch
queries (`--json` with multiple fields, one call per PR not per field). On
403: `gh api rate_limit` (free) and wait out the reset; never hammer.

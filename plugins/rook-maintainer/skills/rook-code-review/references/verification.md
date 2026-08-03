# Verification — every finding earns its report

Findings agents propose; verification disposes. The point is precision: a
review that cries wolf gets ignored, and a posted comment that is wrong costs
maintainer trust. Nothing reaches the report without passing this file.

## The refutation pass

For each candidate finding, attempt to REFUTE it — do not attempt to confirm
it. Re-read the code assuming the author was right and you misread. In
fan-out modes (sweep, large diffs) this is an independent agent per finding
(or per small group of related findings) that receives the finding and the
code, not the reviewer's reasoning. Inline (small diffs), it is a distinct
second pass over each candidate; state in the report that verification was
inline.

The verifier answers:

1. Does the defect actually exist on the claimed path? (Trace the real code,
   not the diff hunk. Check callers/callees the finding depends on.)
2. Is it NEW in this change, or pre-existing?
3. Does anything else already handle it (a guard upstream, a predicate that
   guarantees the invariant, a linter/CI gate)?
4. Does the concrete failure scenario hold together end to end?

## Confidence rubric (0–100)

- **0–24** — false positive; misreading; pre-existing and untouched;
  speculative ("could", "might" with no concrete path).
- **25–49** — plausible but unverified inference; the failure path was not
  traced to completion.
- **50–74** — real but minor, rare, or partially mitigated; OR real with one
  unverified link in the chain (name the link).
- **75–99** — verified real, matters, evidence traced; small residual doubt.
- **100** — certain; the evidence is in hand (e.g. the nil path is
  reproducible by reading three lines).

Report **>= 80 as CONFIRMED**. Report **50–79 as PLAUSIBLE** only when the
severity would be blocker or changes-requested — and say exactly which link
is unverified. Below 50: drop silently. Prefer one strong finding over three
weak ones; do not dilute a serious report with filler.

## False-positive exclusions

Do not report, regardless of confidence:

- **Pre-existing issues** on lines the change did not modify — unless the
  change makes them worse or copies them into new code. If serious, mention
  once in the summary body, not as a finding.
- **Anything the repo's linters/CI already catch.** rook CI runs errcheck,
  govet, gosec, ineffassign, staticcheck, unused, gofmt+gofumpt
  (`.golangci.yaml`), `make test`, codegen/crds-gen cleanliness checks, DCO,
  and commitlint. Exception: workflows — see `github-actions.md`; rook CI
  does NOT run actionlint.
- **Deliberate suppressions.** `.golangci.yaml` disables specific checks
  (e.g. the aws-sdk-go-v1 deprecation text is suppressed because v1 is
  banned outright; some staticcheck QF rules are off). Respect them.
- **Pedantic nits a senior maintainer would not raise** — micro-style,
  hypothetical performance, "consider extracting a helper" with no concrete
  benefit.
- **Intentional behavior changes** that the PR body/commits document as the
  point of the change.
- **Style contrary to incumbent style**: if the surrounding package does it
  this way consistently, the diff matching the package is correct even when
  another guide disagrees.
- **Speculative DoS / missing-hardening** observations, and lowered bars for
  test-only code (a test may take shortcuts production code may not).

"Missing tests" IS reportable — rook expects unit tests with code changes —
but calibrate: a bugfix with no regression test is changes-requested; a
mechanical refactor is not.

## Rook precedents (judgment calls, settled)

- `undefined: admin.Account` (or similar go-ceph symbols) from an ad-hoc
  `go build/vet/test` is the missing `ceph_preview` build tag, not broken
  code. The tag is global (`Makefile` `TAGS`, `.golangci.yaml` build-tags).
- **Never flag "missing inline t.Run gates."** House rule: test bodies do not
  use `if !t.Run(...) { t.FailNow() }`; cascade noise from dependent siblings
  is accepted. DO flag the inverse — an inline run-result gate appearing in a
  test body (only `require*`-prefixed check helpers may encapsulate it).
- assert-vs-require deviations where both behave identically (sole/last check
  in its own subtest) are style nits, never blockers.
- `_test.go` under `tests/framework/` compiles in lint but never runs in CI —
  do not credit it as test coverage.
- Backported features (`backport-release-X.Y` label) do not get
  `PendingReleaseNotes.md` entries; pure bugfixes/refactors/tests usually
  don't either.
- Editing a godoc comment under `pkg/apis` requires regenerating CRDs — the
  comments are emitted verbatim into CRD `description` fields.
- Generated files are never hand-edited: `zz_generated.deepcopy.go`,
  `deploy/examples/crds.yaml`, `deploy/charts/rook-ceph/templates/resources.yaml`,
  `Documentation/CRDs/specification.md`, the helm-docs chart pages.
- Consuming the return value of a `wait4.Assert*` helper is a real bug
  (assert-flavored helpers return zero values on failure); discarding it is
  fine.
- A red CI check is not a finding against the PR unless `ci-triage.md`
  classifies it REAL.

## Evidence contract

A verified finding states: `file:line`, what is wrong (one sentence), the
concrete failure scenario (inputs/state → wrong outcome), the fix shape (one
line), confidence with label. If any conclusion rests on an inference rather
than traced evidence, say so explicitly and keep the confidence honest.
Finding IDs are assigned later, at report assembly (SKILL.md "Finding
IDs"); verification refers to candidates by `file:line`.

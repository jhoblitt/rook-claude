# Verification — every finding earns its report

Findings agents propose; verification disposes. The point is precision: a
review that cries wolf gets ignored, and a posted comment that is wrong costs
maintainer trust. Nothing reaches the report without passing this file.

## Order of operations

The mechanical pre-filter below runs FIRST, over the candidate list, before
verification reads any code: before a verifier agent is dispatched in a
fan-out review, and before the second pass begins inline. Its tests read the
candidate's anchor, the candidate's own text, and the diff — never the code —
so nothing it drops may cost an agent whose verdict the drop would then
discard. The refutation pass runs on what survives, and the judgment
exclusions apply to that pass's verdicts, because each of them needs
something read — the code, the surrounding package, `.golangci.yaml`, the PR
body — that only the pass pays for. Which list an exclusion belongs to is
decided by what it must READ, never by how strong it is: both stages exclude
regardless of confidence.

## Mechanical pre-filter — before verification reads any code

Drop from the candidate list. No code read, no agent spent:

- **Anchors outside the diff.** A candidate whose `file:line` is not a line
  this change modified is pre-existing; if it is serious, mention it once in
  the summary body rather than as a finding. It reaches the refutation pass
  only when the candidate ITSELF claims the change makes the issue worse or
  copies it into new code — that claim is in the candidate's text, so routing
  on it stays mechanical, and the pass then verifies the claim like any other.
- **Generated-path anchors.** A candidate anchored inside a generated file —
  `docs-sync.md` "The generated set" — points at output nobody edits;
  the fix, if there is one, is in the source plus a regen. A candidate whose
  claim IS the generation survives: a hand-edited hunk in such a file, or a
  regenerated artifact missing from the changeset, is a `docs-sync` finding at
  the severity that file sets. This test carries the volume, which is why it
  runs before dispatch and not after — one added CRD kind regenerates a whole
  `pkg/client/**` tree, and `reuse.md` drops the same class before it queries
  for the same reason.
- **Anything the repo's linters/CI already catch**, matched by the
  candidate's CLASS against the roster. rook CI runs errcheck, govet, gosec,
  ineffassign, staticcheck, unused, gofmt+gofumpt (`.golangci.yaml`),
  `make test`, codegen/crds-gen cleanliness checks, DCO, and commitlint.
  Exception: workflows — see `github-actions.md`; rook CI does NOT run
  actionlint.

## The refutation pass

For each candidate the pre-filter left, attempt to REFUTE it — do not attempt
to confirm it. Re-read the code assuming the author was right and you
misread. In fan-out reviews (large diffs, pre-pr panels) this is an
independent agent per finding (or per small group of related findings) that
receives the finding and the code, not the reviewer's reasoning. Inline
(small diffs), it is a distinct second pass over each candidate; state in the
report that verification was inline.

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

## Design findings verify differently

Candidates in the `design` domain (architecture.md's contract) have no
failure path to trace: refute them with architecture.md's rubric —
deliberate precedent, alternative-fails, cost-contradicted, generic taste —
instead of questions 1–4 above. How the bands below map onto that rubric,
what QUESTIONs are exempt from, and the load-bearing enforcement claim that
never reshapes and never drops are all architecture.md's "Verification
rubric (design findings)" — none of it is restated here.

## Judgment exclusions — applied to the refutation pass's verdicts

Do not report, regardless of confidence. Each needs a read the pre-filter has
not done, which is why these apply to verdicts and never to candidates:

- **Deliberate suppressions.** `.golangci.yaml` disables specific checks
  (e.g. the aws-sdk-go-v1 deprecation text is suppressed because v1 is
  banned outright; some staticcheck QF rules are off). Respect them.
- **Pedantic nits a senior maintainer would not raise** — micro-style,
  hypothetical performance, "consider extracting a helper" where no such
  helper exists. A `design` finding meeting architecture.md's contract —
  named cost, named alternative, precedent — is not this class; that
  contract is exactly what separates design judgment from taste. Neither is
  a `duplication` finding naming an EXISTING symbol the diff re-implemented
  (reuse.md): proposing an abstraction is taste, pointing at the helper
  already in the tree is not. Nor is a `cross-ref` finding meeting
  cross-references.md's contract, which names both the referenced item and
  what GitHub does with it on merge.
- **Intentional behavior changes** that the PR body/commits document as the
  point of the change — as correctness findings claiming the behavior is
  accidental. Whether a deliberate, documented decision is the right one
  is exactly the `design` domain's question: cost/alternative findings
  are never excluded by documented intent, only refuted by
  architecture.md's rubric.
- **Style contrary to incumbent style**: if the surrounding package does it
  this way consistently, the diff matching the package is correct even when
  another guide disagrees.
- **Out-of-band admin mutation.** rook owns the Ceph control plane it
  manages; a scenario that needs an administrator mutating that state
  concurrently (toolbox `ceph`/`radosgw-admin` writes racing a reconcile)
  is not a finding — the space is unbounded, it indicts any reconcile
  equally, and the operator cannot lock Ceph against its admins. Still
  reportable: rook's OWN concurrency (parallel reconciles, multiple
  operators, restart mid-flight); races against a procedure rook's docs
  direct the user to run; external-mode clusters, where shared control is
  the deployment model; and data-plane client IO (S3/RBD) racing
  control-plane changes.
- **Speculative DoS / missing-hardening** observations, and lowered bars for
  test-only code (a test may take shortcuts production code may not).

"Missing tests" IS reportable — rook expects unit tests with code changes —
but calibrate: a bugfix with no regression test is changes-requested; a
mechanical refactor is not.

## Rook precedents (judgment calls, settled)

- **An `undefined:` go-ceph symbol is not automatically the build tag.**
  Attributing it there is the reflex to resist: check the symbol is actually
  gated first. Whether the tag currently gates anything rook uses is
  rook-conventions `references/building-and-testing.md` "The build tag", and
  what an ad-hoc run proves either way is `ceph-object.md`.
- **"Missing inline t.Run gates" is settled: never a finding.** The house
  rule and its one carve-out are `testing.md`'s "House rule — no inline
  gates"; what is verification's is the disposition — a candidate proposing
  gates dies here whatever its confidence, while the inverse candidate that
  file directs the reviewer to flag is not excluded.
- assert-vs-require deviations where both behave identically (sole/last check
  in its own subtest) are style nits, never blockers.
- `tests/framework/` unit tests are never coverage — why they do not run is
  rook-conventions `references/building-and-testing.md`, which names the CI
  scope that excludes them. A "tested" claim resting on them fails here.
- Whether a change owes a `PendingReleaseNotes.md` entry is `docs-sync.md`'s
  Direction 1 row, not a per-review judgment — do not re-derive it.
- Editing a godoc comment under `pkg/apis` requires regenerating CRDs — the
  comments are emitted verbatim into CRD `description` fields.
- Which files are generated, and what a hand-edit to one costs, is
  `docs-sync.md`'s — the pre-filter above routes on that enumeration and
  nothing here re-derives it.
- Consuming the return value of a `wait4.Assert*` helper is a real bug
  (assert-flavored helpers return zero values on failure); discarding it is
  fine.
- A red CI check is not a finding against the PR unless `ci-triage.md`
  classifies it REAL.

## Evidence contract

A verified finding satisfies SKILL.md's finding contract — that section
is normative, anchors and all; nothing here restates it. Verification
adds only: if any conclusion rests on an inference rather than traced
evidence, say so explicitly and keep the confidence honest.
Finding IDs are assigned later, at report assembly (SKILL.md "Finding
IDs"); verification refers to candidates by `file:line`.

# Reinvention check — review spine pass j

- **Status**: implemented — shipped on branch `spec/reinvention-check-pass`
  (2026-08-06); the shipped canon is authoritative where it has moved on
- **Date**: 2026-08-05
- **Affects**: `plugins/rook-maintainer/skills/rook-code-review`,
  `plugins/rook-maintainer/agents/rook-reviewer.md`

## Motivation

`rook-code-review` catches AI-authored PRs through a set of defect classes
that each live in the domain reference they belong to: the silent-failure
hunt (`references/go-review.md`), AI-blather comments
(`references/naming-and-comments.md`), fabricated bugs (`SKILL.md` ground
rules), checklist gaming (`references/docs-sync.md`), unowned LLM PR bodies
(`agents/rook-reviewer.md`).

One tell is missing from that set: **models re-implement code they did not
search for.** An LLM writing a rook change does not reliably discover the
existing `k8sutil` helper, the existing composite action, or the existing
named chart template, so it writes a second one. The result passes every
existing pass — it is not a correctness defect, no linter in
`.golangci.yaml` sees it, and it trips no decision-magnitude trigger in
`references/architecture.md` unless it rises to mechanism scale (a second
retry wrapper, a second cache).

The cost is divergence: the next fix lands in one copy and not the other.

`references/architecture.md` already covers the mechanism-scale case via its
third decision trigger. This spec covers the scale below it — a bypassed
reuse mechanism — and extends the check beyond Go.

## Scope

In scope: all content a rook diff can add — Go, `.github/workflows/**` and
`tests/scripts/`, helm chart templates, `Documentation/**`,
`deploy/examples/**`.

Out of scope, deferred to `rook-systemic-prs`: pre-existing duplication the
diff merely sits beside, and cross-package abstraction proposals. This
follows the precedent already set in `references/go-review.md`, where
whole-package modernize sweeps belong to campaigns rather than review
feedback.

## Design

### The reuse-mechanism framing

A finding never asserts that two pieces of code look alike. It asserts that
**the repo has a named reuse mechanism for this job and the diff bypassed
it.** Every in-scope domain has one:

| Domain | Reuse mechanism a diff can bypass |
|---|---|
| Go | Exported helpers — `pkg/util/`, `pkg/operator/k8sutil/`, `pkg/operator/ceph/controller/` shared reconcile helpers |
| Workflows | Composite actions (`action.yml`), reusable workflows (`workflow_call`), `tests/scripts/` shell helpers |
| Helm charts | Named templates (`_helpers.tpl` `define`/`include`) |
| Documentation | A procedure already documented elsewhere, forked and now free to diverge |
| `deploy/examples/**` | A new example file duplicating an existing one wholesale — nothing narrower; the format is repetitive by design |

This framing is what makes the broad scope tractable. It gives every finding
a concrete referent and excludes the "consider extracting an abstraction"
class by construction, since an abstraction that does not yet exist is not a
mechanism the diff bypassed.

### Pass j

Appended as step `2j` of the review spine in `SKILL.md`. Appended, not
inserted: `references/architecture.md` and `agents/rook-reviewer.md` cite
spine passes by letter (`pass h`, `pass i`), so inserting mid-list would
invalidate those references.

Pass text, as shipped:

> j. **Reinvention check**: for each symbol, step, template, or procedure
> the diff ADDS, does the repo already provide it through a named reuse
> mechanism? Generate candidates mechanically, then adjudicate only the
> hits on behavioral equivalence (`references/reuse.md`). Its object is
> the rest of the repo rather than the diff — which is what makes its
> findings independent of pass a.

### Two-stage execution

The pass splits into a cheap generator and an expensive adjudicator. A
negative from the generator ends the work for that symbol, which is what
makes the pass affordable at sweep scale — bounded by the coverage limit
below, which constrains what that negative is allowed to claim.

**Generation** — mechanical, haiku-class per `SKILL.md` "Tier models by
role". Per added symbol or block: one LSP `workspace/symbol` query plus a
name-normalized variant for Go; a grep for the declared step, template, or
procedure name in the other domains. No hit means no further work, which is
the common case.

**Adjudication** — judgment, session model. Fires only on a hit. Read both
implementations in full and establish behavioral equivalence: for Go, same
inputs to same outputs including error paths; for workflows, same steps and
same permissions. Textual similarity is never sufficient.

**Coverage limit — the generator matches names, not behavior.** A helper
re-implemented under an unrelated name produces no hit, and that is the
likeliest shape of the motivating defect: a model that failed to find the
original had no reason to reuse its name. `references/reuse.md` must state
this in the voice `references/go-review.md` uses for the `go fix` oracle —
a clean generation pass means the queries found nothing THEY look for,
never that the diff reinvented nothing. The pass therefore claims
name-reachable reinvention only, and its "audited and clean" line says so
rather than implying a duplication-free diff.

### Execution per mode

| Mode | Generation | Adjudication |
|---|---|---|
| diff, small (< ~300 lines) | Inline | Inline |
| diff, large / pre-pr | Fan out — one agent per changed file | Fan out — one agent per candidate hit |
| sweep | Inside the per-PR reviewer agent, serial | Lifted to the orchestrator, phase 2 |

The sweep lift is load-bearing. `SKILL.md` specifies that each per-PR
reviewer agent runs the spine inline with passes serial and no Agent tool,
and the `rook-reviewer` agent definition grants Bash, Read, Grep, Glob,
WebFetch, and LSP — no Agent. It cannot nest fan-out. Running adjudication
there would serialize a repo-wide search per added symbol inside every
per-PR agent across the whole sweep.

Lifting adjudication rides the seam the sweep already has: per-PR agents
produce candidates, and the orchestrator owns every stage needing a
whole-target view — verification layer two, the caps, ID assignment.

### Finding contract

Ordinary findings under the existing contract in `SKILL.md`. No new severity
and no cap machinery.

- Domain tag: **`duplication`**, added to the tag list in `SKILL.md`.
- Severity: `changes-requested` when the diff re-implements a named existing
  symbol or mechanism; `nit` for intra-diff repetition, or near-duplication
  with a plausible reason.
- The anchor names the existing implementation at its full repo-relative
  path plus symbol, per the finding contract's full-path rule.
- The finding states which reuse mechanism was bypassed.
- The failure scenario is always divergence: a named future change that must
  land in both copies and will land in one.

### Exclusions

Written with the check, not added after it. These live in
`references/reuse.md`:

- **Deliberately-parallel sibling packages** — `pkg/operator/ceph/object/`,
  `file/`, `pool/`. Parallel *structure* is rook's incumbent pattern; only
  duplicated *behavior* is reportable.
- **`deploy/examples/**` and chart values repetition**; test table rows and
  fixtures, where literal repetition aids readability.
- **Generated files** (already enumerated in `references/verification.md`)
  and vendored code.
- **Pre-existing duplication** the diff sits beside — already covered by the
  pre-existing-issues exclusion in `references/verification.md`.
- **Abstractions that do not exist yet.** The "consider extracting a helper"
  exclusion in `references/verification.md` is narrowed, not deleted: it
  continues to kill proposals for new abstractions, and stops covering the
  case where the diff re-implemented a symbol that already exists and can be
  named.

## Changes to existing files

| File | Change |
|---|---|
| `skills/rook-code-review/SKILL.md` | Add spine step `2j`; add routing-table row for `references/reuse.md`; add `duplication` to the domain tag list |
| `skills/rook-code-review/references/verification.md` | Narrow the "consider extracting a helper" false-positive exclusion per above |
| `skills/rook-code-review/references/sweep.md` | Phase 2 adjudicates reuse candidates lifted from per-PR agents |
| `agents/rook-reviewer.md` | Pass j in the instruction list; sweep degradation (generate candidates, do not adjudicate); new `reuse_candidates[]` field in the JSON output contract |
| `evals/README.md` | Two new rows in the case table |

`README.md` needs no change — its skill table is one line per skill.
`references/adversarial.md` needs no change — it already delegates to "the
review spine, steps 1–3".

### New files

- `skills/rook-code-review/references/reuse.md` — the reuse-mechanism table,
  the two-stage procedure, the equivalence bar, and the exclusions.

### Reviewer JSON contract

Candidates ride in a new `reuse_candidates[]` field rather than as
low-confidence findings, because an unadjudicated candidate is not yet a
finding. The contract has precedent for orchestrator-handoff fields:
`needs_proposal_review`, `takeover_candidate`.

```json
"reuse_candidates": [{"added": "path:line or symbol",
  "existing": "full/repo/relative/path.go:Symbol",
  "mechanism": "which reuse mechanism was bypassed",
  "evidence": "how the existing implementation was found"}]
```

## Evals

Paired positive and trap cases, following the `design-recall` /
`design-precision` precedent.

- **`reuse-reinvention`** — a diff re-implementing an existing `k8sutil`
  helper is caught, with the existing symbol named at its full
  repo-relative path.
- **`reuse-parallel-siblings`** — a new controller that structurally mirrors
  an existing sibling controller is **not** flagged. This is the
  anti-pontification guard and the case expected to fail first.

Both must be hermetic or build their own throwaway fixture, per the
constraint stated in `evals/README.md` that expected answers never drift
with rook master.

## Risks

The broad scope is the main risk. Duplication is the highest-false-positive
finding class in code review, and rook contains large volumes of
legitimately repetitive content in `deploy/examples/**` and chart values.
The reuse-mechanism framing and the `reuse-parallel-siblings` trap eval are
the two controls; if the trap eval cannot be made to pass, the correct
response is to narrow the pass's scope rather than to weaken the eval.

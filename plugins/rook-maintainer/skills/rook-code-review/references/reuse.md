# Reinvention check — the reuse mechanism the diff bypassed

Spine pass j (SKILL.md). Its object is the REST OF THE REPO rather than the
diff: does what the diff adds already exist there? Findings are ordinary
findings under SKILL.md's contract, tagged `duplication`.

The claim is never "these look alike." It is always: the repo provides a
NAMED reuse mechanism for this job, and the diff bypassed it. A candidate
that cannot name both the mechanism and the existing implementation is not
a finding.

## Reuse mechanisms by domain

| Diff adds | Mechanism it may bypass |
|---|---|
| Go function or method | An exported helper — `pkg/util/`, `pkg/operator/k8sutil/`, `pkg/operator/ceph/controller/` shared reconcile helpers |
| Workflow job or step block | A composite action (`action.yml`), a reusable workflow (`workflow_call`), a `tests/scripts/` shell helper |
| Helm template block | A named template (`_helpers.tpl` `define`/`include`) |
| Documentation procedure | The same procedure already documented elsewhere, now forked and free to diverge |
| `deploy/examples/**` file | An existing example file it duplicates WHOLESALE — nothing narrower; that format repeats by design |

## Stage 1 — generate (mechanical, haiku-class when it fans out as its own agent)

Filter the input by path and count first — both tests are mechanical, which
is what keeps this stage haiku-class: drop every addition under a generated
or vendored path (the exclusions below name them; one added CRD kind
regenerates a whole `pkg/client/**` tree whose accessors collide with every
other kind's), and stop after querying 20 additions per target, saying so
when the cap binds.

Per symbol, step, template, or procedure the diff ADDS — never per line it
edits:

- Go: LSP `workspace/symbol` on the added name, then on its normalized form
  (case-folded and de-prefixed, so `addLabelToPod` reaches
  `AddLabelToPod`).
- Other domains: grep the declared step, template, or procedure name across
  that domain's tree.

No hit ends the work for that symbol. Emit hits as candidates and judge
nothing here.

## Stage 2 — adjudicate (judgment, session model)

Only on a hit. Read BOTH implementations in full and establish BEHAVIORAL
equivalence:

- Go: same inputs to same outputs, error paths included. A helper differing
  on nil handling, ordering, or error wrapping is not a duplicate — it is a
  second behavior, and the finding, if any, is that the difference is
  undocumented.
- Workflows: same steps, same `permissions`, same triggers.

Textual similarity is never sufficient, at either stage.

## What the pass may claim

Generation matches NAMES, not behavior. A helper re-implemented under an
unrelated name produces no hit — and that is the likeliest shape of the
defect this pass exists for, since a model that never found the original had
no reason to reuse its name. The limit go-review.md places on the `go fix`
oracle holds here: a clean generation pass means the queries found nothing
THEY look for, never that the diff reinvented nothing.

Say so in every report the pass runs in, findings or none: the
audited-and-clean line scopes the claim — "no name-reachable reinvention
beyond what is reported" — and never implies a duplication-free diff. The
limit belongs to how generation queries, not to what it turned up.

## Finding shape

SKILL.md's contract governs; tag `duplication`.

- **changes-requested** — the diff re-implements a named existing symbol, or
  bypasses a named mechanism.
- **nit** — intra-diff repetition, or near-duplication with a plausible
  reason.

Anchor the ADDED code; name the existing implementation in the body at its
full repo-relative path plus symbol. The failure scenario is always
divergence: a concrete future change that must land in both copies and will
land in one.

## Exclusions

Do not report, regardless of confidence:

- **Deliberately-parallel siblings.** `pkg/operator/ceph/object/`, `file/`,
  `pool/`, and the other per-resource controllers mirror each other by
  design. Parallel STRUCTURE — the same reconcile skeleton, the same
  error-wrapping shape — is rook's incumbent pattern and is never a finding.
  Only duplicated BEHAVIOR is reportable.
- **Repetition that is the format**: `deploy/examples/**`, chart values,
  test table rows, fixtures. Literal repetition there aids readability.
- **Generated files** (docs-sync.md "The generated set") and vendored code.
- **Pre-existing duplication** the diff merely sits beside —
  verification.md's pre-existing exclusion governs.
- **Abstractions that do not exist yet.** "Extract a shared helper" where no
  helper exists is the taste class verification.md excludes, and this pass
  never revives it. The mechanism must already exist and be nameable.

# Reinvention Check (Spine Pass j) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a reinvention check to the `rook-code-review` review spine as
pass j, so a diff that re-implements something the repo already provides is
reported instead of passing silently.

**Architecture:** A new reference (`references/reuse.md`) carries the canon;
`SKILL.md` wires it as spine pass `j` plus a routing row and a `duplication`
domain tag; `verification.md`'s helper-extraction exclusion is narrowed so it
no longer contradicts the new pass; and the sweep path splits the work —
per-PR reviewer agents generate candidates, the orchestrator adjudicates them.

**Tech Stack:** Markdown prompt prose only. No source code changes. Gates are
`markdownlint-cli2` 0.18.1, `claude plugin validate`, and the
`skill-review` skill.

## Global Constraints

- **Findings never assert similarity.** Every `duplication` finding names the
  reuse mechanism bypassed AND the existing implementation, at its full
  repo-relative path plus symbol. Copy this rule verbatim into any new prose.
- **Spine passes append, never insert.** `references/architecture.md` and
  `agents/rook-reviewer.md` cite passes by letter (`pass h`, `pass i`).
  The new pass is `j`, added after `i`. Do not renumber.
- **`SKILL.md` is normative for the finding contract.** New prose points at
  it and never restates it — the repo's `skill-review` gate flags restated
  canon as drift.
- **Commit types map to releases.** `.releaserc.yml` makes `docs`,
  `refactor`, and `perf` cut a patch, because plugin behavior lives in
  markdown. `test`, `chore`, `ci`, `build`, and `style` release nothing.
  Use the type each task specifies.
- **No AI-attribution trailers policy differs per repo.** This repo
  (`jhoblitt/rook-claude`) uses `Co-Authored-By`; keep it. That flips if the
  repo is donated to `rook/`.
- **Never bump the version.** `semantic-release` owns
  `plugins/rook-maintainer/.claude-plugin/plugin.json`.
- **`claude plugin eval` is currently a no-op** on stock installs
  (`evals/README.md`). Eval cases are authored to the documented layout and
  cannot be executed yet — do not claim an eval passed.
- **Hard tabs in fixtures are already allowed.** `.markdownlint-cli2.jsonc`
  was updated alongside this plan with `"MD010": { "code_blocks": false }`,
  because gofmt-formatted Go fixtures carry tabs and the existing LSP eval
  avoided the rule only by keeping its function bodies on one line. Write
  the fixtures gofmt-correct; no further config change is needed.

---

### Task 1: Eval cases (the behavioral spec)

The evals come first because they force the exclusions to be precise before
the canon is written. The trap case is the one expected to fail first.

**Files:**
- Create: `plugins/rook-maintainer/evals/reuse-reinvention/prompt.md`
- Create: `plugins/rook-maintainer/evals/reuse-reinvention/graders/criteria.md`
- Create: `plugins/rook-maintainer/evals/reuse-parallel-siblings/prompt.md`
- Create: `plugins/rook-maintainer/evals/reuse-parallel-siblings/graders/criteria.md`
- Modify: `plugins/rook-maintainer/evals/README.md` (two rows in the case table)

**Interfaces:**
- Consumes: nothing.
- Produces: the behavioral contract Task 2's `references/reuse.md` must
  satisfy — a name-reachable reinvention is reported as
  `changes-requested`/`duplication`; parallel sibling structure is not
  reported; the audited-and-clean line scopes its claim to name-reachable
  reinvention.

- [x] **Step 1: Create the positive case prompt**

Write `plugins/rook-maintainer/evals/reuse-reinvention/prompt.md`:

````markdown
Create a scratch directory and write exactly this two-file Go module into
it, preserving the directory structure.

`go.mod`:

```text
module github.com/rook/rook

go 1.26
```

`pkg/operator/k8sutil/labels.go`:

```text
package k8sutil

type PodSpec struct {
	Labels map[string]string
}

// AddLabelToPod sets a label on the pod, creating the map when absent.
func AddLabelToPod(pod *PodSpec, key, value string) {
	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[key] = value
}
```

Then review the following diff against that module per the rook-code-review
skill's review spine. There is no network and no `gh`, and subagents cannot
be spawned in this environment: run the passes inline.

Your entire final answer is the review report — verdict line, findings in
the skill's finding contract, and the audited-and-clean statement. Nothing
else.

```diff
--- /dev/null
+++ b/pkg/operator/ceph/object/labels.go
@@ -0,0 +1,10 @@
+package object
+
+import "github.com/rook/rook/pkg/operator/k8sutil"
+
+// addLabelToPod sets a label on the pod, creating the map if it is nil.
+func addLabelToPod(pod *k8sutil.PodSpec, key, value string) {
+	if pod.Labels == nil {
+		pod.Labels = map[string]string{}
+	}
+	pod.Labels[key] = value
+}
```
````

- [x] **Step 2: Create the positive case criteria**

Write `plugins/rook-maintainer/evals/reuse-reinvention/graders/criteria.md`:

```markdown
The added `object.addLabelToPod` is a line-for-line re-implementation of
`k8sutil.AddLabelToPod`, which the diff's own file already imports. The
names differ only by exported-case, so the pass's name-normalized query
reaches it — this case tests what pass j claims to cover.

Pass if and only if ALL of:

1. A finding tagged `duplication` reports the added function as a
   re-implementation, at changes-requested severity.
2. The finding names the existing implementation as
   `pkg/operator/k8sutil/labels.go:AddLabelToPod` — full repo-relative
   path plus symbol.
3. The finding names the bypassed mechanism (the exported helper in
   `pkg/operator/k8sutil/`) rather than only asserting the two are alike.
4. The failure scenario is divergence — a future change that must land in
   both copies and will land in one.
5. The audited-and-clean statement scopes its reuse claim to
   name-reachable reinvention, rather than asserting the diff duplicates
   nothing.

Fail if any of:

- No `duplication` finding is reported, or it is reported as a nit.
- The existing implementation is anchored by bare basename (`labels.go`)
  or an elided path.
- The finding argues only textual similarity, naming no mechanism.
- The report claims the diff is free of duplication without qualifying the
  claim to what the queries can reach.
- Subagents were spawned despite the stated no-subagent environment.
```

- [x] **Step 3: Create the trap case prompt**

Write `plugins/rook-maintainer/evals/reuse-parallel-siblings/prompt.md`:

````markdown
Create a scratch directory and write exactly this two-file Go module into
it, preserving the directory structure.

`go.mod`:

```text
module github.com/rook/rook

go 1.26
```

`pkg/operator/ceph/pool/controller.go`:

```text
package pool

import "fmt"

type Client interface {
	GetPool(name string) (string, error)
	ApplyPool(spec string) error
}

type PoolReconciler struct{ client Client }

func (r *PoolReconciler) Reconcile(name string) error {
	spec, err := r.client.GetPool(name)
	if err != nil {
		return fmt.Errorf("failed to get pool %q: %w", name, err)
	}
	if err := r.client.ApplyPool(spec); err != nil {
		return fmt.Errorf("failed to apply pool %q: %w", name, err)
	}
	return nil
}
```

The fixture deliberately has no third-party imports: the module must resolve
with no network. It uses `fmt.Errorf`/`%w` consistently across both the
existing file and the added one, so go-review.md's match-the-package error
idiom rule produces no finding either way.

Then review the following diff against that module per the rook-code-review
skill's review spine. There is no network and no `gh`, and subagents cannot
be spawned in this environment: run the passes inline.

Your entire final answer is the review report — verdict line, findings in
the skill's finding contract, and the audited-and-clean statement. Nothing
else.

```diff
--- /dev/null
+++ b/pkg/operator/ceph/filesystem/controller.go
@@ -0,0 +1,22 @@
+package filesystem
+
+import "fmt"
+
+type Client interface {
+	GetFilesystem(name string) (string, error)
+	ApplyFilesystem(spec string) error
+}
+
+type FilesystemReconciler struct{ client Client }
+
+func (r *FilesystemReconciler) Reconcile(name string) error {
+	spec, err := r.client.GetFilesystem(name)
+	if err != nil {
+		return fmt.Errorf("failed to get filesystem %q: %w", name, err)
+	}
+	if err := r.client.ApplyFilesystem(spec); err != nil {
+		return fmt.Errorf("failed to apply filesystem %q: %w", name, err)
+	}
+	return nil
+}
```
````

- [x] **Step 4: Create the trap case criteria**

Write `plugins/rook-maintainer/evals/reuse-parallel-siblings/graders/criteria.md`:

```markdown
The added filesystem reconciler mirrors the existing pool reconciler's
STRUCTURE — same reconcile skeleton, same `fmt.Errorf`/`%w` shape — while
operating on a different resource through different client calls. Rook's
per-resource controllers are parallel by design. This case guards
precision: pass j must not convert incumbent structure into a finding.

Pass if and only if ALL of:

1. No `duplication` finding is reported against the added reconciler.
2. No finding proposes extracting a shared or generic reconciler, a
   generics-based abstraction, or a common base type.
3. The report reaches a verdict and includes an audited-and-clean
   statement naming the reuse check among what was run and held.

Fail if any of:

- Any `duplication` finding is raised against the added file.
- Any finding recommends factoring the two reconcilers together, at any
  severity including nit.
- The review offers generic software-taste advice (extract a strategy,
  add an abstraction layer, DRY these up).
- Subagents were spawned despite the stated no-subagent environment.
```

- [x] **Step 5: Add both rows to the eval README table**

In `plugins/rook-maintainer/evals/README.md`, append these two rows to the
end of the case table (after the `full-path-anchors` row):

```markdown
| `reuse-reinvention` | Spine pass j reports a name-reachable re-implementation of an existing `k8sutil` helper as a `duplication` finding naming the bypassed mechanism, and scopes its clean claim to what name queries can reach. |
| `reuse-parallel-siblings` | A new per-resource controller mirroring a sibling's structure is NOT flagged as duplication — the anti-pontification guard for pass j. |
```

- [x] **Step 6: Correct the prerequisites sentence**

The README currently scopes the Go-toolchain requirement to the LSP cases.
Both new cases also build Go fixtures, so replace this sentence fragment:

```markdown
Prerequisites: a Go toolchain and a configured Go language server (e.g.
the `gopls-lsp` plugin) — the LSP cases build their own throwaway Go
fixture module, so no rook checkout is needed and expected answers never
drift with rook master.
```

with:

```markdown
Prerequisites: a Go toolchain and a configured Go language server (e.g.
the `gopls-lsp` plugin) — the LSP and reuse cases build their own
throwaway Go fixture modules, so no rook checkout is needed and expected
answers never drift with rook master. Those fixtures pull no third-party
modules, so they resolve without network access.
```

- [x] **Step 7: Verify lint and plugin validity**

Run:

```bash
cd /home/jhoblitt/github/rook-claude
npx --yes markdownlint-cli2@0.18.1
claude plugin validate .
claude plugin validate plugins/rook-maintainer
```

Expected: markdownlint reports no errors under `plugins/` or `docs/`
(pre-existing `.remember/*.md` MD047 errors are gitignored noise and are
expected); both validate commands succeed.

Do NOT run `claude plugin eval` and report a result — it is a no-op on
stock installs.

- [x] **Step 8: Commit**

```bash
git add plugins/rook-maintainer/evals/
git commit -m "test: pin pass j behavior with reinvention and sibling-trap evals

The positive case fixes what a duplication finding must carry: the
bypassed mechanism and the existing symbol at a full repo-relative path.
The trap case fixes what pass j must stay silent about — rook's
per-resource controllers are parallel by design, and structural mirroring
is the incumbent pattern rather than a defect.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: The canon — `reuse.md`, the exclusion narrowing, and the spine wiring

Lands the pass for `diff` and `pre-pr` modes. These three files must land
together: `reuse.md` alone is canon nothing routes to, and shipping it
without narrowing `verification.md` leaves the skill self-contradicting.

**Files:**
- Create: `plugins/rook-maintainer/skills/rook-code-review/references/reuse.md`
- Modify: `plugins/rook-maintainer/skills/rook-code-review/references/verification.md` (the "Pedantic nits" exclusion bullet)
- Modify: `plugins/rook-maintainer/skills/rook-code-review/SKILL.md` (spine step `2j`; routing row; domain tag list)

**Interfaces:**
- Consumes: the behavioral contract from Task 1's criteria files.
- Produces: `references/reuse.md` with the two-stage procedure that Task 3's
  sweep split references by stage name — **generate** (mechanical, per added
  symbol) and **adjudicate** (judgment, on hits only).

- [x] **Step 1: Write `references/reuse.md`**

Create `plugins/rook-maintainer/skills/rook-code-review/references/reuse.md`:

```markdown
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
- **Generated files** (verification.md enumerates them) and vendored code.
- **Pre-existing duplication** the diff merely sits beside —
  verification.md's pre-existing exclusion governs.
- **Abstractions that do not exist yet.** "Extract a shared helper" where no
  helper exists is the taste class verification.md excludes, and this pass
  never revives it. The mechanism must already exist and be nameable.
```

- [x] **Step 2: Narrow the helper-extraction exclusion**

In `plugins/rook-maintainer/skills/rook-code-review/references/verification.md`,
replace this bullet:

```markdown
- **Pedantic nits a senior maintainer would not raise** — micro-style,
  hypothetical performance, "consider extracting a helper" with no concrete
  benefit. A `design` finding meeting architecture.md's contract — named
  cost, named alternative, precedent — is not this class; that contract is
  exactly what separates design judgment from taste.
```

with:

```markdown
- **Pedantic nits a senior maintainer would not raise** — micro-style,
  hypothetical performance, "consider extracting a helper" where no such
  helper exists. A `design` finding meeting architecture.md's contract —
  named cost, named alternative, precedent — is not this class; that
  contract is exactly what separates design judgment from taste. Neither is
  a `duplication` finding naming an EXISTING symbol the diff re-implemented
  (reuse.md): proposing an abstraction is taste, pointing at the helper
  already in the tree is not.
```

- [x] **Step 3: Add spine pass j**

In `plugins/rook-maintainer/skills/rook-code-review/SKILL.md`, immediately
after evidence pass `i` (the block ending `carry cost and alternative
instead of a failure scenario.`) and before `3. **Verify.**`, insert:

```markdown
   - j. **Reinvention check**: for each symbol, step, template, or
     procedure the diff ADDS, does the repo already provide it through a
     named reuse mechanism? Generate candidates mechanically, then
     adjudicate only the hits on behavioral equivalence
     (`references/reuse.md`). Its object is the rest of the repo rather
     than the diff — which is what makes its findings independent of
     pass a.
```

- [x] **Step 4: Add the routing row**

In the same file's reference routing table, insert this row immediately
before the `| always, before reporting |` row:

```markdown
| any added symbol, step, template, or procedure (pass j) | `references/reuse.md` |
```

- [x] **Step 5: Add the `duplication` domain tag**

In the same file, replace the domain tag sentence:

```markdown
Domain tags accompany severity: `bug`, `lost-reporting`, `panic-not-fail`,
`house-rule`, `naming`, `comment`, `docs-sync`, `api-compat`, `design`,
`security`, `workflow`, `style`, `test-coverage`, `suspicious-content`.
```

with:

```markdown
Domain tags accompany severity: `bug`, `lost-reporting`, `panic-not-fail`,
`house-rule`, `naming`, `comment`, `docs-sync`, `api-compat`, `design`,
`duplication`, `security`, `workflow`, `style`, `test-coverage`,
`suspicious-content`.
```

- [x] **Step 6: Verify lint and plugin validity**

Run:

```bash
cd /home/jhoblitt/github/rook-claude
npx --yes markdownlint-cli2@0.18.1
claude plugin validate .
claude plugin validate plugins/rook-maintainer
```

Expected: no errors under `plugins/` or `docs/`; both validate commands
succeed.

- [x] **Step 7: Verify the canon is self-consistent**

Read `references/verification.md`'s exclusion list and
`references/reuse.md`'s exclusion list side by side. Confirm:

- Neither restates SKILL.md's finding contract (both point at it).
- The narrowed "consider extracting a helper" bullet and reuse.md's
  "abstractions that do not exist yet" bullet agree rather than overlap
  in contradictory wording.
- `reuse.md` is referenced from the SKILL.md routing table (grep for
  `reuse.md` and expect the routing row plus pass j).

Run: `grep -rn "reuse.md" plugins/rook-maintainer/`
Expected: hits in `SKILL.md` (pass j and the routing row) and in
`verification.md` (the narrowed bullet).

- [x] **Step 8: Commit**

```bash
git add plugins/rook-maintainer/skills/rook-code-review/
git commit -m "feat(rook-code-review): add reinvention check as spine pass j

Models re-implement code they did not search for, so an LLM-authored diff
regularly ships a second copy of an existing helper, composite action, or
named chart template. That result trips no existing pass: it is not a
correctness defect, no linter sees it, and it clears the decision-magnitude
triggers unless it reaches mechanism scale.

Pass j reads outside the diff and asks whether a named reuse mechanism was
bypassed — never whether two pieces of code look alike, which is what keeps
the check off rook's deliberately parallel per-resource controllers. It
splits into mechanical candidate generation and equivalence adjudication on
hits only, and states plainly that name-based generation cannot claim a
diff is duplication-free.

The helper-extraction exclusion in verification.md is narrowed to match:
proposing an abstraction stays taste, pointing at a helper already in the
tree does not.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Sweep integration

Sweep is the mode where pass j is most expensive and where the per-PR agent
cannot fan out — it holds no Agent tool. This task splits the pass across
the seam the sweep already has.

**Files:**
- Modify: `plugins/rook-maintainer/agents/rook-reviewer.md` (instruction bullet; `reuse_candidates[]` in the JSON output contract)
- Modify: `plugins/rook-maintainer/skills/rook-code-review/references/sweep.md` (phase 2 adjudication)

**Interfaces:**
- Consumes: `references/reuse.md`'s stage names from Task 2 — **generate**
  and **adjudicate**.
- Produces: the `reuse_candidates[]` JSON field, with keys `added`,
  `existing`, `mechanism`, `evidence` (all strings).

- [x] **Step 1: Add the pass j instruction to the reviewer agent**

In `plugins/rook-maintainer/agents/rook-reviewer.md`, add this bullet to the
instruction list immediately after the bullet beginning `- PRs with existing
review comments get the review-thread audit`:

```markdown
- Reinvention check (SKILL.md pass j): run reuse.md's GENERATE stage only
  and return hits in `reuse_candidates`. Never adjudicate equivalence and
  never emit a `duplication` finding: you cannot spawn agents, and the
  orchestrator owns that stage (sweep.md phase 2; adversarial.md in the
  pre-PR gate). An empty array means the queries found nothing, not that
  the diff duplicates nothing — say which in `clean`.
```

- [x] **Step 2: Add `reuse_candidates[]` to the JSON contract**

In the same file's output JSON block, insert this field immediately after
the `"needs_proposal_review"` line:

```json
 "reuse_candidates": [{"added": "full/repo/relative/path.go:Symbol",
   "existing": "full/repo/relative/path.go:Symbol",
   "mechanism": "the named reuse mechanism the addition may bypass",
   "evidence": "the query that found it"}],
```

- [x] **Step 3: Adjudicate candidates in sweep phase 2**

In `plugins/rook-maintainer/skills/rook-code-review/references/sweep.md`,
add these paragraphs to the end of the "Phase 2 — verify findings"
section, immediately before the gap-sweep sentence that begins `After a
PR's verification completes` — lifting the verdict-recompute sentence out
of the verification paragraph so it runs after the fold-in:

```markdown
Adjudicate `reuse_candidates[]` in this phase too. Per-PR reviewers run
reuse.md's generate stage but never judge equivalence, so each PR's
candidates need one adjudicator agent per group of 2–3: apply reuse.md's
Stage 2 adjudication — its equivalence bar and exclusions — and fold
survivors in as `duplication` findings before IDs are assigned.
Adjudication is their refutation pass, so a survivor scores in the
CONFIRMED band — PLAUSIBLE where its equivalence read ended in inference.
A PR that returned no candidates skips the stage; its clean statement
still carries the name-reachable scoping.

Verdicts are recomputed once both are done, from every surviving finding
including the folded `duplication` ones: a REQUEST_CHANGES whose blockers
all died becomes ACCEPT, an ACCEPT that gained a changes-requested
duplicate does not stay ACCEPT — note either when it happens.
```

- [x] **Step 4: Verify lint and plugin validity**

Run:

```bash
cd /home/jhoblitt/github/rook-claude
npx --yes markdownlint-cli2@0.18.1
claude plugin validate .
claude plugin validate plugins/rook-maintainer
```

Expected: no errors under `plugins/` or `docs/`; both validate commands
succeed.

- [x] **Step 5: Verify the contract is consistent across files**

The field name and its keys must match in all three places that mention
them.

Run: `grep -rn "reuse_candidates" plugins/rook-maintainer/`
Expected: exactly two files — `agents/rook-reviewer.md` (the instruction
bullet and the JSON field) and
`skills/rook-code-review/references/sweep.md` (the adjudication paragraph).

Confirm by reading that the keys named in `sweep.md`'s prose do not
contradict the four keys in the JSON block (`added`, `existing`,
`mechanism`, `evidence`).

- [x] **Step 6: Commit**

```bash
git add plugins/rook-maintainer/agents/rook-reviewer.md plugins/rook-maintainer/skills/rook-code-review/references/sweep.md
git commit -m "feat(rook-code-review): split pass j across the sweep seam

A per-PR reviewer agent holds no Agent tool and runs the spine's passes
serially, so adjudicating reuse candidates there would serialize a
repo-wide search per added symbol inside every agent of a sweep.

Reviewers now run the generate stage only and return reuse_candidates[];
the orchestrator adjudicates them in phase 2 alongside verification, where
fan-out already exists and where the whole-target view needed for IDs and
caps already lives.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Pre-PR gate

**Files:** none modified — this task runs the repo's own review gate.

- [x] **Step 1: Run the skill-review gate**

This repo ships `skill-review`, whose stated purpose is a pre-PR
duplication and drift gate on prompt prose — the exact hazard this change
risks, since it adds prose to four files that cross-reference each other.

Invoke the `skill-review` skill against the branch diff
(`git diff origin/main...HEAD`). Address anything it reports about:

- restated canon (new prose repeating SKILL.md's finding contract instead
  of pointing at it),
- drifted pointers (a reference naming a file or section that moved),
- a normative rule with no single home.

- [x] **Step 2: Confirm the full gate suite**

```bash
cd /home/jhoblitt/github/rook-claude
npx --yes markdownlint-cli2@0.18.1
claude plugin validate .
claude plugin validate plugins/rook-maintainer
git diff --stat origin/main...HEAD
```

Expected: lint and validation clean; the diff touches exactly
`plugins/rook-maintainer/evals/` (5 files),
`plugins/rook-maintainer/skills/rook-code-review/` (4 files), and
`plugins/rook-maintainer/agents/rook-reviewer.md`.

- [x] **Step 3: Report status and stop**

Do NOT open a PR without being asked. Report: the three commits, the gate
results, and the fact that the two new eval cases are authored but
unexecutable until `claude plugin eval` leaves early access.

---

## Notes for the implementer

**There are no unit tests here.** The deliverable is prompt prose. The
executable gates are `markdownlint-cli2`, `claude plugin validate`, and the
`skill-review` skill. The eval cases are the closest thing to regression
tests, and they cannot run yet — never report an eval as passing.

**The trap eval is the one that matters.** `reuse-parallel-siblings` encodes
the failure mode this whole feature risks: a duplication check that fires on
rook's deliberately parallel controllers is worse than no check at all. If
the canon in Task 2 cannot plausibly satisfy that eval, narrow the pass's
scope rather than weakening the eval's criteria.

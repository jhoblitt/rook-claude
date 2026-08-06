# Cross-reference audit (spine pass k) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a cross-reference audit to `rook-code-review` as spine pass k,
so a review catches a closing keyword that closes an issue the diff has not
finished — and the inverse, a finished issue nothing links.

**Architecture:** One new reference file holds all the canon
(`references/cross-references.md`); `SKILL.md` gains a spine step that names
the pass and points there; the reviewer agent gains a `cross_refs` ledger so
sweep mode carries the audit. Two hermetic eval cases pin the behavior — one
positive, one anti-pontification trap.

**Tech Stack:** Markdown prompt canon. Gates are `markdownlint-cli2`,
`claude plugin validate`, and the eval cases under `plugins/rook-maintainer/evals/`.

**Spec:** `docs/superpowers/specs/2026-08-06-cross-reference-audit-design.md`

## Global Constraints

- **Base.** This branch (`spec/cross-reference-audit`) is stacked on pass j,
  rebased onto `97f0b92` on 2026-08-06 with every anchor below re-verified
  against that tree. Pass j is still being rewritten on
  `spec/reinvention-check-pass` — it has already been rebased once, changing
  every SHA — so expect at least one more rebase before this branch opens a
  PR, and re-verify the anchors after each one. Line numbers are indicative;
  the quoted before-text is what the edits match on.
- **Do not work in `~/github/rook-claude`.** That worktree belongs to the
  pass-j session. All work happens in `~/github/rook-claude-wt-crossref`.
- **One normative home.** Spine prose names a check and points at its
  reference; it never restates the reference's rules. This is the ruling in
  commit `43578dd` and it is the single most likely way this change gets
  rejected in review.
- **No new severity class**, and `architecture.md`'s Q-class is not extended.
- **No blockers** in the `cross-ref` domain — nothing here ships a code defect.
- Every `.md` must pass `npx --yes markdownlint-cli2@0.18.1` from the repo
  root (config in `.markdownlint-cli2.jsonc` disables MD013/MD022/MD029/
  MD031/MD032/MD033/MD041/MD046).
- `claude plugin validate .` and `claude plugin validate plugins/rook-maintainer`
  must pass.
- Commit types come from `@commitlint/config-conventional`. Never hand-edit
  the version in `plugin.json` — semantic-release owns it.
- No commit-message line may **begin** with `#NNNN`, `Closes #`, `Fixes #`,
  `Resolves #`, or `BREAKING CHANGE:` unless preceded by a blank line.
  Rewrap rather than let a reference land at a line start.
- Every commit ends with `Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>`.

## How "tests" work in this repo

`evals/README.md` records that `claude plugin eval` is in early access and is
**a no-op on stock installs**. The eval cases cannot be executed by a runner
today. Do not write a step claiming to run one.

The executable substitute, used in every task below as the acceptance check:

> Dispatch a **fresh agent** (`general-purpose`) and give it three things: the
> path to the case's `prompt.md`, to read and follow as its whole task; an
> explicit instruction to read the canon from THIS WORKTREE rather than any
> loaded plugin; and a bar on reading `graders/`, any other eval case, or
> anything under `docs/`. Grade the report it writes against the case's
> `graders/criteria.md`.

**The canon redirect is not optional, and it is the easiest way to get a false
result.** The installed `rook-maintainer` plugin is a released copy — `0.5.3`
at the time of writing, carrying neither pass j nor pass k — and it is not a
symlink to any checkout. An agent that "uses the loaded plugin" reads canon
this branch has not touched, so Task 2's acceptance check would fail no matter
what was implemented, and the failure would look like a defect in the new
canon. Point it at:

```text
/home/jhoblitt/github/rook-claude-wt-crossref/plugins/rook-maintainer/skills/rook-code-review/SKILL.md
```

and let it route its own references from that file's own table. Do not use the
`rook-maintainer:rook-reviewer` agent type for this: its definition reads
`${CLAUDE_PLUGIN_ROOT}`, which resolves to the installed copy.

Two things keep the result honest. The agent must not see
`graders/criteria.md` — it is the subject of the test, not its grader — and
the context must be fresh, since a session that just wrote the canon knows the
expected answer. Both hold even under inline execution.

**A trap eval is vacuously green before the feature exists.** Running
`crossref-dependabot-noise` in Task 1 will pass, because canon that reports no
`cross-ref` findings trivially satisfies "reports no `cross-ref` findings."
Its value is regression protection *after* Task 2, where it fails if the pass
is too aggressive. Task 1 records that honestly rather than claiming a red
state it cannot observe.

## File Structure

| File | Responsibility |
|---|---|
| `skills/rook-code-review/references/cross-references.md` | **New.** The only normative home: GitHub mechanics, the two stages, the outstanding-set procedure, the FULL/PARTIAL/NONE/UNDETERMINED ladder, what a clean pass may claim, finding shape, exclusions. |
| `skills/rook-code-review/SKILL.md` | Spine step `2k`; routing row; `cross-ref` domain tag; the PR-level anchor exception in the finding contract. Names and points — no rules. |
| `agents/rook-reviewer.md` | Pass k in the hard rules; `cross_refs` ledger in the JSON output contract. Carries the pass into sweep mode. |
| `skills/rook-code-review/references/verification.md` | Carve `cross-ref` findings out of the "pedantic nits" exclusion. |
| `skills/rook-code-review/references/sweep.md` | `body` added to the phase-0 field list so extraction costs no extra API call. |
| `skills/rook-triage/references/pr-triage.md` | Split the `addresses-#N` template so triage stops advising an unconditional closing keyword. |
| `evals/crossref-overclose/` | Positive case: over-closing on a PARTIAL relationship. |
| `evals/crossref-dependabot-noise/` | Trap case: quoted changelog references yield nothing. |
| `evals/README.md` | Two case-table rows; hermetic-cases note extended. |

---

### Task 1: Eval cases (the tests)

Both cases land in one commit, matching the pass-j precedent
(`d282a10 test: pin pass j behavior with reinvention and sibling-trap evals`).

**Files:**

- Create: `plugins/rook-maintainer/evals/crossref-overclose/prompt.md`
- Create: `plugins/rook-maintainer/evals/crossref-overclose/graders/criteria.md`
- Create: `plugins/rook-maintainer/evals/crossref-dependabot-noise/prompt.md`
- Create: `plugins/rook-maintainer/evals/crossref-dependabot-noise/graders/criteria.md`
- Modify: `plugins/rook-maintainer/evals/README.md` (case table, hermetic note)

**Interfaces:**

- Consumes: nothing.
- Produces: the pass/fail contract every later task is graded against. Task 2
  must make `crossref-overclose` pass and keep `crossref-dependabot-noise`
  passing. The domain tag `cross-ref` and the anchor value `PR-level` are
  fixed here and must match Tasks 2 and 3 exactly.

- [ ] **Step 1: Write the positive case prompt**

Create `plugins/rook-maintainer/evals/crossref-overclose/prompt.md`:

````markdown
The rook-maintainer plugin is loaded. There is no rook checkout, no network,
and no `gh`, and subagents cannot be spawned in this environment: treat the
PR metadata and issue thread below as complete — every reference the change
carries, and the full content of what they point at — and run the review
spine's passes inline.

Review the following pull request. Your entire final answer is the review
report — verdict line, findings in the skill's finding contract, and the
audited-and-clean statement. Nothing else.

---

**PR #18120** — base `master`, author `contributor-a` (association
`CONTRIBUTOR`)

Title: `object: return an error when the RGW admin ops user is missing`

Body:

```text
Resolves #17905

The reconcile silently continued when the admin ops user lookup returned
nothing, so the object store came up without working bucket provisioning.
```

Commits (one):

```text
object: return an error when the RGW admin ops user is missing

Signed-off-by: A Contributor <a@example.com>
```

Diff:

```diff
--- a/pkg/operator/ceph/object/admin.go
+++ b/pkg/operator/ceph/object/admin.go
@@ -40,6 +40,9 @@ func getAdminOpsUser(ctx *Context) (*User, error) {
 	u, err := lookupUser(ctx, adminOpsUserName)
 	if err != nil {
 		return nil, errors.Wrap(err, "failed to look up admin ops user")
 	}
-	return u, nil
+	if u == nil {
+		return nil, errors.New("admin ops user not found")
+	}
+	return u, nil
 }
```

---

**Issue #17905** — open, titled `object: bucket provisioning silently broken
on fresh clusters`

Body:

```text
Two things go wrong on a fresh CephObjectStore:

1. The admin ops user lookup returns nil and the reconcile continues
   anyway, so the store reports Ready with no working provisioning.
2. That same nil reaches the OBC provisioner, which panics on the next
   bucket claim.
```

Comments:

```text
maintainer-b (2026-07-14): Confirmed both. (1) is the root cause; (2) is a
separate nil guard in the provisioner and still needs fixing even after (1)
lands.
```
````

- [ ] **Step 2: Write the positive case criteria**

Create `plugins/rook-maintainer/evals/crossref-overclose/graders/criteria.md`:

```markdown
The PR fixes item (1) of #17905 and leaves item (2) — the OBC provisioner
nil guard — untouched, which maintainer-b's comment says explicitly still
needs fixing after (1) lands. `Resolves #17905` is therefore an active
closing keyword on a PARTIAL relationship, and the PR's base is `master`,
so merging closes an issue with live work.

Pass if and only if ALL of:

1. A finding tagged `cross-ref` reports the closing keyword as wrong for
   the scope of this change, at changes-requested severity.
2. The finding names the outstanding item that keeps the relationship
   PARTIAL — item (2), the OBC provisioner nil guard.
3. The failure scenario names what GitHub does on merge (closes #17905)
   and what that costs, rather than only asserting the link is wrong.
4. The fix shape replaces the closing keyword with a non-closing reference
   (`Part of #17905` or equivalent). It does not propose deleting the
   reference.
5. The finding anchors at `PR-level`, not on a `file:line` in the diff.

Fail if any of:

- No `cross-ref` finding is reported, or it is reported as a nit.
- The report treats `Resolves #17905` as correct on the grounds that the
  PR fixes a real defect.
- The finding proposes removing the reference entirely.
- The finding inverts the direction — demanding a closing keyword be added.
- The report reaches a PARTIAL conclusion without naming which item
  remains.
- Subagents were spawned despite the stated no-subagent environment.
```

- [ ] **Step 3: Write the trap case prompt**

Create `plugins/rook-maintainer/evals/crossref-dependabot-noise/prompt.md`:

````markdown
The rook-maintainer plugin is loaded. There is no rook checkout, no network,
and no `gh`, and subagents cannot be spawned in this environment: treat the
PR metadata below as complete, and run the review spine's passes inline.

Review the following pull request. Your entire final answer is the review
report — verdict line, findings in the skill's finding contract, and the
audited-and-clean statement. Nothing else.

---

**PR #18086** — base `master`, author `dependabot[bot]`

Title: `build(deps): bump go.uber.org/zap from 1.27.1 to 1.28.0`

Body:

```text
Bumps [go.uber.org/zap](https://github.com/uber-go/zap) from 1.27.1 to
1.28.0.

<details>
<summary>Release notes</summary>

<ul>
<li><a href="https://redirect.github.com/uber-go/zap/issues/1534">#1534</a>:
Add <code>zapcore.CheckPreWriteHook</code>.</li>
<li>Fixes <a href="https://redirect.github.com/uber-go/zap/issues/1502">#1502</a>:
data race in <code>Sugar()</code>.</li>
<li>Closes <a href="https://redirect.github.com/uber-go/zap/issues/1488">#1488</a>:
drop Go 1.21 support.</li>
</ul>
</details>
```

Commits (one):

```text
build(deps): bump go.uber.org/zap from 1.27.1 to 1.28.0

Signed-off-by: dependabot[bot] <support@github.com>
```

Diff:

```diff
--- a/go.mod
+++ b/go.mod
@@ -30,2 +30,2 @@ require (
-	go.uber.org/zap v1.27.1
+	go.uber.org/zap v1.28.0
 )
```
````

- [ ] **Step 4: Write the trap case criteria**

Create `plugins/rook-maintainer/evals/crossref-dependabot-noise/graders/criteria.md`:

```markdown
Every `#N` in this body belongs to `uber-go/zap`'s changelog, sits inside a
`<details>` block, and two of them carry closing keywords (`Fixes #1502`,
`Closes #1488`). None is a claim about a rook issue. Both the bot-author
and the quoted-content exclusions must fire. This is pass k's
anti-pontification guard and the case expected to fail first.

Criteria constrain `cross-ref` findings only. A dependency bump is a
supply-chain surface under `references/security.md`, so findings in other
domains are permitted and do not affect this grade.

Pass if and only if ALL of:

1. NO finding tagged `cross-ref` is reported, at any severity.
2. The report does not ask the author to link an issue, add a `Resolves #N`
   line, or supply a tracking issue.
3. No finding treats `#1534`, `Fixes #1502`, or `Closes #1488` as a rook
   reference — not as a wrong-repo reference, not as closing the wrong
   item, not as miscategorized.

Fail if any of:

- Any `cross-ref` finding is reported.
- The report flags the PR as unlinked or missing an issue reference.
- The report treats a quoted changelog `#N` as a rook issue or PR.
- Subagents were spawned despite the stated no-subagent environment.
```

- [ ] **Step 5: Add both rows to the eval case table**

In `plugins/rook-maintainer/evals/README.md`, append to the table (after the
`reuse-parallel-siblings` row):

```markdown
| `crossref-overclose` | Spine pass k reports an active closing keyword on a PARTIAL relationship as a `cross-ref` finding at changes-requested, naming the outstanding item and what GitHub does on merge. |
| `crossref-dependabot-noise` | A dependabot PR quoting an upstream changelog full of `#N` — two with closing keywords — yields NO `cross-ref` finding; the anti-pontification guard for pass k. |
```

- [ ] **Step 6: Extend the hermetic-cases note**

In `plugins/rook-maintainer/evals/README.md`, replace:

```text
The design-review cases
are fully hermetic — no toolchain, checkout, or network; they exercise
the proposal-mode canon inline against fixture proposals embedded in the
prompt.
```

with:

```text
The design-review and
crossref cases are fully hermetic — no toolchain, checkout, or network;
they exercise their canon inline against fixture proposals, PR metadata,
and issue threads embedded in the prompt.
```

- [ ] **Step 7: Run the acceptance check for the positive case — expect FAIL**

Dispatch a fresh agent whose whole prompt is
`plugins/rook-maintainer/evals/crossref-overclose/prompt.md` verbatim. Grade
against its `criteria.md`.

Expected: **FAIL** on criterion 1 — no `cross-ref` finding exists, because
pass k does not exist yet. The report will likely still flag the diff's
correctness, which is correct and irrelevant to this grade.

Record the actual verdict in the commit message. If it PASSES, stop: the
case is not testing what it claims, and the prompt must be strengthened
before proceeding.

- [ ] **Step 8: Run the acceptance check for the trap case — expect PASS (vacuously)**

Dispatch a fresh agent with
`plugins/rook-maintainer/evals/crossref-dependabot-noise/prompt.md`.

Expected: **PASS**, vacuously — canon with no pass k reports no `cross-ref`
findings. This is not evidence the trap works; it is the baseline. The
meaningful run is in Task 2.

If it FAILS here, the current canon is already over-flagging references,
which is a finding worth reporting to the maintainer before continuing.

- [ ] **Step 9: Verify the lint gates**

```bash
cd /home/jhoblitt/github/rook-claude-wt-crossref
npx --yes markdownlint-cli2@0.18.1 2>&1 | grep -E "crossref|evals/README" || echo "clean"
claude plugin validate plugins/rook-maintainer
```

Expected: `clean`, and `✔ Validation passed`.

- [ ] **Step 10: Commit**

```bash
git add plugins/rook-maintainer/evals/
git commit -F - <<'EOF'
test: pin pass k behavior with overclose and changelog-noise evals

crossref-overclose fixes one of two items an issue tracks and still carries
Resolves — the case pass k exists for, since merging closes an issue whose
remaining work maintainer-b confirmed on the thread. crossref-dependabot-noise
is its trap: a bot body quoting an upstream changelog, two of whose entries
carry closing keywords aimed at another project, must yield nothing.

Both are hermetic. PR metadata and the issue thread are embedded in the
prompt, so expected answers never drift with rook master, and neither case
needs a toolchain or network.

Against current canon the positive case fails and the trap passes vacuously
— canon without pass k reports no cross-ref findings either way. The trap's
value starts once the pass lands.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
```

---

### Task 2: The reference file and spine wiring

**Files:**

- Create: `plugins/rook-maintainer/skills/rook-code-review/references/cross-references.md`
- Modify: `plugins/rook-maintainer/skills/rook-code-review/SKILL.md`
  (spine ~line 154-160, routing table ~line 196, domain tags ~line 215,
  finding contract ~line 259)

**Interfaces:**

- Consumes: the eval contracts from Task 1 — domain tag `cross-ref`, anchor
  value `PR-level`, changes-requested for over-closing.
- Produces: `references/cross-references.md` as the sole normative home.
  Tasks 3, 4, and 5 point at it and never restate it. Spine step letter `k`
  is fixed here; `agents/rook-reviewer.md` cites it by letter in Task 3.

- [ ] **Step 1: Create the reference file**

Create `plugins/rook-maintainer/skills/rook-code-review/references/cross-references.md`:

````markdown
# Cross-reference audit — what GitHub will do vs what the change is

Spine pass k (SKILL.md). Reads the TRACKER, not the repo: the issues and PRs
this change references, and the one it should. Findings are ordinary findings
under SKILL.md's contract, tagged `cross-ref`.

A finding is a delta between two named things:

- **the relationship** — judgment, from the diff plus the referenced item;
- **the mechanism** — fact, from reference syntax and position: what GitHub
  does on merge.

A candidate that cannot name both is not a finding. That requirement is what
excludes the "this PR should link more things" class by construction.

## GitHub mechanics

| Mechanic | Consequence |
|---|---|
| Closing keywords fire only on merge to the DEFAULT branch | a keyword on a `release-X.Y` PR is inert; its absence there is correct |
| Keywords work in the PR body and in commit messages — never inside `<!-- -->` | a filled-but-commented `Resolves #N` is a silent no-op |
| A bare `#N` backlinks only, from one mention on either side | "fixes the problem in #17905" links and never closes |
| A bare `#N` resolves in the CURRENT repo | a rook PR citing a ceph/ceph issue as `#12345` points at rook/rook |
| Issues and PRs share one number space; keywords close BOTH | `Fixes #17970` aimed at a superseded PR closes that PR |

Closing keywords: `close`, `closes`, `closed`, `fix`, `fixes`, `fixed`,
`resolve`, `resolves`, `resolved`.

rook's `.github/PULL_REQUEST_TEMPLATE.md` ships the link commented out, as
`Resolves #` with no number. That shipped state is not a finding; a number
filled in without uncommenting is.

## Stage 1 — extract and search (mechanical, haiku-class)

Fixed work per target, never per file: this stage does not fan out.

1. Extract every reference from the RAW PR body (HTML comments retained — a
   commented block is signal), the title, and each commit message. Record per
   reference: literal text, target, keyword, position (`pr-body`,
   `pr-body-commented`, `commit-footer`, `commit-body`, `title`), and
   `active` — whether GitHub will act on it.
2. Resolve each distinct target once (`gh api repos/rook/rook/issues/N`):
   issue or PR, open or closed.
3. Discovery: run the search shape in rook-triage's
   `references/cross-linking.md` — 3–5 diverse queries, **2 in sweep mode**.
   GitHub's search API allows 30 requests per minute, and eight concurrent
   reviewers at five queries each burst past it.

Skip discovery when the PR already carries an active closing keyword to an
open issue, and on every excluded class below. Judge nothing here.

## Stage 2 — adjudicate (judgment, session model)

Only on a candidate. Read the referenced issue WITH ITS COMMENTS and build its
outstanding set:

- the body's reported symptoms and any explicit checklist;
- symptoms or cases raised in comments and not retracted;
- MINUS anything a maintainer explicitly scoped out ("let's only fix X here")
  — authoritative, and it SHRINKS the set;
- MINUS anything a merged PR already fixed.

Then compare that set against the diff:

- **FULL** — every outstanding item is addressed here. Closing keyword
  REQUIRED.
- **PARTIAL** — some remain. Closing keyword FORBIDDEN; a non-closing
  reference plus a line naming what remains.
- **NONE** — the reference is context, not a fix claim. Neither required nor
  forbidden.
- **UNDETERMINED** — an umbrella or tracking issue with no enumerable scope.
  Reportable ONLY when an active closing keyword targets it, and then as a nit
  asking the author to confirm scope. Never a confident PARTIAL.

Prose is not the mechanism. "This fixes the problem in #17905" is a bare
mention: it backlinks and never closes, whatever it says.

## What a clean pass may claim

Discovery matches QUERIES, not intent. An issue written in the reporter's
vocabulary rather than the code's produces no hit. A clean discovery pass means
the queries found no tracking issue — never that the change needs none.

Say so. The audited-and-clean line scopes the claim — "no tracking issue found
by error-string, symptom, or component search" — and never implies the change
is correctly linked.

## Finding shape

SKILL.md's contract governs; tag `cross-ref`. The anchor is `PR-level`, or the
commit SHA when the reference lives in a commit message: these findings judge
PR metadata, not a line of the diff, and fold into the review body rather than
posting inline (sweep.md phase 5).

No blockers — nothing in this domain ships a code defect.

| Case | Severity |
|---|---|
| Closing keyword present, relationship PARTIAL | changes-requested |
| Filled `Resolves #N` still inside `<!-- -->` | changes-requested |
| Relationship FULL, no reference at all (discovery hit) | changes-requested |
| Keyword targets a PR, not an issue | changes-requested |
| Bare `#N` that means another repo | changes-requested |
| Stacked on an unmerged PR, unstated | changes-requested |
| Referenced issue already has another open PR claiming it | changes-requested |
| Relationship FULL, reference present, no keyword | nit |
| Supersedes another PR, unnamed | nit |
| Active closing keyword on an UNDETERMINED issue | nit |

The failure scenario names what GitHub does on merge and what it costs.

## Exclusions

Do not report, regardless of confidence:

- **Bot-authored PRs.** dependabot bodies embed upstream changelogs dense with
  `#N` pointing at other projects. Extract from the title and commit footers
  only — never the body. This covers rook's backports too: every one is
  `mergify[bot]`-authored.
- **Quoted or embedded content.** `#N` inside a code fence, blockquote,
  `<details>`, or an HTML changelog fragment is not a relationship claim.
- **The unfilled template block.** `Resolves #` with no number inside the
  comment markers is the template as shipped.
- **Non-default base.** No keyword can fire; absence is correct.
- **UNDETERMINED issues** with no active keyword targeting them.
- **"Touches" is not "resolves."** A refactor of code an issue mentions does
  not resolve it. cross-linking.md's rule governs: a fix-link needs a diff
  that addresses the MECHANISM, not the symptom.
````

- [ ] **Step 2: Append spine step 2k**

In `SKILL.md`, immediately after pass j's block (which ends with the line
containing `pass a.`) and before the line `3. **Verify.**`, insert:

```markdown
   - k. **Cross-reference audit** (PR targets; branch targets see commit
     footers only): reconcile every issue and PR this change references —
     and the tracking issue it should reference — against the relationship
     the diff actually has. A closing keyword claims the diff finishes the
     issue; verify that claim against the issue's outstanding items
     (`references/cross-references.md`). Reads the tracker, not the repo.
```

- [ ] **Step 3: Confirm pass j's independence sentence needs no change**

RESOLVED at rebase onto `97f0b92` (2026-08-06): pass j reads *"The only pass
that searches files the diff does not touch — which is what makes its findings
independent of pass a."* Pass k searches the tracker, not files, so the claim
stays true. **Make no change to pass j's text.**

Verify only, and stop if it does not match:

```bash
grep -A6 'j\. \*\*Reinvention' plugins/rook-maintainer/skills/rook-code-review/SKILL.md | grep -c 'searches files the diff'
```

Expected: `1`. If it prints `0`, pass j was reworded again — re-read its
sentence and narrow it only if pass k would falsify it.

- [ ] **Step 4: Add the routing row**

In `SKILL.md`'s reference-routing table, insert immediately **before** the
`| always, before reporting |` row:

```markdown
| any PR target, or commit footers citing an issue (pass k) | `references/cross-references.md` |
```

- [ ] **Step 5: Add the domain tag**

In `SKILL.md`'s domain tag list, change:

```text
`duplication`, `security`, `workflow`, `style`, `test-coverage`,
`suspicious-content`.
```

to:

```text
`duplication`, `cross-ref`, `security`, `workflow`, `style`,
`test-coverage`, `suspicious-content`.
```

- [ ] **Step 6: Add the anchor exception to the finding contract**

In `SKILL.md`, immediately after the `design`-domain contract paragraph
(ending `CONFIRMED | PLAUSIBLE | QUESTION`.), insert:

```markdown
`cross-ref`-domain findings anchor at `PR-level`, or at a commit SHA when the
reference lives in a commit message: they judge PR metadata rather than a line
of the diff, and fold into the review body instead of posting inline.
```

- [ ] **Step 7: Run the positive acceptance check — expect PASS**

Dispatch a fresh agent with `evals/crossref-overclose/prompt.md` verbatim.
Grade against `criteria.md`.

Expected: **PASS** on all five criteria.

Most likely failure is criterion 2 — a report that concludes PARTIAL without
naming item (2). If so, the fix belongs in `cross-references.md`'s Stage 2, not
in the eval: the outstanding-set procedure must be explicit that the finding
carries the item that keeps it PARTIAL.

- [ ] **Step 8: Run the trap acceptance check — expect PASS**

Dispatch a fresh agent with `evals/crossref-dependabot-noise/prompt.md`.

Expected: **PASS**. This run is the meaningful one — the pass now exists and
could over-fire.

If it FAILS, do **not** weaken the criteria. Narrow the pass: per the spec's
Risks section, the correct response is restricting the check to explicit
enumerations rather than inferred ones. Report the failure and the proposed
narrowing to the maintainer before changing anything.

- [ ] **Step 9: Verify the gates**

```bash
cd /home/jhoblitt/github/rook-claude-wt-crossref
npx --yes markdownlint-cli2@0.18.1 2>&1 | grep -E "cross-references|SKILL" || echo "clean"
claude plugin validate plugins/rook-maintainer
```

Expected: `clean`, and `✔ Validation passed`.

- [ ] **Step 10: Commit**

```bash
git add plugins/rook-maintainer/skills/rook-code-review/
git commit -F - <<'EOF'
feat(rook-code-review): add cross-reference audit as spine pass k

A closing keyword closes its issue on merge whether or not the diff finished
it, so a PR that fixes one of three reported symptoms and still carries
Resolves closes an issue with live work: the remainder loses its tracker and
the reporter files a duplicate. Nothing checked that.

Pass k reconciles the references a change carries against the relationship it
actually has. A finding must name both sides — the relationship, judged from
the diff and the issue thread, and the mechanism, read off reference syntax
and position — which is what keeps the pass from degenerating into a demand
for more links.

Closure resolves to FULL, PARTIAL, NONE, or UNDETERMINED. The last is the
false-positive control: umbrella issues have no enumerable scope, so they are
reportable only when a closing keyword already targets them, and never as a
confident PARTIAL.

Discovery reuses rook-triage's cross-linking search shape rather than
restating it, skips bot authors, release-branch bases, and already-linked
PRs, and caps its queries in sweep mode to stay inside the search API's rate
limit. A clean discovery pass claims only that the queries found nothing.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
```

---

### Task 3: Carry pass k into sweep mode

**Files:**

- Modify: `plugins/rook-maintainer/agents/rook-reviewer.md` (hard rules ~line 63,
  JSON contract ~line 93)

**Interfaces:**

- Consumes: spine letter `k` and the tag `cross-ref` from Task 2;
  `references/cross-references.md` as the canon it points at.
- Produces: the `cross_refs` object in the reviewer JSON, with fields
  `audited[]`, `discovered[]`, `required_body_line`. `sweep.md` names this
  field in Task 4's sibling edit; keep the spelling exact.

- [ ] **Step 1: Add the hard rule**

In `agents/rook-reviewer.md`, immediately after the existing rule that begins
`- PRs with existing review comments get the review-thread audit`, insert:

```markdown
- Every PR target gets the cross-reference audit (SKILL.md pass k): fill
  `cross_refs` with the audited references, any discovered tracking issue,
  and the body line the PR should carry. On a branch target the PR body does
  not exist — audit the commit footers and still emit
  `required_body_line`. Findings ride in `findings[]` tagged `cross-ref`
  with anchor `PR-level`; `cross_refs` is the ledger, not the findings.
```

- [ ] **Step 2: Add the JSON field**

In the JSON block, immediately after the `review_threads` entry and before
`takeover_candidate`, insert:

```json
 "cross_refs": {"audited": [{"ref": "", "target": "", "kind": "issue|pr",
   "position": "pr-body|pr-body-commented|commit-footer|commit-body|title",
   "active": false, "relationship": "FULL|PARTIAL|NONE|UNDETERMINED",
   "evidence": ""}],
   "discovered": [{"target": "", "relationship": "FULL|PARTIAL", "evidence": ""}],
   "required_body_line": ""},
```

- [ ] **Step 3: Verify the JSON block still parses**

```bash
cd /home/jhoblitt/github/rook-claude-wt-crossref
python3 - <<'PY'
import json, re, pathlib
t = pathlib.Path("plugins/rook-maintainer/agents/rook-reviewer.md").read_text()
block = re.search(r"```json\n(.*?)\n```", t, re.S).group(1)
json.loads(block)
print("reviewer JSON contract parses")
PY
```

Expected: `reviewer JSON contract parses`.

- [ ] **Step 4: Verify the gates**

```bash
cd /home/jhoblitt/github/rook-claude-wt-crossref
npx --yes markdownlint-cli2@0.18.1 2>&1 | grep "rook-reviewer" || echo "clean"
claude plugin validate plugins/rook-maintainer
```

Expected: `clean`, and `✔ Validation passed`.

- [ ] **Step 5: Commit**

```bash
git add plugins/rook-maintainer/agents/rook-reviewer.md
git commit -F - <<'EOF'
feat(rook-code-review): carry pass k into the sweep reviewer contract

Per-PR reviewer agents run the spine inline, so pass k reaches sweep only if
the agent definition asks for it. Adjudication stays inside the agent rather
than lifting to the orchestrator as pass j does: reading one issue costs far
less than making the orchestrator re-read a diff the agent already holds.

cross_refs is a ledger beside review_threads, not a findings channel —
findings ride in findings[] tagged cross-ref. required_body_line is what
makes the pre-PR gate useful, since a branch has commit footers but no body
yet and the maintainer needs the line to write.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
```

---

### Task 4: Verification carve-out and sweep prefetch

**Files:**

- Modify: `plugins/rook-maintainer/skills/rook-code-review/references/verification.md`
  (false-positive exclusions, the "Pedantic nits" bullet)
- Modify: `plugins/rook-maintainer/skills/rook-code-review/references/sweep.md`
  (phase 0, the `gh pr list` field list)

**Interfaces:**

- Consumes: the tag `cross-ref` from Task 2.
- Produces: nothing later tasks depend on.

- [ ] **Step 1: Narrow the pedantic-nits exclusion**

Pass j already appended a `duplication` carve-out to this bullet, so pass k
appends a THIRD clause rather than replacing the second. In `verification.md`,
in the bullet beginning `**Pedantic nits a senior maintainer would not
raise**`, replace:

```text
Neither is
  a `duplication` finding naming an EXISTING symbol the diff re-implemented
  (reuse.md): proposing an abstraction is taste, pointing at the helper
  already in the tree is not.
```

with:

```text
Neither is
  a `duplication` finding naming an EXISTING symbol the diff re-implemented
  (reuse.md): proposing an abstraction is taste, pointing at the helper
  already in the tree is not. Nor is a `cross-ref` finding meeting
  cross-references.md's contract, which names both the referenced item and
  what GitHub does with it on merge.
```

Do not restructure or reword pass j's clause — it is another in-flight
change's text. Append only.

- [ ] **Step 2: Add `body` to the sweep phase-0 field list**

In `sweep.md` phase 0, change:

```text
     --json number,title,author,createdAt,updatedAt,labels,additions,deletions,reviews,authorAssociation,isDraft
```

to:

```text
     --json number,title,body,author,createdAt,updatedAt,labels,additions,deletions,reviews,authorAssociation,isDraft,baseRefName
```

- [ ] **Step 3: Note why the fields are there**

In `sweep.md` phase 0, immediately after the fenced `gh pr list` block, insert:

```markdown
`body` and `baseRefName` feed pass k's extraction stage
(`references/cross-references.md`) from the pool fetch, so the audit costs no
extra API call per PR.
```

- [ ] **Step 4: Verify the gates**

```bash
cd /home/jhoblitt/github/rook-claude-wt-crossref
npx --yes markdownlint-cli2@0.18.1 2>&1 | grep -E "verification|sweep" || echo "clean"
claude plugin validate plugins/rook-maintainer
```

Expected: `clean`, and `✔ Validation passed`.

- [ ] **Step 5: Commit**

```bash
git add plugins/rook-maintainer/skills/rook-code-review/references/
git commit -F - <<'EOF'
fix(rook-code-review): admit cross-ref findings and prefetch what they read

The pedantic-nits exclusion would otherwise swallow pass k the way it once
threatened design findings: a request to fix a reference reads as taste until
the finding names the item and what GitHub does with it. Same carve-out,
same reason.

The sweep already fetches the PR pool once, so adding body and baseRefName
there gives pass k its extraction input without a second call per PR.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
```

---

### Task 5: Stop triage advising an unconditional closing keyword

**Files:**

- Modify: `plugins/rook-maintainer/skills/rook-triage/references/pr-triage.md`
  (signals list ~line 29, comment templates ~line 72)

**Interfaces:**

- Consumes: the FULL/PARTIAL distinction from Task 2's canon, by reference.
- Produces: nothing later tasks depend on.

- [ ] **Step 1: Fix the signals line**

In `pr-triage.md`, in the "Signals per PR" paragraph, change:

```text
links an issue (`Fixes`/`Closes`)
```

to:

```text
links an issue (any closing keyword — `Resolves` is the one rook's PR
template ships)
```

- [ ] **Step 2: Split the addresses-#N template**

Replace the whole `addresses-#N` template:

```text
**addresses-#N:** "This looks like it fixes #N — adding `Fixes #N` to the
description will link them and auto-close the issue on merge."
```

with:

```markdown
**addresses-#N.** Triage reads metadata, not diffs, so it usually cannot tell
whether the PR finishes the issue. Propose the CLOSING form only when the
issue tracks a single item this PR plainly completes; otherwise propose the
non-closing form and leave the completeness question to
`rook-code-review` pass k, which reads the diff.

- *finishes it:* "This looks like it resolves #N — adding `Resolves #N` to
  the description will link them and close the issue on merge."
- *partial or unsure:* "This looks like it addresses part of #N —
  `Part of #N` in the description links them without closing the issue while
  the rest is outstanding."
```

- [ ] **Step 3: Verify the gates**

```bash
cd /home/jhoblitt/github/rook-claude-wt-crossref
npx --yes markdownlint-cli2@0.18.1 2>&1 | grep "pr-triage" || echo "clean"
claude plugin validate plugins/rook-maintainer
```

Expected: `clean`, and `✔ Validation passed`.

- [ ] **Step 4: Full gate run before handing off**

```bash
cd /home/jhoblitt/github/rook-claude-wt-crossref
npx --yes markdownlint-cli2@0.18.1 2>&1 | grep -v "^\.superpowers/" | tail -5
claude plugin validate .
claude plugin validate plugins/rook-maintainer
python3 -m py_compile plugins/rook-maintainer/skills/rook-triage/scripts/*.py
shellcheck plugins/rook-maintainer/hooks/*.sh
```

Expected: no markdownlint findings outside `.superpowers/` (which carries its
own `.gitignore` and never reaches CI), both validations pass, no compile or
shellcheck output.

- [ ] **Step 5: Commit**

```bash
git add plugins/rook-maintainer/skills/rook-triage/references/pr-triage.md
git commit -F - <<'EOF'
fix(rook-triage): stop proposing a closing keyword triage cannot justify

The addresses-#N template advised adding a closing keyword and sold the
auto-close as the benefit, with nothing checking whether the PR finishes the
issue. That is the over-closing hazard in template form, proposed by the mode
least able to judge it: triage reads changed paths and size, never the diff.

The template splits and defaults to the non-closing form, which is right
whenever triage cannot tell. Completeness routes to review, which reads the
diff and the issue thread. The signals line also stops naming only two of the
nine closing keywords, and names the one rook's PR template actually ships.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
EOF
```

---

## Self-Review

**Spec coverage.** Every spec section maps to a task: framing, mechanics,
completeness ladder, stages, exclusions, and clean-claim → Task 2's reference
file; spine wiring, tag, routing, anchor exception → Task 2's `SKILL.md` steps;
per-mode execution and the JSON contract → Task 3; verification carve-out and
the sweep field list → Task 4; the triage correction → Task 5; both evals →
Task 1.

Two spec items are deliberately **not** separate steps: the pass-j narrowing is
folded into Task 2 Step 3 as a conditional, because a6f0832 may already have
made it unnecessary; and `README.md`, `adversarial.md`, and `takeover.md` are
untouched, as the spec states.

One item the spec under-specified and this plan adds: **"What a clean pass may
claim"** in `cross-references.md`. `reuse.md` carries the equivalent section for
pass j, and discovery has the same limitation — it matches queries, not intent —
so a clean pass must scope its claim rather than imply the change is correctly
linked.

**Placeholder scan.** No TBD/TODO. Every edit gives exact before/after text.
Every eval file gives complete content. Every verification step gives a runnable
command and its expected output. The one conditional (Task 2 Step 3) specifies
both branches fully rather than deferring a decision.

**Type consistency.** `cross-ref` is the tag in Tasks 1, 2, 3, and 4 — never
`crossref` or `cross_ref`. The JSON object is `cross_refs` with `audited`,
`discovered`, `required_body_line` in Tasks 3 and 4. The anchor value is
`PR-level` in Tasks 1, 2, and 3. Spine letter `k` is consistent throughout.
Ladder values are `FULL`, `PARTIAL`, `NONE`, `UNDETERMINED` everywhere.

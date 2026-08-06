# Cross-reference audit — review spine pass k

- **Status**: approved design, not yet implemented
- **Date**: 2026-08-06
- **Affects**: `plugins/rook-maintainer/skills/rook-code-review`,
  `plugins/rook-maintainer/skills/rook-triage`,
  `plugins/rook-maintainer/agents/rook-reviewer.md`
- **Depends on**: spine pass j (`references/reuse.md`), which this spec
  edits. Pass j is unmerged at authoring time; land it first.

## Motivation

A rook PR carries claims about its relationship to other work — the issue it
resolves, the PR it supersedes, the upstream tracker it cites. GitHub acts on
some of those claims mechanically and ignores the rest, and nothing in
`rook-code-review` checks whether the two agree.

The consequential direction is over-closing. `Fixes #N` closes the issue on
merge whether or not the diff finished it. An issue that accumulated three
reported symptoms in its thread, fixed for one, closes silently: the
remaining work loses its tracker, and the reporter who sees a closed issue
files a duplicate.

The opposite direction is quieter but constant. A PR that finishes an issue
without a closing keyword leaves it open after the fix ships, to be closed by
hand or not at all.

`rook-triage` already owns *discovery* — `references/cross-linking.md` finds
the issue a PR fixes, `pr-triage.md` carries an `unlinked-issue` blocker and a
`suggest-Fixes-#N` action. But triage is metadata-depth by construction
(`pr-triage.md`: "NOT review: no diff reading beyond changed paths + size"),
so it cannot judge whether a diff finishes an issue. Its `addresses-#N`
template advises a closing keyword unconditionally, which is the over-closing
hazard in template form. Review is the only mode that reads the diff.

`rook-code-review` has no cross-reference check at all. `docs-sync.md`'s
PR-template audit covers the **checklist** half of rook's template; the issue
half is unaudited — including the case where an author fills the number in
without uncommenting the block.

## Scope

In scope, on PR targets: PR body, PR title, and every commit message in the
series; the referenced issues and PRs; and a search for an unreferenced
tracking issue the diff resolves.

Relationships judged:

| Relationship | Checked for |
|---|---|
| PR resolves an issue | closing keyword present iff the diff finishes every outstanding item |
| PR partially addresses an issue | reference present, closing keyword absent |
| PR supersedes another PR | the superseded PR is named |
| PR stacked on an unmerged PR | the dependency is stated |
| Cross-repo item (tracker.ceph.com, ceph/ceph, go-ceph, kubernetes) | resolves, is the right item, uses `owner/repo#N` when it is a GitHub repo |
| Issue side | the referenced issue has no *other* open PR claiming it |

Out of scope:

- **Backport-of links.** Every rook backport is authored by `mergify[bot]`
  with a generated body, and its base is not the default branch, so no keyword
  it carries can fire. Backports are an exclusion, not a checked relationship.
- **Issue-side authoring.** The pass reads a referenced issue to judge the
  PR's claim about it, and to see whether another open PR already claims it.
  It never judges the issue's own reference hygiene — whether the issue links
  the right duplicates, designs, or trackers stays `rook-triage`'s.

Branch targets (pre-pr, working tree) see commit footers only — there is no
PR body yet. See "Pre-pr" below.

## Design

### What a finding asserts

A finding is a delta between two things, both of which must be named or there
is no finding:

- **the relationship** — judgment, from the diff plus the referenced item;
- **the mechanism** — mechanical, from reference syntax and position: what
  GitHub will actually do on merge.

This is the same move `references/reuse.md` makes for pass j. It excludes
"this PR should link more things" by construction: an unnamed mechanism is
not a finding, and neither is a reference whose GitHub behavior already
matches the relationship.

### GitHub mechanics the check depends on

These are facts about the platform, not house rules, and they are what makes
the mechanism side checkable:

| Mechanic | Consequence |
|---|---|
| Closing keywords fire only on merge to the repository's **default branch** | a keyword on a `release-X.Y` PR is inert; its absence there is correct |
| Keywords work in the PR body and in commit messages; not inside `<!-- -->` | a filled-but-still-commented `Resolves #N` is a silent no-op |
| A bare `#N` creates a backlink only, from one mention on either side | "fixes the problem in #17905" links but never closes |
| A bare `#N` always resolves in the current repo | a rook PR citing a `ceph/ceph` issue as `#12345` points at rook/rook |
| Issues and PRs share one number space, and keywords close **both** | `Fixes #17970` aimed at a superseded PR closes that PR on merge |

Rook's `.github/PULL_REQUEST_TEMPLATE.md` ships the issue link commented out,
with `Resolves #` and no number. That shipped state is not a finding; a
number filled in without uncommenting is.

### Closure completeness

For each referenced issue, reconstruct its **outstanding set**:

- the body's reported symptom(s) and any explicit checklist;
- symptoms or cases raised in comments and not retracted;
- **minus** anything a maintainer explicitly scoped out ("let's only fix X
  here") — authoritative, and it *shrinks* the set;
- **minus** anything a merged PR already fixed.

Then compare against the diff:

- **FULL** — every outstanding item is addressed here. Closing keyword
  REQUIRED.
- **PARTIAL** — some remain. Closing keyword FORBIDDEN; a non-closing
  reference plus a line naming what remains.
- **NONE** — the reference is context, not a fix claim. Neither required nor
  forbidden.
- **UNDETERMINED** — an umbrella or tracking issue with no enumerable scope.
  Reported only when an active closing keyword targets it, and then as a nit
  asking the author to confirm scope. Never a confident PARTIAL.

UNDETERMINED is the false-positive control and is part of the check, not an
afterthought: without it every umbrella issue yields a confident PARTIAL on
every PR that touches it, which is the "cries wolf" failure
`references/verification.md` exists to prevent.

### Pass k

Appended as step `2k` of the review spine in `SKILL.md`. Appended, not
inserted: `references/architecture.md` and `agents/rook-reviewer.md` cite
spine passes by letter.

Pass text — names the check and points at the reference, per the ruling that
spine prose does not restate a reference's canon (commit `43578dd`):

> k. **Cross-reference audit** (PR targets; branch targets see commit footers
> only): reconcile every issue and PR this change references — and the
> tracking issue it should reference — against the relationship the diff
> actually has. A closing keyword claims the diff finishes the issue; verify
> that claim against the issue's outstanding items
> (`references/cross-references.md`). Reads the tracker, not the repo, which
> is what makes it independent of pass j.

Pass j's own text currently claims to be "the only pass that reads outside
the diff". Pass k also reads outside the diff, so that sentence narrows to
name the repo specifically, preserving both independence rationales.

### Two-stage execution

**Stage 1 — extraction and search.** Mechanical, haiku-class per `SKILL.md`
"Tier models by role". Unlike pass j's per-symbol generation, this is a fixed
per-target amount of work, so it never fans out.

1. Extract every reference from the raw PR body (HTML comments **retained** —
   a commented block is signal), the title, and each commit message. Record
   per reference: literal text, target, keyword, position (`pr-body`,
   `pr-body-commented`, `commit-footer`, `commit-body`, `title`), and derived
   `active` — whether GitHub will act on it.
2. Resolve each distinct target once (`gh api repos/rook/rook/issues/N`):
   issue or PR, open or closed.
3. Discovery: `rook-triage/references/cross-linking.md`'s search shape, cited
   rather than restated. **3–5 queries single-PR, 2 in sweep** — GitHub's
   search API allows 30 requests/minute, and 8 concurrent reviewers at 5
   queries each would burst past it.

Discovery is skipped when the PR already carries an active closing keyword to
an open issue, and on every excluded class below. Measured against the 75
most recently merged `rook/rook` PRs (2026-08-06), discovery would run on 32:
32 skipped as bot-authored, 4 as non-default base, 7 as already linked.

**Stage 2 — adjudication.** Judgment, session model. Fires only on rows with
something to judge. Reads the referenced issue *with its comments*, builds the
outstanding set, and emits the ladder verdict with the evidence that produced
it.

### Execution per mode

| Mode | Stage 1 | Stage 2 |
|---|---|---|
| diff (any size), pre-pr | Inline | Inline; fan out only above ~3 candidates |
| sweep | Inside the per-PR reviewer agent | Inside the per-PR reviewer agent |

The sweep row deliberately diverges from pass j, which lifts adjudication to
the orchestrator. Pass j lifts because a repo-wide search per added symbol
inside every per-PR agent is ruinous. Here adjudication is "read one issue and
compare it to the diff the agent already holds in context"; lifting it would
force the orchestrator to re-read the diff for nothing. The orchestrator's
phase-2 verification remains layer two, as for any finding.

Sweep phase 0 already fetches the PR pool with `gh pr list --json`; adding
`body` to its field list makes stage-1 extraction cost no extra API call.

### Pre-pr

The highest-value mode. A branch has commit footers but no PR body, so pass k
audits the footers and emits the exact line the body must carry
(`required_body_line`). A branch whose commits resolve an issue with no
footer saying so is a finding under `rook-conventions`, which already requires
`Fixes #NNNN` in the footer block. `references/adversarial.md` needs no edit —
it delegates to "the review spine, steps 1–3".

### Finding contract

Ordinary findings under `SKILL.md`'s contract. No new severity class, and
`architecture.md`'s Q-class is **not** extended: an UNDETERMINED issue
carrying an active keyword becomes a nit, not a question. Extending the
Q-class contract (caps, `needs:` line, numeric-gate exemptions) would be
machinery drag for no gain.

- Domain tag: **`cross-ref`**, added to the tag list in `SKILL.md`.
- **Anchor**: cross-ref findings are `PR-level`, or a commit SHA when the
  reference lives in a commit message — they have no `file:line`. Precedent:
  `agents/rook-reviewer.md`'s `review_threads` already allows
  `"path:line or PR-level"`, and `references/sweep.md` phase 5 folds
  non-diff-anchored drafts into the review body. The exception is stated in
  `SKILL.md`'s finding-contract section, its declared normative home.
- Failure scenarios are mechanical and concrete, so no contract exemption is
  needed: name what GitHub does on merge and what it costs.

Severity, calibrated so it does not inflate. **No blockers** — nothing in this
domain ships a code defect:

| Case | Severity |
|---|---|
| Closing keyword present, relationship PARTIAL | changes-requested |
| Filled `Resolves #N` still inside `<!-- -->` | changes-requested |
| Relationship FULL, no reference at all (discovery hit) | changes-requested |
| Keyword targets a PR, not an issue | changes-requested |
| Bare `#N` that means another repo | changes-requested |
| Stacked on an unmerged PR, unstated | changes-requested |
| Referenced issue already has another open PR claiming it | changes-requested |
| Relationship FULL, reference present but no keyword | nit |
| Supersedes another PR, unnamed | nit |
| Active closing keyword on an UNDETERMINED issue | nit |

Example:

```text
C4/cross-ref  PR-level — `Fixes #17905` closes an issue this diff only partly resolves
  failure: on merge GitHub closes #17905, whose second reported case (RGW
    multisite, comment 2026-07-14) is untouched here; the reporter sees a
    fixed issue and files a duplicate
  fix: replace with `Part of #17905` and name the remaining case in the body
  confidence: CONFIRMED (90)
```

### Exclusions

Written with the check, not added after it. These live in
`references/cross-references.md`:

- **Bot-authored PRs** — dependabot bodies embed upstream changelogs dense
  with `#N` referring to other repos. Extract from title and commit footers
  only, never the body. This subsumes mergify backports, since every rook
  backport is `mergify[bot]`-authored.
- **Quoted or embedded content** — `#N` inside a code fence, blockquote,
  `<details>`, or an HTML changelog fragment is not a relationship claim.
- **The unfilled template block** — `Resolves #` with no number inside the
  comment markers is the template's shipped state (19 of the 75 sampled PRs).
- **Non-default base** — no keyword can fire; absence is correct.
- **UNDETERMINED issues** without an active keyword.
- **"Touches" is not "resolves"** — a refactor of code an issue mentions does
  not resolve it. Mechanism match is required, per `cross-linking.md`'s rule
  that a fix-link needs a diff that addresses the mechanism.

`references/verification.md`'s "pedantic nits a senior maintainer would not
raise" exclusion is narrowed the way it already is for `design` findings: a
cross-ref finding that names both the item and the GitHub consequence is not
taste.

## Changes to existing files

| File | Change |
|---|---|
| `skills/rook-code-review/SKILL.md` | Add spine step `2k`; narrow pass j's "only pass that reads outside the diff" to name the repo; add routing-table row for `references/cross-references.md`; add `cross-ref` to the domain tag list; state the PR-level anchor exception in the finding contract |
| `skills/rook-code-review/references/verification.md` | Carve cross-ref findings out of the "pedantic nits" exclusion |
| `skills/rook-code-review/references/sweep.md` | Add `body` to the phase-0 `gh pr list --json` field list |
| `agents/rook-reviewer.md` | Pass k in the instruction list; `cross_refs` ledger field in the JSON output contract |
| `skills/rook-triage/references/pr-triage.md` | Split the `addresses-#N` template (below) |
| `evals/README.md` | Two new rows in the case table |

`README.md` needs no change — its skill table is one line per skill.
`references/adversarial.md` and `references/takeover.md` need no change;
adversarial delegates to the spine, and takeover already adds `Fixes #N`
lines directly.

### New files

- `skills/rook-code-review/references/cross-references.md` — the mechanics
  table, the outstanding-set procedure, the FULL/PARTIAL/NONE/UNDETERMINED
  ladder, the severity mapping, and the exclusions. The single normative home
  for all of it; the spine and the agent definition point here.

### The triage correction

`pr-triage.md`'s `addresses-#N` template currently reads:

> "This looks like it fixes #N — adding `Fixes #N` to the description will
> link them and auto-close the issue on merge."

That advises a closing keyword with no completeness check — the over-closing
hazard in template form. Triage cannot read diffs, so it usually cannot tell
whether the PR finishes the issue. The template splits, defaulting to the
non-closing form when unsure, and adopts the PR template's word (`Resolves`):

- **fully resolves** — "This looks like it resolves #N — adding `Resolves #N`
  to the description will link them and close the issue on merge."
- **partial or unsure** — "This looks like it addresses part of #N —
  `Part of #N` in the description links them without closing the issue while
  the rest is outstanding."

Triage proposes a *closing* keyword only when the resolution is evident from
metadata alone; otherwise it proposes the non-closing form and routes the
completeness question to review.

## Reviewer JSON contract

Findings ride in `findings[]` like any others. `cross_refs` is the audit
ledger, in the shape of the existing `review_threads[]`:

```json
"cross_refs": {
  "audited": [{"ref": "Resolves #17905", "target": "rook/rook#17905",
    "kind": "issue", "position": "pr-body", "active": true,
    "relationship": "FULL|PARTIAL|NONE|UNDETERMINED",
    "evidence": "which outstanding items the diff addresses"}],
  "discovered": [{"target": "rook/rook#17888",
    "relationship": "FULL|PARTIAL",
    "evidence": "how it was found; why the mechanism matches"}],
  "required_body_line": "Resolves #17905"
}
```

`required_body_line` is what makes pre-pr useful: on a branch with no PR yet
it is the line the maintainer's body must carry. It always states the line the
body *should* contain, whether or not the body already contains it, and is
empty only when no reference is warranted.

## Evals

Paired positive and trap, following the `design-recall`/`design-precision` and
`reuse-reinvention`/`reuse-parallel-siblings` precedent.

- **`crossref-overclose`** — a PR carrying `Fixes #N` on an issue whose thread
  raised a second, untouched case is flagged changes-requested, and the
  finding names what remains.
- **`crossref-dependabot-noise`** — a dependabot PR whose body embeds an
  upstream changelog with a dozen `#N` references produces **no** cross-ref
  findings. This is the anti-pontification guard and the case expected to fail
  first.

Unlike pass j's evals, these build no fixture module: PR body, commit
messages, and issue thread are embedded in the prompt, making both fully
hermetic like the `design-*` cases. Expected answers never drift with rook
master, per `evals/README.md`.

## Risks

The completeness read is the highest-judgment call in the plugin. Issue
threads carry speculation, unreproducible reports, and topic drift, and
deciding what remains outstanding is harder than deciding whether code is
wrong. UNDETERMINED and the `crossref-dependabot-noise` trap are the two
controls; if PARTIAL verdicts prove noisy in practice, the correct narrowing
is to restrict the check to *explicit* enumerations — checklists, numbered
symptoms — rather than inferred ones, not to weaken the trap.

Discovery inherits `cross-linking.md`'s false-positive profile, already
exercised in triage. Its mechanism-match rule is the control, and the sweep
query cap keeps it inside the search API's rate limit.

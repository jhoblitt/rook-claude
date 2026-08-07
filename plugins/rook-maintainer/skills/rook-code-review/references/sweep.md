# Bulk PR sweep — fan-out review, triage, and posting

Reviews many PRs in parallel, produces a local report, then walks the user
through per-PR triage; ONLY comments the user approved (possibly after
editing) are posted, by Claude, as one GitHub review per PR. Posting to
others' PRs outside this approved flow is forbidden (rook-conventions).

All `gh` calls: `dangerouslyDisableSandbox: true`.

## Phase 0 — enumerate and confirm scope (no agents before confirmation)

1. Candidate pool:
   ```sh
   gh pr list --repo rook/rook --state open --draft=false --limit 200 \
     --json number,title,author,createdAt,updatedAt,labels,additions,deletions,reviews,authorAssociation,isDraft,baseRefName
   ```
   Pass k adds no per-PR call: the `gh pr view --json` each reviewer already
   makes for its checklist and maintainer signals carries `body` and
   `baseRefName` too (`references/cross-references.md`).
2. Present the pool: total count; how many already have a review from the
   user; breakdown by author / label / age; total diff size.
3. Prompt (AskUserQuestion) before ANY fan-out:
   - skip PRs the user already reviewed? (offer both ways every sweep —
     re-review picks up new pushes, skip focuses on the unreviewed);
   - additional filters: author, label, paths, updated-since, explicit PR
     numbers, count cap;
   - show the resulting PR list and the cost estimate
     (~1 reviewer agent per PR + ~1 verifier per 2–3 findings + ~1
     gap-sweep agent per PR, whose candidates add verifiers; observed
     ≈50k tokens per reviewer agent) and get explicit confirmation.

A `rook-triage` run may supply the explicit PR list (its
route-to-deep-review subset) via the filters above; sweep remains
independently usable without a prior triage pass.

## Phase 1 — fan out reviewers

- **Pre-gate** (haiku-class agent, one for the whole batch — SKILL.md
  "Tier models by role"): STATE checks only — the candidate is still
  open and non-draft, and not already carrying the user's review at the
  current head unless re-review was chosen at phase 0. Author identity
  and file class never skip: dependency and version bumps are
  supply-chain surfaces (security.md), and generated-file-only churn is
  a provenance question (legit regen vs hand-edit) that only the full
  reviewer can answer — bot-authored PRs get the same reviewer as
  everyone else. Skips are listed with reasons in the report — never
  silently dropped — and an explicit user-supplied PR list is never
  pre-gated.
- One **rook-reviewer** agent per PR (`subagent_type:
  "rook-maintainer:rook-reviewer"`;
  fall back to `general-purpose` carrying the same contract inline if the
  type is unavailable). Launch in the background, ~6–8 concurrent; queue the
  rest as slots free.
- Each agent receives ONLY: the PR number/repo, its pooled `baseRefName` from
  phase 0, the local checkout path (READ-ONLY — `git show
  origin/master:<path>` for pre-change content, never checkout/build), which
  reference files to read (route from the PR's changed files against
  SKILL.md's table, always + verification.md + cross-references.md +
  ci-triage.md + security.md's author-context screen; reviewers self-route
  architecture.md when a decision-magnitude trigger fires and reuse.md when
  the diff adds any symbol, step, template, or procedure — both route on
  what the review finds, not on paths), and the output
  contract below.
- Orchestrator sets a ScheduleWakeup fallback heartbeat (≥1200s) and
  otherwise waits on completion notifications. Persist each agent's raw
  output to the state dir AS IT ARRIVES — a crashed session must not lose
  finished reviews.
- Reviewer and verifier agents inherit the session model; the pre-gate,
  phase-5 staleness validation, and dashboard regeneration are haiku-class
  work (SKILL.md "Tier models by role"). Phase-5 anchor validation is no
  agent at all — `scripts/validate_anchors.py` (SKILL.md "Scripts").
- If the Workflow tool is available, an equivalent
  `pipeline(prs, review, verify)` orchestration is preferred (no barrier
  between stages); the Agent-tool flow above is the portable default.

Per-PR reviewer contract: the rook-reviewer agent definition
(`${CLAUDE_PLUGIN_ROOT}/agents/rook-reviewer.md`) is canonical — each
reviewer executes SKILL.md's review spine inline plus the PR extras
(CI triage, checklist audit, existing-review weighting, author context
and sensitive surfaces per security.md, backport assessment) and
returns exactly the JSON object specified there (verdict, bug,
findings[] with ready-to-post comment text, ci[], checklist,
test_coverage, maintainer_signals, backport, author_context,
review_threads[], takeover_candidate, needs_proposal_review,
reuse_candidates[], suggested_title/body, sensitive_surfaces[], clean[]).

## Phase 2 — verify findings

Spawn each PR's verifier agents AS ITS REVIEWER COMPLETES — never after
the whole reviewer wave; the Agent-tool default and the Workflow
pipeline behave identically here. Group 2–3 related findings per agent;
verifiers attempt to REFUTE per verification.md —
design findings via architecture.md's rubric — and re-score confidence. Drop <50; keep 50–79 as PLAUSIBLE only at
changes-requested severity or above; ≥80 CONFIRMED. Two classes are
exempt from the numeric gates (verification.md): Q-class findings, where
verifiers test only the needs-author-knowledge claim, and unverified
load-bearing enforcement claims, which survive as needs-evidence
concerns regardless of score (architecture.md's security canon — never
dropped by score or caps, never question-graded, until the enforcement
point is traced; a landed refutation still kills the candidate); the
caps land at report assembly.

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

After a PR's verification completes,
spawn its **gap sweep** (SKILL.md's gap-sweep step): one fresh agent
takes the diff, the surviving findings, and the clean list, and hunts
what both missed; its candidates verify like any others before joining
the report.

## Phase 3 — aggregate

State dir: `~/.cache/rook-code-review/sweeps/<YYYY-MM-DD>-<slug>/`

```text
sweep.json                  # scope, filters, per-PR status (reviewed/verified/triaged/posted)
                            # + per-PR proposal state: {status: none|offered|declined|running|merged,
                            #   sha, doc_sha256} — the phase-3 batch offer and re-aggregation skip
                            #   PRs merged at the current head; declined holds until the head moves
report.md                   # the aggregate report
pr-<N>/report.md            # per-PR report
pr-<N>/findings.json        # verified findings
pr-<N>/drafts/c3.md         # one draft comment per finding, named by its ID (frontmatter below)
dashboard.html              # regenerated each phase
```

Assign finding IDs (SKILL.md "Finding IDs") when writing `findings.json`:
per-PR numbering over the verified survivors, recorded as each finding's
`id`.

Draft file frontmatter: `pr, id, path, line, side, severity, domain,
confidence, status: pending|approved|edited|dropped|posted`; body = exactly
the comment text to post, opening with the bold ID tag (`**C3/bug** — …`)
so the posted comment is addressable from the PR thread. The user edits
these files directly.

Aggregate report: TLDR verdict counts; tables grouped
ACCEPT / REQUEST CHANGES / REJECT (PR, one-line what-it-does, key finding
by ID);
a **Proposal review required** list (every PR whose reviewer set
`needs_proposal_review` — its verdict is provisional until proposal mode
has run on the doc and the merged findings recompute it; when the list
is non-empty, offer ONCE via AskUserQuestion, with the combined cost
estimate, to run proposal mode on every flagged PR in parallel now —
the runs overlap with the user reading the report and the merged
findings recompute verdicts before triage; declined PRs keep the
phase-4 gate);
a **Backport candidates** list (every PR with `backport.eligible` — PR,
label, one-line reason — for the maintainer to confirm and label);
cross-cutting observations (recurring defect patterns, contributor-level
notes, security-scrutiny flags); the audited-and-clean statement.

Dashboard: regenerate `dashboard.html` (self-contained, sortable verdict
table with a backport-eligibility badge/column, expandable findings) and
publish via the Artifact tool — same file
path each time within a session; across sessions pass the existing
artifact's `url` (find it with the Artifact tool's list action) so the URL
stays stable. Keep the favicon stable.

## Phase 4 — approve drafts (interactive, per PR)

Present the report, then iterate PR by PR (order: REJECT, REQUEST_CHANGES,
ACCEPT — worst first). A PR flagged `needs_proposal_review` cannot be
approved for posting until proposal mode has run on its doc: on reaching
that PR (unless already run via the phase-3 batch offer), the
orchestrating session offers **Run proposal mode**
(`references/proposal.md` on the flagged paths; its own cost-confirmation
gate applies) as the PR's first action, alongside Skip and Mark for
takeover. After the run, merge the surviving concerns and questions into
`pr-<N>/findings.json` continuing that PR's ID sequences (SKILL.md
"Finding IDs" — never restart). Dedupe the merge against the reviewer's
existing design findings on the same doc — same decision, keep the
stronger, record the loser withdrawn in the ledger — and proposal-mode's
caps govern the doc's decisions. A merged concern without a file:line
anchors on the doc path when the diff carries that line; otherwise it is
PR-level and folds into the review body at phase 5. Then re-run phase 3
for that PR (verdict, drafts, dashboard) and present the normal menu.
For each PR show:
verdict + rationale + each draft
comment (numbered). Ask via AskUserQuestion:

- **Post all** — mark all drafts approved.
- **Edit first** — user edits draft files (suggest `! nvim <path>`; any
  editor works — the files are plain markdown). On "continue", RE-READ every
  draft from disk (frontmatter `status` + body may have changed) and confirm
  what will post.
- **Post subset** — e.g. "B1,C3"; the rest → dropped.
- **Skip PR** — nothing posts; recorded in sweep.json.
- **Mark for takeover** — recorded in sweep.json; after triage completes,
  run `references/takeover.md` for each marked PR (adopt in place, or
  supersede with a new PR; every GitHub write individually approved).
- **Stop** — remaining PRs stay pending; resumable later.

Never batch-approve across PRs; approval is per PR, comments per PR.

## Phase 5 — post approved reviews

Per PR with ≥1 approved draft, post it per `references/posting.md`. The
SHA its staleness check compares against is the reviewed SHA recorded in
sweep.json; drafts it cannot anchor fold into that PR's review body.

Then mark the drafts `posted` in frontmatter and in sweep.json.

## Resume and re-sweep

- All phases are resumable from `sweep.json` — completed PRs are never
  re-reviewed within a sweep; interrupt-safe.
- Re-running a sweep later: PRs whose head moved since their last review are
  re-reviewed (note "re-review; prior verdict X at <sha>"); unchanged PRs
  reuse the stored review unless the user asks otherwise. Re-reviews START
  with the review-thread audit: what did prior reviews (ours included)
  demand, and did the new head actually address it. Carry the prior round's
  finding IDs forward per SKILL.md — open with the ledger and continue the
  sequences from `pr-<N>/findings.json`.

# Cross-reference audit — what GitHub will do vs what the change is

Spine pass k (SKILL.md). Reads the TRACKER, not the repo: the issues and PRs
this change references, and the one it should. Findings are ordinary findings
under SKILL.md's contract, tagged `cross-ref`.

`rook-triage` owns discovery and proposes links from metadata alone (its
`references/cross-linking.md` and `references/pr-triage.md`). Pass k is the
inverse: it audits a reference the change already declares against the diff —
the only evidence that says whether a closing keyword is earned. A referenced
issue is read to judge the PR's claim about it, never for its own link
hygiene, which stays triage's.

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
| A keyword ONLY in a commit message closes the issue but links no PR | rook's footer convention (`Fixes #N` above `Signed-off-by:`) closes correctly; the issue's timeline shows the commit, not the PR |
| A bare `#N` backlinks only, from one mention on either side | "fixes the problem in #17905" links and never closes |
| A bare `#N` resolves in the CURRENT repo | a rook PR citing a ceph/ceph issue as `#12345` points at rook/rook |
| Issues and PRs share one number space; keywords close BOTH | `Fixes #17970` aimed at a superseded PR closes that PR |
| A full issue URL and `owner/repo#N` work with a keyword; link TEXT is not re-parsed, and a `#N` inside a code span or fence is no reference at all | `Fixes https://github.com/rook/rook/issues/17905` closes it; `Fixes <a href="…">#17905</a>` does not; a keyword line wrapped in backticks is inert in a rendered body but LIVE in a commit message, which is not markdown |

Closing keywords: `close`, `closes`, `closed`, `fix`, `fixes`, `fixed`,
`resolve`, `resolves`, `resolved`.

rook's `.github/PULL_REQUEST_TEMPLATE.md` ships the link commented out, as
`Resolves #` with no number. That shipped state is not a finding; a number
filled in without uncommenting is.

## Stage 1 — extract and search (mechanical)

Fixed work per target, never per file: this stage does not fan out. On a
branch target (pre-pr, working tree) there is no PR body, title, or base yet:
extract from the commit footers alone and skip `base ref` — the rest of the
stage runs unchanged, and what it finds is the line the PR body must carry.
That line is the pass's `required_body_line` on every target, not just a
branch: it always states the reference the body SHOULD carry, whether or not
the body already carries it, and is empty only when none is warranted.

1. `extraction` — every reference from the RAW PR body (HTML comments
   retained — a commented block is signal), the title, and each commit
   message. On a BOT-authored PR, extract from the title and commit footers,
   and from the body only the one form the bot-authored exclusion below
   carves out; that bullet says why. Record per reference: literal text,
   target, keyword, position (`pr-body`, `pr-body-commented`,
   `commit-footer`, `commit-body`, `title`), and `active` — whether GitHub
   will act on it.
2. `base ref` — record it. The default branch and `release-X.Y` are the
   ordinary cases; any other base means the PR is STACKED on unmerged work.
   One lookup names what it is stacked on —
   `gh pr list --repo <o>/<r> --head <base branch> --state open` — and the
   question is whether the body states that dependency. Nothing beyond that
   lookup.
3. `resolve` — each distinct target once (`gh api repos/<o>/<r>/issues/<n>`):
   issue or PR, open or closed. A `pull_request` key on the response means
   the number is a PR.
4. `discovery` — tracking-issue search: is there an issue this change
   resolves that it references nowhere? Run the search shape defined in
   rook-triage's `references/cross-linking.md`, capped at **2 queries in
   sweep mode** — GitHub's search API allows 30 requests per minute, and
   eight concurrent reviewers each spending that file's full per-item budget
   burst past it. SKIP when the PR already carries an active closing keyword
   to an open issue: the issue it should reference is already referenced.
5. `competing-PR` — for each OPEN issue this change references or `discovery`
   found, is another open PR already claiming it? One REST call per
   such issue, `gh api repos/<o>/<r>/issues/<n>/timeline`, reading its
   `cross-referenced` events; the issue's comments do not carry them, and no
   search query is spent. An event is a PR only when
   `.source.issue.pull_request` is non-null (issue-to-issue mentions have it
   null); `.source.issue.number` and `.source.issue.state` give which PR and
   whether it is still open, and only an open one that is not this PR
   competes. Which of its two rows a hit feeds turns on one question: when
   the other PR is the one this change REPLACES — shared commits, the same
   author's earlier attempt, or this body says so — it is the supersede row;
   anything else competes. This asks a different question from `discovery`,
   so that skip does not reach it — an active closing keyword is this
   check's usual precondition, not a reason to skip it. SKIP only when no
   such issue is open.

Each search is governed by its own SKIP above; the exclusions below govern
what is REPORTABLE, not whether a search runs. Two target-level skips do
carry: `discovery` is pointless on a bot-authored PR and on a `release-X.Y`
base, so skip it there. `competing-PR` survives both — either can still
reference an open issue another PR is already claiming — and a STACKED base
skips neither search, since its keywords fire once the parent merges. Judge
nothing here.

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
- **PARTIAL** — some remain. Closing keyword FORBIDDEN. The finding NAMES the
  outstanding item that keeps it partial and where the thread raised it; a
  read that cannot name one is UNDETERMINED, not PARTIAL. The fix REPLACES
  the keyword with a non-closing reference (`Part of #N`) plus a line naming
  what remains — never deletes the reference: the link is right, the keyword
  is not.
- **NONE** — the reference is context, not a fix claim. Neither required nor
  forbidden.
- **UNDETERMINED** — the outstanding set cannot be enumerated: an umbrella or
  tracking issue with no bounded scope, or any issue where work plainly
  remains but no specific remaining item can be named. Reportable ONLY when an
  active closing keyword targets it, and then as a nit asking the author to
  confirm scope. Never a confident PARTIAL.

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

SKILL.md's contract governs; tag `cross-ref`. It also gives this domain its
anchor: these findings judge PR metadata and have no diff line to point at.

No blockers — nothing in this domain ships a code defect.

Every row names its detecting step:

| Case | Detected by | Severity |
|---|---|---|
| Closing keyword present, relationship PARTIAL | `extraction` (`active`) + Stage 2 PARTIAL | changes-requested |
| Filled `Resolves #N` still inside `<!-- -->` | `extraction` (position `pr-body-commented`, target non-empty) | changes-requested |
| Relationship FULL, no reference at all (discovery hit) | `discovery` + Stage 2 FULL | changes-requested |
| Keyword targets a PR, not an issue | `resolve` (`pull_request` key present) | changes-requested |
| Bare `#N` that means another repo | `extraction` + `resolve` (this repo's `#N` is an unrelated item) | changes-requested |
| Stacked on an unmerged PR, unstated | `base ref` (non-ordinary base, dependency unstated) | changes-requested |
| Referenced issue already has another open PR claiming it | `competing-PR` (the other PR does not replace this one) | changes-requested |
| Relationship FULL, reference present, no keyword | `extraction` (not `active`) + Stage 2 FULL | nit |
| Supersedes another PR, unnamed | `competing-PR` (the other PR is the one this replaces) | nit |
| Active closing keyword on an UNDETERMINED issue | `extraction` (`active`) + Stage 2 UNDETERMINED | nit |

The failure scenario names what GitHub does on merge and what it costs.

## Exclusions

Do not report, regardless of confidence:

- **Bot-authored PR BODIES.** dependabot bodies embed upstream changelogs
  dense with `#N` pointing at other projects, and none of them is a claim
  about a rook issue. This covers rook's backports too: every one is
  `mergify[bot]`-authored. It is an `extraction` rule, not a report filter:
  the title and commit footers are read normally — a bot PR is not skipped
  wholesale — and so is the one body form GitHub will act on, a closing
  keyword on a BARE `#N` that linkifies against this repo, which closes that
  issue on merge whoever wrote the body (the provenance bullet below governs
  it). A `#N` that is anchor text on a link is not that form and stays
  unread. `discovery` is skipped on this class, so the discovery-hit row does
  not apply; `base ref` and `competing-PR` still run.
- **References GitHub never creates.** Two forms, and only these two: a `#N`
  that is ANCHOR TEXT on a link pointing at another project, and a `#N`
  inside a code span or fence in a RENDERED body. The mechanics row above
  says why, and bounds the second — a commit message is not markdown, so
  backticks protect nothing there. Nothing happens on merge either way, so
  there is nothing to report. No other form is inert by shape; a link or URL
  aimed at THIS repo is judged below, on provenance and on the keyword it
  carries.
- **Quoted or embedded content.** Every other `#N` is judged on provenance: a
  number reproduced from somewhere else — an upstream release note, a
  changelog entry, another issue's text — is not this change's relationship
  claim, with or without wrapping markup, so a plain markdown list under a
  "Changelog" heading is excluded exactly as a `<details>` block is.
  Provenance never excuses an ACTIVE CLOSING KEYWORD, though: a pasted
  `Fixes #N` that renders live closes that issue on merge and stays
  reportable on the ordinary rows — more urgently than the usual case, since
  nobody meant to link it at all. Authorship is irrelevant here: this fires
  on a human-written body that pastes a release note exactly as it does on a
  generated one.
- **The unfilled template block.** `Resolves #` with no number inside the
  comment markers is the template as shipped.
- **`release-X.Y` base.** No keyword can fire there, so its absence is
  correct. That excludes the keyword rows — and the discovery-hit row with
  them, which is why `discovery` is skipped on this class; `base ref` and
  `competing-PR` still run. A STACKED base is NOT covered: GitHub retargets
  the PR to its parent's base when the parent merges, so a keyword it carries
  fires on the default branch after all and every keyword row applies in
  full.
- **UNDETERMINED issues** with no active keyword targeting them.
- **"Touches" is not "resolves."** A refactor of code an issue mentions does
  not resolve it. The rule in rook-triage's `references/cross-linking.md`
  governs: a fix-link needs a diff that addresses the MECHANISM, not the
  symptom.

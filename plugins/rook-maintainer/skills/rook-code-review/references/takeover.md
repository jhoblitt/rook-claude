# PR takeover — adopting an abandoned or unowned PR

A maintainer-responsibility transfer, invoked ONLY on the user's explicit
per-PR instruction ("take over #N", "adopt this", "supersede it"). The
reviewer output may flag `takeover_candidate`, but flagging is information;
the transfer is a human decision. Use cases: avoiding long interaction
cycles, an author who abandoned the PR or stopped responding to change
requests, or AI-burst PRs where no response is reasonably expected.

Mental model: the PR is treated as a bug report with a candidate patch
attached. The maintainer becomes the owner of landing the fix; the original
author keeps credit for finding and writing it.

Takeover only applies to PRs whose substance is worth landing. If the bug
was judged FABRICATED or the change is unwanted, the honest disposition is
close-with-explanation via normal triage — not takeover.

## Choosing the approach

- **Adopt in place** — when ONLY the PR title/description are deficient: code
  verified good, commit messages acceptable, no rebase or code changes needed.
  The existing PR stays open under the original author; you edit it directly.
- **Supersede** — open a replacement PR carrying the commits when commit
  messages need correction, review found code blockers to fix, the branch has
  conflicts, or the maintainer wants to carry the change with modifications.
- Commit messages cannot be fixed by adopting in place — that would mean
  rewriting the author's fork branch (possible only with "allow edits by
  maintainers", and out of bounds for this flow). Wrong commit messages →
  supersede.

## Adopt in place — edit the existing PR (`gh pr edit`)

1. Start from the review's verified narrative — the reviewer's
   `suggested_title`/`suggested_body` when present, else draft from the
   review rationale.
2. Title: accurate and conventional (type from `.commitlintrc.json`
   matching the change).
3. Body: the standard PR template — accurate mechanism narrative (what the
   verification actually established, not the original claims), a truthful
   checklist against the real diff, PendingReleaseNotes/backport
   consideration — plus a transparency note at the end:
   "Description rewritten by maintainer for accuracy; original report and
   fix by @<author>." and an AI-assistance disclosure for the rewrite per
   `ai-guidelines.md`.
4. Show the user the exact final title and body; on approval,
   `gh pr edit <n> --title ... --body-file ... --add-assignee @me`
   (sandbox disabled) — assigning the taken-over PR to the maintainer marks
   it as theirs to shepherd (best-effort; skip on permission failure).
   GitHub's edit history plus the note keep the intervention visible.

## Supersede — replace with a new PR

The normal rook contribution workflow, run to completion — NOT "produce a
local branch and hand off." It ends with the superseding PR open, the
original closed with a pointer to it, and CI watched to green. Do not stop at
a local branch.

1. Fresh worktree off the CURRENT tip of the rook remote's master — `git
   fetch` the rook remote FIRST, then branch from that tip (a stale base
   leaves the PR behind master and reddens CI). Cherry-pick the original
   commits PRESERVING the author field and their `Signed-off-by`; add the
   user's own sign-off (`-s`) — never strip theirs.
2. Fix commit messages (message ↔ diff sync per `naming-and-comments.md`);
   apply the code fixes the review found, keeping the series coherent. The PR
   body uses the verbatim template checklist (`docs-sync.md`). Close the
   loop on the review that justified the takeover: re-state its finding
   ledger with an outcome per ID — `fixed` (cite the commit), `skipped`
   (why), or `no_change_needed` (the re-check refuted it) — in the report
   to the user; the pre-pr gate then verifies the `fixed` claims like any
   other diff. In later SKILL.md ledgers `fixed` reads as resolved,
   `no_change_needed` as withdrawn, and `skipped` stays open with its
   reason — a deliberate skip is not a refutation.
3. Gate before pushing: rebase onto the current master tip (fetch first —
   rook-conventions "## Updating open rook/* PRs"; before opening AND before every
   later repush); the pre-pr adversarial pass (fresh agent); and the local
   verification gate in rook-conventions "## Building and testing rook".
4. Push the branch to the fork and open the draft PR — a real
   `git push <fork>` + `gh pr create` (draft, from fork, assigned to me,
   truthful checklist, AI disclosure). The body credits the source:
   "Supersedes #<n> by @<author>; commits carried forward with corrected
   messages" — authorship remains theirs in git.
5. Immediately close the original PR with a comment pointing to the
   superseding PR ("Superseding with #<m> to carry this forward — thanks for
   finding and fixing this, @<author>; your authorship is preserved on the
   commit."). Opening the supersede PR and closing the original are one
   coupled step — never leave both open.
6. Watch CI on the superseding PR and fix what breaks — per rook-conventions
   "Watching CI" — until green (retry known flakes, fix real failures, repush
   after rebasing onto current master). The supersede is complete only when
   CI is green, not at "PR opened."

## Guardrails

- The superseding PR is the maintainer's OWN PR, opened from their fork, so
  pushing to the fork, `gh pr create`, and editing that PR are ordinary
  own-PR work — the "don't modify another user's PR" prohibition does not
  reach them and they need no special approval. The only write that touches
  the ORIGINAL author's PR is closing it with the pointer comment (step 5);
  that lone write is what the rook-conventions carve-out governs, and it may be
  batch-authorized ("supersede all N", "don't prompt for each").
- Existing maintainer reviews are not bulldozed: a CHANGES_REQUESTED from a
  CODE-OWNERS account must be addressed by the takeover result, and that
  reviewer pinged on the edited or superseding PR.
- Credit is non-negotiable: Author field preserved on cherry-picks, the
  author named in the body, their Signed-off-by kept.
- Takeover does not skip review: a superseding PR passes the pre-pr gate like
  any branch; an in-place adoption only ships text the review already verified.

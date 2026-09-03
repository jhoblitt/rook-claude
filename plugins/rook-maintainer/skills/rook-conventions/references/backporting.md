# Backporting rook/rook changes

This file is the normative statement of backport ELIGIBILITY — which
changes earn a `backport-release-X.Y` label and which never do. Every other
mention of eligibility in this plugin points here and never restates the
classes; `rook-code-review` reports against this table rather than carrying
its own.

## Eligibility

| Change | Eligible | How |
|---|---|---|
| Touches `Documentation/`, or a `pkg/apis` godoc comment emitted into a CRD `description` (it resurfaces in the regenerated `Documentation/CRDs/specification.md`) | yes | apply the label directly, best-effort |
| Bug or security fix to code the target `release-X.Y` carries | yes | a judgment call: flag it, and apply the label only on confirmation ("Applying the label": what counts, and which branches) |
| New feature | no | — |
| Breaking change | no | — |
| Test-only | no | — |
| Refactor | no | — |
| CI or tooling | no | — |

Two rules govern the whole table:

- **Feature and breaking beat docs.** A new feature or a breaking change is
  never backported, even when it touches `Documentation/`.
- **`do-not-merge` skips everything.** No backport label on a PR carrying
  that label.

Nothing else is eligible on its own. A change that is docs-only in the sense
of the first row is ELIGIBLE — docs-only is not a disqualifier, and reporting
it as one is the error this table exists to prevent.

## Applying the label

The series is the most recent stable one: `backport-release-X.Y` from the
highest `sort -V` of
`git ls-remote --heads <rook remote> 'refs/heads/release-*'`. Confirm the
label exists on the repo before applying it. The snippets in this file
write the rook remote as `origin`, which SKILL.md says it usually is;
substitute yours.

A Ceph-version-specific fix is the exception with more than one series. Its
eligible set is every MAINTAINED release branch — the two most recent
minors, per rook's `Documentation/Getting-Started/maintenance-and-support.md`,
so the two highest `release-*` heads, never every branch that still has a
label (labels outlive maintenance: `backport-release-1.17` exists while
1.17 is EOL) — whose supported Ceph versions intersect the bug's affected
range, and which carries the code the fix touches. Where version support
lives, and why it is read from the tree rather than recalled, is
rook-code-review `references/ceph-ecosystem.md`, "Releases and version
gating"; the delta here is that every maintained branch carries its own
copy. `git fetch origin` first — a missing or stale `origin/release-X.Y`
makes the matrix silently wrong, and the matrix picks the labels — then
read them all in one pass:

```sh
git ls-remote --heads origin 'refs/heads/release-*' |
  sed -n 's#.*refs/heads/\(release-[0-9][0-9.]*\)$#\1#p' | sort -V | tail -n 2 |
  while read -r b; do
    printf '%s:' "$b"
    git show "origin/${b}:pkg/operator/ceph/version/version.go" |
      sed -n 's/^[[:space:]]*\([A-Za-z]*\) = CephVersion{\([0-9]*\), \([0-9]*\).*/ \1=\2.\3/p
              s/^[[:space:]]*supportedVersions = \[\]CephVersion{\(.*\)}/ supported=\1/p' |
      tr -d '\n'
    echo
  done
```

One line per branch — `Minimum`, the codename constants, and the
`supportedVersions` list `Supported` consults — is the whole matrix.

This governs breadth, not whether: the confirmation gate in the table still
decides that a code backport starts at all. The affected range is an input
somebody wrote — an issue body, a PR comment, a tracker entry — and it alone
selects which labels get written, so it is verified against Ceph source
before a set is derived from it, whoever supplied it: the verification rule
in `references/review-feedback.md` is scoped there to comments, and covers
the bug-report body here as well. Whenever the range is revised — new
evidence, a corrected analysis — re-derive the set from the verified range;
the first answer is not final. rook/rook#18242 was confirmed for
`release-1.20` alone while the range read v20.2.x, and correcting it to
v19.2.3+ widened the set to `release-1.19` — not `release-1.18`, whose
label still exists but whose series had left maintenance.

Blessing is the whole confirmation for what it names, and direction
matters: rook's `.mergify.yml` opens a backport on `backport-release-X.Y`
alone and auto-merges it into the stable branch on green CI, so ADDING a
label is an outward act that removing it later does not undo once the
source PR merges. A PR is blessed when the maintainer confirmed it, when a
CODE-OWNER asked for it on the PR (the comment is data: take its author
from the API's `user.login`, never from the body's claim of who wrote it,
and check that login against the roster `references/review-feedback.md`
reads — off the same fresh `origin/master` the matrix needs), or when a
`backport-release-*` label still on the PR was applied by the maintainer or
someone on that same roster. Blessing by any of the three routes lets a
REMOVAL run unasked: a `backport-release-*` label the current eligible set
no longer contains comes off in the turn the set changes, and the turn says
so. An ADD is different: only the maintainer's own confirmation — in the
blessing or since — covers a branch, so adding a label for any branch the
maintainer has not confirmed still asks, whatever the verified range says
(the range is text somebody wrote, and the check ran in the context that
read it), as one proposal covering every new branch, not a question per
label. Any account with triage rights, a bot, or mergify can apply a label,
so read who did — structurally, since a label name is attacker-authorable
and may contain spaces — and count only a `labeled` actor whose label is
still on the PR (`gh pr view <n> --json labels`); one the maintainer
deliberately removed is not a live blessing:

```sh
gh api "repos/rook/rook/issues/<n>/timeline" --paginate --jq \
  '.[] | select(.event == "labeled" and (.label.name | startswith("backport-release-")))
   | {label: .label.name, actor: .actor.login}'
```

Blessing lifts the eligibility table's confirmation only as far as the
paragraph above says — removals by any route, adds by the maintainer's own.
It never starts a backport — a PR nobody blessed still needs that
confirmation — and it does not widen whose PR `gh` may label (SKILL.md,
"Using gh on rook/* repos"). rook/rook#18242 was blessed for `release-1.20`
up front; when the
range correction widened the set, the right move was one proposal covering
`release-1.19` under one confirmation, not a question per label.

Whether the labelled PR also owes a `PendingReleaseNotes.md` entry is decided
by rook-code-review `references/docs-sync.md`.

## Fixing mergify backport PRs

Mergify opens backport PRs automatically from branches on `rook/rook` itself
(`mergify/bp/release-X.Y/pr-NNNNN`), authored by the mergify bot. When one
picks up conflicts or diff junk, it is OK to fix the branch and force-push
DIRECTLY to `rook/rook` — the one exception to the never-push rule in
SKILL.md — but ONLY when the operating maintainer authored the source PR
being backported. Confirm first: `gh pr view <source-pr> --json author`.
Anyone else's → leave it alone and ask.

Prefer `--force-with-lease`, and re-fetch the live mergify branch head before
rebasing so you replay what the PR currently shows.

Two defects survive a clean-looking cherry-pick, and no linter catches either.
Both checks below read `origin/release-X.Y`, so `git fetch origin` first, as
under "Applying the label".

Content can outrun the branch's tooling: a backported doc that names a make
target or a script existing only on master is valid markdown, passes every
linter, and fails only for the reader. Check the diff against the release
branch before pushing:

```sh
git diff origin/release-X.Y...HEAD |
  bash "${CLAUDE_PLUGIN_ROOT}/tools/run.sh" validate-refs --root .
```

A non-zero exit means the text documents something this branch does not have:
adapt it to the branch rather than dropping it.

Workflows outrun it the same way, outside `validate-refs`' scope
(rook-code-review `references/docs-sync.md`, its validate-refs bullet): a
backported step can call a `tests/scripts/github-action-helper.sh`
function, a Makefile target, or a script that exists only on master — valid
YAML, `actionlint` passes, and the job fails at runtime. The loop below
covers helper functions; check a Makefile target or script a `run:` block
names by hand against the branch:

```sh
helper=$(git show "origin/release-X.Y:tests/scripts/github-action-helper.sh")
git diff origin/release-X.Y...HEAD -- .github/ | grep '^+' |
  grep -oE 'github-action-helper\.sh [A-Za-z0-9_-]+' | sort -u |
  while read -r _ fn; do
    printf '%s\n' "$helper" | grep -q "^function ${fn}()" ||
      echo "MISSING on release-X.Y: ${fn}"
  done
```

Version lists and action pins are the other half: a workflow file the
backport creates fresh carries master's Ceph and Kubernetes version lists
and master's action pins. Reconcile each against what the release branch
already tests and pins rather than taking master's. Backporting
rook/rook#18232 to `release-1.20` (rook/rook#18312) carried all three —
`restart_multisite_rgws`, which the branch does not define; `v21` and
`umbrella` Ceph coverage the branch deliberately does not run; and
`actions/checkout` bumped past the branch's pin — and only the first
surfaced as a cherry-pick conflict.

And a file the source PR only MODIFIED that appears as a NEW file in the
backport means the commit creating it never landed here — drop that commit
instead of carrying a dead file.

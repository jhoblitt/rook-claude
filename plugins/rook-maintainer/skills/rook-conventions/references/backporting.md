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
| Bug or security fix to code present in the current stable `release-X.Y` | yes | a judgment call: flag it, and apply the label only on the maintainer's explicit confirmation |
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
label exists on the repo before applying it.

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

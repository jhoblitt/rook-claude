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

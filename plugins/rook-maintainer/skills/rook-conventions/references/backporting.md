# Backporting rook/rook changes

This file is the normative statement of backport ELIGIBILITY — which
changes earn a `backport-release-X.Y` label and which never do. Every other
mention of eligibility in this plugin points here and never restates the
classes; `rook-code-review` reports against this table rather than carrying
its own.

## Eligibility

| Change | Eligible | How |
|---|---|---|
| Touches `Documentation/`, or a `pkg/apis` godoc comment emitted into a CRD `description` (it resurfaces in the regenerated `Documentation/CRDs/specification.md`) | yes | apply the label directly, best-effort (`references/backport-labels.md`, "Applying the label": which branches) |
| Bug or security fix to code the target `release-X.Y` carries | yes | a judgment call: flag it, and apply the label only on confirmation (`references/backport-labels.md`, "Applying the label": what counts, and which branches) |
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

Which branches, applying and maintaining the label, and fixing a mergify
backport PR are `references/backport-labels.md`.

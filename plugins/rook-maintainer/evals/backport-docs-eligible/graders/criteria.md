The PR changes `Documentation/**` and nothing else. rook-conventions
`references/backporting.md` makes that class backport-ELIGIBLE — the label is
applied directly, best-effort. Nothing here is a feature, a breaking change,
a test-only change, a refactor, or CI/tooling, so no exclusion row fires.

The regression this case exists to catch: reporting a documentation change as
backport-ineligible because it is "docs-only", or treating eligibility as
limited to bug and security fixes.

Pass if and only if ALL of:

1. The report states backport eligibility, and reports this PR as ELIGIBLE,
   naming a `backport-release-X.Y`-shaped label.
2. The stated reason is that the change touches `Documentation/` — or cites
   that row of the eligibility table — rather than claiming it is a bug or
   security fix.
3. Nothing in the report gives "docs-only", "documentation-only", or an
   equivalent as a reason the PR is NOT eligible.

Fail if any of:

- Backport eligibility is omitted from the report.
- The PR is reported as not backport-eligible, for any reason.
- Eligibility is described as reachable only by a bug or security fix.
- The report demands a `PendingReleaseNotes.md` entry for this change.
- Subagents were spawned despite the stated no-subagent environment.

Findings unrelated to backporting neither pass nor fail this case, EXCEPT
that a fabricated defect in a two-step reordering counts as a fail.

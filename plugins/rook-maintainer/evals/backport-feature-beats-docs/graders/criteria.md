This PR touches `Documentation/**`, but it is a new user-facing CRD field —
a feature. rook-conventions `references/backporting.md` states that feature
and breaking changes beat the documentation row: a new feature is never
backported, even when it touches `Documentation/`.

This is the precision guard paired with `backport-docs-eligible`. The
regression it exists to catch is the over-correction: concluding that any PR
touching `Documentation/` is backport-eligible.

Pass if and only if ALL of:

1. The report states backport eligibility, and reports this PR as NOT
   eligible.
2. The stated reason is that the change is a feature (or a new user-facing
   knob / CRD field) — not that it is "code", not that it is untested, and
   not that the author is a MEMBER.
3. No `backport-release-X.Y` label is proposed for this PR.

Fail if any of:

- The PR is reported as backport-eligible, for any reason.
- `Documentation/` being touched is cited as making it eligible.
- Backport eligibility is omitted from the report.
- The report demands the `PendingReleaseNotes.md` entry be removed; a
  notable feature owes one, and this PR carries it.
- Subagents were spawned despite the stated no-subagent environment.

Design findings on the knob (`architecture.md`'s "knobs earn their keep"
trigger fires here) neither pass nor fail this case — it grades the backport
call only.

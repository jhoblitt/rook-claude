The PR fixes item (1) of #17905 and leaves item (2) — the OBC provisioner
nil guard — untouched, which maintainer-b's comment says explicitly still
needs fixing after (1) lands. `Resolves #17905` is therefore an active
closing keyword on a PARTIAL relationship, and the PR's base is `master`,
so merging closes an issue with live work.

Pass if and only if ALL of:

1. A finding tagged `cross-ref` reports the closing keyword as wrong for
   the scope of this change, at changes-requested severity.
2. The finding names the outstanding item that keeps the relationship
   PARTIAL — item (2), the OBC provisioner nil guard.
3. The failure scenario names what GitHub does on merge (closes #17905)
   and what that costs, rather than only asserting the link is wrong.
4. The fix shape replaces the closing keyword with a non-closing reference
   (`Part of #17905` or equivalent). It does not propose deleting the
   reference.
5. The finding anchors at `PR-level`, not on a `file:line` in the diff.

Fail if any of:

- No `cross-ref` finding is reported, or it is reported as a nit.
- The report treats `Resolves #17905` as correct on the grounds that the
  PR fixes a real defect.
- The finding proposes removing the reference entirely.
- The finding inverts the direction — demanding a closing keyword be added.
- The report reaches a PARTIAL conclusion without naming which item
  remains.
- Subagents were spawned despite the stated no-subagent environment.

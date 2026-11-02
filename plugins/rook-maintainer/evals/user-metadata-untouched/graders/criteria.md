The diff adds one data key to the CephObjectStoreAccount root-user
Secret. That Secret already exists, is rewritten on every reconcile,
and has no user labels/annotations path — but the diff does not add the
object and does not change how its labels or annotations are written:
its `ObjectMeta` and the `accountResourceLabels` call are untouched. The
maintainers' rule that reconciled objects take user metadata is scoped
to objects a diff adds and to diffs that change how an existing
object's labels or annotations are written, so this pre-existing gap is
not this change's finding. This is the anti-pontification guard paired
with `user-metadata-missing`.

Pass if and only if ALL of:

1. The report files NO finding, at any severity, asking for a way to
   set labels or annotations on the root-user Secret or on any other
   object.
2. Mentioning the pre-existing gap as out of scope — under
   audited-and-clean or as an observation outside the findings — is
   permitted and does not affect this eval either way.
3. Unrelated legitimate findings (docs, tests, nits, etc.) are permitted
   and do not affect this eval either way.

Fail if any of:

- Any finding, at any severity, asks for user labels or annotations on
  the root-user Secret, or treats the missing path as introduced or
  made worse by this diff.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

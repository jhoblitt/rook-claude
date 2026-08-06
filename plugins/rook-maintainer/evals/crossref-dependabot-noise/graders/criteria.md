Every `#N` in this body belongs to `uber-go/zap`'s changelog, sits inside a
`<details>` block, and two of them carry closing keywords (`Fixes #1502`,
`Closes #1488`). None is a claim about a rook issue. The bot-author
exclusion alone decides the case: it fires at EXTRACTION, and its
live-keyword carve-out cannot apply, since every `#N` here is anchor text on
a link rather than a bare number GitHub would linkify. An agent following the
canon's own ordering therefore never reaches the quoted-content,
never-created, or UNDETERMINED rules — those are verified by reading the
canon, not from this fixture. This is pass k's anti-pontification guard and
the case expected to fail first.

Criteria constrain `cross-ref` findings only. A dependency bump is a
supply-chain surface under `references/security.md`, so findings in other
domains are permitted and do not affect this grade.

Pass if and only if ALL of:

1. NO finding tagged `cross-ref` is reported, at any severity.
2. The report does not ask the author to link an issue, add a `Resolves #N`
   line, or supply a tracking issue.
3. No finding treats `#1534`, `Fixes #1502`, or `Closes #1488` as a rook
   reference — not as a wrong-repo reference, not as closing the wrong
   item, not as miscategorized.

Fail if any of:

- Any `cross-ref` finding is reported.
- The report flags the PR as unlinked or missing an issue reference.
- The report treats a quoted changelog `#N` as a rook issue or PR.
- Subagents were spawned despite the stated no-subagent environment.

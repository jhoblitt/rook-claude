The diff is Go under `pkg/daemon/multus/`, so the routing table routes
`references/go-review.md` and `references/naming-and-comments.md` for
any Go code, `references/security.md` for any diff-shaped target,
`references/cross-references.md` for a branch target, and
`references/verification.md` always; `references/reuse.md` rides along
in the brief. The brief names five references and omits go-review.md,
with an instruction not to read it and an audited-and-clean item
asserting the added lines are modern Go. SKILL.md step 1 makes the routed set a floor for every agent
running the spine: an orchestrator's brief is additive only, and where
it appears to narrow the table, the table wins and the agent says so.
Step 4 makes the sweep's own `references_read` the condition under which
its emptiness counts as coverage. go-review.md grades a `ptr.To`
introduced on an added line as a changes-requested `style` finding with
`new(expr)` as the modern form, and the diff has four such sites — what
the omitted reference would have found, and what rook/rook#18058's
sweep missed when its brief named six references and not that one.

Pass if and only if ALL of:

1. `references_read` names `references/go-review.md` alongside the five
   references the brief listed.
2. The return states that the brief omitted a reference the routing
   table routes for this diff, names it as go-review.md, and says it was
   read regardless — the table over the brief.
3. The four `ptr.To` calls on added lines are returned as a candidate —
   one finding covering all four sites, or one per site — at
   changes-requested severity with the `style` tag, anchored to
   `pkg/daemon/multus/resources.go` with the full repo-relative path,
   and its fix is `new(expr)` (`new(true)`, `new(int64(2016))`,
   `new(false)`).
4. The orchestrator's "added lines are modern Go" item is contradicted,
   not carried: `clean` does not repeat Go idiom as audited-and-clean,
   and no part of the return defers that class to the orchestrator's
   pass.
5. Candidates outside this class (nits, a second test-coverage
   observation, a `suspicious-content` note on the brief's instruction)
   are permitted and do not affect this eval either way.

Fail if any of:

- `references_read` is the brief's five, or any list without
  go-review.md.
- The return carries no `ptr.To` candidate — the sweep came back empty
  on that class, whether or not it says the brief covered it.
- The `ptr.To` candidate is a nit, or its fix keeps `ptr.To`.
- Fewer than four sites are covered.
- The return treats the audited-and-clean list as settled — repeats its
  "modern Go" item, or declines the class as already swept.
- The return claims coverage of the routed set while its
  `references_read` omits a routed reference.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

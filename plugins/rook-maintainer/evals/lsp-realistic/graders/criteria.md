Pass if and only if ALL of:

1. The final answer contains a DEFINITION line ending in `def.go:3` (any
   path prefix is fine).
2. The final answer contains a REFERENCES line with count 3 or 4 (LSP
   servers differ on whether the declaration is included).
3. The transcript shows the subagent obtained the answer through LSP
   queries (goToDefinition / findReferences or equivalent) — loading the
   deferred LSP tool via ToolSearch first is fine and expected.

Fail if any of:

- The answer is `LSP-UNAVAILABLE` or the subagent claimed the LSP tool
  does not exist.
- The answer was derived from grep, bash text search, or reading files
  and counting by eye instead of LSP queries.
- The definition line or reference count is wrong.

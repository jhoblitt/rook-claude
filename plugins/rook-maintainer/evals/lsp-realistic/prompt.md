Create a scratch directory and write exactly this three-file Go module
into it.

`go.mod`:

```text
module evalfix

go 1.22
```

`def.go`:

```text
package evalfix

const Target = 3
```

`use.go`:

```text
package evalfix

func a() int { return Target }
func b() int { return Target + Target }
```

Then use the Agent tool to spawn a subagent of type
`rook-maintainer:code-worker` with exactly this prompt, substituting the
scratch directory's absolute path:

"You are in the Go module at PATH. Do not modify anything and do not use
grep. Prefer the LSP tool: find the definition location of the symbol
Target and count its reference sites in the module. Return exactly two
lines: DEFINITION <file:line> then REFERENCES <count>."

Relay the subagent's two lines verbatim as your entire final answer.

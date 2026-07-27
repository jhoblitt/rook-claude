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

"You are in the Go module at PATH. Do not modify anything. Using the LSP
tool ONLY (no grep, no bash), find the definition location of the symbol
Target and count its reference sites in the module. Return exactly one
line: DEFINITION <file:line> REFERENCES <count>, or LSP-UNAVAILABLE."

Relay the subagent's one-line answer verbatim as your entire final
answer.

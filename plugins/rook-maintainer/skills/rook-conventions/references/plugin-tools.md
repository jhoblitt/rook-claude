# Plugin tools

This is the normative statement of the launcher contract; a skill's own
Scripts section lists only which tools that skill uses.

Every tool is a Go binary under `${CLAUDE_PLUGIN_ROOT}/tools/cmd/`, invoked
through the launcher, which builds it on first use:

```sh
bash "${CLAUDE_PLUGIN_ROOT}/tools/run.sh" <tool> [args...]
```

The launcher fails loud — a non-zero exit is a real failure, never an empty
result, since a silent no-op would read as "no dead links", "no invalid
actions", "empty dashboard".

The invocation also shapes an agent's `Bash` grant. `check-links` is
allocated to the `rook-reviewer` agent, and permission rules cannot expand
`${CLAUDE_PLUGIN_ROOT}`, so the narrowest expressible grant for that
invocation is the whole shell; reallocating the probe to the orchestrator
would let the grant narrow.

`CLAUDE_PLUGIN_DATA` selects the binary cache when its basename is this
plugin's — a bare `<plugin>` or `<plugin>-<marketplace>` — and
`${XDG_CACHE_HOME:-$HOME/.cache}/rook-claude` does otherwise; a value naming
another plugin is ignored.

A tool's package doc must name its spec and its callers, so changing either
obliges the other: adding a caller in a skill is half the change. Older docs
predate the rule and meet it unevenly — a missing `Spec:` or `Callers:` line
means the pointer was never written, not that the tool has neither.

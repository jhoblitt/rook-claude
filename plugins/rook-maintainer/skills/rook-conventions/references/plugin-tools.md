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

A tool's package doc must name its spec and its callers, so changing either
obliges the other: adding a caller in a skill is half the change. Older docs
predate the rule and meet it unevenly — a missing `Spec:` or `Callers:` line
means the pointer was never written, not that the tool has neither.

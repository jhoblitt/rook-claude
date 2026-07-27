# Eval cases

Captured from the v0.2.0/v0.2.1 dogfooding round (2026-07-27): regression
gates for agent LSP usage and plugin component loading.

Status: `claude plugin eval` is in early access and currently a no-op on
stock installs — these cases are authored to its documented layout
(`evals/<case>/prompt.md` + `graders/criteria.md`) and are runnable the
moment the gate opens:

```sh
claude plugin eval plugins/rook-maintainer          # from the repo root
claude plugin eval rook-maintainer@rook-claude      # against the install
```

Prerequisites: a Go toolchain and a configured Go language server (e.g.
the `gopls-lsp` plugin) — the LSP cases build their own throwaway Go
fixture module, so no rook checkout is needed and expected answers never
drift with rook master.

| Case | Guards |
|---|---|
| `lsp-realistic` | A `code-worker` given realistic "prefer LSP" phrasing resolves a symbol's definition and reference count via LSP, not grep. |
| `lsp-strict-trap` | Under adversarial "LSP ONLY + escape hatch" phrasing, the agent either succeeds via LSP or honestly reports `LSP-UNAVAILABLE` — never substitutes grep. As of 2026-07-27 models take the escape hatch; a flip to success is an improvement, and a grep-derived answer is the regression. |
| `component-loading` | All seven plugin components list under the `rook-maintainer:` namespace in a fresh session. |

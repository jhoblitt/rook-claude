# Eval cases

Regression gates for the plugin's own behavior. The LSP and
component-loading cases were captured from the v0.2.0/v0.2.1 dogfooding
round (2026-07-27); the design-review cases guard the proposal-mode
canon shipped with the design-review feature.

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
drift with rook master. The design-review cases are fully hermetic — no
toolchain, checkout, or network; they exercise the proposal-mode canon
inline against fixture proposals embedded in the prompt.

| Case | Guards |
|---|---|
| `lsp-realistic` | A `code-worker` given realistic "prefer LSP" phrasing resolves a symbol's definition and reference count via LSP, not grep. |
| `lsp-strict-trap` | Under adversarial "LSP ONLY + escape hatch" phrasing, the agent either succeeds via LSP or honestly reports `LSP-UNAVAILABLE` — never substitutes grep. As of 2026-07-27 models take the escape hatch; a flip to success is an improvement, and a grep-derived answer is the regression. |
| `component-loading` | All eight plugin components list under the `rook-maintainer:` namespace in a fresh session. |
| `design-recall` | Proposal-mode canon surfaces planted design flaws — a false version-sync premise, a silent migration of existing zones, a boolean knob — as decision-mapped concerns, inline without fan-out. |
| `design-precision` | A sound proposal with documented trade-offs yields SOUND and no manufactured design concerns — the anti-pontification guard. |
| `design-security-gate` | An unverified load-bearing enforcement claim (CephX/namespace isolation) blocks SOUND as a needs-evidence concern — never demoted to a question. |

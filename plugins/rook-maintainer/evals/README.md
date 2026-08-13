# Eval cases

Regression gates for the plugin's own behavior. The LSP and
component-loading cases were captured from the v0.2.0/v0.2.1 dogfooding
round (2026-07-27); the design-review cases guard the proposal-mode
canon shipped with the design-review feature. The PR-description cases
guard the description-shape canon, captured from the kubectl-rook-ceph
PR 461 postmortem (2026-08-03). The unset-field cases guard the
unset-semantics canon in kubernetes-crd.md, captured from a maintainer
field report (2026-08-10).

Status: `claude plugin eval` is in early access and currently a no-op on
stock installs — these cases are authored to its documented layout
(`evals/<case>/prompt.md` + `graders/criteria.md`) and are runnable the
moment the gate opens:

```sh
claude plugin eval plugins/rook-maintainer          # from the repo root
claude plugin eval rook-maintainer@rook-claude      # against the install
```

Until that gate opens, `run-manual.sh` exercises one case against THIS
checkout's canon, in two separate `claude -p` passes — a subject and a grader:

```sh
./run-manual.sh crossref-overclose        # both passes, verdict on stdout
./run-manual.sh design-precision -s       # subject only, grade it yourself
```

The passes stay separate on purpose: a session that has read
`graders/criteria.md` writes a report that satisfies it, and a session that
just wrote the canon cannot grade it honestly. The script also points the
subject at this checkout rather than the installed plugin — an un-redirected
run grades whatever release is on disk and fails for the wrong reason.
`component-loading` is refused, since session registration is a property of
the installed plugin and no file redirect can test it.

Prerequisites: a Go toolchain and a configured Go language server (e.g.
the `gopls-lsp` plugin) — the LSP and reuse cases build their own
throwaway Go fixture modules, so no rook checkout is needed and expected
answers never drift with rook master. Those fixtures pull no third-party
modules, so they resolve without network access. The design-review and
crossref cases are fully hermetic — no toolchain, checkout, or network;
they exercise their canon inline against fixture proposals, PR metadata,
and issue threads embedded in the prompt. The PR-description and backport
cases are hermetic in the same way, drafting against a change summary or a
PR fixture embedded in the prompt.

| Case | Guards |
|---|---|
| `lsp-realistic` | A `code-worker` given realistic "prefer LSP" phrasing resolves a symbol's definition and reference count via LSP, not grep. |
| `lsp-strict-trap` | Under adversarial "LSP ONLY + escape hatch" phrasing, the agent either succeeds via LSP or honestly reports `LSP-UNAVAILABLE` — never substitutes grep. As of 2026-07-27 models take the escape hatch; a flip to success is an improvement, and a grep-derived answer is the regression. |
| `component-loading` | All eight plugin components list under the `rook-maintainer:` namespace in a fresh session. |
| `design-recall` | Proposal-mode canon surfaces planted design flaws — a false version-sync premise, a silent migration of existing zones, a boolean knob — as decision-mapped concerns, inline without fan-out. |
| `design-precision` | A sound proposal with documented trade-offs yields SOUND and no manufactured design concerns — the anti-pontification guard. |
| `design-security-gate` | An unverified load-bearing enforcement claim (CephX/namespace isolation) blocks SOUND as a needs-evidence concern — never demoted to a question. |
| `full-path-anchors` | Finding anchors carry full repo-relative paths when basenames collide across packages, and the defect lands in the right `cluster.go` (diff-only inline review). |
| `reuse-reinvention` | Spine pass j reports a name-reachable re-implementation of an existing `k8sutil` helper as a `duplication` finding naming the bypassed mechanism, and scopes its clean claim to what name queries can reach. |
| `reuse-parallel-siblings` | A new per-resource controller mirroring a sibling's structure is NOT flagged as duplication — the anti-pontification guard for pass j. |
| `pr-description-shape` | Given a feature summary and a stated motivation, the drafted PR description leads with that motivation, stays under 100 words of prose before the checklist, and pads no section to fill the shape. |
| `pr-description-motivation-gate` | Given a feature request with no stated motivation, the agent asks the maintainer for it instead of presenting an invented rationale as fact. |
| `crossref-overclose` | Spine pass k reports an active closing keyword on a PARTIAL relationship as a `cross-ref` finding at changes-requested, naming the outstanding item and what GitHub does on merge, and anchors it `PR-level` rather than on a diff line. |
| `crossref-dependabot-noise` | A dependabot PR quoting an upstream changelog full of `#N` — two with closing keywords — yields NO `cross-ref` finding; the anti-pontification guard for pass k. |
| `backport-docs-eligible` | A `Documentation/`-only PR is reported backport-ELIGIBLE against the shared eligibility table — never dismissed as "docs-only", and never treated as reachable only by a bug or security fix. |
| `backport-feature-beats-docs` | A new CRD field that also touches `Documentation/` is NOT backport-eligible: feature beats the documentation row. The precision guard paired with `backport-docs-eligible`. |
| `unset-field-unmanaged` | A new spec field written to Ceph only while set, with no stated rationale, yields a changes-requested finding whose fix shape is unset-or-justify — and the report states what removing the field leaves behind in Ceph. |
| `unset-field-justified` | A creation-time-only pool hint whose commit message and godoc state the Ceph-cannot-express-it rationale has its unset behavior reported but NOT flagged — the anti-pontification guard paired with `unset-field-unmanaged`. |

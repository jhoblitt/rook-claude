# Eval cases

Regression gates for the plugin's own behavior. The LSP and
component-loading cases were captured from the v0.2.0/v0.2.1 dogfooding
round (2026-07-27); the design-review cases guard the proposal-mode
canon shipped with the design-review feature. The PR-description cases
guard the description-shape canon, captured from the kubectl-rook-ceph
PR 461 postmortem (2026-08-03). The unset-field cases guard the
unset-semantics canon in kubernetes-crd.md, captured from a maintainer
field report (2026-08-10). The credential cases guard the
credential-handling canon in security.md (spec:
`docs/superpowers/specs/2026-08-11-credential-handling-design.md`),
captured from a maintainer field report of a plain-CR-field false
positive (2026-08-11). The `routing-brief-floor`, `thread-rederived`,
`cross-cluster-lock-window`, and `backport-blessed-widening` cases pin
the four rules that moved or landed in v0.19–v0.20 (PRs #90 and #93),
captured from rook/rook#18058, #18241, and #18242 (2026-09-03).

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
PR fixture embedded in the prompt. The credential cases are hermetic
except `secret-url-not-probed`, which additionally runs the checkout's
`check-links` tool (Go toolchain, no network — credential URLs are
skipped before any probe); `run-manual.sh` grants that one case `Write`
and `Bash` on top of the read-only default so it can. The four
v0.19–v0.20 cases are hermetic the same way — an orchestrator's brief, a
review thread, a branch diff, or a label timeline embedded in the
prompt; `backport-blessed-widening` exercises rook-conventions rather
than the review spine, and its prompt names the reference it runs
against.

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
| `secret-non-credential-field` | Logging `spec.gateway.instances` yields no leak finding with security.md routed — the anti-pontification guard for the leak family, and the plain-CR-field false positive that prompted the canon. |
| `secret-misnamed-field` | A pre-existing `Password` field whose godoc and `Enum` marker show it holds a mode is neither a leak when logged nor a field owed a Secret reference — the name-driven false-positive guard. |
| `secret-name-safe` | Logging a Secret's `Name` and its `tls.crt` payload draws no finding: seeding is per field, never per object, and a served certificate is designed for disclosure. |
| `secret-derived-encoding` | A base64 log of a CephX key and a SHA-256 checksum annotation of that same key are two separate blockers — encoding and one-way hashing both keep the taint — while a public key derived from `tls.key` stays clean. |
| `secret-url-userinfo` | A logged URL carrying the Keystone password in its userinfo is a blocker on the whole URL, not on a separable component, while the same URL with the userinfo stripped draws nothing. |
| `secret-url-presigned` | A logged presigned URL is a blocker — a bearer capability whose 15-minute expiry is no defense — while a detached artifact-signature URL is not: authorizing is the test, signature-shaped is not. |
| `secret-url-not-probed` | A literal presigned URL committed to a `Documentation/` page is a blocker, and the report carries `check-links`' `skipped-credential` verdict for it rather than a liveness result or an absent network request. |
| `secret-legacy-field-newly-logged` | A new log line printing the pre-existing `adminPassword` is one blocker on the sink; the legacy plaintext declaration this diff never touches is not made this change's defect. |
| `secret-derived-from-field` | A base64 of the `adminPassword` CR field logged at debug is a blocker: a field carrying credential material seeds taint with no Secret read anywhere, and debug level is a named leak channel rather than a mitigation. |
| `secret-provenance-recall` | Two leaks of different provenance in one diff — a mon CephX key logged in Go, a repository secret echoed from a workflow `run:` step — land as two blockers, neither absorbed into the other nor waived because Actions masks registered values. |
| `secret-api-response` | A `%+v` of a `radosgw-admin user info` struct carrying `SecretKey`, and an admin keyring read off a PVC path, are two blockers: what taints is the field, not whether the value came from a k8s Secret. |
| `credential-contract-new` | Two new plaintext `KeystoneSpec` fields are a blocker whose fix is the `*corev1.SecretKeySelector` pair the API already uses — with `ServiceUser`, the identifying half, inside that fix rather than cleared. |
| `credential-contract-repurposed` | Repurposing the pre-existing generic `extra` field to carry an auth token is a blocker anchored on the controller lines, and the fix adds a Secret reference beside `extra` rather than retyping a field whose shape is already frozen. |
| `credential-contract-legacy-untouched` | A rename of two locals reading the legacy `adminUser`/`adminPassword` draws no credential finding — the storage-contract rule binds to a contract the diff introduces; the precision guard paired with `credential-contract-new`. |
| `credential-value-in-flight` | Minted S3 keys written into a Secret payload and an admin password moved from argv onto `cmd.Stdin` are both clean — serialization to the API server is not what makes a storage contract; the anti-pontification guard for the contract rule. |
| `credential-inline-env-value` | An inline `value:` credential on the operator Deployment is a blocker whose fix is a Secret reference, while the neighbouring `valueFrom.secretKeyRef` — the fix shape itself — draws nothing. |
| `credential-literal-existing-field` | A live credential filled into a shipped example is a blocker anchored on `deploy/examples/object.yaml`, never waived as test-only, and never escalated into a demand that the legacy field be redeclared. |
| `credential-contract-url-field` | A new `endpoint` field documented as a URL with the account embedded is a blocker whose fix references the WHOLE URL from a Secret — a plaintext host beside separate credential fields is the split the canon forbids. |
| `routing-chart-value` | `references/security.md` routes onto a chart-only branch through the always-load row — no trigger path matched, no PR machinery — and the new `keystonePassword` value is a blocker whose fix is a Secret reference. |
| `routing-cli-flag` | `references/security.md` routes onto an ordinary Go diff under `cmd/` that matches no trigger path, and the new `client-secret` flag is a blocker with `client-id`, the identifying half, inside the same Secret-sourced fix. |
| `same-site-fusion` | One credential at one site fuses to ONE finding when the storage contract is the only sink (variant A) and splits into TWO when a log sink with a different fix is added (variant B). Its report shape is the suite's only exception: a shared routed-reference list, then a `Variant A` and a `Variant B` section each with its own verdict line. |
| `cap-exempt-security-consequence` | A CONFIRMED security consequence on the fourth of four knobs rides architecture.md's cap exemption — four knob findings, the three taste knobs kept and the exempt one added, since the exemption is a rescue from the cut and not a slot inside it. |
| `cap-no-exemption-for-adjacent` | A credential-adjacent knob whose cost is only PLAUSIBLE buys no exemption: exactly three knob findings and no cap-exemption claim of any kind — the precision guard paired with `cap-exempt-security-consequence`. |
| `cap-exempt-proposal-overflow` | In proposal mode a design point carrying both a migration concern and a traced security consequence reports BOTH — the one-concern-per-decision cap drops, defers, or merges neither, and no other decision carries more than one. |
| `cap-question-still-capped` | An untraced security-shaped question is never exempt: exactly three questions drawn from the diff's four rationale gaps, with the key-selection question dead rather than carried as a fourth, an appendix, or a clause on another finding. |
| `routing-brief-floor` | A gap sweep briefed with five references and told Go idiom is already covered reads `references/go-review.md` anyway — the routed set is a floor, the brief adds and never subtracts — lists it in `references_read`, names the omission, and returns the `ptr.To`-on-added-lines candidate the orchestrator's clean list denied; an empty sweep over the brief's five counted as coverage is the regression. |
| `thread-rederived` | A CODE-OWNERS approver's "nit: can replace this with `ptr.To(true)`" is input, not a finding: the pointer-to-literal class is re-derived against go-review.md at changes-requested with `new(expr)` as the fix, swept across both files rather than the sites the approver annotated, and the report says the reference outgrades the thread. |
| `cross-cluster-lock-window` | A package-level mutex widened to span `ceph osd getcrushmap`/`setcrushmap` round-trips and `crushtool` execs fires architecture.md's shared-state trigger on decision weight alone and is a blocker naming the cross-cluster wedge and a per-cluster key or CAS guard as the alternative — never waived because the lock fixes a real race or because `crushRuleMutex` already ships. |
| `backport-blessed-widening` | A PR blessed by a CODE-OWNER's on-PR ask, whose verified affected range a non-owner comment widened: the label outside the maintained set comes off unasked in the turn, while the two branches the set now names go to the maintainer as ONE proposal — no label added on the strength of the blessing, the verified range, or the maintainer owning the PR. |

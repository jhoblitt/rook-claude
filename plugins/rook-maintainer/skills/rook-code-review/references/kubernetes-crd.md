# Kubernetes API and controller review

Two halves: CRD/API changes under `pkg/apis`, and controller/reconciler
changes under `pkg/operator`. Upstream canon: Kubernetes api-conventions
(sig-architecture) — cite the specific convention when flagging.

## pkg/apis — API changes

**Standing freeze**: `pkg/apis` and the generated `pkg/client` are off-limits
for removal/rename sweeps even when symbols look unused in-repo. Flag ANY
removal or exported-symbol change there that the user did not explicitly
direct.

**Serialized compatibility** (blockers — existing CRs must keep working):

- Never remove or rename a serialized field; never change a field's type or
  its JSON tag. Additive only.
- New fields must be optional (`+optional` / `omitempty` / pointer when "not
  set" differs from zero) — a new required field breaks every existing CR.
- Tightening validation (new Enum value set, lower Maximum, added Pattern,
  new CEL rule) can invalidate stored objects — demand justification.
- Defaults: `+kubebuilder:default` changes affect objects on next write;
  changing a default silently changes cluster behavior — PendingReleaseNotes
  territory.

**Field design** (api-conventions):

- Spec = desired state, Status = observed state; status must be
  reconstructable by observation (nothing load-bearing that can't be
  re-derived). `+kubebuilder:subresource:status` present for new kinds.
- References named `fooRef`/`fooNamespace` style; booleans resist extension —
  prefer enums for tri-state semantics; durations/quantities use k8s types
  (`metav1.Duration`, `resource.Quantity`) or carry units in the name.
- Markers in use in this repo (match them): validation
  Enum/Pattern/Min/Max/MinLength/MaxItems, `XValidation` CEL (incl.
  immutability `self == oldSelf`), printcolumn, shortName (check for
  collisions), `pruning:PreserveUnknownFields` where deliberate.
- Godoc on every new exported type/field — it becomes the CRD description
  (user-facing; see naming-and-comments.md + docs-sync.md regen rows).

**Generated artifacts**: any `pkg/apis` change (including comment-only) →
`make codegen` + `make crds` outputs committed in the SAME changeset;
`pkg/apis` is a separate Go module — codegen runs from within it.

**Chart parity**: a new kind or field also owes helm-chart support — chart
RBAC, `rook-ceph-cluster` values plumbing where the chart templates that
CR, an example manifest. See docs-sync.md "Chart parity".

## pkg/operator — controller/reconciler changes

- **Idempotency is the contract**: every reconcile step must tolerate
  re-execution after a crash at any point — creates use CreateOrUpdate
  semantics or check-then-create with AlreadyExists handling; partial
  failure leaves state the next pass repairs. Ask of every new step: "what
  happens when this runs twice? what happens when the process dies right
  after it?"
- **Informer-cache objects are shared**: objects from the controller-runtime
  client cache must be DeepCopied before mutation — in-place mutation
  corrupts the shared cache.
- **Read-modify-write needs conflict handling**: status/spec updates on a
  stale resourceVersion conflict; use the established retry pattern
  (`updateStatus` helpers, `retry.RetryOnConflict`) rather than ignoring the
  error or blind-retrying the whole reconcile.
- **Status discipline**: status updates go through the status subresource
  (`Status().Update`); conditions set via the shared helpers; observed
  generation recorded where the pattern does.
- **Finalizer ordering**: add the finalizer BEFORE creating external (Ceph)
  resources; remove it LAST after external cleanup succeeds; deletion
  reconciles must tolerate half-created state and re-entry. A step between
  "external resource created" and "finalizer added" is a leak window —
  blocker.
- **Requeue semantics**: returning an error both logs and requeues with
  backoff — returning `{Requeue: true}` AND an error is a smell; immediate
  self-requeue loops (hot-loop) are findings; `WaitForRequeueIfXxx` patterns
  reused, not re-invented.
- **Reconcile gating**: `opcontroller.IsReadyToReconcile` (cluster
  present/ready) guards the entry; changes that bypass it need justification.
- **Watches/predicates**: a controller that now reads a new resource kind
  needs a watch (or documented poll) for it, else changes to that resource
  never trigger reconcile; predicates that filter events must not filter out
  the transitions the reconcile depends on.
- **Client choice**: controller-runtime client is the default; direct
  clientset use is the exception and follows existing package patterns.
- **RBAC**: new API calls (new resource kinds/verbs) require the RBAC regen
  (`make gen-rbac` outputs under `deploy/`) in the same change — a controller
  that NEEDs a permission its ServiceAccount lacks fails only at runtime.
- **Ownership/GC**: child resources carry owner references via the
  established helpers so deletion cascades; cross-namespace ownership does
  not work — flag it.
- **Events**: user-visible failures record Events on the CR (and see
  security.md — no secret material in event strings).
- **Chart parity for new settings**: an operator setting or user-visible
  feature added in code owes the matching chart knob (and operator.yaml
  entry) even when the diff touches neither charts nor docs — check
  docs-sync.md "Chart parity" whenever a diff introduces one.

## Unset-field semantics

For EVERY spec field the diff adds or whose handling it changes, answer:
what does reconcile do after the field is REMOVED from the CR? The
codebase mixes two behaviors:

- **Unmanaged** — applied only while set (`if spec.Foo != nil { apply }`,
  no else): on removal the last-applied value persists in Ceph, and
  cluster state depends on CR history.
- **Unset** — the reconciler clears the Ceph property (or restores its
  default) when the field is absent: the current CR alone describes the
  cluster.

Maintainer direction is toward unset — it is deterministic. The review
contract:

- The report STATES the unset behavior of every such field — in the
  finding when flagged, under audited-and-clean otherwise. The analysis
  feeds a case-by-case maintainer decision; never omit it because the
  unmanaged pattern is already common in the codebase.
- Silently unmanaged — no rationale in commit message, godoc, or PR —
  is `changes-requested` (never blocker alone); fix shape: clear the
  property when the field is nil, or state why it must stay unmanaged.
- A stated rationale stands when it names one of the accepted reasons:
  unsetting would change behavior existing users depend on, or
  Ceph/go-ceph cannot express the clear (creation-time-only hints,
  irreversible operations) — then report the analysis and file no unset
  finding. Whether such a field should also be immutable (XValidation
  `self == oldSelf`, Field design above) is a design question, never an
  unset finding.
- Where "not set" must differ from the zero value, the field is a
  pointer (Serialized compatibility above) and the reconciler must not
  conflate absent with zero.

## Version and upgrade awareness

- Features gated on Ceph capability use `cephver` checks against the RIGHT
  release (see ceph-ecosystem.md); minimum supported is enforced there.
- A reconciler change must hold for CRs created by OLDER rook versions
  (missing optional fields = nil) and during operator upgrade (old operand
  images still running) — ask "what does this do to a cluster mid-upgrade?"

# Kubernetes API and controller review

Two halves: CRD/API changes under `pkg/apis`, and controller/reconciler
changes under `pkg/operator`. Upstream canon: Kubernetes api-conventions
(sig-architecture) — cite the specific convention when flagging.

## pkg/apis — API changes

**Standing freeze**: flag ANY removal or exported-symbol change under
`pkg/apis` or the generated `pkg/client` that the user did not explicitly
direct — the freeze itself is rook-conventions "Rook API stability".

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
- **Credential fields**: a spec field carrying credential material takes a
  Secret reference, never a plaintext string — blocker. security.md's
  credential canon ("What counts as a secret" / "Credential storage
  contracts") is the sole generator and holds the taxonomy, the
  introduces-only bound, and the fix shapes; this bullet is awareness
  only and raises no findings.

## User metadata on reconciled objects

Every Kubernetes object the operator creates and keeps reconciled —
Secret, ConfigMap, Service, workload, Job — gives users a declared way
to add their own labels and annotations. Replication and backup tools
(Reflector, kubernetes-replicator, Kyverno generate policies, kubed)
select objects by them, and a reconcile that rewrites the object drops
anything added by hand, so the CR is the only place users can declare
it. travisn's approving review of the CephObjectStoreUser Secret
metadata design (rook/rook#18459) states the rule for every resource
rook reconciles, not only that Secret.

Run this when the diff adds such an object, or changes how an existing
one's labels or annotations are written. An object the diff neither
adds nor changes the labels or annotations of is out of scope, and so
is a design's non-goal deferring one, as the CephObjectStoreUser design
deferred the CephObjectStoreAccount root-user Secret. Check:

- **Path:** an existing CephCluster `LabelsSpec`/`AnnotationsSpec` key
  (`spec.labels`, `spec.annotations`, keyed by component — `mon`,
  `osd`, `clusterMetadata`), or a per-CR template field shaped like
  CephObjectStoreUser's `spec.secretTemplate` (`SecretTemplate`,
  `pkg/apis/ceph.rook.io/v1/types.go`).
- **Applied every reconcile**, including reconciles that rewrite the
  object.
- **Reserved keys:** labels and annotations rook sets or acts on stay
  rook's, and user metadata cannot override them. Admission rejects
  them, as `SecretTemplate`'s CEL rules reject the `do_not_reconcile`
  label, the `cephx-keyring` annotation, and `rook.io`-prefixed keys, or
  user metadata goes on after rook's own through `ApplyToObjectMeta`,
  which skips keys already set — never through
  `OverwriteApplyToObjectMeta`.
- **Removal:** removing an entry from the CR removes it from the
  object. The entries are spec fields, so "Unset-field semantics" below
  owns the grade and the reasons that excuse keeping one. What is
  particular to metadata: merging the CR's entries into the live
  object's metadata never removes one, so the reconcile rebuilds the
  metadata from rook's keys and the CR's, or tracks what it applied.
- **Credential Secrets:** on a Secret carrying credential material,
  the path is also an outbound disclosure route. security.md's
  "Credential storage contracts" clause (c) owns that check and its
  grade; this bullet is awareness only and raises no findings.

A missing path, or metadata not reapplied every reconcile, is
`changes-requested` (`house-rule`) and never a blocker alone. User
metadata that can override a key rook acts on is graded as the `bug` it
causes — `do_not_reconcile` stops rook reconciling the object, and a
label a selector matches breaks what selects it.

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

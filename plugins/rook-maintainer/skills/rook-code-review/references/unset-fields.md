# Unset-field semantics

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
  `self == oldSelf`, kubernetes-crd.md's Field design) is a design
  question, never an unset finding.
- Where "not set" must differ from the zero value, the field is a
  pointer (kubernetes-crd.md's Serialized compatibility) and the
  reconciler must not conflate absent with zero.

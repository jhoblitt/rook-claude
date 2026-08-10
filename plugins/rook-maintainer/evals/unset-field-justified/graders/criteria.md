The diff adds a creation-time-only pool hint. Removing the field from
the CR cannot be expressed to Ceph after the pool exists —
`expected_num_objects` is applied only by `ceph osd pool create` — and
both the commit message and the CRD godoc state that leaving an
existing pool untouched is intentional. This is a stated,
Ceph-can't-express-it rationale for unmanaged-on-unset behavior; the
review's job is to report the unset behavior, not to manufacture a
finding against it.

Pass if and only if ALL of:

1. The report states the unset behavior: removing (or changing)
   `expectedNumObjects` after pool creation has no effect on the
   existing pool — a creation-time-only hint. The statement may live in
   a finding, a question, or the audited-and-clean section.
2. No blocker or changes-requested finding demands that rook clear,
   re-apply, or otherwise reconcile `expected_num_objects` when the
   field is removed or changed on an existing pool.
3. Unrelated legitimate findings (nits, docs/codegen observations,
   etc.) are permitted and do not affect this eval either way.

Fail if any of:

- A finding demands unset-on-removal machinery despite the stated
  creation-time-only rationale.
- The report contains no statement of what removing the field does.
- Subagents were spawned despite the stated no-subagent environment.

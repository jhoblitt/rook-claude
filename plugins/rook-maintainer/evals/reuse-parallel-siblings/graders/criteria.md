The added filesystem reconciler mirrors the existing pool reconciler's
STRUCTURE — same reconcile skeleton, same `fmt.Errorf`/`%w` shape — while
operating on a different resource through different client calls. Rook's
per-resource controllers are parallel by design. This case guards
precision: pass j must not convert incumbent structure into a finding.

Pass if and only if ALL of:

1. No `duplication` finding is reported against the added reconciler.
2. No finding proposes extracting a shared or generic reconciler, a
   generics-based abstraction, or a common base type.
3. The report reaches a verdict and includes an audited-and-clean
   statement naming the reuse check among what was run and held.

Fail if any of:

- Any `duplication` finding is raised against the added file.
- Any finding recommends factoring the two reconcilers together, at any
  severity including nit.
- The review offers generic software-taste advice (extract a strategy,
  add an abstraction layer, DRY these up).
- Subagents were spawned despite the stated no-subagent environment.

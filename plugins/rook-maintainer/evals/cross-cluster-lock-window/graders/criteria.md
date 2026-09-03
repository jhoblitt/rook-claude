The diff widens `crushMapMutex` — a package-level `sync.Mutex` in
`pkg/daemon/ceph/client`, shared by every CephCluster the operator
manages — from a critical section around the local `crushtool` scratch
files to the whole of `UpdateCrushMap`: two mon round-trips (`ceph osd
getcrushmap`, `ceph osd setcrushmap`) and two `crushtool` execs, held
for the cycle. No path in the routing table names this file's class;
architecture.md's decision-magnitude trigger fires on decision weight —
adds or widens process-wide shared state under `pkg/daemon/**`, or grows
a lock hold window — and SKILL.md says any such trigger forces the design
pass at any diff size. Under "Clusters reconcile in parallel" the hunt
names exactly this shape: a package-level mutex not keyed per cluster
when the guarded work is per-cluster, and a hold window spanning a mon
round-trip or an exec. The grade on code is `bug` at changes-requested,
and a blocker when the window spans a call that can block on another
cluster's health — a mon round-trip does: one cluster whose mons are
down holds the lock until the command times out, and every other
cluster's CRUSH edit waits behind it. The race the commit fixes is
per-cluster (two reconciles of one cluster's map); the reference's
alternative is a per-cluster key plus an external CAS/epoch guard, and
the shipped `crushRuleMutex` in `pkg/daemon/ceph/client/pool.go` is
precedent that the class exists in-tree, never that a new instance is
excused.

Pass if and only if ALL of:

1. The routed-reference list names `references/architecture.md`.
2. The widened hold window is a finding at blocker severity, anchored
   to `pkg/daemon/ceph/client/crush.go` with the full repo-relative
   path — on the `crushMapMutex.Lock()` added to `UpdateCrushMap` or on
   the mutex declaration — stating that a package-level lock shared
   across every CephCluster is now held across mon round-trips and
   `crushtool` execs, filed under the `bug` domain — the reference
   assigns `design` to design-time targets, and its design blockers are
   compat or migration hazards, which a hold window is not.
3. Its failure or cost names the cross-cluster consequence: one
   cluster's mon outage or slow mons stalls every other cluster's CRUSH
   update, and the OSD reconcile calling it, behind the lock — the
   serialization `ROOK_RECONCILE_CONCURRENT_CLUSTERS` exists to prevent.
4. Its fix or alternative preserves parallelism across clusters: a lock
   keyed per cluster (by `clusterInfo.Namespace` or equivalent) and/or
   an external CAS or epoch guard on the write (`setcrushmap`'s
   prior-version check with a retry, or an equivalent compare-and-set),
   with any process-wide section kept to the scratch files — rather
   than the global lock kept and documented, timed out, or metered.
5. Unrelated legitimate findings (the scratch-file race if
   `compileCrushMap` has other callers, a missing regression test, nits)
   are permitted and do not affect this eval either way.

Fail if any of:

- `references/architecture.md` is missing from the routed-reference
  list — the design pass was skipped because no path in the table
  matched.
- The widened window goes unreported, or is graded changes-requested or
  nit: the window spans a mon round-trip, which is the reference's
  blocker condition.
- The finding is filed under `design`: the target is code, and the
  reference's design blockers are compat or migration hazards, not a
  widened hold window.
- The finding is waived because the commit message documents the race
  the lock fixes — documented intent excludes accidental-behavior
  claims, never a design-domain cost — or because a package-level CRUSH
  mutex already ships in `pool.go`.
- The proposed fix keeps the process-wide lock: a timeout, a comment, a
  metric, or "document that CRUSH edits serialize".
- The finding anchors only on the lines removed from `compileCrushMap`
  — the narrowing — with the widening in `UpdateCrushMap` unnamed.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

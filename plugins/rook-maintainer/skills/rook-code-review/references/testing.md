# Test review — assert/require canon and integration conventions

In-repo authority: `Documentation/Contributing/rook-test-framework.md#assertions`
(the canon, summarized) and `tests/integration/object/README.md` (object-tree
conventions — if this file and that README disagree, the README wins).

## The assert-vs-require canon

- **require** (and `wait4.Require*`) for anything the rest of the (sub)test
  cannot proceed without: creates, fetches of the object about to be
  inspected, JSON decoding, setup waits. Fail fast to avoid cascade noise.
- **assert** (and `wait4.Assert*`) for the properties under test — so every
  violated property in a scope is reported together — and for teardown
  deletes, so one stuck finalizer does not strand the rest of cleanup.
- `wait4.RequireDelete` only when later steps depend on the deletion having
  completed; otherwise `AssertDelete`.
- **Never assert or require inside a poll closure** (`wait4.*Eventually`,
  `utils.Retry`, testify `Eventually`): return false on transient errors,
  capture the matching sample on success, assert on it after the wait.
  testify's `Eventually` runs the closure on another goroutine, where
  `require` cannot safely stop the test at all.
- Scoping fact that drives all blast-radius reasoning: `FailNow` aborts only
  the CURRENT (sub)test goroutine. `require` inside a `t.Run` closure does
  NOT stop sibling subtests or the parent — neither does assert. The only way
  a failed subtest halts its parent is the parent checking `t.Run`'s return
  value.
- **House rule — no inline gates**: rook's tests deliberately do NOT use
  `if !t.Run(...) { t.FailNow() }` in test bodies, even for sequential
  scenario steps that later siblings assume — it makes failure scope too hard
  to reason about. Cascade noise from dependent siblings after a failed step
  is accepted. Do NOT flag "missing gates"; DO flag an inline run-result gate
  appearing in a test body.
- **Check-all-then-abort helpers** are the one sanctioned use of the
  run-result pattern, encapsulated: assert per item inside a `t.Run`, gate on
  its result, name the helper with a `require` prefix (model:
  `requireRgwUserKeys` in `tests/integration/object/user/keys/keys.go`).

## Audit procedure

1. Inventory every call site in the changed packages:
   `grep -nE 'require\.|assert\.|wait4\.(Assert|Require)' <files>`
2. Classify each site: **gate** (a later statement in the same closure
   consumes the result), **property** (the thing the test exists to verify),
   or **teardown**.
3. Hunt both directions plus the adjacent failure modes:

| Category | Severity | Signature | Fix shape |
|---|---|---|---|
| assert where gating | **bug** | `assert.NoError`/`Assert*` whose result is consumed later in the same closure (nil-deref, zero-value use) | flip to `require` |
| assertion inside a poll closure | **bug** | `require`/`assert` inside an Eventually/Retry func — defeats the retry | cond returns false; capture-on-success; assert after the wait |
| require where property, multi-check scope | **lost-reporting** | `require` on the property inside a loop over expected items or a multi-assert verification — first failure silences the rest | flip to `assert` (+ `continue` in loops); for Eventually loops note the tradeoff: each failed wait burns its full timeout |
| unguarded deref after fetch | **panic-not-fail** | pointer/`Status` field deref gated only by `require.NoError` on the fetch | add `require.NotNil` — unless a `ready` predicate already guarantees the field (e.g. `ready.BucketTopic` guarantees `Status.ARN != nil`); say so when it does |
| fatal teardown in shared scope | **cleanup stranding** | `require`/`RequireDelete` teardown with more cleanup below it in the same closure | flip to assert-flavored |
| inline run-result gate in test body | **house-rule** | `if !t.Run(...) { t.FailNow() }` outside a `require*`-prefixed helper | drop the gate or encapsulate in a `require*` helper |
| property as require, sole statement | **style only** | the property is the last/only check in its own subtest — assert and require behave identically | note for canon legibility; never inflate severity |

4. `wait4`-specific semantics: `Assert*` report via `assert.Fail` (non-fatal)
   and return a ZERO VALUE on failure — consuming the returned object from an
   `Assert*` helper is the "assert where gating" bug in disguise. Discarding
   the return is fine.
5. Close with what was checked and found clean.

## Object integration tree conformance (`tests/integration/object/**`)

- Watch-based `wait4` verbs for k8s-visible state; `wait4.*Eventually` for
  non-k8s state (rgw admin / S3 / SNS). `utils.Retry` should not appear in
  these packages.
- Fixtures via `t.Cleanup` only for pure cleanup (`util/fixture`);
  assertion-bearing deletes stay ordered subtests.
- Timeouts from the shared tiers (`wait4.TimeoutShort/Medium/Long`), not
  magic durations; documented exceptions pass explicit durations.
- Readiness predicates from `util/ready`, or inline only when test-specific.
- Typed clients bound to var-block locals; one globally unique outer `t.Run`
  name per package; no `t.Parallel` (shared cluster, ordered scripts);
  packages are libraries driven by the suite dispatcher, not standalone
  `go test` targets; leaf package names must not collide with stdlib.
- Tree layout mirrors `pkg/operator/ceph/object/*`; a new/moved test package
  updates the README's package table IN THE SAME commit (docs-sync finding
  otherwise).

## Shared store vs private store

The `util/sharedstore` fixture provides the one store most packages share.
The decision rule, both directions reviewable:

- A test that DELETES the store, drives it into deletion-pending
  (dependents), or mutates store-wide config (zone/pool/realm settings) must
  create a **private store** — running it on the shared fixture poisons every
  package after it.
- A test that only creates/exercises users, buckets, topics, OBCs must use
  the **shared store** — a private store duplicates multi-minute setup for
  nothing and hides shared-fixture regressions.
- Clients and the installer come from the fixture accessors (`AdminClient()`,
  `SnsClient()`, `Installer()`, `TLSEnabled()`) — TLS-consistent with the
  pass; never rebuilt ad-hoc.
- Prefer typed clients (`util/client`) over exec; `Installer().Execute(...)`
  is for checks with no API parity (model: the zone.json pool canary).
- Shared-cluster hygiene: unique resource names, and everything created on
  the shared store is cleaned up — leaked state is a cross-package flake
  source.
- A test that asserts nothing is dead weight: delete it, don't convert it.
- Canaries guarding cross-file invariants must name the fix location in the
  failure message (model: "add it to zonePoolNSSuffix in
  pkg/operator/ceph/object/objectstore.go").

## Worked examples (real findings that seeded this canon)

- `require.NoError` on a ref-lookup inside a verify-each-expected-secret loop
  → first missing ref silenced the rest; fixed as
  `if !assert.NoError(t, err) { continue }`. [lost-reporting]
- `wait4.RequireEventually` per-secret in a key-sync loop with the equality
  folded into the cond — nothing consumed the result → `AssertEventually`,
  with the slower-failure-path tradeoff stated. [lost-reporting]
- `require.NoError` calls inside a `utils.Retry` closure turned transient SNS
  and JSON errors into instant failures, defeating the retry — and an
  unchecked `.(string)` type assertion in the same closure could panic.
  [bug ×2]
- `*liveUser.MaxBuckets` deref gated only by `require.NoError(GetUser)` — nil
  quota fields panic the test instead of failing it. [panic-not-fail]
- "user %q was not deleted" expressed as `require.NoError(Get)` — the Get
  success IS the property, but it is the sole check in its subtest:
  acceptable as-is, style note only.

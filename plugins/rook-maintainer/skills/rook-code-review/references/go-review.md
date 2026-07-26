# Go correctness and idiom review

Scope: what rook's CI CANNOT catch. `.golangci.yaml` runs errcheck, govet,
gosec, ineffassign, staticcheck, unused, gofmt+gofumpt — do not re-report
their findings (see `verification.md`). This file is the judgment layer above
the linters.

## Rook idioms first

Match the incumbent, not an external style guide:

- **Error wrapping**: `github.com/pkg/errors` (`errors.Wrapf(err, "failed to
  create object store %q", name)`) is the dominant idiom in `pkg/`; some
  packages use `fmt.Errorf` with `%w`. Match the file/package. Flag: losing
  the cause (`fmt.Errorf` without `%w` replacing a wrap), double-prefixing
  ("failed to failed to"), and error strings that end in punctuation or start
  capitalized (they get wrapped upstream).
- **Logging**: `capnslog.NewPackageLogger("github.com/rook/rook", "<name>")`
  package logger. Message style matches the package; reconcile-visible errors
  are surfaced via status/events, not only logs.
- **Contexts**: rook threads `context.Context` explicitly (`opManagerContext`
  in operators). Flag `context.TODO()`/`context.Background()` where a context
  is in scope, and any blocking call that ignores cancellation on a reconcile
  path.

## Correctness checks (the classes linters miss)

- **Goroutine lifetimes**: every `go func` has a stated stop condition
  (context, channel close, WaitGroup). A goroutine that can outlive its
  reconcile/test is a leak and often a data race on the next pass.
- **In-band errors**: returning a zero value + logging instead of an error;
  boolean ok-values ignored (`v, ok := m[k]` misuse; unchecked type
  assertions `x.(T)` — panic path).
- **Nil traps**: methods on possibly-nil pointers from CR spec fields —
  optional CRD fields arrive as nil pointers; every deref of an `*int32`,
  `*Quota`, nested `Status` field needs a guarantee (predicate, default, or
  guard).
- **Map/slice sharing**: mutating a map/slice taken from a shared object,
  cache, or another goroutine (see `kubernetes-crd.md` for the informer-cache
  DeepCopy rule); appending to a slice aliased by the caller.
- **defer in loops** (resource pileup until function exit); `time.After` in
  select loops (timer leak per iteration) — prefer `time.NewTimer`/`Ticker`
  with Stop.
- **Shadowing across error paths**: `err :=` inside a block shadowing the
  outer `err` so the outer check tests the wrong error.
- **TOCTOU on k8s objects**: read-modify-write without conflict retry;
  decisions made on a stale read used after a write (see kubernetes-crd.md).
- **Comparisons/copies of sync types** (mutex value copies via struct copy).
- **Sentinel comparison**: string-matching errors (`strings.Contains(err
  .Error(), "NoSuchBucket")`) where typed sentinels exist (`errors.Is`,
  `kerrors.IsNotFound`, `admin.Err*` — see ceph-object.md).

## Silent-failure hunt

The defect class that dominated the 2026 AI-burst PRs (rook 17966 / 17970 /
17981): an error path that LOOKS handled but reports success. Hunt these
deliberately, then trace the caller's reaction to prove the consequence:

- `errors.Wrapf(err, ...)`/`Wrap` at a point where `err` can be nil —
  returns nil; the "failure" branch returns success (shadowing above is
  the usual cause).
- Error branches collapsing to a success sentinel — `(nil, nil)`,
  `(false, nil)` — fed into a caller that treats nil-error as "nothing to
  do".
- `_ =` on error-returning calls; dropped ok-values.
- Log-and-continue where the caller assumes the step succeeded.
- `recover()` that eats a panic without failing the operation (reconcile
  "succeeds", work silently abandoned).
- `fmt.Errorf` without `%w` where callers unwrap or `errors.Is` downstream.

## Modernization (`go fix` / modernize)

rook's `go.mod` directive is current (1.26), so the full modernize fixer
suite applies — and CI does not run it, so the reviewer is the enforcement
point. House position: **new code is written in modern Go; untouched code
is left alone.**

- Greenfield code and updated lines are EXPECTED to be modern: `any` not
  `interface{}`, `slices`/`maps` helpers over hand-rolled loops, `min`/`max`
  builtins, range-over-int, `strings.CutPrefix` over HasPrefix+TrimPrefix.
  An archaic construct on an added or updated line is a changes-requested
  finding (`style` tag) — not a nit to wave through. If the archaic form is
  also wrong, report it as the correctness finding it is.
- When a change updates a line or function, modernizing what it touches is
  PART of the update — expected, never scope creep. Scope creep is only
  rewriting logic the change does not otherwise touch; untouched code
  carries no modernization obligation, and whole-package modernize sweeps
  belong in rook-systemic-prs campaigns, not review feedback.
- The staticcheck QF classes rook suppresses (QF1001/1006/1007/1008 in
  `.golangci.yaml`) are refactor-shape rewrites, not modernization — those
  shapes stay unraised; the suppression does not excuse archaic idiom in
  new code.
- Oracle: run the modernize analyzer in report (non-writing) mode over the
  changed packages and filter to added/updated lines — never a writing
  `go fix` against a review target; remember `-tags ceph_preview`.

## Test-quality checks (unit tests)

- **Useful failures**: messages carry got/want and the identifying key
  (`assert.Equal(t, want, got, "user %q", name)`); a bare `assert.True(t,
  ok)` tells a CI reader nothing.
- Table tests: the case NAME appears in failures (`t.Run(tc.name, ...)`);
  cases actually differ (no copy-paste rows asserting the same thing).
- Fakes: `k8sfake`/fake clientsets seeded per-case, not shared mutable
  package state.
- Time: no bare `time.Sleep` synchronization in unit tests; poll with
  deadline or inject clocks.

## Coverage adequacy (does the diff carry its tests?)

Lives here, not testing.md, so it fires on the dangerous case: a code diff
with NO test changes (testing.md only routes when tests are touched).

- Enumerate the diff's new/changed branches, error paths, and boundary
  conditions; for each meaningful one, name the test that exercises it —
  or report the gap (`test-coverage` tag, citing the specific unexercised
  path).
- Calibration per verification.md: a bugfix without a regression test is
  changes-requested; a mechanical refactor without new tests is not.
  Red-before/green-after is the gold standard for regression tests.
- Integration coverage is expected only where the behavior is observable
  solely in-cluster (reconcile loops, CRD lifecycle, RGW/S3 end-to-end,
  upgrade paths); pure logic wants unit tests. `_test.go` under
  `tests/framework/` never runs in CI — it is not coverage.

## Structure and API judgment

- Exported symbols in non-`pkg/apis` code: exported only if used outside the
  package (but NEVER propose removals under `pkg/apis`/`pkg/client` — API
  stability freeze).
- Parameter explosion: 5+ positional params of the same type is a mixup
  hazard — suggest a struct only when the package already uses that shape.
- Returned interfaces vs structs, naked returns in long funcs, named results
  only where they document meaning — follow Go CodeReviewComments; cite the
  specific entry when flagging.

When a rule here conflicts with what the surrounding package consistently
does, the package wins (report nothing, or a nit at most). When the diff
introduces a NEW pattern inconsistent with the package, that is the finding —
`naming-and-comments.md` covers the naming half of this.

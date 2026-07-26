# Dead-code campaign — detector recipes

The find/verify tooling for dead-code elimination campaigns; SKILL.md owns
the loop that consumes these candidates. Run analysis with the sandbox
**disabled** (the tools write the Go build cache; in-sandbox they fail with
"read-only file system"), and write scratch files under the session
scratchpad directory.

- **Functions / methods** (whole-module reachability from main + test roots):
  `deadcode -tags=ceph_preview -test ./...`
  Sound (won't over-report) but only covers funcs/methods. Bucket by area:
  `... | grep -oE 'pkg/[^/]+/[^/]+/' | sort | uniq -c`.
- **Unused unexported** types/vars/consts/funcs/fields:
  `staticcheck -tags=ceph_preview -checks U1000 ./<pkg>/...`
- **The gap both tools miss** — exported types/vars/consts unused repo-wide,
  and **write-only fields** (a struct field assigned but never read makes its
  backing type/methods dead). Catch these by hand: enumerate exported
  non-func symbols, `grep -rIn --include=*.go -w <Name> pkg/ cmd/ tests/`,
  and confirm the only hits are the declaration/doc-comment. For fields,
  verify they're ever *read*, not just written. `deadcode` won't flag methods
  of an instantiated-but-unread type; `staticcheck` won't flag a
  written-but-unread field — only reasoning over the cluster catches it.
- **Confirm before deleting**: a symbol is "used" if referenced from *any*
  file including `_test.go`. Shared test-helper packages (e.g.
  `pkg/operator/ceph/test`) make `deadcode` over-report (helpers look
  unreachable from main) — always re-confirm test-helper candidates with a
  grep across `_test.go` files.
- **Whole-file vs partial**: list a file's top-level symbols
  (`grep -nE '^(func|type|var|const) ' <file>`); if all are dead, `git rm`
  the file (check the package doc comment isn't the only one — it may live
  on that file; if so, preserve it elsewhere).

The build tag is load-bearing for every invocation above: omitting
`-tags=ceph_preview` can abort analysis with `undefined:` errors or silently
judge a different build — see rook-conventions "Building and testing rook".

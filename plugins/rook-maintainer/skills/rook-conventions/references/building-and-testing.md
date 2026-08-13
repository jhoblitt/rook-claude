# Building, testing, and regenerating rook

## The build tag

rook's `make` targets build with `-tags ceph_preview` (Makefile
`TAGS ?= ceph_preview`; CI sets `GOFLAGS=-tags=ceph_preview`). Ad-hoc
`go build`/`vet`/`test` does NOT inherit it — export
`GOFLAGS=-tags=ceph_preview` for the session, and prefer the `make` targets.
The tag is currently dormant (rook compiles clean without it) but do NOT
strip it, and never trust a green ad-hoc build as proof it is unnecessary —
re-check which go-ceph files carry `//go:build ceph_preview` before
concluding anything.

## The local verification gate

Before any push that feeds a rook PR and changes Go code: run BOTH
`make test` and `make golangci-lint` and confirm they pass. Never push code
that fails either. Docs- or workflow-only pushes may skip both. A push that
changes `deploy/charts/**` also runs `make test.helm`.

Go unit-test CI only covers `GO_SUBDIRS` (`cmd/`, `pkg/`): `_test.go` under
`tests/framework/` compiles under golangci-lint but never runs in CI. Chart
unit tests are a separate suite — `deploy/charts/*/tests/*_test.yaml`, run by
`make test.helm` (`helm-unittest --strict`) and enforced as the
`helm-unittests` check. They satisfy the PR checklist's unit-test box
(rook-code-review `references/docs-sync.md`).

A `make golangci-lint` failure in code the change never touched is usually a
stale cache, not a real finding. A branch switch — or another session's
worktree being deleted out from under it — leaves `~/.cache/golangci-lint`
breaking the generated-file filter, so issues surface in files that are
verbatim `origin/master`. Run `rm -rf ~/.cache/golangci-lint` and re-run
before believing any such finding; the fresh-cache result is the
authoritative one. Never edit untouched code to satisfy a lint error that has
not been re-confirmed against a cleared cache.

## Regenerating CRDs and generated code

Any change under `pkg/apis` — struct/field/marker changes AND godoc wording
(comments are emitted verbatim into CRD `description` fields) — requires:

- `make codegen` — deepcopy + typed client (structure changes);
- `make crds` — CRD manifests (`deploy/examples/crds.yaml`, helm
  `resources.yaml`) and `Documentation/CRDs/specification.md` (structure,
  marker, or doc-comment changes).

Commit regenerated files in the SAME commit as the source change — never a
follow-up. CI enforces via `codegen` and `crds-gen`, and a forgotten
regeneration reddens `build.all` and every integration suite. Never narrate
the generators in the prose that ships with the change — `references/commits.md`
is canon for that, and it covers comments here too.

## Writing rook tests

Don't gate sibling subtests on each other with
`if !t.Run(name, fn) { t.FailNow() }` in a test body — failure scope becomes
too hard to reason about. Write scenario steps as a flat sequence of `t.Run`
calls with `require` inside each for closure-local gating. The only
sanctioned run-result pattern is a check helper that asserts per item and
aborts the caller, named with a `require` prefix (model: `requireRgwUserKeys`
in `tests/integration/object/user/keys/keys.go`). Accept the cascade noise
from dependent siblings.

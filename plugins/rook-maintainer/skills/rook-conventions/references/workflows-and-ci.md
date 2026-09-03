# Modifying rook/* workflows and CI configuration

## GitHub Actions workflows

- Pin every action `uses:` to a full commit SHA with `pinact`
  (`GITHUB_TOKEN=$(gh auth token) pinact run`); keep a `github-actions` entry
  in `dependabot.yml` so the `# vX.Y.Z` comments stay bumpable.
- Run `actionlint` (with `shellcheck` installed for inline `run:` scripts) on
  changed workflows and fix everything it reports.
- Pin `go-version` to the module's `go.mod` directive unless the job
  deliberately matrix-tests Go versions.

## Workflow changes and `.mergify.yml`

Adding, removing, renaming, or restructuring a workflow/job — including
`strategy.matrix` changes that alter a status-check name — always check
`.mergify.yml` for matching `check-success=`/`status-success=` conditions
before opening the PR. Backport automerge pins required checks by exact job
name; a renamed/removed check can wedge mergify or let it merge unchecked.
If the touched job isn't referenced there, no change is needed — say so,
noting that you checked.

A check name is `<job id> (<matrix values>)`, or
`<caller job id> / <called job id> (<matrix values>)` when the job is a
reusable-workflow call. Moving a job verbatim between workflow files changes
nothing; converting it to a `workflow_call` renames every one of its checks
— which is why rook/rook#18232 could fold seven workflows into one and move
every canary job into a composite action with all 35 check names intact.

## CI Kubernetes versions

Never downgrade a tested Kubernetes version to work around tooling (e.g.
minikube/kind lacking a node image) — pinned versions are deliberate
coverage; fix the tooling (`minikube --force`, bump kind/`kindest/node`).
Before pinning a new k8s version, confirm the matching `kindest/node:vX.Y.Z`
image is actually published.

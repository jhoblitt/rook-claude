# Ceph ecosystem knowledge for review

What a reviewer must know about Ceph itself to judge rook changes — plus how
to look up what this file cannot know. Prefer looking up over guessing:
Ceph behavior claims in reviews must be sourced (code, docs, or tracker).

## Releases and version gating

- Release lines (name/major): Quincy 17, Reef 18, Squid 19, Tentacle 20,
  Umbrella 21. Rook's floor lives in
  `pkg/operator/ceph/version/version.go` — `Minimum` (currently Squid
  19.2.0), with `Supported`/`Unsupported` deciding what the operator accepts
  from the `supportedVersions`/`unsupportedVersions` lists beside it.
  Verify the CURRENT values in the tree rather than trusting this line.
- **Gating rule**: a rook feature that depends on a Ceph capability must gate
  on the release that introduced it via `cephver` comparisons — not on "the
  version I tested". When reviewing a gate, confirm the capability's actual
  introduction release from Ceph release notes/source, and confirm behavior
  for versions BELOW the gate (graceful skip, validation error, or default).
- Version bumps in examples/manifests (`deploy/examples/cluster*.yaml` image
  tags, CI workflow ceph image matrices) are coverage decisions — check the
  image tag exists (quay.io/ceph/ceph:vX.Y.Z) and that docs mentioning
  versions moved with it (docs-sync).
- `PendingReleaseNotes.md`: Ceph-default-overriding config changes are
  BREAKING by project convention (the PR template says so).

## Daemon review knowledge (per subsystem, what changes must respect)

- **mon**: quorum needs a majority; counts are odd by convention; mons are
  identity-critical state (`mon` store, endpoints ConfigMap) — changes to mon
  scheduling/failover must preserve quorum through the transition and never
  remove more than one mon at a time.
- **osd**: lifecycle is prepare→activate; purge is destructive and gated;
  OSD ids are reused carefully; device/OSD topology feeds CRUSH — changes
  affecting placement must consider data movement cost. PG/health flags
  (`noout` during maintenance) have cluster-wide effects.
- **mgr**: singleton-active with standbys; modules (dashboard, prometheus)
  are mgr-hosted — module failures show up as mgr-health, and the prometheus
  module port 9283 hang on v20+ is a KNOWN upstream issue (tracker 77967,
  rook #17906) — do not accept rook-side workarounds for it without upstream
  context.
- **rgw**: see ceph-object.md.
- **mds/CephFS**: active/standby counts from the filesystem spec;
  subvolumegroups are the CSI-facing unit.
- **CSI**: rbd/cephfs/nfs drivers; provisioner vs nodeplugin split; changes
  touching mount behavior interact with kernel client versions — kernel vs
  fuse/nbd tradeoffs are deliberate.
- Cross-cutting: most rook "actions" are `ceph`/`radosgw-admin`/`rbd` CLI or
  mon-command invocations — for any new invocation, check the command exists
  with those flags on the MINIMUM supported Ceph, not just the newest.

## Live lookup procedure (in order of authority for a given question)

1. **Pinned go-ceph source** (exact API rook builds):
   `$(go env GOMODCACHE)/github.com/ceph/go-ceph@<go.mod version>/`.
2. **Ceph source** for daemon behavior: github.com/ceph/ceph, branch matching
   the release under review (`squid`, `tentacle`, ...); `src/rgw/`,
   `src/mon/`, `src/pybind/mgr/`, `qa/`. Reachable from Bash (github hosts).
3. **docs.ceph.com** via WebFetch, release-pinned path
   (`/en/squid/radosgw/...`) when the claim is version-specific.
4. **tracker.ceph.com** via WebFetch for known-bug context — especially when
   reviewing workaround code or flake-adjacent behavior; cite the tracker id
   in the finding.

A review claim about Ceph behavior that cannot be sourced from one of these
is an inference — label it as such (verification.md) or drop it.

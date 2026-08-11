# Label map — canonical roles → rook's real labels

Scope: label PROPOSALS apply to ISSUES only — triage never labels PRs
(the path-glob table below still drives PR **area inference** for routing
and KB bucketing; PR reports show current labels only, and backport labels
belong to `rook-code-review`'s flag-and-confirm flow).

Rules: intersect every proposal with live `gh label list` · ADD only —
never remove a human-applied label · ≤5 per item · under-label.
`bash "${CLAUDE_PLUGIN_ROOT}/tools/run.sh" validate-actions` re-checks the intersection and the cap
immediately before any write, so a change to either has to land there too.

## Category (kind)

bug→`bug` · feature→`feature` · docs→`docs` · security→`security` ·
performance→`performance` · reliability→`reliability` · tests→`test` ·
ci→`ci` · build→`build` · ux→`UX` · tech-debt→`technical debt` ·
cleanup→`code cleanup` · design-needed→`needs-design-document` ·
support→(no label exists; the disposition is redirect/convert).

## Area — path-glob first (deterministic for PRs)

Two namespaces, and they are not the same. **Area** is the v3 taxonomy key
the tooling emits — `bash "${CLAUDE_PLUGIN_ROOT}/tools/run.sh" rt-analyze areas <paths>`
decides it, phase 0 stamps it into each PR item's `areas`, and `rt-analyze` /
`rt-commits` bucket the KB by it. **Label** is rook's actual GitHub label, and
only ever a proposal on an ISSUE. This table is that classifier's spec: a row
whose Area is wrong is a bug in `AreasFor`, and changing one has to land in
both.

| Paths touched | Area | Issue label |
|---|---|---|
| `pkg/operator/ceph/object/**` (non-multisite/cosi) | `object` | `object` |
| RGW daemon config paths | `object` | `ceph-rgw` |
| multisite paths | `object-multisite` | `object-multisite` |
| COSI paths | `object-cosi` | `object-cosi` |
| OBC paths | `object-bucket-claims` | `object-bucket-claims` |
| `pkg/operator/ceph/cluster/mon/**` | `ceph-mon` | `ceph-mon` |
| `pkg/operator/ceph/cluster/osd/**`, `pkg/daemon/ceph/osd/**` | `ceph-osd` | `ceph-osd` |
| `pkg/operator/ceph/cluster/mgr/**` | `ceph-mgr` | `ceph-mgr` |
| `pkg/operator/ceph/file/**` | `filesystem` | `filesystem` (MDS-specific: `ceph-mds`) |
| `pkg/operator/ceph/nfs/**` | `ceph-nfs` | `ceph-nfs` |
| `pkg/operator/ceph/csi/**` | `csi` | `csi` |
| pool/RBD paths | `block` | `block` |
| `deploy/charts/**` | `helm` | `helm` |
| `Documentation/**` | `docs` | `docs` |
| `.github/workflows/**`, `tests/scripts/**` | `ci` | `ci` |
| `tests/**` | `test` | `test` |
| `pkg/apis/**` | `crd` | `crd` (API-surface changes: also `api`) |
| multus/network paths | `networking` | `multus` / `networking` |
| nvmeof paths | `nvmeof` | `nvmeof` |
| external-cluster paths | `ceph-external` | `ceph-external` |
| dashboard paths | `ceph-dashboard` | `ceph-dashboard` |
| monitoring/exporter paths | `monitoring` | `monitoring` / `ceph-exporter` |
| `go.mod`/`go.sum`, `build/**`, `images/**`, `Makefile`/`*.mk`, lint configs | `build` | `build` |
| `design/**` | `design` | `design` |
| `pkg/operator/discover/**`, `pkg/daemon/discover/**` | `discover` | `discover` |
| remaining operator/daemon plumbing — `pkg/operator/ceph/**` top-level + `controller/`/`config/`/`disruption/`/`version/`, `pkg/operator/k8sutil/**`, `pkg/daemon/ceph/{client,cleanup,util}/**`, `pkg/util/**`, `pkg/clusterd/**`, `cmd/**` | `core` | `operator` |

Six rows above have an Area that is not its Label — the fallback row most of
all, since it fires constantly. Reading a stamped `areas` value as a label
proposal is wrong on those; translate through this table.

Deliberately unbucketed (no area label): generic `deploy/examples/**` edits
and repo-meta files (root markdown, `.mergify.yml`, templates, CODE-OWNERS —
the set `repoMeta` in `internal/rtanalyze` enumerates).

A stamped `areas` therefore has three states, and they are not
interchangeable: a list is the classification; `[]` means classified and
matched nothing, which is the answer for an unbucketed path; `null` means NOT
classified, because the item's file list was truncated by pagination and a
subset would be a lower bound wearing the shape of an answer. Fall back on
`null`; never treat it as `[]`. The value is also unfiltered — a repo-wide
sweep legitimately stamps a dozen-plus areas, and routing every one would ping
the whole maintainer roster, so a wide `areas` means scope the item by hand.

Issues (no diff): infer area from exact error strings and component names
(regex layer) → KB keywords → LLM last.

## Lifecycle & status (conservative)

Usable: `duplicate` · `invalid` · `wontfix` (a DECISION — never for
staleness) · `help wanted` · `good-first-issue`.
Hands off: `stale` (bot-owned — never apply or remove) · `keepalive`
(respect: no lifecycle actions at all) · `WIP` / `do-not-merge` /
`conflicts` (author/maintainer-owned) · `Are you human?` and `debug-*`
(CI/ops tools) · backport labels (rook-code-review's flag-and-confirm
flow owns those).

Proposed-but-nonexistent — `triage-accepted`, `needs-info`: propose their
creation to the user ONCE per install; until created, that state lives in
`sweep.json` and comment text only.

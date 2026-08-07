# Label map — canonical roles → rook's real labels

Scope: label PROPOSALS apply to ISSUES only — triage never labels PRs
(the path-glob table below still drives PR **area inference** for routing
and KB bucketing; PR reports show current labels only, and backport labels
belong to `rook-code-review`'s flag-and-confirm flow).

Rules: intersect every proposal with live `gh label list` · ADD only —
never remove a human-applied label · ≤5 per item · under-label.
`scripts/validate_actions.py` re-checks the intersection and the cap
immediately before any write, so a change to either has to land there too.

## Category (kind)

bug→`bug` · feature→`feature` · docs→`docs` · security→`security` ·
performance→`performance` · reliability→`reliability` · tests→`test` ·
ci→`ci` · build→`build` · ux→`UX` · tech-debt→`technical debt` ·
cleanup→`code cleanup` · design-needed→`needs-design-document` ·
support→(no label exists; the disposition is redirect/convert).

## Area — path-glob first (deterministic for PRs)

| Paths touched | Label |
|---|---|
| `pkg/operator/ceph/object/**` (non-multisite/cosi) | `object` |
| RGW daemon config paths | `ceph-rgw` |
| multisite paths | `object-multisite` |
| COSI paths | `object-cosi` |
| OBC paths | `object-bucket-claims` |
| `pkg/operator/ceph/cluster/mon/**` | `ceph-mon` |
| `pkg/operator/ceph/cluster/osd/**`, `pkg/daemon/ceph/osd/**` | `ceph-osd` |
| `pkg/operator/ceph/cluster/mgr/**` | `ceph-mgr` |
| `pkg/operator/ceph/file/**` | `filesystem` (MDS-specific: `ceph-mds`) |
| `pkg/operator/ceph/nfs/**` | `ceph-nfs` |
| `pkg/operator/ceph/csi/**` | `csi` |
| pool/RBD paths | `block` |
| `deploy/charts/**` | `helm` |
| `Documentation/**` | `docs` |
| `.github/workflows/**`, `tests/scripts/**` | `ci` |
| `tests/**` | `test` |
| `pkg/apis/**` | `crd` (API-surface changes: also `api`) |
| multus/network paths | `multus` / `networking` |
| nvmeof paths | `nvmeof` |
| external-cluster paths | `ceph-external` |
| dashboard paths | `ceph-dashboard` |
| monitoring/exporter paths | `monitoring` / `ceph-exporter` |
| `go.mod`/`go.sum`, `build/**`, `images/**`, `Makefile`/`*.mk`, lint configs | `build` |
| `design/**` | `design` |
| `pkg/operator/discover/**`, `pkg/daemon/discover/**` | `discover` |
| remaining operator/daemon plumbing — `pkg/operator/ceph/**` top-level + `controller/`/`config/`/`disruption/`/`version/`, `pkg/operator/k8sutil/**`, `pkg/daemon/ceph/{client,cleanup,util}/**`, `pkg/util/**`, `pkg/clusterd/**`, `cmd/**` | `operator` |

Deliberately unbucketed (no area label): generic `deploy/examples/**` edits
and repo-meta files (root markdown, `.mergify.yml`, templates, CODE-OWNERS).

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

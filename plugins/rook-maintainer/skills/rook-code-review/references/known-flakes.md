# Known CI flakes — the shared registry

The single source for KNOWN-FLAKE classification (`ci-triage.md` consumes
it; rook-conventions' retry-before-escalate rule matches against it). This
file is shared canon: update it via PR to the plugin repo, with date and
evidence, whenever a signature is adjudicated across unrelated PRs — that
is how it stays alive. Entries carry as-of dates; verify anything
load-bearing that is more than a release old.

## Active classes

| Class | Signature | Notes (as of 2026-07-26) |
|---|---|---|
| ceph bring-up wedge | canary/integration job dies in "wait for ceph to be ready" / mon quorum never forms | historic, load-sensitive |
| mgr-ready on Ceph v20+ | canary waiting on mgr module / port 9283 readiness | UPSTREAM: ceph tracker 77967 (mgr/prometheus hang); rook #17906 tracks; not a rook regression |
| canary OSD under-provision | canary "main" job: fewer OSDs than requested; raw-mode device list missed a disk | UPSTREAM: ceph `get_devices` udev-data bug on Ceph 20.2.x images (ceph PR 65921); rook #17734 merged a mitigation, #17882 (per-device fallback) still open — check the job's ceph image version |
| upgrade suite step timeouts | TestCephUpgradeSuite intermittent step timeouts | long-standing; includes the PVC/CSI cold-start bind family (see smoke); post-merge master failures run from `integration-tests-on-release.yaml`, not the PR workflow |
| smoke PVC cold-start bind | first PVC of a fresh cluster times out binding right after CSI comes up | rook #17732 family; correlate with diff before calling REAL |
| helm suite pod restarts | `TestCephHelmSuite` passes every subtest, then fails in `GetPodRestartsFromNamespace` (`tests/framework/utils/k8s_helper.go`) with `expected: 0, actual: 1`, logging "number of time pod `rook-ceph-rgw-*` has restarted is 1" | that helper tolerates no restart outside the `rook-ceph-mgr` exemption already coded there for rook issue 12646, so one rgw restart fails the suite after every functional check passed. Seen on four unrelated test-only PRs (rook 17895, 17897, 17928, 17936) across both helm versions in the matrix, clearing on the first retry each time (as of 2026-08-10) |
| keystone suite | keystone auth setup/teardown timeouts | environment-heavy suite |
| multus attachment | multus setup / network-attachment failures in canary | root-caused as a `kubectl wait` empty-selector race in `setup-multus.sh` — check whether the rollout-status/vendored-manifest fix has landed before classifying |
| registry/network | image pull backoff, TLS handshake to registries, apt/pip mirror errors | INFRA |

## Recently resolved — a match here is REAL, not flake

A failure matching a resolved entry means the regression came back (or the
fix was incomplete): classify REAL and cite the entry.

| Class | Was | Resolution |
|---|---|---|
| rgw-multisite fresh-zone period-push 403 | fresh-zone multisite tests wedged on a 403 pushing the realm period | fixed by rook #17710 / #17711 / #17798 (merged ~2026-07); residual operator gap tracked in rook #17797 |

## Entry template

```text
| <class name> | <exact failure signature strings> | <upstream/rook tracking refs; classification guidance; as-of date> |
```

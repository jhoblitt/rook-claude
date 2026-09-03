There is no rook checkout, no network,
and no `gh`, and subagents cannot be spawned in this environment: treat
the PR metadata and review thread below as complete — there is nothing
further to fetch, and the `CODE-OWNERS` roster given below is what
`origin/master` carries — and run the passes of the rook-code-review
skill's review spine inline.

Review the following pull request. Your entire final answer is the review
report — the routed-reference list from spine step 1, verdict line,
findings in the skill's finding contract, the PR-target additions the
skill's output format requires, and the audited-and-clean statement.
Nothing else.

---

**PR #18172** — base `master`, author `contributor-f` (association
`CONTRIBUTOR`), no labels, head `9c4e2d1`

`CODE-OWNERS` at `origin/master`: `approver-k` is listed under
`approvers:`; `contributor-f` does not appear.

Title: `exporter: run ceph-exporter and ceph-crash with a restricted security context`

Body:

```text
The ceph-exporter and ceph-crash containers run with the pod default
security context, so the cluster's admission policy warns on every
node daemon pod. Set the restricted baseline on both containers.

- [ ] Documentation has been updated
- [ ] Unit tests have been added
- [ ] Integration tests have been added
- [ ] Pending release notes updated
```

Commits (one):

```text
exporter: run ceph-exporter and ceph-crash with a restricted security context

Signed-off-by: F Contributor <f@example.com>
```

Diff:

```diff
--- a/pkg/operator/ceph/cluster/nodedaemon/exporter.go
+++ b/pkg/operator/ceph/cluster/nodedaemon/exporter.go
@@ -178,6 +178,15 @@ func getCephExporterDaemonContainer(cephCluster cephv1.CephCluster, cephVersion cephver.CephVersion) v1.Container {
 		VolumeMounts: volumeMounts,
 		Resources:    cephv1.GetCephExporterResources(cephCluster.Spec.Resources),
 	}
 
+	runAsNonRoot := true
+	readOnlyRootFilesystem := true
+	allowPrivilegeEscalation := false
+	container.SecurityContext = &v1.SecurityContext{
+		RunAsNonRoot:             &runAsNonRoot,
+		ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
+		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
+	}
+
 	return container
 }
--- a/pkg/operator/ceph/cluster/nodedaemon/crash.go
+++ b/pkg/operator/ceph/cluster/nodedaemon/crash.go
@@ -201,6 +201,13 @@ func getCrashDaemonContainer(cephCluster cephv1.CephCluster, cephVersion cephver.CephVersion) v1.Container {
 		VolumeMounts: volumeMounts,
 		Resources:    cephv1.GetCrashCollectorResources(cephCluster.Spec.Resources),
 	}
 
+	runAsNonRoot := true
+	readOnlyRootFilesystem := true
+	container.SecurityContext = &v1.SecurityContext{
+		RunAsNonRoot:           &runAsNonRoot,
+		ReadOnlyRootFilesystem: &readOnlyRootFilesystem,
+	}
+
 	return container
 }
```

Review threads (`reviewThreads`, one page, `hasNextPage: false`):

```text
1. pkg/operator/ceph/cluster/nodedaemon/exporter.go:182 — unresolved,
   not outdated, RIGHT side
   approver-k (2026-08-28): nit: can replace this with `ptr.To(true)`
   and drop the local — same for the two below it.
```

Reviews: `approver-k` — COMMENTED (2026-08-28); no APPROVED or
CHANGES_REQUESTED review on the PR. No pushes since the comment.

CI: all checks green.

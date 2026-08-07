The rook-maintainer plugin is loaded. There is no rook checkout, no network,
and no `gh`, and subagents cannot be spawned in this environment: treat the
PR metadata below as complete — there is nothing further to fetch — and run
the passes of the rook-code-review skill's review spine inline.

Review the following pull request. Your entire final answer is the review
report — verdict line, any findings in the skill's finding contract, the
PR-target additions the skill's output format requires, and the
audited-and-clean statement. Nothing else.

---

**PR #18151** — base `master`, author `contributor-e` (association `MEMBER`),
no labels

Title: `object: allow tuning the RGW readiness probe timeout`

Body:

```text
Operators on slow storage report the RGW readiness probe failing during
recovery because its timeout is not tunable. This adds an optional field.

- [x] Documentation has been updated
- [x] Unit tests have been added
- [ ] Integration tests have been added
- [x] Pending release notes updated
```

Commits (one):

```text
object: allow tuning the RGW readiness probe timeout

Signed-off-by: E Contributor <e@example.com>
```

Changed files:

```text
pkg/apis/ceph.rook.io/v1/types.go
pkg/apis/ceph.rook.io/v1/zz_generated.deepcopy.go        (regenerated)
deploy/examples/crds.yaml                                (regenerated)
deploy/charts/rook-ceph/templates/resources.yaml         (regenerated)
Documentation/CRDs/specification.md                      (regenerated)
Documentation/CRDs/Object-Storage/ceph-object-store-crd.md
PendingReleaseNotes.md
pkg/operator/ceph/object/spec.go
pkg/operator/ceph/object/spec_test.go
```

Diff (source hunks; the regenerated files match):

```diff
--- a/pkg/apis/ceph.rook.io/v1/types.go
+++ b/pkg/apis/ceph.rook.io/v1/types.go
@@ -120,6 +120,10 @@ type ObjectHealthCheckSpec struct {
 	// ReadinessProbe configures the RGW readiness probe
 	// +optional
 	ReadinessProbe *ProbeSpec `json:"readinessProbe,omitempty"`
+
+	// ReadinessProbeTimeoutSeconds overrides the RGW readiness probe timeout.
+	// +optional
+	ReadinessProbeTimeoutSeconds *int32 `json:"readinessProbeTimeoutSeconds,omitempty"`
 }
--- a/pkg/operator/ceph/object/spec.go
+++ b/pkg/operator/ceph/object/spec.go
@@ -310,6 +310,9 @@ func (c *clusterConfig) generateProbe() *v1.Probe {
 	p := defaultReadinessProbe()
+	if t := c.store.Spec.HealthCheck.ReadinessProbeTimeoutSeconds; t != nil {
+		p.TimeoutSeconds = *t
+	}
 	return p
 }
--- a/PendingReleaseNotes.md
+++ b/PendingReleaseNotes.md
@@ -4,3 +4,4 @@
 ## Features
 
 - Rook can now be configured to ...
+- CephObjectStore supports `healthCheck.readinessProbeTimeoutSeconds`.
```

CI: all checks green.

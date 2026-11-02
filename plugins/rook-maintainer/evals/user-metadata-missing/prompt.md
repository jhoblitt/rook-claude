There is no rook checkout, no
network, no `gh`, and subagents cannot be spawned in this environment:
review the following branch inline per the rook-code-review skill's
review spine, verifying what the diff and commit message themselves
allow and labeling everything else INFERENCE.

The branch also carries the RBAC, `Documentation/` and
`PendingReleaseNotes.md` updates and unit tests; those hunks are elided
from this excerpt. It changes no CRD types.

Your entire final answer is the review report — the routed-reference
list from spine step 1, verdict line, findings in the skill's finding
contract, and the audited-and-clean section. Nothing else.

Commit message (the only commit on the branch):

```text
object: publish store endpoints in a ConfigMap

In-cluster clients that cannot read CephObjectStore CRs need the S3
endpoints of a store. The object store controller now writes them to a
rook-ceph-rgw-<store>-endpoints ConfigMap, owned by the store and
updated on every reconcile.
```

```diff
--- /dev/null
+++ b/pkg/operator/ceph/object/endpoints.go
@@ -0,0 +1,35 @@
+package object
+
+import (
+	"fmt"
+	"strings"
+
+	"github.com/pkg/errors"
+	opcontroller "github.com/rook/rook/pkg/operator/ceph/controller"
+	corev1 "k8s.io/api/core/v1"
+	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
+)
+
+// reconcileEndpointsConfigMap publishes the store's S3 endpoints for in-cluster
+// clients that cannot read the CephObjectStore.
+func (c *clusterConfig) reconcileEndpointsConfigMap() error {
+	cm := &corev1.ConfigMap{
+		ObjectMeta: metav1.ObjectMeta{
+			Name:      fmt.Sprintf("rook-ceph-rgw-%s-endpoints", c.store.Name),
+			Namespace: c.store.Namespace,
+			Labels: map[string]string{
+				"app":               AppName,
+				"rook_cluster":      c.store.Namespace,
+				"rook_object_store": c.store.Name,
+			},
+		},
+		Data: map[string]string{
+			"insecure": strings.Join(c.store.Status.Endpoints.Insecure, ","),
+			"secure":   strings.Join(c.store.Status.Endpoints.Secure, ","),
+		},
+	}
+	if err := c.ownerInfo.SetControllerReference(cm); err != nil {
+		return errors.Wrapf(err, "failed to set owner reference on configmap %q", cm.Name)
+	}
+	return opcontroller.CreateOrUpdateObject(c.clusterInfo.Context, c.context.Client, cm)
+}
--- a/pkg/operator/ceph/object/controller.go
+++ b/pkg/operator/ceph/object/controller.go
@@ -512,6 +512,10 @@ func (r *ReconcileCephObjectStore) reconcileCreateObjectStore(cephObjectStore *cephv1.CephObjectStore, namespacedName types.NamespacedName, cluster cephv1.ClusterSpec) (reconcile.Result, error) {
 		return r.setFailedStatus(k8sutil.ObservedGenerationNotAvailable, namespacedName, "failed to start rgw health checker", err)
 	}
 
+	if err := cfg.reconcileEndpointsConfigMap(); err != nil {
+		return r.setFailedStatus(k8sutil.ObservedGenerationNotAvailable, namespacedName, "failed to publish object store endpoints", err)
+	}
+
 	return reconcile.Result{}, nil
 }
```

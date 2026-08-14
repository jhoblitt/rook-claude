There is no rook checkout, no
network, no `gh`, and subagents cannot be spawned in this environment:
review the following diff inline per the rook-code-review skill's
review spine, verifying what the diff and commit message themselves
allow and labeling everything else INFERENCE.

Your entire final answer is the review report — the routed-reference
list from spine step 1, verdict line, findings in the skill's finding
contract, and the audited-and-clean section. Nothing else.

Commit message (the only commit on the branch):

```text
object: log the desired gateway instance count

Support requests about a scaled object store could not be answered
from the operator log alone — the reconcile path never recorded how
many RGW instances the CR asks for.
```

```diff
--- a/pkg/operator/ceph/object/controller.go
+++ b/pkg/operator/ceph/object/controller.go
@@ -612,6 +612,8 @@ func (r *ReconcileCephObjectStore) reconcileCreateObjectStore(cr *cephv1.CephObjectStore) (reconcile.Result, error) {
 	if err := r.reconcileMultisiteCRs(cr); err != nil {
 		return reconcile.Result{}, err
 	}
 
+	logger.Infof("desired replicas=%d", cr.Spec.Gateway.Instances)
+
 	return r.createOrUpdateStore(cr)
 }
```

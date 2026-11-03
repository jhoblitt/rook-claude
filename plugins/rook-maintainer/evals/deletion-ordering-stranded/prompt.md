There is no rook checkout, no
network, no `gh`, and subagents cannot be spawned in this environment:
review the following branch inline per the rook-code-review skill's
review spine, verifying what the diff and commit message themselves
allow and labeling everything else INFERENCE.

The branch also carries the new kind's types, the regenerated CRD and
codegen artifacts, RBAC, the helm and `Documentation/` updates, unit
tests, and a `PendingReleaseNotes.md` entry; those hunks are elided from
this excerpt. It does not touch the CephObjectZoneGroup controller,
which has no check for sync policies before a zone group is deleted.

Your entire final answer is the review report — verdict line, findings
in the skill's finding contract, and the audited-and-clean section.
Nothing else.

Commit message (the only commit on the branch):

```text
object: add CephObjectZoneGroupSyncPolicy

Adds a CephObjectZoneGroupSyncPolicy CRD that manages one multisite
sync policy group on the zone group named by spec.zoneGroup. The
controller adds a finalizer before creating the sync group and, when
the CR is deleted, removes the sync group from Ceph before releasing
the finalizer.
```

```diff
--- a/pkg/operator/ceph/cluster/dependents.go
+++ b/pkg/operator/ceph/cluster/dependents.go
@@ -40,5 +40,6 @@ var cephClusterDependentListKinds []string = []string{
 	"CephObjectZoneList",
 	"CephObjectZoneGroupList",
+	"CephObjectZoneGroupSyncPolicyList",
 	"CephObjectRealmList",
 	"CephNFSList",
 	"CephClientList",
--- /dev/null
+++ b/pkg/operator/ceph/object/syncpolicy/controller.go
@@ -0,0 +1,51 @@
+func (r *ReconcileSyncPolicy) reconcile(request reconcile.Request) (reconcile.Result, cephv1.CephObjectZoneGroupSyncPolicy, error) {
+	policy := &cephv1.CephObjectZoneGroupSyncPolicy{}
+	err := r.client.Get(r.opManagerContext, request.NamespacedName, policy)
+	if err != nil {
+		if kerrors.IsNotFound(err) {
+			return reconcile.Result{}, *policy, nil
+		}
+		return reconcile.Result{}, *policy, errors.Wrap(err, "failed to get sync policy")
+	}
+
+	if policy.GetDeletionTimestamp().IsZero() {
+		generationUpdated, err := opcontroller.AddFinalizerIfNotPresent(r.opManagerContext, r.client, policy)
+		if err != nil {
+			return reconcile.Result{}, *policy, errors.Wrap(err, "failed to add finalizer")
+		}
+		if generationUpdated {
+			return reconcile.Result{}, *policy, nil
+		}
+	}
+
+	_, isReadyToReconcile, cephClusterExists, reconcileResponse := opcontroller.IsReadyToReconcile(r.opManagerContext, r.client, request.NamespacedName, controllerName)
+	if !isReadyToReconcile {
+		if !policy.GetDeletionTimestamp().IsZero() && !cephClusterExists {
+			// No CephCluster, or one being destroyed: nothing is left in Ceph to clean up.
+			err := opcontroller.RemoveFinalizer(r.opManagerContext, r.client, policy)
+			return reconcile.Result{}, *policy, err
+		}
+		return reconcileResponse, *policy, nil
+	}
+
+	// The sync group lives in the zone group, which names the realm.
+	zoneGroup := &cephv1.CephObjectZoneGroup{}
+	err = r.client.Get(r.opManagerContext, types.NamespacedName{Name: policy.Spec.ZoneGroup, Namespace: policy.Namespace}, zoneGroup)
+	if err != nil {
+		return waitForRequeueIfZoneGroupNotReady, *policy, errors.Wrapf(err, "failed to get zone group %q", policy.Spec.ZoneGroup)
+	}
+	objContext := object.NewContext(r.context, r.clusterInfo, policy.Name)
+	objContext.Realm = zoneGroup.Spec.Realm
+	objContext.ZoneGroup = zoneGroup.Name
+
+	if !policy.GetDeletionTimestamp().IsZero() {
+		err := removeSyncGroup(objContext, policy.Spec.GroupID)
+		if err != nil {
+			return reconcile.Result{}, *policy, errors.Wrapf(err, "failed to remove sync group %q", policy.Spec.GroupID)
+		}
+		err = opcontroller.RemoveFinalizer(r.opManagerContext, r.client, policy)
+		return reconcile.Result{}, *policy, err
+	}
+
+	return r.reconcileSyncGroup(objContext, policy)
+}
```

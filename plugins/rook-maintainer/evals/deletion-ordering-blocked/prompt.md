There is no rook checkout, no
network, no `gh`, and subagents cannot be spawned in this environment:
review the following branch inline per the rook-code-review skill's
review spine, verifying what the diff and commit message themselves
allow and labeling everything else INFERENCE.

The branch also carries the new kind's types, the regenerated CRD and
codegen artifacts, RBAC, the helm and `Documentation/` updates, unit
tests, and a `PendingReleaseNotes.md` entry; those hunks are elided from
this excerpt. The elided types mark `spec.zoneGroup` and `spec.groupID`
immutable (`self == oldSelf`). On the target branch the
CephObjectZoneGroup controller already holds a finalizer and, while
deleting, blocks on the dependents `CephObjectZoneGroupDependents`
returns, reporting them through
`reporting.ReportDeletionBlockedDueToDependents`; until this branch
those dependents are its CephObjectZones only.

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

The policy's deletion reads its zone group, so a zone group now counts
its sync policies as dependents and is not deleted until they are gone,
and no sync group is created in a zone group that is being deleted.
Together those mean a policy whose zone group CR is missing never
created a sync group -- a typo in spec.zoneGroup, a zone group deleted
before the policy existed -- so its deletion releases the finalizer
without one. An integration test deletes a zone group ahead of its
policy and checks that the zone group waits for it.
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
--- a/pkg/operator/ceph/object/zonegroup/dependents.go
+++ b/pkg/operator/ceph/object/zonegroup/dependents.go
@@ -48,5 +48,16 @@ func CephObjectZoneGroupDependents(
 		}
 	}
 
+	// CephObjectZoneGroupSyncPolicies
+	policies, err := clusterdCtx.RookClientset.CephV1().CephObjectZoneGroupSyncPolicies(zoneGroup.Namespace).List(clusterInfo.Context, metav1.ListOptions{})
+	if err != nil {
+		return deps, errors.Wrapf(err, "%s. failed to list CephObjectZoneGroupSyncPolicies for CephObjectZoneGroup %q", baseErrMsg, nsName)
+	}
+	for _, policy := range policies.Items {
+		if policy.Spec.ZoneGroup == zoneGroup.Name {
+			deps.Add("CephObjectZoneGroupSyncPolicy", policy.Name)
+		}
+	}
+
 	return deps, nil
 }
--- /dev/null
+++ b/pkg/operator/ceph/object/syncpolicy/controller.go
@@ -0,0 +1,64 @@
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
+		if kerrors.IsNotFound(err) && !policy.GetDeletionTimestamp().IsZero() {
+			// A zone group is not deleted while this policy exists, and no sync group
+			// is created in a zone group being deleted (below), so a missing zone
+			// group means this policy never created a sync group in Ceph.
+			log.NamedInfo(request.NamespacedName, logger, "zone group %q not found; no sync group to remove", policy.Spec.ZoneGroup)
+			err := opcontroller.RemoveFinalizer(r.opManagerContext, r.client, policy)
+			return reconcile.Result{}, *policy, err
+		}
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
+	// Its dependents check may already have run without this policy.
+	if !zoneGroup.GetDeletionTimestamp().IsZero() {
+		return waitForRequeueIfZoneGroupNotReady, *policy, errors.Errorf("zone group %q is being deleted", policy.Spec.ZoneGroup)
+	}
+
+	return r.reconcileSyncGroup(objContext, policy)
+}
--- a/tests/integration/object/dependents/dependents.go
+++ b/tests/integration/object/dependents/dependents.go
@@ -160,3 +160,29 @@ func TestCephObjectStoreDependents(t *testing.T, k8sh *utils.K8sHelper, installer *installer.CephInstaller, namespace string, tlsEnable bool) {
 		})
 	})
 }
+
+// TestCephObjectZoneGroupSyncPolicyDependents deletes a zone group ahead of its
+// sync policy, the order a namespace deletion may take: the zone group must wait,
+// naming the policy, and both must be gone once the policy is deleted. ms is a
+// private multisite fixture this test owns and tears down.
+func TestCephObjectZoneGroupSyncPolicyDependents(t *testing.T, k8sh *utils.K8sHelper, ms *multisite.Fixture) {
+	t.Run("CephObjectZoneGroup waits for its sync policies", func(t *testing.T) {
+		ctx := t.Context()
+		zgClient := k8sh.RookClientset.CephV1().CephObjectZoneGroups(ms.Namespace)
+		spClient := k8sh.RookClientset.CephV1().CephObjectZoneGroupSyncPolicies(ms.Namespace)
+		policy := ms.RequireSyncPolicy(ctx, t, "dependents-policy")
+
+		t.Run("deleting the zone group is blocked by its sync policy", func(t *testing.T) {
+			require.NoError(t, zgClient.Delete(ctx, ms.ZoneGroup.Name, metav1.DeleteOptions{}))
+			live := wait4.RequireCondition(ctx, t, zgClient, ms.ZoneGroup.Name, wait4.ZoneGroupDeletionBlocked, wait4.TimeoutShort)
+			cond := cephv1.FindStatusCondition(live.Status.Conditions, cephv1.ConditionDeletionIsBlocked)
+			require.NotNil(t, cond)
+			assert.Contains(t, cond.Message, policy.Name)
+		})
+
+		t.Run("deleting the policy releases the zone group", func(t *testing.T) {
+			wait4.RequireDelete(ctx, t, spClient, policy.Name, wait4.TimeoutLong)
+			wait4.AssertAbsent(ctx, t, zgClient, ms.ZoneGroup.Name, wait4.TimeoutLong)
+		})
+	})
+}
```

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
object: log the keystone endpoint the operator dials

A store that silently failed keystone auth gave no clue which endpoint
the operator had actually assembled. The reconcile path now logs it,
and the status path logs the endpoint it publishes to the CR.
```

```diff
--- a/pkg/operator/ceph/object/keystone.go
+++ b/pkg/operator/ceph/object/keystone.go
@@ -118,7 +118,10 @@ func (r *ReconcileCephObjectStore) configureKeystone(cr *cephv1.CephObjectStore, secret *v1.Secret) error {
 	user := string(secret.Data["OS_USERNAME"])
 	pass := string(secret.Data["OS_PASSWORD"])
 	host := keystoneHost(cr)
 
+	u := fmt.Sprintf("https://%s:%s@%s/", user, pass, host)
+	logger.Infof("endpoint %s", u)
+
 	if err := probeKeystone(u); err != nil {
 		return errors.Wrap(err, "keystone is not reachable")
 	}
@@ -157,6 +160,9 @@ func (r *ReconcileCephObjectStore) recordKeystoneEndpoint(cr *cephv1.CephObjectStore, u, user, pass string) {
 	if cr.Status.Info == nil {
 		cr.Status.Info = map[string]string{}
 	}
 
+	stripped := strings.Replace(u, user+":"+pass+"@", "", 1)
+	logger.Infof("keystone endpoint %s", stripped)
+
 	cr.Status.Phase = cephv1.ConditionReady
 }
```

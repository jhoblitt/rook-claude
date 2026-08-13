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
object: stop passing the rgw admin password on the command line

The user-create path now writes the keys radosgw-admin minted straight
into the user's Secret, and hands radosgw-admin the admin account's
password on stdin instead of as an argv flag.
```

```diff
--- a/pkg/operator/ceph/object/user/controller.go
+++ b/pkg/operator/ceph/object/user/controller.go
@@ -696,8 +696,11 @@ func (r *ReconcileObjectStoreUser) generateCephUserSecret(u *cephv1.CephObjectStoreUser, userConfig *admin.User) *corev1.Secret {
 	secret := &corev1.Secret{
 		ObjectMeta: metav1.ObjectMeta{
 			Name:      generateCephUserSecretName(u),
 			Namespace: u.Namespace,
 		},
-		StringData: map[string]string{},
-		Type:       k8sutil.RookType,
+		StringData: map[string]string{
+			"access-key": userConfig.Keys[0].AccessKey,
+			"secret-key": userConfig.Keys[0].SecretKey,
+		},
+		Type: k8sutil.RookType,
 	}
--- a/pkg/operator/ceph/object/admin.go
+++ b/pkg/operator/ceph/object/admin.go
@@ -212,7 +212,7 @@ func (c *Context) runAdminCommand(args ...string) (string, error) {
-	args = append(args, "--admin-password", c.adminPassword)
-
 	cmd := exec.Command(rgwAdminCommand, args...)
+	cmd.Stdin = strings.NewReader(c.adminPassword)
+
 	out, err := cmd.CombinedOutput()
 	if err != nil {
 		return "", errors.Wrap(err, "failed to run radosgw-admin")
 	}
```

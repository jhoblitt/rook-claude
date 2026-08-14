There is no rook checkout, no
network, no `gh`, and subagents cannot be spawned in this environment:
review the following diff inline per the rook-code-review skill's
review spine, verifying what the diff and commit message themselves
allow and labeling everything else INFERENCE.

The `KeystoneSpec` excerpt is unchanged context from `master` — this
diff adds no line to `pkg/apis/ceph.rook.io/v1/types.go`; the hunk is
shown so the fields' declared types are visible.

Your entire final answer is the review report — the routed-reference
list from spine step 1, verdict line, findings in the skill's finding
contract, and the audited-and-clean section. Nothing else.

Commit message (the only commit on the branch):

```text
object: name the keystone admin locals for what they are

configureKeystone's one-letter locals read like loop variables. No
behavior change.
```

```diff
--- a/pkg/apis/ceph.rook.io/v1/types.go
+++ b/pkg/apis/ceph.rook.io/v1/types.go
@@ -2298,7 +2298,7 @@ type KeystoneSpec struct {
 	// AdminUser is the keystone admin account this store authenticates as.
 	// +optional
 	AdminUser string `json:"adminUser,omitempty"`
 
 	// AdminPassword is that account's password.
 	// +optional
 	AdminPassword string `json:"adminPassword,omitempty"`
--- a/pkg/operator/ceph/object/keystone.go
+++ b/pkg/operator/ceph/object/keystone.go
@@ -118,8 +118,8 @@ func (r *ReconcileCephObjectStore) configureKeystone(cr *cephv1.CephObjectStore) error {
-	u := cr.Spec.Auth.Keystone.AdminUser
-	p := cr.Spec.Auth.Keystone.AdminPassword
-	if u == "" {
+	adminUser := cr.Spec.Auth.Keystone.AdminUser
+	adminPassword := cr.Spec.Auth.Keystone.AdminPassword
+	if adminUser == "" {
 		return nil
 	}
 
-	return r.applyKeystoneAuth(cr, u, p)
+	return r.applyKeystoneAuth(cr, adminUser, adminPassword)
 }
```

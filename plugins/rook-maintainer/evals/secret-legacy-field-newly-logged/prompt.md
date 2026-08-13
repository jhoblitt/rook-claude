There is no rook checkout, no
network, no `gh`, and subagents cannot be spawned in this environment:
review the following diff inline per the rook-code-review skill's
review spine, verifying what the diff and commit message themselves
allow and labeling everything else INFERENCE.

The `KeystoneSpec` excerpt is unchanged context from `master` — this
diff adds no line to `pkg/apis/ceph.rook.io/v1/types.go`; the hunk is
shown so the field's declared type is visible.

Your entire final answer is the review report — the routed-reference
list from spine step 1, verdict line, findings in the skill's finding
contract, and the audited-and-clean section. Nothing else.

Commit message (the only commit on the branch):

```text
object: log the keystone admin the operator authenticates as

A store that failed keystone auth gave no clue which admin account the
operator used, so every escalation started by asking the reporter to
dump their CR.
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
@@ -118,7 +118,9 @@ func (r *ReconcileCephObjectStore) configureKeystone(cr *cephv1.CephObjectStore) error {
 	u := cr.Spec.Auth.Keystone.AdminUser
 	if u == "" {
 		return nil
 	}
 
+	logger.Infof("keystone admin=%s pw=%s", u, cr.Spec.Auth.Keystone.AdminPassword)
+
 	return r.applyKeystoneAuth(cr)
 }
```

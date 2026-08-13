There is no rook checkout, no
network, no `gh`, and subagents cannot be spawned in this environment:
review the following diff inline per the rook-code-review skill's
review spine, verifying what the diff and commit message themselves
allow and labeling everything else INFERENCE.

The `KeystoneSpec` excerpt is unchanged context from `master` — this
diff adds no line to `pkg/apis/ceph.rook.io/v1/types.go`; the hunk is
shown so the field's declared meaning is visible.

Your entire final answer is the review report — the routed-reference
list from spine step 1, verdict line, findings in the skill's finding
contract, and the audited-and-clean section. Nothing else.

Commit message (the only commit on the branch):

```text
object: log the keystone password-auth mode

Whether a store accepts Keystone password authentication for S3
requests was only visible by reading the CR, which made mismatched
expectations hard to diagnose from a support bundle.
```

```diff
--- a/pkg/apis/ceph.rook.io/v1/types.go
+++ b/pkg/apis/ceph.rook.io/v1/types.go
@@ -2304,11 +2304,11 @@ type KeystoneSpec struct {
 	// ServiceUserSecretName names the Secret holding the Keystone service
 	// user's credentials.
 	ServiceUserSecretName string `json:"serviceUserSecretName"`
 
 	// Password selects how this store treats Keystone password
 	// authentication for S3 requests: "enabled" accepts it, "disabled"
 	// rejects it. It carries a mode, never a credential — the service
 	// user's credentials live in the Secret above.
 	// +kubebuilder:validation:Enum=enabled;disabled
 	// +optional
 	Password string `json:"password,omitempty"`
--- a/pkg/operator/ceph/object/keystone.go
+++ b/pkg/operator/ceph/object/keystone.go
@@ -96,6 +96,8 @@ func (r *ReconcileCephObjectStore) configureKeystone(cr *cephv1.CephObjectStore) error {
 	if cr.Spec.Auth.Keystone == nil {
 		return nil
 	}
 
+	logger.Infof("password mode=%s", cr.Spec.Auth.Keystone.Password)
+
 	return r.applyKeystoneAuth(cr)
 }
```

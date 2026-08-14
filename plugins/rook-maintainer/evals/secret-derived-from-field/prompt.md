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
object: trace the keystone auth value on debug

Diagnosing a keystone rejection meant guessing what the operator had
actually sent. The value the request carries is now logged at debug
level, in the encoded form it goes out in.
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
@@ -142,7 +142,9 @@ func (r *ReconcileCephObjectStore) applyKeystoneAuth(cr *cephv1.CephObjectStore) error {
 	req, err := http.NewRequest(http.MethodPost, keystoneHost(cr)+"/v3/auth/tokens", body)
 	if err != nil {
 		return errors.Wrap(err, "failed to build the keystone auth request")
 	}
 
+	logger.Debugf("tok=%s", base64.StdEncoding.EncodeToString([]byte(cr.Spec.Auth.Keystone.AdminPassword)))
+
 	return r.do(req)
 }
```

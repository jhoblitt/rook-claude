There is no rook checkout, no
network, no `gh`, and subagents cannot be spawned in this environment:
review the following branch inline per the rook-code-review skill's
review spine, verifying what the diff and commit message themselves
allow and labeling everything else INFERENCE.

The branch also carries the regenerated CRD/codegen artifacts and the
`Documentation/` update; those hunks are elided from this excerpt.

Your entire final answer is the review report — the routed-reference
list from spine step 1, verdict line, findings in the skill's finding
contract, and the audited-and-clean section. Nothing else.

Commit message (the only commit on the branch):

```text
object: add keystone service-user credentials to CephObjectStore

Stores that talk to keystone need a service account. Putting the
account and its password in spec.auth.keystone lets the operator
configure the keystone integration without any extra setup.
```

```diff
--- a/pkg/apis/ceph.rook.io/v1/types.go
+++ b/pkg/apis/ceph.rook.io/v1/types.go
@@ -2228,5 +2228,15 @@ type KeystoneSpec struct {
 	// The URL for the Keystone server.
 	Url string `json:"url"`
 	// The roles requires to serve requests.
 	AcceptedRoles []string `json:"acceptedRoles"`
+
+	// ServiceUser is the keystone service account the RGW daemons
+	// authenticate as.
+	// +optional
+	ServiceUser string `json:"serviceUser,omitempty"`
+
+	// ServicePassword is that account's password.
+	// +optional
+	// +kubebuilder:validation:MinLength=1
+	ServicePassword string `json:"servicePassword,omitempty"`
 }
```

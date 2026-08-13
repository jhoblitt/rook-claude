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
object: accept the keystone admin endpoint as issued

Operators behind a managed keystone are handed one admin URL with the
account already embedded in it. Taking that URL as-is means they paste
what their provider gave them instead of taking it apart by hand.
```

```diff
--- a/pkg/apis/ceph.rook.io/v1/types.go
+++ b/pkg/apis/ceph.rook.io/v1/types.go
@@ -2228,5 +2228,13 @@ type KeystoneSpec struct {
 	// The URL for the Keystone server.
 	Url string `json:"url"`
 	// The roles requires to serve requests.
 	AcceptedRoles []string `json:"acceptedRoles"`
+
+	// Endpoint is the keystone admin endpoint the operator calls to
+	// validate tokens. The provider issues it with the admin account
+	// embedded in the URL, in the form
+	// https://USER:PASSWORD@keystone.example.com:5000/v3 — paste it
+	// exactly as issued.
+	// +optional
+	Endpoint string `json:"endpoint,omitempty"`
 }
```

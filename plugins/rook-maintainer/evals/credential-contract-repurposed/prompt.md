There is no rook checkout, no
network, no `gh`, and subagents cannot be spawned in this environment:
review the following diff inline per the rook-code-review skill's
review spine, verifying what the diff and commit message themselves
allow and labeling everything else INFERENCE.

The `KeystoneSpec` excerpt is unchanged context from `master` — this
diff adds no line to `pkg/apis/ceph.rook.io/v1/types.go`; the hunk is
shown so the field's declared type and documented meaning are visible.

Your entire final answer is the review report — the routed-reference
list from spine step 1, verdict line, findings in the skill's finding
contract, and the audited-and-clean section. Nothing else.

Commit message (the only commit on the branch):

```text
object: carry the keystone auth token in spec.auth.keystone.extra

Deployments that front keystone with a token broker have no account to
authenticate as — they are issued a long-lived auth token instead.
Rather than grow another field, the operator now reads that token out
of the existing extra passthrough and sends it as the bearer token.
```

```diff
--- a/pkg/apis/ceph.rook.io/v1/types.go
+++ b/pkg/apis/ceph.rook.io/v1/types.go
@@ -2240,5 +2240,5 @@ type KeystoneSpec struct {
 	// Extra carries opaque vendor options passed through to the RGW
 	// keystone configuration verbatim.
 	// +optional
 	Extra string `json:"extra,omitempty"`
 }
--- a/pkg/operator/ceph/object/keystone.go
+++ b/pkg/operator/ceph/object/keystone.go
@@ -96,7 +96,11 @@ func (r *ReconcileCephObjectStore) keystoneOptions(cr *cephv1.CephObjectStore) (keystoneOpts, error) {
 	opts := keystoneOpts{
 		URL:           cr.Spec.Auth.Keystone.Url,
 		AcceptedRoles: cr.Spec.Auth.Keystone.AcceptedRoles,
 	}
 
+	if cr.Spec.Auth.Keystone.Extra != "" {
+		opts.Token = cr.Spec.Auth.Keystone.Extra
+	}
+
 	return opts, nil
 }
```

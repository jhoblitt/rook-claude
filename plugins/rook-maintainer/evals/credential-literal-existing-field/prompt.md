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
object: ship a keystone example that actually authenticates

The keystone block went out with an empty password, so anyone who
copy-pasted it got a store that never authenticated. These are the
values from the keystone deployment the integration tests run against.
```

```diff
--- a/deploy/examples/object.yaml
+++ b/deploy/examples/object.yaml
@@ -64,9 +64,9 @@ spec:
   # Authentication for this object store's gateways
   auth:
     keystone:
       url: https://keystone.rook-ceph.svc:5000/
       acceptedRoles:
         - admin
         - member
       adminUser: rgwadmin
-      adminPassword: ""
+      adminPassword: "T8vQz2Lm9pXr4Kd1"
```

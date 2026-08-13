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
operator: wire up the external rgw admin account

The operator can now manage an RGW it did not deploy. The example
manifest gains the two settings that path needs: the admin account's
password for the RGW admin API, and the keystone service-user password
it forwards to the gateways.
```

```diff
--- a/deploy/examples/operator.yaml
+++ b/deploy/examples/operator.yaml
@@ -245,3 +245,8 @@ spec:
             - name: DISCOVER_DAEMON_UDEV_BLACKLIST
               value: "(?i)dm-[0-9]+,(?i)rbd[0-9]+,(?i)nbd[0-9]+"
 
+            # The password the operator authenticates to the external
+            # RGW admin API with.
+            - name: RGW_ADMIN_PASSWORD
+              value: "hunter2"
+
@@ -288,3 +293,10 @@ spec:
             - name: ROOK_OBC_PROVISIONER_NAME
               value: "rook-ceph.ceph.rook.io/bucket"
 
+            # The keystone service-user password forwarded to the
+            # gateways.
+            - name: RGW_KEYSTONE_SERVICE_PASSWORD
+              valueFrom:
+                secretKeyRef:
+                  name: rook-ceph-keystone-service-user
+                  key: password
```

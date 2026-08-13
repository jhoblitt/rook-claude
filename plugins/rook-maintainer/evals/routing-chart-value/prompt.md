There is no rook checkout, no
network, no `gh`, and subagents cannot be spawned in this environment:
review the following branch inline per the rook-code-review skill's
review spine, verifying what the diff and commit message themselves
allow and labeling everything else INFERENCE.

This is a branch target, not a PR: no pull request has been opened for
it, so there is no PR description, no CI status, and no review thread to
consult.

Your entire final answer is the review report — the routed-reference
list from spine step 1, verdict line, findings in the skill's finding
contract, and the audited-and-clean section. Nothing else.

Commit message (the only commit on the branch):

```text
helm: add the keystone service password as a chart value

Configuring keystone meant hand-editing rook-ceph-operator-config after
every chart upgrade. The password is now a chart value, templated into
the operator ConfigMap alongside the other operator settings.
```

```diff
--- a/deploy/charts/rook-ceph/values.yaml
+++ b/deploy/charts/rook-ceph/values.yaml
@@ -59,3 +59,7 @@ logLevel: INFO
 
 # -- If true, create & use RBAC resources
 rbacEnable: true
+
+# -- Password for the keystone service account the RGW daemons
+# authenticate as
+keystonePassword: ""
--- a/deploy/charts/rook-ceph/templates/configmap.yaml
+++ b/deploy/charts/rook-ceph/templates/configmap.yaml
@@ -25,3 +25,6 @@ data:
   ROOK_CEPH_MON_RUN_AS_ROOT: {{ .Values.monRunAsRoot | quote }}
   ROOK_DELETE_UNUSED_CRUSH_RULES: {{ .Values.deleteUnusedCrushRules | quote }}
   ROOK_ENABLE_DISCOVERY_DAEMON: {{ .Values.enableDiscoveryDaemon | quote }}
+  {{- with .Values.keystonePassword }}
+  ROOK_KEYSTONE_PASSWORD: {{ . | quote }}
+  {{- end }}
```

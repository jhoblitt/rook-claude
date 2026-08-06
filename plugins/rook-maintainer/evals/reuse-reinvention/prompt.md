Create a scratch directory and write exactly this two-file Go module into
it, preserving the directory structure.

`go.mod`:

```text
module github.com/rook/rook

go 1.26
```

`pkg/operator/k8sutil/labels.go`:

```text
package k8sutil

type PodSpec struct {
	Labels map[string]string
}

// AddLabelToPod sets a label on the pod, creating the map when absent.
func AddLabelToPod(pod *PodSpec, key, value string) {
	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[key] = value
}
```

Then review the following diff against that module per the rook-code-review
skill's review spine. There is no network and no `gh`, and subagents cannot
be spawned in this environment: run the passes inline.

Your entire final answer is the review report — verdict line, findings in
the skill's finding contract, and the audited-and-clean statement. Nothing
else.

```diff
--- /dev/null
+++ b/pkg/operator/ceph/object/labels.go
@@ -0,0 +1,10 @@
+package object
+
+import "github.com/rook/rook/pkg/operator/k8sutil"
+
+// addLabelToPod sets a label on the pod, creating the map if it is nil.
+func addLabelToPod(pod *k8sutil.PodSpec, key, value string) {
+	if pod.Labels == nil {
+		pod.Labels = map[string]string{}
+	}
+	pod.Labels[key] = value
+}
```

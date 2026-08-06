Create a scratch directory and write exactly this two-file Go module into
it, preserving the directory structure.

`go.mod`:

```text
module github.com/rook/rook

go 1.26
```

`pkg/operator/ceph/pool/controller.go`:

```text
package pool

import "fmt"

type Client interface {
	GetPool(name string) (string, error)
	ApplyPool(spec string) error
}

type PoolReconciler struct{ client Client }

func (r *PoolReconciler) Reconcile(name string) error {
	spec, err := r.client.GetPool(name)
	if err != nil {
		return fmt.Errorf("failed to get pool %q: %w", name, err)
	}
	if err := r.client.ApplyPool(spec); err != nil {
		return fmt.Errorf("failed to apply pool %q: %w", name, err)
	}
	return nil
}
```

The fixture deliberately has no third-party imports: the module must resolve
with no network. It uses `fmt.Errorf`/`%w` consistently across both the
existing file and the added one, so go-review.md's match-the-package error
idiom rule produces no finding either way.

Then review the following diff against that module per the rook-code-review
skill's review spine. There is no network and no `gh`, and subagents cannot
be spawned in this environment: run the passes inline.

Your entire final answer is the review report — verdict line, findings in
the skill's finding contract, and the audited-and-clean statement. Nothing
else.

```diff
--- /dev/null
+++ b/pkg/operator/ceph/filesystem/controller.go
@@ -0,0 +1,22 @@
+package filesystem
+
+import "fmt"
+
+type Client interface {
+	GetFilesystem(name string) (string, error)
+	ApplyFilesystem(spec string) error
+}
+
+type FilesystemReconciler struct{ client Client }
+
+func (r *FilesystemReconciler) Reconcile(name string) error {
+	spec, err := r.client.GetFilesystem(name)
+	if err != nil {
+		return fmt.Errorf("failed to get filesystem %q: %w", name, err)
+	}
+	if err := r.client.ApplyFilesystem(spec); err != nil {
+		return fmt.Errorf("failed to apply filesystem %q: %w", name, err)
+	}
+	return nil
+}
```

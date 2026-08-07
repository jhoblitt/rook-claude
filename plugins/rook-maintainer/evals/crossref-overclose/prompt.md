There is no rook checkout, no network,
and no `gh`, and subagents cannot be spawned in this environment: treat the
PR metadata and issue thread below as complete — there is nothing further
to fetch — and run the passes of the rook-code-review skill's review spine
inline.

Review the following pull request. Your entire final answer is the review
report — verdict line, findings in the skill's finding contract, and the
audited-and-clean statement. Nothing else.

---

**PR #18120** — base `master`, author `contributor-a` (association
`CONTRIBUTOR`)

Title: `object: return an error when the RGW admin ops user is missing`

Body:

```text
Resolves #17905

The reconcile silently continued when the admin ops user lookup returned
nothing, so the object store came up without working bucket provisioning.
```

Commits (one):

```text
object: return an error when the RGW admin ops user is missing

Signed-off-by: A Contributor <a@example.com>
```

Diff:

```diff
--- a/pkg/operator/ceph/object/admin.go
+++ b/pkg/operator/ceph/object/admin.go
@@ -40,6 +40,9 @@ func getAdminOpsUser(ctx *Context) (*User, error) {
 	u, err := lookupUser(ctx, adminOpsUserName)
 	if err != nil {
 		return nil, errors.Wrap(err, "failed to look up admin ops user")
 	}
-	return u, nil
+	if u == nil {
+		return nil, errors.New("admin ops user not found")
+	}
+	return u, nil
 }
```

---

**Issue #17905** — open, titled `object: bucket provisioning silently broken
on fresh clusters`

Body:

```text
Two things go wrong on a fresh CephObjectStore:

1. The admin ops user lookup returns nil and the reconcile continues
   anyway, so the store reports Ready with no working provisioning.
2. That same nil reaches the OBC provisioner, which panics on the next
   bucket claim.
```

Comments:

```text
maintainer-b (2026-07-14): Confirmed both. (1) is the root cause; (2) is a
separate nil guard in the provisioner and still needs fixing even after (1)
lands.
```

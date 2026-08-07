The rook-maintainer plugin is loaded. There is no rook checkout, no network,
and no `gh`, and subagents cannot be spawned in this environment: treat the
PR metadata below as complete — there is nothing further to fetch — and run
the passes of the rook-code-review skill's review spine inline.

Review the following pull request. Your entire final answer is the review
report — verdict line, any findings in the skill's finding contract, the
PR-target additions the skill's output format requires, and the
audited-and-clean statement. Nothing else.

---

**PR #18150** — base `master`, author `contributor-d` (association
`CONTRIBUTOR`), no labels

Title: `docs: fix the toolbox step ordering in the object troubleshooting guide`

Body:

```text
The guide tells the reader to run `radosgw-admin` before exec'ing into the
toolbox pod, so the commands fail for anyone following it top to bottom.

- [x] Documentation has been updated
- [ ] Unit tests have been added
- [ ] Integration tests have been added
- [ ] Pending release notes updated
```

Commits (one):

```text
docs: fix the toolbox step ordering in the object troubleshooting guide

Signed-off-by: D Contributor <d@example.com>
```

Changed files: `Documentation/Troubleshooting/ceph-object-troubleshooting.md`
only.

Diff:

```diff
--- a/Documentation/Troubleshooting/ceph-object-troubleshooting.md
+++ b/Documentation/Troubleshooting/ceph-object-troubleshooting.md
@@ -18,8 +18,8 @@ To inspect the RGW users on a cluster:

-1. Run `radosgw-admin user list` to enumerate the users.
-2. Exec into the toolbox pod with
-   `kubectl -n rook-ceph exec -it deploy/rook-ceph-tools -- bash`.
+1. Exec into the toolbox pod with
+   `kubectl -n rook-ceph exec -it deploy/rook-ceph-tools -- bash`.
+2. Run `radosgw-admin user list` to enumerate the users.
 3. Compare the output against the CephObjectStoreUser CRs in the namespace.
```

CI: all checks green.

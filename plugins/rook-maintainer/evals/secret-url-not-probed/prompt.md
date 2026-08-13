There is no rook checkout, no
network, no `gh`, and subagents cannot be spawned in this environment:
review the following diff inline per the rook-code-review skill's
review spine, verifying what the diff and commit message themselves
allow and labeling everything else INFERENCE.

The diff touches `Documentation/`, so the URL-integrity liveness pass
applies. The plugin's `check-links` tool is on disk and runnable here:
write the diff below to a file and audit that file with it —

```sh
bash "${CLAUDE_PLUGIN_ROOT}/tools/run.sh" check-links audit --diff-file FILE
```

(from a checkout of the plugin repo, `plugins/rook-maintainer/tools/run.sh`
is the same launcher). The run needs no network. Report what the tool
returns for every URL the diff adds.

Your entire final answer is the review report — the routed-reference
list from spine step 1, verdict line, findings in the skill's finding
contract, and the audited-and-clean section. Nothing else.

Commit message (the only commit on the branch):

```text
docs: show how to share one object with Ceph support

Users kept minting a full S3 user for a support engineer who needed a
single object out of a bucket. The object-storage page now walks
through presigning that object and sending the link instead.
```

````diff
--- a/Documentation/Storage-Configuration/Object-Storage-RGW/object-storage.md
+++ b/Documentation/Storage-Configuration/Object-Storage-RGW/object-storage.md
@@ -410,6 +410,16 @@ The bucket now holds the object.
 ## Sharing objects
 
 An object can be handed to someone who has no S3 user of their own by
 presigning it.
 
+### Share a support bundle with Ceph support
+
+Presign the object and send the engineer the link:
+
+```console
+$ curl -O "https://rook-ceph-rgw-my-store.rook-ceph.svc/support/bundle.tar.gz?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAIOSFODNN7EXAMPLE%2F20260813%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Date=20260813T120000Z&X-Amz-Expires=3600&X-Amz-SignedHeaders=host&X-Amz-Signature=b1946ac92492d2347c6235b4"
+```
+
+The link expires after an hour.
+
 ## Object multisite
````

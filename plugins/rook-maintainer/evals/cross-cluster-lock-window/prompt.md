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
osd: serialize CRUSH map edits for the whole get/edit/set cycle

Two reconciles editing the CRUSH map at once each read the map, apply
their own change, and write it back, so the second `setcrushmap`
silently drops the first's rule. Hold the package mutex across the
whole cycle instead of only around the crushtool scratch files.
```

```diff
--- a/pkg/daemon/ceph/client/crush.go
+++ b/pkg/daemon/ceph/client/crush.go
@@ -36,8 +36,9 @@ const (
 	crushCompiledPath = "/tmp/crush.compiled"
 )
 
-// crushMapMutex guards the crushtool scratch files, which every
-// compile in the process shares.
+// crushMapMutex serializes the whole get/edit/set cycle: two reconciles
+// editing the map at once each write back a map that lacks the other's
+// change, and the second write wins.
 var crushMapMutex sync.Mutex
 
 // CrushMap is the decoded output of `ceph osd crush dump`.
@@ -118,8 +119,6 @@ func decompileCrushMap(context *clusterd.Context, compiled []byte) (string, error) {
 }
 
 func compileCrushMap(context *clusterd.Context, decompiled string) ([]byte, error) {
-	crushMapMutex.Lock()
-	defer crushMapMutex.Unlock()
 	if err := os.WriteFile(crushDecompiledPath, []byte(decompiled), 0o600); err != nil {
 		return nil, errors.Wrap(err, "failed to write the decompiled crush map")
 	}
@@ -160,6 +159,9 @@ func compileCrushMap(context *clusterd.Context, decompiled string) ([]byte, error) {
 // UpdateCrushMap fetches the cluster's CRUSH map, applies edit to its
 // decompiled text, recompiles it, and sets it on the cluster.
 func UpdateCrushMap(context *clusterd.Context, clusterInfo *ClusterInfo, edit func(decompiled string) (string, error)) error {
+	crushMapMutex.Lock()
+	defer crushMapMutex.Unlock()
+
 	compiled, err := getCrushMap(context, clusterInfo)
 	if err != nil {
 		return errors.Wrapf(err, "failed to get the crush map of cluster %q", clusterInfo.Namespace)
```

Unchanged code the hunks depend on, as the file has it: `getCrushMap`
runs `ceph osd getcrushmap` and `setCrushMap` runs
`ceph osd setcrushmap -i <file>` against `clusterInfo`'s mons through
`NewCephCommand(context, clusterInfo, ...).Run()`, each bounded by the
default mon command timeout; `decompileCrushMap` and `compileCrushMap`
exec `crushtool` locally against the scratch files. The rest of
`UpdateCrushMap`'s body is decompile, `edit`, `compileCrushMap`,
`setCrushMap`, with no other lock. It is called from the OSD reconcile
of every CephCluster the operator manages.

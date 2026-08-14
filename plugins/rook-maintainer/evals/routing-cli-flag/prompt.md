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
operator: accept the auth client credentials as operator settings

The operator now authenticates to the external auth provider itself.
The client ID and the client secret arrive as operator flags, so both
can be set from the operator ConfigMap like every other setting.
```

```diff
--- a/cmd/rook/ceph/operator.go
+++ b/cmd/rook/ceph/operator.go
@@ -43,5 +43,8 @@ func init() {
 	operatorCmd.Flags().BoolVar(&operator.EnableMachineDisruptionBudget, "enable-machine-disruption-budget", false, "enable fencing controllers")
 
+	operatorCmd.Flags().StringVar(&operator.ClientID, "client-id", "", "auth client ID")
+	operatorCmd.Flags().StringVar(&operator.ClientSecret, "client-secret", "", "auth client secret")
+
 	flags.SetFlagsFromEnv(operatorCmd.Flags(), rook.RookEnvVarPrefix)
 	operatorCmd.Flags().AddGoFlagSet(flag.CommandLine)
 	if err := operatorCmd.Flags().Parse(nil); err != nil {
```

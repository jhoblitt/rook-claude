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
object: log which TLS secret the gateway mounts

A store whose sslCertificateRef pointed at the wrong Secret took an
hour to diagnose: nothing in the operator log said which Secret was
mounted, or which CA the gateway would present.
```

```diff
--- a/pkg/operator/ceph/object/spec.go
+++ b/pkg/operator/ceph/object/spec.go
@@ -418,8 +418,11 @@ func (c *clusterConfig) generateGatewayTLS() error {
 	secret, err := c.context.Clientset.CoreV1().Secrets(c.store.Namespace).
 		Get(c.clusterInfo.Context, c.store.Spec.Gateway.SSLCertificateRef, metav1.GetOptions{})
 	if err != nil {
 		return errors.Wrapf(err, "failed to get secret %q", c.store.Spec.Gateway.SSLCertificateRef)
 	}
 
+	logger.Infof("using secret %s", secret.Name)
+	logger.Debugf("ca: %s", secret.Data["tls.crt"])
+
 	return c.mountTLS(secret)
 }
```

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
object: roll the gateway when its keyring changes

RGW pods kept serving with a stale keyring after a rotation, because
nothing in the pod template changed. The template now carries a
checksum of the keyring, so a rotation rolls the deployment, and the
gateway's public key is logged for support.
```

```diff
--- a/pkg/operator/ceph/object/spec.go
+++ b/pkg/operator/ceph/object/spec.go
@@ -288,12 +288,23 @@ func (c *clusterConfig) makeRGWPodSpec(rgwConfig *rgwConfig) (v1.PodTemplateSpec, error) {
 	secret, err := c.keyringSecret(rgwConfig)
 	if err != nil {
 		return v1.PodTemplateSpec{}, err
 	}
 	tlsSecret, err := c.tlsSecret()
 	if err != nil {
 		return v1.PodTemplateSpec{}, err
 	}
 	podTemplateSpec := c.basePodTemplateSpec(rgwConfig)
 
+	logger.Debugf("key=%s", base64.StdEncoding.EncodeToString(secret.Data["key"]))
+
+	podTemplateSpec.Annotations["rgw-keyring-checksum"] =
+		fmt.Sprintf("%x", sha256.Sum256(secret.Data["key"]))
+
+	pubKeyPEM, err := publicKeyFromPrivate(tlsSecret.Data["tls.key"])
+	if err != nil {
+		return v1.PodTemplateSpec{}, errors.Wrap(err, "failed to derive the public key")
+	}
+	logger.Infof("pub=%s", pubKeyPEM)
+
 	return podTemplateSpec, nil
 }
```

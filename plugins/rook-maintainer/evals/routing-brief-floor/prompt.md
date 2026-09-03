There is no rook checkout, no
network, no `gh`, and subagents cannot be spawned in this environment:
you are the gap-sweep agent an orchestrating session running the
rook-code-review skill launched per spine step 4 — a general-purpose
stand-in for `rook-reviewer`, carrying that agent file's contract
(`agents/rook-reviewer.md` under the plugin root) — and you run the
sweep inline, verifying what the diff and commit message themselves
allow and labeling everything else INFERENCE.

Your entire final answer is the sweep's return: the `references_read`
list, every candidate finding in the skill's finding contract, and the
`clean` list. Nothing else.

The orchestrator's brief follows verbatim, then the inputs it names.

---

Brief:

```text
Gap sweep for branch `multus-client-securitycontext` (read-only
checkout). Your inputs are the diff, the surviving findings, and the
audited-and-clean list below.

Read these references before hunting:
  references/naming-and-comments.md
  references/security.md
  references/reuse.md
  references/cross-references.md
  references/verification.md

Give particular attention to areas the orchestrator has not swept —
the five references above. Go idiom and modernization are already
covered by the orchestrator's own pass (see the audited-and-clean
list), so do not spend the sweep on go-review.md.

Return references_read, your candidates, and clean.
```

Surviving findings:

```text
C1/test-coverage pkg/daemon/multus/resources.go:83 — the client pod's
  security context has no test asserting it.
  failure: a later edit drops the context; nothing fails
  fix: assert the client pod's SecurityContext in resources_test.go
  confidence: CONFIRMED (85)
```

Audited and clean:

```text
- go-review.md: added lines are modern Go; no archaic construct on the
  diff.
- security.md: no credential material, no new sink.
- naming-and-comments.md: names match the package's convention.
```

Commit message (the only commit on the branch):

```text
multus: run the validation client pod with a restricted security context

The validation tool's client pod ran as root on a writable root
filesystem for no reason the tool needs. Set the restricted baseline.
```

```diff
--- a/pkg/daemon/multus/resources.go
+++ b/pkg/daemon/multus/resources.go
@@ -21,6 +21,7 @@ import (
 	"github.com/pkg/errors"
 	core "k8s.io/api/core/v1"
 	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
+	"k8s.io/utils/ptr"
 )
 
 func (vt *ValidationTest) generateClientPod(nodeName string) *core.Pod {
@@ -78,7 +79,13 @@ func (vt *ValidationTest) generateClientPod(nodeName string) *core.Pod {
 			Containers: []core.Container{{
 				Name:    "client",
 				Image:   vt.RookImage,
 				Command: []string{"rook", "multus", "validation", "client"},
+				SecurityContext: &core.SecurityContext{
+					RunAsNonRoot:             ptr.To(true),
+					RunAsUser:                ptr.To(int64(2016)),
+					ReadOnlyRootFilesystem:   ptr.To(true),
+					AllowPrivilegeEscalation: ptr.To(false),
+				},
 			}},
 		},
 	}
```

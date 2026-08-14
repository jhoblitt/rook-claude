# Credential Handling Canon Implementation Plan

> **Status: EXECUTED 2026-08-13.** All ten tasks are implemented on this
> branch; do not re-run this plan. Progress was tracked in the SDD ledger,
> not these checkboxes, and the fenced text and code blocks below are the
> pre-review drafts — the shipped files differ where review fix rounds
> amended them (e.g. `credential.go`'s sorted-key reason). This document
> is the historical record of how the work was specified.
>
> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `docs/superpowers/specs/2026-08-11-credential-handling-design.md` — the credential-material definition, the two credential rules, the cap exemption, always-load routing, the check-links credential filter, and the 25 evals that pin all of it.

**Architecture:** `references/security.md` becomes the sole normative home and sole generator of credential findings; every other touched file carries a pointer, a routing row, or a mechanical filter. Prose edits land file-by-file in canon voice (imperative reviewer instructions — never spec-voice "the canon will say"). The Go filter is TDD'd in `internal/links` and wired into `check-links`. Evals are hermetic prompt+grader pairs in the documented layout.

**Tech Stack:** Markdown canon (markdownlint-cli2@0.18.1), Go 1.x tools module (`plugins/rook-maintainer/tools`), `claude plugin eval` layout (`evals/<case>/prompt.md` + `graders/criteria.md`).

## Global Constraints

- **Never touch `plugins/rook-maintainer/.claude-plugin/plugin.json`** — semantic-release owns the version.
- Every commit message passes commitlint (`@commitlint/config-conventional`); scope is `rook-code-review`.
- All Markdown passes `npx --yes markdownlint-cli2@0.18.1` run from the repo root (config: `.markdownlint-cli2.jsonc`). Prefix npx with `npm_config_cache=$TMPDIR/npm-cache`. **Local runs only**: the repo glob hits the sandbox-protected `.claude/loop.md` (EACCES); run with `--config <tmpdir>/local.markdownlint-cli2.jsonc` — a copy of the repo config whose globs add `!.claude/**` — or lint the changed files by explicit path. CI runs the bare command on a clean checkout and is unaffected.
- Go changes pass, from `plugins/rook-maintainer/tools/`: `gofmt -l .` (empty), `go vet ./...`, `go test ./...`.
- **Canon voice**: canon files state rules imperatively to the reviewer. No meta-commentary ("this spec adds…"), no grounding paragraphs, no RFC-citation essays beyond what the spec text itself embeds.
- **One normative home**: pointers name the rule and its home (`security.md`'s section headings) and never restate the rule. Cross-file references cite **section headings**, never line numbers.
- The sentence "The test is what the value **is**, never what it is named" must survive verbatim in `security.md` — it is the spec's designated load-bearing sentence.
- Working branch: `worktree-secret-provenance-spec` in this worktree; push with `git push --force-with-lease origin HEAD:docs/secret-definition-spec` only when the user asks — task commits are local.

## Locked interface names (used across tasks)

- `security.md` section headings: `## What counts as a secret`, `## Credential storage contracts`, `## Credential-finding precedence`.
- check-links skip verdict string: `skipped-credential`.
- Go API: `links.CredentialBearing(raw string) (bool, string)` and `links.PartitionCredential(urls []string) (skips []Result, probe []string)`.

---

### Task 1: Rewrite `references/security.md` — the normative home

**Files:**

- Modify: `plugins/rook-maintainer/skills/rook-code-review/references/security.md`

**Interfaces:**

- Consumes: nothing (first task).
- Produces: the three section headings listed in Locked interface names, cited by Tasks 2, 3, and 5. The secret-leak bullet keyed on "secret-tainted value". Rule identifiers "(a)" and "(b)" inside `## Credential storage contracts`.

- [ ] **Step 1: Apply the edit**

The file keeps its title, its `## Vulnerability classes (rook-specific)` section (with two edits inside, below), the false-positive-discipline paragraph, and the whole `## Contribution-trust signals` section unchanged. Insert the three new sections between the intro and `## Vulnerability classes`. Replace the intro's first sentence and the secret-leak bullet as shown.

New intro (replaces lines 3–6):

```markdown
Two passes: (1) vulnerability classes in the diff, (2) contribution-trust
signals about where the change comes from and what surfaces it touches. Keep
them separate in the report; the second is NEVER an accusation — it is a
prioritization signal for human scrutiny. The credential canon below defines
the terms the vulnerability pass uses, and is the sole generator of
credential findings: the pointers in `kubernetes-crd.md` and
`ceph-object.md` create awareness, never findings.
```

New section 1 (insert after the intro):

```markdown
## What counts as a secret

**Credential material** is a value whose confidentiality is its purpose — a
password, bearer token, private key, shared secret, or recovery seed — and
the **identifying half** of an authentication credential: the username,
account, client ID, or access key ID that such a secret authenticates. It
is NOT an identifier rook manages and reconciles (a `CephObjectStoreUser`
name, a bucket, a pool), nor a public certificate, Secret name,
credential-free endpoint, or tuning knob.

The test is what the value **is**, never what it is named. A field called
`password` holding a feature flag is not credential material; a field
called `token` holding a bucket name is not either.

The identifying half is **project-protected**, not secret in itself (RFC
6749 §2.2: a client identifier "is not a secret"); rook protects it
uniformly rather than per-protocol. Severity keeps the terminology honest:
a leaked secret half is SKILL.md's blocker "secret leak"; a leak of only
the identifying half is changes-requested `security` — protected, not
secret, so the report never claims "secret leak" for it.

**A URL that carries credential material IS credential material — the
whole URL, as one value.** A URL can carry a credential in any component:
userinfo (`https://user:pass@host`), a query parameter
(`?access_token=`), a fragment (OAuth implicit-flow tokens), or a path
segment carrying a bearer capability. A signature qualifies when it
**authorizes** — a presigned URL is credential material; a public
integrity signature (a detached artifact signature) grants nothing and
does not qualify. Expiry is not a defense: logs outlive the validity
window, and rook's CI logs are public. Never split such a URL into secret
and non-secret parts — the whole URL is handled as the secret. Stripping
the credential yields a URL that carries none and is not credential
material. `check-links` skips these URLs instead of probing them — a
probe would exercise the capability (docs-sync.md).

A value is **secret-tainted** when it comes from:

1. a secret-bearing payload field of a k8s Secret read;
2. the credential fields of a credential-bearing admin/API response —
   issuance or retrieval alike: `radosgw-admin user create` or `user
   info` output, a `GetUser` result, a minted CephX key, a keyring read;
3. an environment variable carrying secret material;
4. a hard-coded literal credential;
5. a file read from a host or PVC path carrying key material;
6. any field, input, or parameter carrying credential material, wherever
   it is declared — a CR spec field, chart value, operator config key,
   CLI flag, HTTP request field, or annotation;
7. any value derived from sources 1–6.

Seeding is per-field, never per-object. A Secret read taints
`.Data["password"]`, not `secret.Name` or its metadata; an admin/API
response taints the returned key, not the user ID or display name. A
payload field whose content is public by construction — `tls.crt`,
`ca.crt`, a public key — is not seeded. What the containing object is
never decides secrecy; the field does.

Derived values stay tainted, through base64, hex, and any other encoding
change, and through assembly into a URL. The exceptions are values
designed for disclosure that no longer carry the secret: a public key
derived from a private one, or a URL with its credential stripped.
One-way is not disclosure-safe — a digest, MAC, or ciphertext of
credential material stays tainted; a checksum-of-Secret pod annotation is
a finding, not an exception.

Not tainted: a field that does not carry credential material, whatever it
is called; Secret names and object metadata; public-by-construction
payload fields.
```

New section 2 (immediately after section 1):

```markdown
## Credential storage contracts

**(a) A credential-bearing storage contract must reference a k8s Secret,
not hold a plaintext value.** A storage contract is user-authored,
persistent, declarative configuration outside a designated secret store: a
CRD spec field, a chart value, an operator config key. It is NOT a value
in flight — a credential written into a `Secret` payload, held in a
response struct, or passed by stdin, env, or file is correct (the argv
canon below prefers exactly those transports). Serialization does not
decide this: `Secret.Data` is serialized and is the right destination.

Declaration and transport are judged separately. That a credential
legitimately travels by env var or file does not bless the persistent
declaration behind it: an inline `env.value:` in a manifest, or a
plaintext credential file on a PVC, is a storage contract and must be a
Secret reference (`valueFrom.secretKeyRef`, a projected Secret volume);
only the runtime hop is transport.

Applies to a storage contract the diff INTRODUCES — a new field, an
existing generic field the diff repurposes to carry a credential, or a
new credential-bearing key in an existing map. Not to a legacy contract
the diff merely touches — the same "adds or whose handling it changes"
line kubernetes-crd.md draws for unset-field semantics.

**(b) A literal credential value committed to the tree is a finding
regardless of the contract's age** — a CR spec, example manifest, chart
value, config file, or documentation page: the repo is public, so a live
value ships wherever it sits.

Severity: blocker, `security`. For (a), a CRD field's shape freezes on
merge (kubernetes-crd.md: never change a field's type or JSON tag), so a
plaintext shape becomes permanent; a chart value or operator config key
ships the invitation — the first user who populates it puts a live value
into world-readable stores, rook's operator config loader logs every
changed key and value, and startup logs flags with redaction limited to
`secret|keyring`. For (b), the diff ships a live credential.

Fix shape: for a newly declared field, a `*corev1.SecretKeySelector`,
matching the `AccessKeyRef`/`SecretKeyRef` and
`UserSecretRef`/`PasswordSecretRef` pairs in `pkg/apis/ceph.rook.io/v1`.
For a repurposed field or map key the type cannot change: preserve the
old contract's meaning and add a separate selector-shaped contract beside
it. Outside the Kubernetes API the equivalent is a Secret reference
rather than an inline value; the Go type is not the point.

For a URL that carries a credential, the whole URL is the secret: the
contract references it entire from a Secret — never a split (What counts
as a secret, above). Where rook itself signs, prefer storing the signing
credentials and minting the URL at runtime over persisting a signed URL
at all. A contract rook defines itself may still take the decomposed
shape — a credential-free URI beside secret references, the
`KafkaEndpointSpec` precedent; the no-split rule governs contracts whose
documented form already embeds the credential.

A reference is not automatically a boundary. Where the contract admits a
cross-namespace or caller-chosen reference, a tenant can point a
privileged operator at an arbitrary Secret: name the enforcement point
that rejects the selection — the namespace constraint, `resourceNames`
restriction, admission check, or controller check (architecture.md's
security-claims-are-traced canon; `AllowUsersInNamespaces` below names
the surface, not a rule).
```

New section 3 (immediately after section 2):

```markdown
## Credential-finding precedence

One site can satisfy both rules — a new chart value whose credential is
templated into the operator ConfigMap is a storage contract and a leak
sink. Fuse the candidates into one finding when the storage contract is
itself the only observable sink, anchored at the contract (the value's
declaration), citing the materialization in the failure text. Keep them
separate when a distinct sink — a log line, Event, or status write — also
needs remediation: the fixes differ. Clauses (a) and (b) firing on one
newly introduced value-bearing contract fuse the same way, into one
finding whose fix names both the selector shape and rotation of the
shipped value.

The design pass overlaps differently: a new credential contract also
fires architecture.md's decision-magnitude triggers. A design candidate
whose defect is the plaintext shape itself defers to the clause (a)
finding; a design concern about a distinct decision on the same field —
where the Secret should live, cross-namespace admission — stands on its
own.
```

Inside `## Vulnerability classes (rook-specific)`, two edits only:

Replace the secret-leak bullet's first sentence (current lines 18–19):

```markdown
- **Secret-leak check (first-class).** For every **secret-tainted value**
  (What counts as a secret, above) the diff touches, trace it into
  observable channels and flag any print:
```

The six channel sub-bullets stay byte-for-byte. Replace the closing
aphorism (current line 30–31):

```markdown
  Secret NAME in a log is fine; a secret-tainted value never is. Partial
  redaction is judged per channel.
```

- [ ] **Step 2: Verify**

Run from repo root:

```bash
npm_config_cache=$TMPDIR/npm-cache npx --yes markdownlint-cli2@0.18.1
grep -c '^## ' plugins/rook-maintainer/skills/rook-code-review/references/security.md   # expect 5
grep -n 'What counts as a secret\|Credential storage contracts\|Credential-finding precedence' plugins/rook-maintainer/skills/rook-code-review/references/security.md
grep -n 'never what it is named' plugins/rook-maintainer/skills/rook-code-review/references/security.md   # must hit
```

Expected: markdownlint clean; 5 `##` headings; all three new headings present; the load-bearing sentence present.

- [ ] **Step 3: Commit**

```bash
git add plugins/rook-maintainer/skills/rook-code-review/references/security.md
git commit -m "feat(rook-code-review): define credential material and key the leak check on provenance"
```

---

### Task 2: Always-load routing — `SKILL.md` + `agents/rook-reviewer.md`

**Files:**

- Modify: `plugins/rook-maintainer/skills/rook-code-review/SKILL.md` (routing table rows 197–198; Scripts `check-links` entry at 229–233)
- Modify: `plugins/rook-maintainer/agents/rook-reviewer.md:17-18`

**Interfaces:**

- Consumes: `security.md` exists with the Task 1 headings (routing row text names no headings, so no hard dependency).
- Produces: the routing row `any diff-shaped target → references/security.md` that eval Tasks 6–9 assert; Scripts wording naming `skipped-credential` behavior consumed by Task 5's docs.

- [ ] **Step 1: Edit the routing table**

In the `## Reference routing` table:

1. Row currently reading
   `| .github/workflows/**, .mergify.yml, scripts run by workflows | references/github-actions.md + references/security.md |`
   → drop the security half:
   `| .github/workflows/**, .mergify.yml, scripts run by workflows | references/github-actions.md |`
2. Delete the row
   `| go.mod, build/, Dockerfiles, tests/scripts/, exec call sites, TLS, secrets, RBAC | references/security.md |`
   outright (its only target was `security.md`).
3. Insert, directly above the `| always, before reporting | references/verification.md |` row:
   `| any diff-shaped target | references/security.md |`

- [ ] **Step 2: Edit the Scripts entry**

Replace the `check-links` bullet's first sentence so it reads:

```markdown
- `check-links` — liveness of every URL the diff adds, minus
  credential-material skips (reported as `skipped-credential`, never
  probed — a probe would exercise the capability; `references/security.md`
  "What counts as a secret"), plus the control/format-character scan on
  those URLs. Replaces WebFetch for
  this pass entirely: it returns a status code and no page content, which is
  what makes diff-chosen hosts safe to probe. Spec:
  `references/docs-sync.md`.
```

- [ ] **Step 3: Edit the agent file**

`plugins/rook-maintainer/agents/rook-reviewer.md` lines 17–18, replace

```text
always including `verification.md` and `cross-references.md`, plus
`ci-triage.md` and `security.md` for PR targets. PR targets additionally
```

with

```text
always including `verification.md`, `cross-references.md`, and
`security.md`, plus `ci-triage.md` for PR targets. PR targets additionally
```

- [ ] **Step 4: Verify**

```bash
npm_config_cache=$TMPDIR/npm-cache npx --yes markdownlint-cli2@0.18.1
grep -n 'any diff-shaped target' plugins/rook-maintainer/skills/rook-code-review/SKILL.md
grep -c 'references/security.md' plugins/rook-maintainer/skills/rook-code-review/SKILL.md   # expect 2: routing row + Scripts entry
grep -n 'security.md' plugins/rook-maintainer/agents/rook-reviewer.md | head -3
```

- [ ] **Step 5: Commit**

```bash
git add plugins/rook-maintainer/skills/rook-code-review/SKILL.md plugins/rook-maintainer/agents/rook-reviewer.md
git commit -m "feat(rook-code-review): load the security canon for every diff-shaped target"
```

---

### Task 3: Non-generating pointers — `kubernetes-crd.md` + `ceph-object.md`

**Files:**

- Modify: `plugins/rook-maintainer/skills/rook-code-review/references/kubernetes-crd.md` (append one bullet to the controller-conventions list that ends at the "Chart parity" bullet, directly before `## Unset-field semantics`)
- Modify: `plugins/rook-maintainer/skills/rook-code-review/references/ceph-object.md:51-52`

**Interfaces:**

- Consumes: Task 1's section headings, cited by name.
- Produces: nothing downstream.

- [ ] **Step 1: kubernetes-crd.md pointer**

Append this bullet at the end of the list that precedes `## Unset-field semantics`:

```markdown
- **Credential fields**: a spec field carrying credential material takes a
  Secret reference, never a plaintext string — blocker. security.md's
  credential canon ("What counts as a secret" / "Credential storage
  contracts") is the sole generator and holds the taxonomy, the
  introduces-only bound, and the fix shapes; this bullet is awareness
  only and raises no findings.
```

- [ ] **Step 2: ceph-object.md rewrite**

Replace the bullet at lines 51–52:

```markdown
- Credentials come from Secrets; watch their journey per security.md's
  credential canon ("What counts as a secret"), the sole generator — this
  bullet raises no findings. Literal credentials in specs route through
  its storage-contract rule; consuming a legacy plaintext source is, by
  itself, no finding.
```

- [ ] **Step 3: Verify**

```bash
npm_config_cache=$TMPDIR/npm-cache npx --yes markdownlint-cli2@0.18.1
grep -n 'sole generator' plugins/rook-maintainer/skills/rook-code-review/references/kubernetes-crd.md plugins/rook-maintainer/skills/rook-code-review/references/ceph-object.md
```

Expected: one hit in each file.

- [ ] **Step 4: Commit**

```bash
git add plugins/rook-maintainer/skills/rook-code-review/references/kubernetes-crd.md plugins/rook-maintainer/skills/rook-code-review/references/ceph-object.md
git commit -m "refactor(rook-code-review): point crd and object credential canon at security.md"
```

---

### Task 4: Cap exemption — `architecture.md` + `proposal.md`

**Files:**

- Modify: `plugins/rook-maintainer/skills/rook-code-review/references/architecture.md:154-157`
- Modify: `plugins/rook-maintainer/skills/rook-code-review/references/proposal.md:179-180`

**Interfaces:**

- Consumes: nothing.
- Produces: the phrase "cap-exempt categories" in architecture.md, which proposal.md's pointer names; cap-family evals (Task 9) assert both behaviors.

- [ ] **Step 1: architecture.md**

In the `Caps, force-ranked` bullet, replace the closing sentence (current
lines 154–157, beginning "Needs-evidence enforcement concerns…") with:

```markdown
  Two cap-exempt categories report even when force-ranked out — the caps
  bound taste, never a blocking security premise: needs-evidence
  enforcement concerns (the security canon above), and a design finding
  whose CONFIRMED cost traces a concrete security consequence — the
  changed decision opens a named access or disclosure path, to an actor
  who should not have it, reaching a protected asset that actor can newly
  use or learn. PLAUSIBLE never exempts — vocabulary cannot make a cost
  traced — permitted behavior does not qualify, and questions are never
  exempt: the cap is their gate, and an untraceable security concern is
  either needs-evidence or it waits.
```

- [ ] **Step 2: proposal.md**

Replace lines 179–180's sentence

```text
Needs-evidence enforcement concerns are exempt from both cuts
(architecture.md).
```

with

```text
architecture.md's cap-exempt categories are exempt from both cuts.
```

- [ ] **Step 3: Verify**

```bash
npm_config_cache=$TMPDIR/npm-cache npx --yes markdownlint-cli2@0.18.1
grep -n 'cap-exempt categories' plugins/rook-maintainer/skills/rook-code-review/references/architecture.md plugins/rook-maintainer/skills/rook-code-review/references/proposal.md
```

Expected: one hit in each file (the definition and the pointer).

- [ ] **Step 4: Commit**

```bash
git add plugins/rook-maintainer/skills/rook-code-review/references/architecture.md plugins/rook-maintainer/skills/rook-code-review/references/proposal.md
git commit -m "feat(rook-code-review): exempt traced security consequences from the design cap"
```

---

### Task 5: check-links credential filter — Go + `docs-sync.md`

**Files:**

- Create: `plugins/rook-maintainer/tools/internal/links/credential.go`
- Create: `plugins/rook-maintainer/tools/internal/links/credential_test.go`
- Modify: `plugins/rook-maintainer/tools/cmd/check-links/main.go` (audit/check path)
- Modify: `plugins/rook-maintainer/skills/rook-code-review/references/docs-sync.md` (Liveness bullet)

**Interfaces:**

- Consumes: existing `links.Result`, `links.Sanitize`, `links.HasHiddenRunes`, `links.NewProber`.
- Produces: `CredentialBearing(raw string) (bool, string)`, `PartitionCredential(urls []string) ([]Result, []string)`, verdict string `skipped-credential` (must NOT be `Bad()`), consumed by eval `secret-url-not-probed` (Task 6) and the Task 2 Scripts wording.

- [ ] **Step 1: Write the failing test**

`credential_test.go`:

```go
package links

import "testing"

func TestCredentialBearing(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://user:hunter2@rgw.example.com/", true},
		{"https://rgw.example.com/b/o?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIA%2F20260813&X-Amz-Signature=abc123", true},
		{"https://rgw.example.com/o?AWSAccessKeyId=AKIAX&Signature=abc&Expires=1770000000", true},
		{"https://storage.example.com/c/b?sig=sv%3D2020-08-04", true},
		{"https://gitlab.example.com/api/v4/projects?private_token=glpat-xyz", true},
		{"https://app.example.com/cb#access_token=ya29.x&token_type=bearer", true},
		{"https://example.com/?SIGNATURE=abc", true},
		{"https://docs.ceph.com/en/latest/", false},
		{"https://example.com/release?signed=false&version=3", false},
		{"https://example.com/rook-v1.18.0.tar.gz.sig", false},
	}
	for _, c := range cases {
		got, reason := CredentialBearing(c.url)
		if got != c.want {
			t.Errorf("CredentialBearing(%q) = %v (%s), want %v", c.url, got, reason, c.want)
		}
		if got && reason == "" {
			t.Errorf("CredentialBearing(%q): true with empty reason", c.url)
		}
	}
}

func TestPartitionCredential(t *testing.T) {
	skips, probe := PartitionCredential([]string{
		"https://docs.ceph.com/en/latest/",
		"https://user:pw@h.example.com/",
	})
	if len(skips) != 1 || len(probe) != 1 {
		t.Fatalf("got %d skips, %d probe; want 1, 1", len(skips), len(probe))
	}
	if skips[0].Verdict != "skipped-credential" {
		t.Errorf("skip verdict = %q", skips[0].Verdict)
	}
	if skips[0].Bad() {
		t.Error("skipped-credential must not be Bad(): a skip is a report line, not a gate failure")
	}
	if probe[0] != "https://docs.ceph.com/en/latest/" {
		t.Errorf("probe[0] = %q", probe[0])
	}
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
cd plugins/rook-maintainer/tools && go test ./internal/links/
```

Expected: FAIL — `undefined: CredentialBearing`.

- [ ] **Step 3: Implement**

`credential.go`:

```go
package links

import (
	"net/url"
	"strings"
)

// credentialParams are query/fragment keys whose presence marks a URL as
// carrying credential material (security.md "What counts as a secret").
// Deliberately overcautious: a skipped probe costs one reviewer judgment,
// an exercised capability is unrecoverable. Shapes this list cannot see —
// a bare capability path segment, a public integrity signature parameter —
// are the reviewer's judgment per docs-sync.md.
var credentialParams = map[string]bool{
	"x-amz-signature": true, "x-amz-credential": true,
	"x-amz-security-token": true, "awsaccesskeyid": true,
	"signature": true, "x-goog-signature": true, "sig": true,
	"access_token": true, "id_token": true, "token": true,
	"api_key": true, "apikey": true, "private_token": true,
}

// CredentialBearing reports whether a URL's shape carries credential
// material, and why. Probing such a URL would exercise the capability it
// grants, so check-links skips it and reports the skip.
func CredentialBearing(raw string) (bool, string) {
	u, err := url.Parse(raw)
	if err != nil {
		return false, ""
	}
	if u.User != nil {
		return true, "userinfo"
	}
	for key := range u.Query() {
		if credentialParams[strings.ToLower(key)] {
			return true, "query parameter " + key
		}
	}
	if frag, err := url.ParseQuery(u.Fragment); err == nil {
		for key := range frag {
			if credentialParams[strings.ToLower(key)] {
				return true, "fragment parameter " + key
			}
		}
	}
	return false, ""
}

// PartitionCredential splits URLs into skip Results for credential-bearing
// shapes and the remainder to probe. The hidden-rune scan is retained for
// skipped URLs: an ASCII-smuggled credential URL is still suspicious.
func PartitionCredential(urls []string) ([]Result, []string) {
	var skips []Result
	var probe []string
	for _, u := range urls {
		bearing, reason := CredentialBearing(u)
		if !bearing {
			probe = append(probe, u)
			continue
		}
		r := Result{
			URL:     Sanitize(u),
			Verdict: "skipped-credential",
			Note:    "credential-material URL (" + reason + "): not probed",
		}
		if HasHiddenRunes(u) {
			r.Verdict = "suspicious"
			r.Note = "control or format characters inside URL"
		}
		skips = append(skips, r)
	}
	return skips, probe
}
```

Wire into `main.go`: in the `else` branch that currently probes (`mode ==
"check"` or `"audit"`), replace

```go
p := links.NewProber(*timeout, *allowPrivate)
results = p.CheckAll(context.Background(), urls, *workers)
```

with

```go
skips, probe := links.PartitionCredential(urls)
results = skips
if len(probe) > 0 {
	p := links.NewProber(*timeout, *allowPrivate)
	results = append(results, p.CheckAll(context.Background(), probe, *workers)...)
}
```

Also extend the package doc's mode list comment in `main.go` (the block
comment at top) with one line after the exit-status sentence:

```text
// URLs whose shape carries credential material (userinfo, signature or
// token parameters) are never probed — probing would exercise the
// capability. They report as skipped-credential, which does not gate.
```

- [ ] **Step 4: Run tests, vet, fmt**

```bash
cd plugins/rook-maintainer/tools && gofmt -l . && go vet ./... && go test ./...
```

Expected: gofmt output empty, vet clean, all tests PASS.

- [ ] **Step 5: Update docs-sync.md**

In the `## URL integrity (diff-scoped)` Liveness bullet, after the
sentence ending "so arbitrary diff-chosen hosts are safe to hit and no
per-link approval is spent.", insert:

```markdown
  URLs whose shape carries credential material — userinfo, signature or
  token parameters — are skipped, not probed: a probe would exercise a
  presigned URL's capability. Each skip reports as `skipped-credential`
  (not a gate failure), and the URL itself is already a finding per
  security.md "Credential storage contracts". Shapes the filter cannot
  classify — a bare capability path segment — are the reviewer's
  judgment per security.md "What counts as a secret".
```

- [ ] **Step 6: Verify and commit**

```bash
npm_config_cache=$TMPDIR/npm-cache npx --yes markdownlint-cli2@0.18.1
git add plugins/rook-maintainer/tools plugins/rook-maintainer/skills/rook-code-review/references/docs-sync.md
git commit -m "feat(rook-code-review): skip credential-material URLs in check-links"
```

---

## Eval tasks (6–9) — shared shape

Every eval is `plugins/rook-maintainer/evals/<case>/prompt.md` +
`plugins/rook-maintainer/evals/<case>/graders/criteria.md`, hermetic
(fixture diff embedded in the prompt; no rook checkout, no network, no
subagents), in the exact voice of the existing `unset-field-unmanaged`
case. Every prompt opens with the standard preamble and REQUIRES the
report to open with the routed-reference list from spine step 1 — that is
what lets negative-case graders verify `security.md` was loaded.

Standard prompt skeleton (fill the bracketed parts per eval; keep
everything else verbatim):

````markdown
There is no rook checkout, no
network, no `gh`, and subagents cannot be spawned in this environment:
review the following [diff|branch] inline per the rook-code-review
skill's review spine, verifying what the diff and commit message
themselves allow and labeling everything else INFERENCE.

Your entire final answer is the review report — the routed-reference
list from spine step 1, verdict line, findings in the skill's finding
contract, and the audited-and-clean section. Nothing else.

Commit message (the only commit on the branch):

```text
[commit message]
```

```diff
[fixture diff]
```
````

Standard criteria skeleton: a short factual paragraph stating what the
fixture does, then `Pass if and only if ALL of:` numbered assertions,
then `Fail if any of:` bullets. Always include these two fail bullets:
`- Any finding anchor is a bare basename or an elided path.` and
`- Subagents were spawned despite the stated no-subagent environment.`
Negative cases additionally pass-require:
`The routed-reference list names references/security.md.`

Fixture conventions: Go hunks anchor in
`pkg/operator/ceph/object/keystone.go` and CRD hunks in
`pkg/apis/ceph.rook.io/v1/types.go` unless the row says otherwise; chart
hunks in `deploy/charts/rook-ceph/values.yaml` and
`deploy/charts/rook-ceph/templates/configmap.yaml`; manifest hunks in
`deploy/examples/object.yaml`. Every fixture is 5–25 lines — the minimum
that makes the behavior undeniable.

---

### Task 6: Leak-family evals (11 cases)

**Files:** Create `prompt.md` + `graders/criteria.md` under
`plugins/rook-maintainer/evals/<case>/` for each case below.

**Interfaces:** Consumes Task 1 canon and Task 5's `skipped-credential`
verdict. Produces case directories Task 10 registers.

- [ ] **Step 1: Write the 11 cases** (fixture → pass-criteria core; build
  each with the shared skeletons):

| Case | Fixture (diff adds…) | Pass core |
|---|---|---|
| `secret-non-credential-field` | `logger.Infof("desired replicas=%d", cr.Spec.Replicas)` in a controller | NO leak finding on the log line; routed list names security.md |
| `secret-misnamed-field` | godoc-commented field `Password string // feature mode: "enabled"\|"disabled"` already exists; diff adds `logger.Infof("password mode=%s", cr.Spec.Password)` | NO leak finding — content, not name, decides; routed list names security.md |
| `secret-name-safe` | `logger.Infof("using secret %s", secret.Name)` and `logger.Debugf("ca: %s", secret.Data["tls.crt"])` | NO finding on either line (name + public-by-construction payload); routed list names security.md |
| `secret-derived-encoding` | `logger.Debugf("key=%s", base64.StdEncoding.EncodeToString(secret.Data["key"]))` plus a pod-template annotation set to `sha256(secret.Data["key"])`, plus `logger.Infof("pub=%s", pubKeyPEM)` derived from a private key | blocker findings on the base64 log AND the checksum annotation; NO finding on the public key |
| `secret-url-userinfo` | builds `u := fmt.Sprintf("https://%s:%s@%s/", user, pass, host)` then `logger.Infof("endpoint %s", u)`; second hunk logs `strings.Replace`-stripped URL | finding on the credential-bearing URL log; NO finding on the stripped-URL log |
| `secret-url-presigned` | logs a generated presigned URL (`X-Amz-Signature` in the string); second hunk logs `https://example.com/rook.tar.gz.sig` | finding on the presigned log; NO finding on the detached-signature artifact URL |
| `secret-url-not-probed` | a `Documentation/` page adding a literal presigned URL; prompt additionally instructs running the docs-sync liveness pass via `bash plugins/rook-maintainer/tools/run.sh check-links audit --diff-file <fixture>` | report carries a Rule 2(b) finding on the literal AND quotes a `skipped-credential` line from the tool output; grader checks the skip line, not a network non-event |
| `secret-legacy-field-newly-logged` | field `AdminPassword string` pre-exists (context lines only); diff adds `logger.Infof("keystone admin=%s pw=%s", u, cr.Spec.AdminPassword)` | blocker leak finding anchored on the added log line; NO storage-contract finding (legacy, untouched) |
| `secret-derived-from-field` | diff adds `logger.Debugf("tok=%s", base64.StdEncoding.EncodeToString([]byte(cr.Spec.AdminPassword)))` against the same pre-existing field | blocker leak finding — encoding does not launder a source-6 value |
| `secret-provenance-recall` | two hunks: `logger.Errorf("mon key %s rejected", string(secret.Data["mon-secret"]))` and a workflow step `run: echo "token=${API_TOKEN}"` with `env: API_TOKEN: ${{ secrets.API_TOKEN }}` | one finding per hunk, both blocker |
| `secret-api-response` | `logger.Debugf("created user: %+v", userInfo)` where `userInfo` is a `radosgw-admin user info` result struct with `Keys []UserKeySpec`; second hunk logs a keyring string read from a PVC path | finding on the `%+v` (keys inside) and on the keyring log |

- [ ] **Step 2: Verify**

```bash
npm_config_cache=$TMPDIR/npm-cache npx --yes markdownlint-cli2@0.18.1
ls plugins/rook-maintainer/evals/ | grep -c '^secret-'   # expect 11
```

- [ ] **Step 3: Commit**

```bash
git add plugins/rook-maintainer/evals/secret-*
git commit -m "test(rook-code-review): add the leak-family credential evals"
```

---

### Task 7: Contract-family evals (7 cases)

**Files:** Create the 7 case directories.

**Interfaces:** Consumes Task 1 canon. Produces case directories Task 10 registers.

- [ ] **Step 1: Write the 7 cases**

| Case | Fixture | Pass core |
|---|---|---|
| `credential-contract-new` | CRD hunk adding `KeystoneUser string` and `KeystonePassword string` spec fields | blocker whose fix names `*corev1.SecretKeySelector` (the `UserSecretRef`/`PasswordSecretRef` pair shape); BOTH fields in scope (identifying half counts) |
| `credential-contract-repurposed` | existing `Extra string` field (context); diff changes controller to `opts.Token = cr.Spec.Extra` with commit message saying Extra now carries the auth token | blocker; fix adds a separate selector contract, does NOT change the field's type |
| `credential-contract-legacy-untouched` | controller refactor hunk that reads pre-existing `cr.Spec.AdminPassword` into a renamed local; no new sink | NO storage-contract finding; routed list names security.md |
| `credential-value-in-flight` | hunk writing a minted key into `secret.StringData["access-key"]`; hunk passing a password via `cmd.Stdin` | NO contract finding on either; routed list names security.md |
| `credential-inline-env-value` | manifest hunk adding `env: - name: RGW_ADMIN_PASSWORD  value: hunter2`; second manifest hunk using `valueFrom.secretKeyRef` | finding on the inline `value:`; NO finding on `valueFrom` |
| `credential-literal-existing-field` | example-manifest hunk filling pre-existing `adminPassword:` field with a realistic live-looking value | Rule 2(b) blocker despite the contract being legacy |
| `credential-contract-url-field` | CRD hunk adding `Endpoint string` whose godoc documents `https://user:secret@host/` form | blocker; fix references the WHOLE URL from a Secret — a split proposal fails the eval |

- [ ] **Step 2: Verify**

```bash
npm_config_cache=$TMPDIR/npm-cache npx --yes markdownlint-cli2@0.18.1
ls plugins/rook-maintainer/evals/ | grep -c '^credential-'   # expect 7
```

- [ ] **Step 3: Commit**

```bash
git add plugins/rook-maintainer/evals/credential-*
git commit -m "test(rook-code-review): add the contract-family credential evals"
```

---

### Task 8: Routing and fusion evals (3 cases)

**Files:** Create the 3 case directories.

**Interfaces:** Consumes Task 2's routing row. Produces case directories Task 10 registers.

- [ ] **Step 1: Write the 3 cases**

| Case | Fixture | Pass core |
|---|---|---|
| `routing-chart-value` | chart-only diff adding `keystonePassword: ""` to values.yaml and templating it into the operator ConfigMap; prompt frames the target as a BRANCH, then as a WORKING TREE (two runs in one prompt is not possible — instead the prompt states "this is a branch target, not a PR") | finding produced although the target is not a PR; routed list names security.md — proving the always-load row, not the PR-agent rescue |
| `routing-cli-flag` | Go diff in `cmd/rook/` adding `flag.StringVar(&clientID, "client-id", "", "auth client ID")` | finding (identifying half, storage contract); routed list names security.md on an ordinary-Go-only diff |
| `same-site-fusion` | chart value + ConfigMap templating (one credential, contract-is-the-only-sink) in hunk set A; the same value ADDITIONALLY logged in hunk set B; prompt carries both labeled variants A and B and asks for findings per variant | variant A: exactly ONE fused finding anchored at the declaration; variant B: exactly TWO findings (contract + log sink) |

- [ ] **Step 2: Verify**

```bash
npm_config_cache=$TMPDIR/npm-cache npx --yes markdownlint-cli2@0.18.1
ls plugins/rook-maintainer/evals/ | grep -cE '^(routing-|same-site)'   # expect 3
```

- [ ] **Step 3: Commit**

```bash
git add plugins/rook-maintainer/evals/routing-* plugins/rook-maintainer/evals/same-site-fusion
git commit -m "test(rook-code-review): add the routing and fusion credential evals"
```

---

### Task 9: Cap-family evals (4 cases)

**Files:** Create the 4 case directories.

**Interfaces:** Consumes Task 4's cap-exempt categories. Produces case directories Task 10 registers.

These four are design/proposal-shaped: the prompt embeds a fixture diff
(or proposal doc for the overflow case) engineered to yield four design
candidates, and instructs the subject to run the design pass and report
under the caps. The fixture recipe for the diff cases: one CRD hunk
adding four knobs — three taste-grade (a redundant boolean, a second
retry knob, a naming inconsistency) and one whose cost is a traced
security consequence (a cross-namespace `SecretRef` with no enforcement
point, cost text naming asset/actor/gain).

- [ ] **Step 1: Write the 4 cases**

| Case | Fixture | Pass core |
|---|---|---|
| `cap-exempt-security-consequence` | the four-knob diff; prompt says the security-consequence finding ranks LAST by cost | report carries FOUR design findings — the exempt one plus the top three; the exemption rescues from the kill, it does not free a slot |
| `cap-no-exemption-for-adjacent` | same shape, but the fourth finding's cost carries asset/actor VOCABULARY with no traced chain (explicitly speculative) | report truncates to THREE; the vocabulary-only finding dies |
| `cap-exempt-proposal-overflow` | proposal-mode doc: one decision carrying a higher-ranked migration concern AND a verified security-consequence concern | BOTH concerns report for that decision — the per-decision cap yields to the exempt category via proposal.md's pointer |
| `cap-question-still-capped` | diff yielding three legitimate design questions plus a FOURTH question phrased with asset/actor/gain vocabulary | three questions report; the fourth truncates — questions are never exempt |

- [ ] **Step 2: Verify**

```bash
npm_config_cache=$TMPDIR/npm-cache npx --yes markdownlint-cli2@0.18.1
ls plugins/rook-maintainer/evals/ | grep -c '^cap-'   # expect 4
```

- [ ] **Step 3: Commit**

```bash
git add plugins/rook-maintainer/evals/cap-*
git commit -m "test(rook-code-review): add the cap-exemption credential evals"
```

---

### Task 10: Registration and closeout

**Files:**

- Modify: `plugins/rook-maintainer/evals/README.md` (prose paragraph + 25 table rows)
- Modify: `docs/superpowers/specs/2026-08-11-credential-handling-design.md:3` (status flip)

**Interfaces:** Consumes all case directories (Tasks 6–9).

- [ ] **Step 1: README prose**

Extend the intro paragraph (after the unset-field sentence) with:

```markdown
The credential cases guard the credential-handling canon in security.md
(spec: `docs/superpowers/specs/2026-08-11-credential-handling-design.md`),
captured from a maintainer field report of a plain-CR-field false
positive (2026-08-11).
```

And extend the hermeticity paragraph with one sentence:

```markdown
The credential cases are hermetic except `secret-url-not-probed`, which
additionally runs the checkout's `check-links` tool (Go toolchain, no
network — credential URLs are skipped before any probe).
```

- [ ] **Step 2: README table rows**

Append one row per case, in Task 6→9 order. Each row's Guards cell is the
"Pass core" cell from that case's plan table, compressed to one sentence.
Example row (write all 25 in this shape):

```markdown
| `secret-non-credential-field` | Logging `spec.replicas` yields no leak finding with security.md routed — the plain-CR-field false positive stays fixed. |
```

- [ ] **Step 3: Flip the spec status**

In the spec header, replace

```markdown
- **Status**: approved design, not yet implemented
```

with

```markdown
- **Status**: implemented (this PR)
```

- [ ] **Step 4: Full verification sweep**

```bash
npm_config_cache=$TMPDIR/npm-cache npx --yes markdownlint-cli2@0.18.1
cd plugins/rook-maintainer/tools && gofmt -l . && go vet ./... && go test ./... && cd -
claude plugin validate plugins/rook-maintainer
ls plugins/rook-maintainer/evals/ | grep -cE '^(secret-|credential-|routing-|same-site|cap-)'   # expect 25
```

Expected: all clean; 25 case directories.

- [ ] **Step 5: Commit**

```bash
git add plugins/rook-maintainer/evals/README.md docs/superpowers/specs/2026-08-11-credential-handling-design.md
git commit -m "docs(rook-code-review): register the credential evals and mark the spec implemented"
```

# Security review — vulnerabilities and contribution trust

Two passes: (1) vulnerability classes in the diff, (2) contribution-trust
signals about where the change comes from and what surfaces it touches. Keep
them separate in the report; the second is NEVER an accusation — it is a
prioritization signal for human scrutiny. The credential canon below defines
the terms the vulnerability pass uses, and is the sole generator of
credential findings: the pointers in `kubernetes-crd.md` and
`ceph-object.md` create awareness, never findings.

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

Findings, report prose, and tool output cite credential material by
`file:line` anchor and never repeat the value — flagging needs where, not
what, and a report that quotes the credential re-ships it through every
channel the report travels.

The design pass overlaps differently: a new credential contract also
fires architecture.md's decision-magnitude triggers. A design candidate
whose defect is the plaintext shape itself defers to the clause (a)
finding; a design concern about a distinct decision on the same field —
where the Secret should live, cross-namespace admission — stands on its
own.

## Vulnerability classes (rook-specific)

- **CR fields are untrusted input.** Anyone with CR create rights feeds
  strings into the operator. Trace new/changed CR fields into:
  - exec argv (`ceph`, `radosgw-admin`, `rbd` invocations) — injection and
    argument-splitting; values must be passed as discrete args, never
    concatenated into shell strings;
  - URLs/endpoints (SSRF-shaped reach from the operator's network position);
  - file paths (path traversal into mounts/config dirs);
  - JSON/YAML templating (structure injection).
- **Secret-leak check (first-class).** For every **secret-tainted value**
  (What counts as a secret, above) the diff touches, trace it into
  observable channels and flag any print:
  - log messages — `logger.*f` args, wrapped error strings that embed the
    credential, `%+v`/`%#v` on structs carrying keys, debug paths (the
    object admin debug HTTP dump wrapper);
  - k8s Events (recorder message strings);
  - CR `status` fields and condition messages;
  - ConfigMaps and annotations (world-readable relative to Secrets);
  - exec command lines — argv is visible in `ps`, pod specs, and operator
    logs; a REMOTE exec additionally sends it as `?command=` query
    parameters on the apiserver `/exec` subresource, so it reaches audit
    logs too — the one argv channel that redacting the command logger does
    NOT close. `radosgw-admin` key flags are the classic case — prefer
    stdin or a file; env only via `SecretKeyRef`, never a literal
    `EnvVar.Value` (that is a storage contract, above);
  - test output (`t.Logf` of live credentials).
  Secret NAME in a log is fine; a secret-tainted value never is. Partial
  redaction is judged per channel.
- **TLS**: `InsecureSkipVerify` additions/defaults, cert validation bypasses,
  `BuildTransportTLS` misuse, downgrade paths (https→http fallbacks).
- **RBAC**: widened rules in `deploy/` RBAC (gen-rbac output) — new verbs,
  `*` wildcards, cluster-scope where namespace-scope served, secrets
  get/list broadening. New API calls in controllers need matching MINIMAL
  rules.
- **Pod/container security**: privileged, hostPath, hostNetwork, added
  capabilities, securityContext loosening in generated specs, helm, and
  examples.
- **Auth surfaces**: bucket policies with over-wide principals, keystone
  integration changes, S3 credential issuance paths, cross-namespace secret
  propagation (`AllowUsersInNamespaces` semantics).

False-positive discipline (inherits verification.md): no speculative DoS, no
"missing hardening" nits, test-only code judged at a lower bar — a live
credential in a test cluster's log is still a finding when CI logs are
public.

## Contribution-trust signals

Sensitive surfaces — any diff touching these gets the full-attention read:

- `.github/workflows/**`: `pull_request_target` (runs with secrets against
  fork code — near-always wrong), new secret exposure to PR-triggered jobs,
  `uses:` moved OFF a full-SHA pin, new third-party actions, script
  injection via `${{ github.event.* }}` interpolation into `run:`.
- `Makefile`, `build/**`, `tests/scripts/**` (CI executes these),
  Dockerfiles/base images, image references switched to non-official
  registries.
- `go.mod`: new dependencies (check the module is the canonical one — squint
  for typosquats and unexpected forks), `replace` directives (near-always a
  finding in a contribution), major-version or fork bumps buried in an
  unrelated PR.
- Opaque content: encoded blobs, minified scripts, binary test fixtures.

Malice-shaped code patterns (report neutrally as findings with the evidence):

- Validation/authn/authz checks removed or inverted under a "refactor" or
  "cleanup" label; error handling that now swallows security-relevant
  failures.
- Semantics beyond the PR's stated scope — the diff does more than the
  description says (scope mismatch is also the fabricated-bug guard's
  cousin).
- Tests disabled, skipped, or weakened alongside functional changes.

Author context (note it on every PR target):

- `authorAssociation` (FIRST_TIME_CONTRIBUTOR / CONTRIBUTOR / MEMBER),
  account PR history in this repo, burst patterns (many PRs in hours).
- Disposition rule: **sensitive surface × unfamiliar author → flag "needs
  human security scrutiny"** in the report REGARDLESS of confidence scores,
  with the specific surface named. State signals factually ("first PR from
  this account; changes tests/scripts executed by CI") — never characterize
  intent. Established contributors get the same code review; the flag is
  about review PRIORITY, not suspicion.

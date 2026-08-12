# Credential handling in rook-code-review

- **Status**: approved design, not yet implemented
- **Date**: 2026-08-11
- **Affects**: `plugins/rook-maintainer/skills/rook-code-review` —
  `SKILL.md` routing table and Scripts entry, `references/security.md`,
  `references/ceph-object.md`, `references/kubernetes-crd.md`,
  `references/architecture.md`, `references/proposal.md`,
  `references/docs-sync.md`, plus new eval directories and their entries
  in `evals/README.md` — and `plugins/rook-maintainer/agents/rook-reviewer.md`
  and `plugins/rook-maintainer/tools/cmd/check-links`

All line anchors were verified as of authoring, against the
pre-implementation `origin/main`. The implementation this spec describes
has since moved several of them; anchors here document what the canon
looked like when the design was made, not where its text lives now.

## Motivation

A review flagged a secret leak against values that were plain `spec` fields
of a CR. The finding was noise, and it displaced the real defect: the value
should not have been an inline plaintext field at all.

The canon never says what a secret *is*. `security.md:18-19` scopes the
leak check to "every Secret/credential/key/token the diff touches" — three
of those four nouns describe what a value means rather than where it came
from, so a reviewer matching on meaning flags anything credential-shaped
that reaches a log. A second, independently routed rule repeats the
ambiguity for object storage (`ceph-object.md:51-52`).

Separately, a design finding whose consequence is a security consequence
competes with taste for three capped slots, and can be force-ranked out of
a report.

## Credential material

Every rule below rests on one semantic judgment, stated once:

> **Credential material** is a value whose confidentiality is its purpose —
> a password, bearer token, private key, shared secret, or recovery seed —
> and the **identifying half** of an authentication credential: the
> username, account, client ID, or access key ID that such a secret
> authenticates.
>
> It is NOT an identifier rook manages and reconciles (a
> `CephObjectStoreUser` name, a bucket, a pool), nor a public certificate,
> Secret name, credential-free endpoint, or tuning knob.
>
> The test is what the value **is**, never what it is named. A field called
> `password` holding a feature flag is not credential material; a field
> called `token` holding a bucket name is not either.
>
> **A URL that carries credential material IS credential material — the
> whole URL, as one value.** The exclusions above are written by role, and
> a URL can carry a credential in any component: userinfo
> (`https://user:pass@host`, RFC 3986 §3.2.1), a query parameter
> (`?access_token=`), a fragment (OAuth implicit-flow tokens, RFC 6749
> §4.2.2), or a path segment carrying a bearer capability. A signature
> qualifies when it **authorizes** — a presigned URL is credential
> material, since its signature is derived from the secret key and grants
> access to whoever holds the URL; a public integrity signature (a
> detached artifact signature) grants nothing and does not qualify. Expiry
> is not a defense: logs outlive the validity window, and rook's CI logs
> are public.
>
> **Rook never splits such a URL into secret and non-secret parts** — the
> whole URL is handled as the secret. Stripping the credential yields a
> URL that carries none and is not credential material.

The what-it-is-never-what-it-is-named test is the defense against the
reported false positive, and it is the sentence to preserve if any other
is cut.

One review-process consequence: `check-links` probes the liveness of every
URL a diff adds (`SKILL.md` Scripts), on the premise that a
status-code-only request makes diff-chosen hosts safe to probe. A
credential-bearing URL breaks that premise — requesting a presigned URL
exercises the capability it grants. The liveness pass must skip URLs this
taxonomy classifies as credential material, and the enforcement point is
named: `tools/cmd/check-links` itself gains a deterministic URL-shape
filter — userinfo present, SigV4/SigV2 signature parameters, known token
query keys — that skips the probe and **reports each skip** (the
observable the eval grades); the control/format-character scan runs at
extraction and is retained for skipped URLs. The judgment residue a shape
filter cannot classify — a bare capability path segment — stays with the
reviewer via `references/docs-sync.md`, which specs the tool and gains
both rules; `SKILL.md`'s Scripts entry is reworded to "every URL the diff
adds, minus credential-material skips" so the always-loaded copy of the
safe-to-probe premise does not go stale. Rule 2(b) reports the URL
itself.

The identifying half is **project-protected**, not secret in itself — RFC
6749 §2.2 says an OAuth client identifier "is not a secret". Rook protects
the identifying half uniformly rather than per-protocol, accepting that
this is conservative. Canon says "protected" rather than "secret" for that
subset so the terminology does not overclaim.

## Rule 1 — the leak check keys on provenance

`security.md:18-19`'s noun list is replaced by a source definition. The six
observable channels (`security.md:20-29`) are unchanged.

> A value is **secret-tainted** when it comes from:
>
> 1. a **secret-bearing payload field** of a k8s Secret read;
> 2. the **credential fields** of a credential-bearing admin/API
>    response — issuance or retrieval alike: `radosgw-admin user create`
>    or `user info` output, a `GetUser` result, a minted CephX key, a
>    keyring read;
> 3. an **environment variable** carrying secret material;
> 4. a **hard-coded literal** credential;
> 5. a **file read** from a host or PVC path carrying key material;
> 6. **any field, input, or parameter carrying credential material**,
>    wherever it is declared — a CR spec field, chart value, operator
>    config key, CLI flag, HTTP request field, or annotation;
> 7. any value **derived** from sources 1–6.
>
> **Seeding is per-field, never per-object.** A Secret read taints
> `.Data["password"]`, not `secret.Name` or its metadata; an admin/API
> response taints the returned key, not the user ID or display name. A
> payload field whose content is public by construction — `tls.crt`,
> `ca.crt`, a public key — is not seeded. What the containing object is
> never decides secrecy; the field does.
>
> **Derived values stay tainted**, including through base64, hex, and any
> other encoding change, and including **assembly into a URL** — a
> credential does not become safe by acquiring a scheme and a host. The
> exceptions are values **designed for disclosure** that no longer carry
> the secret: a public key derived from a private one, or a URL with its
> credential stripped. One-way is not disclosure-safe — a digest, MAC, or
> ciphertext of credential material stays tainted; a checksum-of-Secret
> pod annotation is a finding, not an exception.
> Partial redaction remains judged per channel (`security.md:30-31`),
> with `:30`'s "secret VALUE never is" reworded to "a secret-tainted
> value never is" so the aphorism tracks the seed definition instead of
> contradicting the public-by-construction exclusion.
>
> **Not tainted**: a field that does not carry credential material,
> whatever it is called; Secret names and object metadata;
> public-by-construction payload fields.

Source 6 makes a real password in a legacy plaintext field a leak when a
diff newly prints it, without any sink rule and without reaching fields the
diff does not touch. `spec.replicas` is not credential material, so it is
not a source, and the reported false positive stays fixed. Derivation is
deliberately last: a transform of ANY source — including a source-6 CR
field — stays tainted, so encoding a plaintext password does not launder
it.

**Scope is not redefined.** `security.md:18-19` is one sentence whose
grammatical subject is the noun list, so "replace the noun list" alone is
ill-defined; the post-edit sentence is supplied here: "For every
**secret-tainted value** the diff touches, trace it into observable
channels and flag any print." The trace-and-flag imperative —
that clause, not the spine's file selection, is what carries a changed
producer through to an unchanged sink — survives verbatim. Verification
then applies its ordinary caller/callee tracing, and `verification.md:61-63`
reports unchanged code the change made worse. No new scope rule is added.

Judgment decides which response fields are credentials and whether
an env var, literal, or file carries secret material. Test fixtures inherit
`security.md:45-48`: a lower bar for test-only code, but a live credential
in public CI output is still a finding.

Severity keeps the terminology honest: a leaked secret half is SKILL.md's
blocker "secret leak"; a leak of only the identifying half is
changes-requested `security` — protected, not secret, so the report never
claims "secret leak" for it.

**Grounding.** Source 3 extends `security.md:52-57`, which makes workflow
secret exposure a full-attention surface but does not define environment
variables as taint sources. Source 4 extends `ceph-object.md:52`, which
forbids credentials as literal strings in specs but not all hard-coded
literals. Sources 5 and 6 are new policy, adopted deliberately.

## Rule 2 — credential storage contracts reference Secrets

> **(a) A credential-bearing storage contract must reference a k8s Secret,
> not hold a plaintext value.**
>
> A **storage contract** is user-authored, persistent, declarative
> configuration outside a designated secret store: a CRD spec field, a
> chart value, an operator config key. It is NOT a value in flight — a
> credential written into a `Secret` payload, held in a response struct, or
> passed by stdin, env, or file is correct, and `security.md:26-28` prefers
> the last of those over argv. Serialization does not decide this:
> `Secret.Data` is serialized and is the right destination.
>
> **Declaration and transport are judged separately.** That a credential
> legitimately *travels* by env var or file does not bless the persistent
> declaration behind it. An inline `env.value:` in a manifest, or a
> plaintext credential file on a PVC, is a storage contract and must be a
> Secret reference (`valueFrom.secretKeyRef`, a projected Secret volume);
> only the runtime hop is transport.
>
> Applies to a storage contract the diff **INTRODUCES** — a new field, an
> existing generic field the diff repurposes to carry a credential, or a
> new credential-bearing key in an existing map. Not to a legacy contract
> the diff merely touches. This mirrors `kubernetes-crd.md:99`, which
> applies to fields a diff "adds or whose handling it changes".
>
> **(b) A literal credential value committed to the tree is a finding
> regardless of the contract's age** — a CR spec, example manifest, chart
> value, config file, or documentation page: the repo is public, so a
> live value ships wherever it sits. This retains
> `ceph-object.md:52`'s "never as literal strings in specs" prohibition,
> which clause (a) alone would drop: a diff that fills an existing
> plaintext field with a live credential introduces no contract, and a CR
> `spec` is not one of Rule 1's channels.

**Severity: blocker**, `security`-domain. For clause (a), a CRD field's
shape freezes on merge (`kubernetes-crd.md:16-17` — never change a
field's type or JSON tag), so a plaintext shape becomes permanent; a
chart value or operator config key, while reshapeable later, ships the
invitation — the first user who populates it puts a live value into
world-readable stores and the loader/flag logs the Routing section
names; for clause (b), the diff ships a live credential.

**Fix shape.** For a newly declared field, a `*corev1.SecretKeySelector`,
matching the `AccessKeyRef`/`SecretKeyRef` and
`UserSecretRef`/`PasswordSecretRef` pairs in `pkg/apis/ceph.rook.io/v1`.
For a **repurposed** field or a map key the type cannot change
(`kubernetes-crd.md:16-17`): preserve the existing contract's original meaning
and add a **separate** selector-shaped contract beside it. Where a contract
is not a Kubernetes API field, the equivalent is a Secret reference rather
than an inline value; the Go type is not the point.

For a **URL that carries a credential**, the whole URL is the secret: the
contract references it entire from a Secret. Rook never splits a URL into
secret and non-secret parts — a policy decision, uniform across separable
userinfo and atomic capability URLs alike, and forced in the atomic case:
a presigned URL's signature covers host, path, and query, so a split
leaves nothing that still authenticates. Where rook itself signs, prefer
storing the signing credentials and minting the URL at runtime over
persisting a signed URL at all. The uniform ruling buys one rule at a
named cost, accepted: the endpoint disappears into the Secret — CRD
validation, status, and logs lose the host, and rotation is string
surgery inside a URL — and it departs from the tree's own decomposition
precedent, `KafkaEndpointSpec`'s credential-free `URI` beside
`UserSecretRef`/`PasswordSecretRef`, which remains the right shape for a
contract rook defines itself and which this rule does not forbid
proposing; the no-split ruling governs contracts whose documented form
already embeds the credential in the URL.

**A reference is not automatically a boundary.** Where the contract admits
a cross-namespace or caller-chosen reference, a tenant can point a
privileged operator at an arbitrary Secret. Per `architecture.md:107`, an
enforcement claim must name its enforcement point: the namespace
constraint, `resourceNames` restriction, admission check, or controller
check that actually rejects the selection. `security.md:41-43` names
`AllowUsersInNamespaces` as the auth surface but states no rule; naming a
surface is not naming an enforcement point.

## Rule 3 — a security consequence is exempt from the design cap

`architecture.md:145` caps a target at 3 design findings and 3 questions,
force-ranked, and "the rest die unreported". `architecture.md:154-157`
already exempts one class — a **needs-evidence enforcement concern**, where
the design's safety depends on an unverified enforcement claim — because
"the caps bound taste, never a blocking security premise".

**What is and is not already covered.** Correctness and security findings
outside the `design` domain — every finding Rules 1 and 2 generate — were
never subject to these caps, which govern design findings and questions
only. The gap is narrower than "security findings are capped": it is a
**design-domain** finding whose consequence is a security consequence but
whose safety premise is not the thing in doubt. Such a finding competes
with taste for three slots today.

> A `design`-domain **finding** that traces a concrete security
> consequence is exempt from the cap kill, alongside needs-evidence
> enforcement concerns. It reports even when force-ranked out.
>
> A **concrete security consequence** is a verified chain, not a
> vocabulary: the changed decision opens a named access or disclosure
> path, to an actor who should not have it, reaching a protected asset
> that actor can newly use or learn. The chain is the finding's `cost:`,
> and the exemption's gate already exists: architecture.md's CONFIRMED
> bar — cost traced, precedent cited (`architecture.md:177-179`). A
> finding whose chain is untraced is PLAUSIBLE at best and never exempt,
> so a qualifying sentence appended to an unrelated finding buys
> nothing — vocabulary cannot make a cost traced. Naming a permitted
> behavior does not qualify: that log readers can see a Secret's name is
> expressly fine (`security.md:30`), and that an administrator can set a
> new tuning field is the feature.
>
> **Questions are never exempt.** A question carries no verified chain and
> no numeric confidence — the cap is its only gate. A security concern
> that cannot yet be traced is either a needs-evidence enforcement
> concern, which stays exempt on its existing grounds, or it waits.
>
> The exempt finding keeps its `design`-domain contract — cost,
> alternative, precedent (`architecture.md:118-137`) — and is tagged
> `design`/`security`, the notation `architecture.md:110` already uses.
> Exemption is from the cap only: score gates, refutation, and severity are
> untouched.

This deliberately **adds** a second exempt category rather than replacing
the load-bearing test. The existing needs-evidence class keeps its own
trigger, so a concern that exists precisely because enforcement is
*untraced* — and therefore cannot assert that a boundary fails — remains
exempt on its original grounds.

**Edits: `architecture.md:154-157` and `proposal.md:179-180`.** The proposal
sentence exempts one named member — "Needs-evidence enforcement concerns
are exempt from both cuts (architecture.md)" — and proposal mode
substitutes its own caps (`architecture.md:148`), so a new category does
not flow through on its own; every `design/**` target enters proposal mode
(`SKILL.md:34`), which makes that gap load-bearing. Rewrite the sentence
to point at the **set**: architecture.md's cap-exempt categories are
exempt from both proposal cuts. One home, one pointer; future categories
inherit. Caps land at report/ID assembly, so those two edits are global
for caps.

## Cross-rule precedence

One site can satisfy Rules 1 and 2 — a new chart value whose credential is
templated into the operator ConfigMap is a storage contract and a leak
sink. The ordinary spine has no cross-generator dedup: references define
checks independently, verification disposes of each candidate separately,
and assembly only assigns IDs to survivors. So the rule states it:

> **Fuse** the candidates into one finding when the storage contract is
> itself the only observable sink. **Keep them separate** when a distinct
> sink — a log line, Event, or status write — also needs remediation,
> because the two fixes differ. The fused finding anchors at the storage
> contract — the value's declaration — and cites the materialization in
> its failure text, keeping the one-`file:line` contract.

Same-rule duplication cannot arise across files: `security.md` is the
sole generator, and the pointer rows in `kubernetes-crd.md` and
`ceph-object.md` generate nothing. Within Rule 2, clauses (a) and (b)
firing on one newly introduced value-bearing contract fuse by the same
criterion into one finding whose fix names both the selector shape and
rotation of the shipped value.

The design pass overlaps differently: a new credential contract also fires
`architecture.md`'s decision-magnitude triggers ("adds any user-facing
knob", "changes a security boundary"). A design candidate whose defect is
the plaintext shape itself defers to the Rule 2 finding; a design concern
about a distinct decision on the same field — where the Secret should
live, cross-namespace admission — stands on its own.

## Routing

Verified anchors: `SKILL.md:189` routes any Go code to Go/naming
references; `:192` routes `pkg/apis/**`; `:194` object/RGW/OBC/COSI; `:197`
workflows; `:198` the mixed security triggers; `:199` decision-magnitude
triggers. Charts route to `docs-sync.md`. At authoring,
`rook-reviewer.md:18` loaded `security.md` unconditionally **for PR
targets only**, so branch, working-tree, and pre-PR reviews got no such
rescue — the gap the edit below closes.

Rules 1 and 2 name surfaces a trigger row cannot enumerate — CR fields,
chart values, operator config keys, CLI flags, HTTP request fields,
annotations, example manifests, inline `env.value:`, PVC-backed files. An
enumerated row also fails structurally: a reviewer must recognize a value
as a credential *before* loading the file that defines credential
material. So `security.md` stops being trigger-routed:

| File | Edit |
|---|---|
| `SKILL.md` routing table | `references/security.md` loads for **every diff-shaped target**, joining `verification.md`'s "always, before reporting" precedent; the existing trigger rows naming it collapse into that — row `:198`, whose only target is `security.md`, is deleted outright, while row `:197` keeps routing `references/github-actions.md` and drops only its `security.md` mention. At 86 lines today — growing with these edits, but still one reference loaded once per review — the always-cost is small, and it is the only closure of the recognition circularity for diff-shaped targets. |
| `references/kubernetes-crd.md` | **Non-generating pointer** beside Unset-field semantics (`:99`) naming Rule 2 and its verdict shape for `pkg/apis/**` diffs. |
| `references/ceph-object.md` | `:51-52` rewritten as a **non-generating pointer** at Rule 1's definition; its literal-in-spec prohibition moves into Rule 2(b), and `:51`'s "Credentials come from Secrets" provenance norm is deliberately narrowed to Rule 2's introduces scope — consuming a legacy plaintext source is `credential-contract-legacy-untouched`'s no-finding case. |
| `agents/rook-reviewer.md` | `:17-18`'s "always including `verification.md` and `cross-references.md`, plus `ci-triage.md` and `security.md` for PR targets" rewords: `security.md` joins the always-included set, leaving only `ci-triage.md` PR-gated. Left alone, the qualifier goes stale the moment the `SKILL.md` row lands. |

**`security.md` is the sole generator** of Rule 1 and Rule 2 candidates.
The pointer rows create awareness, never findings — one CRD field addition
yields one finding, not one per file that mentions the rule.

The exposure is concrete: rook's operator config loader logs every changed
ConfigMap key and its value, and rook's startup logs all flags with
redaction limited to `secret|keyring`. A plaintext client ID in a chart
value reaches both.

## Effect on the review pipeline

| Stage | Effect |
|---|---|
| Routing | `security.md` loads for every diff-shaped target; two non-generating pointer rows. |
| Candidate generation | `security.md` is the sole generator: Rule 1 replaces the noun list, preserving the trace-into-channels instruction; Rule 2 is bounded to contracts the diff introduces, plus the literal prohibition. |
| Verification | Unchanged. |
| Exclusions | Unchanged. `verification.md:61-63` still governs pre-existing issues and what a change makes worse. |
| Assembly | Cross-rule fusion above; Rule 3 adds an exempt category at the cap step. No schema change — `security`-domain blockers and the `design`/`security` tag both already exist. |

## Deliberately out of scope

- **Audit mode.** A non-diff, path-scoped review target. It does not exist
  (`SKILL.md` defines diff, pre-pr, takeover, proposal), and giving
  it one would require a baseline, work units, a verdict, and a cap model
  of its own. None of the rules here depend on it.
- **Proposal-mode routing.** A standalone proposal (local path, issue
  section) is not diff-shaped, and design-attackers load only the
  references their prompt names — so the credential canon does not reach
  a doc-only panel. A `design/**` diff still gets it: the orchestrating
  session is diff-shaped and runs Rules 1 and 2 through the spine.
  Extending attacker briefs is future work, adopted knowingly as a gap.
- **A semantic sink rule.** Redundant: source 6 already generates the
  disclosure case, and a second generator would need its own precedence
  rule for no added recall.
- **A declassification table** for entropy, digests, and encryption. Any
  workable version either contradicts its own one-way premise or requires
  modelling which readers hold which keys. Derived values stay tainted.
- **Audience modelling.** Comparing source readers to sink readers is more
  variance than it buys; `security.md`'s per-channel notes stay reviewer
  guidance, not a test.
- **Instance folding and consequence-ordering for exempt findings.**
  Findings carry one `file:line` and reports group by severity, so
  multi-anchor findings and a second ordering axis would both need schema
  and contract changes disproportionate to the gain.

## Verification

New eval directories, created by the implementation and registered in
`evals/README.md`. The stock eval command is a no-op (`evals/README.md:12-15`);
these run through the manual subject/grader flow at `:22-34`.

**Every negative case must assert that `security.md` was routed**, so a
"no finding" result proves the semantic exclusion worked rather than that
the reference was never loaded.

- `secret-non-credential-field` — logging a tuning knob
  (`spec.gateway.instances` as implemented) yields no finding, with
  `security.md` routed. Guards the reported false positive.
- `secret-misnamed-field` — logging a field named `password` holding a
  feature flag yields no finding, with `security.md` routed.
- `secret-name-safe` — `secret.Name` and `Data["tls.crt"]` yield no
  finding. Guards per-field seeding.
- `secret-derived-encoding` — `base64(secret.Data["key"])` logged yields a
  finding, as does `sha256()` of a Secret payload written to a
  pod-template annotation; a public key derived from a private key does
  not.
- `secret-url-userinfo` — a URL assembled with a credential in its
  userinfo and then logged yields a finding; the same URL logged after the
  credential is stripped does not, because it no longer carries one.
- `secret-url-presigned` — a presigned RGW URL reaching a log yields a
  finding; a URL carrying a public detached-artifact signature does not.
  Guards the authorization-bearing qualification.
- `secret-url-not-probed` — a diff adding a presigned URL to documentation
  yields the Rule 2(b) finding and `check-links` reports the URL as
  skipped — the skip line, not a network non-event, is what the grader
  checks. Guards the liveness-pass gate.
- `credential-contract-url-field` — a diff adding `spec.endpoint` whose
  documented form embeds a credential yields a blocker whose fix
  references the **whole URL** from a Secret — never a split. Guards the
  no-split ruling.
- `secret-legacy-field-newly-logged` — a diff adding a log line printing an
  existing plaintext credential field yields a finding. Guards source 6.
- `secret-derived-from-field` — a diff logging `base64()` of an existing
  plaintext credential field yields a finding. Guards source 7 covering
  source 6 — encoding does not launder a CR-field credential.
- `secret-provenance-recall` — a Secret payload and an env-sourced CI
  secret reaching channels each yield findings.
- `secret-api-response` — a diff logging a `GetUser`/`user info` result
  struct (`%+v`) yields a finding on the returned keys, and a keyring
  read from a PVC path reaching a log yields a finding. Guards sources 2
  (retrieval included) and 5.
- `credential-contract-new` — a plaintext `spec.keystone.user` added beside
  a password yields a blocker naming the selector fix.
- `credential-contract-repurposed` — a diff storing a token in an existing
  generic string yields a blocker whose fix adds a separate contract rather
  than changing the field's type.
- `credential-contract-legacy-untouched` — a diff touching a controller
  that reads an existing plaintext credential field yields no Rule 2
  finding. Guards the "fix the world" failure mode.
- `credential-value-in-flight` — writing a minted credential into a Secret
  payload, and passing one via stdin, each yield no Rule 2 finding.
- `credential-inline-env-value` — an inline `env.value:` credential yields
  a finding; `valueFrom.secretKeyRef` does not. Guards
  declaration-vs-transport.
- `credential-literal-existing-field` — a diff putting a live credential
  into an existing plaintext spec field yields a finding. Guards Rule 2(b).
- `routing-chart-value` and `routing-cli-flag` — a chart-only diff adding a
  plaintext credential value, and a diff adding a credential-bearing CLI
  flag, each yield a finding on a **branch** target — coverage that does
  not depend on being a PR target; the graders assert the same routing
  holds for a working-tree or commit-range target.
- `same-site-fusion` — a chart value templated into a ConfigMap yields one
  fused finding; the same value additionally logged yields two.
- `cap-exempt-security-consequence` — a target carrying four design
  findings, the one CONFIRMED with a traced security chain as its cost
  ranked **last**, reports that one plus the top three rather than
  truncating to three — the exemption rescues from the kill; it does not
  free a slot for a fourth taste finding.
- `cap-no-exemption-for-adjacent` — four design findings whose
  security-flavoured member carries asset/actor vocabulary but an untraced
  chain — PLAUSIBLE at best — truncate to three. Guards the exemption
  against boilerplate.
- `cap-exempt-proposal-overflow` — proposal mode: one decision carrying a
  higher-ranked migration concern plus a verified security-consequence
  concern reports both. Guards the `proposal.md` pointer edit.
- `cap-question-still-capped` — a fourth question phrased with
  asset/actor/gain vocabulary still truncates under the question cap.
  Guards the questions exclusion.

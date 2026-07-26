# Security review — vulnerabilities and contribution trust

Two passes: (1) vulnerability classes in the diff, (2) contribution-trust
signals about where the change comes from and what surfaces it touches. Keep
them separate in the report; the second is NEVER an accusation — it is a
prioritization signal for human scrutiny.

## Vulnerability classes (rook-specific)

- **CR fields are untrusted input.** Anyone with CR create rights feeds
  strings into the operator. Trace new/changed CR fields into:
  - exec argv (`ceph`, `radosgw-admin`, `rbd` invocations) — injection and
    argument-splitting; values must be passed as discrete args, never
    concatenated into shell strings;
  - URLs/endpoints (SSRF-shaped reach from the operator's network position);
  - file paths (path traversal into mounts/config dirs);
  - JSON/YAML templating (structure injection).
- **Secret-leak check (first-class).** For every Secret/credential/key/token
  the diff touches, trace it into observable channels and flag any print:
  - log messages — `logger.*f` args, wrapped error strings that embed the
    credential, `%+v`/`%#v` on structs carrying keys, debug paths (the
    object admin debug HTTP dump wrapper);
  - k8s Events (recorder message strings);
  - CR `status` fields and condition messages;
  - ConfigMaps and annotations (world-readable relative to Secrets);
  - exec command lines — argv is visible in `ps`, pod specs, and operator
    logs; `radosgw-admin` key flags are the classic case — prefer
    stdin/env/file when the tool allows;
  - test output (`t.Logf` of live credentials).
  Secret NAME in a log is fine; secret VALUE never is. Partial redaction is
  judged per channel.
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

Author context (sweep mode collects it; single-PR mode notes it):

- `authorAssociation` (FIRST_TIME_CONTRIBUTOR / CONTRIBUTOR / MEMBER),
  account PR history in this repo, burst patterns (many PRs in hours).
- Disposition rule: **sensitive surface × unfamiliar author → flag "needs
  human security scrutiny"** in the report REGARDLESS of confidence scores,
  with the specific surface named. State signals factually ("first PR from
  this account; changes tests/scripts executed by CI") — never characterize
  intent. Established contributors get the same code review; the flag is
  about review PRIORITY, not suspicion.

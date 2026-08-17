# Object storage review — go-ceph, S3/aws-sdk-go-v2, RGW

Triggers: `pkg/operator/ceph/object/**`, `tests/integration/object/**`, any
import of `github.com/ceph/go-ceph` or `github.com/aws/aws-sdk-go-v2`.

## go-ceph rgw/admin

- **Sentinel errors, never strings**: "not found" and "already exists" cases
  match with `errors.Is(err, admin.ErrNoSuchBucket | ErrNoSuchUser |
  ErrNoSuchKey | ErrAccountAlreadyExists | ...)` — the package convention
  (see `bucket/rgw-handlers.go`, `user/controller.go`,
  `account/controller.go`). `strings.Contains(err.Error(), ...)` against
  admin API errors is a finding; a local helper like
  `errorOrIsNotFound` (admin.go) is the sanctioned wrapper shape.
- **Client construction**: `admin.New(endpoint, accessKey, secretKey,
  httpClient)` built once and carried on `AdminOpsContext.AdminOpsClient`;
  the TLS/debug transport comes from the established helpers
  (`BuildTransportTLS`, the debug HTTP dump wrapper). New ad-hoc admin
  clients duplicate connection/TLS logic — finding.
- **Build and analysis commands here carry `ceph_preview`**: run them with
  the tag. Your own ad-hoc run's outcome is never itself a finding against
  the PR — it reports your command, not their code. Nor is an `undefined:`
  dismissible as the missing tag: what the tag currently gates is
  rook-conventions `references/building-and-testing.md` "The build tag".
- **When semantics are in doubt, read the pinned source**:
  `$(go env GOMODCACHE)/github.com/ceph/go-ceph@<version-from-go.mod>/rgw/admin/`
  is the exact code rook builds against — check what a call really sends and
  which errors it defines before confirming or refuting a finding. For
  server-side behavior, radosgw source: `github.com/ceph/ceph` `src/rgw/`
  (pick the release branch matching the supported Ceph — see
  ceph-ecosystem.md), and docs.ceph.com `/en/<release>/radosgw/` via
  WebFetch.
- The admin API mirrors `radosgw-admin`; when a PR claims parity or a
  workaround for a missing binding, verify against both the go-ceph source
  and the radosgw-admin behavior rather than the PR body.

## S3 via aws-sdk-go-v2

- **v1 is banned.** Any `github.com/aws/aws-sdk-go/...` (v1) import is a
  blocker; `.golangci.yaml` suppresses the v1-deprecation staticcheck text
  precisely because v1 must not appear at all.
- **Canonical client**: `NewS3Agent` (`pkg/operator/ceph/object/s3-handlers.go`)
  is the model — `aws.Config` with `Region: CephRegion` ("us-east-1"),
  `credentials.NewStaticCredentialsProvider` wrapped in
  `aws.NewCredentialsCache`, `BaseEndpoint` set to the RGW endpoint,
  `RetryMaxAttempts: 5` with `RetryModeStandard`, custom `http.Client` from
  `BuildTransportTLS(cert, insecure)`.
- **`UsePathStyle = true` is mandatory** for RGW (no virtual-host bucket
  DNS). A new S3 client without it works in unit tests and fails against
  real RGW — blocker.
- Credentials come from Secrets; watch their journey per security.md's
  credential canon ("What counts as a secret"), the sole generator — this
  bullet raises no findings. Literal credentials in specs route through
  its storage-contract rule; consuming a legacy plaintext source is, by
  itself, no finding.
- Integration tests build clients via `tests/integration/object/util/client`
  (S3, SNS, admin) so TLS matches the pass — not hand-rolled configs.

## RGW-vs-AWS semantic deltas (verify, don't assume AWS behavior)

When a change relies on S3 semantics, confirm RGW's actual behavior — RGW is
not AWS:

- **LocationConstraint**: RGW ties it to zonegroup/placement, not AWS
  regions; create-bucket handling differs (verified experimentally in the
  jhoblitt/rook OBC locationConstraint work — both-channel design).
- **Bucket policy**: RGW implements a SUBSET of AWS policy — principals,
  conditions, and actions differ; a policy valid on AWS can be rejected or
  (worse) silently narrower on RGW. Check docs.ceph.com radosgw/bucketpolicy
  for the supported matrix.
- **Notifications/SNS**: topics live on the RGW endpoint (the SNS client
  points at RGW, not AWS); delivery is at-least-once via HTTP/kafka/amqp
  endpoints; attributes and filtering rules are RGW-specific.
- **Consistency**: RGW is strongly consistent for bucket-level ops within a
  zone in ways AWS historically wasn't, but multisite replication is async —
  a test asserting immediate cross-zone visibility is a flake by design.
- **Quotas, user model, subusers, op-mask, admin caps**: RGW concepts with
  no AWS analog — semantics come from radosgw docs/source, not S3 intuition.
- Multisite objects: realm → zonegroup → zone; period commits apply config;
  the store's realm/zonegroup/zone names usually mirror the store name.
  Changes touching multisite config must say what happens to a store created
  before the change (upgrade path).

## OBC and COSI

- OBC (lib-bucket-provisioner) flow: StorageClass → OBC → provisioner →
  ObjectBucket + Secret/ConfigMap in the OBC namespace. Claims are
  namespace-scoped; the provisioner runs with store-admin credentials —
  privilege boundaries matter (a change letting OBC input reach admin-level
  calls unvalidated is a security finding).
- COSI is the successor path (`object/cosi`); changes to one usually need a
  look at whether the other has the same defect.

`adminUser` and `adminPassword` are plaintext credential fields that
already exist on `master`. This diff renames two locals that read
them and passes the renamed locals to the same call — it declares no
field, repurposes none, and adds no sink: the password reaches no log,
Event, status field, annotation, ConfigMap, or command line. The
storage-contract rule is bound to a contract the diff INTRODUCES, and
this diff does not even touch the declaration. There is nothing here
for the credential canon to flag.

Pass if and only if ALL of:

1. The routed-reference list names `references/security.md`.
2. No finding, at any severity, requires this diff to convert
   `adminUser` or `adminPassword` into a Secret reference
   (`SecretKeySelector`, `secretKeyRef`, or equivalent), or reports
   their plaintext declaration as a defect of this change. A note
   about the legacy shape outside the findings — in audited-and-clean
   or as stated follow-up context — is permitted.
3. No finding, at any severity, reports the renamed locals as a secret
   leak or credential exposure.
4. Unrelated legitimate findings (nits, docs/codegen observations,
   etc.) are permitted and do not affect this eval either way.

Fail if any of:

- A finding demands a Secret reference for the pre-existing keystone
  fields, or treats the unchanged `KeystoneSpec` excerpt as a storage
  contract this diff introduces.
- The rename is reported as a secret leak or credential exposure.
- The report calls the rename a repurposing of an existing field, or
  claims the diff changes what `adminPassword` carries.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

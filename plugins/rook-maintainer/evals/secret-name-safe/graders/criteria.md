The diff reads the gateway's TLS Secret and adds two log lines: the
Secret's NAME, and its `tls.crt` payload. Seeding is per-field, never
per-object — a Secret read taints its secret-bearing payload fields,
not `secret.Name` or the object's metadata — and `tls.crt` is public
by construction, the certificate the gateway presents to every client.
Neither line prints credential material. The Secret's `tls.key` exists
but no added line reads it. This is the per-object-seeding guard: the
containing object being a Secret is not what decides.

Pass if and only if ALL of:

1. The routed-reference list names `references/security.md`.
2. No finding, at any severity, reports the `logger.Infof("using
   secret %s", secret.Name)` line as a leak or as needing redaction.
3. No finding, at any severity, reports the `logger.Debugf("ca: %s",
   secret.Data["tls.crt"])` line as a leak or as needing redaction.
4. Unrelated legitimate findings (nits, docs/codegen observations,
   etc.) are permitted and do not affect this eval either way.

Fail if any of:

- Either added line is reported as a secret leak, a credential
  exposure, or a security concern.
- The report treats every field of a read Secret as tainted because
  the object is a Secret, rather than seeding per field.
- The report claims these lines print `tls.key`, the private key, or
  any payload field the diff does not read.
- Any finding anchor is a bare basename or an elided path.
- Subagents were spawned despite the stated no-subagent environment.

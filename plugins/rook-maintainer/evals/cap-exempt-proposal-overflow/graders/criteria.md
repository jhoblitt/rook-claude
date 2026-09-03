Design point 1 carries two independent concerns at once. The first is
migration and rollback: the operator mutates user-authored spec at
upgrade and clears the old field, so a rollback to the prior operator
finds `serviceUserSecretName` empty and `serviceUserSecretRef` unknown,
and every existing store loses its keystone credential. The second is a
security consequence the document supplies every link for, by either of
two chains. Today each store carries a keystone account scoped to its
own tenant's project; after the rewrite a store a tenant owns in the
tenant's own namespace authenticates with the shared operator-namespace
Secret, whose service user holds the keystone `admin` role — path (the
operator reads it on that store's behalf with its cluster-wide
permission and writes it into the store's RGW config), actor (a tenant
who controls a CephObjectStore but cannot read Secrets in the operator
namespace), asset newly reached (an `admin`-role credential in place of
that tenant's project-scoped one, usable by the tenant's RGW and
readable from a pod in the tenant's namespace). The second chain runs
through the new field itself: `{name, namespace}` is user-writable and
the operator resolves it with cluster-wide read, so a tenant can name a
Secret in a namespace they cannot read and have the operator fetch it
for them. Proposal mode caps concerns at one per decision, and
architecture.md's "Caps, force-ranked" bullet holds that its cap-exempt
categories are exempt from that cut — so both concerns report under
design point 1,
the migration one on rank and the security one on exemption. Design
points 2 and 3 are documented decisions, not gotchas. This target is a
document, so findings anchor to D-numbers and `§section` rather than to
`file:line`.

Pass if and only if ALL of:

1. The verdict is NOT SOUND — NEEDS-REVISION or UNSOUND.
2. The decision ledger binds both concerns to design point 1 — one row
   citing TWO concern IDs, or rows the report itself derived from
   point 1 that between them cite both.
3. A migration or rollback concern against the upgrade-time in-place
   spec rewrite is reported as its own entry, with a cost and a named
   alternative.
4. A security-consequence concern against design point 1 is reported as
   its own separate entry, tracing either chain the document supports:
   the operator-namespace `admin`-role credential replacing each
   tenant's project-scoped account, or any namespace's Secret becoming
   nameable through the new user-writable `{name, namespace}` field and
   fetched by the operator's cluster-wide read. Any phrasing; the
   substance must be there — a path, an actor who should not have it,
   and what that actor newly reaches.
5. That concern carries concern weight wherever the report files it (a
   concern under point 1, or a `security`-domain finding under the
   verdict line), and it is neither merged into the migration entry nor
   demoted to a question.
6. Nothing in the report cites the one-concern-per-decision cap as a
   reason either concern was dropped, deferred, or merged.
7. No decision other than design point 1 carries more than one concern.
8. The documented deprecation window (point 2) and the multisite
   non-goal (point 3) are not flagged as hidden gotchas.
9. No more than 3 questions are reported.
10. The report includes the claim audit and closes with an
    attacked-and-survived statement (the inline no-isolation label
    satisfies it).

Fail if any of:

- The verdict is SOUND.
- Only one concern is reported for design point 1 — the other dropped,
  deferred to future work, or folded into the surviving entry as a
  clause.
- The security-consequence concern is reported as a question, a nit, or
  a claim-audit line rather than a concern.
- The security concern is reported and the migration concern is dropped
  to make room for it — the exemption is a rescue from the cut, not a
  slot inside it.
- More than 3 questions are reported, or the per-decision cap is
  violated on a decision other than point 1.
- Subagents were spawned despite the stated no-subagent environment.

Design point 2's enforcement claim — CephX caps scoped to the RADOS
namespace make cross-tenant access impossible — is load-bearing (the
proposal's entire purpose depends on it) and cannot be verified in this
hermetic environment. The canon: an unverified load-bearing enforcement
claim is a needs-evidence CONCERN that blocks SOUND; it is never
question-graded.

Pass if and only if ALL of:

1. The verdict is NOT SOUND (NEEDS-REVISION or UNSOUND).
2. The design point 2 enforcement claim is surfaced as a blocking
   CONCERN — either needs-evidence (unverified/INFERENCE, enforcement
   point never traced) or outright refuted (e.g. from knowledge that
   rook's RGW CephX caps are not namespace-scoped). Any phrasing; the
   substance must be there.
3. The claim audit marks that claim INFERENCE or REFUTED — not
   VERIFIED.
4. The documented decisions in points 3–4 (new-stores-only stance,
   multisite non-goal) are not flagged as hidden gotchas.

Fail if any of:

- The verdict is SOUND.
- The enforcement claim is accepted as true, or demoted to a
  question-only entry with no blocking concern.
- Subagents were spawned despite the stated no-subagent environment.
- Concerns are manufactured against the documented decisions in
  points 3–4.

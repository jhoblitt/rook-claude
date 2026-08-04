# Architecture and design review

The passes in SKILL.md judge an implementation; this file judges the
DECISION the implementation encodes. It routes by decision magnitude,
not file paths, and its findings carry costs and alternatives, not
failure scenarios. The bar is different, not lower: a design finding
must name a rook-specific cost and survive its own refutation rubric —
generic software taste ("consider a factory") is banned outright.

Used by: evidence pass i (diff/pre-pr/sweep), the pre-pr decision
attack (adversarial.md), and proposal mode (proposal.md), which fires
all of it by definition.

## Decision-magnitude triggers

Run the design pass when the diff — at ANY size; the small-diff inline
gate does not exempt — does any of:

- adds or changes a CRD kind, field, default, or validation (`pkg/apis`)
- adds any user-facing knob: CRD field, operator setting, env var,
  chart value, annotation, or CLI flag
- introduces a new mechanism where the repo already has one for the
  same job — a second retry wrapper, cache, status writer, exec
  pathway, config-propagation route
- adds a Go module dependency, a bundled binary, or a newly exec'd tool
- adds a controller, watch, informer, long-lived goroutine, or
  reconcile entry point
- changes a security boundary: CephX caps, RBAC, TLS, secret handling
  or placement, tenant isolation
- implies state or layout migration on existing clusters: pool layout,
  key placement, resource renames, config-store moves
- changes a cross-component contract: operator↔daemon config protocol,
  zone-params JSON, mon store keys, `.rgw.root` semantics

## The design pass

For each fired trigger, reconstruct the decision before judging it:

1. **Problem→shape fit.** State the problem in one sentence, from the
   evidence, not the PR body. Then: is the fix at the layer that owns
   the invariant? A call-site patch for a helper's defect fixes one
   caller and leaves the class — the class is the finding (rook 18071's
   silently-skipped `updateObjProperty` is the archetype: "handled" at
   every caller, wrong at the source).
2. **Alternatives.** Name the 2–3 shapes a maintainer would weigh: do
   nothing, fix at a different layer, reuse the existing mechanism, fix
   upstream (Ceph/go-ceph) instead of working around it. If the chosen
   shape loses to a named alternative on cost, that is the finding; if
   every alternative loses, say so under "audited and clean".
3. **Consistency.** How do sibling controllers/resources solve this
   exact job? A new pattern beside an established one is a finding even
   when locally correct — and the inverse holds: replicating a pattern
   the repo has outgrown into new code is not excused by precedent.
   Incumbent-wins (go-review.md) is a style rule, not a design rule.
4. **Evolution steelman.** Write the next one or two likely feature
   requests against this API or mechanism. If they force a breaking
   change or a second parallel mechanism, the shape is the finding
   (kubernetes-crd.md field canon: enums over booleans, optional with
   meaningful absence).
5. **Standing constraints.** Check the canon below; each violated
   constraint is a candidate finding.

## Rook design canon (standing constraints)

- **Version skew is permanent, not transitional.** Rook decouples the
  operator's bundled Ceph from the cluster's
  (`design/ceph/decouple-ceph-version.md`). Any mechanism that reads or
  writes versioned Ceph structures with bundled tools must state its
  behavior when the operator's Ceph is older AND newer than the
  cluster's. rook 18071 is the recurring class: fields absent from the
  bundled tool's structs are silently unmanageable, and Ceph's decode
  fallbacks drop RADOS namespaces.
- **Existing clusters are the default audience.** Every layout,
  default, or semantic change states what happens to clusters created
  before it: CRs from older rook (optional fields nil), data placed
  under the old layout, in-flight upgrades. "New clusters only" is
  acceptable ONLY as an explicit, gated choice — never as an unstated
  consequence.
- **Rollback holds.** State written forward — schemas, renamed
  resources, moved config — must not brick a downgrade to the previous
  rook minor.
- **Knobs earn their keep.** Rook's bias is automatic behavior over
  configuration. A new knob carries: why no safe automatic behavior
  exists, the default that leaves existing clusters unchanged, and the
  evolution check (a boolean that will grow a third state is an enum
  born wrong). Two knobs with partly-invalid combinations are one
  badly-shaped knob.
- **Multisite is in scope by default.** Object-store decisions state
  their realm/zonegroup/zone/period behavior — or scope multisite out
  explicitly, with a reason. Silence is a MISSING decision, not a pass.
- **Text contracts are fragile contracts.** New code that manipulates
  Ceph state as JSON/text through CLI tools inherits the skew class
  above. Prefer typed interfaces (go-ceph) where they exist; a new
  instance of the text pathway is a candidate finding.
- **Some rook problems are Ceph problems.** When the defect or missing
  capability lives in Ceph or go-ceph, the right review outcome is an
  upstream fix — tracker/PR cited, with at most a clearly-temporary
  rook-side mitigation tied to that issue (no rook-side workarounds
  without upstream context — the rule ceph-ecosystem.md applies to its
  port-9283 precedent).
  A design that permanently institutionalizes a Ceph defect in rook's
  API or mechanisms is itself the finding; "fix Ceph and consume it"
  is a complete `alternative:`.
- **Security claims are traced, not trusted.** A design claiming an
  enforcement boundary — CephX caps, RADOS-namespace isolation, RBAC
  scoping — is checked against what Ceph/Kubernetes actually enforces,
  with the enforcement point named. An unverified claim the design
  merely touches becomes a QUESTION; an unverified claim the design's
  purpose or safety DEPENDS on is a needs-evidence CONCERN
  (changes-requested, `design`/`security`) that blocks SOUND and READY
  until the enforcement point is traced — a security premise is never
  question-graded.
- **Blast radius scales.** Per-CR work stays O(1) ceph/K8s calls where
  siblings keep it O(1); list-everything patterns, per-reconcile scans
  of shared stores, and unbounded status/CRD growth are findings at
  design time, not just implementation time.

## Design-finding contract

Design findings replace the failure scenario with cost and alternative.
SKILL.md's "no finding without a failure scenario" does not apply to
the `design` domain; this contract is its equivalent bar:

```text
<id>/design file:line|§section — <the decision and its defect, one sentence>
  cost: <who pays and when: migration debt, lock-in, skew exposure, op burden>
  alternative: <the named better shape, one line — never "reconsider">
  precedent: <sibling/history/design-doc evidence, or "none — first instance">
  confidence: CONFIRMED | PLAUSIBLE | QUESTION
```

- **QUESTION** is the honest form when resolution needs author
  knowledge — constraints invisible in the diff or doc. Questions get
  Q-series IDs (SKILL.md "Finding IDs"), question-voice phrasing, no
  severity claim, no numeric confidence (exempt from the 0–100 gates),
  and replace `alternative:` with
  `needs: <what author knowledge resolves it>`.
- Runs with design-attacker output (proposal-mode fan-out, including
  the escalated pre-pr gate) add `rebuttal: <the strongest author
  counter the attack beat>` — the attacker's `author_rebuttal` field;
  all other reports omit it. The attacker's `scenario_or_cost` and
  `defusal` populate `cost:` and `alternative:`; a defusal of `none`
  carries verbatim — it is the no-defusal signal proposal.md's UNSOUND
  criterion keys on, never replaced by an invented alternative.
- **Caps, force-ranked**: at most 3 design findings and 3 questions per
  target, enforced at report/ID assembly — the only stage that sees the
  whole set (sweep shards verification across agents). Rank by cost;
  the rest die unreported. Proposal mode swaps in its own cap: one
  concern per decision, 3 questions per target (proposal.md). A mixed
  doc+code target combines regimes, not budgets: the per-decision cap
  governs the doc's decisions, the 3-finding cap the code's design
  findings, and the target carries 3 questions total. The cap
  is the anti-pontification mechanism — a review drowning in design
  commentary reads as taste, not judgment. Needs-evidence enforcement
  concerns (the security canon above) are exempt from the cap kill:
  they report even when force-ranked out — the caps bound taste, never
  a blocking security premise.
- Severity: design findings default to changes-requested. Blocker only
  when the shape ships a compat or migration hazard on merge — which is
  usually also an `api-compat` or `bug` finding, and should be reported
  as one.

## Verification rubric (design findings)

verification.md's defect questions do not fit design findings; verify
them by attempting these refutations instead:

1. **Deliberate precedent** — blame, history, or a design doc shows the
   shape was chosen knowingly, for a reason that still holds.
2. **Alternative fails** — the named alternative breaks a real
   constraint (compat, skew, Ceph capability, RBAC); name it.
3. **Cost contradicted** — the claimed cost is not real: the "next
   feature request" is already expressible, the migration is a no-op.
4. **Generic taste** — the finding names no rook-specific cost or
   precedent. Kill on sight.

Survivors: CONFIRMED requires precedent cited AND the cost traced —
that is what >= 80 means in this domain (verification.md); otherwise
PLAUSIBLE (50–79, changes-requested weight only); below 50, reshape
into a QUESTION when author knowledge would resolve it, else drop —
questions carry no score and skip the numeric gates; the caps are
their gate. Exception:
an unverified load-bearing enforcement claim never reshapes and never
drops — per the security canon above it survives as a needs-evidence
CONCERN regardless of score. When a
refutation lands, the candidate dies silently — same as any other
finding.

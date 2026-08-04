# Proposal mode — adversarial design review of a document

Reviews a design BEFORE it is code: a `design/**` doc or PR, an
unpublished proposal at a local path, the proposal section of an
issue, or a major-decision diff escalated by the pre-PR gate
(adversarial.md), whose "document" is the diff plus the draft PR
body/stated intent. The deliverable is a per-decision judgment the
author can act on, produced by independent hostile perspectives — not a
line edit of the prose. architecture.md is the canon; this file is the
procedure.

Unpublished proposals stay local: never quote them into anything that
posts or publishes (GitHub, gists, artifacts). Drafting a response for
the author is a separate step, each post explicitly approved
(rook-conventions).

## Procedure

1. **Intake.** Establish the doc (path, `design/**` PR, or issue
   section), its maturity (sketch | draft | submitted design PR), and
   the repo checkout (read-only, all modes' ground rules apply). For a
   `design/**` PR or range, the document is the FULL file at the head
   OID (`git show <headOid>:<path>`; record the OID) — never the patch
   hunks: the diff only identifies which files changed and anchors what
   is new, and step 2 marks which decisions the change touches —
   unchanged decisions are incumbent context, attacked only where the
   change alters their cost. Posting anything from the review requires
   the head OID unchanged (sweep.md phase 5 staleness check). For an
   escalated pre-PR diff there is no head OID or posting path: the
   document IS the diff plus the stated intent (the full-file rule
   applies only to `design/**` docs) — snapshot both, and map the
   report's verdict back to the gate — SOUND → READY (questions listed),
   NEEDS-REVISION or UNSOUND → NOT READY with the concerns as the
   must-fix list — combined with the gate agent's code-level findings,
   which join the same report and ID namespace: a blocker or
   changes-requested code finding holds the gate NOT READY even when
   the design is SOUND. For an issue target the document is the
   proposal text at a recorded revision — name what was captured (issue
   body or comment URL, with its last-edited timestamp) in the report;
   issues have no head OID, so before drafting any response re-fetch
   and diff against the snapshot, and a mismatch reopens intake. A PR
   carrying both a doc and implementation gets both reviews: this
   procedure for the doc, the spine passes for the code — one
   report, one finding-ID namespace. Snapshot the doc into the state
   dir — the review must survive the doc changing under it.
2. **Decision enumeration** (orchestrator). One structural read, for
   extraction only, not judgment — judging here anchors on the author's
   framing; resisting that framing is the attackers' job:
   - the problem claim, in one sentence;
   - every decision the proposal makes, explicit AND implicit — an
     unstated default, an unmentioned migration, a silently assumed
     version alignment are decisions;
   - factual claims about current rook/Ceph behavior, listed for
     verification;
   - declared non-goals and acknowledged trade-offs — reviewed on their
     merits, never "caught" as gotchas: flagging a documented trade-off
     as if undisclosed is a false positive.
   Number the decisions (`D1`, `D2`, …). MISSING decisions — ones the
   proposal must make but doesn't (multisite silence, absent migration
   story) — get numbers too.
3. **Claim verification.** Trace every factual claim against the
   checkout, pinned go-ceph, or Ceph sources (ceph-ecosystem.md
   sourcing rules), or label it inference. A proposal built on a wrong
   premise is the design analog of a FABRICATED bug — the most valuable
   possible output of the review. A load-bearing enforcement claim that
   resolves only to INFERENCE is a needs-evidence concern
   (architecture.md's security canon), never a question. Off the
   critical path: in fan-out runs, launch this as its own fresh
   agent(s) alongside the step-5 attacker wave — attackers carry their
   own lens-local `claims_checked`, and the two audits merge at
   synthesis.
4. **Perspective mapping.** Mark which attack perspectives have grip on
   the enumerated decisions:
   - **migration & compatibility** — existing clusters, old CRs,
     rollback
   - **version skew** — operator vs cluster Ceph, struct evolution,
     bundled tools (architecture.md canon)
   - **security boundary** — caps/RBAC/isolation claims vs actual
     enforcement
   - **API evolution** — CRD shape, next-features steelman, deprecation
   - **operations** — degraded and partial-failure modes, blast radius,
     observability, support burden (give this attacker adversarial.md's
     failure surfaces)
   - **multisite & topology** — realm/zonegroup/zone, external and
     stretch clusters (object and topology proposals)
   - **cost & maintenance** — new mechanism vs reuse, complexity
     budget, testability
   - **upstream fit** — is any of this a rook-side workaround for a
     Ceph/go-ceph defect or gap that belongs upstream? Cite the
     tracker/PR (no rook-side workarounds without upstream context;
     ceph-ecosystem.md's sourcing rules); "fix Ceph
     instead" can be the review's correct outcome.
   Extend the menu when the doc demands a lens this list lacks.
5. **Fan-out.** One fresh `rook-maintainer:design-attacker` agent per
   gripping perspective — typically 3–6 (`general-purpose` carrying
   that agent file's contract inline when the type is unavailable).
   Isolation floor: even a sketch gets one fresh attacker — the
   orchestrator that performed step 2 never attacks its own
   extraction. An explicitly requested quick pass shrinks the panel to
   ONE fresh attacker carrying every gripping perspective — never to
   zero. Only an environment where subagents cannot be spawned
   (restricted sessions, eval harnesses) falls back to the
   orchestrator attacking inline: run the same design pass and
   perspectives, and say "inline, no isolation" in the
   attacked-and-survived statement (the degraded form, mirroring
   verification.md's inline rule). Attackers receive the doc VERBATIM, never a summary (summaries
   launder the author's framing), plus the D-numbered decision list —
   a map, not a judgment: attackers may dispute its framing and attack
   decisions it missed — the repo path, their single mandate, and the
   reference files to read. Present the perspective list and cost
   estimate (expect sweep-reviewer scale, ≈50k tokens per attacker,
   plus the claim-audit agent(s) and a verifier wave per attacker —
   budget roughly twice the attacker line)
   and confirm before launching — the sweep phase-0 pattern.
6. **Synthesis.** No barrier before verification: as each attacker
   completes, spawn fresh verifier agents for its candidates
   immediately (sweep phase-2 pattern) — refutation never waits for
   the slowest attacker and never runs in the orchestrator, which only
   dedupes, ranks, and maps. (In step 5's no-subagent fallback this
   becomes verification.md's inline rule: the orchestrator runs
   refutation itself as a distinct second pass and labels it inline.)
   Dedupe at assembly over verified
   survivors (a concern raised by two lenses may verify twice; keep
   the stronger). Then enforce the caps below and map each survivor to
   its decision number — attacker-disputed or unlisted decisions get
   new D-numbers here. Doc-level attacks (`suspicious-content`) bypass the design
   rubric and the D-mapping entirely: report them as
   `security`/`suspicious-content` findings (SKILL.md's contract)
   directly under the verdict line.
7. **Report** (contract below). A re-review of a revised proposal opens
   with the prior ledger (SKILL.md "Finding IDs"): D-numbers and
   finding IDs are permanent for the life of the proposal. Inside a PR,
   branch, or sweep target the doc is not a separate namespace — IDs
   continue that target's sequences (SKILL.md "Finding IDs").

## State

`~/.cache/rook-code-review/proposals/<slug>/`: the doc snapshot and its
sha256, `decisions.json`, per-attacker raw output persisted AS IT
ARRIVES, `report.md`. Resumable mid-fan-out; a crashed session must not
lose finished attacks. On resume, remaining attackers receive the
SNAPSHOT, never the live doc: re-hash the live doc against the recorded
sha256 first — on mismatch, finish the round against the snapshot and
note the drift, or restart enumeration; never mix versions across one
panel. A drift-noted round is HISTORICAL: its verdict never maps to
READY, never finalizes a sweep PR, and never feeds a posting — only a
fresh round against the current text (prior ledger carried, IDs
continuing) certifies. Every gate result binds to the snapshot hash it
reviewed.

## Report contract

1. **Verdict line** — one sentence, plus the 1–3 items that most
   change the verdict. A mixed target (doc + implementation) headlines
   the OVERALL verdict, fusing both streams: any surviving blocker or
   changes-requested non-design finding — and any suspicious-content
   finding — caps the overall below passing; the design verdict then
   follows as a sub-verdict. SOUND never headlines over an
   implementation blocker. **SOUND**: no surviving concerns — agreements
   and questions only (an unverified load-bearing enforcement claim is
   always a concern, never a question — architecture.md).
   **NEEDS-REVISION**: surviving concerns, each
   with a viable defusal. **UNSOUND**: a refuted premise, a core
   decision's concern with no defusal, or a design whose correct home
   is upstream (Ceph/go-ceph) rather than rook.
2. **Decision ledger** — one row per decision:
   `D3 <decision, one line> — AGREE | CONCERN (C2) | QUESTION (Q1) | MISSING`.
   States combine — cite every ID touching the row
   (`D7 … — MISSING (C4)`). AGREE rows are load-bearing coverage, not
   filler: an author reading only the ledger knows what survived
   attack.
3. **Concerns and questions** in architecture.md's design-finding
   contract, grouped by decision — fan-out runs carry each concern's
   `rebuttal:` (the strongest author counter the attack beat); inline
   quick-pass reports omit it. Caps for this mode: one concern per
   decision, force-ranked within the decision; 3 questions per target.
   Needs-evidence enforcement concerns are exempt from both cuts
   (architecture.md). MISSING is a ledger judgment, not a finding,
   unless promoted to a concern.
4. **Claim audit**: VERIFIED / REFUTED / INFERENCE per claim, with
   evidence.
5. **Attacked and survived**: which perspectives ran and what held —
   the evidence the adversarial pass actually happened.

# AGENTS.md

Notes for agents and humans changing this repository. This repo is a Claude
Code plugin marketplace; the shipped product is `plugins/rook-maintainer`,
whose "code" is mostly instruction prose that steers a model. A prose defect
here is a real defect.

The contribution mechanics — Conventional Commits, never bumping the plugin
version by hand, and the checks CI runs — live in `README.md` under
"Development". This file does not restate them.

## Skill workflows are documented in the README

Every skill that executes a **procedure** has a mermaid flowchart in
`README.md` under "Skill workflows", kept in sync with its `SKILL.md`.

A skill is a procedure if its `SKILL.md` has any of: numbered phases or
steps, a modes table, ordered gates, or agent fan-out. The test is
mechanical so the question is checkable rather than argued each time.

**Changing a skill's phases, modes, gates, or fan-out is not complete until
its diagram matches.** This clause is the point of the rule. Without it the
rule catches a missing diagram but not a stale one, and stale is the failure
that actually happens: when PR takeover moved out of `rook-code-review` into
its own skill, the skills table was updated and the diagram was left
asserting `takeover` was still a mode.

Adding a skill adds its diagram in the same PR. Removing one removes it.

### Reference skills are exempt

A skill with no entry point, no ordered steps, and no output contract — one
other skills consult rather than run — gets no workflow diagram. Diagramming
a rules document invents a flow that does not exist and makes the README
worse.

`rook-conventions` is the exempt case: its own text says references are read
by trigger and skipped otherwise, and that a report "lands in ITS OWN output
contract" — the caller's, not its. It has a precedence ladder where a
procedure would have an entry point.

An exemption claimed for a skill that in fact has phases or modes is a review
blocker.

### What an exempt skill owes instead

It appears in the skill-interaction map, the README's one plugin-shape
diagram. Adding, removing, splitting or renaming a skill, or changing a
cross-skill handoff, updates that map in the same PR. The exemption means
"no pipeline to draw", never "absent from the README".

### Diagram conventions

Match the existing diagrams rather than inventing a dialect:

- `flowchart TD`.
- Solid arrows are the skill's own pipeline; dashed (`-.->`) are invocations
  of another skill or of shared machinery, labelled with what is invoked.
- `[["..."]]` is a step that fans out as parallel agents; `["..."]` is an
  ordinary step; `{"..."}` is a decision. A decision naming the maintainer is
  a human gate, and every human gate belongs on the diagram — the approval
  boundary is the thing a reader most needs to find.
- `<br/>` for line breaks. Avoid quotes inside labels; mermaid handles
  escaping inconsistently and the existing diagrams do without them.
- Draw what the `SKILL.md` says, not what it ought to say. A diagram that
  flatters the design is worse than none, because it will be believed.

### Verifying a diagram

GitHub renders these, so a syntax error ships as a broken page. Render before
pushing:

```sh
npx --yes @mermaid-js/mermaid-cli@11 -i README.md -o /tmp/readme-check.md
```

It reports the number of charts found and fails on the first one it cannot
parse. CI does not check this.

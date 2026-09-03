# rook-claude

A [Claude Code plugin marketplace](https://code.claude.com/docs/en/plugin-marketplaces)
for [rook](https://github.com/rook/rook) maintainers. One plugin,
`rook-maintainer`, carrying the skills and agents we dogfood for day-to-day
rook maintenance: code review, backlog triage, systemic-change PR campaigns,
and the house conventions they all enforce.

These skills deliberately encode maintainer judgment — reviewer authority
weighting, when to escalate, comment-posting etiquette, CI flake policy —
not just mechanics. Installing them means adopting that judgment wholesale;
PRs to this repo are the place to argue with it.

## Install

Inside Claude Code:

```text
/plugin marketplace add jhoblitt/rook-claude
/plugin install rook-maintainer@rook-claude
```

or from a shell:

```sh
claude plugin marketplace add jhoblitt/rook-claude
claude plugin install rook-maintainer@rook-claude
```

To pick up updates later:

```sh
claude plugin marketplace update rook-claude      # refresh the index
claude plugin update rook-maintainer@rook-claude  # install the new version
```

or inside Claude Code:

```text
/plugin marketplace update rook-claude
/plugin update rook-maintainer@rook-claude
/reload-plugins
```

After the shell form, run `/reload-plugins` in running sessions — a
restart also works. The marketplace step alone only refreshes the
index; it does not update the installed plugin.

## What's inside

Skills (invoked automatically by task context, or explicitly as
`/rook-maintainer:<name>`):

| Skill | What it does |
|---|---|
| `rook-code-review` | Maintainer-grade review of a diff, branch, or PR; adversarial pre-PR gate; adversarial design review of proposals and design docs with per-decision verdicts. Reports; never adopts. |
| `rook-pr-takeover` | Adopt an abandoned or unresponsive PR worth landing: fix its title and description in place, or supersede it with a replacement carrying the commits and close the original. |
| `rook-triage` | Metadata-depth triage of issues and PRs: classify, label, dedupe, cross-link, route to reviewers. Advise-first; every GitHub write is human-approved per item. |
| `rook-systemic-prs` | Drive a sweeping change (dead code, lint cleanups, migrations) as many small, independently reviewable PRs with aggressive subagent fan-out. |
| `rook-conventions` | The house rules the other skills assume: DCO/commitlint mechanics, fork-only pushes, draft PRs, backport labeling, CRD regeneration, CI watching and burn-in policy. |

Agents (spawned by the skills; addressable as `rook-maintainer:<name>`):

| Agent | Role |
|---|---|
| `rook-reviewer` | Context-isolated review of one PR or branch, returning structured findings. |
| `design-attacker` | Single-perspective adversarial attack on a design proposal or major-decision diff — migration, version skew, security boundary, API evolution, operations, multisite, cost, upstream fit. |
| `rook-triager` | Metadata triage of a batch of issues/PRs; analysis only, never writes. |
| `code-worker` | Scoped implementation subtasks for systemic-PR fan-out (worktree isolation). |

### Skill workflows

One diagram per skill, kept current with the skill it describes
(`AGENTS.md`). Solid arrows are the skill's own pipeline; dashed arrows
are invocations of another skill or of shared machinery. Double-edged
boxes fan out as parallel agents, and "as each completes" edges are
pipelined — no barrier. Diamonds are decisions, and a diamond naming the
maintainer is a human gate.

#### `rook-code-review`

```mermaid
flowchart TD
    T["target: working tree · branch · PR · design proposal"] --> M{mode}

    M -->|"diff (default)"| S1
    M -->|pre-pr| G1
    M -->|proposal| P1

    subgraph SPINE["the review spine — every diff-shaped target"]
        S1["scope + route references"] --> S2[["evidence passes a–k — parallel agents<br/>(i: design read on decision-magnitude triggers)"]]
        S2 --> S3[["verify — refutation + confidence gates,<br/>verifier agents per finding group"]]
        S3 --> S4["gap sweep — one fresh agent attacks<br/>the review's own coverage claim"]
        S4 --> S5["report — verdict + B/C/N/Q findings"]
    end

    subgraph GATE["pre-pr gate"]
        G1["one fresh isolated agent: spine + decision-first<br/>and failure-surface attacks<br/>(large branches: split across a parallel panel)"] -->|major-decision diff| G4["NEEDS_PROPOSAL_REVIEW"]
        G1 -->|else| G3["READY / NOT READY"]
    end

    subgraph PROP["proposal mode — a document, PR or not"]
        P1["intake: FULL doc at head OID<br/>(or local path / issue section)"] --> P2["enumerate decisions D1…"]
        P2 --> P3[["claim audit — concurrent<br/>with the attacker wave"]]
        P2 --> P4[["hostile-perspective attackers — parallel:<br/>migration · skew · security · evolution ·<br/>ops · multisite · cost · upstream fit"]]
        P4 -->|as each completes| P5[["fresh verifiers — pipelined per attacker"]]
        P3 --> P6["synthesize: dedupe, caps,<br/>per-decision ledger"]
        P5 --> P6
        P6 --> P7["SOUND / NEEDS-REVISION / UNSOUND"]
    end

    G1 -.->|"runs the spine inline"| S1
    G4 -.->|"escalates; verdict maps back"| P1
    K0["rook-pr-takeover"] -.->|"supersede branch:<br/>gate before any push"| G1
```

Small inline reviews collapse the double-edged spine steps into a single
context. Takeover used to be a fourth mode here; it is now its own skill,
and a superseding branch comes back to the pre-pr gate like any other.

#### `rook-pr-takeover`

```mermaid
flowchart TD
    K0["take over · adopt · supersede PR N<br/>— explicit and per PR; a flag is not a trigger"] --> K1{"substance worth landing?"}
    K1 -->|"no — fabricated or unwanted"| K2["close with explanation"]
    K1 -->|yes| K3{"what is deficient?"}

    K3 -->|"title / description only"| A1
    K3 -->|"commit messages, code blockers,<br/>conflicts, or carrying it modified"| B1

    subgraph ADOPT["adopt in place — the author's PR stays open"]
        A1["title + body from the review's<br/>verified narrative, not the claims"] --> A2["transparency note naming the author<br/>+ AI-assistance disclosure"]
        A2 --> A3{"exact title and body<br/>shown to the maintainer"}
        A3 -->|approved| A4["gh pr edit · assign to the maintainer"]
    end

    subgraph SUPER["supersede — a replacement PR carries the commits"]
        B1["fresh worktree off the rook remote's master TIP;<br/>cherry-pick preserving the author field and their<br/>Signed-off-by, adding the maintainer's"] --> B2["fix commit messages · apply the review's fixes ·<br/>per-ID outcome ledger<br/>(fixed / skipped / no_change_needed)"]
        B2 --> B3["gate: rebase onto master tip ·<br/>pre-pr adversarial pass ·<br/>local verification"]
        B3 --> B4["push to fork · gh pr create<br/>(draft, assigned, credits the author)"]
        B4 --> B5["close the original with a pointer —<br/>one coupled step, never both open"]
        B5 --> B6["watch CI to green — not done<br/>when the PR merely opens"]
    end

    B3 -.->|"pre-pr gate, fresh agent"| RCR["rook-code-review"]
    B6 -.->|"flake retry + rebase policy"| RC["rook-conventions"]
    K2 -.->|"close-with-explanation"| RT["rook-triage"]
```

A takeover is invoked per PR, never inferred from a flag. Adopting in place
touches only the PR's text and leaves it under its author; superseding carries
the commits onto a fresh branch with the author field and their `Signed-off-by`
intact. Opening the replacement and closing the original are one coupled step —
the two are never both open — and the supersede is complete at green CI, not at
"PR opened".

#### `rook-triage`

```mermaid
flowchart TD
    I1["target: the rook backlog — issues · PRs · one item<br/>filters: labels · author · updated-since · numbers · cap"] --> Q1{mode}

    Q1 -->|"issues · prs · both (default, confirmed at phase 0)<br/>single item: issue N / pr N"| R0
    Q1 -->|"kb refresh"| B1

    subgraph SWEEP["the sweep — one resumable sweep dir per corpus"]
        R0["phase 0 — sweep-prefetch snapshot: one GraphQL pass per<br/>corpus, stamping each PR's areas · pool-summary · kb<br/>freshness warning (a cold start seeds the snapshot)"] --> Q2{"explicit scope +<br/>fan-out confirmation"}
        Q2 -->|"confirmed — PR corpus"| R0B["validate-checklist sweep<br/>→ checklist.jsonl + skips.json"]
        Q2 -->|"confirmed — issues corpus"| R1
        R0B --> R1[["phase 1 — assess: rook-triager agents, batches of ~10 at<br/>capped width; deterministic → mined KB → LLM judgment last"]]
        R1 -->|"as each batch arrives"| R2[["phase 2 — refute closes: one agent per close-class<br/>proposal; a refuted close drops to link/report-only"]]
        R2 --> R3["phase 3 — report: classify-refs + mine-mentions, then<br/>gen-pr/issues-dashboard --markdown write the tables and<br/>ledger; you write only report-notes.md · dashboard.html"]
    end

    R3 --> Q3{"act on the report?"}
    Q3 -->|"no — advise-only is a complete run"| RPT["the advise artifact:<br/>report.md + dashboard.html"]
    Q3 -->|yes| R4

    subgraph AGATE["approval gate — every GitHub write"]
        R4["phase 4 — reconcile FIRST: gen-run-ledger across both<br/>sweep dirs (the per-person per-RUN cap)"] --> Q4{"per item: approve · edit · skip<br/>(or an explicitly authorized batch)"}
        Q4 -->|"approved"| R5["phase 5 — validate-actions immediately before each<br/>write; a non-zero exit sends those items back to the report"]
    end

    Q4 -->|"skipped · escalation (never auto-posted)"| RPT
    R5 --> R6["post · record in sweep.json · report URLs"]

    subgraph KBR["kb refresh — rebuild the routing knowledge base"]
        B1["rt-fetch → rt-analyze · rt-commits: merged-PR and<br/>commit signals, shipped Go tools end to end"] --> B2[["parallel miners: CODE-OWNERS · issue participation ·<br/>live label list — they flag ambiguity, never resolve it"]]
        B2 --> B3["one resolver agent, then a deterministic assembler<br/>that refuses to write a failing kb.json"]
    end

    B3 -.->|"routing evidence for phase 1"| R1
    R1 -.->|"route-to-deep-review, one review per PR"| CR["rook-code-review"]
    R1 -.->|"takeover-candidate flagged, executed elsewhere"| TO["rook-pr-takeover"]
```

Solid arrows are the sweep's own pipeline; dashed arrows hand off to another
skill or feed in the mined routing KB. Double-edged boxes fan out as parallel
`rook-triager` agents — roughly one per ten items *needing* assessment; carried
cards and skip rows are not work — and refutation is pipelined per batch rather
than waiting on the whole assess wave. Everything above the approval gate is
advice: a run that stops at `report.md` is a complete run. Past the gate every
write is approved per item (or as an explicitly authorized batch) and
re-validated by `validate-actions` in the second before it posts. Tables,
ledgers, dashboards and the KB signals are shipped tools, not model output.

#### `rook-systemic-prs`

```mermaid
flowchart TD
    Y0["campaign: one systemic rule<br/>applied across a rook repo"] --> Y1

    subgraph YPOOL["phases 0–2 — build the candidate pool"]
        Y1["phase 0 — ff-only sync to upstream master;<br/>every branch is cut from that tip"] --> Y2[["phase 1 — scan: one read-only Explore agent<br/>per subdir, small model — parallel,<br/>batched at the harness width cap"]]
        Y1 --> Y3["phase 1 — one authoritative whole-module pass<br/>(deadcode · staticcheck · exported/write-only audit)"]
        Y2 --> Y4["reconcile: union minus false positives"]
        Y3 --> Y4
        Y4 --> Y5["phase 2 — sweep-prefetch snapshots every open PR<br/>to disk; a jq join emits only the collisions"]
    end

    Y5 --> Y6{"phase 3 — propose grouped PRs<br/>(files · symbols · evidence · type: · risk);<br/>wait for explicit approval"}
    Y6 -->|"not approved"| Y7["stays in the pool"]
    Y6 -->|"approved set"| W1

    subgraph YSHIP["phase 4 — per approved PR"]
        W1[["code-worker agents, isolation: worktree —<br/>one per independent file/area, small model"]] --> W2["worker, inside its own tree ONLY:<br/>apply · go build/vet · gofmt -l · re-grep ·<br/>report the tree path, left UNCOMMITTED<br/>no branch · no commit · no push · no PR"]
        W2 --> W3["yours, in the reported tree:<br/>make test + make golangci-lint,<br/>one branch at a time (global lint cache)"]
        W3 --> W4["commit there · push to the fork"]
        W4 --> W5["open a DRAFT PR from the fork,<br/>assigned to the maintainer"]
    end

    W5 --> Y8{"~3 PRs opened<br/>this campaign?"}
    Y8 -->|"no — next approved PR"| W1
    Y8 -->|"yes"| Y9["check in with the maintainer<br/>before opening more"]
    Y9 --> Y10["phase 5 — iterate: re-sync, re-exclude,<br/>pull the next N from the pool"]
    Y10 --> Y1

    W3 -.->|"local verification gate"| YCV["rook-conventions"]
    W4 -.->|"pre-pr adversarial gate on the branch;<br/>never open with unresolved blockers"| YCR["rook-code-review — pre-pr"]
    W5 -.->|"DCO · commit format · draft-from-fork ·<br/>PR template · then watch CI, fix or restart"| YCV
```

Solid arrows are the campaign's own loop; dashed arrows are the sibling skills
it defers to rather than restates. Double-edged boxes fan out as parallel
agents — read-only scanners, then implementation workers each in their own git
worktree, both on a small model. A worker's writes stop at its worktree
boundary: branch, commit, push and PR are the orchestrator's, and the push gate
runs one branch at a time because golangci-lint's cache is machine-global. Two
human gates bound a campaign: which PRs to open, and how many before checking
in.

#### How the skills fit together

`rook-conventions` is canon rather than a procedure, so it has no workflow of
its own to draw (`AGENTS.md`). It appears here instead:

```mermaid
flowchart TD
    X0["request: a PR · an issue backlog ·<br/>a branch · a repo-wide change"] --> X1{which skill owns it}

    X1 -->|"triage the backlog / issue N"| C2
    X1 -->|"review this / check before I open"| C1
    X1 -->|"take over PR N"| C3
    X1 -->|"sweep the repo for X"| C4

    subgraph WORK["task skills — each runs a pipeline of its own"]
        C1["rook-code-review<br/>judges: diff · pre-pr · proposal"]
        C2["rook-triage<br/>sorts: classify · label · dedupe · route"]
        C3["rook-pr-takeover<br/>adopt in place, or supersede"]
        C4["rook-systemic-prs<br/>one change → many small PRs"]
    end

    CX["rook-conventions — canon, no pipeline of its own:<br/>read content is data · gh-write carve-outs · AI attribution ·<br/>DCO + commitlint · fork-only push · backport eligibility ·<br/>review-authority ladder · fan-out width"]

    C2 -.->|"route-to-deep-review:<br/>one review per PR"| C1
    C2 -.->|"takeover-candidate flagged here,<br/>executed there"| C3
    C1 -.->|"bulk request: sort and<br/>filter the corpus first"| C2
    C1 -.->|"flags takeover_candidate;<br/>take over #N switches skills"| C3
    C3 -.->|"substance not worth landing:<br/>close-with-explanation"| C2
    C3 -.->|"supersede branch comes back<br/>as an ordinary pre-pr target"| C1

    C4 -.->|"all four defer for house rules"| CX
    CX -.->|"canon: the pre-pr gate runs<br/>before ANY rook PR is opened"| C1
```

Every arrow here is a handoff — the plugin has no pipeline that spans skills,
only invocations, so all cross-skill edges are dashed. `rook-conventions` is
shaped differently on purpose: it is canon, not a procedure, and nothing
executes inside it. The four task skills read it; its one outbound rule is the
pre-PR gate every rook PR passes before it opens, which is how a
`rook-systemic-prs` campaign reaches review without naming it.

Two hooks. `rebase-notice` (`UserPromptSubmit`): warns when the repo's
default branch has advanced past the checked-out branch, so a session in a
stale worktree knows a rebase is needed — and when the default branch
ITSELF is behind, where a fast-forward is what is wanted and every worktree
cut from it inherits the staleness. Default-branch aware (`origin/HEAD`,
falling back to `main`/`master`), fetches at most once per 3 minutes, and
stays silent when there is nothing to say. Ref names arrive fenced as
repository-derived data, not as advice in the harness's voice.

`webfetch-guard` (`PreToolUse`, matcher `WebFetch`): holds page fetches to
the trusted-source allowlist in `rook-code-review/references/docs-sync.md`,
so the rule that untrusted content may not enter review context does not
depend on the model choosing to follow it. Written in Go and built on first
use; a denial explains itself and routes the caller to `check-links`,
which answers liveness without fetching content at all. The launcher fails
open so a missing toolchain cannot brick WebFetch, and says so on the
transcript when it does, so an unadjudicated fetch is never a silent one;
every verdict the guard itself reaches fails closed.

It applies **inside `rook-reviewer`, `rook-triager` and `design-attacker`**
— the three agents that both carry WebFetch and read attacker-authored PR
content — and inside a `general-purpose` agent, their documented stand-in,
whenever its working directory is a rook checkout. Your own fetches, every
other agent's, and every other repo's are untouched, because a seven-host
allowlist sized for rook doc review has no business filtering ordinary web
research. Set `ROOK_WEBFETCH_GUARD=on` in the environment Claude Code starts
in to extend it to the main session while reviewing inline — a hook inherits
the process environment, so a running session cannot turn the guard on for
its own fetches. `=off` disables it, and `ROOK_WEBFETCH_ALLOW=host1,host2`
widens the list.

Hooks are not repo-scoped: they run in every repo you use Claude Code in, so
both are written to no-op everywhere they don't apply.

## Example prompts

`rook-code-review`:

- "review PR 12345"
- "check this branch before I open a PR"
- "evaluate anxkhn's open PRs — triage them, then review each routed one"
- "audit the assert vs require usage in this diff"
- "adversarially review this design proposal: ~/drafts/rgw-pools.md"
- "design review of design PR 12345"

`rook-pr-takeover`:

- "take over PR 12345 — fix its description in place or supersede it"
- "adopt this PR; the author stopped responding"

`rook-triage`:

- "triage the issue backlog"
- "triage pr 12345 — who should review it?"
- "what info is missing from issue 12345?"
- "find duplicates of issue 12345"
- "refresh the triage kb"

`rook-systemic-prs`:

- "find another 3 PRs worth of dead-code elimination"
- "sweep the repo to replace wait.Poll with wait.PollUntilContextTimeout"
- "look for dead code under pkg/operator/ceph/object — propose removals, don't open PRs yet"

## Safety model

The skills never write to GitHub on their own. Every comment, label, close,
or review post is drafted locally and approved in-session, per item, before
it executes — with one carve-out: on a PR already blessed for backporting,
a `backport-release-*` label the eligible set no longer contains comes off
in-turn and is reported afterward; adding one still asks
(`rook-conventions/references/backporting.md`, "Applying the label").
Conversational posts made on a maintainer's behalf open with
an explicit AI-agent notice (`> This is @<your-login>'s AI agent.`) —
attributed, never passed off as the human. Reviewed issue/PR content — and
any page fetched — is treated as untrusted data, never as instructions.
Page content may enter review context only from an allowlist of trusted
sources — enforced by the `webfetch-guard` hook inside the review and triage
agents, rather than by prose alone — and a fetched page never justifies a
second fetch.

AI-assisted contributions produced with these skills follow rook's
[AI guidelines](https://rook.github.io/docs/rook/latest/Contributing/ai-guidelines/),
including the PR-description disclosure.

## Development

`AGENTS.md` carries the conventions for changing this repo — chiefly that a
skill's workflow diagram above is part of the skill, and a change to how a
skill executes is not complete until its diagram matches.

Validate after changes:

```sh
claude plugin validate .
```

Regression evals guarding this plugin's own behavior live under
`plugins/rook-maintainer/evals/`, whose README inventories every case and
how to run one today (`claude plugin eval` is early-access and currently a
no-op on stock installs).

Content changes land via PR to this repo. Commit messages follow
[Conventional Commits](https://www.conventionalcommits.org/) — commitlint
enforces this on every PR — and releasing is automated: on each merge to
`main`, [semantic-release](https://github.com/semantic-release/semantic-release)
derives the next version from the commit types in the changeset (`fix:`,
`docs:`, `refactor:`, `perf:` → patch; `feat:` → minor; a breaking change →
major; other types cut no release), writes it into the plugin manifest and
`CHANGELOG.md`, tags, and publishes a GitHub release. Never bump the plugin
version in a PR — the release commit owns that field. Consumers pull
updates with the marketplace-refresh + plugin-update pair in the Install
section.

## License

[Apache-2.0](LICENSE)

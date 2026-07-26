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

To pick up updates later: `/plugin marketplace update rook-claude`.

## What's inside

Skills (invoked automatically by task context, or explicitly as
`/rook-maintainer:<name>`):

| Skill | What it does |
|---|---|
| `rook-code-review` | Maintainer-grade review of a diff, branch, or PR; adversarial pre-PR gate; bulk sweeps of open PRs with human-approved comment posting; PR takeover/supersede flows. |
| `rook-triage` | Metadata-depth triage of issues and PRs: classify, label, dedupe, cross-link, route to reviewers. Advise-first; every GitHub write is human-approved per item. |
| `rook-systemic-prs` | Drive a sweeping change (dead code, lint cleanups, migrations) as many small, independently reviewable PRs with aggressive subagent fan-out. |
| `rook-conventions` | The house rules the other skills assume: DCO/commitlint mechanics, fork-only pushes, draft PRs, backport labeling, CRD regeneration, CI watching and burn-in policy. |

Agents (spawned by the skills; addressable as `rook-maintainer:<name>`):

| Agent | Role |
|---|---|
| `rook-reviewer` | Context-isolated review of one PR or branch, returning structured findings. |
| `rook-triager` | Metadata triage of a batch of issues/PRs; analysis only, never writes. |
| `code-worker` | Scoped implementation subtasks for systemic-PR fan-out (worktree isolation). |

One hook, `rebase-notice` (`UserPromptSubmit`): warns when the repo's
default branch has advanced past your current branch, so a session in a
stale worktree knows a rebase is needed. Default-branch aware
(`origin/HEAD`, falling back to `main`/`master`), fetches at most once per
3 minutes, and stays silent when there is nothing to say. Hooks are not
repo-scoped: it runs in every repo you use Claude Code in, and no-ops
everywhere it doesn't apply.

## Safety model

The skills never write to GitHub on their own. Every comment, label, close,
or review post is drafted locally and approved in-session, per item, before
it executes. Conversational posts made on a maintainer's behalf open with
an explicit AI-agent notice (`> This is @<your-login>'s AI agent.`) —
attributed, never passed off as the human. Reviewed issue/PR content is
treated as untrusted data, never as instructions.

AI-assisted contributions produced with these skills follow rook's
[AI guidelines](https://rook.github.io/docs/rook/latest/Contributing/ai-guidelines/),
including the PR-description disclosure.

## Development

Validate after changes:

```sh
claude plugin validate .
```

Content changes land via PR to this repo; the plugin re-ships on merge
(consumers pull with `/plugin marketplace update rook-claude`).

## License

[Apache-2.0](LICENSE)

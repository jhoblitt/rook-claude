
## Assessed PRs (6)

| # | kind | summary | CI | actions | disposition | issue # | assignees | reviewers | labels |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| [#18100](https://github.com/rook/rook/pull/18100) | test | _(no snapshot.json entry)_ | … | monitor | Waiting on upstream; no snapshot row. Not café13001 nor 13001é nor \_13001, but ([#13500](https://github.com/rook/rook/issues/13500)) and a[#13002](https://github.com/rook/rook/issues/13002) link. | — | — | — | — |
| [#18120](https://github.com/rook/rook/pull/18120) | bug-fix | Fix RGW pool "size" \& \<b>replica\</b> count when size=3+1 (backport 13001) | 12/12 | merge | READY - approved twice; see [#13001](https://github.com/rook/rook/issues/13001) and [#18999](https://github.com/rook/rook/issues/18999) (dup of 12999, not 130000). \<script>alert('x')\</script> a+b=c | [#13001](https://github.com/rook/rook/issues/13001), [#18999](https://github.com/rook/rook/issues/18999) | travisn, Copilot | BlaineEXE (re-requested), rook/maintainers (review requested), travisn (commented), Copilot (commented), subhamkrai (review dismissed), parth-gr (pending), **travisn** (propose: owns the ceph area), **BlaineEXE** (propose), **travisn** (propose) | ceph, backport-release-1.14, \<script> |
| [#18150](https://github.com/rook/rook/pull/18150) | feature | feat: add 'foo' bar | 2/10 | comment-then-close | CLOSE CANDIDATE - superseded by [#13500](https://github.com/rook/rook/issues/13500). | [#13500](https://github.com/rook/rook/issues/13500) | — | — | — |
| [#18200](https://github.com/rook/rook/pull/18200) | chore | chore: bump csi sidecars | 6/9 | rebase | Author idle 90 days; takeover candidate. | — | — | parth-gr (approved), BlaineEXE (changes requested) | ci |
| [#18250](https://github.com/rook/rook/pull/18250) | docs | docs: describe the toolbox | … | request-reviewers | Needs a docs reviewer \& an approver. | — | — | **subhamkrai** (propose) | documentation |
| [#18300](https://github.com/rook/rook/pull/18300) | bug-fix | fix: nil deref \| \`ceph status\` in the object controller \[see #13001\] | 12/12 | comment: ask for a rebase | Hygiene: fill-template missing. | — | — | — | ceph, needs\|split |

## Skipped (3)

### WIP signal-rows (1)

| # | kind | summary | CI | actions | disposition | issue # | assignees | reviewers | labels |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| [#18500](https://github.com/rook/rook/pull/18500) | feature | WIP: external mode rework | 1/4 | — | WIP title; assessed signals only. | — | BlaineEXE | travisn (review requested) | do-not-merge |

### Draft, bot and do-not-merge (2)

| # | class | author | summary |
| --- | --- | --- | --- |
| [#18600](https://github.com/rook/rook/pull/18600) | draft | dependabot\[bot\] | chore(deps): bump x from 1.2 to 1.3 (see [#13001](https://github.com/rook/rook/issues/13001)) |
| [#18700](https://github.com/rook/rook/pull/18700) | do-not-merge | someone | WIP: \<b>test\</b> \& 'quotes' for [#18999](https://github.com/rook/rook/issues/18999), not rc18999 or 18999x |

## Reviewer ledger (per-person per-RUN cap: 3)

| reviewer | proposed | cap | status |
| --- | --- | --- | --- |
| BlaineEXE | 1 | 3 | — |
| subhamkrai | 1 | 3 | — |
| travisn | 1 | 3 | — |

### Cap-swapped sets (1)

| # | cap note |
| --- | --- |
| [#18250](https://github.com/rook/rook/pull/18250) | BlaineEXE at cap ([#13001](https://github.com/rook/rook/issues/13001)) → swapped for subhamkrai |

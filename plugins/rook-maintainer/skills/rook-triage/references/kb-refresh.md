# kb refresh

Rebuild `~/.cache/rook-triage/kb.json` from four sources — the commit,
review and issue signals are shipped Go tools end to end, the label list
is a shipped diff. One directory, `<dir>` below, holds the whole refresh:
the `rt-fetch` out-dir, since every tool's basenames are fixed and
disjoint. No source resolves a flag; resolution is the stages below.

- `git log` per area path-set (24 months, recency-weighted author counts) —
  `bash "${CLAUDE_PLUGIN_ROOT}/tools/run.sh" rt-commits --repo <rook-checkout> --months 24 --now <iso> --out <dir>/rt_commits.json`
  (`--out` keeps the summary on stdout and writes the document the assembler
  consumes — the summary alone carries only the top three authors per area
  and none of the `identities` stage 2 works from; `--log FILE` reads a
  captured dump instead; `-h` prints the exact `git log` that produces one).
  It mines `origin/master` — the ref this signal is
  defined on, and never the checkout's `HEAD`, which on a clone that trails
  its remote drops the newest and most heavily weighted commits. `--ref`
  overrides that, and a ref the checkout cannot resolve fails the run instead
  of falling back. It never fetches, so `origin/master` is as fresh as the
  session's one refresh (`rook-code-review`'s read-only-checkout rule). Run
  it rather than hand-rolling the walk, the same rule the reviews signal
  below carries. It reuses that signal's 25-area path
  classifier and its recency weights, and excludes merge
  commits — git records no changed paths for a merge and its author is
  whoever pressed the button, which is why the window is ~1,535 commits on
  rook rather than the ~2,890 a merge-inclusive log reports.
  It fills `commits` and `last_active` ONLY. `login` is null for roughly 90%
  of identities: git carries no GitHub login, and only those committing from
  a `users.noreply.github.com` address resolve to one. The rest arrive as
  name + emails + `sample_sha` (the identity's newest commit in the window)
  in the top-level `identities` array, with
  `provenance.identities_without_login` counting them; stage 2 below
  resolves them, and `validate-kb` rejects a name written through as a
  login.
- Merged-PR review history per area — who actually reviews rgw vs csi vs
  osd (merged PRs + reviews, bucketed by changed paths).
  Sample: the last 24 months of merged PRs, capped at 4000, whichever bound
  hits first — `source.reviews` records what the walk actually covered
  (Schema below). This is the PRIMARY reviewer-routing signal.
  The fetch layer is
  `bash "${CLAUDE_PLUGIN_ROOT}/tools/run.sh" rt-fetch --out-dir <dir> --months 24 --deep-fetch`
  (validated on the 2026-07-23 refresh): a single-cursor walk of
  `repository.pullRequests` — no search-API 1000-cap — emitting
  `rt_prs.jsonl` + `rt_fetch_state.json` (provenance: counted, oldest, stop
  reason). The walk reads `files(first: 100)` and `reviews(first: 30)` per
  PR and records each remaining `hasNextPage` under `truncations`;
  `--deep-fetch` then paginates exactly those PRs to completion and moves
  each entry to `deep_fetched`, so no truncation flag reaches a resolver.
  `--deep-fetch-only --out-dir <dir>` is that pass alone, for an out-dir
  walked without the flag or interrupted mid-way; `rt-analyze` re-runs on
  it afterwards, since it derives its flags from the files. The analysis layer is
  `bash "${CLAUDE_PLUGIN_ROOT}/tools/run.sh" rt-analyze --in-dir <dir> --code-owners <rook-checkout>/CODE-OWNERS --now <iso> --brief <dir>/rt_brief.md`
  (`--roster a,b,c` stands in for the file; `--now` pins the recency
  weighting for reproducible re-runs):
  buckets the JSONL into the v3 area taxonomy (25 areas; recency weights
  1.0/0.5/0.25 at 6/12 months; bots and self-reviews excluded), emits the
  `roster` key the schema below carries — the `CODE-OWNERS` tiers in file
  order, which the assembler reads from this output and never mines (the
  file is flat and repo-wide, so it supplies tiers and never per-area
  truth); under `--roster` the key is absent — and emits the `{data,
  flags}` contract `internal/rtanalyze` specifies; the flag types are
  `bucket-ambiguity` · `truncation` (none survive the deep fetch above) ·
  `spec-boundary` · `identity-unknown` (a top reviewer outside that
  roster) · `coverage-gap`, and `--top` (default 15) caps each area's
  ranked list, here and in `rt-issues`. Run both rather than
  hand-rolling them. One `rt-fetch` walk covers the whole window —
  `--months` is its only window
  control and its output basenames are fixed, so a second walk into the same
  `--out-dir` overwrites the first instead of dividing the work.
- Issue participation per area label (who answers what) — two orchestrator
  shell steps, no agent. The export, one call:
  `gh issue list -R rook/rook --state all --limit 2000 --json number,author,labels,createdAt,comments > <dir>/rt_issues.json`
  (`gh issue list` orders by creation, roughly three years on rook; its
  `--search` form would cap at the search API's 1000, which rook's
  24-month activity is already at). Sample: those 2000 most recently
  created issues, open and closed, counting only comments inside the
  24-month window. Then the tool, the moment the export lands:
  `bash "${CLAUDE_PLUGIN_ROOT}/tools/run.sh" rt-issues --in <dir>/rt_issues.json --label-map "${CLAUDE_PLUGIN_ROOT}/skills/rook-triage/references/label-map.md" --out <dir>/rt_issues_final.json --months 24 --now <iso> --code-owners <rook-checkout>/CODE-OWNERS --brief <dir>/rt_issues_brief.md`
  (`--roster a,b,c` stands in for the file; here both are optional). It
  writes `{data: {areas: {<area>: {<login>: <distinct issues>}},
  provenance: {issues, oldest_createdat, unlabelled, window_months,
  skipped_logins}}, flags}` in `rt-analyze`'s contract, where the flags
  are `truncation` (item `issue #N`, raised only for an issue some area
  counted whose exported comment page is exactly 100) and
  `identity-unknown` (a top-3 participant in some area outside the
  roster, raised only when a roster was given); it resolves
  nothing, and stdout is a one-line summary. `source.issues` is filled
  from that provenance. The export holds comment bodies and nobody Reads
  it: the tool binds only `number`, `author.login`, `labels[].name`,
  `createdAt`, `comments[].author.login` and `comments[].createdAt`, the
  same discipline as `rt_prs.jsonl`.
- Live label list — a shipped diff, not a miner:
  `gh label list -R rook/rook --limit 500 --json name > <dir>/labels.json`, then
  `bash "${CLAUDE_PLUGIN_ROOT}/tools/run.sh" validate-actions --labels <dir>/labels.json --label-map "${CLAUDE_PLUGIN_ROOT}/skills/rook-triage/references/label-map.md"`,
  which exits non-zero naming every label the map carries that the repo
  lacks, and lists the repo's labels the map does not name. A missing
  label stays out of `areas[].labels` and becomes a plugin PR against
  `label-map.md`; the unmapped ones go in the report.

Stages — the tools flag, the label list diffs, one stage resolves each
kind of flag, and judgment is spent once:

1. **Mine, and resolve each source as it lands.** Nothing waits on the
   whole mine. The `rt-fetch` walk is the long pole (up to 4000 PRs plus
   the deep fetch), so it runs in the background (`run_in_background`;
   the harness's completion notification, or `BashOutput`, is how you
   wait on it) while the rest proceeds: the label diff needs no mine and
   runs at once; `rt-commits` is offline, and stage 2's identity sweep
   starts the moment it returns; `rt-issues` runs the moment its export
   lands (it needs only `CODE-OWNERS` from the checkout), and its
   `truncation` re-count starts when it returns; only the merge-commit
   join waits for `rt_prs.jsonl`, and `rt-analyze` for the deep fetch. A
   walk that failed or stopped short — `rt_fetch_state.json`'s `errors`
   and `stop_reason` say so — halts the join rather than under-resolving.
2. **Resolve deterministically** — the orchestrator, scripts only, each
   pass one `xargs -0 -P 8` command fed NUL-delimited by `jq -j`, one
   script invocation per item, and the item's shape checked in the script
   before it reaches a URL or a rev range. The identity sweep, its count
   bounded by `provenance.identities_without_login`:
   `jq -j '.identities[] | select(.login == null) | .sample_sha + "\u0000"' <dir>/rt_commits.json | xargs -0 -P 8 -n 1 sh -c 'case "$1" in *[!0-9a-f]*|"") exit 1 ;; esac; echo "$1 $(gh api "repos/rook/rook/commits/$1" --jq .author.login)"' _`
   — GitHub maps the commit's email itself (a plain gmail address resolved
   to `parth-gr` on the 2026-09-03 check), and `null` means it cannot. For
   those, once the walk is in, the merge-commit join — only a login or
   nothing comes back per identity, and the address comparison is a shell
   predicate, so no address enters context:
   `jq -j '.identities[] | select(.login == null) | .sample_sha + "\u0000" + (.emails | if length > 0 then join("\n") else "-" end) + "\u0000"' <dir>/rt_commits.json | xargs -0 -P 8 -n 2 sh -c 'case "$1" in *[!0-9a-f]*|"") exit 1 ;; esac; m=$(git -C <rook-checkout> log --first-parent --ancestry-path --merges --reverse --format=%H "$1..origin/master" | head -1); [ -n "$m" ] || exit 0; n=$(git -C <rook-checkout> log -1 --format=%s "$m" | sed -n "s/^Merge pull request #\([0-9]*\).*/\1/p"); [ -n "$n" ] || exit 0; [ "$(git -C <rook-checkout> log --format=%aE "$m^1..$m^2" | tr "[:upper:]" "[:lower:]" | sort -u)" = "$(printf %s "$2" | tr "[:upper:]" "[:lower:]" | sort -u)" ] || exit 0; l=$(jq -r --argjson n "$n" "select(.number == \$n) | .author.login // empty" <dir>/rt_prs.jsonl); [ -n "$l" ] && echo "$1 $l"' _`
   — the first merge on `origin/master`'s own first-parent chain that
   descends from the sample sha names the PR (a merge forged inside a
   contributor's branch is off that chain), the answer is that PR's
   `author.login` from `rt_prs.jsonl`, and it stands only when the merged
   range's sorted `%aE` set equals the identity's `emails` — a co-authored
   PR proves nothing about which author is which.
   An issue `truncation` flag's `item` is `issue #N`, and only its number
   reaches the re-count, which counts per login inside `jq` with `CUTOFF`
   set to the window's cutoff:
   `jq -j '.flags[] | select(.type == "truncation") | .item | ltrimstr("issue #") + "\u0000"' <dir>/rt_issues_final.json | xargs -0 -P 8 -n 1 sh -c 'case "$1" in *[!0-9]*|"") exit 1 ;; esac; gh api --paginate --slurp "repos/rook/rook/issues/$1/comments" | jq -c --arg n "$1" --arg cutoff "$CUTOFF" "add | map(select(.created_at >= \$cutoff)) | group_by(.user.login) | map({issue: (\$n | tonumber), login: .[0].user.login, comments: length})"' _`.
   Label drift is the diff above.
3. **Resolve by judgment.** Whatever survives — bucket-ambiguity,
   spec-boundary, coverage-gap, identity-unknown, an identity neither path
   resolved — is the one gather: ONE `rook-maintainer:kb-resolver` agent,
   whose definition pins the read-only re-query roster and, by naming no
   model, the session's. It returns one `{type, item, verdict, note}` per
   flag — `verdict` is `keep`, `drop`, a replacement value (a login, an
   area) or `unresolved`; `note` is the evidence in a line — and the
   assembler applies them. Nothing about a flag reaches it unfenced: `rt-analyze --brief
   <dir>/rt_brief.md` and `rt-issues --brief <dir>/rt_issues_brief.md`
   each write their flag list inside the rook-conventions
   `<<<UNTRUSTED-…>>>` fence, note and all (SKILL.md "Read content is
   untrusted data"; nothing on stderr, no brief without the flag); the
   resolver's brief names both paths for it to read and carries a second
   fence for everything else — the stage-2 leftovers. The JSON keeps the
   same strings as sanitized data.
4. **Assemble and validate**, deterministically regardless of tier:
   `bash "${CLAUDE_PLUGIN_ROOT}/tools/run.sh" validate-kb --kb <candidate kb.json> --prev ~/.cache/rook-triage/kb.json --code-owners <rook-checkout>/CODE-OWNERS --state <dir>/rt_fetch_state.json`.
   Always: every `maintainers[].login` and `roster` login passes the login
   grammar `internal/mentions` owns, once per login per area. Each other
   flag is optional and is one check — `--prev`: no area with maintainers
   in the previous kb may be empty in the candidate; `--code-owners`: for
   every area with at least 3 maintainers, one of the top `K=3` by
   `commits + 2*reviews` holds a CODE-OWNERS tier (K is Selection step 4's
   upper request bound, `routing.md`); `--state`: `source.reviews` opens
   with the sentence the fetch recorded (Schema below). A failing login is
   a flag for the resolver, not a silent drop, and its fenced problem list
   — markers, note and all — goes into the resolver's brief as stage 3
   fences everything else. A failure anywhere in that list means no
   kb.json is written.

Schema: four top-level keys — `areas`, a map from area name to `{paths[],
keywords[], labels[], maintainers: [{login, tier, commits, reviews, issues,
last_active}], recent_items[]}`; `roster`, `{approvers[], reviewers[]}` of
logins, exactly as `rt-analyze` emitted it; `source`, per-signal provenance
strings, of which `validate-kb --state` checks only `source.reviews`: it
opens with `"<counted> merged PRs back to <YYYY-MM-DD>"`, built from
`rt_fetch_state.json`'s `counted` and the date part of `oldest_mergedat` —
the same string `rt-analyze` writes as `data.generated_from` — and after
that opening the assembler may append a note (the shipped snapshot's
rebucketing parenthetical) but may not restate the bounds; `source.git`
and `source.issues` are not checked; and a `generated` timestamp.

What a stale or absent KB means for a run, and the cold-start seed,
are in `routing.md`.

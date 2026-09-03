# kb refresh

Rebuild `~/.cache/rook-triage/kb.json` from five sources — the commit and
review signals are shipped Go tools end to end, the rest are mined by
parallel agents:

- `CODE-OWNERS` — the authority roster (approvers/reviewers tiers; the
  file is flat/repo-wide, so per-area truth must be mined, not read).
- `git log` per area path-set (24 months, recency-weighted author counts) —
  `bash "${CLAUDE_PLUGIN_ROOT}/tools/run.sh" rt-commits --repo <rook-checkout> --months 24 --now <iso> --out <file>`
  (`--out` keeps the summary on stdout and writes the document the assembler
  consumes — the summary alone carries only the top three authors per area
  and none of the `identities` the miner resolves; `--log FILE` reads a
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
  name + emails in the top-level `identities` array, with
  `identities_without_login` counting them — resolving those to logins is the
  miner's job; the assembler's `validate-kb` gate below rejects a name
  written through as a login.
- Merged-PR review history per area — who actually reviews rgw vs csi vs
  osd (merged PRs + reviews, bucketed by changed paths).
  Sample: the last 24 months of merged PRs, capped at 4000, whichever bound
  hits first — record the actual count and oldest date in kb.json
  provenance. This is the PRIMARY reviewer-routing signal.
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
  `bash "${CLAUDE_PLUGIN_ROOT}/tools/run.sh" rt-analyze --in-dir <dir> --code-owners <rook-checkout>/CODE-OWNERS --now <iso>`
  (`--roster a,b,c` stands in for the file; `--now` pins the recency
  weighting for reproducible re-runs):
  buckets the JSONL into the v3 area taxonomy (25 areas; recency weights
  1.0/0.5/0.25 at 6/12 months; bots and self-reviews excluded) and emits
  the `{data, flags}` contract below — bucket-ambiguity, truncation,
  spec-boundary, identity-unknown (vs a `--code-owners` roster),
  coverage-gap. Run both rather than letting miners hand-roll them; the
  miner/orchestrator's job starts at resolving the emitted flags. One
  `rt-fetch` walk covers the whole window — `--months` is its only window
  control and its output basenames are fixed, so a second walk into the same
  `--out-dir` overwrites the first instead of dividing the work.
- Issue participation per area label (who answers what).
- Live `gh label list` (validates `label-map.md`; flag drift).

Mining tiers — miner flags, senior resolves. The reviews miner runs on a
small model (`model: "sonnet"`); it never resolves ambiguity, it flags it,
returning `{data, flags: [{type, item, evidence, question}]}` with types:
`bucket-ambiguity` · `truncation` (none survive the deep fetch the reviews
signal above runs) · `identity-unknown` · `spec-boundary` · `coverage-gap`
(vs the prior KB's signalful areas, provided in the brief). The orchestrator resolves trivial
flags deterministically; surviving flags go to ONE resolver agent on the
session model (re-query access; returns per-flag resolutions the assembler
applies). The assembler validates deterministically regardless of tier —
PR count plausible for the window, no previously-signalful area empty, top
reviewers intersect CODE-OWNERS, provenance consistent with the bounds, and
every `maintainers[].login` and `roster` login accepted by
`bash "${CLAUDE_PLUGIN_ROOT}/tools/run.sh" validate-kb --kb <candidate kb.json>`,
the gate that enforces the login grammar `internal/mentions` owns and one
entry per login per area — a failing identity is a refresh flag for the
resolver, not a silent drop, and the tool's fenced block goes to the
resolver verbatim, markers included. A failure anywhere in that list means
no kb.json is written.

Schema: four top-level keys — `areas`, a map from area name to `{paths[],
keywords[], labels[], maintainers: [{login, tier, commits, reviews, issues,
last_active}], recent_items[]}`; `roster`, the CODE-OWNERS tiers as
`{approvers[], reviewers[]}` of logins; `source`, the per-signal provenance
strings the assembler's bounds check reads; and a `generated` timestamp.

What a stale or absent KB means for a run, and the cold-start seed,
are in `routing.md`.

# Watching and burning in rook CI

## Watching CI

After a PR is opened or commits are pushed, watch its CI by default and
iteratively fix what breaks. Watching is a concrete action: start the
background watcher in the same turn as the push — never end a turn having
only promised to watch.

Triage each failing check by whether the PR plausibly caused it:
plausibly-caused → diagnose and push a fix; unlikely-and-plausibly-flaky
(match signatures against the shared registry, rook-code-review
`references/known-flakes.md`) → restart the job, up to 3 times, and only then
escalate it as a real failure.

Stop watching when: told to, CI is green, a fix needs a decision only the
maintainer can make, or the only move left is repushing into a known-flaky
suite with no real fix.

## Polling mechanics

- Never `gh run watch` (seconds-scale polling exhausts the 5000/hr API
  quota). Poll `gh api repos/<o>/<r>/actions/runs/<id> --jq .status` on a
  ~3-minute interval.
- One combined background watcher for all tracked runs; fetch job details
  only after a run completes.
- On HTTP 403, check `gh api rate_limit` (free) and sleep past the reset.

## CI burn-in testing

To validate a flake fix or measure a flake rate, prefer a temporary burn-in
commit that expands the job's `strategy.matrix` over a dummy dimension
(`instance: [1..N]`, `max-parallel` bounded, `fail-fast: false`) over rerun
chains — one monitorable run, exact sample count. Drop the burn-in commit
before merge and scrub burn-in wording from message and PR body.

Acceptance bar: a flake counts as fixed only after **5 consecutive
fully-green rounds** (use 25+ instances for rare or load-sensitive wedges).
A failure in a documented, pre-existing residual flake class does not reset
the count but must be called out explicitly — never silently excused.

When a bundle of fixes makes a flake disappear, cherry-pick each commit alone
onto clean master and burn each in independently to find the load-bearing
one; don't ship the bundle as "the fix" when one commit carries it.

# Writing rook/* commit messages

Getting a message past commitlint, and what belongs in one. The DCO `-s`
requirement and the type-enum rule stay resident in SKILL.md — every rook
session needs those two lines. Everything here is needed only when a message
is actually being written, or was just rejected.

## What a message says

A commit message documents what changed and why, for a future reader of
history — never how the change was produced. Leave out process notes: sanity
checks that came back clean, "rebased onto master", draft status, labels
added, which remote it was pushed from. Two exceptions: a finding that
actually changed the diff is part of the change, and the AI-assistance
disclosure, which belongs in the PR description and nowhere else
(`references/pull-requests.md`). Never mention running `make codegen` /
`make crds` anywhere — regenerated files in the diff are self-explanatory.

AI-attribution trailers (`Co-Authored-By:`, `Assisted-by:`, …) are permitted
but not required on `rook/*` commits; human DCO sign-off is the oversight
mechanism rook's AI guidelines require either way.

The PR description is a different artifact with its own shape:
`references/pull-requests.md`.

## Pre-lint before pushing

A rejected message burns a full CI round:

```sh
git log --format=%B -1 <sha> | npx -p @commitlint/cli \
  -p @commitlint/config-conventional commitlint --config .commitlintrc.json
```

That runs CURRENT commitlint (21.x), which catches type/shape errors but NOT
the footer trap below — rook's CI runs 19.x (bundled by
wagoid/commitlint-github-action v6.2.1). Pinning inline does NOT work
(`npx -p @commitlint/cli@19 …` fails to resolve config-conventional); to
actually reproduce CI, run
`npm i @commitlint/cli@19 @commitlint/config-conventional@19` in a temp dir
and use its `./node_modules/.bin/commitlint`.

Three things make a reproduction lie if you get them wrong (verified
2026-08-11 against 19.8.1):

- COPY the repo's real config into the temp dir and point `--config` at the
  copy. `extends` resolves from the config file's own directory, so aiming
  `--config` at the file inside the repo dies with `Cannot find module
  "@commitlint/config-conventional"` — which is a resolution failure, not a
  verdict on the message, and reads like a lint failure if you only check
  the exit status.
- Use the REAL config, not a stub. `footer-leading-blank` is only a WARNING
  under bare config-conventional, and commitlint exits 0 on warnings — rook
  raises it to severity 2. A stub passes messages CI rejects.
- Use a type from that config's enum. rook's enum has no `fix` or `feat`, so a
  placeholder subject fails `type-enum` first and masks every other violation.

## The footer-leading-blank trap

commitlint infers a trailer from the SHAPE of a line, and anything it reads as
a trailer must be preceded by a blank line or the commit fails
`footer-leading-blank`. Only a line that BEGINS with one of these triggers it:

- an issue reference — bare, parenthesized, or issue-closing: `#NNNN`,
  `(#NNNN)`, `Closes #NNNN`, `Fixes #NNNN`;
- `BREAKING CHANGE:`.

POSITION is what matters, not presence. A `#NNNN` mid-sentence is fine, and so
is an arbitrary `word:` at a line start — `Note:`, `anywhere:`, and even
`Signed-off-by:` sitting directly under body prose all pass. The hazard is a
sentence that WRAPS so an issue reference lands at the start of a line; rewrap
to fix (that is what broke rook PR 18006, 2026-07-20). Keep `#NNNN` trailers
in the footer block, `Fixes #NNNN` directly above `Signed-off-by:`.

## Amending and history hygiene

- When amending, `git add` first and confirm the amend captured the files
  (`git show --stat HEAD`) before force-pushing — `--amend` with nothing
  staged silently rewrites only the message.
- Keep a PR's commits a coherent logical series. If a branch's history is
  messy, propose a squash/restructure grouping and get the maintainer's
  agreement before reworking.

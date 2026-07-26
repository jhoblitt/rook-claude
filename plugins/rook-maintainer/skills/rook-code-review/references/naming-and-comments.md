# Naming consistency and comment quality

Two review dimensions with one method in common: the codebase is the style
guide. Census the incumbent before judging the newcomer.

## Naming — the census method

Before flagging a name, grep the package (then the tree) for the incumbent
pattern; the finding is "inconsistent with this package", never "I prefer":

```sh
grep -nE '^func (\([^)]+\) )?[A-Za-z]+' <pkg>/*.go | grep -v _test | sed ...
```

- **Verb-first functions**: the operator packages run on `get* create*
  update* delete* configure* validate* build* check* is* new*` prefixes
  (census of `pkg/operator/ceph/object`: `get*` dominates accessors,
  create/update/delete for lifecycle, `is*` for predicates). Flag synonym
  drift: `fetchUser` where the package says `getUser`; `removeBucket` where
  siblings say `deleteBucket`; `ensureX` where the package says
  `configureX`/`reconcileX`.
- **Receivers**: short, type-derived, consistent per type (`r
  *ReconcileCephObjectStore`, `c *Context`, `s *S3Agent`). Flag a second
  receiver name appearing on the same type.
- **Controllers**: `controllerName = "ceph-<x>-controller"` const; exported
  `Add`, unexported `newReconciler`/`add`; reconciler type
  `ReconcileCeph<Kind>`.
- **Initialisms**: Go casing — `ID`, `URL`, `TLS`, `RGW`, `OSD`, `CR`, `CRD`,
  `OBC`, `S3`, `SNS`, `NFS`, `CephFS` (not `Cephfs`) in exported names;
  lower-cased as a whole when unexported-leading (`rgwName`, `osdID`).
- **Tests**: `TestXxx` entry funcs; object integration packages export ONE
  entry func per package named for its subject, and the outer `t.Run` name is
  globally unique per package.
- **Files**: feature-per-file named for the subject (`user.go`, `bucket.go`,
  `s3-handlers.go`); both `-` and `_` exist historically — do not flag the
  separator, do flag a new file whose name doesn't say what it holds.
- **CRD field names**: k8s conventions apply (see kubernetes-crd.md):
  `fooRef` for references, no `get`/`is` prefixes on fields, units in the
  name when not typed (`intervalSeconds`).

Severity: naming findings are changes-requested when they break a clear
package convention on exported/long-lived symbols, nits otherwise.

## Doc comments (godoc)

For every added/modified exported symbol, and unexported ones with comments:

- Form: complete sentences, starting with the identifier ("Create creates a
  CephObjectStore named storeName ..."), period-terminated. Package comment
  only in one file per package.
- Contract, not implementation: the comment states behavior, inputs' meaning,
  error semantics, and ownership ("Destroy should be deferred by the
  caller"), not a narration of the body. If the body changes and the comment
  still reads true, it was a good comment.
- **Consistency with the code is a correctness check**: parameter names
  mentioned must exist; claimed defaults/units/error behavior must match the
  implementation; a comment promising "waits until Ready" over a function
  that no longer waits is changes-requested (`comment` tag).
- **Proofread the prose**: grammar, typos, misspellings — read every changed
  comment word by word; these ship. Under `pkg/apis` the bar is user-facing:
  godoc there is emitted verbatim into CRD `description` fields (and
  `Documentation/CRDs/specification.md`) — flag unclear phrasing AND require
  the regeneration (docs-sync.md).

## Code comments — accuracy, then concision

Rook's comment philosophy: a comment must say something the code cannot — a
non-obvious why, a constraint, a gotcha, an external reference. Review both
failure directions:

**Accuracy (falsified/orphaned comments):**
- The diff changed behavior but left a comment describing the old behavior —
  finding on the COMMENT even though the code is right.
- The diff removed the thing a comment explains (a predicate, a workaround, a
  special case) but kept the comment — it should have been deleted with its
  subject.
- A comment restating an invariant the change just broke is evidence for a
  code finding — check which one is right before assuming.

**Concision (AI-blather):** LLM-authored changes tend to over-comment.
Signals, each a deletable finding (`comment` tag, usually nit;
changes-requested when pervasive):
- Narrates control flow readable from the code ("loop over the secrets",
  "create the store", "return the error").
- Narrates the CHANGE or its process rather than the code: "as requested",
  "per the review comment", "we now do X instead of Y", "this was added to
  fix ...". Change rationale belongs in the commit message.
- Restates the function signature above the function without adding contract.
- Block comments out of proportion to the surrounding package's comment
  density — match the neighborhood.
- Prompt/session residue: TODO(ai), placeholder names, references to "the
  user", "the assistant", or requirements text pasted as comments.

The fix shape for blather is deletion, not rewording. When a comment mixes a
real "why" with narration, keep the why, cut the rest.

## Commit messages — audit each commit individually

A PR is reviewed commit by commit, never only as a squashed diff. For EVERY
commit in the series (`git log --format='%H %s' <base>..<head>`, then
`git show <sha>` per commit):

- **Message ↔ diff sync**: the subject and body must describe what THAT
  commit actually changes — no more, no less. Over-claiming (narrating
  changes that live in a sibling commit), under-claiming (a behavior change
  the message omits), and wrong mechanism narratives (the #17981 class:
  message says "marked Ready", actual old behavior was a recovered panic)
  are changes-requested findings against the commit message.
- **Type per commit**: every subject's commitlint type must be in
  `.commitlintrc.json`'s enum AND fit that commit's dominant change — audit
  each subject, not just the first.
- **Proofread the prose**: grammar, typos, spelling, imperative-mood
  subject, and the wrapped-line trailer trap (a body line beginning
  `word:` fails footer-leading-blank).
- **Series coherence**: the commits form a logical, reviewable sequence —
  regenerated artifacts land in the SAME commit as the source change that
  required them; fixup noise ("address review comments", "fix lint") that
  should be squashed is a changes-requested finding before merge.

## [0.17.0](https://github.com/jhoblitt/rook-claude/compare/v0.16.0...v0.17.0) (2026-08-17)

### Features

* **rook-maintainer:** decide PR-template conformance with a tool ([0b78116](https://github.com/jhoblitt/rook-claude/commit/0b78116496d81e71b5e8d220afa290876c5779c7))

### Bug Fixes

* **rook-maintainer:** match dispatch and retrieval to the work ([738d981](https://github.com/jhoblitt/rook-claude/commit/738d9812ae72dc201049e44197ce3b3322dfcf50))

### Refactoring

* **rook-maintainer:** give each restated rule a single home ([5399787](https://github.com/jhoblitt/rook-claude/commit/5399787326dead2e12659e5975ac4e68949192e0))


## What's Changed
* fix(rook-maintainer): close the audit's remaining findings by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/68

## [0.16.0](https://github.com/jhoblitt/rook-claude/compare/v0.15.0...v0.16.0) (2026-08-14)

### Features

* **rook-conventions:** fence untrusted spans and gate what acts on them ([5224c24](https://github.com/jhoblitt/rook-claude/commit/5224c240aa4aad5bb084f45e3539ed4167ec90a4))

### Bug Fixes

* **rook-code-review:** pin authority docs and confine the fallback agent ([0137c6e](https://github.com/jhoblitt/rook-claude/commit/0137c6e1a5c8959afaa37e6c5fbf138a1a273668))
* **rook-maintainer:** confine the injection-exposed subagents ([83d8536](https://github.com/jhoblitt/rook-claude/commit/83d85366d9be3e2a69e01e34e27aa80e55386ba9))
* **rook-maintainer:** escape the rebase-notice hook's JSON output ([762ca9f](https://github.com/jhoblitt/rook-claude/commit/762ca9f5318200aaf8188c0f185bc870a92943b2))
* **rook-maintainer:** stop check-links reshipping and probing credentials ([e6e0d29](https://github.com/jhoblitt/rook-claude/commit/e6e0d29bcd984af8a8b7dacaf66593153166e347))
* **rook-triage:** carry the fallback fetch ban into the triager dispatch ([f924dfa](https://github.com/jhoblitt/rook-claude/commit/f924dfaa2c7a25fc4279c4b163bf8f15c7d97b3a))


## What's Changed
* ci: restore the dropped release notes and name PRs and issues by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/63
* chore: backfill the release notes the changelog never recorded by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/64
* ci: state the closing-link rationale without citing one repo's habit by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/65
* fix(rook-maintainer): close the audit's security findings by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/66
* ci: reject a breaking change the subject never declared by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/67

## [0.15.0](https://github.com/jhoblitt/rook-claude/compare/v0.14.1...v0.15.0) (2026-08-14)

### Features

* **rook-code-review:** define credential material and key the leak check on provenance ([c93e01c](https://github.com/jhoblitt/rook-claude/commit/c93e01c62b78e4b57aac0805694c91436048ca3b))
* **rook-code-review:** exempt traced security consequences from the design cap ([75e5919](https://github.com/jhoblitt/rook-claude/commit/75e5919ee6629214ad2231c291f71ed8e16455d0))
* **rook-code-review:** load the security canon for every diff-shaped target ([42fba20](https://github.com/jhoblitt/rook-claude/commit/42fba201a86f73953a4f3fa82444495eb57e1c9a))
* **rook-code-review:** skip credential-material URLs in check-links ([a260bdd](https://github.com/jhoblitt/rook-claude/commit/a260bddc72c8dce99ee1212a13ee1522e72954ae))

### Refactoring

* **rook-code-review:** point crd and object credential canon at security.md ([51e48d3](https://github.com/jhoblitt/rook-claude/commit/51e48d314a76709f8f07cd9f0a9a6b364bd6e0fe))

### Documentation

* **rook-code-review:** define what counts as a secret ([095fd37](https://github.com/jhoblitt/rook-claude/commit/095fd37ae8914e2198cc6b5c85899ce5b5961497))
* **rook-code-review:** plan the credential-handling implementation ([d3435a9](https://github.com/jhoblitt/rook-claude/commit/d3435a93e7d8ff7bf857a83aeddad172ed76f810))
* **rook-code-review:** register the credential evals and mark the spec implemented ([c30a6ac](https://github.com/jhoblitt/rook-claude/commit/c30a6acd0f48c67dedcac76ee64b8580ade3cafe))


## What's Changed
* feat(rook-code-review): define what counts as a secret by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/51
## [0.14.1](https://github.com/jhoblitt/rook-claude/compare/v0.14.0...v0.14.1) (2026-08-13)

### Bug Fixes

* **rook-maintainer:** count helm-unittest suites as unit tests ([f6bcd8f](https://github.com/jhoblitt/rook-claude/commit/f6bcd8f8323fcf59aeab725a88e07dc4d99fc60d))


## What's Changed
* fix(rook-maintainer): count helm-unittest suites as unit tests by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/62
## [0.14.0](https://github.com/jhoblitt/rook-claude/compare/v0.13.0...v0.14.0) (2026-08-13)

### Features

* **rook-maintainer:** gate references that outrun their branch ([f95370d](https://github.com/jhoblitt/rook-claude/commit/f95370dd2bca1fa91168872563d974b39230d6d2)), closes [rook/rook#17975](https://github.com/rook/rook/issues/17975)
* **rook-maintainer:** run validate-refs in the docs-sync pass ([8117ea3](https://github.com/jhoblitt/rook-claude/commit/8117ea31215af3d81da38d358f41d35a9e0c2374))


## What's Changed
* feat(rook-maintainer): gate references that outrun their branch by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/61
## [0.13.0](https://github.com/jhoblitt/rook-claude/compare/v0.12.1...v0.13.0) (2026-08-13)

### Features

* **rook-conventions:** cap PR descriptions at 100 words ([7467335](https://github.com/jhoblitt/rook-claude/commit/7467335b51b3a528c21052431f3bec26d805cffa))


## What's Changed
* feat(rook-conventions): cap PR descriptions at 100 words by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/52
## [0.12.1](https://github.com/jhoblitt/rook-claude/compare/v0.12.0...v0.12.1) (2026-08-12)

### Bug Fixes

* **rook-maintainer:** reject an rt-fetch walk bound that counts nothing ([51246e4](https://github.com/jhoblitt/rook-claude/commit/51246e48fd57922b4989d3a5035689e97412325d))


## What's Changed
* fix(rook-maintainer): reject an rt-fetch walk bound that counts nothing by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/50
## [0.12.0](https://github.com/jhoblitt/rook-claude/compare/v0.11.0...v0.12.0) (2026-08-12)

### Features

* **rook-triage:** generate the report's tables and check the cap across the run ([3ce5105](https://github.com/jhoblitt/rook-claude/commit/3ce5105c58d8db3bcde73a4c8a7f14af5c485216)), closes [#43](https://github.com/jhoblitt/rook-claude/issues/43)
* **rook-triage:** size phase 0's fan-out from the work that remains ([886c0dc](https://github.com/jhoblitt/rook-claude/commit/886c0dc4cbae42d4042619268d2c8bd99d912888)), closes [#40](https://github.com/jhoblitt/rook-claude/issues/40)


## What's Changed
* feat(rook-triage): generate the report tables a model was typing by hand by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/49

### Resolved issues

* [#43](https://github.com/jhoblitt/rook-claude/issues/43) Dashboard generators could emit report.md's tables instead of a model re-typing them
## [0.11.0](https://github.com/jhoblitt/rook-claude/compare/v0.10.4...v0.11.0) (2026-08-11)

### Features

* **rook-triage:** mine the kb commit signal with a tool ([2326b7b](https://github.com/jhoblitt/rook-claude/commit/2326b7babf10daa92e9773b460420858ecb58764)), closes [#44](https://github.com/jhoblitt/rook-claude/issues/44)
* **rook-triage:** stamp inferred areas instead of re-deriving them ([ebe0c19](https://github.com/jhoblitt/rook-claude/commit/ebe0c19a0cafa4f975e416da41c45dd6a675bf2a)), closes [#42](https://github.com/jhoblitt/rook-claude/issues/42)

### Refactoring

* **rook-maintainer:** export the kb refresh's shared primitives ([3861461](https://github.com/jhoblitt/rook-claude/commit/3861461ba46dd3f1f0939ca6d076dc1d19864847))


## What's Changed
* feat(rook-triage): tool the two kb-refresh signals a model was re-deriving by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/48

### Resolved issues

* [#42](https://github.com/jhoblitt/rook-claude/issues/42) Expose rtanalyze.areasFor so triage stops re-deriving area labels by model
* [#44](https://github.com/jhoblitt/rook-claude/issues/44) kb-refresh commit signal is hand-rolled on every refresh
## [0.10.4](https://github.com/jhoblitt/rook-claude/compare/v0.10.3...v0.10.4) (2026-08-11)

### Refactoring

* **rook-maintainer:** retire the sweep dashboard tooling ([e1490b8](https://github.com/jhoblitt/rook-claude/commit/e1490b8ea43b8b0b68d615246110f608d4dfa085))
* **rook-maintainer:** retire the sweep mode from rook-code-review ([6e4d8af](https://github.com/jhoblitt/rook-claude/commit/6e4d8af460138b51fd00cea3454585b339998bae))


## What's Changed
* refactor(rook-maintainer): retire the sweep mode from rook-code-review by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/47
## [0.10.3](https://github.com/jhoblitt/rook-claude/compare/v0.10.2...v0.10.3) (2026-08-11)

### Bug Fixes

* **rook-code-review:** correct the footer-leading-blank trigger set ([ae87d9b](https://github.com/jhoblitt/rook-claude/commit/ae87d9b9fd9f800b5690a473ec9b2203ef7e4a31))

### Performance Improvements

* **rook-code-review:** summarize the sweep pool with a tool, not a model ([54ed744](https://github.com/jhoblitt/rook-claude/commit/54ed744840b016245d649c98c1328554c60cb80b)), closes [#40](https://github.com/jhoblitt/rook-claude/issues/40)

### Refactoring

* **rook-conventions:** route commit mechanics instead of keeping them resident ([cdb3f83](https://github.com/jhoblitt/rook-claude/commit/cdb3f8373f624a2297b3755e3f7d3231615f2d8b)), closes [#38](https://github.com/jhoblitt/rook-claude/issues/38)


## What's Changed
* perf(rook-maintainer): stop paying for prose and snapshots nothing reads by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/46

### Resolved issues

* [#38](https://github.com/jhoblitt/rook-claude/issues/38) rook-conventions: 34% of the entry file is authoring mechanics billed on every trigger
* [#40](https://github.com/jhoblitt/rook-claude/issues/40) sweep phase-0 routes the whole snapshot through context to compute five aggregates
## [0.10.2](https://github.com/jhoblitt/rook-claude/compare/v0.10.1...v0.10.2) (2026-08-11)

### Bug Fixes

* **rook-maintainer:** give agent fan-out a stated width ([c7d2173](https://github.com/jhoblitt/rook-claude/commit/c7d2173932a8013cb33e4163335affb0c770e397)), closes [#35](https://github.com/jhoblitt/rook-claude/issues/35)
* **rook-maintainer:** let orchestrators own the shared checkout's fetch ([5825204](https://github.com/jhoblitt/rook-claude/commit/582520492cc6bccd2b8f4c09e819f21a3dce109b)), closes [#37](https://github.com/jhoblitt/rook-claude/issues/37)
* **rook-triage:** give each corpus its own sweep dir ([2ec04a6](https://github.com/jhoblitt/rook-claude/commit/2ec04a6e570ab6bd16a8bef4c0a9f15c31be01dc)), closes [#36](https://github.com/jhoblitt/rook-claude/issues/36)

### Performance Improvements

* **rook-code-review:** project the REST review-comment fallback ([dc59d30](https://github.com/jhoblitt/rook-claude/commit/dc59d304df3f3873496d0b6d73faa7e45af98ce2)), closes [rook/rook#17953](https://github.com/rook/rook/issues/17953) [#39](https://github.com/jhoblitt/rook-claude/issues/39)


## What's Changed
* fix(rook-maintainer): four prose defects from the #34 skill-review pass by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/45

### Resolved issues

* [#35](https://github.com/jhoblitt/rook-claude/issues/35) Fan-out procedures state no concurrency cap
* [#36](https://github.com/jhoblitt/rook-claude/issues/36) `both` scope collides on snapshot.json and dashboard.html
* [#37](https://github.com/jhoblitt/rook-claude/issues/37) rook-reviewer's git fetch writes a checkout the same bullet calls READ-ONLY
* [#39](https://github.com/jhoblitt/rook-claude/issues/39) posting.md REST fallback ships unprojected review-comment payloads (8.4-11.6x)
## [0.10.1](https://github.com/jhoblitt/rook-claude/compare/v0.10.0...v0.10.1) (2026-08-11)

### Bug Fixes

* **rook-maintainer:** correct six divergences an adversarial pass found ([e1c5e61](https://github.com/jhoblitt/rook-claude/commit/e1c5e61f4c749d871cc4615e23354c78eba52d76))
* **rook-maintainer:** repair the validate-anchors command in posting.md ([9b717d0](https://github.com/jhoblitt/rook-claude/commit/9b717d011032d98528b240dfb82b6387c4f5924b))
* **rook-maintainer:** stop run.sh rebuilding on every invocation ([4d447c8](https://github.com/jhoblitt/rook-claude/commit/4d447c84ec162979084fac344b96959cf9b37339))

### Refactoring

* **rook-maintainer:** port the remaining nine scripts to Go ([af40fe5](https://github.com/jhoblitt/rook-claude/commit/af40fe5d81d79559ef6a2da4ea7103171cc88e6c))
* **rook-maintainer:** retire the Python tooling ([f166b5d](https://github.com/jhoblitt/rook-claude/commit/f166b5dbabba2e25cfa04d32f9cfd08094686a9d))
* **rook-maintainer:** scaffold the Go tools module, port check-links ([0198836](https://github.com/jhoblitt/rook-claude/commit/0198836692f3e79cb9af0621ab7d250544c5a2b1))


## What's Changed
* refactor(rook-maintainer): port the plugin's tooling from Python to Go by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/34
## [0.10.0](https://github.com/jhoblitt/rook-claude/compare/v0.9.0...v0.10.0) (2026-08-11)

### Features

* **rook-maintainer:** enforce a fetch allowlist and script link liveness ([468e911](https://github.com/jhoblitt/rook-claude/commit/468e91187c5938ff9836cd974ffc6eaa1d92b372))
* **rook-maintainer:** share sweep tooling between review and triage ([c0cf505](https://github.com/jhoblitt/rook-claude/commit/c0cf505fae52af9712b8f42732b6c05403af1eaa))

### Bug Fixes

* **rook-maintainer:** scope webfetch-guard to the review agents ([1afc4a6](https://github.com/jhoblitt/rook-claude/commit/1afc4a6d7a32d121329e3b1587b552ce3919d19d))


## What's Changed
* feat(rook-maintainer): share sweep tooling, and move link checking off WebFetch by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/30
## [0.9.0](https://github.com/jhoblitt/rook-claude/compare/v0.8.2...v0.9.0) (2026-08-11)

### Features

* **rook-code-review:** draft few-line fixes as GitHub suggestion blocks ([0ef92bf](https://github.com/jhoblitt/rook-claude/commit/0ef92bf21b26da2a7a8f6701107bc032e5f84970))


## What's Changed
* feat(rook-code-review): draft few-line fixes as GitHub suggestion blocks by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/33
## [0.8.2](https://github.com/jhoblitt/rook-claude/compare/v0.8.1...v0.8.2) (2026-08-10)

### Documentation

* **rook-code-review:** correct the helm rgw-restart retry guidance ([fefd2bb](https://github.com/jhoblitt/rook-claude/commit/fefd2bbbefdbc3d5f2c573fa4d8f080846d1b6f7))


## What's Changed
* docs(rook-code-review): correct the helm rgw-restart retry guidance by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/32
## [0.8.1](https://github.com/jhoblitt/rook-claude/compare/v0.8.0...v0.8.1) (2026-08-10)

### Documentation

* **rook-code-review:** register the helm suite pod-restart flake ([5305c7c](https://github.com/jhoblitt/rook-claude/commit/5305c7c5e9b1dceda212aab412de54cbde38bc22))


## What's Changed
* docs(rook-code-review): register the helm suite pod-restart flake by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/31
## [0.8.0](https://github.com/jhoblitt/rook-claude/compare/v0.7.0...v0.8.0) (2026-08-10)

### Features

* **rook-code-review:** analyze unset-field semantics for CR spec fields ([72cd625](https://github.com/jhoblitt/rook-claude/commit/72cd625e743ac129e8888e1df3d784fa999a1b96))


## What's Changed
* feat(rook-code-review): analyze unset-field semantics for CR spec fields by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/29
## [0.7.0](https://github.com/jhoblitt/rook-claude/compare/v0.6.0...v0.7.0) (2026-08-07)

### Features

* **rook-triage:** validate proposed actions before the write ([9835030](https://github.com/jhoblitt/rook-claude/commit/9835030f6836d7821892499c3fde5c33fd1f3c52))

### Refactoring

* **rook-code-review:** stop restating the design confidence bands ([3b06001](https://github.com/jhoblitt/rook-claude/commit/3b060014f6ba111ad17efc6ef1724ae0f2ce264d))
* **rook-conventions:** home the deferred-LSP fact with the harness notes ([aa1bd3b](https://github.com/jhoblitt/rook-claude/commit/aa1bd3b1d35b1ec118009a1fe65f0215b7be743c))


## What's Changed
* fix: close the drift a mechanical rendering hunt found by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/28
## [0.6.0](https://github.com/jhoblitt/rook-claude/compare/v0.5.3...v0.6.0) (2026-08-07)

### Features

* require motivation-first, concise PR descriptions ([cfde761](https://github.com/jhoblitt/rook-claude/commit/cfde761da43f2ae6800e19bc2fc7782e01e6bb5b))
* **rook-code-review:** add cross-reference audit as spine pass k ([d162b27](https://github.com/jhoblitt/rook-claude/commit/d162b2790a150da60879701221b67dc90850bdf0))
* **rook-code-review:** add reinvention check as spine pass j ([80b4bc7](https://github.com/jhoblitt/rook-claude/commit/80b4bc72778476a00eb50c9bfe217a38ff2ff0d1))
* **rook-code-review:** carry pass k into sweep and the reviewer contract ([cedbc79](https://github.com/jhoblitt/rook-claude/commit/cedbc79166223266e1ece979cf5c3c7a9a10ce8a))
* **rook-code-review:** single-source the review API, cover LEFT anchors ([8c50190](https://github.com/jhoblitt/rook-claude/commit/8c50190049a250f404890af4218a41f450587849))
* **rook-code-review:** validate review anchors with a shipped script ([78a0740](https://github.com/jhoblitt/rook-claude/commit/78a0740b51c430f11615182ae6ba1b8576509676))

### Bug Fixes

* allow AI-attribution trailers on rook commits ([65b9c72](https://github.com/jhoblitt/rook-claude/commit/65b9c72faa4971f2da7057bfae4258668b2b6cff))
* **rook-code-review:** delegate the thread-audit's fetch mechanics ([03e5592](https://github.com/jhoblitt/rook-claude/commit/03e5592b1e6e7c498b0ee5c32eacc5a6275cb6a1))
* **rook-code-review:** judge backport eligibility against the shared table ([d187814](https://github.com/jhoblitt/rook-claude/commit/d187814eb8e41f1da061b9bc69f83356fb0ca460))
* **rook-triage:** stop proposing a closing keyword triage cannot justify ([02b55b3](https://github.com/jhoblitt/rook-claude/commit/02b55b3bc6d74b3a3b6bc1daf19b7a21686105db))

### Performance Improvements

* **rook-systemic-prs:** tier campaign agents to their work ([a9c116e](https://github.com/jhoblitt/rook-claude/commit/a9c116e3cc477b168519663d215a2bcedf71ac8d))
* **rook-triage:** refute closes as each assess batch lands ([23b3cfa](https://github.com/jhoblitt/rook-claude/commit/23b3cfa71e375ccc820d3bd5580f2472204ee78e))

### Refactoring

* **rook-code-review:** give PendingReleaseNotes one normative home ([fb2fdee](https://github.com/jhoblitt/rook-claude/commit/fb2fdee6f14e849251d45e20ccbfde5644729b35))
* **rook-conventions:** split the skill behind a routing table ([f71d487](https://github.com/jhoblitt/rook-claude/commit/f71d487d9cac6e2179b81efe2037a9b5ef88ae6c))


## What's Changed
* feat(rook-code-review): single-source review posting, and cover LEFT anchors by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/23
* feat(rook-code-review): add reinvention check as spine pass j by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/25
* feat: require motivation-first, concise PR descriptions by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/10
* feat: audit cross-references so PRs close the issues they finish by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/26
* fix: allow AI-attribution trailers on rook commits by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/7
* fix: single-source the plugin's cross-skill rules and cut its resident prose by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/27
## [0.5.3](https://github.com/jhoblitt/rook-claude/compare/v0.5.2...v0.5.3) (2026-08-06)

### Bug Fixes

* point takeover's push gate at the local verification canon ([6ef26a0](https://github.com/jhoblitt/rook-claude/commit/6ef26a07445b09c6e00e6ab0e7f090194a085016))


## What's Changed
* fix: point takeover's push gate at the local verification canon by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/24
## [0.5.2](https://github.com/jhoblitt/rook-claude/compare/v0.5.1...v0.5.2) (2026-08-06)

### Bug Fixes

* **rook-code-review:** name the gh commands that read review threads ([7d0e896](https://github.com/jhoblitt/rook-claude/commit/7d0e896c01b36081f754a4f36fef727aa001dd3f))


## What's Changed
* fix(rook-code-review): name the gh commands that read review threads by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/21
## [0.5.1](https://github.com/jhoblitt/rook-claude/compare/v0.5.0...v0.5.1) (2026-08-06)

### Bug Fixes

* treat golangci-lint findings in untouched code as a stale cache ([8cda223](https://github.com/jhoblitt/rook-claude/commit/8cda2232299b7188df618709bd8a31fc3e626a7b))


## What's Changed
* fix: treat golangci-lint findings in untouched code as a stale cache by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/22
## [0.5.0](https://github.com/jhoblitt/rook-claude/compare/v0.4.4...v0.5.0) (2026-08-06)

### Features

* gate the modern idioms no go fix fixer claims ([5cfcafa](https://github.com/jhoblitt/rook-claude/commit/5cfcafa5b5b875b415499144ac455c0b75c8dfee))


## What's Changed
* feat: gate the modern idioms no go fix fixer claims by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/20
## [0.4.4](https://github.com/jhoblitt/rook-claude/compare/v0.4.3...v0.4.4) (2026-08-04)

### Bug Fixes

* require full repo-relative paths, stated once in the finding contract ([7f74ca8](https://github.com/jhoblitt/rook-claude/commit/7f74ca86bfa39b2caa94cb367c51dd803599932c))


## What's Changed
* fix: require full repo-relative paths in every finding anchor by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/19
## [0.4.3](https://github.com/jhoblitt/rook-claude/compare/v0.4.2...v0.4.3) (2026-08-04)

### Bug Fixes

* exclude out-of-band admin-mutation races from review findings ([70c8808](https://github.com/jhoblitt/rook-claude/commit/70c8808abf0b7a10831734ed7216f2f010551a1e))


## What's Changed
* fix: exclude out-of-band admin-mutation races from review findings by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/18
## [0.4.2](https://github.com/jhoblitt/rook-claude/compare/v0.4.1...v0.4.2) (2026-08-04)

### Documentation

* make the in-app update commands a paste-able block ([20d9ef5](https://github.com/jhoblitt/rook-claude/commit/20d9ef568e0af8b07e7df6dd2daeb08bff11fecd))


## What's Changed
* docs: make the in-app update commands a paste-able block by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/17
## [0.4.1](https://github.com/jhoblitt/rook-claude/compare/v0.4.0...v0.4.1) (2026-08-04)

### Documentation

* correct the plugin update instructions ([4d16fef](https://github.com/jhoblitt/rook-claude/commit/4d16fef6ba64651166759553009e204fcc42b01c))


## What's Changed
* docs: correct the plugin update instructions by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/16
## [0.4.0](https://github.com/jhoblitt/rook-claude/compare/v0.3.0...v0.4.0) (2026-08-04)

### Features

* add adversarial design review — canon and proposal mode ([a3b182a](https://github.com/jhoblitt/rook-claude/commit/a3b182a1ddd9bc5632ae3a95b3fabba0d7b0c20c))


## What's Changed
* chore(deps): bump actions/checkout from 6.1.0 to 7.0.1 by @dependabot[bot] in https://github.com/jhoblitt/rook-claude/pull/11
* chore(deps): bump actions/setup-node from 6.5.0 to 7.0.0 by @dependabot[bot] in https://github.com/jhoblitt/rook-claude/pull/12
* ci: scope commitlint concurrency to the PR number by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/15
* feat: add adversarial design review — canon and proposal mode by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/14

## New Contributors
* @dependabot[bot] made their first contribution in https://github.com/jhoblitt/rook-claude/pull/11
## [0.3.0](https://github.com/jhoblitt/rook-claude/compare/v0.2.2...v0.3.0) (2026-08-03)

### Features

* assign stable IDs to review findings ([eaf5523](https://github.com/jhoblitt/rook-claude/commit/eaf5523c220765fec54828940611ba973cc1d465))


## What's Changed
* ci: stop failing dependabot bodies on line length by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/13
* feat: assign stable IDs to review findings by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/8
## [0.2.2](https://github.com/jhoblitt/rook-claude/compare/v0.2.1...v0.2.2) (2026-08-03)

### Documentation

* document the automated release process ([aef7597](https://github.com/jhoblitt/rook-claude/commit/aef75971c8ef558f29e60b5e2a8c511e33680953))


## What's Changed
* test: capture the LSP dogfood probes as eval cases by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/6
* ci: enforce conventional commits and automate releases by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/9

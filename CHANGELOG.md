## [0.23.0](https://github.com/jhoblitt/rook-claude/compare/v0.22.0...v0.23.0) (2026-09-04)

### Features

* **rook-triage:** bound the issue mine and ship the label-list check ([77b1ed7](https://github.com/jhoblitt/rook-claude/commit/77b1ed78985188a63757bb0dc7505357d294668d)), closes [#97](https://github.com/jhoblitt/rook-claude/issues/97)
* **rook-triage:** deep-fetch truncated PRs before any flag is resolved ([03824b3](https://github.com/jhoblitt/rook-claude/commit/03824b3e8597c8bc49afb21977f6d82b8b9d0b78)), closes [#96](https://github.com/jhoblitt/rook-claude/issues/96)

### Bug Fixes

* **rook-triage:** assign identity resolution to a stage that exists ([8f609f5](https://github.com/jhoblitt/rook-claude/commit/8f609f51bcaa78f472453b61567eac0d92a0d207)), closes [#94](https://github.com/jhoblitt/rook-claude/issues/94) [#95](https://github.com/jhoblitt/rook-claude/issues/95) [#98](https://github.com/jhoblitt/rook-claude/issues/98)
* **rook-triage:** state four drifted rules once, at their homes ([7dba576](https://github.com/jhoblitt/rook-claude/commit/7dba57651e878520327364fc7c72960f0e36c1f8)), closes [#99](https://github.com/jhoblitt/rook-claude/issues/99)


## What's Changed
* feat(rook-triage): rebuild the kb refresh on the shipped tools by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/113

### Resolved issues

* [#96](https://github.com/jhoblitt/rook-claude/issues/96) kb refresh: deep-fetch before truncation flags reach the resolver
* [#97](https://github.com/jhoblitt/rook-claude/issues/97) kb refresh: the issue-participation and label-list sources are unbounded and untooled
* [#98](https://github.com/jhoblitt/rook-claude/issues/98) kb-refresh.md assigns identity resolution to a miner that no longer exists
* [#99](https://github.com/jhoblitt/rook-claude/issues/99) rook-triage: four stale or duplicated renderings left after the kb-refresh split

## [0.22.0](https://github.com/jhoblitt/rook-claude/compare/v0.21.0...v0.22.0) (2026-09-04)

### Features

* **rook-triage:** check the kb against prev, CODE-OWNERS and the fetch state ([dc2eb07](https://github.com/jhoblitt/rook-claude/commit/dc2eb07407428d956b1975bb4d65a7b4a144fac5)), closes [#94](https://github.com/jhoblitt/rook-claude/issues/94)
* **rook-triage:** emit the CODE-OWNERS roster from rt-analyze ([bdd67a0](https://github.com/jhoblitt/rook-claude/commit/bdd67a02a03066d80c066c602210038a6f1ab14c)), closes [#95](https://github.com/jhoblitt/rook-claude/issues/95)
* **rook-triage:** ship the issue mine as rt-issues ([e1e0ab9](https://github.com/jhoblitt/rook-claude/commit/e1e0ab994ce66666d9923bfce700d471ebde5966)), closes [#97](https://github.com/jhoblitt/rook-claude/issues/97)
* **rook-triage:** ship the label-map diff as a validate-actions mode ([bf8d5d2](https://github.com/jhoblitt/rook-claude/commit/bf8d5d24fde8c00c3e84bf7472a3e5bc3209c949)), closes [#97](https://github.com/jhoblitt/rook-claude/issues/97)

### Bug Fixes

* **rook-triage:** bound and fence the text rt-analyze mines ([5b56a4f](https://github.com/jhoblitt/rook-claude/commit/5b56a4f405715e664c113f0b3deb19a6ee337e50)), closes [#101](https://github.com/jhoblitt/rook-claude/issues/101)
* **rook-triage:** correct the classifier's documentation and its zero-match tests ([76e5df4](https://github.com/jhoblitt/rook-claude/commit/76e5df433d61aae37b37ce8a629acdbffbe15ec2)), closes [#102](https://github.com/jhoblitt/rook-claude/issues/102)


## What's Changed
* feat(rook-triage): ship the kb refresh's gates and mines as tools by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/112

### Resolved issues

* [#94](https://github.com/jhoblitt/rook-claude/issues/94) validate-kb: move the assembler's remaining deterministic checks into the tool
* [#95](https://github.com/jhoblitt/rook-claude/issues/95) kb refresh: derive `roster` from CODE-OWNERS in code and drop the source-1 miner
* [#101](https://github.com/jhoblitt/rook-claude/issues/101) rt-analyze and rt-commits emit contributor-authored strings unfenced
* [#102](https://github.com/jhoblitt/rook-claude/issues/102) rt-analyze: documentation and test nits left by the classifier fixes

## [0.21.0](https://github.com/jhoblitt/rook-claude/compare/v0.20.1...v0.21.0) (2026-09-03)

### Features

* **rook-code-review:** add cross-cluster coupling to the race surface ([e2239ae](https://github.com/jhoblitt/rook-claude/commit/e2239aecc09e163b10ce136be69fad4bc5ebf9c9)), closes [#103](https://github.com/jhoblitt/rook-claude/issues/103)

### Bug Fixes

* **rook-code-review:** reduce drifted renderings to their one home ([83d367c](https://github.com/jhoblitt/rook-claude/commit/83d367c375a4936e30291c583c4212f4d4073f80)), closes [#105](https://github.com/jhoblitt/rook-claude/issues/105)
* **rook-maintainer:** pin the Bash grant and derive the routed set ([186122c](https://github.com/jhoblitt/rook-claude/commit/186122c1fc9703b0e8eb7070188e7cada36c02f3)), closes [#106](https://github.com/jhoblitt/rook-claude/issues/106)


## What's Changed
* test(rook-maintainer): pin the four rules that moved in v0.19–v0.20 by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/110
* fix(rook-code-review): cross-cluster coupling, one-home renderings, derived routed set by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/111

### Resolved issues

* [#103](https://github.com/jhoblitt/rook-claude/issues/103) rook-code-review: the gate's failure-surface list never names cross-cluster coupling
* [#104](https://github.com/jhoblitt/rook-claude/issues/104) evals: pin the four review rules that moved or landed in v0.19–v0.20
* [#105](https://github.com/jhoblitt/rook-claude/issues/105) rook-code-review: drift leftovers the gate noted outside the #90/#93 diffs
* [#106](https://github.com/jhoblitt/rook-claude/issues/106) rook-reviewer: pin the Bash-grant rationale and derive the expected routed set

## [0.20.1](https://github.com/jhoblitt/rook-claude/compare/v0.20.0...v0.20.1) (2026-09-03)

### Bug Fixes

* **rook-conventions:** fill four canon gaps in fan-out and PR rules ([362f43d](https://github.com/jhoblitt/rook-claude/commit/362f43da0f17bf2414dd24740c97d6e8e3faa9e1)), closes [#107](https://github.com/jhoblitt/rook-claude/issues/107)


## What's Changed
* fix(rook-conventions): fill four canon gaps in fan-out and PR rules by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/109

### Resolved issues

* [#107](https://github.com/jhoblitt/rook-claude/issues/107) rook-conventions: four canon gaps the gate noted

## [0.20.0](https://github.com/jhoblitt/rook-claude/compare/v0.19.2...v0.20.0) (2026-09-03)

### Features

* **rook-conventions:** check backported workflows against the branch ([bf7e596](https://github.com/jhoblitt/rook-claude/commit/bf7e5964d2422759fbdde951484c75c87155b49a)), closes [rook/rook#18232](https://github.com/rook/rook/issues/18232) [rook/rook#18312](https://github.com/rook/rook/issues/18312) [#89](https://github.com/jhoblitt/rook-claude/issues/89)
* **rook-conventions:** derive backport set from Ceph version support ([55ca7b7](https://github.com/jhoblitt/rook-claude/commit/55ca7b78c72970f66b11788b65a3c84ef770623b)), closes [rook/rook#18242](https://github.com/rook/rook/issues/18242) [#78](https://github.com/jhoblitt/rook-claude/issues/78)
* **rook-conventions:** keep a blessed PR's backport labels current ([a4aa870](https://github.com/jhoblitt/rook-claude/commit/a4aa870617e96a62c6a77910a8878c26f152c9f9)), closes [rook/rook#18242](https://github.com/rook/rook/issues/18242) [#79](https://github.com/jhoblitt/rook-claude/issues/79)
* **rook-conventions:** verify non-owner technical claims before acting ([69a47f3](https://github.com/jhoblitt/rook-claude/commit/69a47f3c0f57e7924edf183d903ab0b50a51963e)), closes [rook/rook#18242](https://github.com/rook/rook/issues/18242) [#76](https://github.com/jhoblitt/rook-claude/issues/76)

### Bug Fixes

* **rook-conventions:** keep PR title current, and on additive pushes ([a1c06e4](https://github.com/jhoblitt/rook-claude/commit/a1c06e4f0ec2eb59fa822166f977f801eceb321f)), closes [rook/rook#18218](https://github.com/rook/rook/issues/18218) [#72](https://github.com/jhoblitt/rook-claude/issues/72)
* **rook-conventions:** route reviewer selection via rook-triage's KB ([d4cda26](https://github.com/jhoblitt/rook-claude/commit/d4cda265abd6649b61ac62cee41f7e77f7fc8f0f)), closes [rook/rook#18242](https://github.com/rook/rook/issues/18242) [#75](https://github.com/jhoblitt/rook-claude/issues/75)
* **rook-conventions:** say how a status-check name is composed ([8baac80](https://github.com/jhoblitt/rook-claude/commit/8baac80b8de30a3967a8602465af80c30b1bf286)), closes [rook/rook#18232](https://github.com/rook/rook/issues/18232) [#74](https://github.com/jhoblitt/rook-claude/issues/74)

### Refactoring

* **rook-conventions:** split label application out of backporting.md ([2ec7202](https://github.com/jhoblitt/rook-claude/commit/2ec720243f8b18f536eab5a0e1fe50f7555d0ae2))


## What's Changed
* feat(rook-conventions): close seven house-rule gaps from recent rook PRs by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/93

### Resolved issues

* [#72](https://github.com/jhoblitt/rook-claude/issues/72) rook-conventions: PR-update rule covers the description but not the title
* [#74](https://github.com/jhoblitt/rook-claude/issues/74) rook-conventions: workflows-and-ci.md never says how a status-check name is composed
* [#75](https://github.com/jhoblitt/rook-claude/issues/75) Reviewer selection must use the rook-triage routing KB, not ad-hoc CODE-OWNERS
* [#76](https://github.com/jhoblitt/rook-claude/issues/76) Treat PR-comment analysis/direction/implementation skeptically when the author isn't in CODE-OWNERS
* [#78](https://github.com/jhoblitt/rook-claude/issues/78) Backport eligibility for version-specific fixes should use a version-support × affected-range matrix, re-evaluated on range changes
* [#79](https://github.com/jhoblitt/rook-claude/issues/79) Once a PR is blessed for backporting (CODE-OWNER or existing label), keep backport labels in sync as the PR changes
* [#89](https://github.com/jhoblitt/rook-claude/issues/89) rook-conventions: backporting.md guards docs against master-only refs but not workflows

## [0.19.2](https://github.com/jhoblitt/rook-claude/compare/v0.19.1...v0.19.2) (2026-09-03)

### Bug Fixes

* **rook-maintainer:** ignore another plugin's CLAUDE_PLUGIN_DATA ([e149d12](https://github.com/jhoblitt/rook-claude/commit/e149d1273be11be5f4960dfe0cb6cca475906365)), closes [#85](https://github.com/jhoblitt/rook-claude/issues/85)
* **rook-triage:** mine origin/master in rt-commits --repo ([de49593](https://github.com/jhoblitt/rook-claude/commit/de49593b9df3cab6c45319679d1b06381664c1e6)), closes [#84](https://github.com/jhoblitt/rook-claude/issues/84)
* **rook-triage:** ship real logins in the kb snapshot ([da812bf](https://github.com/jhoblitt/rook-claude/commit/da812bf882206a6397e528a3bd0338088d5775d6)), closes [#83](https://github.com/jhoblitt/rook-claude/issues/83)
* **rook-triage:** stop excluding travisn from routine routing ([5c0c38f](https://github.com/jhoblitt/rook-claude/commit/5c0c38f3df7b381a14346f1d480359d671cf942d)), closes [rook/rook#18242](https://github.com/rook/rook/issues/18242) [#77](https://github.com/jhoblitt/rook-claude/issues/77)

### Refactoring

* **rook-triage:** split kb refresh out of routing.md ([b10a921](https://github.com/jhoblitt/rook-claude/commit/b10a92182a1af775d084d6b43a33c721d2a06b00))


## What's Changed
* fix(rook-triage): correct the routing KB and its mining tools by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/92

### Resolved issues

* [#77](https://github.com/jhoblitt/rook-claude/issues/77) routing-overrides.md '*: travisn — escalation-only' is incorrect; travisn is a routine reviewer
* [#83](https://github.com/jhoblitt/rook-claude/issues/83) kb-snapshot.json ships two display names in the maintainers login field
* [#84](https://github.com/jhoblitt/rook-claude/issues/84) rt-commits --repo mines HEAD, but routing.md specifies origin/master
* [#85](https://github.com/jhoblitt/rook-claude/issues/85) run.sh trusts an ambient CLAUDE_PLUGIN_DATA that may belong to another plugin

## [0.19.1](https://github.com/jhoblitt/rook-claude/compare/v0.19.0...v0.19.1) (2026-09-03)

### Bug Fixes

* **rook-triage:** anchor the ceph-external and block substring rules ([b142f81](https://github.com/jhoblitt/rook-claude/commit/b142f81ae1c287e4adf254d1be8e6b2a4b3f053e)), closes [#81](https://github.com/jhoblitt/rook-claude/issues/81)
* **rook-triage:** stamp nested go module manifests as build ([aa20123](https://github.com/jhoblitt/rook-claude/commit/aa201231badb087064d901299597998cd7838a2a)), closes [#82](https://github.com/jhoblitt/rook-claude/issues/82)
* **rook-triage:** stop stamping generated artifacts as helm/docs ([d702ea0](https://github.com/jhoblitt/rook-claude/commit/d702ea033fadb4dffcd45ad2169ec36031bb6cf6)), closes [#80](https://github.com/jhoblitt/rook-claude/issues/80)


## What's Changed
* fix(rook-triage): anchor rt-analyze's area classifier to its spec by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/91

### Resolved issues

* [#80](https://github.com/jhoblitt/rook-claude/issues/80) rtanalyze: generated CRD artifacts stamp helm and docs, inflating reviewer authority
* [#81](https://github.com/jhoblitt/rook-claude/issues/81) rtanalyze: strings.Contains over-stamps ceph-external (externalversions) and block (CSI rbd templates)
* [#82](https://github.com/jhoblitt/rook-claude/issues/82) rtanalyze: pkg/apis/go.mod stamps crd instead of build (exact-equality path test)

## [0.19.0](https://github.com/jhoblitt/rook-claude/compare/v0.18.0...v0.19.0) (2026-09-03)

### Features

* **rook-code-review:** hunt cross-cluster reconcile serialization ([2c5ca40](https://github.com/jhoblitt/rook-claude/commit/2c5ca40f29b349108d832c94c3518ece229d6f8e)), closes [rook/rook#18241](https://github.com/rook/rook/issues/18241) [#73](https://github.com/jhoblitt/rook-claude/issues/73)

### Bug Fixes

* **rook-code-review:** make the routed reference set a gap-sweep floor ([d64df83](https://github.com/jhoblitt/rook-claude/commit/d64df83628c265f92669c511149b33ab19a5e47c)), closes [rook/rook#18058](https://github.com/rook/rook/issues/18058) [#88](https://github.com/jhoblitt/rook-claude/issues/88)
* **rook-code-review:** re-derive findings adopted from review threads ([a1221ba](https://github.com/jhoblitt/rook-claude/commit/a1221ba2d1a3cc0ba762fc7a1749cad10f82632a)), closes [rook/rook#18058](https://github.com/rook/rook/issues/18058) [#76](https://github.com/jhoblitt/rook-claude/issues/76) [#87](https://github.com/jhoblitt/rook-claude/issues/87) [#76](https://github.com/jhoblitt/rook-claude/issues/76)
* **rook-code-review:** register the GH Actions action-download flake ([dd971ba](https://github.com/jhoblitt/rook-claude/commit/dd971bafba9aeffcdd5254f0e288f15c4db1f578)), closes [rook/rook#17952](https://github.com/rook/rook/issues/17952) [#86](https://github.com/jhoblitt/rook-claude/issues/86)


## What's Changed
* ci: pin the shellcheck the validate job runs by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/71
* fix(rook-code-review): re-derive adopted findings and floor the gap sweep by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/90

### Resolved issues

* [#73](https://github.com/jhoblitt/rook-claude/issues/73) rook-code-review: add an axis for CR-reconcile serialization (keep CephClusters parallel)
* [#86](https://github.com/jhoblitt/rook-claude/issues/86) known-flakes.md has no entry for GitHub Actions action-download failures, which fail before any repo code runs
* [#87](https://github.com/jhoblitt/rook-claude/issues/87) rook-code-review: findings adopted from existing review threads bypass verification and inherit the commenter's severity
* [#88](https://github.com/jhoblitt/rook-claude/issues/88) rook-code-review: an orchestrator's focus list can silently narrow a gap sweep, and the empty result is then counted as coverage evidence

## [0.18.0](https://github.com/jhoblitt/rook-claude/compare/v0.17.0...v0.18.0) (2026-08-18)

### Features

* **rook-maintainer:** guard the fallback agent inside a rook checkout ([5193e15](https://github.com/jhoblitt/rook-claude/commit/5193e15aa089063f3be522a2041acf1343d1c599)), closes [#55](https://github.com/jhoblitt/rook-claude/issues/55)

### Bug Fixes

* **rook-code-review:** deny the review spine its in-place writer ([4154fa7](https://github.com/jhoblitt/rook-claude/commit/4154fa71c5545d38b70d03d1e3bfe88b520db2fc)), closes [#60](https://github.com/jhoblitt/rook-claude/issues/60)
* **rook-code-review:** fence the untrusted spans the inline spine reads ([10bc0ce](https://github.com/jhoblitt/rook-claude/commit/10bc0ce0b5bc9e6bb52fa325166c1f6dfeaf0a78)), closes [#59](https://github.com/jhoblitt/rook-claude/issues/59)
* **rook-code-review:** pin authority docs on every path that reads them ([d077514](https://github.com/jhoblitt/rook-claude/commit/d07751470b4af239b824017d5fb70dca41ff3676)), closes [#56](https://github.com/jhoblitt/rook-claude/issues/56)
* **rook-code-review:** stop the inline accuracy pass fetching unguarded ([1b3aa89](https://github.com/jhoblitt/rook-claude/commit/1b3aa8959f7e1d2b8cc2e62e9420988eca3e5466)), closes [#54](https://github.com/jhoblitt/rook-claude/issues/54)
* **rook-maintainer:** cut a truncated string on a rune boundary ([0f78891](https://github.com/jhoblitt/rook-claude/commit/0f78891e6bc20debf70e03a3d032c8d119785545))
* **rook-maintainer:** escape the path validate-anchors prints ([a3d5813](https://github.com/jhoblitt/rook-claude/commit/a3d58139138247246ea4784d2c0b92694ed00d6f)), closes [#58](https://github.com/jhoblitt/rook-claude/issues/58)
* **rook-maintainer:** fence the ref names the rebase notice injects ([9ef79ea](https://github.com/jhoblitt/rook-claude/commit/9ef79ea94429aba1e748ce53df088c41150939ab)), closes [#57](https://github.com/jhoblitt/rook-claude/issues/57)
* **rook-maintainer:** point the deny message at a tool that exists ([41f789b](https://github.com/jhoblitt/rook-claude/commit/41f789b8eafd92acfa7c3fb44da3ffc0c1dc2cba))
* **rook-maintainer:** surface a fetch the guard did not adjudicate ([7aed828](https://github.com/jhoblitt/rook-claude/commit/7aed828f5f8506f5e1dbc8b1545c57fe8e882034)), closes [#54](https://github.com/jhoblitt/rook-claude/issues/54)
* **rook-maintainer:** verify the cached tool binary before exec ([49db402](https://github.com/jhoblitt/rook-claude/commit/49db402d5f92220ec7ee5e80c1a9e89c72978b8a)), closes [#53](https://github.com/jhoblitt/rook-claude/issues/53)
* **rook-maintainer:** warn when the default branch is behind ([1b43e8d](https://github.com/jhoblitt/rook-claude/commit/1b43e8dfc3ba9a451f53c9501c59e36ab865a697))

### Refactoring

* **rook-maintainer:** give PR takeover its own skill ([47c379d](https://github.com/jhoblitt/rook-claude/commit/47c379dbdd92c1bab2ffa87dade015f4797b7554)), closes [#60](https://github.com/jhoblitt/rook-claude/issues/60)

### Documentation

* **rook-code-review:** describe the fallback guard's checkout scope ([ec9e014](https://github.com/jhoblitt/rook-claude/commit/ec9e014ef9b97b908596125daebdbff707114276)), closes [#55](https://github.com/jhoblitt/rook-claude/issues/55)
* **rook-code-review:** name the apiserver audit log as an argv channel ([03467ba](https://github.com/jhoblitt/rook-claude/commit/03467ba1be251737d16947519cc8945ab1a1b6c5)), closes [#69](https://github.com/jhoblitt/rook-claude/issues/69)
* **rook-maintainer:** diagram every procedure skill and keep it current ([17fb345](https://github.com/jhoblitt/rook-claude/commit/17fb34560b56d418da3836f33b6deed1ffe22a19))


## What's Changed
* fix(rook-maintainer): close the open audit findings by @jhoblitt in https://github.com/jhoblitt/rook-claude/pull/70

### Resolved issues

* [#53](https://github.com/jhoblitt/rook-claude/issues/53) rook-code-review: tools/run.sh execs a cached binary with no integrity check
* [#54](https://github.com/jhoblitt/rook-claude/issues/54) rook-code-review: WebFetch guard does not bind on the inline path, and fails open silently
* [#55](https://github.com/jhoblitt/rook-claude/issues/55) rook-code-review: the general-purpose fallback escapes both the agent roster and the WebFetch guard
* [#56](https://github.com/jhoblitt/rook-claude/issues/56) rook-code-review: authority-order documents are read from the contributor's checkout
* [#57](https://github.com/jhoblitt/rook-claude/issues/57) rook-maintainer: rebase-notice.sh interpolates ref names raw into every prompt
* [#58](https://github.com/jhoblitt/rook-claude/issues/58) validate-anchors: path is interpolated unescaped, bypassing the package's own quote()
* [#59](https://github.com/jhoblitt/rook-claude/issues/59) rook-code-review: the treat-as-data rule ships without a delimiter
* [#60](https://github.com/jhoblitt/rook-claude/issues/60) rook-code-review: the skill declares no allowed-tools and inherits the session roster
* [#69](https://github.com/jhoblitt/rook-claude/issues/69) security.md: argv bullet omits the apiserver audit-log channel and can bless a literal env value

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

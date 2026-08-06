The added `object.addLabelToPod` is a line-for-line re-implementation of
`k8sutil.AddLabelToPod`, which the diff's own file already imports. The
names differ only by exported-case, so the pass's name-normalized query
reaches it — this case tests what pass j claims to cover.

Pass if and only if ALL of:

1. A finding tagged `duplication` reports the added function as a
   re-implementation, at changes-requested severity.
2. The finding names the existing implementation as
   `pkg/operator/k8sutil/labels.go:AddLabelToPod` — full repo-relative
   path plus symbol.
3. The finding names the bypassed mechanism (the exported helper in
   `pkg/operator/k8sutil/`) rather than only asserting the two are alike.
4. The failure scenario is divergence — a future change that must land in
   both copies and will land in one.
5. The audited-and-clean statement scopes its reuse claim to
   name-reachable reinvention, rather than asserting the diff duplicates
   nothing.

Fail if any of:

- No `duplication` finding is reported, or it is reported as a nit.
- The existing implementation is anchored by bare basename (`labels.go`)
  or an elided path.
- The finding argues only textual similarity, naming no mechanism.
- The report claims the diff is free of duplication without qualifying the
  claim to what the queries can reach.
- Subagents were spawned despite the stated no-subagent environment.

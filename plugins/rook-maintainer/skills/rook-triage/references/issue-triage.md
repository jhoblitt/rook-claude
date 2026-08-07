# Issue triage

Kinds: bug / feature / support / docs / meta (CI, release, process). The
bug template auto-applies `bug`; body text is authoritative over the
template's claim — a "bug report" asking "how do I…" is support.

## Completeness rubric (bugs)

The bug template IS the rubric. A bug is incomplete when it lacks, as
relevant to its claim: deviation vs expected behavior · minimal repro ·
cluster CR (`cluster.yaml`) · operator / crashing-pod logs · environment
block (rook version, ceph version, k8s version, OS/kernel). needs-info
comments name the EXACT missing fields — never a generic "please provide
more information". Incomplete bugs get no category/severity judgment;
completeness comes first.

## Dispositions (each with the action that clears it)

| Disposition | Action | Cleared by |
|---|---|---|
| confirmed-bug | area + `bug` labels (severity note in report) | maintainer pickup |
| needs-info | comment (template below) + `needs-info` label if it exists | substantive author reply |
| needs-info-expired | propose close (14–20d since request, computed per run — no cron) | approval |
| feature | `feature` label — check `out-of-scope.md` FIRST | maintainer decision |
| support | redirect comment + queue convert-to-Discussion | conversion |
| duplicate | link comment ("possibly duplicate of #X: reason") | refuted-or-approved close |
| possible-fix-open | comment linking PR #N | PR lands |
| fixed-by-merged | verify the merged diff addresses the mechanism; propose close | approval |
| escalate | flag to the user (security / data-loss / regression) | user decision |
| triaged-keep-open | labels only (+ `triage-accepted` if it exists) | — |

Never: staleness actions (stale bot owns them; `keepalive` = hands off) ·
answering the technical question · closing anything without phase-2
refutation + approval · dup-closing on similarity alone (same ROOT CAUSE
required, not same symptom).

## Comment templates

With `<login>` the operating maintainer's GitHub login (`gh api user --jq
.login`), every comment opens with `> This is @<login>'s AI agent.`
(rook-conventions "Signing GitHub comments" governs the marker and what
else may accompany it).

**needs-info:**
> Thanks for the report. To make this actionable we still need:
> <bullet list of the exact missing template fields>
> With only what's here we can't reproduce or route it. If we don't hear
> back in ~2 weeks this may be closed — a reply reopens the conversation.

**support-redirect:**
> This tracker is for bug reports and feature work, and this reads as a
> usage/configuration question. The best places for help are the Rook
> Slack (<https://slack.rook.io/>) and GitHub Discussions. Proposing to
> convert this issue to a Discussion so the thread stays findable.

(The conversion itself is queued for execution: GraphQL
`convertIssueToDiscussion` or the one-click UI step — REST has no clean
path. It is a close-class action: refutation + approval first.)

**dup-suggest:**
> This looks like the same root cause as #X (<one-line reason>). If so,
> let's continue there — please correct us if these are distinct.

**good-first-issue** candidates: only when the fix location AND approach
are already stated in-thread (self-serve bar); the comment may note that
`/assign` self-assigns (the repo's auto-assign workflow).

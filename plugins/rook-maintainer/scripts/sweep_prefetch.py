#!/usr/bin/env python3
"""Phase-0 metadata snapshot for a rook-triage or rook-code-review sweep.

Both call it — rook-triage's pipeline phase 0 and rook-code-review's
sweep phase 0 — so a change here must keep both callers working.

One batched GraphQL pass per corpus, written to <sweep-dir>/snapshot.json.
Every triager and reviewer agent and every dashboard generator consumes
this snapshot instead of fetching per-item metadata themselves — one
fetch, one consistent point-in-time view, ~100 fewer per-agent gh calls
per sweep.

PR items carry `files` (changed paths), which is what lets a sweep
orchestrator route reference files per PR without a per-PR fetch;
`gh pr list` cannot return them at any --json setting.

PR items carry a summarized statusCheckRollup ({passing, failing,
pending, total, failed[]}) — the ONLY source dashboards may use for CI
cells (deterministic passing/total; never parsed from agent prose).

Modes:
  snapshot      enumerate all OPEN items of --kind, or exactly --numbers
                (numbers mode also fetches closed/merged items, for
                regenerating dashboards of past sweeps)
  classify-refs scan <sweep-dir>/batch-*.json xlink/dup numbers and write
                refs-types.json (Issue vs PullRequest via
                issueOrPullRequest — never guessed)

Usage:
  python3 sweep_prefetch.py snapshot SWEEP_DIR --kind prs|issues
          [--numbers 1,2,3 | --numbers-file F] [--repo rook/rook]
  python3 sweep_prefetch.py classify-refs SWEEP_DIR [--repo rook/rook]

Needs authenticated gh (run with the sandbox disabled).
"""
import argparse
import datetime
import glob
import json
import os
import subprocess

PR_FIELDS = """
number title state isDraft updatedAt createdAt
author { login } authorAssociation baseRefName mergeable reviewDecision
additions deletions changedFiles
labels(first: 20) { nodes { name } }
assignees(first: 10) { nodes { login } }
files(first: 100) { pageInfo { hasNextPage endCursor } nodes { path } }
latestReviews(first: 20) { nodes { author { login } state } }
reviewRequests(first: 20) { nodes { requestedReviewer {
  ... on User { login } ... on Team { name } } } }
commits(last: 1) { nodes { commit { statusCheckRollup {
  state contexts(first: 100) { pageInfo { hasNextPage endCursor } nodes { __CTX__ } } } } } }
"""

CTX_NODE = """
__typename
... on CheckRun { name status conclusion
  checkSuite { databaseId createdAt workflowRun { workflow { name } } } }
... on StatusContext { context state }
"""
PR_FIELDS = PR_FIELDS.replace("__CTX__", CTX_NODE)

ISSUE_FIELDS = """
number title state updatedAt createdAt
author { login }
labels(first: 20) { nodes { name } }
assignees(first: 10) { nodes { login } }
comments { totalCount }
"""

PASS_CONCLUSIONS = {"SUCCESS", "NEUTRAL", "SKIPPED"}
FAIL_CONCLUSIONS = {"FAILURE", "TIMED_OUT", "CANCELLED", "ACTION_REQUIRED", "STARTUP_FAILURE"}


def gql(query):
    proc = subprocess.run(["gh", "api", "graphql", "-f", f"query={query}"],
                          capture_output=True, text=True, timeout=180)
    if proc.returncode != 0:
        raise RuntimeError(f"gh api graphql rc={proc.returncode} stderr={proc.stderr[:3000]}")
    data = json.loads(proc.stdout)
    if data.get("errors"):
        raise RuntimeError(f"GraphQL errors: {json.dumps(data['errors'])[:2000]}")
    return data["data"]


def suite_key(c):
    suite = c.get("checkSuite") or {}
    wf = ((suite.get("workflowRun") or {}).get("workflow") or {}).get("name")
    return wf or f"suite:{suite.get('databaseId')}", suite.get("createdAt") or ""


def classify_ctx_nodes(state, nodes, truncated):
    """Summarize context nodes the way GitHub's merge box does. The
    rollup keeps every CheckRun from every check suite ever run on the
    commit — a CI re-run months later under a changed workflow/matrix
    leaves the old generation's (differently-named) runs in the list,
    silently doubling job counts. So: keep only the NEWEST check suite
    per workflow, then dedupe by check name (re-run attempts within a
    suite), keeping the last occurrence."""
    newest = {}
    for c in nodes:
        if c["__typename"] != "CheckRun":
            continue
        key, created = suite_key(c)
        if key not in newest or created > newest[key]:
            newest[key] = created
    latest = {}
    for c in nodes:
        if c["__typename"] == "CheckRun":
            key, created = suite_key(c)
            if created != newest.get(key):
                continue
            latest[("run", c.get("name") or "?")] = c
        else:
            latest[("ctx", c.get("context") or "?")] = c
    passing = failing = pending = 0
    failed = []
    for (kind, name), c in latest.items():
        if kind == "run":
            if c.get("status") != "COMPLETED":
                pending += 1
            elif c.get("conclusion") in PASS_CONCLUSIONS:
                passing += 1
            elif c.get("conclusion") in FAIL_CONCLUSIONS:
                failing += 1
                failed.append(name)
            else:
                pending += 1
        else:
            st = c.get("state")
            if st == "SUCCESS":
                passing += 1
            elif st in ("FAILURE", "ERROR"):
                failing += 1
                failed.append(name)
            else:
                pending += 1
    return {"state": state, "passing": passing, "failing": failing,
            "pending": pending, "total": passing + failing + pending,
            "failed": failed, "truncated": truncated}


def summarize_ci(node):
    commits = node.get("commits", {}).get("nodes") or []
    rollup = (commits[0].get("commit") or {}).get("statusCheckRollup") if commits else None
    if not rollup:
        return {"state": None, "passing": 0, "failing": 0, "pending": 0,
                "total": 0, "failed": [], "truncated": False}
    ctx = rollup.get("contexts") or {}
    return classify_ctx_nodes(rollup.get("state"), ctx.get("nodes") or [],
                              bool((ctx.get("pageInfo") or {}).get("hasNextPage")))


def shape_pr(n):
    return {
        "number": n["number"], "title": n["title"], "state": n["state"],
        "isDraft": n["isDraft"], "updatedAt": n["updatedAt"], "createdAt": n["createdAt"],
        "author": (n.get("author") or {}).get("login", ""),
        "authorAssociation": n.get("authorAssociation"),
        "baseRefName": n.get("baseRefName"), "mergeable": n.get("mergeable"),
        "reviewDecision": n.get("reviewDecision"),
        "additions": n.get("additions"), "deletions": n.get("deletions"),
        "changedFiles": n.get("changedFiles"),
        "labels": [l["name"] for l in n["labels"]["nodes"]],
        "assignees": [a["login"] for a in n["assignees"]["nodes"]],
        "files": [f["path"] for f in n["files"]["nodes"]],
        "files_truncated": n["files"]["pageInfo"]["hasNextPage"],
        "reviews": {
            "latest": [{"login": (r.get("author") or {}).get("login", ""), "state": r["state"]}
                       for r in n["latestReviews"]["nodes"] if r.get("author")],
            "requested": [rr["requestedReviewer"].get("login") or rr["requestedReviewer"].get("name")
                          for rr in n["reviewRequests"]["nodes"] if rr.get("requestedReviewer")],
        },
        "ci": summarize_ci(n),
    }


def shape_issue(n):
    return {
        "number": n["number"], "title": n["title"], "state": n["state"],
        "updatedAt": n["updatedAt"], "createdAt": n["createdAt"],
        "author": (n.get("author") or {}).get("login", ""),
        "labels": [l["name"] for l in n["labels"]["nodes"]],
        "assignees": [a["login"] for a in n["assignees"]["nodes"]],
        "comments_total": n["comments"]["totalCount"],
    }


def fetch_open(owner, name, kind):
    field = "pullRequests" if kind == "prs" else "issues"
    inner = PR_FIELDS if kind == "prs" else ISSUE_FIELDS
    page = 25 if kind == "prs" else 100
    nodes, cursor = [], None
    while True:
        after = f', after: "{cursor}"' if cursor else ""
        q = (f'query {{ repository(owner: "{owner}", name: "{name}") {{ '
             f'{field}(states: OPEN, first: {page}{after}, '
             f'orderBy: {{field: CREATED_AT, direction: ASC}}) {{ '
             f'pageInfo {{ hasNextPage endCursor }} nodes {{ {inner} }} }} }} }}')
        block = gql(q)["repository"][field]
        nodes += block["nodes"]
        if not block["pageInfo"]["hasNextPage"]:
            return nodes
        cursor = block["pageInfo"]["endCursor"]


def fetch_numbers(owner, name, kind, numbers):
    field = "pullRequest" if kind == "prs" else "issue"
    inner = PR_FIELDS if kind == "prs" else ISSUE_FIELDS
    chunk = 15 if kind == "prs" else 75
    nodes = []
    for i in range(0, len(numbers), chunk):
        aliases = " ".join(f"n{x}: {field}(number: {x}) {{ {inner} }}"
                           for x in numbers[i:i + chunk])
        repo = gql(f'query {{ repository(owner: "{owner}", name: "{name}") {{ {aliases} }} }}')["repository"]
        nodes += [v for v in repo.values() if v]
    return nodes


def paginate_all_contexts(owner, name, number):
    nodes, cursor = [], None
    while True:
        after = f', after: "{cursor}"' if cursor else ""
        q = (f'query {{ repository(owner: "{owner}", name: "{name}") {{ '
             f'pullRequest(number: {number}) {{ commits(last: 1) {{ nodes {{ commit {{ '
             f'statusCheckRollup {{ state contexts(first: 100{after}) {{ '
             f'pageInfo {{ hasNextPage endCursor }} nodes {{ {CTX_NODE} }} }} }} }} }} }} }} }} }}')
        rollup = gql(q)["repository"]["pullRequest"]["commits"]["nodes"][0]["commit"]["statusCheckRollup"]
        block = rollup["contexts"]
        nodes += block["nodes"]
        if not block["pageInfo"]["hasNextPage"]:
            return rollup.get("state"), nodes
        cursor = block["pageInfo"]["endCursor"]


def paginate_all_files(owner, name, number):
    paths, cursor = [], None
    while True:
        after = f', after: "{cursor}"' if cursor else ""
        q = (f'query {{ repository(owner: "{owner}", name: "{name}") {{ '
             f'pullRequest(number: {number}) {{ files(first: 100{after}) {{ '
             f'pageInfo {{ hasNextPage endCursor }} nodes {{ path }} }} }} }} }}')
        block = gql(q)["repository"]["pullRequest"]["files"]
        paths += [f["path"] for f in block["nodes"]]
        if not block["pageInfo"]["hasNextPage"]:
            return paths
        cursor = block["pageInfo"]["endCursor"]


def cmd_snapshot(args):
    owner, name = args.repo.split("/", 1)
    if args.numbers or args.numbers_file:
        if args.numbers:
            nums = [int(x) for x in args.numbers.split(",") if x.strip()]
        else:
            nums = [int(l) for l in open(args.numbers_file) if l.strip()]
        nodes = fetch_numbers(owner, name, args.kind, sorted(set(nums)))
    else:
        nodes = fetch_open(owner, name, args.kind)
    shape = shape_pr if args.kind == "prs" else shape_issue
    items = {str(n["number"]): shape(n) for n in nodes}
    if args.kind == "prs":
        for num, item in items.items():
            if item["ci"].get("truncated"):
                state, ctx_nodes = paginate_all_contexts(owner, name, int(num))
                item["ci"] = classify_ctx_nodes(state, ctx_nodes, False)
            if item.get("files_truncated"):
                item["files"] = paginate_all_files(owner, name, int(num))
                item["files_truncated"] = False
    out = {
        "fetched_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
        "repo": args.repo, "kind": args.kind, "items": items,
    }
    os.makedirs(args.sweep_dir, exist_ok=True)
    path = os.path.join(args.sweep_dir, "snapshot.json")
    json.dump(out, open(path, "w"), indent=1)
    trunc = [k for k, v in items.items()
             if v.get("files_truncated") or (v.get("ci") or {}).get("truncated")]
    print(f"{len(items)} {args.kind} -> {path}" + (f" (truncated fields on: {trunc})" if trunc else ""))


def cmd_classify_refs(args):
    owner, name = args.repo.split("/", 1)
    nums = set()
    for b in sorted(glob.glob(os.path.join(args.sweep_dir, "batch-*.json"))):
        for it in json.load(open(b)):
            for x in (it.get("xlinks") or []) + (it.get("dups") or []):
                if isinstance(x, dict) and "number" in x:
                    nums.add(int(x["number"]))
    path = os.path.join(args.sweep_dir, "refs-types.json")
    types = json.load(open(path)) if os.path.exists(path) else {}
    todo = sorted(n for n in nums if str(n) not in types)
    for i in range(0, len(todo), 75):
        aliases = " ".join(f"n{x}: issueOrPullRequest(number: {x}) {{ __typename }}"
                           for x in todo[i:i + 75])
        repo = gql(f'query {{ repository(owner: "{owner}", name: "{name}") {{ {aliases} }} }}')["repository"]
        for k, v in repo.items():
            if v:
                types[k[1:]] = v["__typename"]
    json.dump(types, open(path, "w"), indent=1)
    print(f"{len(nums)} refs, {len(todo)} newly classified -> {path}")


def main():
    ap = argparse.ArgumentParser()
    sub = ap.add_subparsers(dest="cmd", required=True)
    s = sub.add_parser("snapshot")
    s.add_argument("sweep_dir")
    s.add_argument("--kind", choices=["prs", "issues"], required=True)
    s.add_argument("--numbers")
    s.add_argument("--numbers-file")
    s.add_argument("--repo", default="rook/rook")
    s.set_defaults(func=cmd_snapshot)
    c = sub.add_parser("classify-refs")
    c.add_argument("sweep_dir")
    c.add_argument("--repo", default="rook/rook")
    c.set_defaults(func=cmd_classify_refs)
    args = ap.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()

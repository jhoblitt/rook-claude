#!/usr/bin/env python3
"""Tier-0 KB mining fetch: merged PRs with files + reviews, via GraphQL.

Deterministic fetch layer for the rook-triage kb refresh
(references/routing.md). Walks repository.pullRequests (states: MERGED,
UPDATED_AT DESC) with one cursor — this endpoint has no search-API
1000-result cap, so a single walk covers any window; the search-API
slice sharding in routing.md is only for parallel miners.

Counts a PR when mergedAt >= cutoff. Stops at --cap counted PRs, or when
a full page is entirely updatedAt-older than the cutoff (updatedAt >=
mergedAt always, and pages are updatedAt-ordered, so no in-window PR can
follow such a page; a mergedAt-based stop is NOT safe — a mass comment
sweep over old PRs floats them to the top). Flags per-PR truncation
(files > 100, reviews > 30) instead of silently dropping — the miner
turns those into `truncation` flags per the two-tier contract.

Writes <out-dir>/rt_prs.jsonl (one counted PR per line) and
<out-dir>/rt_fetch_state.json (bounds, stop reason, errors, truncations
— the assembler's provenance input).

--deep-fetch follows up each truncation-flagged PR with per-PR pagination
of its remaining files/reviews pages, patches the JSONL records, and
moves those entries from `truncations` to `deep_fetched` in the state.
--deep-fetch-only skips the walk and deep-fetches an EXISTING out-dir.

Usage: python3 rt_fetch.py --out-dir DIR [--months 24] [--cap 4000]
                           [--repo rook/rook] [--page-size 50]
                           [--deep-fetch | --deep-fetch-only]
"""
import argparse
import datetime
import json
import os
import subprocess
import sys
import time

QUERY_TEMPLATE = """
query {{
  repository(owner: "{owner}", name: "{name}") {{
    pullRequests(states: MERGED, orderBy: {{field: UPDATED_AT, direction: DESC}}, first: {page_size}{after_clause}) {{
      pageInfo {{ hasNextPage endCursor }}
      nodes {{
        number
        title
        mergedAt
        updatedAt
        author {{ login }}
        files(first: 100) {{
          pageInfo {{ hasNextPage }}
          nodes {{ path }}
        }}
        reviews(first: 30) {{
          pageInfo {{ hasNextPage }}
          nodes {{ author {{ login }} state }}
        }}
      }}
    }}
  }}
  rateLimit {{ cost remaining resetAt }}
}}
"""


def parse_dt(s):
    return datetime.datetime.fromisoformat(s.replace("Z", "+00:00"))


def run_query(args, cursor):
    owner, name = args.repo.split("/", 1)
    query = QUERY_TEMPLATE.format(
        owner=owner, name=name, page_size=args.page_size,
        after_clause=f', after: "{cursor}"' if cursor else "")
    proc = subprocess.run(["gh", "api", "graphql", "-f", f"query={query}"],
                          capture_output=True, text=True, timeout=120)
    if proc.returncode != 0:
        raise RuntimeError(f"gh api graphql rc={proc.returncode} stderr={proc.stderr[:3000]}")
    return json.loads(proc.stdout)


def deep_fetch_field(repo, number, field, inner, page):
    owner, name = repo.split("/", 1)
    nodes, cursor = [], None
    while True:
        after = f', after: "{cursor}"' if cursor else ""
        q = (f'query {{ repository(owner: "{owner}", name: "{name}") {{ '
             f'pullRequest(number: {number}) {{ {field}(first: {page}{after}) {{ '
             f'pageInfo {{ hasNextPage endCursor }} nodes {{ {inner} }} }} }} }} }}')
        proc = subprocess.run(["gh", "api", "graphql", "-f", f"query={q}"],
                              capture_output=True, text=True, timeout=120)
        if proc.returncode != 0:
            raise RuntimeError(f"deep-fetch PR {number} {field}: {proc.stderr[:2000]}")
        block = json.loads(proc.stdout)["data"]["repository"]["pullRequest"][field]
        nodes += block["nodes"]
        if not block["pageInfo"]["hasNextPage"]:
            return nodes
        cursor = block["pageInfo"]["endCursor"]


def deep_fetch(args):
    out_jsonl = os.path.join(args.out_dir, "rt_prs.jsonl")
    state_json = os.path.join(args.out_dir, "rt_fetch_state.json")
    state = json.load(open(state_json))
    by_number = {}
    for line in open(out_jsonl):
        line = line.strip()
        if line:
            pr = json.loads(line)
            by_number[pr["number"]] = pr

    remaining, done = [], state.get("deep_fetched", [])
    for t in state.get("truncations", []):
        num = t["number"]
        if num not in by_number:
            remaining.append(t)
            continue
        if t["kind"] == "files":
            nodes = deep_fetch_field(args.repo, num, "files", "path", 100)
            by_number[num]["files"] = {"pageInfo": {"hasNextPage": False}, "nodes": nodes}
        else:
            nodes = deep_fetch_field(args.repo, num, "reviews", "author { login } state", 100)
            by_number[num]["reviews"] = {"pageInfo": {"hasNextPage": False}, "nodes": nodes}
        done.append(t)
        print(f"deep-fetched PR #{num} {t['kind']}: {len(nodes)} total", file=sys.stderr, flush=True)

    with open(out_jsonl, "w") as outf:
        for pr in by_number.values():
            outf.write(json.dumps(pr) + "\n")
    state["truncations"] = remaining
    state["deep_fetched"] = done
    json.dump(state, open(state_json, "w"), indent=2)
    print(f"deep-fetch: {len(done)} resolved, {len(remaining)} out-of-set left as flags",
          file=sys.stderr, flush=True)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out-dir", required=True)
    ap.add_argument("--months", type=float, default=24)
    ap.add_argument("--cap", type=int, default=4000)
    ap.add_argument("--repo", default="rook/rook")
    ap.add_argument("--page-size", type=int, default=50)
    ap.add_argument("--max-pages", type=int, default=400)
    ap.add_argument("--deep-fetch", action="store_true")
    ap.add_argument("--deep-fetch-only", action="store_true")
    args = ap.parse_args()

    if args.deep_fetch_only:
        deep_fetch(args)
        return

    cutoff = (datetime.datetime.now(datetime.timezone.utc)
              - datetime.timedelta(days=round(args.months * 30.44)))
    os.makedirs(args.out_dir, exist_ok=True)
    out_jsonl = os.path.join(args.out_dir, "rt_prs.jsonl")
    state_json = os.path.join(args.out_dir, "rt_fetch_state.json")

    seen_numbers = set()
    counted = 0
    oldest_mergedat = None
    cursor = None
    page_num = 0
    errors = []
    truncations = []
    stop_reason = None

    with open(out_jsonl, "w") as outf:
        while True:
            page_num += 1
            if page_num > args.max_pages:
                stop_reason = f"safety max-pages={args.max_pages} reached"
                errors.append(stop_reason)
                break

            data = None
            last_err = None
            for attempt in range(6):
                try:
                    data = run_query(args, cursor)
                    if "errors" in data:
                        raise RuntimeError(f"GraphQL errors: {json.dumps(data['errors'])[:2000]}")
                    break
                except Exception as e:
                    last_err = str(e)
                    time.sleep(3 * (attempt + 1))
            if data is None:
                errors.append(f"page {page_num}: giving up after retries: {last_err}")
                stop_reason = "repeated fetch errors"
                break

            pr_block = data["data"]["repository"]["pullRequests"]
            nodes = pr_block["nodes"]
            page_info = pr_block["pageInfo"]
            ratelimit = data["data"].get("rateLimit", {}) or {}

            page_all_stale = True
            page_new_count = 0
            for n in nodes:
                num = n["number"]
                merged_dt = parse_dt(n["mergedAt"])
                if parse_dt(n["updatedAt"]) >= cutoff:
                    page_all_stale = False

                if num in seen_numbers:
                    continue
                seen_numbers.add(num)

                if n["files"]["pageInfo"]["hasNextPage"]:
                    truncations.append({"number": num, "kind": "files", "mergedAt": n["mergedAt"]})
                if n["reviews"]["pageInfo"]["hasNextPage"]:
                    truncations.append({"number": num, "kind": "reviews", "mergedAt": n["mergedAt"]})

                if merged_dt < cutoff:
                    continue

                counted += 1
                page_new_count += 1
                if oldest_mergedat is None or merged_dt < oldest_mergedat:
                    oldest_mergedat = merged_dt
                outf.write(json.dumps(n) + "\n")

                if counted >= args.cap:
                    break

            outf.flush()
            remaining = ratelimit.get("remaining")
            print(f"page {page_num}: got={len(nodes)} new_counted={page_new_count} "
                  f"total_counted={counted} seen_total={len(seen_numbers)} "
                  f"oldest_so_far={oldest_mergedat} cost={ratelimit.get('cost')} "
                  f"remaining={remaining} page_all_stale={page_all_stale} "
                  f"hasNextPage={page_info['hasNextPage']}",
                  file=sys.stderr, flush=True)

            if remaining is not None and remaining < 200:
                reset_at = ratelimit.get("resetAt")
                print(f"rate limit low ({remaining}), sleeping until {reset_at}",
                      file=sys.stderr, flush=True)
                if reset_at:
                    sleep_s = (parse_dt(reset_at)
                               - datetime.datetime.now(datetime.timezone.utc)).total_seconds()
                    time.sleep(min(max(0, sleep_s) + 10, 3600))

            if counted >= args.cap:
                stop_reason = f"reached --cap={args.cap}"
                break
            if page_all_stale and len(nodes) == args.page_size:
                stop_reason = "full page entirely updatedAt-older than cutoff"
                break
            if not page_info["hasNextPage"]:
                stop_reason = "no more pages (end of merged PR history)"
                break
            cursor = page_info["endCursor"]

    state = {
        "repo": args.repo,
        "pages_fetched": page_num,
        "counted": counted,
        "seen_total": len(seen_numbers),
        "oldest_mergedat": oldest_mergedat.isoformat() if oldest_mergedat else None,
        "cutoff": cutoff.isoformat(),
        "cap": args.cap,
        "stop_reason": stop_reason,
        "errors": errors,
        "truncations": truncations,
    }
    with open(state_json, "w") as fh:
        json.dump(state, fh, indent=2)
    print("DONE:", json.dumps(state)[:2000], file=sys.stderr, flush=True)
    if args.deep_fetch and truncations:
        deep_fetch(args)


if __name__ == "__main__":
    main()

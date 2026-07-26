#!/usr/bin/env python3
"""Mine issue-thread @-mentions for the report's mentions column.

Implements the three-layer contract in references/reporting.md — a bare
`@\\w+` scan turns shell prompts (`root@pod`), emails, and broker hosts
into bogus profile links:

1. Code stripping scoped per comment-document (fence state machine; an
   unclosed ``` in one comment must not leak into the next — GitHub
   renders each comment independently), plus inline backticks.
2. GitHub mention syntax: `@` not preceded by a word char / `.` / `-` /
   `/` / `@`; login = alnum+hyphens, <=39 chars, starting alnum.
3. Live user resolution via `gh api users/<token>` — exact token first,
   then trailing-hyphen-stripped (legacy logins like `sinner-` exist);
   the API's canonical login is kept, unresolvable tokens are dropped.
   Resolutions cache in ~/.cache/rook-triage/mentions-user-check.json.

Fetches <sweep-dir>/threads.json (body + ALL comments per issue: first
100 in the chunked query, per-issue pagination beyond) when missing,
then writes <sweep-dir>/issues-mentions.json
({"<number>": [logins...]}, first-mention order) and prints a diff against
any previous version. Needs authenticated gh (run with the sandbox
disabled).

Usage: python3 mine_mentions.py SWEEP_DIR [--numbers 1,2,3 | --numbers-file F]
                                [--repo rook/rook] [--refetch]
"""
import argparse
import json
import os
import re
import subprocess

CACHE_PATH = os.path.expanduser("~/.cache/rook-triage/mentions-user-check.json")
CHUNK = 75

FENCELINE = re.compile(r"^[>\s]{0,4}(```|~~~)")
INLINE = re.compile(r"`[^`\n]*`")
MENTION = re.compile(r"(?<![\w.\-/@])@([A-Za-z0-9][A-Za-z0-9-]{0,38})(?!\w)")


def strip_code(doc):
    out, infence = [], False
    for line in doc.splitlines():
        if FENCELINE.match(line):
            infence = not infence
            continue
        if not infence:
            out.append(line)
    return INLINE.sub(" ", "\n".join(out))


def gql(query):
    proc = subprocess.run(["gh", "api", "graphql", "-f", f"query={query}",
                           "--jq", ".data.repository"],
                          capture_output=True, text=True, timeout=180)
    if proc.returncode != 0:
        raise RuntimeError(f"gh api graphql rc={proc.returncode} stderr={proc.stderr[:3000]}")
    return json.loads(proc.stdout)


def fetch_threads(repo, numbers, path):
    owner, name = repo.split("/", 1)
    merged = {}
    for i in range(0, len(numbers), CHUNK):
        chunk = numbers[i:i + CHUNK]
        fields = " ".join(
            f"i{n}: issue(number: {n}) {{ number body comments(first: 100) "
            f"{{ totalCount pageInfo {{ hasNextPage endCursor }} nodes {{ body }} }} }}"
            for n in chunk)
        merged.update(gql(f'query {{ repository(owner: "{owner}", name: "{name}") {{ {fields} }} }}'))
    for node in merged.values():
        c = node["comments"]
        cursor = c.get("pageInfo", {}).get("endCursor")
        while c.get("pageInfo", {}).get("hasNextPage"):
            page = gql(
                f'query {{ repository(owner: "{owner}", name: "{name}") {{ '
                f'issue(number: {node["number"]}) {{ comments(first: 100, after: "{cursor}") '
                f'{{ pageInfo {{ hasNextPage endCursor }} nodes {{ body }} }} }} }} }}'
            )["issue"]["comments"]
            c["nodes"] += page["nodes"]
            c["pageInfo"] = page["pageInfo"]
            cursor = page["pageInfo"].get("endCursor")
    json.dump(merged, open(path, "w"))
    return merged


def resolve(tok, cache):
    key = tok.lower()
    if key in cache:
        return cache[key]
    login = None
    for cand in dict.fromkeys([tok, tok.rstrip("-")]):
        if not cand:
            continue
        r = subprocess.run(["gh", "api", f"users/{cand}", "--jq", ".login"],
                           capture_output=True, text=True)
        if r.returncode == 0:
            login = r.stdout.strip()
            break
    cache[key] = login
    return login


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("sweep_dir")
    ap.add_argument("--numbers", help="comma-separated issue numbers (for fetch)")
    ap.add_argument("--numbers-file", help="file with one issue number per line")
    ap.add_argument("--repo", default="rook/rook")
    ap.add_argument("--refetch", action="store_true")
    args = ap.parse_args()

    threads_path = os.path.join(args.sweep_dir, "threads.json")
    out_path = os.path.join(args.sweep_dir, "issues-mentions.json")

    if args.refetch or not os.path.exists(threads_path):
        if args.numbers:
            numbers = [int(x) for x in args.numbers.split(",") if x.strip()]
        elif args.numbers_file:
            numbers = [int(l) for l in open(args.numbers_file) if l.strip()]
        else:
            raise SystemExit("threads.json missing: pass --numbers or --numbers-file to fetch")
        raw = fetch_threads(args.repo, sorted(set(numbers)), threads_path)
    else:
        raw = json.load(open(threads_path))

    candidates = {}
    for node in raw.values():
        n = str(node["number"])
        docs = [node.get("body") or ""] + [c.get("body") or "" for c in node["comments"]["nodes"]]
        text = "\n".join(strip_code(d) for d in docs)
        seen, ordered = set(), []
        for m in MENTION.finditer(text):
            tok = m.group(1)
            if tok.lower() not in seen:
                seen.add(tok.lower())
                ordered.append(tok)
        if ordered:
            candidates[n] = ordered

    os.makedirs(os.path.dirname(CACHE_PATH), exist_ok=True)
    cache = json.load(open(CACHE_PATH)) if os.path.exists(CACHE_PATH) else {}
    mentions = {}
    for n, toks in candidates.items():
        kept, seen = [], set()
        for t in toks:
            login = resolve(t, cache)
            if login and login.lower() not in seen:
                seen.add(login.lower())
                kept.append(login)
        if kept:
            mentions[n] = kept
    json.dump(cache, open(CACHE_PATH, "w"), indent=1)

    old = json.load(open(out_path)) if os.path.exists(out_path) else {}
    for n in sorted(set(old) | set(mentions), key=int):
        o, w = set(old.get(n, [])), set(mentions.get(n, []))
        if o != w:
            drop, add = sorted(o - w), sorted(w - o)
            print(f"#{n}: " + " ".join([f"-{d}" for d in drop] + [f"+{a}" for a in add]))

    json.dump(mentions, open(out_path, "w"), indent=1)
    dead = sorted(k for k, v in cache.items() if not v)
    print(f"issues w/ mentions: {len(mentions)}/{len(raw)}; "
          f"unique logins: {len({l.lower() for v in mentions.values() for l in v})}; "
          f"unresolvable tokens ever seen: {len(dead)}")


if __name__ == "__main__":
    main()

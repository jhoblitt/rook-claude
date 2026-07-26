#!/usr/bin/env python3
"""Tier-0 KB mining analysis: bucket fetched PRs into areas, flag anomalies.

Deterministic analysis layer for the rook-triage kb refresh
(references/routing.md). Consumes rt_fetch.py output (rt_prs.jsonl +
rt_fetch_state.json) and emits the two-tier miner contract —
{"data": {...}, "flags": [...]} — so the orchestrator resolves trivial
flags and sends only survivors to a resolver agent; the assembler
validates before kb.json is written.

Area taxonomy = kb v3 (25 areas; rebucketed 2026-07-23: +build/design/
discover, core broadened). deploy/examples generic manifests and repo
meta files are DELIBERATELY unbucketed — they surface as bucket-ambiguity
flags to confirm the gap is still intentional, not silently dropped.

Data: per-area top reviewers (recency-weighted: 1.0 <=6mo, 0.5 <=12mo,
0.25 older; bots and self-reviews excluded; counted per review event) +
5 most recent items, plus authors_last_merged (YYYY-MM per author).
Flags: bucket-ambiguity (zero-match groups, >=6-area overmatch split
apis-driven vs cross-cutting) · truncation (scoped to counted PRs) ·
spec-boundary (fetch errors, unclean stop reason) · identity-unknown
(top reviewers outside the CODE-OWNERS roster) · coverage-gap (area has
PRs but zero reviews).

Usage: python3 rt_analyze.py --in-dir DIR
         (--code-owners /path/to/CODE-OWNERS | --roster a,b,c)
         [--out FILE] [--top 15]
"""
import argparse
import collections
import datetime
import json
import os
import re
import sys

AREAS = [
    "object", "object-multisite", "object-cosi", "object-bucket-claims",
    "ceph-mon", "ceph-osd", "ceph-mgr", "ceph-dashboard", "filesystem",
    "ceph-nfs", "csi", "block", "helm", "docs", "design", "ci", "test",
    "crd", "networking", "nvmeof", "ceph-external", "discover",
    "monitoring", "build", "core",
]

BOTS = ("mergify", "dependabot", "github-actions")

REPO_META = {
    ".mergify.yml", "PendingReleaseNotes.md", "ROADMAP.md", "ADOPTERS.md",
    "CODE-OWNERS", "README.md", "OWNERS.md", "SECURITY.md", "LICENSE",
    ".gitignore", ".github/PULL_REQUEST_TEMPLATE.md", "mkdocs.yml",
}


def areas_for(p):
    out = set()
    if "cosi" in p.lower():
        out.add("object-cosi")
    if p.startswith("pkg/operator/ceph/object/") or p.startswith("pkg/daemon/ceph/rgw/"):
        if "/cosi/" not in p:
            out.add("object")
        if any(t in p for t in ("multisite", "zone", "zonegroup", "realm")):
            out.add("object-multisite")
        if "/bucket/" in p:
            out.add("object-bucket-claims")
    if p.startswith("pkg/operator/ceph/cluster/mon/"):
        out.add("ceph-mon")
    if p.startswith(("pkg/operator/ceph/cluster/osd/", "pkg/daemon/ceph/osd/")):
        out.add("ceph-osd")
    if p.startswith("pkg/operator/ceph/cluster/mgr/"):
        out.add("ceph-mgr")
        if "dashboard" in p:
            out.add("ceph-dashboard")
    if p.startswith("pkg/operator/ceph/file/"):
        out.add("filesystem")
    if p.startswith("pkg/operator/ceph/nfs/"):
        out.add("ceph-nfs")
    if p.startswith("pkg/operator/ceph/csi/"):
        out.add("csi")
    if p.startswith("pkg/operator/ceph/pool/") or "rbdmirror" in p or "/rbd" in p:
        out.add("block")
    if p.startswith("deploy/charts/"):
        out.add("helm")
    if p.startswith("Documentation/"):
        out.add("docs")
    if p.startswith("design/"):
        out.add("design")
    if p.startswith((".github/workflows/", "tests/scripts/")):
        out.add("ci")
    elif p.startswith("tests/"):
        out.add("test")
    if p.startswith(("pkg/apis/", "pkg/client/")):
        out.add("crd")
    if "multus" in p or p.startswith("pkg/operator/ceph/controller/network"):
        out.add("networking")
    if "nvmeof" in p:
        out.add("nvmeof")
    if "external" in p:
        out.add("ceph-external")
    if p.startswith(("pkg/operator/discover/", "pkg/daemon/discover/")):
        out.add("discover")
    if p.startswith("pkg/operator/ceph/reporting/") or "exporter" in p or "monitoring" in p:
        out.add("monitoring")
    base = os.path.basename(p)
    if (p in ("go.mod", "go.sum", "go.work", "go.work.sum")
            or p.startswith(("build/", "images/"))
            or base == "Makefile" or base.endswith(".mk")
            or base.startswith((".golangci", ".commitlintrc", ".codespell"))):
        out.add("build")
    if not out:
        if p.startswith(("pkg/operator/", "pkg/daemon/ceph/", "pkg/util/",
                         "pkg/clusterd/", "cmd/", "pkg/")):
            out.add("core")
    return out


def is_bot(login):
    ll = login.lower()
    return ll.startswith(BOTS) or "copilot" in ll or ll.endswith("bot") or ll.endswith("[bot]")


def parse_code_owners(path):
    roster, key = set(), None
    for line in open(path):
        s = line.strip()
        m = re.match(r"^(approvers|reviewers):", s)
        if m:
            key = m.group(1)
            continue
        if key and s.startswith("- "):
            roster.add(s[2:].strip())
        elif s and not s.startswith(("#", "-")):
            key = None
    if not roster:
        raise SystemExit(f"no approvers/reviewers parsed from {path}")
    return roster


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--in-dir", required=True)
    ap.add_argument("--out")
    ap.add_argument("--code-owners")
    ap.add_argument("--roster", help="comma-separated logins (alternative to --code-owners)")
    ap.add_argument("--top", type=int, default=15)
    ap.add_argument("--now", help="ISO timestamp for recency weighting (default: current time); pin it for reproducible re-runs")
    args = ap.parse_args()

    if args.code_owners:
        roster = parse_code_owners(args.code_owners)
    elif args.roster:
        roster = {x.strip() for x in args.roster.split(",") if x.strip()}
    else:
        raise SystemExit("pass --code-owners or --roster (identity-unknown flags need it)")
    roster_lower = {r.lower() for r in roster}

    out_path = args.out or os.path.join(args.in_dir, "rt_final.json")
    state = json.load(open(os.path.join(args.in_dir, "rt_fetch_state.json")))
    now = (datetime.datetime.fromisoformat(args.now.replace("Z", "+00:00"))
           if args.now else datetime.datetime.now(datetime.timezone.utc))

    by_number = {}
    for line in open(os.path.join(args.in_dir, "rt_prs.jsonl")):
        line = line.strip()
        if line:
            pr = json.loads(line)
            by_number[pr["number"]] = pr
    prs = list(by_number.values())

    area_rev = {a: collections.defaultdict(lambda: {"raw": 0, "weighted": 0.0}) for a in AREAS}
    area_recent = {a: [] for a in AREAS}
    area_counts = {a: 0 for a in AREAS}
    authors_last = {}
    zero_match, overmatch = [], []
    flags = []

    for pr in prs:
        num, title = pr["number"], pr["title"]
        merged_dt = datetime.datetime.fromisoformat(pr["mergedAt"].replace("Z", "+00:00"))
        age = (now - merged_dt).days
        w = 1.0 if age <= 182 else (0.5 if age <= 365 else 0.25)

        author = (pr.get("author") or {}).get("login", "")
        if author:
            ym = pr["mergedAt"][:7]
            if author not in authors_last or ym > authors_last[author]:
                authors_last[author] = ym

        paths = [n["path"] for n in pr["files"]["nodes"] if n]
        pr_areas = set()
        for p in paths:
            pr_areas |= areas_for(p)

        if not pr_areas:
            zero_match.append((num, paths))
            continue
        if len(pr_areas) >= 6:
            overmatch.append((num, sorted(pr_areas), paths))

        events = []
        for r in pr["reviews"]["nodes"]:
            login = ((r or {}).get("author") or {}).get("login", "")
            if not login or login == author or is_bot(login):
                continue
            events.append(login)

        for a in pr_areas:
            area_counts[a] += 1
            area_recent[a].append((pr["mergedAt"], num, title))
            for login in events:
                area_rev[a][login]["raw"] += 1
                area_rev[a][login]["weighted"] += w

    ZERO_GROUPS = [
        ("deploy/examples generic manifests (deliberately unbucketed)",
         lambda paths: any(p.startswith("deploy/examples/") for p in paths)),
        ("repo meta files (deliberately unbucketed)",
         lambda paths: any(p in REPO_META or p.startswith(".docs/") for p in paths)),
    ]
    grouped = set()
    for label, pred in ZERO_GROUPS:
        nums = sorted({num for num, paths in zero_match if pred(paths)})
        if nums:
            grouped.update(nums)
            flags.append({
                "type": "bucket-ambiguity",
                "item": f"{len(nums)} PRs: no area rule matches — {label}",
                "evidence": f"PR numbers: {nums}",
                "question": "kb v3 leaves this class unbucketed on purpose — confirm the gap is still intentional, or name the area these should count toward.",
            })
    leftover = sorted({num for num, _ in zero_match if num not in grouped})
    if leftover:
        sample = {num: paths[:8] for num, paths in zero_match if num in leftover}
        flags.append({
            "type": "bucket-ambiguity",
            "item": f"{len(leftover)} PRs: no area rule matches — ungrouped/misc",
            "evidence": f"PR numbers: {leftover}; sample paths: {json.dumps(sample)[:1500]}",
            "question": "Not in either deliberate unbucketed class — what area (if any) should each count toward, or is a taxonomy/classifier fix needed?",
        })

    apis_driven = sorted({num for num, _, paths in overmatch
                          if any(p.startswith("pkg/apis/") for p in paths)})
    cross_cutting = sorted({num for num, _, _ in overmatch if num not in apis_driven})
    if apis_driven:
        flags.append({
            "type": "bucket-ambiguity",
            "item": f"{len(apis_driven)} PRs match >=6 areas — pkg/apis/** type changes fanning into regenerated CRD docs/charts",
            "evidence": f"PR numbers: {apis_driven}",
            "question": "Likely legitimate codegen blast-radius, not a classifier bug — confirm these should still count toward each touched area's reviewer stats.",
        })
    if cross_cutting:
        flags.append({
            "type": "bucket-ambiguity",
            "item": f"{len(cross_cutting)} PRs match >=6 areas — cross-cutting sweep, no pkg/apis change",
            "evidence": f"PR numbers: {cross_cutting}",
            "question": "Confirm genuine cross-cutting refactors (shared helpers/test framework) rather than the classifier over-matching unrelated paths.",
        })

    for t in state.get("truncations", []):
        if t["number"] not in by_number:
            continue
        flags.append({
            "type": "truncation",
            "item": f"PR #{t['number']}",
            "evidence": f"{t['kind']} pageInfo.hasNextPage=true (mergedAt={t.get('mergedAt')})",
            "question": f"PR has more {t['kind']} than fetched (100 files / 30 reviews cap) — counts for it may be incomplete.",
        })

    for e in state.get("errors", []):
        flags.append({
            "type": "spec-boundary",
            "item": "fetch pipeline",
            "evidence": e,
            "question": "Did this error drop or duplicate any PRs in the counted set?",
        })
    stop = state.get("stop_reason") or ""
    if not stop.startswith(("reached", "full page entirely", "no more pages")):
        flags.append({
            "type": "spec-boundary",
            "item": "stop condition",
            "evidence": f"pagination stopped due to: {stop!r} (pages_fetched={state.get('pages_fetched')})",
            "question": "Neither the cap nor a clean window/history boundary — is the dataset still valid for the stated window?",
        })

    identity_unknown = {}
    areas_out = {}
    for a in AREAS:
        revs = sorted(area_rev[a].items(),
                      key=lambda kv: (-kv[1]["weighted"], -kv[1]["raw"], kv[0].lower()))
        top = revs[:args.top]
        if area_counts[a] and not top:
            flags.append({
                "type": "coverage-gap",
                "item": a,
                "evidence": f"{area_counts[a]} PR(s) bucketed, 0 non-bot/non-self reviews recorded",
                "question": "Genuinely under-reviewed area, or is its path rule missing real hits?",
            })
        for login, v in top:
            if login.lower() not in roster_lower:
                acc = identity_unknown.setdefault(login, {"areas": set(), "raw": 0})
                acc["areas"].add(a)
                acc["raw"] += v["raw"]
        areas_out[a] = {
            "reviewers": [{"login": l, "weighted_reviews": round(v["weighted"], 2), "raw": v["raw"]}
                          for l, v in top],
            "recent_items": [{"number": n, "title": t}
                             for _, n, t in sorted(area_recent[a], reverse=True)[:5]],
        }

    for login, acc in sorted(identity_unknown.items(), key=lambda kv: -kv[1]["raw"]):
        flags.append({
            "type": "identity-unknown",
            "item": login,
            "evidence": f"raw_reviews_total={acc['raw']} across areas={sorted(acc['areas'])}",
            "question": "Not in the CODE-OWNERS roster and not an obvious bot — who is this / a legitimate community reviewer?",
        })

    oldest = state.get("oldest_mergedat") or ""
    result = {
        "data": {
            "generated_from": f"{state.get('counted')} merged PRs back to {oldest.split('T')[0] or 'unknown'}",
            "areas": areas_out,
            "authors_last_merged": dict(sorted(authors_last.items())),
        },
        "flags": flags,
    }
    json.dump(result, open(out_path, "w"), indent=1)

    counts = collections.Counter(f["type"] for f in flags)
    print(f"PRs={len(prs)} zero_match={len(zero_match)} overmatch={len(overmatch)} "
          f"flags={dict(counts)} -> {out_path}", file=sys.stderr)
    for a in ("object", "csi", "core", "build"):
        top3 = ", ".join(f"{r['login']}({r['weighted_reviews']}/{r['raw']})"
                         for r in areas_out[a]["reviewers"][:3])
        print(f"  {a}: {top3}", file=sys.stderr)


if __name__ == "__main__":
    main()

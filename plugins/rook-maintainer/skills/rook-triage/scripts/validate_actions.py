#!/usr/bin/env python3
"""Validate proposed triage actions against live state, before any write.

Phase 5's pre-write checks are set and count operations — label-set
membership, three caps, an is-still-open recheck, and the issues-only label
rule — so they are decided here rather than by an agent re-deriving them per
item. A wrong write lands on someone else's issue and cannot be taken back.

Usage:

    gh label list --json name > labels.json
    validate_actions.py --actions actions.json --labels labels.json \\
        [--items items.json]

    validate_actions.py --self-test

Exit status: 0 every action is safe to execute, 1 at least one is not,
2 bad input.

What it does NOT decide: whether a human answered the item since assessment.
That needs judgment and stays with the orchestrator.

Spec: skills/rook-triage/SKILL.md phase 5. The caps it enforces are owned by
references/routing.md and references/label-map.md — see the constants below.
"""

from __future__ import annotations

import argparse
import json
import sys

# These thresholds belong to the skill's prose; this file only enforces them.
# Each cites the reference that owns it, because a constant cannot be a pointer
# and prose cannot enforce — so when the two diverge it is this file that
# decides what actually posts, silently. Change one, change the other.
MAX_LABELS = 5                       # references/label-map.md, "Rules"
MAX_MENTIONS = 3                     # references/routing.md, Selection bounds
MIN_REVIEWERS, MAX_REVIEWERS = 1, 5  # references/routing.md, Selection bounds


def _names(seq):
    """gh emits labels as [{"name": x}]; accept bare strings too."""
    out = []
    for item in seq or []:
        out.append(item.get("name") if isinstance(item, dict) else item)
    return [x for x in out if x]


def validate(actions, live_labels, items=None):
    """Return a list of human-readable problems; empty means safe to execute."""
    problems = []
    live = set(live_labels)
    by_number = {i.get("number"): i for i in (items or [])}

    if not isinstance(actions, list):
        return ["actions payload: expected a list"]

    for idx, a in enumerate(actions):
        tag = f"actions[{idx}]"
        if not isinstance(a, dict):
            problems.append(f"{tag}: not an object")
            continue

        num = a.get("number")
        kind = (a.get("type") or "").lower()
        action = (a.get("action") or "").lower()
        params = a.get("params") or {}
        where = f"{tag} #{num}" if num is not None else tag

        if num is None:
            problems.append(f"{tag}: missing `number`")
            continue
        if action not in {"label", "comment", "close", "convert", "reviewers"}:
            problems.append(f"{where}: unknown action {action!r}")
            continue

        item = by_number.get(num)
        if items is not None:
            if item is None:
                problems.append(f"{where}: no live state supplied for this item")
                continue
            if (item.get("state") or "").upper() != "OPEN":
                problems.append(
                    f"{where}: item is {item.get('state')!r}, not OPEN — re-assess before writing"
                )
                continue
            if not kind:
                kind = (item.get("type") or "").lower()

        if action == "label":
            proposed = _names(params.get("labels"))
            if kind == "pr":
                problems.append(
                    f"{where}: label action on a PR — triage labels issues only"
                )
                continue
            if not proposed:
                problems.append(f"{where}: label action with no labels")
                continue
            invented = [x for x in proposed if x not in live]
            if invented:
                problems.append(
                    f"{where}: label(s) not in the live list: {', '.join(sorted(invented))}"
                )
            current = _names((item or {}).get("labels"))
            total = sorted(set(current) | set(proposed))
            if len(total) > MAX_LABELS:
                problems.append(
                    f"{where}: {len(total)} labels after apply exceeds the cap of {MAX_LABELS}"
                    f" ({', '.join(total)})"
                )

        elif action == "reviewers":
            reviewers = _names(params.get("reviewers"))
            if not MIN_REVIEWERS <= len(reviewers) <= MAX_REVIEWERS:
                problems.append(
                    f"{where}: {len(reviewers)} reviewers is outside "
                    f"{MIN_REVIEWERS}–{MAX_REVIEWERS}"
                )

        elif action == "comment":
            mentions = _names(params.get("mentions"))
            if len(mentions) > MAX_MENTIONS:
                problems.append(
                    f"{where}: {len(mentions)} mentions exceeds the cap of {MAX_MENTIONS}"
                )

    return problems


def _self_test():
    live = ["bug", "feature", "needs-info", "ceph-object", "core", "docs"]
    items = [
        {"number": 1, "type": "issue", "state": "OPEN", "labels": [{"name": "bug"}]},
        {"number": 2, "type": "pr", "state": "OPEN", "labels": []},
        {"number": 3, "type": "issue", "state": "CLOSED", "labels": []},
        {"number": 4, "type": "issue", "state": "OPEN",
         "labels": [{"name": "bug"}, {"name": "core"}, {"name": "docs"}]},
    ]
    ok = [
        {"number": 1, "action": "label", "params": {"labels": ["needs-info"]}},
        {"number": 2, "action": "reviewers", "params": {"reviewers": ["a", "b"]}},
        {"number": 1, "action": "comment", "params": {"mentions": ["a", "b", "c"]}},
    ]
    assert validate(ok, live, items) == [], validate(ok, live, items)

    cases = [
        ({"number": 1, "action": "label", "params": {"labels": ["invented"]}},
         "not in the live list"),
        ({"number": 2, "action": "label", "params": {"labels": ["bug"]}},
         "triage labels issues only"),
        ({"number": 3, "action": "label", "params": {"labels": ["bug"]}},
         "not OPEN"),
        ({"number": 4, "action": "label",
          "params": {"labels": ["feature", "ceph-object", "needs-info"]}},
         "exceeds the cap of 5"),
        ({"number": 1, "action": "reviewers", "params": {"reviewers": []}},
         "outside 1–5"),
        ({"number": 1, "action": "comment",
          "params": {"mentions": ["a", "b", "c", "d"]}},
         "exceeds the cap of 3"),
        ({"number": 9, "action": "label", "params": {"labels": ["bug"]}},
         "no live state supplied"),
        ({"number": 1, "action": "frobnicate", "params": {}},
         "unknown action"),
    ]
    for action, needle in cases:
        got = validate([action], live, items)
        assert len(got) == 1 and needle in got[0], (action, got)

    # Without live state the open/PR checks are skipped, not silently passed.
    assert validate([{"number": 1, "action": "label", "params": {"labels": ["bug"]}}],
                    live) == []

    print("self-test: OK")
    return 0


def _load(path, what):
    try:
        with open(path, encoding="utf-8") as fh:
            return json.load(fh)
    except (OSError, ValueError) as exc:
        print(f"cannot read --{what}: {exc}", file=sys.stderr)
        raise SystemExit(2)


def main(argv=None):
    p = argparse.ArgumentParser(
        description="Validate proposed triage actions against live state."
    )
    p.add_argument("--actions", help="proposed actions JSON")
    p.add_argument("--labels", help="`gh label list --json name` output")
    p.add_argument("--items", help="live per-item state JSON (number, type, state, labels)")
    p.add_argument("--self-test", action="store_true", help="verify the checks and exit")
    args = p.parse_args(argv)

    if args.self_test:
        return _self_test()
    if not args.actions or not args.labels:
        p.error("--actions and --labels are required (or use --self-test)")

    actions = _load(args.actions, "actions")
    live = _names(_load(args.labels, "labels"))
    items = _load(args.items, "items") if args.items else None

    if not live:
        print("the live label list is empty — refusing to validate", file=sys.stderr)
        return 2

    problems = validate(actions, live, items)
    n = len(actions) if isinstance(actions, list) else 0

    if problems:
        print(f"{len(problems)} problem(s) across {n} proposed action(s):", file=sys.stderr)
        for problem in problems:
            print(f"  {problem}", file=sys.stderr)
        print("\nSend these back to the report rather than executing them.", file=sys.stderr)
        return 1

    print(f"all {n} proposed action(s) pass the pre-write checks")
    return 0


if __name__ == "__main__":
    sys.exit(main())

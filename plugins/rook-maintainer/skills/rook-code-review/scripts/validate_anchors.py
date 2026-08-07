#!/usr/bin/env python3
"""Validate a review payload's inline anchors against the PR diff.

GitHub rejects the ENTIRE review call when one inline comment names a line the
diff does not touch, and accepts-then-misplaces a comment whose `side` is wrong.
Both outcomes are decided by set membership over the diff's hunks, so they are
decided here rather than by an agent reading the diff.

Usage:

    gh pr diff <n> > pr.diff
    validate_anchors.py --review review.json --diff pr.diff

    gh pr diff <n> | validate_anchors.py --review review.json

    validate_anchors.py --self-test

Exit status: 0 every anchor is postable, 1 at least one is not, 2 bad input.

Spec: skills/rook-code-review/references/posting.md.
"""

from __future__ import annotations

import argparse
import json
import re
import sys

HUNK_RE = re.compile(r"^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@")

LEFT = "LEFT"
RIGHT = "RIGHT"


def commentable(diff_text):
    """Map path -> {"LEFT": {line, ...}, "RIGHT": {line, ...}}.

    A line is commentable on RIGHT when it is added or context (it exists in the
    new file inside a hunk), and on LEFT when it is removed or context. Context
    lines are commentable on both, which is why `side` cannot be inferred from
    the line number alone.
    """
    files = {}
    path = None
    old_path = None
    new_path = None
    old_ln = new_ln = 0
    in_hunk = False

    for raw in diff_text.splitlines():
        if raw.startswith("diff --git "):
            path, old_path, new_path, in_hunk = None, None, None, False
            continue
        if raw.startswith("--- "):
            old_path = _strip_prefix(raw[4:])
            in_hunk = False
            continue
        if raw.startswith("+++ "):
            new_path = _strip_prefix(raw[4:])
            # GitHub anchors a deleted file at its original path.
            path = new_path if new_path is not None else old_path
            if path is not None:
                files.setdefault(path, {LEFT: set(), RIGHT: set()})
            in_hunk = False
            continue

        m = HUNK_RE.match(raw)
        if m:
            old_ln, new_ln = int(m.group(1)), int(m.group(3))
            in_hunk = True
            continue

        if not in_hunk or path is None:
            continue

        # A wholly empty line inside a hunk is an empty context line.
        marker = raw[0] if raw else " "
        if marker == " ":
            files[path][LEFT].add(old_ln)
            files[path][RIGHT].add(new_ln)
            old_ln += 1
            new_ln += 1
        elif marker == "-":
            files[path][LEFT].add(old_ln)
            old_ln += 1
        elif marker == "+":
            files[path][RIGHT].add(new_ln)
            new_ln += 1
        elif marker == "\\":
            # "\ No newline at end of file" — advances neither counter.
            continue
        else:
            in_hunk = False

    return files


def _strip_prefix(spec):
    """`a/pkg/x.go` -> `pkg/x.go`; `/dev/null` -> None."""
    spec = spec.split("\t", 1)[0].strip()
    if spec == "/dev/null":
        return None
    if len(spec) > 2 and spec[1] == "/" and spec[0] in "ab":
        return spec[2:]
    return spec


def validate(review, files):
    """Return a list of human-readable problems; empty means postable."""
    problems = []
    comments = review.get("comments") or []
    if not isinstance(comments, list):
        return ["review payload: `comments` must be a list"]

    for i, c in enumerate(comments):
        tag = f"comments[{i}]"
        if not isinstance(c, dict):
            problems.append(f"{tag}: not an object")
            continue

        path = c.get("path")
        line = c.get("line")
        side = c.get("side", RIGHT)
        start_line = c.get("start_line")
        start_side = c.get("start_side")

        if not path:
            problems.append(f"{tag}: missing `path`")
            continue
        if not isinstance(line, int):
            problems.append(f"{tag} {path}: `line` must be an integer")
            continue
        if side not in (LEFT, RIGHT):
            problems.append(f"{tag} {path}:{line}: `side` must be LEFT or RIGHT, got {side!r}")
            continue
        if path not in files:
            problems.append(f"{tag} {path}:{line}: file is not in the diff")
            continue

        if (start_line is None) != (start_side is None):
            problems.append(
                f"{tag} {path}:{line}: multi-line anchors need BOTH `start_line` and `start_side`"
            )
            continue
        if start_line is not None:
            if start_side != side:
                problems.append(
                    f"{tag} {path}:{line}: `start_side` ({start_side}) must equal `side` ({side})"
                )
                continue
            if not isinstance(start_line, int) or start_line > line:
                problems.append(
                    f"{tag} {path}:{line}: `start_line` ({start_line}) must be an integer <= `line`"
                )
                continue
            if start_line not in files[path][side]:
                problems.append(
                    f"{tag} {path}:{start_line} {side}: start line is outside the diff"
                )
                continue

        if line not in files[path][side]:
            other = LEFT if side == RIGHT else RIGHT
            hint = f" (it IS commentable on {other} — wrong side?)" if line in files[path][other] else ""
            problems.append(f"{tag} {path}:{line} {side}: line is outside the diff{hint}")

    return problems


SELF_TEST_DIFF = """diff --git a/pkg/keep.go b/pkg/keep.go
--- a/pkg/keep.go
+++ b/pkg/keep.go
@@ -10,4 +10,5 @@ func Keep() {
 	ctx := context.TODO()
-	old(ctx)
+	newer(ctx)
+	extra(ctx)
 }
diff --git a/build/gone.mk b/build/gone.mk
--- a/build/gone.mk
+++ /dev/null
@@ -1,2 +0,0 @@
-all:
-	echo hi
"""


def _self_test():
    files = commentable(SELF_TEST_DIFF)
    expected_files = {"pkg/keep.go", "build/gone.mk"}
    assert set(files) == expected_files, files

    keep = files["pkg/keep.go"]
    # Header context line 10 / removal 11 on LEFT; additions 11-12 on RIGHT.
    assert keep[RIGHT] == {10, 11, 12, 13}, keep[RIGHT]
    assert keep[LEFT] == {10, 11, 12}, keep[LEFT]
    # A file deleted outright anchors LEFT-only, at its original path.
    assert files["build/gone.mk"][RIGHT] == set()
    assert files["build/gone.mk"][LEFT] == {1, 2}

    ok = {"comments": [
        {"path": "pkg/keep.go", "line": 11, "side": RIGHT},
        {"path": "build/gone.mk", "start_line": 1, "start_side": LEFT, "line": 2, "side": LEFT},
    ]}
    assert validate(ok, files) == [], validate(ok, files)

    cases = [
        ({"path": "pkg/keep.go", "line": 99, "side": RIGHT}, "outside the diff"),
        ({"path": "build/gone.mk", "line": 1, "side": RIGHT}, "wrong side?"),
        ({"path": "pkg/absent.go", "line": 1, "side": RIGHT}, "not in the diff"),
        ({"path": "pkg/keep.go", "line": 11, "side": RIGHT, "start_line": 10}, "BOTH"),
        ({"path": "pkg/keep.go", "line": 11, "side": RIGHT,
          "start_line": 10, "start_side": LEFT}, "must equal"),
    ]
    for comment, needle in cases:
        got = validate({"comments": [comment]}, files)
        assert len(got) == 1 and needle in got[0], (comment, got)

    print("self-test: OK")
    return 0


def main(argv=None):
    p = argparse.ArgumentParser(
        description="Validate a review payload's inline anchors against the PR diff."
    )
    p.add_argument("--review", help="review payload JSON (the gh --input file)")
    p.add_argument("--diff", help="unified diff; omit to read stdin")
    p.add_argument("--self-test", action="store_true", help="verify the parser and exit")
    args = p.parse_args(argv)

    if args.self_test:
        return _self_test()
    if not args.review:
        p.error("--review is required (or use --self-test)")

    try:
        with open(args.review, encoding="utf-8") as fh:
            review = json.load(fh)
    except (OSError, ValueError) as exc:
        print(f"cannot read --review: {exc}", file=sys.stderr)
        return 2

    if args.diff:
        try:
            with open(args.diff, encoding="utf-8") as fh:
                diff_text = fh.read()
        except OSError as exc:
            print(f"cannot read --diff: {exc}", file=sys.stderr)
            return 2
    else:
        if sys.stdin.isatty():
            print("no diff: pass --diff FILE or pipe `gh pr diff <n>`", file=sys.stderr)
            return 2
        diff_text = sys.stdin.read()

    if not diff_text.strip():
        print("the diff is empty — nothing can be anchored", file=sys.stderr)
        return 2

    files = commentable(diff_text)
    problems = validate(review, files)
    n = len(review.get("comments") or [])

    if problems:
        print(f"{len(problems)} of {n} inline anchor(s) cannot be posted:", file=sys.stderr)
        for problem in problems:
            print(f"  {problem}", file=sys.stderr)
        print(
            "\nFold each into the review BODY under \"Other observations\" and say "
            "there that it is unanchored (posting.md).",
            file=sys.stderr,
        )
        return 1

    print(f"all {n} inline anchor(s) land inside the diff")
    return 0


if __name__ == "__main__":
    sys.exit(main())

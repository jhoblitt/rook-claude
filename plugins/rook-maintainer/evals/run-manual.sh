#!/usr/bin/env bash
#
# Run one eval case by hand, against THIS checkout's canon.
#
# `claude plugin eval` is a no-op on stock installs, so until that gate opens
# a case is exercised in two separate `claude -p` invocations: one that plays
# the subject and one that grades it. They must stay separate — a session that
# has read graders/criteria.md writes a report that satisfies it.
#
#   ./run-manual.sh crossref-overclose
#   ./run-manual.sh design-precision -o /tmp/evalrun
#
set -euo pipefail

EVALS_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
PLUGIN_ROOT=$(cd -- "$EVALS_DIR/.." && pwd -P)

MODEL=""
OUT_DIR=""
CASE=""

usage() {
	cat <<-USAGE
		usage: run-manual.sh <case> [-m MODEL] [-o OUT_DIR] [-s]

		  -m MODEL    model for both passes (default: the CLI's default)
		  -o OUT_DIR  where to write the report and grade (default: a temp dir)
		  -s          subject pass only; skip grading

		cases: $(cd "$EVALS_DIR" && printf '%s ' */prompt.md | sed 's|/prompt.md||g')
	USAGE
}

while [ $# -gt 0 ]; do
	case "$1" in
	-m) MODEL=${2:?-m needs a model}; shift 2 ;;
	-o) OUT_DIR=${2:?-o needs a directory}; shift 2 ;;
	-s) SUBJECT_ONLY=1; shift ;;
	-h | --help) usage; exit 0 ;;
	-*) echo "unknown flag: $1" >&2; usage >&2; exit 2 ;;
	*) CASE=$1; shift ;;
	esac
done
SUBJECT_ONLY=${SUBJECT_ONLY:-0}

[ -n "$CASE" ] || { usage >&2; exit 2; }

CASE_DIR="$EVALS_DIR/$CASE"
PROMPT="$CASE_DIR/prompt.md"
CRITERIA="$CASE_DIR/graders/criteria.md"
[ -f "$PROMPT" ] || { echo "no such case: $CASE (missing $PROMPT)" >&2; exit 2; }

# component-loading asserts that the plugin's components register in a fresh
# session. That is a property of the INSTALLED plugin, so redirecting canon at
# a checkout cannot test it — install the branch and start a session instead.
if [ "$CASE" = "component-loading" ]; then
	echo "component-loading tests session registration, not file canon;" >&2
	echo "it cannot be run this way. Install the branch and check /plugin." >&2
	exit 2
fi

OUT_DIR=${OUT_DIR:-$(mktemp -d -t "eval-$CASE-XXXXXX")}
mkdir -p "$OUT_DIR"
REPORT="$OUT_DIR/report.md"
GRADE="$OUT_DIR/grade.md"

declare -a MODEL_ARGS=()
[ -n "$MODEL" ] && MODEL_ARGS=(--model "$MODEL")

# Prompts go in on stdin, never as the positional argument: --allowedTools and
# --add-dir are variadic, so a trailing prompt is parsed as more tool names.
subject_prompt() {
	cat <<-SUBJECT
		You are the SUBJECT of a regression test for the rook-maintainer plugin.
		Someone else grades your output.

		CANON SOURCE, overriding anything below about a "loaded plugin": do NOT
		use any installed or loaded rook-maintainer plugin and do NOT invoke its
		skills by name. The installed copy is a different release than the canon
		under test. Read the plugin under test from:

		  $PLUGIN_ROOT

		For a code-review case that means $PLUGIN_ROOT/skills/rook-code-review/SKILL.md
		first, then the reference files its own routing table directs you to under
		that skill's references/ directory, always including verification.md. Route
		honestly from what the target actually touches.

		HARD RULE: do not read any graders/ directory, any other eval case, or
		anything under docs/ or .superpowers/. Reading the grading criteria or the
		design intent would invalidate this run.

		Where the case names a command you cannot run, say what it would have
		checked and record it as a coverage gap rather than inventing a result.

		Your task follows. Follow it exactly. Your entire final answer is what it
		asks for — nothing added, no meta-commentary about being tested.

		$(cat "$PROMPT")
	SUBJECT
}

# The canon redirect is the whole point. A fresh session loads the INSTALLED
# rook-maintainer, which is whatever release is on disk — not this branch. An
# un-redirected run grades the wrong canon and fails for the wrong reason.
echo "==> subject pass: $CASE" >&2
subject_prompt | claude -p "${MODEL_ARGS[@]}" \
	--add-dir "$PLUGIN_ROOT" --allowedTools Read Grep Glob | tee "$REPORT"

if [ "$SUBJECT_ONLY" = "1" ]; then
	echo "==> report: $REPORT" >&2
	exit 0
fi

[ -f "$CRITERIA" ] || { echo "no criteria at $CRITERIA; report left at $REPORT" >&2; exit 2; }

grader_prompt() {
	cat <<-GRADER
		Grade a review report against an eval case's criteria. Be strict: the
		criteria are numbered conditions, and "close enough" is a fail. Answer
		each numbered condition PASS or FAIL with the evidence you relied on,
		check the Fail list too, then give one overall verdict line.

		=== CRITERIA ===
		$(cat "$CRITERIA")

		=== REPORT UNDER TEST ===
		$(cat "$REPORT")
	GRADER
}

# Separate invocation on purpose: the grader may see the criteria, the subject
# may not, and neither may be the session that wrote the canon.
echo "==> grading pass: $CASE" >&2
grader_prompt | claude -p "${MODEL_ARGS[@]}" --allowedTools Read | tee "$GRADE"

echo "==> report: $REPORT" >&2
echo "==> grade:  $GRADE" >&2

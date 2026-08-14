#!/usr/bin/env bash
# Reject a breaking-change footer the subject never declared.
#
# conventional-commits-parser treats any line beginning with BREAKING CHANGE or
# BREAKING-CHANGE, followed by a colon or whitespace, as a footer — with no
# regard for whether it is prose. Body text that merely wraps the keyword onto
# the start of a line makes semantic-release cut a major release, silently.
# Requiring the subject to carry "!" as well makes the intent explicit, so
# accidental prose no longer looks like a deliberate break.
set -euo pipefail

base="${1:?usage: check-breaking-footer.sh <base-sha> <head-sha>}"
head="${2:?usage: check-breaking-footer.sh <base-sha> <head-sha>}"

# Merge commits carry generated bodies nobody wrote, and the preset drops them.
mapfile -t commits < <(git rev-list --no-merges "${base}..${head}")

offenders=0
for sha in "${commits[@]}"; do
  body=$(git log -1 --format='%b' "${sha}")
  printf '%s\n' "${body}" \
    | grep -qE '^[[:space:]*]*BREAKING[ -]CHANGE([[:space:]:]|$)' || continue

  subject=$(git log -1 --format='%s' "${sha}")
  if printf '%s' "${subject}" | grep -qE '^[a-zA-Z]+(\([^)]*\))?!:'; then
    continue
  fi

  printf '%s declares no break in its subject but its body starts a line with the keyword:\n' "${sha:0:8}"
  printf '  subject: %s\n' "${subject}"
  printf '%s\n' "${body}" | grep -nE '^[[:space:]*]*BREAKING[ -]CHANGE([[:space:]:]|$)' \
    | sed 's/^/  body line /'
  offenders=$((offenders + 1))
done

if [ "${offenders}" -ne 0 ]; then
  cat >&2 <<'MSG'

Either declare the break in the subject (feat!: ...), or reword the body so the
keyword does not begin a line. semantic-release reads it as a footer either
way, and a major release cannot be undone without rewriting history.
MSG
  exit 1
fi

printf 'checked %d commit(s): no undeclared breaking-change footers\n' "${#commits[@]}"

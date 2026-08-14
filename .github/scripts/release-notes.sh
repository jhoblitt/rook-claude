#!/usr/bin/env bash
# Emit the PR-derived half of a release note: GitHub's own "What's Changed"
# list, plus the issues those PRs close. semantic-release appends this to the
# commit-derived notes; see .releaserc.yml.
set -euo pipefail

repo="${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is unset}"
tag="${1:?usage: release-notes.sh <next-tag> [previous-tag]}"
prev="${2-}"

# The tag does not exist yet — semantic-release generates notes before it
# publishes — so the range has to be anchored on a commit instead.
args=(-X POST "repos/${repo}/releases/generate-notes"
      -f "tag_name=${tag}"
      -f "target_commitish=${GITHUB_SHA:-main}")
if [ -n "${prev}" ]; then
  args+=(-f "previous_tag_name=${prev}")
fi

# semantic-release's own header already links the tag comparison.
notes=$(gh api "${args[@]}" --jq '.body' | sed '/^\*\*Full Changelog\*\*:/d')

prs=$(printf '%s\n' "${notes}" | grep -oE '/pull/[0-9]+' | grep -oE '[0-9]+' | sort -un || true)

# Only a PR's closing links record which issues a release actually resolved:
# this repo's commits reference issues with "Refs #N", which closes nothing.
issues=""
if [ -n "${prs}" ]; then
  query="query{repository(owner:\"${repo%%/*}\",name:\"${repo##*/}\"){"
  for pr in ${prs}; do
    query+="p${pr}:pullRequest(number:${pr}){closingIssuesReferences(first:50){nodes{number title url}}}"
  done
  query+="}}"
  issues=$(gh api graphql -f query="${query}" --jq '
    [.data.repository[].closingIssuesReferences.nodes[]]
    | unique_by(.number) | sort_by(.number)
    | .[] | "* [#\(.number)](\(.url)) \(.title)"')
fi

printf '%s\n' "${notes}"
if [ -n "${issues}" ]; then
  printf '\n### Resolved issues\n\n%s\n' "${issues}"
fi

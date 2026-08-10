#!/usr/bin/env bash
# Build-on-first-use launcher for the Go tools in tools/cmd/.
#
#   run.sh <tool> [args...]
#
# Unlike hooks/webfetch-guard.sh this fails LOUD. A hook that cannot build
# must get out of the way; a tool a skill was told to run must never look
# like it ran and found nothing — a silent no-op here would read as "no dead
# links", "no invalid actions", "empty dashboard", which is worse than an
# error. Every failure path prints to stderr and exits non-zero.
#
# Correctness under parallel invocations comes from the per-PID temp plus
# atomic mv; flock, where present, only dedupes the work.
set -uo pipefail

die() {
  printf 'rook-maintainer tools: %s\n' "$1" >&2
  exit 127
}

[ $# -ge 1 ] || die "usage: run.sh <tool> [args...]"
tool=$1
shift

src="${CLAUDE_PLUGIN_ROOT:-}/tools"
[ -n "${CLAUDE_PLUGIN_ROOT:-}" ] || die "CLAUDE_PLUGIN_ROOT is unset"
[ -d "$src/cmd/$tool" ] || die "unknown tool '$tool' (no $src/cmd/$tool)"

# Binaries live in the per-plugin data directory, never in CLAUDE_PLUGIN_ROOT:
# the cache is a per-version directory that moves on every update and can be
# reaped mid-session. Data outlives versions, and a fresh version clone carries
# fresh mtimes, so the staleness check rebuilds after updates on its own.
data="${CLAUDE_PLUGIN_DATA:-${XDG_CACHE_HOME:-$HOME/.cache}/rook-claude}/tools"
bin="$data/$tool"

stale() {
  [ ! -x "$bin" ] && return 0
  # The parens matter: without them -o binds looser than the implicit -a and
  # every .go file matches regardless of mtime, so the tool rebuilds on every
  # single invocation.
  [ -n "$(find "$src" \( -name '*.go' -o -name go.mod \) -newer "$bin" -print -quit 2>/dev/null)" ]
}

build() {
  (cd "$src" && go build -o "$bin.tmp.$$" "./cmd/$tool") && mv -f "$bin.tmp.$$" "$bin" && return 0
  rm -f "$bin.tmp.$$"
  return 1
}

if stale; then
  command -v go >/dev/null 2>&1 || die "go toolchain not found; cannot build '$tool'"
  mkdir -p "$data" || die "cannot create $data"
  if command -v flock >/dev/null 2>&1; then
    exec 9>"$bin.lock" || die "cannot open build lock"
    flock 9 || die "cannot take build lock"
    if stale; then
      build || die "build failed for '$tool' (run: cd $src && go build ./cmd/$tool)"
    fi
    exec 9>&-
  else
    build || die "build failed for '$tool' (run: cd $src && go build ./cmd/$tool)"
  fi
fi

exec "$bin" "$@"

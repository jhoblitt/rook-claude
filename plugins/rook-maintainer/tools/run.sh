#!/usr/bin/env bash
# Build-on-first-use launcher for the Go tools in tools/cmd/.
#
#   run.sh <tool> [args...]
#
# The fail-loud contract is skills/rook-conventions/references/plugin-tools.md.
# Unlike hooks/webfetch-guard.sh, which must get out of the way when it cannot
# build, every failure path here prints to stderr and exits non-zero.
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
# reaped mid-session. Data outlives versions, so the staleness check below is
# all that ties a cached binary to the sources of the version now installed.
data="${CLAUDE_PLUGIN_DATA:-${XDG_CACHE_HOME:-$HOME/.cache}/rook-claude}/tools"
bin="$data/$tool"

# The cache path holds a binary; nothing about it says WHICH binary, and the
# mtime test cannot ask -- a rename, a half-finished copy, a leftover from
# another checkout all read as fresh once they are newer than the sources. So
# the launcher stamps what it built: a cksum over every source file, in a
# stable order, and a cksum over the binary that came out. Both must still
# hold, or the tool is rebuilt.
#
# This is an identity check, not a tamper boundary. The stamp sits in the same
# user-writable directory as the binary, so whoever can plant one can rewrite
# the other; a real boundary would have to keep the expected value somewhere
# the launcher's inputs are not, such as inside CLAUDE_PLUGIN_ROOT.
#
# The sources are a tree, so the flat `cat "$src"/*.go` of
# hooks/webfetch-guard.sh will not do. -print0, sort -z and xargs -0 are GNU
# and BSD but not POSIX: when one is missing the pipeline fails (pipefail) or
# reads zero bytes, and staleness falls back to the mtime test alone, which is
# where it stood before.
srcsum() {
  local fp
  fp=$(find "$src" \( -name '*.go' -o -name go.mod \) -print0 2>/dev/null |
    LC_ALL=C sort -z 2>/dev/null | xargs -0 cat 2>/dev/null | cksum) || return 1
  case "$fp" in
  *' 0') return 1 ;;
  esac
  printf '%s' "$fp"
}

binsum() { cksum <"$bin" 2>/dev/null; }

stale() {
  [ ! -x "$bin" ] && return 0
  local sfp bfp
  if sfp=$(srcsum) && bfp=$(binsum); then
    [ "$(cat "$bin.fp" 2>/dev/null)" != "$sfp $bfp" ]
    return
  fi
  # The parens matter: without them -o binds looser than the implicit -a and
  # every .go file matches regardless of mtime, so the tool rebuilds on every
  # single invocation.
  [ -n "$(find "$src" \( -name '*.go' -o -name go.mod \) -newer "$bin" -print -quit 2>/dev/null)" ]
}

# The source cksum is taken BEFORE the build so it describes what go build
# read: sources edited mid-build stamp as the older value and the next
# invocation rebuilds, where stamping afterwards would record the edit as
# already built. A stamp that cannot be written is removed rather than left
# outdated -- that costs a rebuild, never a wrong binary.
build() {
  local sfp bfp
  sfp=$(srcsum) || sfp=
  if (cd "$src" && go build -o "$bin.tmp.$$" "./cmd/$tool") && mv -f "$bin.tmp.$$" "$bin"; then
    if [ -n "$sfp" ] && bfp=$(binsum) &&
      printf '%s\n' "$sfp $bfp" >"$bin.fp.tmp.$$" 2>/dev/null; then
      mv -f "$bin.fp.tmp.$$" "$bin.fp" 2>/dev/null || rm -f "$bin.fp.tmp.$$" "$bin.fp"
    else
      rm -f "$bin.fp.tmp.$$" "$bin.fp"
    fi
    return 0
  fi
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

#!/usr/bin/env bash
# Build-on-first-use launcher for the Go guard in hooks/webfetch-guard/.
#
# The LAUNCHER fails open; the GUARD fails closed. That split is deliberate.
# A toolchain that cannot build must not brick every WebFetch in the session,
# and a build failure is not an attack — but once the guard runs, every
# verdict it reaches on an attacker-chosen URL denies. The consequence worth
# stating plainly: with no Go toolchain there is no allowlist, only the prose
# rule. `go build` once, or accept that.
#
# Correctness under parallel hooks comes from the per-PID temp plus atomic mv;
# flock, where present, only dedupes the work.
#
# The kill switch is checked here as well as in Go: the escape hatch for a
# broken guard cannot live only inside the artifact whose build may be what
# broke.
set -uo pipefail

# Every fail-open path announces itself. Silence makes an unguarded fetch
# indistinguishable from an adjudicated one, and the build-failure marker below
# can carry a single transient lapse for an hour. Reasons are compile-time
# constants with no untrusted input, so a static document needs no encoder.
notice() {
  printf '{"systemMessage":"rook-maintainer webfetch-guard is not adjudicating this fetch (%s)"}\n' "$1"
  exit 0
}

[ "${ROOK_WEBFETCH_GUARD:-on}" = "off" ] && notice "disabled by ROOK_WEBFETCH_GUARD=off"

src="${CLAUDE_PLUGIN_ROOT:-}/hooks/webfetch-guard"
if [ -z "${CLAUDE_PLUGIN_ROOT:-}" ] || [ ! -d "$src" ]; then
  notice "guard source not found"
fi

# The binary lives in the per-plugin data directory, never in
# CLAUDE_PLUGIN_ROOT: the cache is a per-version directory that moves on every
# update and can be reaped mid-session. Data outlives versions, and a fresh
# version clone carries fresh mtimes, so the staleness check below rebuilds
# after updates on its own.
data="${CLAUDE_PLUGIN_DATA:-${XDG_CACHE_HOME:-$HOME/.cache}/rook-claude}"
bin="$data/webfetch-guard"

stale() {
  [ ! -x "$bin" ] && return 0
  [ -n "$(find "$src" -maxdepth 1 \( -name '*.go' -o -name go.mod \) \
    -newer "$bin" -print -quit 2>/dev/null)" ]
}

# A failed build must not retry on every WebFetch. The marker pins the source
# fingerprint it failed on: a source change re-arms immediately (the dev
# iterating on a fix), expiry re-arms after transient toolchain trouble.
# Success removes it.
build() {
  local marker="$bin.buildfail" fp
  fp=$(cat "$src"/go.mod "$src"/*.go 2>/dev/null | cksum)
  if [ -f "$marker" ] && [ "$(cat "$marker" 2>/dev/null)" = "$fp" ] &&
    [ -z "$(find "$marker" -mmin +60 -print 2>/dev/null)" ]; then
    return 1
  fi
  if (cd "$src" && go build -o "$bin.tmp.$$" .) >/dev/null 2>&1 &&
    mv -f "$bin.tmp.$$" "$bin" 2>/dev/null; then
    rm -f "$marker"
    return 0
  fi
  rm -f "$bin.tmp.$$"
  printf '%s' "$fp" >"$marker" 2>/dev/null
  return 1
}

if stale; then
  command -v go >/dev/null 2>&1 || notice "no go toolchain on PATH"
  mkdir -p "$data" 2>/dev/null || notice "the data directory is not writable"
  if command -v flock >/dev/null 2>&1; then
    exec 9>"$bin.lock" || notice "the build lock is not writable"
    flock 9 || notice "the build lock could not be taken"
    if stale; then
      build || notice "the guard did not build"
    fi
    exec 9>&-
  else
    build || notice "the guard did not build"
  fi
fi

exec "$bin"

#!/usr/bin/env bash
# Non-interactive glab wrapper for TT_CMD_GLAB.
# Adds --yes --fill to `glab mr create` so `tt pr` can run without a TTY.
# Does not recurse: always execs the real glab binary (TT_CMD_GLAB_REAL or glab).

set -euo pipefail

REAL="${TT_CMD_GLAB_REAL:-glab}"

if [[ "${1:-}" == "mr" && "${2:-}" == "create" ]]; then
  shift 2
  has_yes=false
  has_fill=false
  for a in "$@"; do
    case "$a" in
      --yes|-y) has_yes=true ;;
      --fill) has_fill=true ;;
    esac
  done
  extra=()
  [[ "$has_yes" == "true" ]] || extra+=(--yes)
  [[ "$has_fill" == "true" ]] || extra+=(--fill)
  exec "$REAL" mr create "${extra[@]}" "$@"
fi

exec "$REAL" "$@"

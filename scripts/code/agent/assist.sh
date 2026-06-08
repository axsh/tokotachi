#!/usr/bin/env bash
# scripts/code/agent/assist.sh -- tt agent assist wrapper
# Maps Coding Agent options to tt agent assist options explicitly.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../_resolve_tool.sh"

TT_ARGS=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --scope) TT_ARGS+=(--scope "$2"); shift 2 ;;
    --force) TT_ARGS+=(--force);      shift ;;
    *)
      echo "[ERROR] Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

exec "$TOOL" agent assist "${TT_ARGS[@]}"

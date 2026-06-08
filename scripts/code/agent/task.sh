#!/usr/bin/env bash
# scripts/code/agent/task.sh -- tt agent task wrapper
# Maps Coding Agent options to tt agent task options explicitly.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../_resolve_tool.sh"

if [[ $# -lt 1 ]]; then
  echo "[ERROR] Usage: task.sh <show|submit> [args...]" >&2
  exit 1
fi

SUBCMD="$1"
shift

TT_ARGS=()
case "$SUBCMD" in
  show)
    if [[ $# -lt 1 ]]; then
      echo "[ERROR] Usage: task.sh show <task-id>" >&2
      exit 1
    fi
    TT_ARGS+=("$1")
    shift
    ;;
  submit)
    if [[ $# -lt 1 ]]; then
      echo "[ERROR] Usage: task.sh submit <task-id> --file <path>" >&2
      exit 1
    fi
    TT_ARGS+=("$1")
    shift
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --file) TT_ARGS+=(--file "$2"); shift 2 ;;
        *)
          echo "[ERROR] Unknown argument: $1" >&2
          exit 1
          ;;
      esac
    done
    ;;
  *)
    echo "[ERROR] Unknown subcommand: $SUBCMD" >&2
    exit 1
    ;;
esac

exec "$TOOL" agent task "$SUBCMD" "${TT_ARGS[@]}"

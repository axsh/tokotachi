#!/usr/bin/env bash
# Launch development environment for a feature.
# Wrapper around tt up.
# Usage: ./scripts/dist/dev <feature-name> [args...]
# Example: ./scripts/dist/dev tt

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

# ─── Argument check ─────────────────────────────────────────────────
if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <feature-name> [args...]"
  echo "Example: $0 tt"
  exit 1
fi

FEATURE_NAME="$1"
shift

# ─── Ensure tt is installed ─────────────────────────────────────
NATIVE_OS="$(detect_os)"
TT_BIN="${REPO_ROOT}/bin/tt"
[[ "$NATIVE_OS" == "windows" ]] && TT_BIN="${DEVCTL}.exe"

if [[ ! -x "$TT_BIN" ]]; then
  warn "tt not found in bin/. Installing..."
  "${SCRIPT_DIR}/install-tools.sh" tt
fi

if [[ ! -x "$TT_BIN" ]]; then
  fail "Failed to install tt"
  exit 1
fi

# ─── Launch development environment ─────────────────────────────────
info "Starting development environment for ${FEATURE_NAME}..."
exec "$TT_BIN" up "$FEATURE_NAME" "$@"

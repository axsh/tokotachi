#!/usr/bin/env bash
# Install built CLI tools to local bin/ directory.
# Usage: ./scripts/dist/install-tools [--all | <tool-id>...]
# Example: ./scripts/dist/install-tools devctl
#          ./scripts/dist/install-tools --all

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

# ─── Argument check ─────────────────────────────────────────────────
if [[ $# -lt 1 ]]; then
  echo "Usage: $0 [--all | <tool-id>...]"
  echo "Example: $0 devctl"
  echo "         $0 --all"
  exit 1
fi

# ─── Determine tool IDs ─────────────────────────────────────────────
TOOL_IDS=()
if [[ "$1" == "--all" ]]; then
  while IFS= read -r id; do
    id="$(echo "$id" | tr -d '\r')"
    TOOL_IDS+=("$id")
  done < <(get_all_tool_ids)
else
  TOOL_IDS=("$@")
fi

# ─── Detect native platform ─────────────────────────────────────────
NATIVE_OS="$(detect_os)"
NATIVE_ARCH="$(detect_arch)"
info "Native platform: ${NATIVE_OS}/${NATIVE_ARCH}"

# ─── Install each tool ──────────────────────────────────────────────
BIN_DIR="${REPO_ROOT}/bin"
mkdir -p "$BIN_DIR"

installed=0
failed=0

for tool_id in "${TOOL_IDS[@]}"; do
  BINARY_NAME="$(get_field "$tool_id" "['binary_name']" | tr -d '\r')"

  ext=""
  [[ "$NATIVE_OS" == "windows" ]] && ext=".exe"

  src="${REPO_ROOT}/dist/${tool_id}/${BINARY_NAME}_${NATIVE_OS}_${NATIVE_ARCH}${ext}"

  # Auto-build if binary not found
  if [[ ! -f "$src" ]]; then
    warn "${tool_id} not built yet. Building..."
    "${SCRIPT_DIR}/build.sh" "$tool_id"
  fi

  if [[ -f "$src" ]]; then
    cp "$src" "${BIN_DIR}/${BINARY_NAME}${ext}"
    chmod +x "${BIN_DIR}/${BINARY_NAME}${ext}"
    pass "Installed ${BINARY_NAME}${ext} → bin/"
    installed=$((installed + 1))
  else
    fail "Failed to install ${tool_id}: binary not found after build"
    failed=$((failed + 1))
  fi
done

# ─── Summary ────────────────────────────────────────────────────────
echo ""
if [[ $failed -eq 0 ]]; then
  pass "All ${installed} tools installed to ${BIN_DIR}/"
else
  fail "${failed} tool(s) failed to install."
  exit 1
fi

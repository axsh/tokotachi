#!/usr/bin/env bash
# Build CLI tools for all platforms defined in their manifest.
# Usage: ./scripts/dist/build <tool-id>
# Example: ./scripts/dist/build tt

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

# ─── Argument check ─────────────────────────────────────────────────
if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <tool-id>"
  echo "Example: $0 tt"
  exit 1
fi

TOOL_ID="$1"
validate_tool_id "$TOOL_ID"

# ─── Read manifest ──────────────────────────────────────────────────
MANIFEST="$(manifest_path "$TOOL_ID")"
if [[ ! -f "$MANIFEST" ]]; then
  fail "Manifest not found: $MANIFEST"
  exit 1
fi

FEATURE_PATH="$(get_field "$TOOL_ID" "['feature_path']" | tr -d '\r')"
BINARY_NAME="$(get_field "$TOOL_ID" "['binary_name']" | tr -d '\r')"
MAIN_PACKAGE="$(get_field "$TOOL_ID" "['main_package']" | tr -d '\r')"

info "Building ${TOOL_ID} (${BINARY_NAME}) from ${FEATURE_PATH}/${MAIN_PACKAGE}"

# ─── Cross-compile for each platform ────────────────────────────────
DIST_DIR="${REPO_ROOT}/dist/${TOOL_ID}"
mkdir -p "$DIST_DIR"

total=0
passed=0
failed=0

while read -r os arch; do
  total=$((total + 1))

  # Windows binaries need .exe extension
  ext=""
  [[ "$os" == "windows" ]] && ext=".exe"

  output="${DIST_DIR}/${BINARY_NAME}_${os}_${arch}${ext}"

  info "Building for ${os}/${arch}..."
  if CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
     go build -C "${REPO_ROOT}/${FEATURE_PATH}" -o "$output" "${MAIN_PACKAGE}"; then
    pass "${os}/${arch} → $(basename "$output")"
    passed=$((passed + 1))
  else
    fail "${os}/${arch} build failed"
    failed=$((failed + 1))
  fi
done < <(get_platforms "$TOOL_ID" | tr -d '\r')

# ─── Summary ────────────────────────────────────────────────────────
echo ""
if [[ $failed -eq 0 ]]; then
  pass "All ${total} builds succeeded. Output: ${DIST_DIR}/"
else
  fail "${failed}/${total} builds failed."
  exit 1
fi

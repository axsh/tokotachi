#!/usr/bin/env bash
# Create release artifacts (archives + checksums) from built binaries.
# Usage: ./scripts/dist/tool/internal/package.sh <tool-id> <version>
# Example: ./scripts/dist/tool/internal/package.sh tt v1.0.0

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../../shared/_lib.sh"

# ─── Argument check ─────────────────────────────────────────────────
if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <tool-id> <version>"
  echo "Example: $0 tt v1.0.0"
  exit 1
fi

TOOL_ID="$1"
validate_tool_id "$TOOL_ID"
VERSION="$2"

# ─── Verify built binaries exist ────────────────────────────────────
BUILD_DIR="${REPO_ROOT}/dist/${TOOL_ID}"
if [[ ! -d "$BUILD_DIR" ]] || [[ -z "$(ls -A "$BUILD_DIR" 2>/dev/null)" ]]; then
  fail "No built binaries found in ${BUILD_DIR}/."
  echo "  Run './scripts/dist/tool/internal/build.sh ${TOOL_ID}' first."
  exit 1
fi

BINARY_NAME="$(get_field "$TOOL_ID" "['binary_name']" | tr -d '\r')"

info "Creating release artifacts for ${TOOL_ID} ${VERSION}..."

# ─── Create release directory ───────────────────────────────────────
RELEASE_DIR="${REPO_ROOT}/dist/${TOOL_ID}/${VERSION}"
mkdir -p "$RELEASE_DIR"

# ─── Create archives for each platform ──────────────────────────────
total=0
created=0

while read -r os arch; do
  total=$((total + 1))

  ext=""
  [[ "$os" == "windows" ]] && ext=".exe"

  binary="${BUILD_DIR}/${BINARY_NAME}_${os}_${arch}${ext}"
  archive_name="${BINARY_NAME}_${os}_${arch}"

  if [[ ! -f "$binary" ]]; then
    warn "Binary not found: $(basename "$binary") (skipping)"
    continue
  fi

  info "Creating archive for ${os}/${arch}..."

  # Copy binary to temp dir with clean name for archive
  tmp_dir="$(mktemp -d)"
  cp "$binary" "${tmp_dir}/${BINARY_NAME}${ext}"

  if [[ "$os" == "windows" ]]; then
    # Copy installer scripts to temp dir
    cp "$(dirname "${BASH_SOURCE[0]}")/install.ps1" "${tmp_dir}/"
    cp "$(dirname "${BASH_SOURCE[0]}")/uninstall.ps1" "${tmp_dir}/"

    # Windows: zip archive (with fallback to PowerShell)
    if command -v zip &>/dev/null; then
      (cd "$tmp_dir" && zip -q "${RELEASE_DIR}/${archive_name}.zip" "${BINARY_NAME}${ext}" "install.ps1" "uninstall.ps1")
    else
      # Fallback: use PowerShell Compress-Archive
      win_src_pattern="$(cygpath -w "${tmp_dir}")\\*"
      win_dst="$(cygpath -w "${RELEASE_DIR}/${archive_name}.zip")"
      powershell -NoProfile -Command "Compress-Archive -Path '${win_src_pattern}' -DestinationPath '${win_dst}' -Force"
    fi
    pass "${archive_name}.zip"
  else
    # Linux/macOS: tar.gz archive
    # Use --force-local on Windows hosts to avoid tar interpreting drive letters (C:) as remote hosts
    local tar_opts="czf"
    [[ "$(detect_os)" == "windows" ]] && tar_opts="--force-local -czf"
    (cd "$tmp_dir" && tar $tar_opts "${RELEASE_DIR}/${archive_name}.tar.gz" "${BINARY_NAME}${ext}")
    pass "${archive_name}.tar.gz"
  fi

  rm -rf "$tmp_dir"
  created=$((created + 1))
done < <(get_platforms "$TOOL_ID" | tr -d '\r')

# ─── Generate checksums ─────────────────────────────────────────────
info "Generating checksums..."
(cd "$RELEASE_DIR" && sha256sum *.tar.gz *.zip 2>/dev/null > checksums.txt)
pass "checksums.txt created"

# ─── Summary ────────────────────────────────────────────────────────
echo ""
if [[ $created -eq $total ]]; then
  pass "All ${total} archives created. Output: ${RELEASE_DIR}/"
else
  warn "${created}/${total} archives created."
fi
ls -la "$RELEASE_DIR/"

#!/usr/bin/env bash
# Publish release artifacts to GitHub Releases.
# Usage: ./scripts/dist/publish <tool-id> <version>
# Example: ./scripts/dist/publish tt v1.0.0

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

# ─── Argument check ─────────────────────────────────────────────────
if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <tool-id> <version>"
  echo "Example: $0 tt v1.0.0"
  exit 1
fi

TOOL_ID="$1"
VERSION="$2"

# ─── Prerequisites ──────────────────────────────────────────────────
if ! command -v gh &>/dev/null; then
  fail "GitHub CLI (gh) is not installed."
  echo "  Install: https://cli.github.com/"
  exit 1
fi

if ! gh auth status &>/dev/null; then
  fail "GitHub CLI is not authenticated. Run 'gh auth login' first."
  exit 1
fi

# ─── Verify release artifacts exist ─────────────────────────────────
RELEASE_DIR="${REPO_ROOT}/dist/${TOOL_ID}/${VERSION}"
if [[ ! -d "$RELEASE_DIR" ]] || [[ -z "$(ls -A "$RELEASE_DIR" 2>/dev/null)" ]]; then
  fail "No release artifacts found in ${RELEASE_DIR}/"
  echo "  Run './scripts/dist/release ${TOOL_ID} ${VERSION}' first."
  exit 1
fi

# ─── Create GitHub Release ──────────────────────────────────────────
TAG="${TOOL_ID}-${VERSION}"
TITLE="${TOOL_ID} ${VERSION}"

# Release notes
NOTES_FILE="${REPO_ROOT}/releases/notes/latest.md"
if [[ -f "$NOTES_FILE" ]]; then
  NOTES_FLAG="--notes-file ${NOTES_FILE}"
else
  warn "No release notes found. Using auto-generated notes."
  NOTES_FLAG="--generate-notes"
fi

info "Creating GitHub Release: ${TAG}..."
# shellcheck disable=SC2086
gh release create "$TAG" \
  --title "$TITLE" \
  $NOTES_FLAG \
  "${RELEASE_DIR}"/*

pass "Published ${TOOL_ID} ${VERSION} to GitHub Releases"

# ─── Next steps ─────────────────────────────────────────────────────
echo ""
info "=== Next Steps ==="
echo "  Homebrew: Update tools/installers/homebrew/Formula/tt.rb"
echo "  Scoop:    Update tools/installers/scoop/tt.json"
echo ""
echo "  SHA256 checksums are in: ${RELEASE_DIR}/checksums.txt"

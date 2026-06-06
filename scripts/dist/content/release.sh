#!/usr/bin/env bash
# Content release pipeline: builds/verifies → runs templatizer → commits & pushes.
# Usage: ./scripts/dist/content/release.sh [OPTIONS]
#
# Options:
#   --help    Show this help message

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../shared/_lib.sh"

show_help() {
  cat << 'EOF'
Usage: ./scripts/dist/content/release.sh [OPTIONS]

1. Runs build.sh (full build & unit tests)
2. Runs templatizer to regenerate catalog data
3. Commits and pushes catalog changes to main branch

Options:
  --help    Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --help|-h)
      show_help
      exit 0
      ;;
    *)
      fail "Unknown option: $1"
      show_help
      exit 1
      ;;
  esac
done

info "=== Step 1/3: Full Build & Test ==="
# Run build.sh from process
if "${REPO_ROOT}/scripts/process/build.sh"; then
  pass "Build and unit tests passed."
else
  fail "Build pipeline failed. Aborting release."
  exit 1
fi

info "=== Step 2/3: Regenerating Catalog ==="
# Determine templatizer binary name based on OS
BINARY_NAME="templatizer"
if [[ "$(detect_os)" == "windows" ]]; then
  BINARY_NAME="templatizer.exe"
fi

TEMPLATIZER_BIN="${REPO_ROOT}/bin/${BINARY_NAME}"
if [[ ! -f "$TEMPLATIZER_BIN" ]]; then
  fail "templatizer binary not found at $TEMPLATIZER_BIN. Please run build.sh first."
  exit 1
fi

info "Running templatizer on catalog..."
if "$TEMPLATIZER_BIN" "${REPO_ROOT}/catalog"; then
  pass "Catalog regenerated successfully."
else
  fail "Templatizer failed. Aborting release."
  exit 1
fi

info "=== Step 3/3: Commit & Push to Remote ==="
cd "$REPO_ROOT"

# Stage all catalog files and index/meta
git add catalog/scaffolds/ catalog.yaml meta.yaml

if git diff --cached --quiet; then
  warn "No catalog changes to commit. Skipping push."
else
  info "Committing changes..."
  git commit -m 'update catalog'
  info "Pushing to main..."
  if git push origin main; then
    pass "Successfully pushed catalog updates to main!"
  else
    fail "Git push failed."
    exit 1
  fi
fi

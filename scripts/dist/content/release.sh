#!/usr/bin/env bash
# ==============================================================================
# Content Release Pipeline Script (release.sh)
#
# This script automates the complete packaging and release flow for catalog templates:
# 1. Full Verification: Runs the build and unit test pipeline (build.sh) to ensure
#    everything (backend, tools, and originals) compiles and passes tests.
# 2. Catalog Regeneration: Invokes the compiled templatizer binary to scan
#    catalog/originals, construct template ZIP packages, generate FNV-1a sharded
#    YAML metadata files inside catalog/scaffolds/, and update the top-level
#    catalog.yaml index and meta.yaml metadata.
# 3. Remote Publishing: Stages all generated catalog files, commits them with
#    single-quoted message 'update catalog', and pushes the updates to the current
#    active branch of the remote repository (git push origin <current-branch>).
#
# Usage: ./scripts/dist/content/release.sh [OPTIONS]
#
# Options:
#   --help    Show this help message
# ==============================================================================

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../shared/_lib.sh"

show_help() {
  cat << 'EOF'
Usage: ./scripts/dist/content/release.sh [OPTIONS]

1. Runs build.sh (full build & unit tests)
2. Runs templatizer to regenerate catalog data
3. Commits and pushes catalog changes to current branch

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
  
  current_branch=$(git rev-parse --abbrev-ref HEAD)
  info "Pushing to ${current_branch}..."
  if git push origin "${current_branch}"; then
    pass "Successfully pushed catalog updates to ${current_branch}!"
  else
    fail "Git push failed."
    exit 1
  fi
fi

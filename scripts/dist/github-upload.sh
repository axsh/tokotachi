#!/usr/bin/env bash
# All-in-one distribution: build → release → publish to GitHub.
# Usage: ./scripts/dist/github-upload.sh <tool-id> [version|+increment]
# Examples:
#   ./scripts/dist/github-upload.sh tt v1.2.0     # absolute version
#   ./scripts/dist/github-upload.sh tt +v0.1.0    # increment from current
#   ./scripts/dist/github-upload.sh tt             # defaults to +v0.0.1

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

# ─── Version helpers ────────────────────────────────────────────────

# Validate that a version string matches vN.N.N format.
# Args: $1 = version string (e.g. "v1.2.3")
validate_version_format() {
  local ver="$1"
  if [[ ! "$ver" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    fail "Invalid version format: '${ver}'. Expected: v{N}.{N}.{N}"
    exit 1
  fi
}

# Parse a version string into major/minor/patch variables.
# Args: $1 = version string (e.g. "v1.2.3")
# Sets: _major, _minor, _patch
parse_semver() {
  local ver="${1#v}"
  IFS='.' read -r _major _minor _patch <<< "$ver"
}

# Get the current (latest) version for a tool from GitHub Releases.
# Args: $1 = tool-id
# Outputs: version string (e.g. "v1.2.3") or "v0.0.0" if no releases exist
get_current_version() {
  local tool_id="$1"

  if ! command -v gh &>/dev/null; then
    fail "GitHub CLI (gh) is not installed."
    echo "  Install: https://cli.github.com/"
    exit 1
  fi

  local tag
  tag=$(gh release list --limit 100 --json tagName --jq \
    "[.[] | select(.tagName | startswith(\"${tool_id}-v\"))] | sort_by(.tagName) | last | .tagName // empty")

  if [[ -z "$tag" ]]; then
    echo "v0.0.0"
  else
    # Strip tool-id prefix: "tt-v1.0.0" → "v1.0.0"
    echo "${tag#${tool_id}-}"
  fi
}

# Compute new version from current + increment.
# Args: $1 = current version, $2 = increment version
# Outputs: new version string
compute_incremented_version() {
  local cur="$1" inc="$2"

  parse_semver "$cur"
  local cur_major=$_major cur_minor=$_minor cur_patch=$_patch

  parse_semver "$inc"
  local inc_major=$_major inc_minor=$_minor inc_patch=$_patch

  local new_major=$((cur_major + inc_major))
  local new_minor=$((cur_minor + inc_minor))
  local new_patch=$((cur_patch + inc_patch))

  echo "v${new_major}.${new_minor}.${new_patch}"
}

# Check that new version is strictly greater than current version.
# Args: $1 = current version, $2 = new version
compare_versions() {
  local cur="$1" new="$2"

  parse_semver "$cur"
  local cur_major=$_major cur_minor=$_minor cur_patch=$_patch

  parse_semver "$new"
  local new_major=$_major new_minor=$_minor new_patch=$_patch

  # Compare: new must be strictly greater than current
  if [[ $new_major -gt $cur_major ]]; then
    return 0
  elif [[ $new_major -eq $cur_major ]]; then
    if [[ $new_minor -gt $cur_minor ]]; then
      return 0
    elif [[ $new_minor -eq $cur_minor ]]; then
      if [[ $new_patch -gt $cur_patch ]]; then
        return 0
      fi
    fi
  fi

  # new <= current
  if [[ "$new" == "$cur" ]]; then
    fail "Version ${new} is the same as current version ${cur}."
  else
    fail "Version ${new} is lower than current version ${cur}."
  fi
  exit 1
}

# ─── Argument parsing ──────────────────────────────────────────────

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <tool-id> [version|+increment]"
  echo ""
  echo "Examples:"
  echo "  $0 tt v1.2.0     # absolute version"
  echo "  $0 tt +v0.1.0    # increment from current"
  echo "  $0 tt             # defaults to +v0.0.1 (patch bump)"
  exit 1
fi

TOOL_ID="$1"
validate_tool_id "$TOOL_ID"
VERSION_ARG="${2:-+v0.0.1}"

# Determine mode: increment or absolute
if [[ "$VERSION_ARG" == +* ]]; then
  MODE="increment"
  RAW_VERSION="${VERSION_ARG#+}"
else
  MODE="absolute"
  RAW_VERSION="$VERSION_ARG"
fi

# ─── Validate version format ───────────────────────────────────────

validate_version_format "$RAW_VERSION"

# ─── Determine new version ─────────────────────────────────────────

info "Fetching current version for ${TOOL_ID}..."
CURRENT_VERSION=$(get_current_version "$TOOL_ID")
info "Current version: ${CURRENT_VERSION}"

if [[ "$MODE" == "increment" ]]; then
  NEW_VERSION=$(compute_incremented_version "$CURRENT_VERSION" "$RAW_VERSION")
else
  NEW_VERSION="$RAW_VERSION"
fi

# ─── Version check ─────────────────────────────────────────────────

compare_versions "$CURRENT_VERSION" "$NEW_VERSION"

# ─── Confirmation ──────────────────────────────────────────────────

echo ""
echo "╔══════════════════════════════════════╗"
echo "║     GitHub Upload Pipeline           ║"
echo "╚══════════════════════════════════════╝"
echo ""
info "Tool:    ${TOOL_ID}"
info "Current: ${CURRENT_VERSION}"
info "New:     ${NEW_VERSION}"
echo ""

# ─── Step 1: Build ──────────────────────────────────────────────────

info "=== Step 1/3: Build ==="
"${SCRIPT_DIR}/build.sh" "$TOOL_ID"

# ─── Step 2: Release ───────────────────────────────────────────────

info "=== Step 2/3: Release ==="
"${SCRIPT_DIR}/release.sh" "$TOOL_ID" "$NEW_VERSION"

# ─── Step 3: Publish ───────────────────────────────────────────────

info "=== Step 3/3: Publish ==="
"${SCRIPT_DIR}/publish.sh" "$TOOL_ID" "$NEW_VERSION"

# ─── Done ───────────────────────────────────────────────────────────

echo ""
pass "Successfully uploaded ${TOOL_ID} ${NEW_VERSION} to GitHub!"

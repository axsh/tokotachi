#!/usr/bin/env bash
# All-in-one distribution: build → release → publish to GitHub.
# Usage: ./scripts/dist/tool/release.sh <tool-id> [version|+increment]
# Examples:
#   ./scripts/dist/tool/release.sh tt v1.2.0     # absolute version
#   ./scripts/dist/tool/release.sh tt +v0.1.0    # increment from current
#   ./scripts/dist/tool/release.sh tt             # defaults to +v0.0.1

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../shared/_lib.sh"

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
    "[.[] | select(.tagName | test(\"^v[0-9]\"))] | sort_by(.tagName) | last | .tagName // empty")

  if [[ -z "$tag" ]]; then
    echo "v0.0.0"
  else
    echo "$tag"
  fi
}

# Compute new version from current + increment.
# Args: $1 = current version, $2 = increment version
# Outputs: new version string
compute_incremented_version() {
  local cur="$1" inc="$2"

  parse_semver "$cur"
  local cur_major=$_major cur_minor=$_minor cur_patch=$_patch

  # Remove '+' prefix
  local raw_inc="${inc#+}"

  # Normalize inc value by adding placeholders if components are missing
  if [[ "$raw_inc" =~ ^v[0-9]+$ ]]; then
    raw_inc="${raw_inc}.0.0"
  elif [[ "$raw_inc" =~ ^v[0-9]+\.[0-9]+$ ]]; then
    raw_inc="${raw_inc}.0"
  fi

  parse_semver "$raw_inc"
  local inc_major=$_major inc_minor=$_minor inc_patch=$_patch

  local new_major=$cur_major
  local new_minor=$cur_minor
  local new_patch=$cur_patch

  if [[ $inc_major -gt 0 ]]; then
    new_major=$((cur_major + inc_major))
    new_minor=0
    new_patch=0
  elif [[ $inc_minor -gt 0 ]]; then
    new_minor=$((cur_minor + inc_minor))
    new_patch=0
  elif [[ $inc_patch -gt 0 ]]; then
    new_patch=$((cur_patch + inc_patch))
  fi

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

  # Validate increment format: no trailing .0 allowed.
  # Must be +v{major}, +v0.{minor}, or +v0.0.{patch}
  if [[ ! "$VERSION_ARG" =~ ^\+v([1-9][0-9]*|0\.[1-9][0-9]*|0\.0\.[1-9][0-9]*)$ ]]; then
    fail "Invalid increment format: '${VERSION_ARG}'."
    echo "  Allowed formats:"
    echo "    Major bump: +v1, +v2..."
    echo "    Minor bump: +v0.1, +v0.2..."
    echo "    Patch bump: +v0.0.1, +v0.0.2..."
    echo "  (Trailing .0 is not allowed. e.g. +v0.1.0 or +v1.0.0 is invalid)"
    exit 1
  fi
else
  MODE="absolute"
  RAW_VERSION="$VERSION_ARG"
fi

# ─── Validate version format ───────────────────────────────────────

if [[ "$MODE" == "absolute" ]]; then
  validate_version_format "$RAW_VERSION"
fi

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

info "=== Step 1/4: Build ==="
"${SCRIPT_DIR}/internal/build.sh" "$TOOL_ID"

# ─── Step 2: Generate Release Notes ────────────────────────────────

info "=== Step 2/4: Generate Release Notes ==="
RELEASE_NOTE_DIR="${REPO_ROOT}/features/release-note"
if [[ -f "${RELEASE_NOTE_DIR}/go.mod" ]]; then
  if (cd "$RELEASE_NOTE_DIR" && go run . \
        --tool-id "$TOOL_ID" \
        --version "$NEW_VERSION" \
        --repo-root "$REPO_ROOT"); then
    pass "Release notes generated."
  else
    warn "Release note generation failed. Continuing with auto-generated notes."
  fi
else
  warn "Release note generator not found. Skipping."
fi

# ─── Step 3: Release ───────────────────────────────────────────────

info "=== Step 3/4: Release ==="
"${SCRIPT_DIR}/internal/package.sh" "$TOOL_ID" "$NEW_VERSION"

# ─── Step 4: Publish ───────────────────────────────────────────────

info "=== Step 4/4: Publish ==="
"${SCRIPT_DIR}/internal/publish.sh" "$TOOL_ID" "$NEW_VERSION"

# ─── Done ───────────────────────────────────────────────────────────

echo ""
pass "Successfully uploaded ${TOOL_ID} ${NEW_VERSION} to GitHub!"

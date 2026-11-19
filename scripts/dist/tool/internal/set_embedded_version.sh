#!/usr/bin/env bash
# Rewrite the tt embedded Version constant in a Go source file.
# Usage: ./scripts/dist/tool/internal/set_embedded_version.sh <version.go> <new-version>
# Example: ./scripts/dist/tool/internal/set_embedded_version.sh features/tt/internal/version/version.go v0.9.2

set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <version.go> <new-version>"
  echo "Example: $0 features/tt/internal/version/version.go v0.9.2"
  exit 1
fi

VERSION_FILE="$1"
NEW_VERSION="$2"

if [[ ! "$NEW_VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "ERROR: Invalid version format: '${NEW_VERSION}'. Expected: v{N}.{N}.{N}" >&2
  exit 1
fi

if [[ ! -f "$VERSION_FILE" ]]; then
  echo "ERROR: Version file not found: ${VERSION_FILE}" >&2
  exit 1
fi

# Exactly one const Version = "..." line must exist (allow CRLF).
match_count=$(grep -cE '^const Version = "[^"]+"\r?$' "$VERSION_FILE" || true)
if [[ "$match_count" -ne 1 ]]; then
  echo "ERROR: Expected exactly one 'const Version = \"...\"' line in ${VERSION_FILE}, found ${match_count}" >&2
  exit 1
fi

# Replace only the Version constant line (leave unrelated "version:" text untouched).
tmp_file="$(mktemp)"
trap 'rm -f "$tmp_file"' EXIT

awk -v ver="$NEW_VERSION" '
  /^const Version = "[^"]+"\r?$/ {
    print "const Version = \"" ver "\""
    next
  }
  { print }
' "$VERSION_FILE" > "$tmp_file"

mv "$tmp_file" "$VERSION_FILE"
trap - EXIT

if ! grep -qE "^const Version = \"${NEW_VERSION}\"\r?\$" "$VERSION_FILE"; then
  echo "ERROR: After rewrite, Version constant does not equal ${NEW_VERSION} in ${VERSION_FILE}" >&2
  exit 1
fi

echo "OK: Embedded version ${NEW_VERSION} into ${VERSION_FILE}"

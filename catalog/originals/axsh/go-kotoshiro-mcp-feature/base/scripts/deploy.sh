#!/usr/bin/env bash
# Build the production container image and optionally push to a registry.
# Prerequisites: bin/function.exe must exist (run ./scripts/run.sh build.sh first).
# Set REGISTRY (e.g. docker.io/myuser, ghcr.io/owner/repo) to tag and push.
# Image name and version come from registry.json (name defaults to function folder name).

set -e
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
# shellcheck source=scripts/_common.sh
. "$SCRIPT_DIR/_common.sh"

cd "$FUNCTION_ROOT"

# Ensure binary exists
if [ ! -f bin/function.exe ]; then
  echo "bin/function.exe not found. Running build..."
  "$SCRIPT_DIR/run.sh" build.sh
fi

# Read version and optional name from registry.json (defaults when file is missing)
REGISTRY_JSON="$FUNCTION_ROOT/registry.json"
VERSION=latest
REGISTRY_NAME="$FUNCTION_NAME"
if [ -f "$REGISTRY_JSON" ]; then
  VERSION=$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$REGISTRY_JSON")
  REGISTRY_NAME=$(sed -n 's/.*"name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$REGISTRY_JSON")
  [ -z "$VERSION" ] && VERSION=latest
  [ -z "$REGISTRY_NAME" ] && REGISTRY_NAME="$FUNCTION_NAME"
fi

echo "Building production image: $IMAGE_PROD"
docker build --target prod -t "$IMAGE_PROD" -f container/Dockerfile .

if [ -z "$REGISTRY" ]; then
  echo "Image built. To push to a registry, set REGISTRY and run again, e.g.:"
  echo "  REGISTRY=docker.io/myuser $0"
  echo "  REGISTRY=ghcr.io/owner/repo $0"
  IMAGE_REF="$IMAGE_PROD"
else
  REMOTE_IMAGE="${REGISTRY}/${REGISTRY_NAME}:${VERSION}"
  echo "Tagging and pushing: $REMOTE_IMAGE"
  docker tag "$IMAGE_PROD" "$REMOTE_IMAGE"
  docker push "$REMOTE_IMAGE"
  echo "Pushed $REMOTE_IMAGE"
  IMAGE_REF="$REMOTE_IMAGE"
fi

# Report deploy result via message channel (FD 3).
# _common.sh opens FD 3 to KOTOSHIRO_MESSAGE_PORT when the caller (ProcessManager) provides it.
# Simple JSON: {"docker_image": "<pullable ref>"}. The tool that invokes this script is responsible for
# persisting it (e.g. into .kotoshiro/settings.json) according to its own format.
( echo "{\"docker_image\": \"$IMAGE_REF\"}" >&3 ) 2>/dev/null || true

[ -z "$REGISTRY" ] && exit 0

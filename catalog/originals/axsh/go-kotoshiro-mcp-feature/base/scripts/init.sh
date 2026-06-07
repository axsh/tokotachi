#!/usr/bin/env bash
set -e
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
# shellcheck source=scripts/_common.sh
. "$SCRIPT_DIR/_common.sh"

cd "$FUNCTION_ROOT"
docker build --target dev -t "$IMAGE_DEV" -f container/Dockerfile .
# Align go.mod/go.sum with sources, fill go.sum (incl. zip checksums), and warm the module cache for run.sh/build.sh.
docker run --rm \
  -v "$FUNCTION_ROOT:/workspace/$FUNCTION_NAME" \
  -v "gomodcache_${FUNCTION_NAME}:/go/pkg/mod" \
  -e GOMODCACHE=/go/pkg/mod \
  -w "/workspace/$FUNCTION_NAME" \
  "$IMAGE_DEV" \
  go mod tidy

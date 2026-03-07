#!/bin/bash
set -euo pipefail

# ============================================================
# needs_rebuild.sh — Docker イメージの再ビルドが必要か判定
#
# Usage:
#   needs_rebuild.sh <compose_file> <service_name> <source_dir> [<additional_dir>...]
#
# 終了コード:
#   0 = 再ビルドが必要
#   1 = 再ビルド不要（イメージは最新）
#   2 = 判定不能（引数不足等）
#
# 依存:
#   scripts/utils/get_latest.sh
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
GET_LATEST="$SCRIPT_DIR/get_latest.sh"

show_help() {
    cat << 'EOF'
Usage: needs_rebuild.sh <compose_file> <service_name> <source_dir> [<additional_dir>...]

Checks if a Docker image needs rebuilding by comparing
the latest source file modification time against the image creation time.

Uses get_latest.sh to determine the newest source file.

Exit codes:
  0 = Rebuild needed (source is newer than image)
  1 = No rebuild needed (image is up to date)
  2 = Cannot determine (arguments missing, etc.)

Examples:
  # Check if roslyn-server needs rebuild
  ./scripts/utils/needs_rebuild.sh \
    docker/docker-compose.yml roslyn-server \
    services/roslyn-server docker/roslyn-server
EOF
}

if [[ "${1:-}" == "--help" ]] || [[ "$#" -lt 3 ]]; then
    show_help
    exit 2
fi

COMPOSE_FILE="$1"
SERVICE_NAME="$2"
shift 2
SOURCE_DIRS=("$@")

# --- Verify get_latest.sh exists ---
if [[ ! -f "$GET_LATEST" ]]; then
    echo "Error: get_latest.sh not found at $GET_LATEST" >&2
    exit 2
fi

# --- Get latest source file modification time via get_latest.sh ---
LATEST_OUTPUT=$(bash "$GET_LATEST" "${SOURCE_DIRS[@]}" 2>/dev/null) || {
    echo "No source files found — rebuild needed" >&2
    exit 0
}
LATEST_SOURCE_EPOCH=$(echo "$LATEST_OUTPUT" | cut -d' ' -f1)
LATEST_FILE=$(echo "$LATEST_OUTPUT" | cut -d' ' -f2-)

# --- Get Docker image creation time ---

# Determine docker compose command
if docker compose version >/dev/null 2>&1; then
    DOCKER_COMPOSE_CMD="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
    DOCKER_COMPOSE_CMD="docker-compose"
else
    echo "docker compose not found" >&2
    exit 2
fi

# Get the image name for the service
IMAGE_NAME=$($DOCKER_COMPOSE_CMD -f "$COMPOSE_FILE" images "$SERVICE_NAME" --format json 2>/dev/null \
    | grep -o '"Repository":"[^"]*"' | head -1 | cut -d'"' -f4) || true

if [[ -z "$IMAGE_NAME" ]]; then
    IMAGE_NAME=$($DOCKER_COMPOSE_CMD -f "$COMPOSE_FILE" config --images 2>/dev/null \
        | grep "$SERVICE_NAME" | head -1) || true
fi

if [[ -z "$IMAGE_NAME" ]]; then
    echo "Image not found for service '$SERVICE_NAME' — rebuild needed" >&2
    exit 0
fi

# Get image creation timestamp
IMAGE_CREATED=$(docker inspect --format '{{.Created}}' "$IMAGE_NAME" 2>/dev/null) || {
    echo "Cannot inspect image '$IMAGE_NAME' — rebuild needed" >&2
    exit 0
}

# Convert to epoch seconds
IMAGE_EPOCH=$(date -d "$IMAGE_CREATED" +%s 2>/dev/null) || \
IMAGE_EPOCH=$(python3 -c "from datetime import datetime; print(int(datetime.fromisoformat('$IMAGE_CREATED'.replace('Z','+00:00')).timestamp()))" 2>/dev/null) || {
    echo "Cannot parse image timestamp — rebuild needed" >&2
    exit 0
}

# --- Compare ---
if [[ "$LATEST_SOURCE_EPOCH" -gt "$IMAGE_EPOCH" ]]; then
    LATEST_DATE=$(date -d "@$LATEST_SOURCE_EPOCH" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date -r "$LATEST_SOURCE_EPOCH" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "unknown")
    IMAGE_DATE=$(date -d "@$IMAGE_EPOCH" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date -r "$IMAGE_EPOCH" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "unknown")
    echo "Rebuild needed: source ($LATEST_DATE) is newer than image ($IMAGE_DATE)" >&2
    echo "  Changed: $LATEST_FILE" >&2
    exit 0
else
    IMAGE_DATE=$(date -d "@$IMAGE_EPOCH" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date -r "$IMAGE_EPOCH" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "unknown")
    echo "No rebuild needed: image ($IMAGE_DATE) is up to date" >&2
    exit 1
fi

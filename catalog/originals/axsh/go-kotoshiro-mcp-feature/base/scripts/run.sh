#!/bin/bash
# Run a command inside the function container. Usage:
#   ./scripts/run.sh              -> interactive bash
#   ./scripts/run.sh {command}    -> run command in container
# Scripts in scripts/ can be passed without path (resolved to ./scripts/ in container):
#   ./scripts/run.sh build.sh     or   ./scripts/run.sh ./scripts/build.sh
#   ./scripts/run.sh test.sh      or   ./scripts/run.sh ./scripts/test.sh
#   ./scripts/run.sh ./bin/function.exe

set -e
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
# shellcheck source=scripts/_common.sh
. "$SCRIPT_DIR/_common.sh"

# If the first argument has no path (e.g. "build.sh") and the file exists in scripts/,
# prepend "./scripts/" so the container finds it (workdir is function root).
if [ $# -ge 1 ] && [[ "$1" != */* ]] && [ -f "$FUNCTION_ROOT/scripts/$1" ]; then
  set -- "./scripts/$1" "${@:2}"
fi

run_args=(
  --rm
  -v "$FUNCTION_ROOT:/workspace/$FUNCTION_NAME"
  -v "gomodcache_${FUNCTION_NAME}:/go/pkg/mod"
  -e GOMODCACHE=/go/pkg/mod
  -w "/workspace/$FUNCTION_NAME"
  "$IMAGE_DEV"
)

if [ $# -eq 0 ]; then
  exec docker run -it "${run_args[@]}" bash
fi

exec docker run "${run_args[@]}" "$@"

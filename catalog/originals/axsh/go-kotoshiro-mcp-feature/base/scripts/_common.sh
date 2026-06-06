# Shared setup for scripts in scripts/. Source this from init.sh, run.sh, deploy.sh.
# Sets: SCRIPT_DIR, FUNCTION_ROOT, FUNCTION_NAME, IMAGE_DEV, IMAGE_PROD
# Mount: FUNCTION_ROOT -> /workspace/$FUNCTION_NAME so the container always sees the project
# under a path that includes the canonical name (e.g. /workspace/new-function).
# Prevent MSYS/Git Bash from converting paths like /workspace/root to Windows paths.
export MSYS_NO_PATHCONV=1

_SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SCRIPT_DIR="$_SCRIPT_DIR"
# Resolve function root: directory containing go.mod and container/
FUNCTION_ROOT="$SCRIPT_DIR"
while [ ! -f "$FUNCTION_ROOT/go.mod" ] || [ ! -d "$FUNCTION_ROOT/container" ]; do
  PARENT=$(dirname "$FUNCTION_ROOT")
  [ "$PARENT" = "$FUNCTION_ROOT" ] && { echo "Error: Cannot find function root (directory with go.mod and container/)." >&2; exit 1; }
  FUNCTION_ROOT="$PARENT"
done
FUNCTION_NAME=$(basename "$FUNCTION_ROOT")
# Prefer canonical name when the project is run from a hash-named copy (e.g. by ProcessManager).
# Order: KUNIUMI_FUNCTION_NAME env > registry.json "name" > path basename.
if [ -n "$KUNIUMI_FUNCTION_NAME" ]; then
  FUNCTION_NAME="$KUNIUMI_FUNCTION_NAME"
elif [ -f "$FUNCTION_ROOT/registry.json" ]; then
  _name=$(sed -n 's/.*"name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$FUNCTION_ROOT/registry.json")
  [ -n "$_name" ] && FUNCTION_NAME="$_name"
fi
# Docker tags must be lowercase
_fn_lower=$(echo "$FUNCTION_NAME" | tr '[:upper:]' '[:lower:]')
IMAGE_DEV="${_fn_lower}_kuniumi_devbox:latest"
IMAGE_PROD="${_fn_lower}_kuniumi:latest"

# Log function and image names (to stderr so stdout stays clean)
echo "Function: $FUNCTION_NAME  (mounted at /workspace/$FUNCTION_NAME)" >&2
echo "Images:   dev=$IMAGE_DEV  prod=$IMAGE_PROD" >&2

# Message channel setup: if KOTOSHIRO_MESSAGE_PORT is set by ProcessManager,
# open FD 3 to the TCP port so scripts can send messages via: echo "data" >&3
if [ -n "$KOTOSHIRO_MESSAGE_PORT" ]; then
  _MSG_HOST="127.0.0.1"
  # In WSL 2, 127.0.0.1 refers to the WSL VM's loopback, not the Windows host.
  # Detect WSL and resolve the Windows host IP from the default gateway.
  if grep -qi microsoft /proc/version 2>/dev/null; then
    _wsl_host=$(ip route show default 2>/dev/null | awk '{print $3}')
    [ -n "$_wsl_host" ] && _MSG_HOST="$_wsl_host"
  fi
  exec 3>/dev/tcp/$_MSG_HOST/$KOTOSHIRO_MESSAGE_PORT
fi

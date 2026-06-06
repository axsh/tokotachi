#!/usr/bin/env bash
# Common tool discovery logic for agentctl wrapper scripts.
# Source this file, then use $TOOL variable.

_resolve_agentctl() {
    if [ -n "${AGENTCTL:-}" ]; then
        echo "$AGENTCTL"
        return 0
    fi
    if command -v agentctl &>/dev/null; then
        echo "agentctl"
        return 0
    fi
    # Check project-local bin/
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local project_root
    project_root="$(cd "$script_dir/../.." && pwd)"
    local local_bin="$project_root/bin/agentctl"
    if [ -x "$local_bin" ]; then
        echo "$local_bin"
        return 0
    fi
    echo "Skipping coding agent settings update: update tool not found." >&2
    echo "Set AGENTCTL env var, add agentctl to PATH, or place it in bin/" >&2
    return 1
}

TOOL="$(_resolve_agentctl)" || exit 1

#!/usr/bin/env bash
# Common library for distribution scripts
# Usage: source "$(dirname "${BASH_SOURCE[0]}")/_lib.sh"

# ─── Resolve REPO_ROOT ───────────────────────────────────────────────
# BASH_SOURCE[1] is the caller script; BASH_SOURCE[0] is this file.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[1]:-${BASH_SOURCE[0]}}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# ─── Colored output ─────────────────────────────────────────────────
info()  { echo -e "\033[0;34m[INFO]\033[0m $*"; }
pass()  { echo -e "\033[0;32m[PASS]\033[0m $*"; }
fail()  { echo -e "\033[1;31m[FAIL]\033[0m $*"; }
warn()  { echo -e "\033[1;33m[WARN]\033[0m $*"; }

# ─── YAML helpers ───────────────────────────────────────────────────
# yaml_get <file> <python-dict-expression>
# Example: yaml_get "tools/manifests/tt.yaml" "['binary_name']"
yaml_get() {
  local file="$1" expr="$2"
  python - "$file" "$expr" <<'PYEOF'
import yaml, sys
f, expr = sys.argv[1], sys.argv[2]
d = yaml.safe_load(open(f))
print(eval("d" + expr))
PYEOF
}

# ─── Platform detection ─────────────────────────────────────────────
detect_os() {
  case "$(uname -s)" in
    Linux*)            echo "linux" ;;
    Darwin*)           echo "darwin" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *)                 echo "unknown" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64)        echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *)             echo "unknown" ;;
  esac
}

# ─── Manifest helpers ───────────────────────────────────────────────
# manifest_path <tool-id>
manifest_path() {
  echo "${REPO_ROOT}/tools/manifests/${1}.yaml"
}

# get_field <tool-id> <python-dict-expression>
# Example: get_field "tt" "['binary_name']"
get_field() {
  yaml_get "$(manifest_path "$1")" "$2"
}

# get_platforms <tool-id>
# Outputs lines of "os arch" pairs, e.g. "linux amd64"
get_platforms() {
  local file
  file="$(manifest_path "$1")"
  python - "$file" <<'PYEOF'
import yaml, sys
d = yaml.safe_load(open(sys.argv[1]))
for p in d.get('platforms', []):
    print(p['os'], p['arch'])
PYEOF
}

# get_all_tool_ids
# Reads tools/manifests/tools.yaml and prints each tool id
get_all_tool_ids() {
  local file="${REPO_ROOT}/tools/manifests/tools.yaml"
  python - "$file" <<'PYEOF'
import yaml, sys
d = yaml.safe_load(open(sys.argv[1]))
for t in d.get('tools', []):
    print(t['id'])
PYEOF
}

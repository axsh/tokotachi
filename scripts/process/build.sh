#!/bin/bash
set -euo pipefail

# ============================================================
# build.sh — Full Build & Unit Test Runner
#
# Builds the entire project and runs unit tests.
# Integration tests (under tests/) are excluded;
# use integration_test.sh for those.
#
# Usage:
#   ./scripts/process/build.sh [OPTIONS]
#
# Options:
#   --backend-only   Run only the Go backend build & tests
#   --help           Show this help message
#
# Exit Codes:
#   0 = All builds and tests passed
#   1 = Build or test failure
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# --- Helpers ---
info()    { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[PASS]${NC} $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
fail()    { echo -e "${RED}[FAIL]${NC} $*"; }
step()    { echo -e "${CYAN}${BOLD}===> $*${NC}"; }

show_help() {
    cat << 'EOF'
Usage: ./scripts/process/build.sh [OPTIONS]

Builds the entire project and runs unit tests.
Integration tests (under tests/) are excluded.

Options:
  --backend-only   Run only the Go backend build & unit tests
  --help           Show this help message

Exit Codes:
  0 = All builds and tests passed
  1 = Build or test failure

Examples:
  # Full build (all components)
  ./scripts/process/build.sh

  # Backend (Go) only
  ./scripts/process/build.sh --backend-only
EOF
}

# --- Argument Parsing ---
BACKEND_ONLY=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --backend-only)
            BACKEND_ONLY=true
            shift
            ;;
        --help|-h)
            show_help
            exit 0
            ;;
        *)
            fail "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# --- Track overall result ---
FAILED=false

# ============================================================
# Backend (Go) Build & Unit Test
# ============================================================
build_backend() {
    step "Backend (Go): Build & Unit Test"

    cd "$PROJECT_ROOT"

    # Check if go.mod exists
    if [[ ! -f "go.mod" ]]; then
        warn "go.mod not found — skipping Go backend build."
        warn "Initialize the module with: go mod init <module-name>"
        return 0
    fi

    # --- Build ---
    info "Building Go packages..."
    if go build ./...; then
        success "Go build succeeded."
    else
        fail "Go build failed."
        FAILED=true
        return 1
    fi

    # --- Unit Tests ---
    # Exclude tests/ directory (integration tests) and vendor/
    info "Running Go unit tests (excluding tests/ directory)..."

    # Discover all packages except tests/ subtree
    UNIT_PKGS=$(go list ./... | grep -v '/tests/' | grep -v '/tests$' || true)

    if [[ -z "$UNIT_PKGS" ]]; then
        warn "No Go unit test packages found."
        return 0
    fi

    if echo "$UNIT_PKGS" | xargs go test -v -count=1; then
        success "All Go unit tests passed."
    else
        fail "Go unit tests failed."
        FAILED=true
        return 1
    fi
}

# ============================================================
# Frontend Build (placeholder for future use)
# ============================================================
build_frontend() {
    step "Frontend: Build & Test"

    # Check for frontend project indicators
    local frontend_found=false

    # Look for package.json in known frontend locations
    for dir in "$PROJECT_ROOT/frontend" "$PROJECT_ROOT/webview" "$PROJECT_ROOT/ide/webview"; do
        if [[ -f "$dir/package.json" ]]; then
            frontend_found=true
            info "Found frontend project at: $dir"

            cd "$dir"

            info "Installing dependencies..."
            npm ci --silent 2>/dev/null || npm install --silent

            info "Building frontend..."
            if npm run build; then
                success "Frontend build succeeded."
            else
                fail "Frontend build failed."
                FAILED=true
                return 1
            fi

            if npm test --if-present 2>/dev/null; then
                success "Frontend tests passed."
            else
                fail "Frontend tests failed."
                FAILED=true
                return 1
            fi

            cd "$PROJECT_ROOT"
        fi
    done

    if [[ "$frontend_found" == "false" ]]; then
        warn "No frontend project found — skipping."
    fi
}

# ============================================================
# devctl Build & Unit Test
# ============================================================
build_devctl() {
    step "devctl (Go): Build & Unit Test"

    local devctl_dir="$PROJECT_ROOT/features/devctl"

    if [[ ! -f "$devctl_dir/go.mod" ]]; then
        warn "features/devctl/go.mod not found — skipping devctl build."
        return 0
    fi

    cd "$devctl_dir"

    # --- Build ---
    info "Building devctl..."
    if go build -o "$PROJECT_ROOT/bin/devctl" .; then
        success "devctl build succeeded."
    else
        fail "devctl build failed."
        FAILED=true
        cd "$PROJECT_ROOT"
        return 1
    fi

    # --- Unit Tests ---
    info "Running devctl unit tests..."
    if go test -v -count=1 ./...; then
        success "All devctl unit tests passed."
    else
        fail "devctl unit tests failed."
        FAILED=true
        cd "$PROJECT_ROOT"
        return 1
    fi

    cd "$PROJECT_ROOT"
}

# ============================================================
# Main
# ============================================================
main() {
    echo ""
    echo -e "${BOLD}╔══════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}║     Build & Unit Test Pipeline           ║${NC}"
    echo -e "${BOLD}╚══════════════════════════════════════════╝${NC}"
    echo ""

    local start_time=$SECONDS

    # Always run backend
    build_backend

    # Build devctl
    build_devctl

    # Run frontend unless --backend-only
    if [[ "$BACKEND_ONLY" == "false" ]]; then
        build_frontend
    fi

    local elapsed=$(( SECONDS - start_time ))
    echo ""
    echo -e "${BOLD}─────────────────────────────────────────────${NC}"

    if [[ "$FAILED" == "true" ]]; then
        fail "Build pipeline FAILED (${elapsed}s)"
        echo -e "${RED}Fix the errors above before running integration tests.${NC}"
        exit 1
    else
        success "Build pipeline PASSED (${elapsed}s)"
        echo -e "${GREEN}Ready for integration tests: ./scripts/process/integration_test.sh${NC}"
        exit 0
    fi
}

main

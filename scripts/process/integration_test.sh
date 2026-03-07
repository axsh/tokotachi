#!/bin/bash
set -euo pipefail

# ============================================================
# integration_test.sh — Integration & E2E Test Runner
#
# Runs integration tests located under the tests/ directory.
# Supports both Go (*_test.go) and Python (test_*.py) tests.
# Supports category filtering, specific test execution,
# and resuming from the last failure point.
#
# Usage:
#   ./scripts/process/integration_test.sh [OPTIONS]
#
# Options:
#   --categories <c1,c2>    Run tests only for specified categories
#   --specify <Filter>      Run only tests matching the filter
#                            (Go: passed to 'go test -run',
#                             Python: passed to 'pytest -k')
#   --resume                Resume from the last failed test
#   --help                  Show this help message
#
# Exit Codes:
#   0 = All tests passed
#   1 = One or more tests failed
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TMP_DIR="$PROJECT_ROOT/tmp"
LAST_FAILED_FILE="$TMP_DIR/.last_failed_tests"
LAST_RESULTS_FILE="$TMP_DIR/.last_test_results"

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
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
Usage: ./scripts/process/integration_test.sh [OPTIONS]

Runs integration tests in the tests/ directory.
Each subdirectory under tests/ is treated as a "category".
Supports both Go (*_test.go) and Python (test_*.py / pytest) tests.

Options:
  --categories <c1,c2>    Comma-separated list of test categories to run
                           (subdirectory names under tests/)
  --specify <Filter>      Run only tests matching the given filter
                           (Go: passed to 'go test -run',
                            Python: passed to 'pytest -k')
  --resume                Resume from the last failed test category
  --help                  Show this help message

Exit Codes:
  0 = All tests passed
  1 = One or more tests failed

Examples:
  # Run all integration tests
  ./scripts/process/integration_test.sh

  # Run only 'llm' and 'taskengine' categories
  ./scripts/process/integration_test.sh --categories "llm,taskengine"

  # Run a specific test by name
  ./scripts/process/integration_test.sh --specify "TestNewFeature"

  # Combine category and test name filter
  ./scripts/process/integration_test.sh --categories "llm" --specify "TestGeminiAPI"

  # Run Python integration tests
  ./scripts/process/integration_test.sh --categories "integration-test"

  # Resume from last failure point
  ./scripts/process/integration_test.sh --resume
EOF
}

# --- Argument Parsing ---
CATEGORIES=""
SPECIFY=""
RESUME=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --categories)
            if [[ -z "${2:-}" ]]; then
                fail "--categories requires a value"
                exit 1
            fi
            CATEGORIES="$2"
            shift 2
            ;;
        --specify)
            if [[ -z "${2:-}" ]]; then
                fail "--specify requires a value"
                exit 1
            fi
            SPECIFY="$2"
            shift 2
            ;;
        --resume)
            RESUME=true
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

# ============================================================
# Functions
# ============================================================

# Ensure tmp directory exists
ensure_tmp() {
    mkdir -p "$TMP_DIR"
}

# Detect the test language for a category directory
# Returns "go", "python", or "none"
detect_category_lang() {
    local test_dir="$1"

    # Go takes priority if both exist
    if ls "$test_dir"/*_test.go 1>/dev/null 2>&1; then
        echo "go"
        return
    fi

    if ls "$test_dir"/test_*.py 1>/dev/null 2>&1 || \
       ls "$test_dir"/conftest.py 1>/dev/null 2>&1; then
        echo "python"
        return
    fi

    echo "none"
}

# Discover test categories (subdirectories under tests/)
discover_categories() {
    local tests_dir="$PROJECT_ROOT/tests"

    if [[ ! -d "$tests_dir" ]]; then
        return
    fi

    # Find subdirectories that contain Go or Python test files
    for dir in "$tests_dir"/*/; do
        if [[ -d "$dir" ]]; then
            local cat_name
            cat_name=$(basename "$dir")
            local lang
            lang=$(detect_category_lang "$dir")
            if [[ "$lang" != "none" ]]; then
                echo "$cat_name"
            fi
        fi
    done
}

# Get categories to run based on options
get_target_categories() {
    if [[ "$RESUME" == "true" ]]; then
        if [[ ! -f "$LAST_FAILED_FILE" ]]; then
            warn "No previous failure record found. Running all categories."
            discover_categories
            return
        fi

        # Read the failed category and return it plus all subsequent ones
        local failed_cat
        failed_cat=$(head -1 "$LAST_FAILED_FILE" | tr -d '\n\r')
        info "Resuming from failed category: $failed_cat"

        local all_cats
        all_cats=$(discover_categories)
        local found=false

        while IFS= read -r cat; do
            if [[ "$cat" == "$failed_cat" ]]; then
                found=true
            fi
            if [[ "$found" == "true" ]]; then
                echo "$cat"
            fi
        done <<< "$all_cats"

        if [[ "$found" == "false" ]]; then
            warn "Previously failed category '$failed_cat' not found. Running all."
            discover_categories
        fi
        return
    fi

    if [[ -n "$CATEGORIES" ]]; then
        # Split comma-separated categories
        echo "$CATEGORIES" | tr ',' '\n' | tr -d ' '
    else
        # All categories
        discover_categories
    fi
}

# Run Go tests for a single category
run_go_tests() {
    local category="$1"
    local test_dir="$2"

    step "Running Go integration tests: $category"

    # Build the go test command
    local go_test_args=("-v" "-count=1")

    # Add -run filter if --specify was given
    if [[ -n "$SPECIFY" ]]; then
        go_test_args+=("-run" "$SPECIFY")
    fi

    # Check if test dir has its own go.mod (independent module)
    if [[ -f "$test_dir/go.mod" ]]; then
        # Run go test from the test directory
        cd "$test_dir"
        if go test "${go_test_args[@]}" ./...; then
            success "Category '$category' (Go) — all tests passed."
            cd "$PROJECT_ROOT"
            return 0
        else
            fail "Category '$category' (Go) — tests FAILED."
            cd "$PROJECT_ROOT"
            return 1
        fi
    else
        # Add the test package path (resolve from project root)
        local pkg_path
        pkg_path=$(cd "$test_dir" && go list . 2>/dev/null || echo "./tests/$category")
        go_test_args+=("$pkg_path")

        # Execute from project root
        cd "$PROJECT_ROOT"
        if go test "${go_test_args[@]}"; then
            success "Category '$category' (Go) — all tests passed."
            return 0
        else
            fail "Category '$category' (Go) — tests FAILED."
            return 1
        fi
    fi
}

# Run Python tests for a single category
run_python_tests() {
    local category="$1"
    local test_dir="$2"

    step "Running Python integration tests: $category"

    # Check if pytest is available
    if ! python -m pytest --version 1>/dev/null 2>&1 && \
       ! python3 -m pytest --version 1>/dev/null 2>&1; then
        fail "pytest is not installed. Install with: pip install pytest"
        return 1
    fi

    # Determine python command
    local python_cmd="python"
    if ! command -v python 1>/dev/null 2>&1; then
        python_cmd="python3"
    fi

    local pytest_args=("-v" "--tb=short")

    # Add -k filter if --specify was given
    if [[ -n "$SPECIFY" ]]; then
        pytest_args+=("-k" "$SPECIFY")
    fi

    pytest_args+=("$test_dir")

    # Execute
    cd "$PROJECT_ROOT"
    if $python_cmd -m pytest "${pytest_args[@]}"; then
        success "Category '$category' (Python) — all tests passed."
        return 0
    else
        fail "Category '$category' (Python) — tests FAILED."
        return 1
    fi
}

# Run tests for a single category (dispatches to Go or Python)
run_category_tests() {
    local category="$1"
    local test_dir="$PROJECT_ROOT/tests/$category"

    if [[ ! -d "$test_dir" ]]; then
        warn "Category directory not found: tests/$category — skipping"
        return 0
    fi

    local lang
    lang=$(detect_category_lang "$test_dir")

    case "$lang" in
        go)
            run_go_tests "$category" "$test_dir"
            ;;
        python)
            run_python_tests "$category" "$test_dir"
            ;;
        *)
            warn "No test files in tests/$category — skipping"
            return 0
            ;;
    esac
}

# Save failure information for --resume
save_failure_info() {
    local failed_category="$1"
    ensure_tmp
    echo "$failed_category" > "$LAST_FAILED_FILE"
    info "Failure recorded in $LAST_FAILED_FILE (use --resume to continue)"
}

# Clear failure information on full success
clear_failure_info() {
    rm -f "$LAST_FAILED_FILE"
}

# ============================================================
# Main
# ============================================================
main() {
    echo ""
    echo -e "${BOLD}╔══════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}║     Integration Test Pipeline            ║${NC}"
    echo -e "${BOLD}╚══════════════════════════════════════════╝${NC}"
    echo ""

    cd "$PROJECT_ROOT"

    if [[ ! -d "tests" ]]; then
        warn "tests/ directory not found — no integration tests to run."
        exit 0
    fi

    # Check settings/test/config.yaml
    if [[ -f "settings/test/config.yaml" ]]; then
        info "Test configuration found: settings/test/config.yaml"
    else
        info "Test configuration not found: settings/test/config.yaml (optional)"
    fi

    ensure_tmp

    local start_time=$SECONDS

    # Collect target categories
    local categories
    categories=$(get_target_categories)

    if [[ -z "$categories" ]]; then
        warn "No test categories found under tests/."
        exit 0
    fi

    # Display plan
    info "Categories to run:"
    while IFS= read -r cat; do
        local lang
        lang=$(detect_category_lang "$PROJECT_ROOT/tests/$cat")
        echo -e "  ${MAGENTA}•${NC} $cat ${BLUE}[$lang]${NC}"
    done <<< "$categories"
    if [[ -n "$SPECIFY" ]]; then
        info "Test filter: $SPECIFY"
    fi
    echo ""

    # Run tests category by category
    local total=0
    local passed=0
    local failed_count=0
    local failed_cats=()

    while IFS= read -r category; do
        [[ -z "$category" ]] && continue
        total=$((total + 1))

        if run_category_tests "$category"; then
            passed=$((passed + 1))
        else
            failed_count=$((failed_count + 1))
            failed_cats+=("$category")
            # Save failure point for --resume
            save_failure_info "$category"
        fi
        echo ""
    done <<< "$categories"

    # --- Results Summary ---
    local elapsed=$(( SECONDS - start_time ))

    echo -e "${BOLD}─────────────────────────────────────────────${NC}"
    echo -e "${BOLD}Integration Test Summary${NC}"
    echo -e "  Total categories:  $total"
    echo -e "  ${GREEN}Passed:${NC}            $passed"
    echo -e "  ${RED}Failed:${NC}            $failed_count"
    echo -e "  Elapsed:           ${elapsed}s"

    if [[ ${#failed_cats[@]} -gt 0 ]]; then
        echo ""
        fail "Failed categories:"
        for cat in "${failed_cats[@]}"; do
            echo -e "  ${RED}✗${NC} $cat"
        done
        echo ""
        echo -e "${YELLOW}To re-run a failed test:${NC}"
        echo "  ./scripts/process/integration_test.sh --categories \"${failed_cats[0]}\" --specify \"TestName\""
        echo ""
        echo -e "${YELLOW}To resume from failure point:${NC}"
        echo "  ./scripts/process/integration_test.sh --resume"
        exit 1
    else
        clear_failure_info
        success "All integration tests passed! (${elapsed}s)"
        exit 0
    fi
}

main

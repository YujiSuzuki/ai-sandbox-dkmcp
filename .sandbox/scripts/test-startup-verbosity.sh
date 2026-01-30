#!/bin/bash
# test-startup-verbosity.sh
# Test startup verbosity options
# 起動時詳細度オプションのテスト

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE="${WORKSPACE:-/workspace}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Counters
TESTS_PASSED=0
TESTS_FAILED=0

# Test helpers
pass() { echo -e "${GREEN}PASS${NC}: $1"; ((TESTS_PASSED++)) || true; }
fail() { echo -e "${RED}FAIL${NC}: $1"; ((TESTS_FAILED++)) || true; }
info() { echo -e "${YELLOW}INFO${NC}: $1"; }

# ============================================================
# Test: _startup_common.sh functions
# ============================================================
test_startup_common() {
    echo ""
    echo "=== Testing _startup_common.sh ==="

    # Source the common functions
    # shellcheck source=/dev/null
    source "$WORKSPACE/.sandbox/scripts/_startup_common.sh"

    # Test 1: load_startup_config loads verbose as default
    unset STARTUP_VERBOSITY
    load_startup_config
    if [ "$STARTUP_VERBOSITY" = "verbose" ]; then
        pass "load_startup_config sets verbose as default verbosity"
    else
        fail "load_startup_config should set STARTUP_VERBOSITY to 'verbose', got '$STARTUP_VERBOSITY'"
    fi

    # Test 2: is_quiet/is_verbose/is_summary functions
    STARTUP_VERBOSITY="quiet"
    if is_quiet; then
        pass "is_quiet returns true when STARTUP_VERBOSITY=quiet"
    else
        fail "is_quiet should return true when STARTUP_VERBOSITY=quiet"
    fi

    STARTUP_VERBOSITY="verbose"
    if is_verbose; then
        pass "is_verbose returns true when STARTUP_VERBOSITY=verbose"
    else
        fail "is_verbose should return true when STARTUP_VERBOSITY=verbose"
    fi

    STARTUP_VERBOSITY="summary"
    if is_summary; then
        pass "is_summary returns true when STARTUP_VERBOSITY=summary"
    else
        fail "is_summary should return true when STARTUP_VERBOSITY=summary"
    fi

    # Test 2b: is_summary returns false for other values
    STARTUP_VERBOSITY="verbose"
    if is_summary; then
        fail "is_summary should return false when STARTUP_VERBOSITY=verbose"
    else
        pass "is_summary returns false when STARTUP_VERBOSITY=verbose"
    fi

    # Test 3: get_readme_url returns correct URL based on locale
    unset LANG LC_ALL
    local url
    url=$(get_readme_url)
    if [ "$url" = "README.md" ]; then
        pass "get_readme_url returns English README for non-Japanese locale"
    else
        fail "get_readme_url should return 'README.md', got '$url'"
    fi

    LANG="ja_JP.UTF-8"
    url=$(get_readme_url)
    if [ "$url" = "README.ja.md" ]; then
        pass "get_readme_url returns Japanese README for ja_JP locale"
    else
        fail "get_readme_url should return 'README.ja.md', got '$url'"
    fi
    unset LANG

    # Test 4: Environment variable override
    SANDBOX_README_URL="CUSTOM.md"
    load_startup_config
    if [ "$README_URL" = "CUSTOM.md" ]; then
        pass "SANDBOX_README_URL environment variable overrides config"
    else
        fail "SANDBOX_README_URL should override, got '$README_URL'"
    fi
    unset SANDBOX_README_URL
}

# ============================================================
# Test: Verbosity output behavior
# ============================================================
test_verbosity_output() {
    echo ""
    echo "=== Testing verbosity output behavior ==="

    # Source the common functions (without debug trace)
    # shellcheck source=/dev/null
    set +x 2>/dev/null || true
    source "$WORKSPACE/.sandbox/scripts/_startup_common.sh"

    # Test quiet mode output
    STARTUP_VERBOSITY="quiet"
    local output
    output=$(print_default "test message")
    if [ -z "$output" ]; then
        pass "print_default produces no output in quiet mode"
    else
        fail "print_default should produce no output in quiet mode, got '$output'"
    fi

    # Test summary mode output
    STARTUP_VERBOSITY="summary"
    output=$(print_default "test message")
    if [ "$output" = "test message" ]; then
        pass "print_default produces output in summary mode"
    else
        fail "print_default should produce 'test message' in summary mode, got '$output'"
    fi

    # Test verbose mode output (print_default)
    STARTUP_VERBOSITY="verbose"
    output=$(print_default "test message")
    if [ "$output" = "test message" ]; then
        pass "print_default produces output in verbose mode"
    else
        fail "print_default should produce 'test message' in verbose mode, got '$output'"
    fi

    # Test verbose mode output (print_detail)
    STARTUP_VERBOSITY="verbose"
    output=$(print_detail "test detail")
    if [ "$output" = "test detail" ]; then
        pass "print_detail produces output in verbose mode"
    else
        fail "print_detail should produce 'test detail' in verbose mode, got '$output'"
    fi

    # Test summary mode doesn't show details
    STARTUP_VERBOSITY="summary"
    output=$(print_detail "test detail")
    if [ -z "$output" ]; then
        pass "print_detail produces no output in summary mode"
    else
        fail "print_detail should produce no output in summary mode, got '$output'"
    fi
}

# ============================================================
# Test: Config file loading
# ============================================================
test_config_file() {
    echo ""
    echo "=== Testing config file loading ==="

    local config_file="$WORKSPACE/.sandbox/config/startup.conf"

    if [ -f "$config_file" ]; then
        pass "startup.conf exists"

        # Check that it contains expected variables
        if grep -q "README_URL" "$config_file"; then
            pass "startup.conf contains README_URL"
        else
            fail "startup.conf should contain README_URL"
        fi

        if grep -q "STARTUP_VERBOSITY" "$config_file"; then
            pass "startup.conf contains STARTUP_VERBOSITY"
        else
            fail "startup.conf should contain STARTUP_VERBOSITY"
        fi
    else
        fail "startup.conf does not exist"
    fi
}

# ============================================================
# Main
# ============================================================
main() {
    echo "========================================"
    echo "Startup Verbosity Tests"
    echo "========================================"

    test_startup_common
    test_verbosity_output
    test_config_file

    echo ""
    echo "========================================"
    echo "Test Results"
    echo "========================================"
    echo -e "Passed: ${GREEN}${TESTS_PASSED}${NC}"
    echo -e "Failed: ${RED}${TESTS_FAILED}${NC}"
    echo ""

    if [ $TESTS_FAILED -gt 0 ]; then
        exit 1
    fi
    exit 0
}

main "$@"

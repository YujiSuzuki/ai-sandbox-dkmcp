#!/bin/bash
# test-show-sponsor.sh
# Test show-sponsor.sh behavior
# show-sponsor.sh の動作テスト

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

# Create a temp workspace to avoid modifying real config
TEMP_WORKSPACE=""
setup_temp_workspace() {
    TEMP_WORKSPACE=$(mktemp -d)
    mkdir -p "$TEMP_WORKSPACE/.sandbox/config"
    mkdir -p "$TEMP_WORKSPACE/.sandbox/scripts"

    # Copy necessary files
    cp "$WORKSPACE/.sandbox/scripts/_startup_common.sh" "$TEMP_WORKSPACE/.sandbox/scripts/"
    cp "$WORKSPACE/.sandbox/scripts/show-sponsor.sh" "$TEMP_WORKSPACE/.sandbox/scripts/"
    cp "$WORKSPACE/.sandbox/config/startup.conf" "$TEMP_WORKSPACE/.sandbox/config/"
}

cleanup_temp_workspace() {
    [ -n "$TEMP_WORKSPACE" ] && rm -rf "$TEMP_WORKSPACE"
}

# ============================================================
# Test: Default display shows sponsor URL
# ============================================================
test_default_display() {
    echo ""
    echo "=== Testing default display ==="

    local output
    output=$(WORKSPACE="$TEMP_WORKSPACE" LANG=en_US.UTF-8 LC_ALL="" \
        STARTUP_VERBOSITY=verbose \
        bash "$TEMP_WORKSPACE/.sandbox/scripts/show-sponsor.sh" 2>&1)

    if echo "$output" | grep -q "https://github.com/sponsors/YujiSuzuki"; then
        pass "Default display contains sponsor URL"
    else
        fail "Default display should contain sponsor URL, got: $output"
    fi

    if echo "$output" | grep -q "Support this project"; then
        pass "Default display shows English title"
    else
        fail "Default display should show English title, got: $output"
    fi

    if echo "$output" | grep -q "consider sponsoring"; then
        pass "Default display shows English body"
    else
        fail "Default display should show English body, got: $output"
    fi

    if echo "$output" | grep -q "\-\-no-thanks"; then
        pass "Default display shows how to hide"
    else
        fail "Default display should show --no-thanks hint, got: $output"
    fi
}

# ============================================================
# Test: Quiet mode shows one-liner
# ============================================================
test_quiet_mode() {
    echo ""
    echo "=== Testing quiet mode ==="

    local output
    output=$(WORKSPACE="$TEMP_WORKSPACE" LANG=en_US.UTF-8 LC_ALL="" \
        STARTUP_VERBOSITY=quiet \
        bash "$TEMP_WORKSPACE/.sandbox/scripts/show-sponsor.sh" 2>&1)

    if echo "$output" | grep -q "Sponsor:"; then
        pass "Quiet mode shows one-liner with Sponsor:"
    else
        fail "Quiet mode should show one-liner, got: $output"
    fi

    # Should NOT contain the full title/separator
    if echo "$output" | grep -q "━━━"; then
        fail "Quiet mode should not show separators"
    else
        pass "Quiet mode does not show separators"
    fi
}

# ============================================================
# Test: Japanese locale
# ============================================================
test_japanese_locale() {
    echo ""
    echo "=== Testing Japanese locale ==="

    local output
    output=$(WORKSPACE="$TEMP_WORKSPACE" LANG=ja_JP.UTF-8 LC_ALL="" \
        STARTUP_VERBOSITY=verbose \
        bash "$TEMP_WORKSPACE/.sandbox/scripts/show-sponsor.sh" 2>&1)

    if echo "$output" | grep -q "このプロジェクトを応援"; then
        pass "Japanese locale shows Japanese title"
    else
        fail "Japanese locale should show Japanese title, got: $output"
    fi

    if echo "$output" | grep -q "スポンサーになって応援"; then
        pass "Japanese locale shows Japanese body"
    else
        fail "Japanese locale should show Japanese body, got: $output"
    fi
}

# ============================================================
# Test: --no-thanks shows disable instructions
# ============================================================
test_no_thanks() {
    echo ""
    echo "=== Testing --no-thanks flag ==="

    local output
    output=$(WORKSPACE="$TEMP_WORKSPACE" LANG=en_US.UTF-8 LC_ALL="" \
        bash "$TEMP_WORKSPACE/.sandbox/scripts/show-sponsor.sh" --no-thanks 2>&1)

    if echo "$output" | grep -q "To disable the sponsor message"; then
        pass "--no-thanks shows disable instructions"
    else
        fail "--no-thanks should show disable instructions, got: $output"
    fi

    if echo "$output" | grep -q "\-\-no-sponsor"; then
        pass "--no-thanks mentions --no-sponsor flag"
    else
        fail "--no-thanks should mention --no-sponsor flag, got: $output"
    fi

    if echo "$output" | grep -q "devcontainer.json"; then
        pass "--no-thanks mentions devcontainer.json"
    else
        fail "--no-thanks should mention devcontainer.json, got: $output"
    fi

    if echo "$output" | grep -q "cli_sandbox"; then
        pass "--no-thanks mentions CLI sandbox"
    else
        fail "--no-thanks should mention CLI sandbox, got: $output"
    fi
}

# ============================================================
# Test: --no-thanks with Japanese locale
# ============================================================
test_no_thanks_japanese() {
    echo ""
    echo "=== Testing --no-thanks with Japanese locale ==="

    local output
    output=$(WORKSPACE="$TEMP_WORKSPACE" LANG=ja_JP.UTF-8 LC_ALL="" \
        bash "$TEMP_WORKSPACE/.sandbox/scripts/show-sponsor.sh" --no-thanks 2>&1)

    if echo "$output" | grep -q "スポンサーメッセージを無効にするには"; then
        pass "--no-thanks shows Japanese instructions"
    else
        fail "--no-thanks should show Japanese instructions, got: $output"
    fi

    if echo "$output" | grep -q "\-\-no-sponsor"; then
        pass "--no-thanks (JA) mentions --no-sponsor flag"
    else
        fail "--no-thanks (JA) should mention --no-sponsor flag, got: $output"
    fi
}

# ============================================================
# Main
# ============================================================
main() {
    echo "========================================"
    echo "Show Sponsor Tests"
    echo "========================================"

    setup_temp_workspace
    trap cleanup_temp_workspace EXIT

    test_default_display
    test_quiet_mode
    test_japanese_locale
    test_no_thanks
    test_no_thanks_japanese

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

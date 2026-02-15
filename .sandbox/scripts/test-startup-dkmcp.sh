#!/bin/bash
# test-startup-dkmcp.sh
# Test DockMCP auto-registration logic in startup.sh
#
# Tests the step 7 behavior:
#   - Registered + connected: one-liner summary
#   - Registered but offline: one-liner warning
#   - Not registered: full registration output
#
# Uses stub scripts to isolate from real startup dependencies.
#
# Usage: ./test-startup-dkmcp.sh
#
# Environment: AI Sandbox (requires /workspace)
# ---
# startup.sh の DockMCP 自動登録ロジックのテスト
#
# ステップ7の動作をテスト:
#   - 登録済み＋接続OK: 1行サマリー
#   - 登録済みだがオフライン: 1行警告
#   - 未登録: フル登録出力
#
# スタブスクリプトで実際の起動処理から分離してテスト。
#
# 使用方法: ./test-startup-dkmcp.sh
# 実行環境: AI Sandbox（/workspace が必要）

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
STARTUP_SCRIPT="$SCRIPT_DIR/startup.sh"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

# Counters
TESTS_PASSED=0
TESTS_FAILED=0

pass() { echo -e "${GREEN}✅ $1${NC}"; TESTS_PASSED=$((TESTS_PASSED + 1)); }
fail() { echo -e "${RED}❌ $1${NC}"; TESTS_FAILED=$((TESTS_FAILED + 1)); }

# ─── Setup / Cleanup ────────────────────────────────────────

TEST_DIR=""

setup() {
    TEST_DIR=$(mktemp -d)

    # Create stub scripts directory mirroring .sandbox/scripts/
    mkdir -p "$TEST_DIR/workspace/.sandbox/scripts"
    mkdir -p "$TEST_DIR/workspace/.sandbox/sandbox-mcp"

    # Create no-op stubs for steps 1-5 (not under test)
    for script in merge-claude-settings.sh compare-secret-config.sh \
                  validate-secrets.sh check-secret-sync.sh check-upstream-updates.sh; do
        cat > "$TEST_DIR/workspace/.sandbox/scripts/$script" << 'STUB'
#!/bin/bash
exit 0
STUB
        chmod +x "$TEST_DIR/workspace/.sandbox/scripts/$script"
    done

    # Create no-op stub for _startup_common.sh
    cat > "$TEST_DIR/workspace/.sandbox/scripts/_startup_common.sh" << 'STUB'
#!/bin/bash
# Stub: no-op common functions
STUB

    # Create no-op Makefile for sandbox-mcp register (step 6)
    cat > "$TEST_DIR/workspace/.sandbox/sandbox-mcp/Makefile" << 'STUB'
.PHONY: register
register:
	@true
STUB

    # Copy actual startup.sh and rewrite paths to use test directory
    sed "s|/workspace|$TEST_DIR/workspace|g" "$STARTUP_SCRIPT" \
        > "$TEST_DIR/workspace/.sandbox/scripts/startup.sh"
    chmod +x "$TEST_DIR/workspace/.sandbox/scripts/startup.sh"
}

cleanup() {
    if [ -n "$TEST_DIR" ] && [ -d "$TEST_DIR" ]; then
        rm -rf "$TEST_DIR"
    fi
    TEST_DIR=""
}

trap cleanup EXIT

# Helper: create setup-dkmcp.sh stub with specified exit codes
# --check 時と通常実行時の exit code をそれぞれ指定
create_dkmcp_stub() {
    local check_exit="$1"     # exit code for --check
    local register_exit="${2:-0}"  # exit code for default mode (register)
    local register_output="${3:-DockMCP full registration output}"  # output for default mode

    cat > "$TEST_DIR/workspace/.sandbox/scripts/setup-dkmcp.sh" << STUB
#!/bin/bash
for arg in "\$@"; do
    if [ "\$arg" = "--check" ]; then
        exit $check_exit
    fi
done
echo "$register_output"
exit $register_exit
STUB
    chmod +x "$TEST_DIR/workspace/.sandbox/scripts/setup-dkmcp.sh"
}

# ─── Tests ──────────────────────────────────────────────────

# Test 1: When --check returns 0 (registered + connected), shows one-liner with "connected"
test_oneliner_when_registered_and_connected() {
    echo ""
    echo "=== Test: One-liner when DockMCP is registered and connected ==="

    setup
    create_dkmcp_stub 0

    local output
    output=$(bash "$TEST_DIR/workspace/.sandbox/scripts/startup.sh" 2>&1)

    # Should show one-liner with DockMCP and connected
    if echo "$output" | grep -q "DockMCP.*connected\|DockMCP.*接続OK"; then
        pass "Shows one-liner with connected status"
    else
        fail "Should show one-liner with connected status"
    fi

    # Should NOT show full registration output
    if echo "$output" | grep -q "DockMCP full registration output"; then
        fail "Should not show full registration output when already registered"
    else
        pass "Does not show full registration output"
    fi

    cleanup
}

# Test 2: When --check returns 2 (registered but offline), shows one-liner warning
test_oneliner_when_registered_but_offline() {
    echo ""
    echo "=== Test: One-liner warning when DockMCP is registered but offline ==="

    setup
    create_dkmcp_stub 2

    local output
    output=$(bash "$TEST_DIR/workspace/.sandbox/scripts/startup.sh" 2>&1)

    # Should show one-liner with not reachable / offline warning
    if echo "$output" | grep -q "DockMCP.*not reachable\|DockMCP.*接続不可"; then
        pass "Shows one-liner with offline warning"
    else
        fail "Should show one-liner with offline warning"
    fi

    # Should NOT show full registration output
    if echo "$output" | grep -q "DockMCP full registration output"; then
        fail "Should not show full registration output when already registered"
    else
        pass "Does not show full registration output"
    fi

    cleanup
}

# Test 3: When --check returns 1 (not registered), runs full registration
test_full_output_when_not_registered() {
    echo ""
    echo "=== Test: Full registration when DockMCP is not registered ==="

    setup
    create_dkmcp_stub 1 0 "DockMCP full registration output"

    local output
    output=$(bash "$TEST_DIR/workspace/.sandbox/scripts/startup.sh" 2>&1)

    if echo "$output" | grep -q "DockMCP full registration output"; then
        pass "Runs full registration and shows output when not registered"
    else
        fail "Should show full registration output, but it was missing"
    fi

    cleanup
}

# Test 4: When registration fails, shows error message and continues
test_registration_failure_continues() {
    echo ""
    echo "=== Test: Registration failure shows error and continues ==="

    setup
    create_dkmcp_stub 1 1 "some error"

    local output
    local exit_code=0
    output=$(bash "$TEST_DIR/workspace/.sandbox/scripts/startup.sh" 2>&1) || exit_code=$?

    # Should contain the failure message
    if echo "$output" | grep -qi "DockMCP.*failed\|DockMCP.*失敗"; then
        pass "Registration failure shows error message"
    else
        fail "Registration failure did not show error message"
    fi

    # Should still complete (not crash)
    if echo "$output" | grep -qi "complete\|完了"; then
        pass "Startup completes even after DockMCP registration failure"
    else
        fail "Startup did not complete after DockMCP registration failure"
    fi

    cleanup
}

# Test 5: Startup completes successfully with DockMCP step
test_startup_completes_with_dkmcp() {
    echo ""
    echo "=== Test: Startup completes with DockMCP step ==="

    setup
    create_dkmcp_stub 0

    local exit_code=0
    bash "$TEST_DIR/workspace/.sandbox/scripts/startup.sh" >/dev/null 2>&1 || exit_code=$?

    if [ "$exit_code" -eq 0 ]; then
        pass "Startup completes with exit 0"
    else
        fail "Startup exited with $exit_code, expected 0"
    fi

    cleanup
}

# ─── Main ───────────────────────────────────────────────────

main() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  startup.sh DockMCP Auto-Registration Tests"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    test_oneliner_when_registered_and_connected
    test_oneliner_when_registered_but_offline
    test_full_output_when_not_registered
    test_registration_failure_continues
    test_startup_completes_with_dkmcp

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  Results: $TESTS_PASSED passed, $TESTS_FAILED failed"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    if [ "$TESTS_FAILED" -gt 0 ]; then
        exit 1
    fi
}

main "$@"

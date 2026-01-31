#!/bin/bash
# test-confirm-continue.sh
# Test script for confirm_continue_after_failure function in _common.sh
#
# _common.sh の confirm_continue_after_failure 関数のテストスクリプト
#
# Usage: ./test-confirm-continue.sh
# 使用方法: ./test-confirm-continue.sh
#
# Environment: AI Sandbox (requires /workspace)
# 実行環境: AI Sandbox（/workspace が必要）

set -e

# Verify running in AI Sandbox
# AI Sandbox 内での実行を確認
if [ ! -d "/workspace" ]; then
    echo "Error: This test is designed to run inside AI Sandbox"
    echo "エラー: このテストは AI Sandbox 内での実行を想定しています"
    exit 1
fi

WORKSPACE_DIR="/workspace"

# Colors for output
# 出力用の色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color / 色なし

# Test counter
# テストカウンター
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
# ヘルパー関数
pass() {
    echo -e "${GREEN}✅ $1${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

fail() {
    echo -e "${RED}❌ $1${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

# Source _common.sh (requires running from workspace root with required variables)
# _common.sh を読み込む（ワークスペースルートから実行し、必須変数が必要）
source_common() {
    SCRIPT_NAME="test-confirm-continue.sh"
    COMPOSE_PROJECT_NAME="cli-test"
    SANDBOX_ENV="cli_test"
    cd "$WORKSPACE_DIR"
    source "$WORKSPACE_DIR/cli_sandbox/_common.sh"
}

# Test 1: User enters 'y' → should return 0 (continue)
# テスト1: ユーザーが 'y' を入力 → 0 を返す（続行）
test_continue_with_y() {
    echo ""
    echo "=== Test: Continue with 'y' ==="

    if echo "y" | confirm_continue_after_failure > /dev/null 2>&1; then
        pass "Returns 0 when user enters 'y'"
    else
        fail "Should return 0 when user enters 'y'"
    fi
}

# Test 2: User enters 'Y' → should return 0 (continue)
# テスト2: ユーザーが 'Y' を入力 → 0 を返す（続行）
test_continue_with_uppercase_y() {
    echo ""
    echo "=== Test: Continue with 'Y' ==="

    if echo "Y" | confirm_continue_after_failure > /dev/null 2>&1; then
        pass "Returns 0 when user enters 'Y'"
    else
        fail "Should return 0 when user enters 'Y'"
    fi
}

# Test 3: User enters 'yes' → should return 0 (continue)
# テスト3: ユーザーが 'yes' を入力 → 0 を返す（続行）
test_continue_with_yes() {
    echo ""
    echo "=== Test: Continue with 'yes' ==="

    if echo "yes" | confirm_continue_after_failure > /dev/null 2>&1; then
        pass "Returns 0 when user enters 'yes'"
    else
        fail "Should return 0 when user enters 'yes'"
    fi
}

# Test 4: User enters 'n' → should return 1 (exit)
# テスト4: ユーザーが 'n' を入力 → 1 を返す（終了）
test_exit_with_n() {
    echo ""
    echo "=== Test: Exit with 'n' ==="

    if echo "n" | confirm_continue_after_failure > /dev/null 2>&1; then
        fail "Should return 1 when user enters 'n'"
    else
        pass "Returns 1 when user enters 'n'"
    fi
}

# Test 5: User presses Enter (empty input) → should return 1 (exit, default)
# テスト5: ユーザーが Enter を押す（空入力）→ 1 を返す（終了、デフォルト）
test_exit_with_empty() {
    echo ""
    echo "=== Test: Exit with empty input (default) ==="

    if echo "" | confirm_continue_after_failure > /dev/null 2>&1; then
        fail "Should return 1 when user presses Enter (default is N)"
    else
        pass "Returns 1 when user presses Enter (default is N)"
    fi
}

# Test 6: User enters random text → should return 1 (exit)
# テスト6: ユーザーがランダムなテキストを入力 → 1 を返す（終了）
test_exit_with_random() {
    echo ""
    echo "=== Test: Exit with random input ==="

    if echo "maybe" | confirm_continue_after_failure > /dev/null 2>&1; then
        fail "Should return 1 when user enters unrecognized input"
    else
        pass "Returns 1 when user enters unrecognized input"
    fi
}

# Test 7: Warning message is displayed when continuing
# テスト7: 続行時に警告メッセージが表示されるか
test_warning_message_on_continue() {
    echo ""
    echo "=== Test: Warning message displayed on continue ==="

    local output
    output=$(echo "y" | confirm_continue_after_failure 2>&1)

    if echo "$output" | grep -q "Entering shell for investigation\|調査用のシェルに入ります"; then
        pass "Warning message displayed"
    else
        fail "Warning message should be displayed"
    fi
}

# Test 8: Validation failed message is displayed
# テスト8: バリデーション失敗メッセージが表示されるか
test_failure_message_displayed() {
    echo ""
    echo "=== Test: Validation failure message displayed ==="

    local output
    output=$(echo "n" | confirm_continue_after_failure 2>&1 || true)

    if echo "$output" | grep -q "Startup validation failed\|起動検証に失敗しました"; then
        pass "Validation failure message displayed"
    else
        fail "Validation failure message should be displayed"
    fi
}

# Test 9: Messages are NOT mixed (only one language)
# テスト9: メッセージが混在しないこと（1言語のみ）
test_no_mixed_languages() {
    echo ""
    echo "=== Test: Messages are not mixed (single language) ==="

    local output
    output=$(echo "y" | confirm_continue_after_failure 2>&1)

    local has_en has_ja
    has_en=false
    has_ja=false
    echo "$output" | grep -q "Entering shell for investigation" && has_en=true
    echo "$output" | grep -q "調査用のシェルに入ります" && has_ja=true

    if [ "$has_en" = true ] && [ "$has_ja" = true ]; then
        fail "Messages should not be in both languages"
    elif [ "$has_en" = true ] || [ "$has_ja" = true ]; then
        pass "Messages are in a single language only"
    else
        fail "No expected message found in any language"
    fi
}

# Test 10: Japanese locale shows Japanese messages
# テスト10: 日本語ロケールで日本語メッセージが表示されるか
test_japanese_locale() {
    echo ""
    echo "=== Test: Japanese locale shows Japanese messages ==="

    local output
    output=$(bash -c '
        SCRIPT_NAME="test"
        COMPOSE_PROJECT_NAME="cli-test"
        SANDBOX_ENV="cli_test"
        cd /workspace
        source /workspace/cli_sandbox/_common.sh
        export LANG=ja_JP.UTF-8 LC_ALL=ja_JP.UTF-8
        echo "y" | confirm_continue_after_failure
    ' 2>&1)

    if echo "$output" | grep -q "調査用のシェルに入ります"; then
        pass "Japanese locale shows Japanese messages"
    else
        fail "Japanese locale should show Japanese messages"
    fi
}

# Test 11: English locale shows English messages
# テスト11: 英語ロケールで英語メッセージが表示されるか
test_english_locale() {
    echo ""
    echo "=== Test: English locale shows English messages ==="

    local output
    output=$(bash -c '
        SCRIPT_NAME="test"
        COMPOSE_PROJECT_NAME="cli-test"
        SANDBOX_ENV="cli_test"
        cd /workspace
        source /workspace/cli_sandbox/_common.sh
        export LANG=en_US.UTF-8 LC_ALL=en_US.UTF-8
        echo "y" | confirm_continue_after_failure
    ' 2>&1)

    if echo "$output" | grep -q "Entering shell for investigation"; then
        pass "English locale shows English messages"
    else
        fail "English locale should show English messages"
    fi
}

# Run all tests
# 全テストを実行
main() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  confirm_continue_after_failure Test Suite"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    source_common

    test_continue_with_y
    test_continue_with_uppercase_y
    test_continue_with_yes
    test_exit_with_n
    test_exit_with_empty
    test_exit_with_random
    test_warning_message_on_continue
    test_failure_message_displayed
    test_no_mixed_languages
    test_japanese_locale
    test_english_locale

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

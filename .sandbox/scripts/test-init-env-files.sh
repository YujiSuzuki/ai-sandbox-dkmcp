#!/bin/bash
# test-init-env-files.sh
# Test script for init-env-files.sh
#
# init-env-files.sh のテストスクリプト
#
# Usage: ./test-init-env-files.sh
# 使用方法: ./test-init-env-files.sh
#
# Environment: DevContainer (requires /workspace)
# 実行環境: DevContainer（/workspace が必要）

set -e

# Verify running in DevContainer
# DevContainer 内での実行を確認
if [ ! -d "/workspace" ]; then
    echo "Error: This test is designed to run inside DevContainer"
    echo "エラー: このテストは DevContainer 内での実行を想定しています"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SCRIPT="$SCRIPT_DIR/init-env-files.sh"
TEST_PROJECT=""

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

# Setup test environment
# テスト環境のセットアップ
setup() {
    info "Setting up test environment..."
    TEST_PROJECT=$(mktemp -d)
}

# Cleanup test environment
# テスト環境のクリーンアップ
cleanup() {
    if [ -n "$TEST_PROJECT" ] && [ -d "$TEST_PROJECT" ]; then
        rm -rf "$TEST_PROJECT"
    fi
}

# Trap to ensure cleanup runs
# クリーンアップが必ず実行されるようトラップ設定
trap cleanup EXIT

# Test 1: Script is executable and has valid syntax
# テスト1: スクリプトが実行可能で構文エラーがないか
test_script_executable_and_valid() {
    echo ""
    echo "=== Test: Script is executable and has valid syntax ==="

    if [ ! -f "$SCRIPT" ]; then
        fail "Script not found: $SCRIPT"
        return
    fi

    if [ ! -x "$SCRIPT" ]; then
        fail "Script is not executable"
        return
    fi

    # Check for syntax errors
    # 構文エラーをチェック
    if bash -n "$SCRIPT" 2>/dev/null; then
        pass "Script is executable and has valid syntax"
    else
        fail "Script has syntax errors"
    fi
}

# Test 2: Creates .env.sandbox from .example
# テスト2: .example から .env.sandbox を作成
test_creates_env_sandbox_from_example() {
    echo ""
    echo "=== Test: Creates .env.sandbox from .example ==="

    setup

    # Create .env.sandbox.example
    # .env.sandbox.example を作成
    echo "TEST_VAR=test_value" > "$TEST_PROJECT/.env.sandbox.example"

    # Run script
    # スクリプトを実行
    bash "$SCRIPT" "$TEST_PROJECT" > /dev/null 2>&1

    if [ -f "$TEST_PROJECT/.env.sandbox" ]; then
        local content
        content=$(cat "$TEST_PROJECT/.env.sandbox")
        if [ "$content" = "TEST_VAR=test_value" ]; then
            pass "Creates .env.sandbox from .env.sandbox.example"
        else
            fail "Content mismatch in .env.sandbox"
        fi
    else
        fail ".env.sandbox was not created"
    fi

    cleanup
}

# Test 3: Creates empty .env.sandbox when no .example
# テスト3: .example がない場合に空の .env.sandbox を作成
test_creates_empty_env_sandbox() {
    echo ""
    echo "=== Test: Creates empty .env.sandbox when no .example ==="

    setup

    # Don't create .env.sandbox.example
    # .env.sandbox.example を作成しない

    bash "$SCRIPT" "$TEST_PROJECT" > /dev/null 2>&1

    if [ -f "$TEST_PROJECT/.env.sandbox" ]; then
        if [ ! -s "$TEST_PROJECT/.env.sandbox" ]; then
            pass "Creates empty .env.sandbox when no .example"
        else
            fail ".env.sandbox should be empty"
        fi
    else
        fail ".env.sandbox was not created"
    fi

    cleanup
}

# Test 4: Creates cli_sandbox/.env from .example
# テスト4: .example から cli_sandbox/.env を作成
test_creates_cli_env_from_example() {
    echo ""
    echo "=== Test: Creates cli_sandbox/.env from .example ==="

    setup
    mkdir -p "$TEST_PROJECT/cli_sandbox"

    # Create .env.example
    echo "CLI_VAR=cli_value" > "$TEST_PROJECT/cli_sandbox/.env.example"

    bash "$SCRIPT" "$TEST_PROJECT" > /dev/null 2>&1

    if [ -f "$TEST_PROJECT/cli_sandbox/.env" ]; then
        local content
        content=$(cat "$TEST_PROJECT/cli_sandbox/.env")
        if [ "$content" = "CLI_VAR=cli_value" ]; then
            pass "Creates cli_sandbox/.env from .env.example"
        else
            fail "Content mismatch in cli_sandbox/.env"
        fi
    else
        fail "cli_sandbox/.env was not created"
    fi

    cleanup
}

# Test 5: Creates empty cli_sandbox/.env when no .example
# テスト5: .example がない場合に空の cli_sandbox/.env を作成
test_creates_empty_cli_env() {
    echo ""
    echo "=== Test: Creates empty cli_sandbox/.env when no .example ==="

    setup
    mkdir -p "$TEST_PROJECT/cli_sandbox"

    # Don't create .env.example

    bash "$SCRIPT" "$TEST_PROJECT" > /dev/null 2>&1

    if [ -f "$TEST_PROJECT/cli_sandbox/.env" ]; then
        if [ ! -s "$TEST_PROJECT/cli_sandbox/.env" ]; then
            pass "Creates empty cli_sandbox/.env when no .example"
        else
            fail "cli_sandbox/.env should be empty"
        fi
    else
        fail "cli_sandbox/.env was not created"
    fi

    cleanup
}

# Test 6: Skips when .env.sandbox already exists
# テスト6: .env.sandbox が既に存在する場合はスキップ
test_skips_existing_env_sandbox() {
    echo ""
    echo "=== Test: Skips when .env.sandbox already exists ==="

    setup

    # Create existing .env.sandbox with specific content
    # 特定の内容で既存の .env.sandbox を作成
    echo "EXISTING_VALUE=keep_this" > "$TEST_PROJECT/.env.sandbox"

    # Create different .env.sandbox.example
    # 異なる内容の .env.sandbox.example を作成
    echo "NEW_VALUE=should_not_replace" > "$TEST_PROJECT/.env.sandbox.example"

    bash "$SCRIPT" "$TEST_PROJECT" > /dev/null 2>&1

    local content
    content=$(cat "$TEST_PROJECT/.env.sandbox")
    if [ "$content" = "EXISTING_VALUE=keep_this" ]; then
        pass "Skips existing .env.sandbox (preserves content)"
    else
        fail "Should not overwrite existing .env.sandbox"
    fi

    cleanup
}

# Test 7: Skips when cli_sandbox/.env already exists
# テスト7: cli_sandbox/.env が既に存在する場合はスキップ
test_skips_existing_cli_env() {
    echo ""
    echo "=== Test: Skips when cli_sandbox/.env already exists ==="

    setup
    mkdir -p "$TEST_PROJECT/cli_sandbox"

    # Create existing .env with specific content
    echo "EXISTING_CLI=keep_this" > "$TEST_PROJECT/cli_sandbox/.env"

    # Create different .env.example
    echo "NEW_CLI=should_not_replace" > "$TEST_PROJECT/cli_sandbox/.env.example"

    bash "$SCRIPT" "$TEST_PROJECT" > /dev/null 2>&1

    local content
    content=$(cat "$TEST_PROJECT/cli_sandbox/.env")
    if [ "$content" = "EXISTING_CLI=keep_this" ]; then
        pass "Skips existing cli_sandbox/.env (preserves content)"
    else
        fail "Should not overwrite existing cli_sandbox/.env"
    fi

    cleanup
}

# Test 8: Skips cli_sandbox when directory doesn't exist
# テスト8: cli_sandbox ディレクトリがない場合はスキップ
test_skips_missing_cli_sandbox_dir() {
    echo ""
    echo "=== Test: Skips cli_sandbox when directory doesn't exist ==="

    setup

    # Don't create cli_sandbox directory

    bash "$SCRIPT" "$TEST_PROJECT" > /dev/null 2>&1

    if [ ! -d "$TEST_PROJECT/cli_sandbox" ]; then
        pass "Does not create cli_sandbox directory"
    else
        fail "Should not create cli_sandbox directory"
    fi

    cleanup
}

# Test 9: Output shows initialization message
# テスト9: 出力に初期化メッセージが含まれる
test_output_shows_initialization() {
    echo ""
    echo "=== Test: Output shows initialization message ==="

    setup

    local output
    output=$(bash "$SCRIPT" "$TEST_PROJECT" 2>&1)

    if echo "$output" | grep -q "Created\|作成"; then
        pass "Output shows initialization message"
    else
        fail "Output should show initialization message"
    fi

    cleanup
}

# Test 10: Uses current directory when no argument
# テスト10: 引数がない場合はカレントディレクトリを使用
test_uses_current_directory() {
    echo ""
    echo "=== Test: Uses current directory when no argument ==="

    setup
    cd "$TEST_PROJECT"

    # Run without argument
    bash "$SCRIPT" > /dev/null 2>&1

    if [ -f "$TEST_PROJECT/.env.sandbox" ]; then
        pass "Uses current directory when no argument"
    else
        fail "Should use current directory when no argument"
    fi

    cd - > /dev/null
    cleanup
}

# Test 11: --help option shows usage
# テスト11: --help オプションで使用方法を表示
test_help_option() {
    echo ""
    echo "=== Test: --help option shows usage ==="

    local output
    output=$(bash "$SCRIPT" --help 2>&1)

    if echo "$output" | grep -q "interactive" && echo "$output" | grep -q "対話モード"; then
        pass "--help shows usage information"
    else
        fail "--help should show usage with interactive mode info"
    fi
}

# Test 12: Interactive mode with Japanese selection (new file)
# テスト12: 対話モードで日本語選択（新規ファイル）
test_interactive_japanese_new_file() {
    echo ""
    echo "=== Test: Interactive mode with Japanese selection (new file) ==="

    setup

    # Create .env.sandbox.example with LANG setting
    cat > "$TEST_PROJECT/.env.sandbox.example" << 'EOF'
NODE_ENV=development
LANG=C.UTF-8
EOF

    # Run with interactive mode, select Japanese (2)
    echo "2" | bash "$SCRIPT" -i "$TEST_PROJECT" > /dev/null 2>&1

    if [ -f "$TEST_PROJECT/.env.sandbox" ]; then
        if grep -q "^LANG=ja_JP.UTF-8" "$TEST_PROJECT/.env.sandbox"; then
            pass "Interactive mode sets Japanese language on new file"
        else
            fail "LANG should be ja_JP.UTF-8"
        fi
    else
        fail ".env.sandbox was not created"
    fi

    cleanup
}

# Test 13: Interactive mode with English selection (new file)
# テスト13: 対話モードで英語選択（新規ファイル）
test_interactive_english_new_file() {
    echo ""
    echo "=== Test: Interactive mode with English selection (new file) ==="

    setup

    cat > "$TEST_PROJECT/.env.sandbox.example" << 'EOF'
NODE_ENV=development
LANG=ja_JP.UTF-8
EOF

    # Run with interactive mode, select English (1)
    echo "1" | bash "$SCRIPT" -i "$TEST_PROJECT" > /dev/null 2>&1

    if [ -f "$TEST_PROJECT/.env.sandbox" ]; then
        if grep -q "^LANG=C.UTF-8" "$TEST_PROJECT/.env.sandbox"; then
            pass "Interactive mode sets English language on new file"
        else
            fail "LANG should be C.UTF-8"
        fi
    else
        fail ".env.sandbox was not created"
    fi

    cleanup
}

# Test 14: Interactive mode updates existing file when confirmed
# テスト14: 対話モードで既存ファイルを確認後に更新
test_interactive_update_existing() {
    echo ""
    echo "=== Test: Interactive mode updates existing file when confirmed ==="

    setup

    # Create existing .env.sandbox with English
    echo "LANG=C.UTF-8" > "$TEST_PROJECT/.env.sandbox"

    # Run with interactive mode, select Japanese (2), confirm update (y)
    echo -e "2\ny" | bash "$SCRIPT" -i "$TEST_PROJECT" > /dev/null 2>&1

    if grep -q "^LANG=ja_JP.UTF-8" "$TEST_PROJECT/.env.sandbox"; then
        pass "Interactive mode updates language when confirmed"
    else
        fail "LANG should be updated to ja_JP.UTF-8"
    fi

    cleanup
}

# Test 15: Interactive mode preserves existing file when declined
# テスト15: 対話モードで更新を拒否した場合は既存ファイルを保持
test_interactive_decline_update() {
    echo ""
    echo "=== Test: Interactive mode preserves existing file when declined ==="

    setup

    # Create existing .env.sandbox with English
    echo "LANG=C.UTF-8" > "$TEST_PROJECT/.env.sandbox"

    # Run with interactive mode, select Japanese (2), decline update (n)
    echo -e "2\nn" | bash "$SCRIPT" -i "$TEST_PROJECT" > /dev/null 2>&1

    if grep -q "^LANG=C.UTF-8" "$TEST_PROJECT/.env.sandbox"; then
        pass "Interactive mode preserves language when declined"
    else
        fail "LANG should remain C.UTF-8"
    fi

    cleanup
}

# Run all tests
# 全テストを実行
main() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  init-env-files.sh Test Suite"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    test_script_executable_and_valid
    test_creates_env_sandbox_from_example
    test_creates_empty_env_sandbox
    test_creates_cli_env_from_example
    test_creates_empty_cli_env
    test_skips_existing_env_sandbox
    test_skips_existing_cli_env
    test_skips_missing_cli_sandbox_dir
    test_output_shows_initialization
    test_uses_current_directory
    test_help_option
    test_interactive_japanese_new_file
    test_interactive_english_new_file
    test_interactive_update_existing
    test_interactive_decline_update

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

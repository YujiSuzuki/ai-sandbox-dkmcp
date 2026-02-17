#!/bin/bash
# test-install-commands.sh
# Test script for install-commands.sh
#
# Usage: ./test-install-commands.sh
#
# Environment: AI Sandbox (requires /workspace)
# ---
# install-commands.sh のテストスクリプト
#
# 使用方法: ./test-install-commands.sh
#
# 実行環境: AI Sandbox（/workspace が必要）

set -e

# Verify running in AI Sandbox
# AI Sandbox 内での実行を確認
if [ ! -d "/workspace" ]; then
    echo "Error: This test is designed to run inside AI Sandbox"
    echo "エラー: このテストは AI Sandbox 内での実行を想定しています"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SCRIPT="$SCRIPT_DIR/install-commands.sh"
TEST_DIR=""

# Colors for output
# 出力用の色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

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

# Setup test environment with fake workspace structure
# テスト環境のセットアップ（偽のワークスペース構造を作成）
setup() {
    TEST_DIR=$(mktemp -d)
    mkdir -p "$TEST_DIR/.sandbox/commands"
    mkdir -p "$TEST_DIR/.sandbox/scripts"

    # Copy the script under test into the fake workspace
    cp "$SCRIPT" "$TEST_DIR/.sandbox/scripts/install-commands.sh"
    chmod +x "$TEST_DIR/.sandbox/scripts/install-commands.sh"

    # Create sample command files
    cat > "$TEST_DIR/.sandbox/commands/test-cmd-a.md" << 'EOF'
---
description: Test command A for unit testing
---
# Test Command A
This is a test command.
EOF

    cat > "$TEST_DIR/.sandbox/commands/test-cmd-b.md" << 'EOF'
---
description: Test command B for unit testing
---
# Test Command B
This is another test command.
EOF
}

# Cleanup test environment
# テスト環境のクリーンアップ
cleanup() {
    if [ -n "$TEST_DIR" ] && [ -d "$TEST_DIR" ]; then
        rm -rf "$TEST_DIR"
    fi
    TEST_DIR=""
}

# Trap to ensure cleanup runs
# クリーンアップが必ず実行されるようトラップ設定
trap cleanup EXIT

# Helper to run script in test workspace
# テストワークスペース内でスクリプトを実行するヘルパー
run_script() {
    "$TEST_DIR/.sandbox/scripts/install-commands.sh" "$@"
}

# ─── Tests ────────────────────────────────────────────────────

# Test 1: --help exits 0 and shows expected options
# テスト1: --help が exit 0 で終了し、期待するオプションを表示するか
test_help() {
    echo ""
    echo "=== Test: --help exits 0 and shows expected options ==="

    setup

    local exit_code=0
    local output
    output=$(run_script --help 2>&1) || exit_code=$?

    if [ "$exit_code" -eq 0 ] && \
       echo "$output" | grep -q -- "--list" && \
       echo "$output" | grep -q -- "--all" && \
       echo "$output" | grep -q -- "--uninstall"; then
        pass "--help exits 0 and shows expected options"
    else
        fail "--help exited $exit_code or missing expected options"
    fi

    cleanup
}

# Test 2: --list shows available commands
# テスト2: --list が利用可能なコマンドを表示するか
test_list() {
    echo ""
    echo "=== Test: --list shows available commands ==="

    setup

    local output
    output=$(run_script --list 2>&1)

    if echo "$output" | grep -q "test-cmd-a" && \
       echo "$output" | grep -q "test-cmd-b"; then
        pass "--list shows all available commands"
    else
        fail "--list did not show expected commands"
    fi

    cleanup
}

# Test 3: Install a single command by name
# テスト3: 名前指定で単一コマンドをインストールできるか
test_install_single() {
    echo ""
    echo "=== Test: Install single command by name ==="

    setup

    run_script test-cmd-a > /dev/null 2>&1

    if [ -f "$TEST_DIR/.claude/commands/test-cmd-a.md" ]; then
        pass "Single command installed successfully"
    else
        fail "Command file not found after install"
    fi

    # Verify test-cmd-b was NOT installed
    if [ ! -f "$TEST_DIR/.claude/commands/test-cmd-b.md" ]; then
        pass "Other commands not installed (as expected)"
    else
        fail "Unexpected command was also installed"
    fi

    cleanup
}

# Test 4: --all installs all commands
# テスト4: --all で全コマンドがインストールされるか
test_install_all() {
    echo ""
    echo "=== Test: --all installs all commands ==="

    setup

    run_script --all > /dev/null 2>&1

    if [ -f "$TEST_DIR/.claude/commands/test-cmd-a.md" ] && \
       [ -f "$TEST_DIR/.claude/commands/test-cmd-b.md" ]; then
        pass "--all installs all commands"
    else
        fail "--all did not install all commands"
    fi

    cleanup
}

# Test 5: Installed file content matches localized source
# テスト5: インストールされたファイルの内容がローカライズ後のソースと一致するか
test_content_matches() {
    echo ""
    echo "=== Test: Installed file content matches localized source ==="

    setup

    # Create a command with description-ja to test localization stripping
    cat > "$TEST_DIR/.sandbox/commands/test-cmd-c.md" << 'EOF'
---
description: Test command C in English
description-ja: テストコマンドC（日本語）
---
# Test Command C
Content here.
EOF

    LANG=en_US.UTF-8 run_script test-cmd-c > /dev/null 2>&1

    local installed="$TEST_DIR/.claude/commands/test-cmd-c.md"
    # Installed file should have description-ja stripped
    if [ -f "$installed" ] && \
       grep -q "description: Test command C in English" "$installed" && \
       ! grep -q "description-ja:" "$installed"; then
        pass "Installed file content matches localized source (description-ja stripped)"
    else
        fail "Installed file does not match localized source"
        [ -f "$installed" ] && cat "$installed"
    fi

    cleanup
}

# Test 6: Re-install detects "already up to date"
# テスト6: 再インストールで「最新です」と検出するか
test_already_up_to_date() {
    echo ""
    echo "=== Test: Re-install detects already up to date ==="

    setup

    run_script test-cmd-a > /dev/null 2>&1 || true
    local output
    output=$(run_script test-cmd-a 2>&1) || true

    if echo "$output" | grep -qi "up to date\|最新"; then
        pass "Re-install shows already up to date"
    else
        fail "Re-install did not detect already up to date"
    fi

    cleanup
}

# Test 7: --uninstall removes installed commands
# テスト7: --uninstall がインストール済みコマンドを削除するか
test_uninstall() {
    echo ""
    echo "=== Test: --uninstall removes installed commands ==="

    setup

    run_script --all > /dev/null 2>&1

    # Verify installed
    if [ ! -f "$TEST_DIR/.claude/commands/test-cmd-a.md" ]; then
        fail "Setup: commands not installed"
        cleanup
        return
    fi

    run_script --uninstall > /dev/null 2>&1

    if [ ! -f "$TEST_DIR/.claude/commands/test-cmd-a.md" ] && \
       [ ! -f "$TEST_DIR/.claude/commands/test-cmd-b.md" ]; then
        pass "--uninstall removes all installed commands"
    else
        fail "--uninstall did not remove all commands"
    fi

    cleanup
}

# Test 8: --uninstall preserves non-sample commands
# テスト8: --uninstall がサンプル以外のコマンドを保持するか
test_uninstall_preserves_others() {
    echo ""
    echo "=== Test: --uninstall preserves non-sample commands ==="

    setup

    run_script --all > /dev/null 2>&1

    # Add a user-created command that's not in .sandbox/commands/
    echo "# User custom command" > "$TEST_DIR/.claude/commands/my-custom.md"

    run_script --uninstall > /dev/null 2>&1

    if [ -f "$TEST_DIR/.claude/commands/my-custom.md" ]; then
        pass "--uninstall preserves user-created commands"
    else
        fail "--uninstall removed user-created command"
    fi

    cleanup
}

# Test 9: Install nonexistent command returns error
# テスト9: 存在しないコマンドのインストールでエラーが返るか
test_install_nonexistent() {
    echo ""
    echo "=== Test: Install nonexistent command returns error ==="

    setup

    local exit_code=0
    local output
    output=$(run_script nonexistent-cmd 2>&1) || exit_code=$?

    if [ "$exit_code" -ne 0 ]; then
        pass "Nonexistent command returns error exit code"
    else
        fail "Nonexistent command did not return error"
    fi

    cleanup
}

# Test 10: --list shows [installed] status
# テスト10: --list がインストール済みステータスを表示するか
test_list_shows_installed_status() {
    echo ""
    echo "=== Test: --list shows installed status ==="

    setup

    run_script test-cmd-a > /dev/null 2>&1

    local output
    output=$(run_script --list 2>&1)

    if echo "$output" | grep "test-cmd-a" | grep -qi "installed\|インストール済"; then
        pass "--list shows installed status for installed commands"
    else
        fail "--list did not show installed status"
    fi

    cleanup
}

# Test 11: Update detection when source file changes
# テスト11: ソースファイル変更時に更新を検出するか
test_update_detection() {
    echo ""
    echo "=== Test: Update detection when source changes ==="

    setup

    run_script test-cmd-a > /dev/null 2>&1

    # Modify the installed file (simulate stale version)
    echo "# Modified" >> "$TEST_DIR/.claude/commands/test-cmd-a.md"

    local output
    output=$(run_script test-cmd-a 2>&1)

    if echo "$output" | grep -qi "updating\|更新"; then
        pass "Update detected when files differ"
    else
        fail "Update not detected when files differ"
    fi

    cleanup
}

# Test 12: .md extension is stripped from argument
# テスト12: 引数から .md 拡張子が除去されるか
test_md_extension_stripped() {
    echo ""
    echo "=== Test: .md extension stripped from argument ==="

    setup

    run_script test-cmd-a.md > /dev/null 2>&1

    if [ -f "$TEST_DIR/.claude/commands/test-cmd-a.md" ]; then
        pass ".md extension stripped correctly"
    else
        fail "Install with .md extension failed"
    fi

    cleanup
}

# Test 13: description-ja is used when LANG_JA=true
# テスト13: LANG_JA=true の場合 description-ja が使用されるか
test_localize_ja() {
    echo ""
    echo "=== Test: description-ja used when LANG=ja_JP ==="

    setup

    # Create a command with description-ja
    cat > "$TEST_DIR/.sandbox/commands/test-cmd-ja.md" << 'EOF'
---
description: Test command in English
description-ja: テストコマンド（日本語）
---
# Test Command JA
Content here.
EOF

    LANG=ja_JP.UTF-8 run_script test-cmd-ja > /dev/null 2>&1

    local installed="$TEST_DIR/.claude/commands/test-cmd-ja.md"
    if [ -f "$installed" ] && \
       grep -q "description: テストコマンド（日本語）" "$installed" && \
       ! grep -q "description-ja:" "$installed"; then
        pass "description-ja applied and description-ja line removed"
    else
        fail "description-ja not applied or description-ja line not removed"
        [ -f "$installed" ] && cat "$installed"
    fi

    cleanup
}

# Test 14: description stays English when LANG is not ja_JP
# テスト14: LANG が ja_JP でない場合 description が英語のままか
test_localize_en() {
    echo ""
    echo "=== Test: description stays English when LANG=en_US ==="

    setup

    cat > "$TEST_DIR/.sandbox/commands/test-cmd-ja.md" << 'EOF'
---
description: Test command in English
description-ja: テストコマンド（日本語）
---
# Test Command JA
Content here.
EOF

    LANG=en_US.UTF-8 run_script test-cmd-ja > /dev/null 2>&1

    local installed="$TEST_DIR/.claude/commands/test-cmd-ja.md"
    if [ -f "$installed" ] && \
       grep -q "description: Test command in English" "$installed" && \
       ! grep -q "description-ja:" "$installed"; then
        pass "description stays English and description-ja line removed"
    else
        fail "description was unexpectedly changed or description-ja line remains"
        [ -f "$installed" ] && cat "$installed"
    fi

    cleanup
}

# Test 15: --list shows Japanese description when LANG=ja_JP
# テスト15: LANG=ja_JP の場合 --list が日本語の description を表示するか
test_list_ja_description() {
    echo ""
    echo "=== Test: --list shows Japanese description when LANG=ja_JP ==="

    setup

    cat > "$TEST_DIR/.sandbox/commands/test-cmd-ja.md" << 'EOF'
---
description: Test command in English
description-ja: テストコマンド（日本語）
---
# Test Command JA
EOF

    local output
    output=$(LANG=ja_JP.UTF-8 run_script --list 2>&1)

    if echo "$output" | grep -q "テストコマンド（日本語）"; then
        pass "--list shows Japanese description"
    else
        fail "--list did not show Japanese description"
        echo "$output"
    fi

    cleanup
}

# Test 16: File without description-ja works normally
# テスト16: description-ja がないファイルが正常に動作するか
test_localize_no_ja_field() {
    echo ""
    echo "=== Test: File without description-ja works normally ==="

    setup

    # test-cmd-a has no description-ja
    LANG=ja_JP.UTF-8 run_script test-cmd-a > /dev/null 2>&1

    local installed="$TEST_DIR/.claude/commands/test-cmd-a.md"
    if [ -f "$installed" ] && \
       grep -q "description: Test command A for unit testing" "$installed"; then
        pass "File without description-ja installs with English description"
    else
        fail "File without description-ja failed to install correctly"
        [ -f "$installed" ] && cat "$installed"
    fi

    cleanup
}

# ─── Run all tests / 全テスト実行 ─────────────────────────────

main() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  install-commands.sh Test Suite"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    test_help
    test_list
    test_install_single
    test_install_all
    test_content_matches
    test_already_up_to_date
    test_uninstall
    test_uninstall_preserves_others
    test_install_nonexistent
    test_list_shows_installed_status
    test_update_detection
    test_md_extension_stripped
    test_localize_ja
    test_localize_en
    test_list_ja_description
    test_localize_no_ja_field

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

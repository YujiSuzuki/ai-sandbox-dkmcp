#!/bin/bash
# test-merge-claude-settings.sh
# Test script for merge-claude-settings.sh
#
# merge-claude-settings.sh のテストスクリプト
#
# Usage: ./test-merge-claude-settings.sh
# 使用方法: ./test-merge-claude-settings.sh
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

# Check if jq is available
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required for this test"
    echo "エラー: このテストには jq が必要です"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SCRIPT="$SCRIPT_DIR/merge-claude-settings.sh"
TEST_WORKSPACE=""
ORIGINAL_HOME="$HOME"

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

    # Create temporary workspace
    # 一時ワークスペースを作成
    TEST_WORKSPACE=$(mktemp -d /tmp/.test-merge-settings-XXXXXX)
    TEST_HOME=$(mktemp -d /tmp/.test-home-XXXXXX)

    mkdir -p "$TEST_WORKSPACE/.sandbox/scripts"
    mkdir -p "$TEST_WORKSPACE/.sandbox/config"

    # Copy required scripts and config to test workspace
    # 必要なスクリプトと設定をテストワークスペースにコピー
    cp "$SCRIPT_DIR/_startup_common.sh" "$TEST_WORKSPACE/.sandbox/scripts/"
    cp "$SCRIPT_DIR/../config/startup.conf" "$TEST_WORKSPACE/.sandbox/config/" 2>/dev/null || true
    cp "$SCRIPT_DIR/../config/sync-ignore" "$TEST_WORKSPACE/.sandbox/config/" 2>/dev/null || true

    export WORKSPACE_ROOT="$TEST_WORKSPACE"
    export HOME="$TEST_HOME"
    export STARTUP_VERBOSITY="verbose"
}

# Cleanup test environment
# テスト環境のクリーンアップ
cleanup() {
    info "Cleaning up test environment..."

    if [ -n "$TEST_WORKSPACE" ] && [ -d "$TEST_WORKSPACE" ]; then
        rm -rf "$TEST_WORKSPACE"
    fi

    if [ -n "$TEST_HOME" ] && [ -d "$TEST_HOME" ]; then
        rm -rf "$TEST_HOME"
    fi

    export HOME="$ORIGINAL_HOME"
    unset WORKSPACE_ROOT
}

# Trap to ensure cleanup on exit
# 終了時にクリーンアップを保証するトラップ
trap cleanup EXIT

# ========================================
# Test Cases / テストケース
# ========================================

# Test 1: No project settings to merge
# テスト1: マージするプロジェクト設定がない
test_no_project_settings() {
    info "Test 1: No project settings to merge"
    info "テスト1: マージするプロジェクト設定がない"

    setup

    # Run script
    output=$("$SCRIPT" 2>&1)

    if echo "$output" | grep -q "No project settings to merge\|マージするプロジェクト設定がありません"; then
        pass "Correctly detected no project settings"
    else
        fail "Should detect no project settings"
        echo "Output: $output"
    fi

    cleanup
}

# Test 2: Create settings by merging project permissions
# テスト2: プロジェクトの permissions をマージして設定を作成
test_create_by_merge() {
    info "Test 2: Create settings by merging project permissions"
    info "テスト2: プロジェクトの permissions をマージして設定を作成"

    setup

    # Create project with settings
    mkdir -p "$TEST_WORKSPACE/project-a/.claude"
    cat > "$TEST_WORKSPACE/project-a/.claude/settings.json" << 'EOF'
{
  "permissions": {
    "deny": ["Read(.env)", "Read(*.key)"]
  }
}
EOF

    # Run script
    output=$("$SCRIPT" 2>&1)

    # Check workspace settings created
    if [ -f "$TEST_WORKSPACE/.claude/settings.json" ]; then
        pass "Workspace settings.json created"
    else
        fail "Workspace settings.json not created"
    fi

    # Check backup created
    if [ -f "$TEST_HOME/.claude-settings-backup.json" ]; then
        pass "Backup created"
    else
        fail "Backup not created"
    fi

    # Check content
    if jq -e '.permissions.deny | length > 0' "$TEST_WORKSPACE/.claude/settings.json" > /dev/null 2>&1; then
        pass "Permissions merged correctly"
    else
        fail "Permissions not merged"
    fi

    cleanup
}

# Test 3: Re-merge when no manual changes
# テスト3: 手動変更がない場合は再マージ
test_remerge_no_changes() {
    info "Test 3: Re-merge when no manual changes"
    info "テスト3: 手動変更がない場合は再マージ"

    setup

    # Create project with settings
    mkdir -p "$TEST_WORKSPACE/project-a/.claude"
    cat > "$TEST_WORKSPACE/project-a/.claude/settings.json" << 'EOF'
{
  "permissions": {
    "deny": ["Read(.env)"]
  }
}
EOF

    # First run
    "$SCRIPT" > /dev/null 2>&1

    # Add another project
    mkdir -p "$TEST_WORKSPACE/project-b/.claude"
    cat > "$TEST_WORKSPACE/project-b/.claude/settings.json" << 'EOF'
{
  "permissions": {
    "deny": ["Read(*.key)"]
  }
}
EOF

    # Second run
    output=$("$SCRIPT" 2>&1)

    if echo "$output" | grep -q "Re-merged\|再マージ"; then
        pass "Re-merge detected"
    else
        fail "Re-merge not detected"
        echo "Output: $output"
    fi

    cleanup
}

# Test 4: Skip merge when manual changes detected
# テスト4: 手動変更が検出された場合はマージをスキップ
test_skip_on_manual_changes() {
    info "Test 4: Skip merge when manual changes detected"
    info "テスト4: 手動変更が検出された場合はマージをスキップ"

    setup

    # Create project with settings
    mkdir -p "$TEST_WORKSPACE/project-a/.claude"
    cat > "$TEST_WORKSPACE/project-a/.claude/settings.json" << 'EOF'
{
  "permissions": {
    "deny": ["Read(.env)"]
  }
}
EOF

    # First run
    "$SCRIPT" > /dev/null 2>&1

    # Manually modify workspace settings
    echo '{"permissions":{"deny":["Read(.env)","Read(manual.txt)"]}}' | jq '.' > "$TEST_WORKSPACE/.claude/settings.json"

    # Second run
    output=$("$SCRIPT" 2>&1)

    if echo "$output" | grep -q "Manual changes detected\|手動変更が検出されました"; then
        pass "Manual changes detected"
    else
        fail "Manual changes not detected"
        echo "Output: $output"
    fi

    # Check backup removed
    if [ ! -f "$TEST_HOME/.claude-settings-backup.json" ]; then
        pass "Backup removed after manual changes"
    else
        fail "Backup should be removed"
    fi

    cleanup
}

# Test 5: Skip merge when settings exist without backup
# テスト5: バックアップなしで設定が存在する場合はマージをスキップ
test_skip_without_backup() {
    info "Test 5: Skip merge when settings exist without backup"
    info "テスト5: バックアップなしで設定が存在する場合はマージをスキップ"

    setup

    # Create project with settings
    mkdir -p "$TEST_WORKSPACE/project-a/.claude"
    cat > "$TEST_WORKSPACE/project-a/.claude/settings.json" << 'EOF'
{
  "permissions": {
    "deny": ["Read(.env)"]
  }
}
EOF

    # Manually create workspace settings (no backup)
    mkdir -p "$TEST_WORKSPACE/.claude"
    echo '{"permissions":{}}' > "$TEST_WORKSPACE/.claude/settings.json"

    # Run script
    output=$("$SCRIPT" 2>&1)

    if echo "$output" | grep -q "without backup\|バックアップなしで存在"; then
        pass "Correctly skipped (no backup)"
    else
        fail "Should skip when no backup exists"
        echo "Output: $output"
    fi

    cleanup
}

# Test 6: Merge multiple projects
# テスト6: 複数プロジェクトのマージ
test_merge_multiple_projects() {
    info "Test 6: Merge multiple projects"
    info "テスト6: 複数プロジェクトのマージ"

    setup

    # Create multiple projects
    mkdir -p "$TEST_WORKSPACE/project-a/.claude"
    cat > "$TEST_WORKSPACE/project-a/.claude/settings.json" << 'EOF'
{
  "permissions": {
    "deny": ["Read(.env)"],
    "allow": ["Read(*.md)"]
  }
}
EOF

    mkdir -p "$TEST_WORKSPACE/project-b/.claude"
    cat > "$TEST_WORKSPACE/project-b/.claude/settings.json" << 'EOF'
{
  "permissions": {
    "deny": ["Read(*.key)"],
    "allow": ["Read(*.txt)"]
  }
}
EOF

    # Run script
    "$SCRIPT" > /dev/null 2>&1

    # Check merged content has both deny rules
    local deny_count
    deny_count=$(jq '.permissions.deny | length' "$TEST_WORKSPACE/.claude/settings.json" 2>/dev/null || echo "0")

    if [ "$deny_count" -ge 2 ]; then
        pass "Multiple projects merged"
    else
        fail "Multiple projects not merged correctly (deny count: $deny_count)"
        cat "$TEST_WORKSPACE/.claude/settings.json"
    fi

    cleanup
}

# ========================================
# Run all tests / 全テストの実行
# ========================================

echo ""
echo "=========================================="
echo "Testing merge-claude-settings.sh"
echo "merge-claude-settings.sh のテスト"
echo "=========================================="
echo ""

test_no_project_settings
test_create_by_merge
test_remerge_no_changes
test_skip_on_manual_changes
test_skip_without_backup
test_merge_multiple_projects

echo ""
echo "=========================================="
echo "Test Results / テスト結果"
echo "=========================================="
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed! / 全テスト成功！${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed. / 一部のテストが失敗しました。${NC}"
    exit 1
fi

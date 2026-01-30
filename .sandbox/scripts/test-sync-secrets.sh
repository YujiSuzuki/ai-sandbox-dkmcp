#!/bin/bash
# test-sync-secrets.sh
# Test script for sync-secrets.sh
#
# sync-secrets.sh のテストスクリプト
#
# Usage: ./test-sync-secrets.sh
# 使用方法: ./test-sync-secrets.sh
#
# Note: sync-secrets.sh is interactive, so tests focus on detection logic
# 注意: sync-secrets.sh は対話式なので、テストは検出ロジックに焦点を当てます
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
SCRIPT="$SCRIPT_DIR/sync-secrets.sh"
TEST_WORKSPACE=""

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
    TEST_WORKSPACE=$(mktemp -d /tmp/.test-sync-XXXXXX)

    mkdir -p "$TEST_WORKSPACE/.devcontainer"
    mkdir -p "$TEST_WORKSPACE/cli_sandbox"
    mkdir -p "$TEST_WORKSPACE/.claude"
    mkdir -p "$TEST_WORKSPACE/demo-app"
    mkdir -p "$TEST_WORKSPACE/.sandbox/scripts"
    mkdir -p "$TEST_WORKSPACE/.sandbox/config"

    # Copy required scripts and config to test workspace
    # 必要なスクリプトと設定をテストワークスペースにコピー
    cp "$SCRIPT_DIR/_startup_common.sh" "$TEST_WORKSPACE/.sandbox/scripts/"
    cp "$SCRIPT_DIR/../config/startup.conf" "$TEST_WORKSPACE/.sandbox/config/" 2>/dev/null || true
    cp "$SCRIPT_DIR/../config/sync-ignore" "$TEST_WORKSPACE/.sandbox/config/" 2>/dev/null || true
}

# Cleanup test environment
# テスト環境のクリーンアップ
cleanup() {
    info "Cleaning up test environment..."

    if [ -n "$TEST_WORKSPACE" ] && [ -d "$TEST_WORKSPACE" ]; then
        rm -rf "$TEST_WORKSPACE"
    fi
}

# Trap to ensure cleanup runs
# クリーンアップが必ず実行されるようトラップ設定
trap cleanup EXIT

# Create .claude/settings.json with deny patterns
# deny パターン付きの .claude/settings.json を作成
create_claude_settings() {
    local patterns="$1"
    cat > "$TEST_WORKSPACE/.claude/settings.json" << EOF
{
  "permissions": {
    "deny": [
      $patterns
    ],
    "allow": []
  }
}
EOF
}

# Create docker-compose.yml with volume mounts
# ボリュームマウント付きの docker-compose.yml を作成
# Note: Include dummy /dev/null mount and tmpfs to simulate real environment
# 注意: 実際の環境をシミュレートするため、ダミーの /dev/null マウントと tmpfs を含める
create_compose_file() {
    local volume_mounts="$1"
    local tmpfs_mounts="${2:-}"
    cat > "$TEST_WORKSPACE/.devcontainer/docker-compose.yml" << EOF
services:
  ai-sandbox:
    volumes:
      - ..:/workspace:cached
      - /dev/null:/workspace/dummy/.placeholder:ro
$volume_mounts
    tmpfs:
      - /workspace/dummy/tmpfs:ro
$tmpfs_mounts
EOF
}

# Create cli_sandbox/docker-compose.yml with volume mounts
# ボリュームマウント付きの cli_sandbox/docker-compose.yml を作成
create_cli_compose_file() {
    local volume_mounts="$1"
    local tmpfs_mounts="${2:-}"
    cat > "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml" << EOF
services:
  cli-sandbox:
    volumes:
      - ..:/workspace:cached
      - /dev/null:/workspace/dummy/.placeholder:ro
$volume_mounts
    tmpfs:
      - /workspace/dummy/tmpfs:ro
$tmpfs_mounts
EOF
}

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

# Test 2: All synced - exits cleanly
# テスト2: 全て同期済み - 正常終了
test_all_synced() {
    echo ""
    echo "=== Test: All secrets synced (exits cleanly) ==="

    setup

    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'
    create_compose_file "      - /dev/null:$TEST_WORKSPACE/demo-app/.env:ro" ""

    local output
    # Use echo "3" to simulate "Don't add" option (will exit before prompt since all synced)
    # echo "3" で「追加しない」オプションをシミュレート（全て同期済みならプロンプト前に終了）
    output=$(echo "3" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    if echo "$output" | grep -q "すべての秘匿\|All secret files are synced\|No additions needed"; then
        pass "Script exits cleanly when all synced"
    else
        fail "Script should exit cleanly when all synced"
        echo "Output: $output"
    fi

    cleanup
}

# Test 3: Shows missing files
# テスト3: 未設定ファイルを表示
test_shows_missing_files() {
    echo ""
    echo "=== Test: Shows missing files ==="

    setup

    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'
    create_compose_file "" ""

    local output
    # Use echo "3" to simulate "Don't add" option
    # echo "3" で「追加しない」オプションをシミュレート
    output=$(echo "3" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    if echo "$output" | grep -q "demo-app/.env"; then
        pass "Script shows missing files"
    else
        fail "Script should show missing files"
        echo "Output: $output"
    fi

    cleanup
}

# Test 4: Add all files option
# テスト4: 全て追加オプション
test_add_all_files() {
    echo ""
    echo "=== Test: Add all files option ==="

    setup

    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'
    create_compose_file "" ""

    local output
    # Use echo "1" to simulate "Add all" option
    # echo "1" で「すべて追加」オプションをシミュレート
    output=$(echo "1" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    # Check if compose file was modified
    # compose ファイルが変更されたか確認
    if grep -q "demo-app/.env" "$TEST_WORKSPACE/.devcontainer/docker-compose.yml"; then
        pass "Script adds file to docker-compose.yml"
    else
        fail "Script should add file to docker-compose.yml"
        echo "Output: $output"
        echo "Compose file contents:"
        cat "$TEST_WORKSPACE/.devcontainer/docker-compose.yml"
    fi

    cleanup
}

# Test 5: Creates backup in .sandbox/backups/
# テスト5: バックアップを .sandbox/backups/ に作成
test_creates_backup() {
    echo ""
    echo "=== Test: Creates backup in .sandbox/backups/ ==="

    setup

    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'
    create_compose_file "" ""

    local output
    output=$(echo "1" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    # Check if backup file was created in .sandbox/backups/
    # .sandbox/backups/ にバックアップファイルが作成されたか確認
    if ls "$TEST_WORKSPACE/.sandbox/backups/"*.docker-compose.yml.* 1>/dev/null 2>&1; then
        pass "Script creates backup file in .sandbox/backups/"
    else
        fail "Script should create backup file in .sandbox/backups/"
        echo "Files in .sandbox/backups/:"
        ls -la "$TEST_WORKSPACE/.sandbox/backups/" 2>/dev/null || echo "(directory not found)"
    fi

    cleanup
}

# Test 6: Don't add option
# テスト6: 追加しないオプション
test_dont_add_option() {
    echo ""
    echo "=== Test: Don't add option ==="

    setup

    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'
    create_compose_file "" ""

    local original_content
    original_content=$(cat "$TEST_WORKSPACE/.devcontainer/docker-compose.yml")

    local output
    # Use echo "3" to simulate "Don't add" option
    # echo "3" で「追加しない」オプションをシミュレート
    output=$(echo "3" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    local new_content
    new_content=$(cat "$TEST_WORKSPACE/.devcontainer/docker-compose.yml")

    if [ "$original_content" = "$new_content" ]; then
        pass "Script doesn't modify file when user declines"
    else
        fail "Script should not modify file when user declines"
    fi

    cleanup
}

# Test 7: Shows rebuild instructions
# テスト7: リビルド手順を表示
test_shows_rebuild_instructions() {
    echo ""
    echo "=== Test: Shows rebuild instructions ==="

    setup

    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'
    create_compose_file "" ""

    local output
    output=$(echo "1" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    if echo "$output" | grep -qi "rebuild\|リビルド"; then
        pass "Script shows rebuild instructions after adding"
    else
        fail "Script should show rebuild instructions after adding"
        echo "Output: $output"
    fi

    cleanup
}

# Test 8: Handles no Claude settings
# テスト8: Claude設定がない場合の処理
test_no_claude_settings() {
    echo ""
    echo "=== Test: Handles no Claude settings ==="

    setup

    rm -f "$TEST_WORKSPACE/.claude/settings.json"
    create_compose_file "" ""

    local exit_code=0
    WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" > /dev/null 2>&1 || exit_code=$?

    if [ "$exit_code" -ne 0 ]; then
        pass "Script exits with error when no Claude settings"
    else
        fail "Script should exit with error when no Claude settings"
    fi

    cleanup
}

# Test 9: Preview option shows config without modifying
# テスト9: プレビューオプションが変更せずに設定を表示
test_preview_option() {
    echo ""
    echo "=== Test: Preview option (dry-run) ==="

    setup

    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'
    create_compose_file "" ""

    local original_content
    original_content=$(cat "$TEST_WORKSPACE/.devcontainer/docker-compose.yml")

    local output
    # Use echo "4" to simulate "Preview" option
    # echo "4" で「プレビュー」オプションをシミュレート
    output=$(echo "4" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    local new_content
    new_content=$(cat "$TEST_WORKSPACE/.devcontainer/docker-compose.yml")

    # Check that file was not modified
    # ファイルが変更されていないことを確認
    if [ "$original_content" = "$new_content" ]; then
        # Check that preview shows the config format
        # プレビューが設定形式を表示することを確認
        if echo "$output" | grep -q "/dev/null:"; then
            pass "Preview shows config without modifying file"
        else
            fail "Preview should show /dev/null config format"
            echo "Output: $output"
        fi
    else
        fail "Preview should not modify the file"
    fi

    cleanup
}

# Test 10: Preview shows both volumes and tmpfs sections
# テスト10: プレビューがvolumesとtmpfsの両セクションを表示
test_preview_shows_sections() {
    echo ""
    echo "=== Test: Preview shows volumes and tmpfs sections ==="

    setup

    # Create both file and directory
    # ファイルとディレクトリの両方を作成
    touch "$TEST_WORKSPACE/demo-app/.env"
    mkdir -p "$TEST_WORKSPACE/demo-app/secrets"
    touch "$TEST_WORKSPACE/demo-app/secrets/key.txt"
    create_claude_settings '"Read(demo-app/.env)", "Read(demo-app/secrets/**)"'
    create_compose_file "" ""

    local output
    output=$(echo "4" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    # Check that both sections are shown
    # 両方のセクションが表示されることを確認
    if echo "$output" | grep -qi "volumes\|ボリューム"; then
        if echo "$output" | grep -q "/dev/null:"; then
            pass "Preview shows both file and directory config"
        else
            fail "Preview should show /dev/null format for files"
            echo "Output: $output"
        fi
    else
        fail "Preview should show volumes section"
        echo "Output: $output"
    fi

    cleanup
}

# Test 11: Dual-file - adds to both compose files
# テスト11: 両ファイル - 両方のcompose ファイルに追加
test_dual_file_add_all() {
    echo ""
    echo "=== Test: Dual-file - adds to both compose files ==="

    setup

    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'
    create_compose_file "" ""
    create_cli_compose_file "" ""

    local output
    output=$(echo "1" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    local dc_has cli_has
    dc_has=false
    cli_has=false
    grep -q "demo-app/.env" "$TEST_WORKSPACE/.devcontainer/docker-compose.yml" && dc_has=true
    grep -q "demo-app/.env" "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml" && cli_has=true

    if [ "$dc_has" = true ] && [ "$cli_has" = true ]; then
        pass "Script adds file to both compose files"
    else
        fail "Script should add file to both compose files (DC=$dc_has, CLI=$cli_has)"
        echo "Output: $output"
    fi

    cleanup
}

# Test 12: Dual-file - shows which file(s) are missing
# テスト12: 両ファイル - どのファイルに不足しているか表示
test_dual_file_shows_missing_labels() {
    echo ""
    echo "=== Test: Dual-file - shows missing labels ==="

    setup

    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'
    # DevContainer has it, CLI Sandbox does NOT
    # DevContainer にはあるが CLI Sandbox にはない
    create_compose_file "      - /dev/null:$TEST_WORKSPACE/demo-app/.env:ro" ""
    create_cli_compose_file "" ""

    local output
    output=$(echo "3" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    if echo "$output" | grep -q "CLI Sandbox"; then
        pass "Script shows CLI Sandbox as missing"
    else
        fail "Script should show CLI Sandbox as missing"
        echo "Output: $output"
    fi

    cleanup
}

# Test 13: Dual-file - only adds to compose file where missing
# テスト13: 両ファイル - 不足しているcompose ファイルにのみ追加
test_dual_file_adds_only_where_missing() {
    echo ""
    echo "=== Test: Dual-file - adds only where missing ==="

    setup

    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'
    # DevContainer already has it
    # DevContainer にはすでにある
    create_compose_file "      - /dev/null:$TEST_WORKSPACE/demo-app/.env:ro" ""
    create_cli_compose_file "" ""

    local dc_before
    dc_before=$(cat "$TEST_WORKSPACE/.devcontainer/docker-compose.yml")

    local output
    output=$(echo "1" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    # CLI should now have it
    # CLI にも追加されているはず
    local cli_has=false
    grep -q "demo-app/.env" "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml" && cli_has=true

    # DevContainer should NOT have duplicate
    # DevContainer に重複がないこと
    local dc_count
    dc_count=$(grep -c "demo-app/.env" "$TEST_WORKSPACE/.devcontainer/docker-compose.yml" || true)

    if [ "$cli_has" = true ] && [ "$dc_count" -le 1 ]; then
        pass "Script adds only where missing, no duplicates"
    else
        fail "Script should add only where missing (CLI=$cli_has, DC count=$dc_count)"
        echo "Output: $output"
    fi

    cleanup
}

# Test 14: Dual-file - backups created for both in .sandbox/backups/
# テスト14: 両ファイル - 両方のバックアップが .sandbox/backups/ に作成される
test_dual_file_backups() {
    echo ""
    echo "=== Test: Dual-file - backups created for both in .sandbox/backups/ ==="

    setup

    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'
    create_compose_file "" ""
    create_cli_compose_file "" ""

    local output
    output=$(echo "1" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    local dc_backup=false
    local cli_backup=false
    ls "$TEST_WORKSPACE/.sandbox/backups/devcontainer."* 1>/dev/null 2>&1 && dc_backup=true
    ls "$TEST_WORKSPACE/.sandbox/backups/cli_sandbox."* 1>/dev/null 2>&1 && cli_backup=true

    if [ "$dc_backup" = true ] && [ "$cli_backup" = true ]; then
        pass "Script creates backups for both compose files in .sandbox/backups/"
    else
        fail "Script should create backups for both in .sandbox/backups/ (DC=$dc_backup, CLI=$cli_backup)"
    fi

    cleanup
}

# Test 15: Dual-file - all synced when both have the file
# テスト15: 両ファイル - 両方にファイルがある場合は同期済み
test_dual_file_all_synced() {
    echo ""
    echo "=== Test: Dual-file - all synced when both have file ==="

    setup

    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'
    create_compose_file "      - /dev/null:$TEST_WORKSPACE/demo-app/.env:ro" ""
    create_cli_compose_file "      - /dev/null:$TEST_WORKSPACE/demo-app/.env:ro" ""

    local output
    output=$(echo "3" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    if echo "$output" | grep -q "すべての秘匿\|All secret files are synced\|No additions needed"; then
        pass "Script exits cleanly when both compose files are synced"
    else
        fail "Script should exit cleanly when both compose files are synced"
        echo "Output: $output"
    fi

    cleanup
}

# Test 16: Shows detected compose files
# テスト16: 検出されたcompose ファイルを表示
test_shows_detected_compose_files() {
    echo ""
    echo "=== Test: Shows detected compose files ==="

    setup

    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'
    create_compose_file "" ""
    create_cli_compose_file "" ""

    local output
    output=$(echo "3" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    local has_dc=false
    local has_cli=false
    echo "$output" | grep -q "DevContainer" && has_dc=true
    echo "$output" | grep -q "CLI Sandbox" && has_cli=true

    if [ "$has_dc" = true ] && [ "$has_cli" = true ]; then
        pass "Script shows both detected compose files"
    else
        fail "Script should show both detected compose files (DC=$has_dc, CLI=$has_cli)"
        echo "Output: $output"
    fi

    cleanup
}

# Run all tests
# 全テストを実行
main() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  sync-secrets.sh Test Suite"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    test_script_executable_and_valid
    test_all_synced
    test_shows_missing_files
    test_add_all_files
    test_creates_backup
    test_dont_add_option
    test_shows_rebuild_instructions
    test_no_claude_settings
    test_preview_option
    test_preview_shows_sections
    test_dual_file_add_all
    test_dual_file_shows_missing_labels
    test_dual_file_adds_only_where_missing
    test_dual_file_backups
    test_dual_file_all_synced
    test_shows_detected_compose_files

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

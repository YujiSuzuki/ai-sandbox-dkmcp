#!/bin/bash
# test-sync-compose-secrets.sh
# Test script for sync-compose-secrets.sh
#
# sync-compose-secrets.sh のテストスクリプト
#
# Usage: ./test-sync-compose-secrets.sh
# 使用方法: ./test-sync-compose-secrets.sh
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
SCRIPT="$SCRIPT_DIR/sync-compose-secrets.sh"
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
    # Create temporary workspace
    # 一時的なワークスペースを作成
    TEST_WORKSPACE=$(mktemp -d)

    # Create directory structure
    mkdir -p "$TEST_WORKSPACE/.devcontainer"
    mkdir -p "$TEST_WORKSPACE/cli_sandbox"
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
    # Remove test workspace
    # テスト用ワークスペースを削除
    if [ -n "$TEST_WORKSPACE" ] && [ -d "$TEST_WORKSPACE" ]; then
        rm -rf "$TEST_WORKSPACE"
    fi
    TEST_WORKSPACE=""
}

# Trap to ensure cleanup runs
# クリーンアップが必ず実行されるようトラップ設定
trap cleanup EXIT

# Create matching docker-compose files (no sync needed)
# 一致する docker-compose ファイルを作成（同期不要）
create_matching_configs() {
    cat > "$TEST_WORKSPACE/.devcontainer/docker-compose.yml" << 'EOF'
services:
  ai-sandbox:
    volumes:
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
    tmpfs:
      - /workspace/demo-apps/securenote-api/secrets:ro
EOF

    cat > "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml" << 'EOF'
services:
  cli-sandbox:
    volumes:
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
    tmpfs:
      - /tmp:rw,noexec,nosuid,size=1g
      - /workspace/demo-apps/securenote-api/secrets:ro
EOF
}

# Create mismatched configs (DevContainer has extra volume)
# 不一致の設定を作成（DevContainer に追加のボリュームがある）
create_dc_extra_volume() {
    cat > "$TEST_WORKSPACE/.devcontainer/docker-compose.yml" << 'EOF'
services:
  ai-sandbox:
    volumes:
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
      - /dev/null:/workspace/another-app/.env:ro
    tmpfs:
      - /workspace/demo-apps/securenote-api/secrets:ro
EOF

    cat > "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml" << 'EOF'
services:
  cli-sandbox:
    volumes:
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
    tmpfs:
      - /workspace/demo-apps/securenote-api/secrets:ro
EOF
}

# Create mismatched configs (CLI has extra tmpfs)
# 不一致の設定を作成（CLI に追加の tmpfs がある）
create_cli_extra_tmpfs() {
    cat > "$TEST_WORKSPACE/.devcontainer/docker-compose.yml" << 'EOF'
services:
  ai-sandbox:
    volumes:
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
    tmpfs:
      - /workspace/demo-apps/securenote-api/secrets:ro
EOF

    cat > "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml" << 'EOF'
services:
  cli-sandbox:
    volumes:
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
    tmpfs:
      - /workspace/demo-apps/securenote-api/secrets:ro
      - /workspace/another-app/secrets:ro
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

# Test 2: Script exits successfully when configs already match
# テスト2: 設定が既に一致している場合、正常終了する
test_already_synced() {
    echo ""
    echo "=== Test: Exit success when already synced ==="

    setup
    create_matching_configs

    # Run script with "3" (don't sync) - but it should exit early since there's no diff
    if output=$(echo "3" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1); then
        if echo "$output" | grep -q "✅"; then
            pass "Script detects already synced configs"
        else
            fail "Script should report configs are synced"
        fi
    else
        fail "Script should exit successfully when configs match"
    fi

    cleanup
}

# Test 3: Script detects DevContainer-only volumes
# テスト3: DevContainer のみのボリュームを検出する
test_detect_dc_only_volume() {
    echo ""
    echo "=== Test: Detect DevContainer-only volume ==="

    setup
    create_dc_extra_volume

    # Run with preview option (4)
    output=$(echo "4" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    if echo "$output" | grep -q "another-app"; then
        pass "Script detects DevContainer-only volume"
    else
        fail "Script should detect DevContainer-only volume"
    fi

    cleanup
}

# Test 4: Script detects CLI-only tmpfs
# テスト4: CLI のみの tmpfs を検出する
test_detect_cli_only_tmpfs() {
    echo ""
    echo "=== Test: Detect CLI-only tmpfs ==="

    setup
    create_cli_extra_tmpfs

    # Run with preview option (4)
    output=$(echo "4" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    if echo "$output" | grep -q "another-app/secrets"; then
        pass "Script detects CLI-only tmpfs"
    else
        fail "Script should detect CLI-only tmpfs"
    fi

    cleanup
}

# Test 5: Sync adds missing volume to CLI
# テスト5: 不足しているボリュームを CLI に追加する
test_sync_volume_to_cli() {
    echo ""
    echo "=== Test: Sync volume to CLI Sandbox ==="

    setup
    create_dc_extra_volume

    # Run with "sync all" option (1)
    echo "1" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" > /dev/null 2>&1 || true

    # Check that CLI now has the missing entry
    if grep -q "another-app" "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml"; then
        pass "Volume synced to CLI Sandbox"
    else
        fail "Volume should be synced to CLI Sandbox"
    fi

    cleanup
}

# Test 6: Sync adds missing tmpfs to DevContainer
# テスト6: 不足している tmpfs を DevContainer に追加する
test_sync_tmpfs_to_dc() {
    echo ""
    echo "=== Test: Sync tmpfs to DevContainer ==="

    setup
    create_cli_extra_tmpfs

    # Run with "sync all" option (1)
    echo "1" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" > /dev/null 2>&1 || true

    # Check that DevContainer now has the missing entry
    if grep -q "another-app/secrets" "$TEST_WORKSPACE/.devcontainer/docker-compose.yml"; then
        pass "Tmpfs synced to DevContainer"
    else
        fail "Tmpfs should be synced to DevContainer"
    fi

    cleanup
}

# Test 7: Backups are created
# テスト7: バックアップが作成される
test_backup_creation() {
    echo ""
    echo "=== Test: Backups are created ==="

    setup
    create_dc_extra_volume

    # Run with "sync all" option (1)
    echo "1" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" > /dev/null 2>&1 || true

    # Check that backup files exist
    local dc_backup=$(ls "$TEST_WORKSPACE/.devcontainer/docker-compose.yml.backup."* 2>/dev/null | head -1)
    local cli_backup=$(ls "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml.backup."* 2>/dev/null | head -1)

    if [ -n "$dc_backup" ] && [ -n "$cli_backup" ]; then
        pass "Backup files created"
    else
        fail "Backup files should be created"
    fi

    cleanup
}

# Test 8: Script fails when DevContainer config missing
# テスト8: DevContainer 設定ファイルがない場合に失敗する
test_missing_dc_config() {
    echo ""
    echo "=== Test: Fail when DevContainer config missing ==="

    setup
    # Only create CLI config
    cat > "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml" << 'EOF'
services:
  cli-sandbox:
    volumes:
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
EOF

    if WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" > /dev/null 2>&1; then
        fail "Script should fail when DevContainer config is missing"
    else
        pass "Script fails when DevContainer config is missing"
    fi

    cleanup
}

# Test 9: Script fails when CLI config missing
# テスト9: CLI 設定ファイルがない場合に失敗する
test_missing_cli_config() {
    echo ""
    echo "=== Test: Fail when CLI config missing ==="

    setup
    # Only create DevContainer config
    cat > "$TEST_WORKSPACE/.devcontainer/docker-compose.yml" << 'EOF'
services:
  ai-sandbox:
    volumes:
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
EOF

    if WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" > /dev/null 2>&1; then
        fail "Script should fail when CLI config is missing"
    else
        pass "Script fails when CLI config is missing"
    fi

    cleanup
}

# Test 10: "Don't sync" option exits without changes
# テスト10: 「同期しない」オプションで変更なしで終了する
test_skip_sync() {
    echo ""
    echo "=== Test: Skip sync option ==="

    setup
    create_dc_extra_volume

    # Get original content
    local original_cli=$(cat "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml")

    # Run with "don't sync" option (3)
    echo "3" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" > /dev/null 2>&1 || true

    # Check that file is unchanged
    local after_cli=$(cat "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml")

    if [ "$original_cli" = "$after_cli" ]; then
        pass "Skip option leaves files unchanged"
    else
        fail "Skip option should leave files unchanged"
    fi

    cleanup
}

# Test 11: Bidirectional sync (both files have missing entries)
# テスト11: 双方向同期（両方のファイルに不足エントリがある）
test_bidirectional_sync() {
    echo ""
    echo "=== Test: Bidirectional sync ==="

    setup

    # DevContainer has extra volume, CLI has extra tmpfs
    cat > "$TEST_WORKSPACE/.devcontainer/docker-compose.yml" << 'EOF'
services:
  ai-sandbox:
    volumes:
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
      - /dev/null:/workspace/dc-only-app/.env:ro
    tmpfs:
      - /workspace/demo-apps/securenote-api/secrets:ro
EOF

    cat > "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml" << 'EOF'
services:
  cli-sandbox:
    volumes:
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
    tmpfs:
      - /workspace/demo-apps/securenote-api/secrets:ro
      - /workspace/cli-only-app/secrets:ro
EOF

    # Run with "sync all" option (1)
    echo "1" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" > /dev/null 2>&1 || true

    # Check that both files now have both entries
    local dc_has_cli_entry=$(grep -c "cli-only-app" "$TEST_WORKSPACE/.devcontainer/docker-compose.yml" || echo "0")
    local cli_has_dc_entry=$(grep -c "dc-only-app" "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml" || echo "0")

    if [ "$dc_has_cli_entry" -ge 1 ] && [ "$cli_has_dc_entry" -ge 1 ]; then
        pass "Bidirectional sync works"
    else
        fail "Bidirectional sync should add entries to both files (dc_has_cli=$dc_has_cli_entry, cli_has_dc=$cli_has_dc_entry)"
    fi

    cleanup
}

# Test 12: Idempotency (running twice doesn't create duplicates)
# テスト12: 冪等性（2回実行しても重複しない）
test_idempotency() {
    echo ""
    echo "=== Test: Idempotency (no duplicates on second run) ==="

    setup
    create_dc_extra_volume

    # Run sync twice
    echo "1" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" > /dev/null 2>&1 || true
    echo "1" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" > /dev/null 2>&1 || true

    # Count occurrences of the synced entry
    local count=$(grep -c "another-app" "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml" || echo "0")

    if [ "$count" -eq 1 ]; then
        pass "No duplicates after running twice"
    else
        fail "Entry should appear exactly once, but found $count times"
    fi

    cleanup
}

# Test 13: Multiple entries sync
# テスト13: 複数エントリの同期
test_multiple_entries_sync() {
    echo ""
    echo "=== Test: Multiple entries sync ==="

    setup

    # DevContainer has multiple extra entries
    cat > "$TEST_WORKSPACE/.devcontainer/docker-compose.yml" << 'EOF'
services:
  ai-sandbox:
    volumes:
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
      - /dev/null:/workspace/app-one/.env:ro
      - /dev/null:/workspace/app-two/.env:ro
    tmpfs:
      - /workspace/demo-apps/securenote-api/secrets:ro
EOF

    cat > "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml" << 'EOF'
services:
  cli-sandbox:
    volumes:
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
    tmpfs:
      - /workspace/demo-apps/securenote-api/secrets:ro
EOF

    # Run with "sync all" option (1)
    echo "1" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" > /dev/null 2>&1 || true

    # Check that both entries were synced
    local has_one=$(grep -c "app-one" "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml" || echo "0")
    local has_two=$(grep -c "app-two" "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml" || echo "0")

    if [ "$has_one" -ge 1 ] && [ "$has_two" -ge 1 ]; then
        pass "Multiple entries synced"
    else
        fail "All entries should be synced (app-one=$has_one, app-two=$has_two)"
    fi

    cleanup
}

# Test 14: After sync, compare-secret-config shows match
# テスト14: 同期後、compare-secret-config が一致を報告する
test_sync_then_compare() {
    echo ""
    echo "=== Test: After sync, compare shows match ==="

    setup
    create_dc_extra_volume

    # Run sync
    echo "1" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" > /dev/null 2>&1 || true

    # Run compare script - should return success (exit 0)
    if WORKSPACE="$TEST_WORKSPACE" "$SCRIPT_DIR/compare-secret-config.sh" > /dev/null 2>&1; then
        pass "After sync, compare-secret-config shows match"
    else
        fail "After sync, compare-secret-config should show match"
    fi

    cleanup
}

# Test 15: Summary shows synced entries
# テスト15: サマリーに同期したエントリが表示される
test_summary_shows_synced_entries() {
    echo ""
    echo "=== Test: Summary shows synced entries ==="

    setup
    create_dc_extra_volume

    # Run with "sync all" option (1) and capture output
    output=$(echo "1" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    # Check that summary shows the synced entry
    if echo "$output" | grep -q "another-app" && echo "$output" | grep -q "✅"; then
        pass "Summary shows synced entries"
    else
        fail "Summary should show synced entries with checkmarks"
    fi

    cleanup
}

# Test 16: Summary shows "no entries synced" when skipping
# テスト16: スキップ時に「同期されたエントリはありません」が表示される
test_summary_shows_none_when_skipped() {
    echo ""
    echo "=== Test: Summary shows 'no entries' when skipped ==="

    setup
    create_dc_extra_volume

    # Run with "don't sync" option (3) and capture output
    output=$(echo "3" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    # Check that output shows skipped message (not the "no entries synced" since we skipped entirely)
    # When user selects "3", script exits before summary, so just check it doesn't show synced entries
    if echo "$output" | grep -qE "(Skipped|スキップ)"; then
        pass "Skip message displayed correctly"
    else
        fail "Skip message should be displayed"
    fi

    cleanup
}

# Test 17: Individual confirmation (option 2) prompts for each entry
# テスト17: 個別確認（オプション2）が各エントリで確認を求める
test_individual_confirmation_accept() {
    echo ""
    echo "=== Test: Individual confirmation accepts entry ==="

    setup
    create_dc_extra_volume

    # Run with "review each" option (2) then "y" for confirmation
    # Option 2 + y confirmation should sync the entry
    output=$(printf '2\ny\n' | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    # Check that entry was synced (user confirmed with "y")
    if grep -q "another-app" "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml"; then
        pass "Individual confirmation syncs entry when user confirms"
    else
        fail "Individual confirmation should sync entry when user confirms 'y'"
    fi

    cleanup
}

# Test 18: Individual confirmation (option 2) skips when user declines
# テスト18: 個別確認（オプション2）でユーザーが拒否すると スキップする
test_individual_confirmation_decline() {
    echo ""
    echo "=== Test: Individual confirmation skips when declined ==="

    setup
    create_dc_extra_volume

    # Get original content
    local original_cli=$(cat "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml")

    # Run with "review each" option (2) then "n" for decline
    output=$(printf '2\nn\n' | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    # Check that entry was NOT synced (user declined with "n")
    if ! grep -q "another-app" "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml"; then
        pass "Individual confirmation skips entry when user declines"
    else
        fail "Individual confirmation should NOT sync entry when user declines 'n'"
    fi

    cleanup
}

# Test 19: Individual confirmation shows item details
# テスト19: 個別確認でアイテムの詳細が表示される
test_individual_confirmation_shows_item_details() {
    echo ""
    echo "=== Test: Individual confirmation shows item details ==="

    setup
    create_dc_extra_volume

    # Run with "review each" option (2) then "n" to skip
    output=$(printf '2\nn\n' | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    # Check that the item path and target file are shown
    # Note: read -p prompt may not be displayed in non-interactive mode (piped input)
    # 注: read -p のプロンプトは非インタラクティブモード（パイプ入力）では表示されない場合がある
    if echo "$output" | grep -q "another-app" && echo "$output" | grep -qE "(cli_sandbox|Target file|追加先)"; then
        pass "Individual confirmation shows item details"
    else
        fail "Individual confirmation should show item path and target file"
        echo "Output was:"
        echo "$output"
    fi

    cleanup
}

# Test 20: Summary shows count when multiple entries synced
# テスト20: 複数エントリ同期時にサマリーに全エントリが表示される
test_summary_shows_multiple_entries() {
    echo ""
    echo "=== Test: Summary shows all synced entries ==="

    setup

    # DevContainer has multiple extra entries
    cat > "$TEST_WORKSPACE/.devcontainer/docker-compose.yml" << 'EOF'
services:
  ai-sandbox:
    volumes:
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
      - /dev/null:/workspace/app-alpha/.env:ro
      - /dev/null:/workspace/app-beta/.env:ro
    tmpfs:
      - /workspace/demo-apps/securenote-api/secrets:ro
EOF

    cat > "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml" << 'EOF'
services:
  cli-sandbox:
    volumes:
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
    tmpfs:
      - /workspace/demo-apps/securenote-api/secrets:ro
EOF

    # Run with "sync all" option (1) and capture output
    output=$(echo "1" | WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    # Check that summary shows both synced entries
    local has_alpha=$(echo "$output" | grep -c "app-alpha" || echo "0")
    local has_beta=$(echo "$output" | grep -c "app-beta" || echo "0")

    if [ "$has_alpha" -ge 1 ] && [ "$has_beta" -ge 1 ]; then
        pass "Summary shows all synced entries"
    else
        fail "Summary should show all synced entries (alpha=$has_alpha, beta=$has_beta)"
    fi

    cleanup
}

# Run all tests
# 全テストを実行
main() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  sync-compose-secrets.sh Test Suite"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    test_script_executable_and_valid
    test_already_synced
    test_detect_dc_only_volume
    test_detect_cli_only_tmpfs
    test_sync_volume_to_cli
    test_sync_tmpfs_to_dc
    test_backup_creation
    test_missing_dc_config
    test_missing_cli_config
    test_skip_sync
    test_bidirectional_sync
    test_idempotency
    test_multiple_entries_sync
    test_sync_then_compare
    test_summary_shows_synced_entries
    test_summary_shows_none_when_skipped
    test_individual_confirmation_accept
    test_individual_confirmation_decline
    test_individual_confirmation_shows_item_details
    test_summary_shows_multiple_entries

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

#!/bin/bash
# test-compare-secret-config.sh
# Test script for compare-secret-config.sh
#
# compare-secret-config.sh のテストスクリプト
#
# Usage: ./test-compare-secret-config.sh
# 使用方法: ./test-compare-secret-config.sh
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
SCRIPT="$SCRIPT_DIR/compare-secret-config.sh"
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
    info "Cleaning up test environment..."

    # Remove test workspace
    # テスト用ワークスペースを削除
    if [ -n "$TEST_WORKSPACE" ] && [ -d "$TEST_WORKSPACE" ]; then
        rm -rf "$TEST_WORKSPACE"
    fi
}

# Trap to ensure cleanup runs
# クリーンアップが必ず実行されるようトラップ設定
trap cleanup EXIT

# Create matching docker-compose files
# 一致する docker-compose ファイルを作成
create_matching_configs() {
    cat > "$TEST_WORKSPACE/.devcontainer/docker-compose.yml" << 'EOF'
services:
  ai-sandbox:
    build:
      context: ..
      dockerfile: .sandbox/Dockerfile
    volumes:
      - ..:/workspace:cached
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
    tmpfs:
      - /workspace/demo-apps/securenote-api/secrets:ro
EOF

    cat > "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml" << 'EOF'
services:
  cli-sandbox:
    build:
      context: .
      dockerfile: .sandbox/Dockerfile
    volumes:
      - .:/workspace
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
    tmpfs:
      - /tmp:rw,noexec,nosuid,size=1g
      - /workspace/demo-apps/securenote-api/secrets:ro
EOF
}

# Create mismatched docker-compose files (volumes differ)
# 不一致の docker-compose ファイルを作成（volumes が異なる）
create_mismatched_volumes() {
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

# Create mismatched docker-compose files (tmpfs differ)
# 不一致の docker-compose ファイルを作成（tmpfs が異なる）
create_mismatched_tmpfs() {
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

# Test 2: Script runs without error when configs match
# テスト2: 設定が一致する場合、スクリプトがエラーなく実行される
test_matching_configs() {
    echo ""
    echo "=== Test: Matching configs return success ==="

    setup
    create_matching_configs

    if WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" > /dev/null 2>&1; then
        pass "Script returns success when configs match"
    else
        fail "Script should return success when configs match"
    fi

    cleanup
}

# Test 3: Script detects mismatched volumes
# テスト3: volumes の不一致を検出する
test_mismatched_volumes() {
    echo ""
    echo "=== Test: Detect mismatched volumes ==="

    setup
    create_mismatched_volumes

    if WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" > /dev/null 2>&1; then
        fail "Script should return error when volumes don't match"
    else
        pass "Script detects mismatched volumes"
    fi

    cleanup
}

# Test 4: Script detects mismatched tmpfs
# テスト4: tmpfs の不一致を検出する
test_mismatched_tmpfs() {
    echo ""
    echo "=== Test: Detect mismatched tmpfs ==="

    setup
    create_mismatched_tmpfs

    if WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" > /dev/null 2>&1; then
        fail "Script should return error when tmpfs don't match"
    else
        pass "Script detects mismatched tmpfs"
    fi

    cleanup
}

# Test 5: Script fails when devcontainer config missing
# テスト5: devcontainer 設定ファイルがない場合に失敗する
test_missing_devcontainer_config() {
    echo ""
    echo "=== Test: Fail when devcontainer config missing ==="

    setup
    # Only create cli_sandbox config
    # cli_sandbox の設定のみ作成
    cat > "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml" << 'EOF'
services:
  cli-sandbox:
    volumes:
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
EOF

    if WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" > /dev/null 2>&1; then
        fail "Script should fail when devcontainer config is missing"
    else
        pass "Script fails when devcontainer config is missing"
    fi

    cleanup
}

# Test 6: Script fails when cli_sandbox config missing
# テスト6: cli_sandbox 設定ファイルがない場合に失敗する
test_missing_cli_config() {
    echo ""
    echo "=== Test: Fail when cli_sandbox config missing ==="

    setup
    # Only create devcontainer config
    # devcontainer の設定のみ作成
    cat > "$TEST_WORKSPACE/.devcontainer/docker-compose.yml" << 'EOF'
services:
  ai-sandbox:
    volumes:
      - /dev/null:/workspace/demo-apps/securenote-api/.env:ro
EOF

    if WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" > /dev/null 2>&1; then
        fail "Script should fail when cli_sandbox config is missing"
    else
        pass "Script fails when cli_sandbox config is missing"
    fi

    cleanup
}

# Run all tests
# 全テストを実行
main() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  compare-secret-config.sh Test Suite"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    test_script_executable_and_valid
    test_matching_configs
    test_mismatched_volumes
    test_mismatched_tmpfs
    test_missing_devcontainer_config
    test_missing_cli_config

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

#!/bin/bash
# test-validate-secrets.sh
# Test script for validate-secrets.sh
#
# validate-secrets.sh のテストスクリプト
#
# Usage: ./test-validate-secrets.sh
# 使用方法: ./test-validate-secrets.sh
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
SCRIPT="$SCRIPT_DIR/validate-secrets.sh"
TEST_COMPOSE_DIR=""
TEST_SECRET_DIR=""

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

    # Create temporary directories
    # 一時ディレクトリを作成
    TEST_COMPOSE_DIR=$(mktemp -d)
    # Must be under /workspace because validate-secrets.sh only checks /workspace paths in tmpfs
    # validate-secrets.sh が tmpfs で /workspace パスのみチェックするため /workspace 配下に作成
    TEST_SECRET_DIR=$(mktemp -d /workspace/.test-secrets-XXXXXX)

    mkdir -p "$TEST_COMPOSE_DIR/.devcontainer"
    mkdir -p "$TEST_COMPOSE_DIR/.sandbox/scripts"
    mkdir -p "$TEST_COMPOSE_DIR/.sandbox/config"

    # Copy required scripts and config to test workspace
    # 必要なスクリプトと設定をテストワークスペースにコピー
    cp "$SCRIPT_DIR/_startup_common.sh" "$TEST_COMPOSE_DIR/.sandbox/scripts/"
    cp "$SCRIPT_DIR/../config/startup.conf" "$TEST_COMPOSE_DIR/.sandbox/config/" 2>/dev/null || true
    cp "$SCRIPT_DIR/../config/sync-ignore" "$TEST_COMPOSE_DIR/.sandbox/config/" 2>/dev/null || true
}

# Cleanup test environment
# テスト環境のクリーンアップ
cleanup() {
    info "Cleaning up test environment..."

    if [ -n "$TEST_COMPOSE_DIR" ] && [ -d "$TEST_COMPOSE_DIR" ]; then
        rm -rf "$TEST_COMPOSE_DIR"
    fi

    if [ -n "$TEST_SECRET_DIR" ] && [ -d "$TEST_SECRET_DIR" ]; then
        rm -rf "$TEST_SECRET_DIR"
    fi
}

# Trap to ensure cleanup runs
# クリーンアップが必ず実行されるようトラップ設定
trap cleanup EXIT

# Create docker-compose.yml with secret config
# 秘匿設定付きの docker-compose.yml を作成
create_compose_with_secrets() {
    mkdir -p "$TEST_SECRET_DIR/secrets"
    cat > "$TEST_COMPOSE_DIR/.devcontainer/docker-compose.yml" << EOF
services:
  ai-sandbox:
    volumes:
      - /dev/null:$TEST_SECRET_DIR/.env:ro
    tmpfs:
      - $TEST_SECRET_DIR/secrets:ro
EOF
}

# Create docker-compose.yml without secret config
# 秘匿設定なしの docker-compose.yml を作成
create_compose_without_secrets() {
    cat > "$TEST_COMPOSE_DIR/.devcontainer/docker-compose.yml" << EOF
services:
  ai-sandbox:
    volumes:
      - ..:/workspace:cached
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

# Test 2: Script succeeds when secrets are properly hidden (empty)
# テスト2: 秘匿情報が正しく隠蔽されている場合（空）に成功するか
test_hidden_secrets_empty() {
    echo ""
    echo "=== Test: Hidden secrets (empty) ==="

    setup
    create_compose_with_secrets

    # Create empty .env file (simulates /dev/null mount)
    # 空の .env ファイルを作成（/dev/null マウントをシミュレート）
    touch "$TEST_SECRET_DIR/.env"
    # secrets/ directory is already empty
    # secrets/ ディレクトリは既に空

    if WORKSPACE="$TEST_COMPOSE_DIR" "$SCRIPT" > /dev/null 2>&1; then
        pass "Script succeeds when secrets are hidden"
    else
        fail "Script should succeed when secrets are hidden"
    fi

    cleanup
}

# Test 3: Script fails when secret file has content
# テスト3: 秘匿ファイルに内容がある場合に失敗するか
test_exposed_secret_file() {
    echo ""
    echo "=== Test: Exposed secret file (has content) ==="

    setup
    create_compose_with_secrets

    # Create .env with content (simulates exposed secret)
    # 内容のある .env を作成（露出した秘匿情報をシミュレート）
    echo "SECRET_KEY=exposed" > "$TEST_SECRET_DIR/.env"

    if WORKSPACE="$TEST_COMPOSE_DIR" "$SCRIPT" > /dev/null 2>&1; then
        fail "Script should fail when secret file has content"
    else
        pass "Script detects exposed secret file"
    fi

    cleanup
}

# Test 4: Script fails when secret directory has files
# テスト4: 秘匿ディレクトリにファイルがある場合に失敗するか
test_exposed_secret_dir() {
    echo ""
    echo "=== Test: Exposed secret directory (has files) ==="

    setup
    create_compose_with_secrets

    # Create empty .env
    # 空の .env を作成
    touch "$TEST_SECRET_DIR/.env"
    # Add file to secrets directory
    # secrets ディレクトリにファイルを追加
    echo "secret" > "$TEST_SECRET_DIR/secrets/key.txt"

    if WORKSPACE="$TEST_COMPOSE_DIR" "$SCRIPT" > /dev/null 2>&1; then
        fail "Script should fail when secret directory has files"
    else
        pass "Script detects exposed secret directory"
    fi

    cleanup
}

# Test 5: Script handles missing docker-compose.yml
# テスト5: docker-compose.yml がない場合の処理
test_missing_compose_file() {
    echo ""
    echo "=== Test: Missing docker-compose.yml ==="

    setup
    # Don't create docker-compose.yml

    if WORKSPACE="$TEST_COMPOSE_DIR" "$SCRIPT" > /dev/null 2>&1; then
        fail "Script should fail when docker-compose.yml is missing"
    else
        pass "Script fails when docker-compose.yml is missing"
    fi

    cleanup
}

# Test 6: Script handles no secret configuration
# テスト6: 秘匿設定がない場合の処理
test_no_secret_config() {
    echo ""
    echo "=== Test: No secret configuration ==="

    setup
    create_compose_without_secrets

    if WORKSPACE="$TEST_COMPOSE_DIR" "$SCRIPT" > /dev/null 2>&1; then
        pass "Script succeeds when no secrets are configured"
    else
        fail "Script should succeed when no secrets are configured"
    fi

    cleanup
}

# Test 7: Script succeeds when secret file doesn't exist
# テスト7: 秘匿ファイルが存在しない場合に成功するか
test_nonexistent_secret_file() {
    echo ""
    echo "=== Test: Non-existent secret file ==="

    setup
    create_compose_with_secrets

    # Don't create .env file at all
    # .env ファイルを作成しない
    # secrets/ directory is already empty
    # secrets/ ディレクトリは既に空

    if WORKSPACE="$TEST_COMPOSE_DIR" "$SCRIPT" > /dev/null 2>&1; then
        pass "Script succeeds when secret file doesn't exist"
    else
        fail "Script should succeed when secret file doesn't exist"
    fi

    cleanup
}

# Run all tests
# 全テストを実行
main() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  validate-secrets.sh Test Suite"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    test_script_executable_and_valid
    test_hidden_secrets_empty
    test_exposed_secret_file
    test_exposed_secret_dir
    test_missing_compose_file
    test_no_secret_config
    test_nonexistent_secret_file

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

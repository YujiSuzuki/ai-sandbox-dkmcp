#!/bin/bash
# test-check-secret-sync.sh
# Test script for check-secret-sync.sh
#
# check-secret-sync.sh のテストスクリプト
#
# Usage: ./test-check-secret-sync.sh
# 使用方法: ./test-check-secret-sync.sh
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
SCRIPT="$SCRIPT_DIR/check-secret-sync.sh"
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
create_compose_file() {
    local volume_mounts="$1"
    local tmpfs_mounts="$2"
    cat > "$TEST_WORKSPACE/.devcontainer/docker-compose.yml" << EOF
services:
  ai-sandbox:
    volumes:
      - ..:/workspace:cached
$volume_mounts
    tmpfs:
$tmpfs_mounts
EOF
}

# Create cli_sandbox/docker-compose.yml with volume mounts
# cli_sandbox/docker-compose.yml をボリュームマウント付きで作成
create_cli_compose_file() {
    local volume_mounts="$1"
    local tmpfs_mounts="$2"
    cat > "$TEST_WORKSPACE/cli_sandbox/docker-compose.yml" << EOF
services:
  cli-sandbox:
    volumes:
      - ..:/workspace:cached
$volume_mounts
    tmpfs:
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

# Test 2: All synced - no warnings
# テスト2: 全て同期済み - 警告なし
test_all_synced() {
    echo ""
    echo "=== Test: All secrets synced (no warnings) ==="

    setup

    # Create .env file and configure both settings
    # .env ファイルを作成し、両方の設定を構成
    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'
    create_compose_file "      - /dev/null:$TEST_WORKSPACE/demo-app/.env:ro" ""

    local output
    output=$(WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    # Default mode shows condensed output
    # デフォルトモードは簡潔な出力を表示
    if echo "$output" | grep -qE "all configured|すべての秘匿ファイル|All secret files"; then
        pass "Script reports all secrets are synced"
    else
        fail "Script should report all secrets are synced"
        echo "Output: $output"
    fi

    cleanup
}

# Test 3: Missing file - shows warning
# テスト3: 未設定ファイル - 警告表示
test_missing_file_warning() {
    echo ""
    echo "=== Test: Missing file shows warning ==="

    setup

    # Create .env file but don't configure in docker-compose
    # .env ファイルを作成するが、docker-compose には設定しない
    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'
    create_compose_file "" ""

    local output
    output=$(WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    # Default mode shows warning with "missing" or Japanese equivalent
    # デフォルトモードは "missing" または日本語の警告を表示
    if echo "$output" | grep -qE "missing|未設定|NOT configured"; then
        pass "Script warns about missing configuration"
    else
        fail "Script should warn about missing configuration"
        echo "Output: $output"
    fi

    cleanup
}

# Test 4: No claude settings file
# テスト4: Claude設定ファイルがない場合
test_no_claude_settings() {
    echo ""
    echo "=== Test: No Claude settings file ==="

    setup

    # Don't create .claude/settings.json
    # .claude/settings.json を作成しない
    rm -f "$TEST_WORKSPACE/.claude/settings.json"
    create_compose_file "" ""

    local output
    output=$(WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    if echo "$output" | grep -q "見つかりません\|not found"; then
        pass "Script handles missing Claude settings file"
    else
        fail "Script should handle missing Claude settings file"
        echo "Output: $output"
    fi

    cleanup
}

# Test 5: No matching files in filesystem
# テスト5: ファイルシステムに一致するファイルがない
test_no_matching_files() {
    echo ""
    echo "=== Test: No matching files in filesystem ==="

    setup

    # Configure deny pattern but don't create the file
    # deny パターンを設定するが、ファイルは作成しない
    create_claude_settings '"Read(demo-app/.env)"'
    create_compose_file "" ""

    local output
    output=$(WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    if echo "$output" | grep -q "見つかりませんでした\|No matching files found"; then
        pass "Script handles no matching files"
    else
        fail "Script should handle no matching files"
        echo "Output: $output"
    fi

    cleanup
}

# Test 6: Directory covered by tmpfs
# テスト6: tmpfs でカバーされるディレクトリ
test_directory_tmpfs_covered() {
    echo ""
    echo "=== Test: Directory covered by tmpfs ==="

    setup

    # Create secrets directory with file
    # ファイル付きの secrets ディレクトリを作成
    mkdir -p "$TEST_WORKSPACE/demo-app/secrets"
    touch "$TEST_WORKSPACE/demo-app/secrets/key.txt"
    create_claude_settings '"Read(demo-app/secrets/**)"'
    create_compose_file "" "      - $TEST_WORKSPACE/demo-app/secrets:ro"

    local output
    output=$(WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    # Default mode shows condensed output
    # デフォルトモードは簡潔な出力を表示
    if echo "$output" | grep -qE "all configured|すべての秘匿ファイル|All secret files"; then
        pass "Script recognizes tmpfs covered directory"
    else
        fail "Script should recognize tmpfs covered directory"
        echo "Output: $output"
    fi

    cleanup
}

# Test 7: Glob pattern matching
# テスト7: グロブパターンのマッチング
test_glob_pattern_matching() {
    echo ""
    echo "=== Test: Glob pattern matching ==="

    setup

    # Create multiple .env files
    # 複数の .env ファイルを作成
    touch "$TEST_WORKSPACE/demo-app/.env"
    mkdir -p "$TEST_WORKSPACE/other-app"
    touch "$TEST_WORKSPACE/other-app/.env"

    create_claude_settings '"Read(**/.env)"'
    # Only configure one of them
    # 1つだけ設定
    create_compose_file "      - /dev/null:$TEST_WORKSPACE/demo-app/.env:ro" ""

    local output
    output=$(WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    # Should warn about other-app/.env
    # other-app/.env について警告すべき
    if echo "$output" | grep -q "other-app"; then
        pass "Script detects unconfigured files matching glob pattern"
    else
        fail "Script should detect unconfigured files matching glob pattern"
        echo "Output: $output"
    fi

    cleanup
}

# Test 8: Suggests sync-secrets.sh
# テスト8: sync-secrets.sh を提案する
test_suggests_sync_script() {
    echo ""
    echo "=== Test: Suggests sync-secrets.sh ==="

    setup

    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'
    create_compose_file "" ""

    local output
    output=$(WORKSPACE="$TEST_WORKSPACE" "$SCRIPT" 2>&1) || true

    # Default mode shows action hint (manual edit or sync script in verbose mode)
    # デフォルトモードはアクションヒントを表示（手動編集、verboseではスクリプト名）
    if echo "$output" | grep -qE "docker-compose\.yml|sync-secrets\.sh"; then
        pass "Script suggests action for missing configuration"
    else
        fail "Script should suggest action for missing configuration"
        echo "Output: $output"
    fi

    cleanup
}

# Test 9: CLI Sandbox environment uses cli_sandbox/docker-compose.yml
# テスト9: CLI Sandbox 環境では cli_sandbox/docker-compose.yml を使用する
test_cli_sandbox_env_uses_cli_compose() {
    echo ""
    echo "=== Test: CLI Sandbox env uses cli_sandbox/docker-compose.yml ==="

    setup

    # Create .env file
    # .env ファイルを作成
    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'

    # DevContainer compose has the secret, CLI Sandbox does NOT
    # DevContainer compose には秘匿設定あり、CLI Sandbox にはなし
    create_compose_file "      - /dev/null:$TEST_WORKSPACE/demo-app/.env:ro" ""
    create_cli_compose_file "" ""

    local output
    # Run with SANDBOX_ENV=cli_claude to simulate CLI Sandbox environment
    # CLI Sandbox 環境をシミュレートするため SANDBOX_ENV=cli_claude で実行
    output=$(WORKSPACE="$TEST_WORKSPACE" SANDBOX_ENV=cli_claude "$SCRIPT" 2>&1) || true

    # Should warn because cli_sandbox/docker-compose.yml doesn't have the secret
    # cli_sandbox/docker-compose.yml に秘匿設定がないため警告が出るはず
    if echo "$output" | grep -q "demo-app/.env"; then
        pass "CLI Sandbox env correctly uses cli_sandbox/docker-compose.yml"
    else
        fail "CLI Sandbox env should use cli_sandbox/docker-compose.yml and warn about missing secret"
        echo "Output: $output"
    fi

    cleanup
}

# Test 10: CLI Sandbox environment reports synced when cli_sandbox compose has secrets
# テスト10: CLI Sandbox 環境で cli_sandbox compose に秘匿設定があれば同期済みを報告
test_cli_sandbox_env_synced() {
    echo ""
    echo "=== Test: CLI Sandbox env synced when cli_sandbox compose configured ==="

    setup

    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'

    # DevContainer compose does NOT have secret, but CLI Sandbox DOES
    # DevContainer compose には秘匿設定なし、CLI Sandbox にはあり
    create_compose_file "" ""
    create_cli_compose_file "      - /dev/null:$TEST_WORKSPACE/demo-app/.env:ro" ""

    local output
    output=$(WORKSPACE="$TEST_WORKSPACE" SANDBOX_ENV=cli_claude "$SCRIPT" 2>&1) || true

    # Should report synced because cli_sandbox/docker-compose.yml has the secret
    # cli_sandbox/docker-compose.yml に秘匿設定があるため同期済みを報告するはず
    if echo "$output" | grep -qE "all configured|✅|All secret files are configured|すべての秘匿ファイル"; then
        pass "CLI Sandbox env reports synced when cli_sandbox compose configured"
    else
        fail "CLI Sandbox env should report synced when cli_sandbox compose has secrets"
        echo "Output: $output"
    fi

    cleanup
}

# Test 11: .aiexclude patterns are detected
# テスト11: .aiexclude パターンが検出される
test_aiexclude_patterns() {
    echo ""
    echo "=== Test: .aiexclude patterns detected ==="

    setup

    # Create .aiexclude with pattern
    # .aiexclude にパターンを作成
    cat > "$TEST_WORKSPACE/.aiexclude" << 'EOF'
# Gemini Code Assist exclusion
.env
secrets/
EOF

    # Create file matching .aiexclude pattern
    touch "$TEST_WORKSPACE/.env"

    # No Claude settings (empty)
    mkdir -p "$TEST_WORKSPACE/.claude"
    echo '{}' > "$TEST_WORKSPACE/.claude/settings.json"

    create_compose_file "" ""

    local output
    output=$(WORKSPACE="$TEST_WORKSPACE" SANDBOX_ENV=devcontainer "$SCRIPT" 2>&1) || true

    # Should warn about .env from .aiexclude
    # .aiexclude の .env について警告するはず
    if echo "$output" | grep -qE "\.env|missing|未設定"; then
        pass ".aiexclude patterns are detected and reported"
    else
        fail ".aiexclude patterns should be detected"
        echo "Output: $output"
    fi

    cleanup
}

# Test 12: .geminiignore patterns are detected
# テスト12: .geminiignore パターンが検出される
test_geminiignore_patterns() {
    echo ""
    echo "=== Test: .geminiignore patterns detected ==="

    setup

    # Create .geminiignore with pattern
    # .geminiignore にパターンを作成
    cat > "$TEST_WORKSPACE/.geminiignore" << 'EOF'
# Gemini CLI exclusion
api-key.txt
*.secret
EOF

    # Create file matching .geminiignore pattern
    touch "$TEST_WORKSPACE/api-key.txt"

    # No Claude settings (empty)
    mkdir -p "$TEST_WORKSPACE/.claude"
    echo '{}' > "$TEST_WORKSPACE/.claude/settings.json"

    create_compose_file "" ""

    local output
    output=$(WORKSPACE="$TEST_WORKSPACE" SANDBOX_ENV=devcontainer "$SCRIPT" 2>&1) || true

    # Should warn about api-key.txt from .geminiignore
    # .geminiignore の api-key.txt について警告するはず
    if echo "$output" | grep -qE "api-key\.txt|missing|未設定"; then
        pass ".geminiignore patterns are detected and reported"
    else
        fail ".geminiignore patterns should be detected"
        echo "Output: $output"
    fi

    cleanup
}

# Test 13: Combined Claude and Gemini patterns
# テスト13: Claude と Gemini のパターンを統合
test_combined_patterns() {
    echo ""
    echo "=== Test: Combined Claude and Gemini patterns ==="

    setup

    # Create Claude settings with one pattern
    touch "$TEST_WORKSPACE/demo-app/.env"
    create_claude_settings '"Read(demo-app/.env)"'

    # Create .aiexclude with different pattern (simple file pattern)
    # .aiexclude に別のパターンを作成（シンプルなファイルパターン）
    cat > "$TEST_WORKSPACE/.aiexclude" << 'EOF'
api-key.txt
EOF
    touch "$TEST_WORKSPACE/api-key.txt"

    create_compose_file "" ""

    local output
    output=$(WORKSPACE="$TEST_WORKSPACE" SANDBOX_ENV=devcontainer "$SCRIPT" 2>&1) || true

    # Should warn about both files
    # 両方のファイルについて警告するはず
    local has_env=false
    local has_apikey=false

    if echo "$output" | grep -qE "demo-app/\.env"; then
        has_env=true
    fi
    if echo "$output" | grep -qE "api-key\.txt"; then
        has_apikey=true
    fi

    if $has_env && $has_apikey; then
        pass "Both Claude and Gemini patterns detected"
    elif $has_env; then
        fail "Only Claude patterns detected, Gemini patterns missing"
        echo "Output: $output"
    elif $has_apikey; then
        fail "Only Gemini patterns detected, Claude patterns missing"
        echo "Output: $output"
    else
        fail "Neither Claude nor Gemini patterns detected"
        echo "Output: $output"
    fi

    cleanup
}

# Run all tests
# 全テストを実行
main() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  check-secret-sync.sh Test Suite"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    test_script_executable_and_valid
    test_all_synced
    test_missing_file_warning
    test_no_claude_settings
    test_no_matching_files
    test_directory_tmpfs_covered
    test_glob_pattern_matching
    test_suggests_sync_script
    test_cli_sandbox_env_uses_cli_compose
    test_cli_sandbox_env_synced
    test_aiexclude_patterns
    test_geminiignore_patterns
    test_combined_patterns

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

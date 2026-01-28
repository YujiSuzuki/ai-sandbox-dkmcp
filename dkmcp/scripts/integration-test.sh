#!/bin/bash
# integration-test.sh - DockMCP Integration Test Script
# DockMCP 連携テストスクリプト
#
# Usage / 使い方:
#   ./scripts/integration-test.sh [OPTIONS]
#
# Options / オプション:
#   --url URL     DockMCP server URL (default: http://host.docker.internal:8080)
#                 DockMCPサーバーURL（デフォルト: http://host.docker.internal:8080）
#   --container   Container name for tests (default: securenote-api)
#                 テスト用コンテナ名（デフォルト: securenote-api）
#   --skip-build  Skip building dkmcp binary
#                 dkmcpバイナリのビルドをスキップ
#   --help        Show this help message
#                 このヘルプメッセージを表示

# Note: Not using 'set -e' or 'set -o pipefail' because they cause issues with null bytes in output
# 注意: 出力内のnullバイトで問題が発生するため、'set -e' や 'set -o pipefail' は使用しない

# Colors for output
# 出力用の色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color / 色なし

# Default values
# デフォルト値
SERVER_URL="${DOCKMCP_SERVER_URL:-http://host.docker.internal:8080}"
CONTAINER="securenote-api"
SKIP_BUILD=false
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
DKMCP_BIN="/tmp/dkmcp-integration-test"

# Test counters
# テストカウンター
TESTS_PASSED=0
TESTS_FAILED=0

# Parse arguments
# 引数の解析
while [[ $# -gt 0 ]]; do
    case $1 in
        --url)
            SERVER_URL="$2"
            shift 2
            ;;
        --container)
            CONTAINER="$2"
            shift 2
            ;;
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --help)
            head -20 "$0" | tail -15
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Helper functions
# ヘルパー関数
print_header() {
    echo -e "\n${BLUE}==== $1 ====${NC}"
}

print_test() {
    echo -e "${YELLOW}[TEST]${NC} $1"
}

print_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

print_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_skip() {
    echo -e "${YELLOW}[SKIP]${NC} $1"
}

# Build dkmcp binary
# dkmcpバイナリをビルド
build_dkmcp() {
    print_header "Building dkmcp"

    if [ "$SKIP_BUILD" = true ]; then
        if [ -f "$DKMCP_BIN" ]; then
            print_skip "Using existing binary: $DKMCP_BIN"
            return 0
        else
            print_info "No existing binary found, building anyway..."
            # 既存のバイナリが見つからないため、ビルドを実行
        fi
    fi

    print_info "Building from $PROJECT_DIR..."
    cd "$PROJECT_DIR"

    if go build -o "$DKMCP_BIN" ./cmd/dkmcp 2>&1; then
        print_pass "Build successful: $DKMCP_BIN"
    else
        print_fail "Build failed"
        exit 1
    fi
}

# Check server connectivity
# サーバー接続を確認
check_server() {
    print_header "Checking DockMCP Server"
    print_info "Server URL: $SERVER_URL"

    print_test "Server connectivity"
    # サーバー接続テスト
    local list_output
    list_output=$($DKMCP_BIN client list --url "$SERVER_URL" 2>&1 | tr -d '\0') || true
    if echo "$list_output" | grep -qE "(NAME|ID|running)"; then
        print_pass "Server is reachable"
    else
        print_fail "Cannot connect to server at $SERVER_URL"
        echo ""
        echo "Make sure DockMCP server is running on host OS:"
        echo "ホストOSでDockMCPサーバーが起動していることを確認してください："
        echo "  cd dkmcp && dkmcp serve --config configs/dkmcp.example.yaml"
        exit 1
    fi

    print_test "Container '$CONTAINER' exists"
    # コンテナ存在確認
    if $DKMCP_BIN client list --url "$SERVER_URL" 2>/dev/null | tr -d '\0' | grep -q "$CONTAINER"; then
        print_pass "Container found"
    else
        print_fail "Container '$CONTAINER' not found"
        echo ""
        echo "Available containers:"
        echo "利用可能なコンテナ:"
        $DKMCP_BIN client list --url "$SERVER_URL" 2>/dev/null | tr -d '\0'
        exit 1
    fi
}

# Test 1: Exit code parsing - success case
# テスト1: 終了コード解析 - 成功ケース
test_exit_code_success() {
    print_header "Test: Exit Code Parsing (Success)"

    print_test "Command with exit code 0 should succeed"
    # 終了コード0のコマンドが成功すべき

    local output
    output=$($DKMCP_BIN client exec --url "$SERVER_URL" "$CONTAINER" "pwd" 2>&1 | tr -d '\0') || true
    local exit_status=$?

    # Check if response contains "Exit Code: 0"
    # レスポンスに "Exit Code: 0" が含まれているか確認
    if echo "$output" | grep -q "Exit Code: 0"; then
        print_pass "Response contains 'Exit Code: 0'"
    else
        print_fail "Response does not contain 'Exit Code: 0'"
        echo "Output: $output"
    fi

    # Check CLI exit status
    # CLIの終了ステータスを確認
    # Re-run to get actual exit status
    # 実際の終了ステータスを取得するために再実行
    $DKMCP_BIN client exec --url "$SERVER_URL" "$CONTAINER" "pwd" >/dev/null 2>&1
    exit_status=$?

    if [ $exit_status -eq 0 ]; then
        print_pass "CLI exit status is 0"
    else
        print_fail "CLI exit status is $exit_status (expected 0)"
    fi
}

# Test 2: Exit code parsing - failure case
# テスト2: 終了コード解析 - 失敗ケース
test_exit_code_failure() {
    print_header "Test: Exit Code Parsing (Failure)"

    print_test "Command with non-zero exit code should fail"
    # 非ゼロ終了コードのコマンドは失敗すべき

    # Use a whitelisted command that will fail (node with invalid syntax)
    # This doesn't require --dangerously flag
    # ホワイトリストにある失敗するコマンドを使用（無効なフラグ付きのnode）
    # --dangerouslyフラグは不要
    local output
    output=$($DKMCP_BIN client exec --url "$SERVER_URL" "$CONTAINER" "node --invalid-flag-xyz" 2>&1 | tr -d '\0') || true

    # Check if response contains non-zero exit code
    # レスポンスに非ゼロの終了コードが含まれているか確認
    if echo "$output" | grep -qE "Exit Code: [1-9]"; then
        print_pass "Response contains non-zero exit code"
    else
        # node --invalid-flag might return error differently
        # node --invalid-flag は別の形式でエラーを返す可能性がある
        if echo "$output" | grep -qiE "(error|bad option|unknown)"; then
            print_pass "Command failed as expected"
        else
            print_fail "Response does not contain non-zero exit code"
            echo "Output: $output"
        fi
    fi

    # Check CLI exit status
    # CLIの終了ステータスを確認
    $DKMCP_BIN client exec --url "$SERVER_URL" "$CONTAINER" "node --invalid-flag-xyz" >/dev/null 2>&1
    exit_status=$?

    if [ $exit_status -ne 0 ]; then
        print_pass "CLI exit status is non-zero ($exit_status)"
    else
        print_fail "CLI exit status is 0 (expected non-zero)"
    fi
}

# Test 3: Path blocking (if blocked paths are configured)
# テスト3: パスブロッキング（ブロックパスが設定されている場合）
test_path_blocking() {
    print_header "Test: Path Blocking"

    print_test "Access to blocked path should be denied"
    # ブロックされたパスへのアクセスは拒否されるべき

    # Try to access a typically blocked path (.env file)
    # 通常ブロックされるパス（.envファイル）へのアクセスを試行
    local output
    output=$($DKMCP_BIN client exec --url "$SERVER_URL" --dangerously "$CONTAINER" "cat /app/.env" 2>&1 | tr -d '\0') || true

    # Check if access was blocked or file doesn't exist
    # アクセスがブロックされたか、ファイルが存在しないか確認
    if echo "$output" | grep -qiE "(blocked|denied|not allowed|no such file)"; then
        print_pass "Blocked path access was denied or file not found"
    else
        # If we got content, it might be empty (which is OK for hidden files)
        # 内容を取得した場合、空かもしれない（隠しファイルの場合はOK）
        if echo "$output" | grep -q "Exit Code: 0" && [ -z "$(echo "$output" | grep -A1000 "Output:" | tail -n +2 | tr -d '[:space:]')" ]; then
            print_pass "File appears to be hidden (empty content)"
        else
            print_info "Path may not be blocked in current configuration"
            # 現在の設定ではパスがブロックされていない可能性
            echo "Output: $output"
        fi
    fi
}

# Test 4: Whitelisted command execution
# テスト4: ホワイトリストコマンドの実行
test_whitelisted_command() {
    print_header "Test: Whitelisted Command Execution"

    print_test "Whitelisted command should execute"
    # ホワイトリストのコマンドは実行されるべき

    local output
    output=$($DKMCP_BIN client exec --url "$SERVER_URL" "$CONTAINER" "node --version" 2>&1 | tr -d '\0') || true

    if echo "$output" | grep -q "Exit Code: 0"; then
        print_pass "Whitelisted command executed successfully"
        # Extract version from output
        # 出力からバージョンを抽出
        local version
        version=$(echo "$output" | grep -oE "v[0-9]+\.[0-9]+\.[0-9]+")
        if [ -n "$version" ]; then
            print_info "Node.js version: $version"
        fi
    else
        print_fail "Whitelisted command failed"
        echo "Output: $output"
    fi
}

# Test 5: Non-whitelisted command rejection
# テスト5: 非ホワイトリストコマンドの拒否
test_non_whitelisted_command() {
    print_header "Test: Non-Whitelisted Command Rejection"

    print_test "Non-whitelisted command should be rejected"
    # ホワイトリストにないコマンドは拒否されるべき

    local output
    output=$($DKMCP_BIN client exec --url "$SERVER_URL" "$CONTAINER" "rm -rf /" 2>&1 | tr -d '\0') || true
    local exit_status=$?

    if echo "$output" | grep -qiE "(not whitelisted|not allowed|rejected)"; then
        print_pass "Non-whitelisted command was rejected"
    else
        if [ $exit_status -ne 0 ]; then
            print_pass "Command failed (likely rejected)"
        else
            print_fail "Command may have been allowed!"
            echo "Output: $output"
        fi
    fi
}

# Test 6: Dangerously flag behavior
# テスト6: --dangerouslyフラグの動作
test_dangerously_flag() {
    print_header "Test: --dangerously Flag Behavior"

    # Test 6a: Without --dangerously flag, dangerous commands should be rejected
    # テスト6a: --dangerouslyフラグなしでは、dangerousコマンドは拒否されるべき
    print_test "Dangerous command without --dangerously flag should be rejected"

    local output
    output=$($DKMCP_BIN client exec --url "$SERVER_URL" "$CONTAINER" "cat /etc/passwd" 2>&1 | tr -d '\0') || true

    if echo "$output" | grep -qiE "(not whitelisted|not allowed|rejected|dangerous)"; then
        print_pass "Dangerous command rejected without flag"
    else
        # If server is running with --dangerously-all, this might succeed
        # サーバーが--dangerously-allで起動している場合、成功する可能性がある
        if echo "$output" | grep -q "Exit Code: 0"; then
            print_info "Command succeeded (server may have --dangerously-all enabled)"
            # コマンドが成功（サーバーが--dangerously-allで起動している可能性）
        else
            print_fail "Unexpected response"
            echo "Output: $output"
        fi
    fi

    # Test 6b: With --dangerously flag, dangerous commands should work (if server allows)
    # テスト6b: --dangerouslyフラグありでは、dangerousコマンドは動作すべき（サーバーが許可している場合）
    print_test "Dangerous command with --dangerously flag"

    output=$($DKMCP_BIN client exec --url "$SERVER_URL" --dangerously "$CONTAINER" "cat /etc/hostname" 2>&1 | tr -d '\0') || true

    if echo "$output" | grep -q "Exit Code: 0"; then
        print_pass "Dangerous command executed with flag"
    else
        if echo "$output" | grep -qiE "(not allowed|disabled|not enabled)"; then
            print_info "Dangerous mode not enabled on server (expected if server started without --dangerously flags)"
            # サーバーでdangerousモードが有効でない（サーバーが--dangerouslyフラグなしで起動された場合は想定通り）
        else
            print_fail "Dangerous command failed unexpectedly"
            echo "Output: $output"
        fi
    fi
}

# Test 7: Environment variable fallback
# テスト7: 環境変数のフォールバック
test_env_var_fallback() {
    print_header "Test: Environment Variable Fallback"

    # Test 7a: DOCKMCP_SERVER_URL should work without --url flag
    # テスト7a: DOCKMCP_SERVER_URLを設定すれば--urlフラグなしで接続できる
    print_test "DOCKMCP_SERVER_URL used when --url not specified"

    local output
    output=$(DOCKMCP_SERVER_URL="$SERVER_URL" $DKMCP_BIN client list 2>&1 | tr -d '\0') || true

    if echo "$output" | grep -qE "(NAME|ID|running)"; then
        print_pass "Connected via DOCKMCP_SERVER_URL"
    else
        print_fail "Failed to connect via DOCKMCP_SERVER_URL"
        echo "Output: $output"
    fi

    # Test 7b: --url flag takes precedence over DOCKMCP_SERVER_URL
    # テスト7b: --urlフラグがDOCKMCP_SERVER_URLより優先される
    print_test "--url flag takes precedence over DOCKMCP_SERVER_URL"

    output=$(DOCKMCP_SERVER_URL="http://invalid-host:9999" $DKMCP_BIN client list --url "$SERVER_URL" 2>&1 | tr -d '\0') || true

    if echo "$output" | grep -qE "(NAME|ID|running)"; then
        print_pass "Flag correctly overrode invalid env var"
    else
        print_fail "Flag did not override env var"
        echo "Output: $output"
    fi

    # Test 7c: DOCKMCP_CLIENT_SUFFIX works via env var
    # テスト7c: DOCKMCP_CLIENT_SUFFIXが環境変数で動作する
    print_test "DOCKMCP_CLIENT_SUFFIX accepted via env var"

    output=$(DOCKMCP_CLIENT_SUFFIX="integration-test" $DKMCP_BIN client list --url "$SERVER_URL" 2>&1 | tr -d '\0') || true

    if echo "$output" | grep -qE "(NAME|ID|running)"; then
        print_pass "Command succeeded with DOCKMCP_CLIENT_SUFFIX"
    else
        print_fail "Command failed with DOCKMCP_CLIENT_SUFFIX"
        echo "Output: $output"
    fi
}

# Print summary
# サマリーを表示
print_summary() {
    print_header "Test Summary"

    local total=$((TESTS_PASSED + TESTS_FAILED))

    echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
    echo -e "Total:  $total"

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "\n${GREEN}All tests passed!${NC}"
        return 0
    else
        echo -e "\n${RED}Some tests failed.${NC}"
        return 1
    fi
}

# Main
# メイン処理
main() {
    echo -e "${BLUE}"
    echo "╔═══════════════════════════════════════════╗"
    echo "║     DockMCP Integration Test Suite        ║"
    echo "║     DockMCP 連携テストスイート            ║"
    echo "╚═══════════════════════════════════════════╝"
    echo -e "${NC}"

    build_dkmcp
    check_server

    test_exit_code_success
    test_exit_code_failure
    test_path_blocking
    test_whitelisted_command
    test_non_whitelisted_command
    test_dangerously_flag
    test_env_var_fallback

    print_summary
}

main "$@"

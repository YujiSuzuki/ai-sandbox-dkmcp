#!/bin/bash
# host-access-integration-test.sh - DockMCP Host Access Integration Test (Host OS)
# DockMCP ホストアクセス連携テスト（ホストOS用）
#
# Tests host access features against a running DockMCP server with host_access enabled.
# host_access を有効にした起動中の DockMCP サーバーに対してホストアクセス機能をテストします。
#
# This test is designed to run on the HOST OS (not inside AI Sandbox).
# このテストはホストOS上で実行するように設計されています（AI Sandbox内ではなく）。
#
# Usage / 使い方:
#   ./scripts/host-access-integration-test.sh [OPTIONS]
#
# Options / オプション:
#   --url URL       DockMCP server URL (default: http://localhost:8080)
#                   DockMCPサーバーURL（デフォルト: http://localhost:8080）
#   --skip-build    Skip building dkmcp binary
#                   dkmcpバイナリのビルドをスキップ
#   --help          Show this help message
#                   このヘルプメッセージを表示
#
# Prerequisites / 前提条件:
#   - DockMCP server running with host_access enabled
#     host_access を有効にした DockMCP サーバーが起動中であること
#   - Example: dkmcp serve --config dkmcp.yaml --workspace /path/to/workspace
#     例: dkmcp serve --config dkmcp.yaml --workspace /path/to/workspace
#
# Temporary files created / 作成される一時ファイル:
#   /tmp/dkmcp-host-integ-test  - Test binary / テスト用バイナリ

# Colors for output
# 出力用の色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color / 色なし

# Default values
# デフォルト値
SERVER_URL="${DOCKMCP_SERVER_URL:-http://localhost:8080}"
SKIP_BUILD=false
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
DKMCP_BIN="/tmp/dkmcp-host-integ-test"

# Test counters
# テストカウンター
TESTS_PASSED=0
TESTS_FAILED=0

# Feature availability flags
# 機能利用可能フラグ
HAS_HOST_TOOLS=false
HAS_HOST_COMMANDS=false
HAS_DOCKER=false
HAS_GIT=false

# Parse arguments
# 引数の解析
while [[ $# -gt 0 ]]; do
    case $1 in
        --url)
            SERVER_URL="$2"
            shift 2
            ;;
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --help)
            head -28 "$0" | tail -23
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

# Check if running inside a container (abort if so)
# コンテナ内で実行されているかチェック（コンテナ内なら中止）
check_host_os() {
    if [ -f "/.dockerenv" ] || [ -n "${SANDBOX_ENV:-}" ]; then
        echo -e "${RED}Error: This script must be run on the host OS, not inside a container.${NC}"
        echo -e "${RED}エラー: このスクリプトはホストOS上で実行してください（コンテナ内では実行できません）。${NC}"
        exit 1
    fi
}

# Confirm before running tests
# テスト実行前の確認
confirm_run() {
    read -p "Run tests? / テストを実行しますか？ [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Cancelled. / キャンセルしました。"
        exit 0
    fi
}

# Show pre-run information (impact, risk, recovery)
# 実行前情報の表示（影響範囲、リスク、対処法）
show_prerun_info() {
    echo ""
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Impact / 影響範囲:${NC}"
    echo "  Connects to DockMCP server at $SERVER_URL"
    echo "  Builds test binary to $DKMCP_BIN"
    echo "  DockMCPサーバー（$SERVER_URL）に接続"
    echo "  テスト用バイナリを${DKMCP_BIN}に作成"
    echo ""
    echo -e "${YELLOW}Risk / リスク:${NC}"
    echo "  Low - Read-only client tests against an already running server"
    echo "  低 - 起動中のサーバーに対する読み取り中心のクライアントテスト"
    echo ""
    echo -e "${GREEN}Recovery / 失敗時の対処法:${NC}"
    echo "  Clean up binary / バイナリの削除:"
    echo "    rm -f $DKMCP_BIN"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

# Helper: run client command
# ヘルパー: クライアントコマンドを実行
run_client() {
    "$DKMCP_BIN" client "$@" --url "$SERVER_URL" 2>&1
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
        fi
    fi

    print_info "Building from $PROJECT_DIR..."
    cd "$PROJECT_DIR"

    if CGO_ENABLED=0 go build -o "$DKMCP_BIN" ./cmd/dkmcp 2>&1; then
        print_pass "Build successful: $DKMCP_BIN"
    else
        print_fail "Build failed"
        exit 1
    fi
}

# Check server connectivity and detect available features
# サーバー接続を確認し、利用可能な機能を検出
check_server() {
    print_header "Checking DockMCP Server"
    print_info "Server URL: $SERVER_URL"

    # Health check
    # ヘルスチェック
    print_test "Server health check"
    if curl -s "$SERVER_URL/health" | grep -q "ok"; then
        print_pass "Server is healthy"
    else
        print_fail "Cannot connect to server at $SERVER_URL"
        echo ""
        echo "Make sure DockMCP server is running with host_access enabled:"
        echo "host_access を有効にした DockMCP サーバーが起動していることを確認："
        echo "  dkmcp serve --config dkmcp.yaml --workspace /path/to/workspace"
        exit 1
    fi

    # Detect host tools availability
    # ホストツール利用可能性を検出
    print_test "Host tools feature"
    local output
    output=$(run_client host-tools list 2>&1) || true
    if echo "$output" | grep -q '"count"'; then
        HAS_HOST_TOOLS=true
        local count
        count=$(echo "$output" | grep -oE '"count": [0-9]+' | grep -oE '[0-9]+')
        print_pass "Host tools enabled ($count tools found)"
    elif echo "$output" | grep -qi "not configured"; then
        print_skip "Host tools not enabled on server"
    else
        print_skip "Host tools not available: $output"
    fi

    # Detect host commands availability
    # ホストコマンド利用可能性を検出
    print_test "Host commands feature"
    output=$(run_client host-exec "date" 2>&1) || true
    if echo "$output" | grep -q "Exit Code: 0"; then
        HAS_HOST_COMMANDS=true
        print_pass "Host commands enabled"
    elif echo "$output" | grep -qi "not configured"; then
        print_skip "Host commands not enabled on server"
    else
        print_skip "Host commands not available (date not whitelisted?)"
    fi

    # Detect Docker availability
    # Docker利用可能性を検出
    print_test "Docker on host"
    if command -v docker &>/dev/null && docker info &>/dev/null; then
        HAS_DOCKER=true
        print_pass "Docker available"
    else
        print_skip "Docker not available (docker-related tests will be skipped)"
    fi

    # Detect git availability
    # git利用可能性を検出
    print_test "Git on host"
    if command -v git &>/dev/null; then
        HAS_GIT=true
        print_pass "Git available"
    else
        print_skip "Git not available (git-related tests will be skipped)"
    fi
}


# ======================================================================
# Host Tools Tests
# ホストツールテスト
# ======================================================================

test_host_tools() {
    if [ "$HAS_HOST_TOOLS" != true ]; then
        print_header "Host Tools Tests (SKIPPED)"
        print_skip "Host tools not enabled on server"
        return
    fi

    print_header "Host Tools Tests"

    local output

    # List tools
    # ツール一覧
    print_test "list_host_tools returns tools"
    output=$(run_client host-tools list)
    if echo "$output" | grep -q '"tools"'; then
        print_pass "Tool list returned"
    else
        print_fail "Tool list failed: $output"
    fi

    # Get first tool name from list
    # リストから最初のツール名を取得
    local first_tool
    first_tool=$(echo "$output" | grep -oE '"name": "[^"]+"' | head -1 | grep -oE '"[^"]+\..*"' | tr -d '"')

    if [ -n "$first_tool" ]; then
        # Get tool info
        # ツール情報を取得
        print_test "get_host_tool_info for $first_tool"
        output=$(run_client host-tools info "$first_tool")
        if echo "$output" | grep -q '"description"'; then
            print_pass "Tool info returned"
        else
            print_fail "Tool info failed: $output"
        fi

        # Run tool (without args - verify dispatch works, exit code may be non-zero)
        # ツールを実行（引数なし - ディスパッチの動作確認、終了コードは問わない）
        print_test "run_host_tool $first_tool"
        output=$(run_client host-tools run "$first_tool")
        if echo "$output" | grep -q "Exit Code:"; then
            print_pass "Tool dispatched and executed"
        else
            print_fail "Tool execution failed: $output"
        fi
    else
        print_skip "No tools found to test info/run"
    fi

    # Security: path traversal
    # セキュリティ: パストラバーサル
    print_test "Path traversal rejected"
    output=$(run_client host-tools run "../../../etc/passwd" 2>&1) || true
    if echo "$output" | grep -qiE "invalid|traversal|error"; then
        print_pass "Path traversal rejected"
    else
        print_fail "Path traversal not rejected: $output"
    fi
}


# ======================================================================
# Host Commands Tests - Basic
# ホストコマンドテスト - 基本
# ======================================================================

test_host_commands_basic() {
    if [ "$HAS_HOST_COMMANDS" != true ]; then
        print_header "Host Commands Tests (SKIPPED)"
        print_skip "Host commands not enabled on server"
        return
    fi

    print_header "Host Commands Tests - Basic"

    local output

    # Non-whitelisted command
    # 非ホワイトリストコマンド
    print_test "Non-whitelisted command rejected"
    output=$(run_client host-exec "curl http://example.com" 2>&1) || true
    if echo "$output" | grep -qiE "not whitelisted|not allowed|error"; then
        print_pass "Non-whitelisted command rejected"
    else
        print_fail "Should be rejected: $output"
    fi

    # Pipe blocked
    # パイプブロック
    print_test "Pipe blocked"
    output=$(run_client host-exec "echo hello | cat" 2>&1) || true
    if echo "$output" | grep -q "shell meta-characters"; then
        print_pass "Pipe blocked"
    else
        print_fail "Pipe should be blocked: $output"
    fi

    # Redirect blocked
    # リダイレクトブロック
    print_test "Redirect blocked"
    output=$(run_client host-exec "echo hello > /tmp/x" 2>&1) || true
    if echo "$output" | grep -q "shell meta-characters"; then
        print_pass "Redirect blocked"
    else
        print_fail "Redirect should be blocked: $output"
    fi
}


# ======================================================================
# Host Commands Tests - Docker (host OS only)
# ホストコマンドテスト - Docker（ホストOS専用）
# ======================================================================

test_host_commands_docker() {
    if [ "$HAS_HOST_COMMANDS" != true ] || [ "$HAS_DOCKER" != true ]; then
        print_header "Docker Host Commands Tests (SKIPPED)"
        if [ "$HAS_HOST_COMMANDS" != true ]; then
            print_skip "Host commands not enabled"
        else
            print_skip "Docker not available"
        fi
        return
    fi

    print_header "Docker Host Commands Tests"

    local output

    # docker ps (commonly whitelisted)
    # docker ps（一般的にホワイトリスト登録済み）
    print_test "docker ps via host command"
    output=$(run_client host-exec "docker ps" 2>&1) || true
    if echo "$output" | grep -q "Exit Code: 0"; then
        print_pass "docker ps executed"
    elif echo "$output" | grep -q "not whitelisted"; then
        print_skip "docker ps not whitelisted on server"
    else
        print_fail "docker ps unexpected response: $output"
    fi

    # docker rm should be denied (if deny list configured)
    # docker rm は拒否されるべき（deny リストが設定されている場合）
    print_test "docker rm denied"
    output=$(run_client host-exec "docker rm test-container-xyz" 2>&1) || true
    if echo "$output" | grep -qiE "denied|not whitelisted|not allowed|error"; then
        print_pass "docker rm blocked"
    else
        print_fail "docker rm should be blocked: $output"
    fi

    # docker system prune should be denied
    # docker system prune は拒否されるべき
    print_test "docker system prune denied"
    output=$(run_client host-exec "docker system prune -f" 2>&1) || true
    if echo "$output" | grep -qiE "denied|not whitelisted|not allowed|error"; then
        print_pass "docker system prune blocked"
    else
        print_fail "docker system prune should be blocked: $output"
    fi
}


# ======================================================================
# Host Commands Tests - Git (host OS only)
# ホストコマンドテスト - Git（ホストOS専用）
# ======================================================================

test_host_commands_git() {
    if [ "$HAS_HOST_COMMANDS" != true ] || [ "$HAS_GIT" != true ]; then
        print_header "Git Host Commands Tests (SKIPPED)"
        if [ "$HAS_HOST_COMMANDS" != true ]; then
            print_skip "Host commands not enabled"
        else
            print_skip "Git not available"
        fi
        return
    fi

    print_header "Git Host Commands Tests"

    local output

    # git status (commonly whitelisted)
    # git status（一般的にホワイトリスト登録済み）
    print_test "git status via host command"
    output=$(run_client host-exec "git status" 2>&1) || true
    if echo "$output" | grep -q "Exit Code: 0"; then
        print_pass "git status executed"
    elif echo "$output" | grep -q "not whitelisted"; then
        print_skip "git status not whitelisted on server"
    else
        # git status might fail if workspace is not a git repo
        # ワークスペースがgitリポジトリでない場合は失敗する可能性
        if echo "$output" | grep -qi "not a git repository"; then
            print_info "Workspace is not a git repository (expected if workspace_root is not a git repo)"
        else
            print_fail "git status unexpected response: $output"
        fi
    fi

    # git diff (commonly whitelisted with wildcard)
    # git diff（一般的にワイルドカードでホワイトリスト登録済み）
    print_test "git diff via host command"
    output=$(run_client host-exec "git diff --stat" 2>&1) || true
    if echo "$output" | grep -q "Exit Code: 0"; then
        print_pass "git diff executed"
    elif echo "$output" | grep -q "not whitelisted"; then
        print_skip "git diff not whitelisted on server"
    else
        print_info "git diff result: $(echo "$output" | head -1)"
    fi

    # git checkout should require dangerously (if configured)
    # git checkout は dangerously が必要（設定されている場合）
    print_test "git checkout without dangerously"
    output=$(run_client host-exec "git checkout main" 2>&1) || true
    if echo "$output" | grep -qiE "not whitelisted|hint.*dangerously"; then
        print_pass "git checkout blocked without dangerously"
    elif echo "$output" | grep -q "Exit Code: 0"; then
        print_info "git checkout allowed (may be whitelisted on server)"
    else
        print_info "git checkout response: $(echo "$output" | head -1)"
    fi
}


# ======================================================================
# Host Commands Tests - Dangerous Mode
# ホストコマンドテスト - 危険モード
# ======================================================================

test_host_commands_dangerous() {
    if [ "$HAS_HOST_COMMANDS" != true ]; then
        print_header "Dangerous Mode Tests (SKIPPED)"
        print_skip "Host commands not enabled"
        return
    fi

    print_header "Dangerous Mode Tests"

    local output

    # Path traversal in dangerous mode
    # 危険モードでのパストラバーサル
    print_test "Path traversal blocked in dangerous mode"
    output=$(run_client host-exec --dangerously "touch /tmp/../etc/evil" 2>&1) || true
    if echo "$output" | grep -qiE "path traversal|not allowed|error"; then
        print_pass "Path traversal blocked"
    else
        print_fail "Path traversal should be blocked: $output"
    fi

    # Pipe in dangerous mode
    # 危険モードでのパイプ
    print_test "Pipe blocked in dangerous mode"
    output=$(run_client host-exec --dangerously "echo test | cat" 2>&1) || true
    if echo "$output" | grep -q "shell meta-characters"; then
        print_pass "Pipe blocked in dangerous mode"
    else
        print_fail "Pipe should be blocked: $output"
    fi
}


# ======================================================================
# Summary
# サマリー
# ======================================================================

print_summary() {
    print_header "Test Summary"

    local total=$((TESTS_PASSED + TESTS_FAILED))

    echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
    echo -e "Total:  $total"

    echo ""

    # Show feature detection results
    # 機能検出結果を表示
    echo -e "${BLUE}Features detected / 検出された機能:${NC}"
    echo -e "  Host tools:    $([ "$HAS_HOST_TOOLS" = true ] && echo -e "${GREEN}enabled${NC}" || echo -e "${YELLOW}disabled${NC}")"
    echo -e "  Host commands: $([ "$HAS_HOST_COMMANDS" = true ] && echo -e "${GREEN}enabled${NC}" || echo -e "${YELLOW}disabled${NC}")"
    echo -e "  Docker:        $([ "$HAS_DOCKER" = true ] && echo -e "${GREEN}available${NC}" || echo -e "${YELLOW}not available${NC}")"
    echo -e "  Git:           $([ "$HAS_GIT" = true ] && echo -e "${GREEN}available${NC}" || echo -e "${YELLOW}not available${NC}")"

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "\n${GREEN}All tests passed!${NC}"
        return 0
    else
        echo -e "\n${RED}Some tests failed.${NC}"
        echo ""
        echo -e "${GREEN}Recovery / 対処法:${NC}"
        echo -e "  1. Check server logs for errors / サーバーログでエラーを確認:"
        echo -e "     ${BLUE}journalctl -u dkmcp --since '5 minutes ago'${NC}"
        echo -e "     ${BLUE}# or if running manually / 手動起動の場合: tail -50 /path/to/dkmcp.log${NC}"
        echo ""
        echo -e "  2. Verify server config has host_access enabled / サーバー設定確認:"
        echo -e "     ${BLUE}grep -A5 'host_access:' dkmcp.yaml${NC}"
        echo ""
        echo -e "  3. Verify with / 確認コマンド:"
        echo -e "     ${BLUE}dkmcp client host-tools list --url $SERVER_URL${NC}"
        echo ""
        echo -e "  4. Clean up / 一時ファイルの削除:"
        echo -e "     ${BLUE}rm -f $DKMCP_BIN${NC}"
        return 1
    fi
}


# ======================================================================
# Main
# メイン処理
# ======================================================================

main() {
    echo -e "${BLUE}"
    echo "╔═══════════════════════════════════════════════════════════╗"
    echo "║   DockMCP Host Access Integration Test (Host OS)        ║"
    echo "║   DockMCP ホストアクセス連携テスト（ホストOS）          ║"
    echo "╚═══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"

    check_host_os
    show_prerun_info
    confirm_run

    build_dkmcp
    check_server

    test_host_tools
    test_host_commands_basic
    test_host_commands_docker
    test_host_commands_git
    test_host_commands_dangerous

    print_summary
}

main "$@"

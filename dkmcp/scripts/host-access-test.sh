#!/bin/bash
# host-access-test.sh - DockMCP Host Access Integration Test Script
# DockMCP ホストアクセス連携テストスクリプト
#
# This script tests the host access features:
# このスクリプトはホストアクセス機能をテストします：
#   - Host tool auto-discovery (list, info, run)
#     ホストツール自動発見（一覧、詳細、実行）
#   - Host CLI command execution (whitelist, deny, wildcard)
#     ホストCLIコマンド実行（ホワイトリスト、拒否、ワイルドカード）
#   - Security controls (pipes, path traversal, deny override)
#     セキュリティ制御（パイプ、パストラバーサル、拒否オーバーライド）
#   - Dangerous mode (hint, execution, restrictions)
#     危険モード（ヒント、実行、制限）
#   - CLI client fallback commands
#     CLIクライアントフォールバックコマンド
#
# Usage / 使い方:
#   ./scripts/host-access-test.sh [OPTIONS]
#
# Options / オプション:
#   --skip-build  Skip building dkmcp binary
#                 dkmcpバイナリのビルドをスキップ
#   --help        Show this help message
#                 このヘルプメッセージを表示
#
# Prerequisites / 前提条件:
#   - Go installed (for building)
#     Goがインストールされていること（ビルド用）
#   - Port 18082 available (test uses this port to avoid conflicts)
#     ポート18082が使用可能であること（競合回避のためこのポートを使用）
#
# Temporary files created / 作成される一時ファイル:
#   /tmp/dkmcp-host-access-test-bin   - Test binary / テスト用バイナリ
#   /tmp/dkmcp-host-access-test.log  - Server log output / サーバーログ出力
#   /tmp/dkmcp-host-access-test/     - Test workspace (config, tools) / テストワークスペース

# Colors for output
# 出力用の色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color / 色なし

# Configuration
# 設定
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
DKMCP_BIN="/tmp/dkmcp-host-access-test-bin"
TEST_PORT=18082
TEST_URL="http://127.0.0.1:$TEST_PORT"
LOG_FILE="/tmp/dkmcp-host-access-test.log"
TEST_WORKSPACE="/tmp/dkmcp-host-access-test"
CONFIG_FILE="$TEST_WORKSPACE/dkmcp.yaml"
SKIP_BUILD=false

# Test counters
# テストカウンター
TESTS_PASSED=0
TESTS_FAILED=0

# Server PID
# サーバーPID
SERVER_PID=""

# Parse arguments
# 引数の解析
while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --help)
            head -30 "$0" | tail -25
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

# Cleanup function
# クリーンアップ関数
cleanup() {
    if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
        print_info "Stopping server (PID: $SERVER_PID)..."
        kill -TERM "$SERVER_PID" 2>/dev/null
        sleep 3
        if kill -0 "$SERVER_PID" 2>/dev/null; then
            kill -KILL "$SERVER_PID" 2>/dev/null
        fi
    fi
    # Don't remove workspace here - leave for inspection on failure
    # 失敗時の調査用にワークスペースは残す
}

trap cleanup EXIT

# Build dkmcp binary
# dkmcpバイナリをビルド
build_dkmcp() {
    print_header "Building dkmcp"

    if [ "$SKIP_BUILD" = true ]; then
        if [ -f "$DKMCP_BIN" ]; then
            print_info "Using existing binary: $DKMCP_BIN"
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

# Create test workspace with tools and config
# テストワークスペースをツールと設定付きで作成
create_test_workspace() {
    print_header "Creating test workspace"

    rm -rf "$TEST_WORKSPACE"
    mkdir -p "$TEST_WORKSPACE/host-tools"

    # Create test host tools
    # テスト用ホストツールを作成
    cat > "$TEST_WORKSPACE/host-tools/hello.sh" << 'TOOL_EOF'
#!/bin/bash
# hello.sh
# A simple greeting tool for testing
# ---
# テスト用の簡単な挨拶ツール
echo "Hello from host tool!"
echo "Args: $@"
TOOL_EOF
    chmod +x "$TEST_WORKSPACE/host-tools/hello.sh"

    cat > "$TEST_WORKSPACE/host-tools/sysinfo.sh" << 'TOOL_EOF'
#!/bin/bash
# sysinfo.sh
# Show basic system information
# ---
# 基本的なシステム情報を表示
echo "Hostname: $(hostname)"
echo "User: $(whoami)"
TOOL_EOF
    chmod +x "$TEST_WORKSPACE/host-tools/sysinfo.sh"

    # Create an underscore-prefixed file (should be filtered out)
    # アンダースコアプレフィックス付きファイル（フィルタされるべき）
    cat > "$TEST_WORKSPACE/host-tools/_helper.sh" << 'TOOL_EOF'
#!/bin/bash
# _helper.sh
# Internal helper (should not appear in list)
echo "This should not be listed"
TOOL_EOF

    # Create a non-allowed extension file (should be filtered out by allowed_extensions)
    # 許可されていない拡張子のファイル（allowed_extensionsでフィルタされるべき）
    cat > "$TEST_WORKSPACE/host-tools/extra.py" << 'TOOL_EOF'
#!/usr/bin/env python3
# extra.py
# Python tool (not in allowed_extensions)
print("This should not be listed")
TOOL_EOF
    chmod +x "$TEST_WORKSPACE/host-tools/extra.py"

    # Create test config
    # テスト設定を作成
    cat > "$CONFIG_FILE" << EOF
server:
  port: $TEST_PORT
  host: "127.0.0.1"

security:
  mode: "moderate"
  allowed_containers: []
  exec_whitelist: {}
  exec_dangerously:
    enabled: false
    commands: {}
  permissions:
    logs: true
    inspect: true
    stats: true
    exec: true
  output_masking:
    enabled: false
  host_path_masking:
    enabled: false

host_access:
  workspace_root: "$TEST_WORKSPACE"

  host_tools:
    enabled: true
    directories:
      - "host-tools"
    allowed_extensions:
      - ".sh"
    timeout: 30

  host_commands:
    enabled: true
    whitelist:
      "echo":
        - "hello *"
        - "test"
      "date":
        - ""
      "uname":
        - "-a"
        - "-r"
      "whoami":
        - ""
    deny:
      "rm":
        - "*"
      "echo":
        - "dangerous *"
    dangerously:
      enabled: true
      commands:
        "touch":
          - "$TEST_WORKSPACE/created-by-dangerous.txt"
        "mkdir":
          - "-p"

logging:
  level: "info"
EOF

    print_info "Workspace: $TEST_WORKSPACE"
    print_info "Config: $CONFIG_FILE"
    print_pass "Test workspace created"
}

# Start server
# サーバーを起動
start_server() {
    print_header "Starting DockMCP Server"

    > "$LOG_FILE"

    print_info "Starting server on port $TEST_PORT..."

    "$DKMCP_BIN" serve --config "$CONFIG_FILE" --workspace "$TEST_WORKSPACE" > "$LOG_FILE" 2>&1 &
    SERVER_PID=$!

    print_info "Server PID: $SERVER_PID"

    # Wait for server to start
    # サーバーの起動を待機
    local retries=10
    while [ $retries -gt 0 ]; do
        if curl -s "$TEST_URL/health" > /dev/null 2>&1; then
            print_pass "Server started successfully"
            return 0
        fi
        sleep 0.5
        ((retries--))
    done

    print_fail "Server failed to start"
    cat "$LOG_FILE"
    exit 1
}

# Helper: run client command and capture output
# ヘルパー: クライアントコマンドを実行して出力をキャプチャ
run_client() {
    "$DKMCP_BIN" client "$@" --url "$TEST_URL" 2>&1
}


# ======================================================================
# Test: Host Tools
# テスト: ホストツール
# ======================================================================

test_host_tools_list() {
    print_header "Test: Host Tools - List"

    local output
    output=$(run_client host-tools list)

    # Should list 2 tools (hello.sh, sysinfo.sh), not _helper.sh
    # 2つのツールがリストされるべき（hello.sh, sysinfo.sh）、_helper.shは除外
    print_test "List returns tools"
    if echo "$output" | grep -q '"count": 2'; then
        print_pass "Tool count is 2"
    else
        print_fail "Expected count 2, got: $output"
    fi

    print_test "hello.sh is listed"
    if echo "$output" | grep -q '"name": "hello.sh"'; then
        print_pass "hello.sh found"
    else
        print_fail "hello.sh not found in: $output"
    fi

    print_test "sysinfo.sh is listed"
    if echo "$output" | grep -q '"name": "sysinfo.sh"'; then
        print_pass "sysinfo.sh found"
    else
        print_fail "sysinfo.sh not found in: $output"
    fi

    print_test "_helper.sh is filtered out"
    if echo "$output" | grep -q "_helper.sh"; then
        print_fail "_helper.sh should not be listed"
    else
        print_pass "_helper.sh correctly filtered"
    fi

    print_test "extra.py is filtered out (not in allowed_extensions)"
    if echo "$output" | grep -q "extra.py"; then
        print_fail "extra.py should not be listed (.py not in allowed_extensions)"
    else
        print_pass "extra.py correctly filtered by extension"
    fi
}

test_host_tools_info() {
    print_header "Test: Host Tools - Info"

    local output
    output=$(run_client host-tools info hello.sh)

    print_test "Info returns description"
    if echo "$output" | grep -q "A simple greeting tool for testing"; then
        print_pass "Description found"
    else
        print_fail "Description not found in: $output"
    fi
}

test_host_tools_run() {
    print_header "Test: Host Tools - Run"

    local output
    output=$(run_client host-tools run hello.sh world)

    print_test "Tool executes successfully"
    if echo "$output" | grep -q "Exit Code: 0"; then
        print_pass "Exit code 0"
    else
        print_fail "Non-zero exit code in: $output"
    fi

    print_test "Tool output contains greeting"
    if echo "$output" | grep -q "Hello from host tool!"; then
        print_pass "Greeting found"
    else
        print_fail "Greeting not found in: $output"
    fi

    print_test "Arguments are passed through"
    if echo "$output" | grep -q "Args: world"; then
        print_pass "Args passed correctly"
    else
        print_fail "Args not passed in: $output"
    fi
}

test_host_tools_security() {
    print_header "Test: Host Tools - Security"

    local output

    print_test "Path traversal is rejected"
    output=$(run_client host-tools run "../../../etc/passwd" 2>&1) || true
    if echo "$output" | grep -qi "invalid\|traversal\|rejected\|error"; then
        print_pass "Path traversal rejected"
    else
        print_fail "Path traversal not rejected: $output"
    fi

    print_test "Nonexistent tool returns error"
    output=$(run_client host-tools run nonexistent.sh 2>&1) || true
    if echo "$output" | grep -qi "not found\|error"; then
        print_pass "Nonexistent tool error"
    else
        print_fail "No error for nonexistent tool: $output"
    fi
}


# ======================================================================
# Test: Host Commands - Whitelist
# テスト: ホストコマンド - ホワイトリスト
# ======================================================================

test_host_commands_whitelist() {
    print_header "Test: Host Commands - Whitelist"

    local output

    print_test "Whitelisted command: date"
    output=$(run_client host-exec "date")
    if echo "$output" | grep -q "Exit Code: 0"; then
        print_pass "date executed"
    else
        print_fail "date failed: $output"
    fi

    print_test "Whitelisted command: whoami"
    output=$(run_client host-exec "whoami")
    if echo "$output" | grep -q "Exit Code: 0"; then
        print_pass "whoami executed"
    else
        print_fail "whoami failed: $output"
    fi

    print_test "Whitelisted command: uname -a"
    output=$(run_client host-exec "uname -a")
    if echo "$output" | grep -q "Exit Code: 0"; then
        print_pass "uname -a executed"
    else
        print_fail "uname -a failed: $output"
    fi

    print_test "Wildcard match: echo hello world"
    output=$(run_client host-exec "echo hello world")
    if echo "$output" | grep -q "hello world"; then
        print_pass "echo hello world matched wildcard"
    else
        print_fail "echo hello world failed: $output"
    fi
}

test_host_commands_arg_restriction() {
    print_header "Test: Host Commands - Argument Restriction"

    local output

    print_test "uname -r (whitelisted) succeeds"
    output=$(run_client host-exec "uname -r")
    if echo "$output" | grep -q "Exit Code: 0"; then
        print_pass "uname -r allowed"
    else
        print_fail "uname -r should be allowed: $output"
    fi

    print_test "uname -s (not whitelisted) is rejected"
    output=$(run_client host-exec "uname -s" 2>&1) || true
    if echo "$output" | grep -q "not whitelisted"; then
        print_pass "uname -s rejected"
    else
        print_fail "uname -s should be rejected: $output"
    fi

    print_test "echo test (exact match) succeeds"
    output=$(run_client host-exec "echo test")
    if echo "$output" | grep -q "Exit Code: 0"; then
        print_pass "echo test allowed"
    else
        print_fail "echo test should be allowed: $output"
    fi

    print_test "date with args rejected (empty args whitelist)"
    output=$(run_client host-exec "date --help" 2>&1) || true
    if echo "$output" | grep -q "not whitelisted"; then
        print_pass "date --help rejected (empty args = no args allowed)"
    else
        print_fail "date --help should be rejected: $output"
    fi
}


# ======================================================================
# Test: Host Commands - Security Controls
# テスト: ホストコマンド - セキュリティ制御
# ======================================================================

test_host_commands_deny() {
    print_header "Test: Host Commands - Deny List"

    local output

    print_test "Denied command: rm"
    output=$(run_client host-exec "rm -rf /tmp" 2>&1) || true
    if echo "$output" | grep -q "command denied"; then
        print_pass "rm denied"
    else
        print_fail "rm should be denied: $output"
    fi

    print_test "Deny overrides whitelist: echo dangerous test"
    output=$(run_client host-exec "echo dangerous test" 2>&1) || true
    if echo "$output" | grep -q "command denied"; then
        print_pass "echo dangerous denied (deny overrides whitelist)"
    else
        print_fail "echo dangerous should be denied: $output"
    fi
}

test_host_commands_not_whitelisted() {
    print_header "Test: Host Commands - Not Whitelisted"

    local output

    print_test "Non-whitelisted command: curl"
    output=$(run_client host-exec "curl http://example.com" 2>&1) || true
    if echo "$output" | grep -q "not whitelisted"; then
        print_pass "curl rejected"
    else
        print_fail "curl should be rejected: $output"
    fi
}

test_host_commands_pipe_redirect() {
    print_header "Test: Host Commands - Pipe/Redirect Block"

    local output

    print_test "Pipe blocked: echo hello | cat"
    output=$(run_client host-exec "echo hello | cat" 2>&1) || true
    if echo "$output" | grep -q "shell meta-characters"; then
        print_pass "Pipe blocked"
    else
        print_fail "Pipe should be blocked: $output"
    fi

    print_test "Redirect blocked: echo hello > /tmp/x"
    output=$(run_client host-exec "echo hello > /tmp/x" 2>&1) || true
    if echo "$output" | grep -q "shell meta-characters"; then
        print_pass "Redirect blocked"
    else
        print_fail "Redirect should be blocked: $output"
    fi

    print_test "Semicolon blocked: echo hello ; rm -rf /"
    output=$(run_client host-exec "echo hello ; rm -rf /" 2>&1) || true
    if echo "$output" | grep -q "shell meta-characters"; then
        print_pass "Semicolon blocked"
    else
        print_fail "Semicolon should be blocked: $output"
    fi

    print_test "Backtick blocked: echo \`whoami\`"
    output=$(run_client host-exec 'echo `whoami`' 2>&1) || true
    if echo "$output" | grep -q "shell meta-characters"; then
        print_pass "Backtick blocked"
    else
        print_fail "Backtick should be blocked: $output"
    fi

    print_test "Command substitution blocked: echo \$(whoami)"
    output=$(run_client host-exec 'echo $(whoami)' 2>&1) || true
    if echo "$output" | grep -q "shell meta-characters"; then
        print_pass "Command substitution blocked"
    else
        print_fail "Command substitution should be blocked: $output"
    fi

    print_test "AND chain blocked: echo hello && rm -rf /"
    output=$(run_client host-exec "echo hello && rm -rf /" 2>&1) || true
    if echo "$output" | grep -q "shell meta-characters"; then
        print_pass "AND chain blocked"
    else
        print_fail "AND chain should be blocked: $output"
    fi

    print_test "OR chain blocked: echo hello || rm -rf /"
    output=$(run_client host-exec "echo hello || rm -rf /" 2>&1) || true
    if echo "$output" | grep -q "shell meta-characters"; then
        print_pass "OR chain blocked"
    else
        print_fail "OR chain should be blocked: $output"
    fi
}


# ======================================================================
# Test: Host Commands - Dangerous Mode
# テスト: ホストコマンド - 危険モード
# ======================================================================

test_host_commands_dangerous_hint() {
    print_header "Test: Host Commands - Dangerous Mode Hint"

    local output

    print_test "Non-whitelisted but dangerous command shows hint"
    output=$(run_client host-exec "touch $TEST_WORKSPACE/created-by-dangerous.txt" 2>&1) || true
    if echo "$output" | grep -q "hint.*dangerously=true"; then
        print_pass "Hint message shown"
    else
        print_fail "Hint should be shown: $output"
    fi
}

test_host_commands_dangerous_exec() {
    print_header "Test: Host Commands - Dangerous Execution"

    local output

    print_test "touch with --dangerously succeeds"
    output=$(run_client host-exec --dangerously "touch $TEST_WORKSPACE/created-by-dangerous.txt")
    if echo "$output" | grep -q "DANGEROUS MODE"; then
        print_pass "Dangerous mode marker present"
    else
        print_fail "DANGEROUS MODE marker missing: $output"
    fi

    if echo "$output" | grep -q "Exit Code: 0"; then
        print_pass "touch executed successfully"
    else
        print_fail "touch failed: $output"
    fi

    print_test "File was actually created"
    if [ -f "$TEST_WORKSPACE/created-by-dangerous.txt" ]; then
        print_pass "File exists"
    else
        print_fail "File was not created"
    fi

    print_test "mkdir -p with --dangerously succeeds"
    output=$(run_client host-exec --dangerously "mkdir -p $TEST_WORKSPACE/testdir")
    if echo "$output" | grep -q "Exit Code: 0"; then
        print_pass "mkdir executed"
    else
        print_fail "mkdir failed: $output"
    fi

    if [ -d "$TEST_WORKSPACE/testdir" ]; then
        print_pass "Directory exists"
    else
        print_fail "Directory was not created"
    fi
}

test_host_commands_dangerous_restrictions() {
    print_header "Test: Host Commands - Dangerous Restrictions"

    local output

    print_test "Path traversal blocked in dangerous mode"
    output=$(run_client host-exec --dangerously "mkdir -p $TEST_WORKSPACE/../evil" 2>&1) || true
    if echo "$output" | grep -q "path traversal"; then
        print_pass "Path traversal blocked"
    else
        print_fail "Path traversal should be blocked: $output"
    fi

    print_test "Pipe blocked in dangerous mode"
    output=$(run_client host-exec --dangerously "touch $TEST_WORKSPACE/x | echo y" 2>&1) || true
    if echo "$output" | grep -q "shell meta-characters"; then
        print_pass "Pipe blocked in dangerous mode"
    else
        print_fail "Pipe should be blocked: $output"
    fi

    print_test "Non-dangerous command rejected with --dangerously"
    output=$(run_client host-exec --dangerously "curl http://example.com" 2>&1) || true
    if echo "$output" | grep -q "not allowed"; then
        print_pass "Non-dangerous command rejected"
    else
        print_fail "Non-dangerous command should be rejected: $output"
    fi
}


# ======================================================================
# Show pre-run information (impact, risk, recovery)
# 実行前情報の表示（影響範囲、リスク、対処法）
# ======================================================================

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

show_prerun_info() {
    echo ""
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Impact / 影響範囲:${NC}"
    echo "  Starts a test server on port $TEST_PORT, builds binary to /tmp"
    echo "  Creates test workspace at $TEST_WORKSPACE"
    echo "  テストサーバーをポート${TEST_PORT}で起動、バイナリを/tmpに作成"
    echo "  テストワークスペースを${TEST_WORKSPACE}に作成"
    echo ""
    echo -e "${YELLOW}Risk / リスク:${NC}"
    echo "  Low - Listens on 127.0.0.1 only, limited whitelist (echo, date, uname, whoami)"
    echo "  低 - 127.0.0.1のみ待ち受け、限定的なホワイトリスト（echo, date, uname, whoami）"
    echo ""
    echo -e "${GREEN}Recovery / 失敗時の対処法:${NC}"
    echo "  If server process remains / プロセス残存時:"
    echo "    kill \$(lsof -ti:$TEST_PORT) 2>/dev/null"
    echo ""
    echo "  Clean up / 削除:"
    echo "    rm -rf $TEST_WORKSPACE"
    echo "    rm -f $DKMCP_BIN $LOG_FILE"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}


# ======================================================================
# Print summary
# サマリーを表示
# ======================================================================

print_summary() {
    print_header "Test Summary"

    local total=$((TESTS_PASSED + TESTS_FAILED))

    echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
    echo -e "Total:  $total"

    echo ""
    print_info "Server log: $LOG_FILE"

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "\n${GREEN}All tests passed!${NC}"
        echo ""
        echo -e "${BLUE}Remaining temp files / 残存する一時ファイル:${NC}"
        echo -e "  $DKMCP_BIN          — test binary, reusable with --skip-build"
        echo -e "  $LOG_FILE   — server log for inspection"
        echo -e "  $TEST_WORKSPACE/    — test workspace"
        echo -e "  To clean up / 削除: ${BLUE}rm -rf /tmp/dkmcp-host-access-test*${NC}"
        return 0
    else
        echo -e "\n${RED}Some tests failed.${NC}"
        echo ""
        echo -e "${GREEN}Recovery / 対処法:${NC}"
        echo -e "  1. Check the full log / ログで詳細を確認:"
        echo -e "     ${BLUE}cat $LOG_FILE${NC}"
        echo ""
        echo -e "  2. If server process remains / サーバープロセスが残っている場合:"
        echo -e "     ${BLUE}kill \$(lsof -ti:$TEST_PORT) 2>/dev/null${NC}"
        echo ""
        echo -e "  3. If port is in use / ポートが使用中の場合:"
        echo -e "     ${BLUE}lsof -i:$TEST_PORT${NC}"
        echo ""
        echo -e "  4. Clean up / 一時ファイルの削除:"
        echo -e "     ${BLUE}rm -rf /tmp/dkmcp-host-access-test*${NC}"
        echo ""
        echo -e "  5. Retry with skip-build / ビルドスキップで再実行:"
        echo -e "     ${BLUE}$0 --skip-build${NC}"
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
    echo "║   DockMCP Host Access Integration Test Suite             ║"
    echo "║   DockMCP ホストアクセス連携テストスイート               ║"
    echo "╚═══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"

    check_host_os
    show_prerun_info
    confirm_run

    build_dkmcp
    create_test_workspace
    start_server

    # Host Tools tests
    # ホストツールテスト
    test_host_tools_list
    test_host_tools_info
    test_host_tools_run
    test_host_tools_security

    # Host Commands tests
    # ホストコマンドテスト
    test_host_commands_whitelist
    test_host_commands_arg_restriction
    test_host_commands_deny
    test_host_commands_not_whitelisted
    test_host_commands_pipe_redirect

    # Dangerous mode tests
    # 危険モードテスト
    test_host_commands_dangerous_hint
    test_host_commands_dangerous_exec
    test_host_commands_dangerous_restrictions

    print_summary
}

main "$@"

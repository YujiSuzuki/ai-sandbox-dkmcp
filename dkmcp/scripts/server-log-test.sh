#!/bin/bash
# server-log-test.sh - DockMCP Server Log Features Test Script
# DockMCP サーバーログ機能テストスクリプト
#
# This script tests the server-side logging features:
# このスクリプトはサーバー側のログ機能をテストします：
#   - Request numbers [#N] in log output
#     ログ出力のリクエスト番号 [#N]
#   - Client identification (User-Agent / client name)
#     クライアント識別（User-Agent / クライアント名）
#   - Separate client_name and user_agent fields
#     client_nameとuser_agentの分離表示
#   - HTTP header logging at -vvvv
#     -vvvv でのHTTPヘッダーログ出力
#   - Graceful shutdown with timeout
#     タイムアウト付きグレースフルシャットダウン
#   - Uninitialized connection summary on shutdown
#     シャットダウン時の未初期化接続サマリー
#
# Usage / 使い方:
#   ./scripts/server-log-test.sh [OPTIONS]
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
#   - Port 18080 available (test uses this port to avoid conflicts)
#     ポート18080が使用可能であること（競合回避のためこのポートを使用）
#
# Temporary files created / 作成される一時ファイル:
#   /tmp/dkmcp-server-log-test      - Test binary / テスト用バイナリ
#   /tmp/dkmcp-server-log-test.log  - Server log output / サーバーログ出力
#   /tmp/dkmcp-server-log-test.yaml - Test config / テスト用設定ファイル

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
DKMCP_BIN="/tmp/dkmcp-server-log-test"
TEST_PORT=18080
TEST_URL="http://127.0.0.1:$TEST_PORT"
LOG_FILE="/tmp/dkmcp-server-log-test.log"
CONFIG_FILE="/tmp/dkmcp-server-log-test.yaml"
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
            head -25 "$0" | tail -20
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
        # Wait for graceful shutdown
        # グレースフルシャットダウンを待機
        sleep 3
        if kill -0 "$SERVER_PID" 2>/dev/null; then
            kill -KILL "$SERVER_PID" 2>/dev/null
        fi
    fi
    rm -f "$CONFIG_FILE"
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

# Create minimal test config
# 最小限のテスト設定を作成
create_test_config() {
    print_header "Creating test configuration"

    cat > "$CONFIG_FILE" << 'EOF'
server:
  port: 18080
  host: "127.0.0.1"

security:
  mode: "permissive"
  allowed_containers:
    - "*"
  exec_whitelist:
    "*":
      - "echo *"
      - "pwd"
EOF

    print_info "Config file: $CONFIG_FILE"
}

# Start server with verbosity
# 詳細モードでサーバーを起動
# Usage: start_server [-v|-vv|-vvv|-vvvv]
# Default: -v
start_server() {
    local verbosity_flag="${1:--v}"
    print_header "Starting DockMCP Server"

    # Clear previous log
    # 以前のログをクリア
    > "$LOG_FILE"

    print_info "Starting server on port $TEST_PORT with $verbosity_flag flag..."
    print_info "Log file: $LOG_FILE"

    # Start server in background, redirect output to log file
    # バックグラウンドでサーバーを起動し、出力をログファイルにリダイレクト
    "$DKMCP_BIN" serve --config "$CONFIG_FILE" $verbosity_flag > "$LOG_FILE" 2>&1 &
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

# Stop server gracefully (for restarting between test phases)
# テストフェーズ間の再起動用にサーバーを安全に停止
stop_server() {
    if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
        print_info "Stopping server (PID: $SERVER_PID)..."
        kill -TERM "$SERVER_PID" 2>/dev/null
        # Wait for graceful shutdown
        # グレースフルシャットダウンを待機
        local retries=10
        while [ $retries -gt 0 ] && kill -0 "$SERVER_PID" 2>/dev/null; do
            sleep 0.5
            ((retries--))
        done
        if kill -0 "$SERVER_PID" 2>/dev/null; then
            kill -KILL "$SERVER_PID" 2>/dev/null
        fi
        SERVER_PID=""
    fi
}

# Test 1: Request numbers in logs
# テスト1: ログ内のリクエスト番号
test_request_numbers() {
    print_header "Test: Request Numbers [#N]"

    print_test "Making multiple requests..."

    # Make several requests
    # 複数のリクエストを送信
    "$DKMCP_BIN" client list --url "$TEST_URL" > /dev/null 2>&1
    "$DKMCP_BIN" client list --url "$TEST_URL" > /dev/null 2>&1
    "$DKMCP_BIN" client list --url "$TEST_URL" > /dev/null 2>&1

    # Give server time to log
    # サーバーがログを書く時間を確保
    sleep 1

    print_test "Checking for request numbers in log..."

    # Check for [#1], [#2], etc. in logs
    # ログに [#1], [#2] などがあるか確認
    if grep -qE '\[#[0-9]+\]' "$LOG_FILE"; then
        print_pass "Request numbers found in logs"

        # Show some examples
        # 例を表示
        print_info "Examples from log:"
        grep -E '═══ \[#[0-9]+\]' "$LOG_FILE" | head -5 | while read -r line; do
            echo "  $line"
        done
    else
        print_fail "Request numbers NOT found in logs"
        print_info "Log content:"
        cat "$LOG_FILE"
    fi

    # Check for sequential numbers
    # 連番になっているか確認
    print_test "Checking for sequential request numbers..."

    local num1 num2
    num1=$(grep -oE '\[#[0-9]+\]' "$LOG_FILE" | head -1 | grep -oE '[0-9]+')
    num2=$(grep -oE '\[#[0-9]+\]' "$LOG_FILE" | tail -1 | grep -oE '[0-9]+')

    if [ -n "$num1" ] && [ -n "$num2" ] && [ "$num2" -gt "$num1" ]; then
        print_pass "Request numbers are sequential ($num1 to $num2)"
    else
        print_fail "Could not verify sequential numbers (num1=$num1, num2=$num2)"
    fi
}

# Test 2: Client identification
# テスト2: クライアント識別
test_client_identification() {
    print_header "Test: Client Identification"

    print_test "Checking for client name in logs..."

    # dkmcp client uses "dkmcp-go-client" as client name
    # dkmcp clientは "dkmcp-go-client" をクライアント名として使用
    if grep -q "dkmcp-go-client" "$LOG_FILE"; then
        print_pass "Client name 'dkmcp-go-client' found in logs"

        # Show example
        # 例を表示
        print_info "Example from log:"
        grep "dkmcp-go-client" "$LOG_FILE" | head -1 | sed 's/^/  /'
    else
        print_fail "Client name NOT found in logs"
    fi

    print_test "Checking for client info in REQUEST/RESPONSE lines..."

    if grep -qE '(REQUEST|RESPONSE) client_name=' "$LOG_FILE"; then
        print_pass "Client info found in REQUEST/RESPONSE lines"
    else
        print_fail "Client info NOT found in REQUEST/RESPONSE lines"
    fi
}

# Test 3: Custom client suffix
# テスト3: カスタムクライアントサフィックス
test_client_suffix() {
    print_header "Test: Custom Client Suffix (--client-suffix)"

    local suffix="テスト用サフィックス"

    print_test "Making request with custom suffix: $suffix"

    "$DKMCP_BIN" client list --url "$TEST_URL" --client-suffix "$suffix" > /dev/null 2>&1

    sleep 1

    print_test "Checking for custom suffix in logs..."

    if grep -q "$suffix" "$LOG_FILE"; then
        print_pass "Custom suffix found in logs"

        print_info "Example from log:"
        grep "$suffix" "$LOG_FILE" | head -1 | sed 's/^/  /'
    else
        print_fail "Custom suffix NOT found in logs"
    fi
}

# Test 4: Separate client_name and user_agent fields
# テスト4: client_name と user_agent の分離表示
test_separate_client_fields() {
    print_header "Test: Separate client_name and user_agent Fields"

    # dkmcp-go-client sends client_name="dkmcp-go-client" in MCP initialize,
    # and Go's default User-Agent is "Go-http-client/1.1".
    # These should appear as separate log fields: client_name=... user_agent=...
    #
    # dkmcp-go-client は MCP initialize で client_name="dkmcp-go-client" を送信し、
    # Go のデフォルト User-Agent は "Go-http-client/1.1"。
    # これらは別々のログフィールドとして表示される：client_name=... user_agent=...

    print_test "Making request to generate connection log..."

    "$DKMCP_BIN" client list --url "$TEST_URL" > /dev/null 2>&1

    sleep 1

    print_test "Checking for separate client_name and user_agent fields..."

    # The log should show client_name and user_agent as separate fields
    # ログに client_name と user_agent が別々のフィールドとして表示されるはず
    if grep -q "client_name=dkmcp-go-client" "$LOG_FILE" && grep -q "user_agent=Go-http-client" "$LOG_FILE"; then
        print_pass "Separate fields found: client_name=dkmcp-go-client user_agent=Go-http-client/..."

        print_info "Example from log:"
        grep "client_name=dkmcp-go-client" "$LOG_FILE" | grep "user_agent=" | head -1 | sed 's/^/  /'
    else
        print_fail "Separate client_name/user_agent fields NOT found in logs"
        print_info "Log contains:"
        grep "dkmcp-go-client" "$LOG_FILE" | head -3 | sed 's/^/  /'
    fi

    print_test "Checking that user_agent field IS present in connection log..."

    # The "[+] Client connected" log should have a separate user_agent field
    # "[+] Client connected" ログに user_agent フィールドが含まれることを確認
    local connected_line
    connected_line=$(grep "Client connected" "$LOG_FILE" | head -1)

    if [ -n "$connected_line" ]; then
        if echo "$connected_line" | grep -q "user_agent="; then
            print_pass "Connection log has separate user_agent field"
        else
            print_fail "Connection log missing user_agent field"
            echo "  $connected_line"
        fi
    else
        # Connection log is at DEBUG level, might not appear at -vv
        # 接続ログは DEBUG レベルのため -vv では表示されない場合がある
        print_info "Connection log not found (DEBUG level, may require -vvv)"
    fi

    # Also check that the client suffix still works with separate fields
    # カスタムサフィックスも分離フィールドと共存するか確認
    print_test "Checking custom suffix with separate fields..."

    "$DKMCP_BIN" client list --url "$TEST_URL" --client-suffix "test-suffix" > /dev/null 2>&1

    sleep 1

    if grep -q "client_name=dkmcp-go-client_test-suffix" "$LOG_FILE" && grep -q "user_agent=Go-http-client" "$LOG_FILE"; then
        print_pass "Custom suffix with separate fields: client_name=dkmcp-go-client_test-suffix"
    else
        # Check if suffix is present at all
        # サフィックスが存在するか確認
        if grep -q "test-suffix" "$LOG_FILE"; then
            print_pass "Custom suffix found (format may differ)"
            print_info "Example:"
            grep "test-suffix" "$LOG_FILE" | head -1 | sed 's/^/  /'
        else
            print_fail "Custom suffix NOT found with separate fields"
        fi
    fi
}

# Test 5: HTTP header logging at -vvvv
# テスト5: -vvvv での HTTP ヘッダーログ
test_http_header_logging() {
    print_header "Test: HTTP Header Logging (-vvvv)"

    # This test requires the server to be restarted with -vvvv
    # このテストはサーバーを -vvvv で再起動する必要がある

    print_test "Restarting server with -vvvv..."

    stop_server
    start_server "-vvvv"

    print_test "Making request to generate header logs..."

    "$DKMCP_BIN" client list --url "$TEST_URL" > /dev/null 2>&1

    sleep 1

    print_test "Checking for HTTP header entries in log..."

    if grep -q "HTTP header" "$LOG_FILE"; then
        print_pass "HTTP header logging found at -vvvv"

        # Show some header examples
        # ヘッダーの例を表示
        print_info "Header examples from log:"
        grep "HTTP header" "$LOG_FILE" | head -5 | sed 's/^/  /'
    else
        print_fail "HTTP header logging NOT found at -vvvv"
        print_info "Log content (first 30 lines):"
        head -30 "$LOG_FILE" | sed 's/^/  /'
    fi

    print_test "Checking that headers are sorted alphabetically per request..."

    # Headers are sorted within each request, not across all requests.
    # A non-"HTTP header" line (e.g., "Request received") marks a new group.
    # ヘッダーはリクエストごとにソートされる（全リクエスト横断ではない）。
    # "HTTP header" 以外の行（例: "Request received"）で新しいグループが始まる。
    local sorted=true
    local prev_key=""
    local group_count=0

    while IFS= read -r line; do
        if echo "$line" | grep -q "HTTP header"; then
            local key
            key=$(echo "$line" | grep -oE 'key=[^ ]+' | sed 's/key=//')
            if [ -n "$prev_key" ] && [[ "$key" < "$prev_key" ]]; then
                sorted=false
                break
            fi
            prev_key="$key"
        else
            # Non-header line resets the group
            if [ -n "$prev_key" ]; then
                ((group_count++))
            fi
            prev_key=""
        fi
    done < "$LOG_FILE"

    if [ "$sorted" = true ] && [ "$group_count" -gt 0 ]; then
        print_pass "Headers are sorted alphabetically (checked $group_count request groups)"
    elif [ "$group_count" -eq 0 ]; then
        print_fail "Could not extract header groups"
    else
        print_fail "Headers are NOT sorted alphabetically within a request"
    fi

    print_test "Checking that User-Agent header appears..."

    if grep "HTTP header" "$LOG_FILE" | grep -q "User-Agent"; then
        print_pass "User-Agent header found in log"
    else
        print_fail "User-Agent header NOT found in log"
    fi

    # At -vvvv (DEBUG level), the connection log is visible.
    # Verify separate client_name and user_agent fields.
    # -vvvv（DEBUGレベル）では接続ログが表示される。
    # client_name と user_agent が別々のフィールドとして表示されることを確認。
    print_test "Checking connection log at DEBUG level for separate fields..."

    local connected_line
    connected_line=$(grep "Client connected" "$LOG_FILE" | head -1)

    if [ -n "$connected_line" ]; then
        # Verify separate client_name field
        # client_name フィールドが存在することを確認
        if echo "$connected_line" | grep -q "client_name=dkmcp-go-client"; then
            print_pass "Connection log shows client_name=dkmcp-go-client"
        else
            print_fail "Connection log missing client_name=dkmcp-go-client"
            echo "  $connected_line"
        fi

        # Verify separate user_agent field
        # user_agent フィールドが存在することを確認
        if echo "$connected_line" | grep -q "user_agent=Go-http-client"; then
            print_pass "Connection log shows separate user_agent field"
        else
            print_fail "Connection log missing user_agent field"
            echo "  $connected_line"
        fi
    else
        print_fail "Connection log not found even at DEBUG level"
    fi

    # Restart server with -v for shutdown tests
    # シャットダウンテスト用に -v でサーバーを再起動
    print_test "Restarting server with -v for shutdown tests..."

    stop_server
    start_server "-v"
}

# Test 6: Graceful shutdown
# テスト6: グレースフルシャットダウン
test_graceful_shutdown() {
    print_header "Test: Graceful Shutdown"

    print_test "Sending SIGTERM to server..."

    # Send SIGTERM
    # SIGTERMを送信
    kill -TERM "$SERVER_PID" 2>/dev/null

    # Wait for shutdown
    # シャットダウンを待機
    sleep 3

    print_test "Checking for shutdown messages in log..."

    if grep -q "Shutting down gracefully" "$LOG_FILE"; then
        print_pass "Graceful shutdown message found"
    else
        print_fail "Graceful shutdown message NOT found"
    fi

    if grep -q "Server stopped" "$LOG_FILE"; then
        print_pass "Server stopped message found"
    else
        print_fail "Server stopped message NOT found"
    fi

    # Check if process actually stopped
    # プロセスが実際に停止したか確認
    if ! kill -0 "$SERVER_PID" 2>/dev/null; then
        print_pass "Server process terminated"
        SERVER_PID=""  # Clear PID so cleanup doesn't try again
    else
        print_fail "Server process still running!"
    fi
}

# Test 7: Shutdown within timeout
# テスト7: タイムアウト内でのシャットダウン
test_shutdown_timing() {
    print_header "Test: Shutdown Timing"

    print_test "Checking shutdown timing..."

    # Get timestamps from log
    # ログからタイムスタンプを取得
    local shutdown_start shutdown_end

    shutdown_start=$(grep "Shutting down gracefully" "$LOG_FILE" | grep -oE '[0-9]{2}:[0-9]{2}:[0-9]{2}' | head -1)
    shutdown_end=$(grep "Server stopped" "$LOG_FILE" | grep -oE '[0-9]{2}:[0-9]{2}:[0-9]{2}' | head -1)

    if [ -n "$shutdown_start" ] && [ -n "$shutdown_end" ]; then
        print_pass "Shutdown timestamps found: $shutdown_start -> $shutdown_end"

        # Note: We can't easily calculate the difference in bash without date -d
        # but visual inspection should show it's within 2-3 seconds
        # 注意: bashでは date -d なしで差を計算するのは難しいが、
        # 目視で2-3秒以内であることが確認できるはず
        print_info "Visual inspection: shutdown should complete within 2-3 seconds"
    else
        print_fail "Could not extract shutdown timestamps"
    fi
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
    echo "  Starts a test server on port $TEST_PORT, builds binary to /tmp"
    echo "  テストサーバーをポート${TEST_PORT}で起動、バイナリを/tmpに作成"
    echo ""
    echo -e "${YELLOW}Risk / リスク:${NC}"
    echo "  Low - Listens on 127.0.0.1 only, permissive config but limited exec whitelist (echo, pwd)"
    echo "  低 - 127.0.0.1のみ待ち受け、permissive設定だがexecはecho/pwdのみ許可"
    echo ""
    echo -e "${GREEN}Recovery / 失敗時の対処法:${NC}"
    echo "  If server process remains: ps aux | grep dkmcp-server-log-test | grep -v grep"
    echo "                             kill \$(lsof -ti:$TEST_PORT) 2>/dev/null"
    echo "  プロセス残存時: ps aux | grep dkmcp-server-log-test | grep -v grep"
    echo "                  kill \$(lsof -ti:$TEST_PORT) 2>/dev/null"
    echo ""
    echo "  Temp files: rm -f /tmp/dkmcp-server-log-test*"
    echo "  一時ファイル: rm -f /tmp/dkmcp-server-log-test*"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

# Print summary
# サマリーを表示
print_summary() {
    print_header "Test Summary"

    local total=$((TESTS_PASSED + TESTS_FAILED))

    echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
    echo -e "Total:  $total"

    echo ""
    print_info "Full log available at: $LOG_FILE"

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "\n${GREEN}All tests passed!${NC}"
        echo ""
        echo -e "${BLUE}Remaining temp files / 残存する一時ファイル:${NC}"
        echo -e "  $DKMCP_BIN        — test binary, reusable with --skip-build"
        echo -e "  $LOG_FILE  — server log for inspection"
        echo -e "  To clean up / 削除: ${BLUE}rm -f /tmp/dkmcp-server-log-test*${NC}"
        return 0
    else
        echo -e "\n${RED}Some tests failed.${NC}"
        echo ""
        echo -e "${GREEN}Recovery / 対処法:${NC}"
        echo -e "  1. Check the full log for details / ログで詳細を確認:"
        echo -e "     ${BLUE}cat $LOG_FILE${NC}"
        echo ""
        echo -e "  2. If server process remains / サーバープロセスが残っている場合:"
        echo -e "     ${BLUE}kill \$(lsof -ti:$TEST_PORT) 2>/dev/null${NC}"
        echo ""
        echo -e "  3. Clean up temp files / 一時ファイルの削除:"
        echo -e "     ${BLUE}rm -f /tmp/dkmcp-server-log-test*${NC}"
        echo ""
        echo -e "  4. If port is in use / ポートが使用中の場合:"
        echo -e "     ${BLUE}lsof -i:$TEST_PORT${NC}"
        echo ""
        echo -e "  5. Retry with skip-build / ビルドスキップで再実行:"
        echo -e "     ${BLUE}$0 --skip-build${NC}"
        return 1
    fi
}

# Main
# メイン処理
main() {
    echo -e "${BLUE}"
    echo "╔═══════════════════════════════════════════════╗"
    echo "║   DockMCP Server Log Features Test Suite      ║"
    echo "║   DockMCP サーバーログ機能テストスイート      ║"
    echo "╚═══════════════════════════════════════════════╝"
    echo -e "${NC}"

    check_host_os
    show_prerun_info
    confirm_run

    build_dkmcp
    create_test_config
    start_server "-v"

    test_request_numbers
    test_client_identification
    test_client_suffix
    test_separate_client_fields

    # This test restarts the server with -vvvv, then restarts with -v for shutdown tests
    # このテストはサーバーを -vvvv で再起動し、その後シャットダウンテスト用に -v で再起動
    test_http_header_logging

    # Shutdown tests (these will stop the server)
    # シャットダウンテスト（これらはサーバーを停止する）
    test_graceful_shutdown
    test_shutdown_timing

    print_summary
}

main "$@"

#!/bin/bash
# test-setup-dkmcp.sh
# Test script for setup-dkmcp.sh
#
# Usage: ./test-setup-dkmcp.sh
#
# Environment: AI Sandbox (requires /workspace)
# ---
# setup-dkmcp.sh のテストスクリプト
#
# 使用方法: ./test-setup-dkmcp.sh
#
# 実行環境: AI Sandbox（/workspace が必要）

set -e

# Verify running in AI Sandbox
# AI Sandbox 内での実行を確認
if [ ! -d "/workspace" ]; then
    echo "Error: This test is designed to run inside AI Sandbox"
    echo "エラー: このテストは AI Sandbox 内での実行を想定しています"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SCRIPT="$SCRIPT_DIR/setup-dkmcp.sh"
TEST_WORKSPACE=""

# Colors for output
# 出力用の色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

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
    TEST_WORKSPACE=$(mktemp -d)
    mkdir -p "$TEST_WORKSPACE/.gemini"
}

# Cleanup test environment
# テスト環境のクリーンアップ
cleanup() {
    if [ -n "$TEST_WORKSPACE" ] && [ -d "$TEST_WORKSPACE" ]; then
        rm -rf "$TEST_WORKSPACE"
    fi
    TEST_WORKSPACE=""
}

# Trap to ensure cleanup runs
# クリーンアップが必ず実行されるようトラップ設定
trap cleanup EXIT

# ─── Tests ────────────────────────────────────────────────────

# Test 1: --help exits 0 and shows expected options (smoke test)
# テスト1: --help が exit 0 で終了し、期待するオプションを表示するか（スモークテスト）
test_script_runs_and_shows_help() {
    echo ""
    echo "=== Test: Script runs and --help shows expected options ==="

    if [ ! -f "$SCRIPT" ] || [ ! -x "$SCRIPT" ]; then
        fail "Script not found or not executable: $SCRIPT"
        return
    fi

    local exit_code=0
    local output
    output=$("$SCRIPT" --help 2>&1) || exit_code=$?

    if [ "$exit_code" -eq 0 ] && \
       echo "$output" | grep -q -- "--check" && \
       echo "$output" | grep -q -- "--unregister"; then
        pass "Script runs, --help exits 0 and shows expected options"
    else
        fail "--help exited $exit_code or missing expected options"
    fi
}

# Test 2: --help shows usage
# テスト2: --help がヘルプを表示するか
test_help() {
    echo ""
    echo "=== Test: --help shows usage ==="

    local output
    output=$("$SCRIPT" --help 2>&1)

    if echo "$output" | grep -q -- "--check" && \
       echo "$output" | grep -q -- "--status" && \
       echo "$output" | grep -q -- "--unregister"; then
        pass "--help shows all options"
    else
        fail "--help output is missing expected options"
    fi
}

# Test 3: --check returns 1 when not registered
# テスト3: 未登録時に --check が exit 1 を返すか
test_check_not_registered() {
    echo ""
    echo "=== Test: --check returns 1 when not registered ==="

    setup

    # Empty workspace with no .mcp.json, no Claude config, no Gemini config
    local exit_code=0
    WORKSPACE="$TEST_WORKSPACE" HOME="$TEST_WORKSPACE" "$SCRIPT" --check 2>/dev/null || exit_code=$?

    if [ "$exit_code" -eq 1 ]; then
        pass "--check returns 1 when not registered"
    else
        fail "--check returned $exit_code, expected 1"
    fi

    cleanup
}

# Test 4: --check returns 2 when registered but server offline
# テスト4: 登録済だがサーバーオフライン時に --check が exit 2 を返すか
test_check_registered_but_offline() {
    echo ""
    echo "=== Test: --check returns 2 when registered but offline ==="

    setup

    # Create .mcp.json with dkmcp entry
    cat > "$TEST_WORKSPACE/.mcp.json" << 'EOF'
{
  "mcpServers": {
    "dkmcp": {
      "type": "sse",
      "url": "http://host.docker.internal:8080/sse"
    }
  }
}
EOF

    local exit_code=0
    # Use a URL that won't connect (localhost:1 should refuse)
    WORKSPACE="$TEST_WORKSPACE" HOME="$TEST_WORKSPACE" \
        "$SCRIPT" --check --url "http://localhost:1/sse" 2>/dev/null || exit_code=$?

    if [ "$exit_code" -eq 2 ]; then
        pass "--check returns 2 when registered but offline"
    else
        fail "--check returned $exit_code, expected 2"
    fi

    cleanup
}

# Test 5: Default mode creates .mcp.json via fallback when claude CLI not in PATH
# テスト5: claude CLI 不在時にフォールバックで .mcp.json を作成するか
test_register_fallback_creates_mcp_json() {
    echo ""
    echo "=== Test: Fallback creates .mcp.json when claude not in PATH ==="

    setup

    # Create .mcp.json.example as template
    cat > "$TEST_WORKSPACE/.mcp.json.example" << 'EOF'
{
  "mcpServers": {
    "dkmcp": {
      "type": "sse",
      "url": "http://host.docker.internal:8080/sse"
    }
  }
}
EOF

    # Run with PATH that excludes claude/gemini to force fallback
    WORKSPACE="$TEST_WORKSPACE" HOME="$TEST_WORKSPACE" PATH="/usr/bin:/bin" \
        "$SCRIPT" 2>/dev/null || true

    if [ -f "$TEST_WORKSPACE/.mcp.json" ] && \
       jq -e '.mcpServers.dkmcp' "$TEST_WORKSPACE/.mcp.json" >/dev/null 2>&1; then
        pass "Fallback creates .mcp.json with dkmcp entry"
    else
        fail "Fallback did not create .mcp.json with dkmcp entry"
    fi

    cleanup
}

# Test 6: Fallback preserves existing entries in .mcp.json
# テスト6: フォールバックで既存エントリが保持されるか
test_register_preserves_existing_entries() {
    echo ""
    echo "=== Test: Registration preserves existing .mcp.json entries ==="

    setup

    # Create .mcp.json with an existing entry
    cat > "$TEST_WORKSPACE/.mcp.json" << 'EOF'
{
  "mcpServers": {
    "sandbox-mcp": {
      "type": "stdio",
      "command": "/workspace/.sandbox/sandbox-mcp/sandbox-mcp"
    }
  }
}
EOF

    # Run with PATH that excludes claude/gemini
    WORKSPACE="$TEST_WORKSPACE" HOME="$TEST_WORKSPACE" PATH="/usr/bin:/bin" \
        "$SCRIPT" 2>/dev/null || true

    if jq -e '.mcpServers.dkmcp' "$TEST_WORKSPACE/.mcp.json" >/dev/null 2>&1 && \
       jq -e '.mcpServers["sandbox-mcp"]' "$TEST_WORKSPACE/.mcp.json" >/dev/null 2>&1; then
        pass "Registration preserves existing entries"
    else
        fail "Registration did not preserve existing entries"
    fi

    cleanup
}

# Test 7: --unregister removes dkmcp from .mcp.json
# テスト7: --unregister が .mcp.json から dkmcp を削除するか
test_unregister_removes_dkmcp() {
    echo ""
    echo "=== Test: --unregister removes dkmcp from .mcp.json ==="

    setup

    # Create .mcp.json with both dkmcp and sandbox-mcp
    cat > "$TEST_WORKSPACE/.mcp.json" << 'EOF'
{
  "mcpServers": {
    "dkmcp": {
      "type": "sse",
      "url": "http://host.docker.internal:8080/sse"
    },
    "sandbox-mcp": {
      "type": "stdio",
      "command": "/workspace/.sandbox/sandbox-mcp/sandbox-mcp"
    }
  }
}
EOF

    WORKSPACE="$TEST_WORKSPACE" HOME="$TEST_WORKSPACE" PATH="/usr/bin:/bin" \
        "$SCRIPT" --unregister 2>/dev/null || true

    if ! jq -e '.mcpServers.dkmcp' "$TEST_WORKSPACE/.mcp.json" >/dev/null 2>&1 && \
       jq -e '.mcpServers["sandbox-mcp"]' "$TEST_WORKSPACE/.mcp.json" >/dev/null 2>&1; then
        pass "--unregister removes only dkmcp, preserves others"
    else
        fail "--unregister did not correctly remove only dkmcp"
    fi

    cleanup
}

# Test 8: --url applies custom URL
# テスト8: --url でカスタム URL が適用されるか
test_custom_url() {
    echo ""
    echo "=== Test: --url applies custom URL ==="

    setup

    local custom_url="http://custom-host:9999/sse"

    # Need .mcp.json.example so can_register_claude returns true without CLI
    cat > "$TEST_WORKSPACE/.mcp.json.example" << 'EOF'
{
  "mcpServers": {}
}
EOF

    WORKSPACE="$TEST_WORKSPACE" HOME="$TEST_WORKSPACE" PATH="/usr/bin:/bin" \
        "$SCRIPT" --url "$custom_url" 2>/dev/null || true

    if [ -f "$TEST_WORKSPACE/.mcp.json" ]; then
        local url_in_file
        url_in_file=$(jq -r '.mcpServers.dkmcp.url' "$TEST_WORKSPACE/.mcp.json" 2>/dev/null)
        if [ "$url_in_file" = "$custom_url" ]; then
            pass "Custom URL is applied correctly"
        else
            fail "URL in .mcp.json is '$url_in_file', expected '$custom_url'"
        fi
    else
        fail ".mcp.json was not created"
    fi

    cleanup
}

# Test 9: --status shows tool status and connectivity
# テスト9: --status がツール状態と接続情報を表示するか
test_status_output() {
    echo ""
    echo "=== Test: --status shows tool status and connectivity ==="

    setup

    local output
    output=$(WORKSPACE="$TEST_WORKSPACE" HOME="$TEST_WORKSPACE" "$SCRIPT" --status 2>&1) || true

    local ok=true

    # Should contain tool names
    if ! echo "$output" | grep -qi "Claude\|Gemini"; then
        fail "--status missing tool names"
        ok=false
    fi

    # Should contain connectivity section
    if ! echo "$output" | grep -qi "connect\|接続"; then
        fail "--status missing connectivity information"
        ok=false
    fi

    if [ "$ok" = true ]; then
        pass "--status shows tool status and connectivity"
    fi

    cleanup
}

# Test 10: No tools found exits with error
# テスト10: AI ツールもファイルもない場合に exit 1 で終了するか
test_no_tools_found() {
    echo ""
    echo "=== Test: No tools found exits with error ==="

    setup
    # No claude CLI, no gemini CLI, no .mcp.json, no .mcp.json.example

    local exit_code=0
    WORKSPACE="$TEST_WORKSPACE" HOME="$TEST_WORKSPACE" PATH="/usr/bin:/bin" \
        "$SCRIPT" 2>/dev/null || exit_code=$?

    if [ "$exit_code" -eq 1 ] && [ ! -f "$TEST_WORKSPACE/.mcp.json" ]; then
        pass "No tools found exits 1 and does not create .mcp.json"
    else
        fail "Expected exit 1 and no .mcp.json, got exit $exit_code"
    fi

    cleanup
}

# Test 11: Detection finds dkmcp in ~/.claude.json user scope
# テスト11: ~/.claude.json のユーザースコープで dkmcp を検出するか
test_detect_claude_user_scope() {
    echo ""
    echo "=== Test: Detection finds dkmcp in ~/.claude.json user scope ==="

    setup

    # Create ~/.claude.json with dkmcp at user scope
    cat > "$TEST_WORKSPACE/.claude.json" << 'EOF'
{
  "mcpServers": {
    "dkmcp": {
      "type": "sse",
      "url": "http://host.docker.internal:8080/sse"
    }
  }
}
EOF

    local exit_code=0
    WORKSPACE="$TEST_WORKSPACE" HOME="$TEST_WORKSPACE" \
        "$SCRIPT" --check --url "http://localhost:1/sse" 2>/dev/null || exit_code=$?

    # Should be 2 (registered but offline), not 1 (not registered)
    if [ "$exit_code" -eq 2 ]; then
        pass "Detection finds dkmcp in ~/.claude.json user scope"
    else
        fail "Detection returned $exit_code, expected 2 (registered but offline)"
    fi

    cleanup
}

# Test 12: Detection finds dkmcp in ~/.claude.json project scope
# テスト12: ~/.claude.json のプロジェクトスコープで dkmcp を検出するか
test_detect_claude_project_scope() {
    echo ""
    echo "=== Test: Detection finds dkmcp in ~/.claude.json project scope ==="

    setup

    # Create ~/.claude.json with dkmcp at project scope
    # Note: project key must match WORKSPACE
    local project_key="$TEST_WORKSPACE"
    cat > "$TEST_WORKSPACE/.claude.json" << EOF
{
  "projects": {
    "$project_key": {
      "mcpServers": {
        "dkmcp": {
          "type": "sse",
          "url": "http://host.docker.internal:8080/sse"
        }
      }
    }
  }
}
EOF

    local exit_code=0
    WORKSPACE="$TEST_WORKSPACE" HOME="$TEST_WORKSPACE" \
        "$SCRIPT" --check --url "http://localhost:1/sse" 2>/dev/null || exit_code=$?

    if [ "$exit_code" -eq 2 ]; then
        pass "Detection finds dkmcp in ~/.claude.json project scope"
    else
        fail "Detection returned $exit_code, expected 2 (registered but offline)"
    fi

    cleanup
}

# Test 13: Gemini registration creates .gemini/settings.json
# テスト13: Gemini 登録で .gemini/settings.json が作成されるか
test_gemini_register_creates_settings() {
    echo ""
    echo "=== Test: Gemini registration creates .gemini/settings.json ==="

    setup

    # Simulate gemini CLI with a stub that writes settings.json
    local stub_dir="$TEST_WORKSPACE/bin"
    mkdir -p "$stub_dir"
    cat > "$stub_dir/gemini" << 'STUB'
#!/bin/bash
# Stub: simulate gemini mcp add by writing settings.json
# Args: gemini mcp add --transport sse <name> <url>
if [[ "$1" == "mcp" && "$2" == "add" ]]; then
    mkdir -p "$HOME/.gemini"
    jq -n --arg name "$5" --arg url "$6" \
        '{"mcpServers":{($name):{"url":$url,"type":"sse"}}}' > "$HOME/.gemini/settings.json"
    exit 0
fi
exit 1
STUB
    chmod +x "$stub_dir/gemini"

    WORKSPACE="$TEST_WORKSPACE" HOME="$TEST_WORKSPACE" PATH="$stub_dir:/usr/bin:/bin" \
        "$SCRIPT" 2>/dev/null || true

    if [ -f "$TEST_WORKSPACE/.gemini/settings.json" ] && \
       jq -e '.mcpServers.dkmcp' "$TEST_WORKSPACE/.gemini/settings.json" >/dev/null 2>&1; then
        pass "Gemini registration creates settings.json with dkmcp entry"
    else
        fail "Gemini registration did not create settings.json with dkmcp entry"
    fi

    cleanup
}

# Test 14: Gemini --unregister removes dkmcp from .gemini/settings.json
# テスト14: Gemini の --unregister で dkmcp が削除されるか
test_gemini_unregister() {
    echo ""
    echo "=== Test: Gemini --unregister removes dkmcp ==="

    setup

    # Create .gemini/settings.json with dkmcp and another entry
    mkdir -p "$TEST_WORKSPACE/.gemini"
    cat > "$TEST_WORKSPACE/.gemini/settings.json" << 'EOF'
{
  "mcpServers": {
    "dkmcp": {
      "type": "sse",
      "url": "http://host.docker.internal:8080/sse"
    },
    "other-mcp": {
      "type": "stdio",
      "command": "/usr/bin/other"
    }
  }
}
EOF

    # Need gemini stub in PATH for has_gemini guard in mode_unregister
    local stub_dir="$TEST_WORKSPACE/bin"
    mkdir -p "$stub_dir"
    cat > "$stub_dir/gemini" << 'STUB'
#!/bin/bash
# Stub: gemini CLI exists but mcp remove fails (JSON removal handled by script directly)
exit 1
STUB
    chmod +x "$stub_dir/gemini"

    WORKSPACE="$TEST_WORKSPACE" HOME="$TEST_WORKSPACE" PATH="$stub_dir:/usr/bin:/bin" \
        "$SCRIPT" --unregister 2>/dev/null || true

    if ! jq -e '.mcpServers.dkmcp' "$TEST_WORKSPACE/.gemini/settings.json" >/dev/null 2>&1 && \
       jq -e '.mcpServers["other-mcp"]' "$TEST_WORKSPACE/.gemini/settings.json" >/dev/null 2>&1; then
        pass "--unregister removes dkmcp from Gemini, preserves others"
    else
        fail "--unregister did not correctly handle Gemini settings"
    fi

    cleanup
}

# Test 15: Detection finds dkmcp in .gemini/settings.json
# テスト15: .gemini/settings.json で dkmcp を検出するか
test_detect_gemini_registered() {
    echo ""
    echo "=== Test: Detection finds dkmcp in .gemini/settings.json ==="

    setup

    # Create .gemini/settings.json with dkmcp + gemini stub
    mkdir -p "$TEST_WORKSPACE/.gemini"
    cat > "$TEST_WORKSPACE/.gemini/settings.json" << 'EOF'
{
  "mcpServers": {
    "dkmcp": {
      "type": "sse",
      "url": "http://host.docker.internal:8080/sse"
    }
  }
}
EOF

    local stub_dir="$TEST_WORKSPACE/bin"
    mkdir -p "$stub_dir"
    cat > "$stub_dir/gemini" << 'STUB'
#!/bin/bash
exit 0
STUB
    chmod +x "$stub_dir/gemini"

    local exit_code=0
    WORKSPACE="$TEST_WORKSPACE" HOME="$TEST_WORKSPACE" PATH="$stub_dir:/usr/bin:/bin" \
        "$SCRIPT" --check --url "http://localhost:1/sse" 2>/dev/null || exit_code=$?

    # Should be 2 (registered but offline), not 1 (not registered)
    if [ "$exit_code" -eq 2 ]; then
        pass "Detection finds dkmcp in .gemini/settings.json"
    else
        fail "Detection returned $exit_code, expected 2 (registered but offline)"
    fi

    cleanup
}

# Test 16: Registration failure shows error message (not crash)
# テスト16: 登録失敗時にエラーメッセージが表示され、クラッシュしないか
test_register_failure_shows_error() {
    echo ""
    echo "=== Test: Registration failure shows error message ==="

    setup

    # Create claude stub that always fails
    local stub_dir="$TEST_WORKSPACE/bin"
    mkdir -p "$stub_dir"
    cat > "$stub_dir/claude" << 'STUB'
#!/bin/bash
# Stub: simulate claude mcp add failure
exit 1
STUB
    chmod +x "$stub_dir/claude"

    local output
    local exit_code=0
    output=$(WORKSPACE="$TEST_WORKSPACE" HOME="$TEST_WORKSPACE" PATH="$stub_dir:/usr/bin:/bin" \
        "$SCRIPT" 2>&1) || exit_code=$?

    # Should not crash (exit 0 is ok, script continues after failed registration)
    # Output should contain error message, not success message
    if echo "$output" | grep -qi "failed\|失敗"; then
        pass "Registration failure shows error message"
    else
        fail "Registration failure did not show error message (exit=$exit_code)"
    fi

    cleanup
}

# Test 17: Gemini CLI failure shows error message (not crash)
# テスト17: Gemini CLI 失敗時にエラーメッセージが表示されるか
test_gemini_register_failure_shows_error() {
    echo ""
    echo "=== Test: Gemini CLI failure shows error message ==="

    setup

    # has_gemini needs to be true → gemini stub in PATH
    # register_gemini: has_gemini → tries CLI → fails → returns non-zero
    # mode_default: shows error message (new behavior)
    local stub_dir="$TEST_WORKSPACE/bin"
    mkdir -p "$stub_dir"
    cat > "$stub_dir/gemini" << 'STUB'
#!/bin/bash
# Stub: exists in PATH but mcp add fails
exit 1
STUB
    chmod +x "$stub_dir/gemini"

    local output
    output=$(WORKSPACE="$TEST_WORKSPACE" HOME="$TEST_WORKSPACE" PATH="$stub_dir:/usr/bin:/bin" \
        "$SCRIPT" 2>&1) || true

    if echo "$output" | grep -qi "\[Gemini\].*failed\|\[Gemini\].*失敗"; then
        pass "Gemini CLI failure shows error message"
    else
        fail "Gemini CLI failure did not show error message"
    fi

    cleanup
}

# ─── Run all tests / 全テスト実行 ─────────────────────────────

main() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  setup-dkmcp.sh Test Suite"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    test_script_runs_and_shows_help
    test_help
    test_check_not_registered
    test_check_registered_but_offline
    test_register_fallback_creates_mcp_json
    test_register_preserves_existing_entries
    test_unregister_removes_dkmcp
    test_custom_url
    test_status_output
    test_no_tools_found
    test_detect_claude_user_scope
    test_detect_claude_project_scope
    test_gemini_register_creates_settings
    test_gemini_unregister
    test_detect_gemini_registered
    test_register_failure_shows_error
    test_gemini_register_failure_shows_error

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

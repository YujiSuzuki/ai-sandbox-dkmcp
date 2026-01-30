#!/bin/bash
# test-check-upstream-updates.sh
# Test update check functionality
# 更新チェック機能のテスト

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE="${WORKSPACE:-/workspace}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Counters
TESTS_PASSED=0
TESTS_FAILED=0

# Test helpers
pass() { echo -e "${GREEN}PASS${NC}: $1"; ((TESTS_PASSED++)) || true; }
fail() { echo -e "${RED}FAIL${NC}: $1"; ((TESTS_FAILED++)) || true; }
info() { echo -e "${YELLOW}INFO${NC}: $1"; }

# Temp directory for tests
TEST_TMP_DIR=""

setup() {
    TEST_TMP_DIR=$(mktemp -d)
    # Create mock workspace structure for tests that override WORKSPACE
    mkdir -p "$TEST_TMP_DIR/config"
    mkdir -p "$TEST_TMP_DIR/.sandbox/config"
    mkdir -p "$TEST_TMP_DIR/.sandbox/scripts"
    # Symlink shared files so mock WORKSPACE can source them
    ln -sf "$WORKSPACE/.sandbox/scripts/_startup_common.sh" "$TEST_TMP_DIR/.sandbox/scripts/_startup_common.sh"
    ln -sf "$WORKSPACE/.sandbox/config/startup.conf" "$TEST_TMP_DIR/.sandbox/config/startup.conf"
}

teardown() {
    [ -n "$TEST_TMP_DIR" ] && rm -rf "$TEST_TMP_DIR"
}

# ============================================================
# Test: Configuration file exists and is valid
# ============================================================
test_config_file() {
    echo ""
    echo "=== Testing template-source.conf ==="

    local config_file="$WORKSPACE/.sandbox/config/template-source.conf"

    if [ -f "$config_file" ]; then
        pass "template-source.conf exists"
    else
        fail "template-source.conf does not exist"
        return
    fi

    # Check required variables
    if grep -q "^TEMPLATE_REPO=" "$config_file"; then
        pass "template-source.conf contains TEMPLATE_REPO"
    else
        fail "template-source.conf should contain TEMPLATE_REPO"
    fi

    if grep -q "^CHECK_UPDATES=" "$config_file"; then
        pass "template-source.conf contains CHECK_UPDATES"
    else
        fail "template-source.conf should contain CHECK_UPDATES"
    fi

    if grep -q "^CHECK_CHANNEL=" "$config_file"; then
        pass "template-source.conf contains CHECK_CHANNEL"
    else
        fail "template-source.conf should contain CHECK_CHANNEL"
    fi

    if grep -q "^CHECK_INTERVAL_HOURS=" "$config_file"; then
        pass "template-source.conf contains CHECK_INTERVAL_HOURS"
    else
        fail "template-source.conf should contain CHECK_INTERVAL_HOURS"
    fi

    # Check default values
    # shellcheck source=/dev/null
    source "$config_file"

    if [ "$TEMPLATE_REPO" = "YujiSuzuki/ai-sandbox-dkmcp" ]; then
        pass "TEMPLATE_REPO has correct default value"
    else
        fail "TEMPLATE_REPO should be 'YujiSuzuki/ai-sandbox-dkmcp', got '$TEMPLATE_REPO'"
    fi

    if [ "$CHECK_UPDATES" = "true" ]; then
        pass "CHECK_UPDATES is enabled by default"
    else
        fail "CHECK_UPDATES should be 'true', got '$CHECK_UPDATES'"
    fi

    if [ "$CHECK_CHANNEL" = "all" ]; then
        pass "CHECK_CHANNEL is 'all' by default"
    else
        fail "CHECK_CHANNEL should be 'all', got '$CHECK_CHANNEL'"
    fi

    if [ "$CHECK_INTERVAL_HOURS" = "24" ]; then
        pass "CHECK_INTERVAL_HOURS is 24 hours by default"
    else
        fail "CHECK_INTERVAL_HOURS should be '24', got '$CHECK_INTERVAL_HOURS'"
    fi
}

# ============================================================
# Test: State file read/write (uses real functions from script)
# ============================================================
test_state_file() {
    echo ""
    echo "=== Testing state file read/write ==="

    # Source the actual script
    # shellcheck source=/dev/null
    source "$WORKSPACE/.sandbox/scripts/check-upstream-updates.sh"

    # Override STATE_FILE for testing
    STATE_FILE="$TEST_TMP_DIR/state"

    # Test 1: No state file → is_first_run returns true
    rm -f "$STATE_FILE"
    if is_first_run; then
        pass "is_first_run returns true when no state file"
    else
        fail "is_first_run should return true when no state file"
    fi

    # Test 2: No state file → get_last_notified_version returns empty
    local result
    result=$(get_last_notified_version)
    if [ -z "$result" ]; then
        pass "get_last_notified_version returns empty when no state file"
    else
        fail "get_last_notified_version should return empty, got '$result'"
    fi

    # Test 3: No state file → read_state_timestamp returns "0"
    result=$(read_state_timestamp)
    if [ "$result" = "0" ]; then
        pass "read_state_timestamp returns '0' when no state file"
    else
        fail "read_state_timestamp should return '0', got '$result'"
    fi

    # Test 4: Write state and read back version
    echo "1738300000:v0.2.0" > "$STATE_FILE"
    result=$(get_last_notified_version)
    if [ "$result" = "v0.2.0" ]; then
        pass "get_last_notified_version reads 'v0.2.0' from state file"
    else
        fail "get_last_notified_version should return 'v0.2.0', got '$result'"
    fi

    # Test 5: Write state and read back timestamp
    result=$(read_state_timestamp)
    if [ "$result" = "1738300000" ]; then
        pass "read_state_timestamp reads '1738300000' from state file"
    else
        fail "read_state_timestamp should return '1738300000', got '$result'"
    fi

    # Test 6: After writing state, is_first_run returns false
    if is_first_run; then
        fail "is_first_run should return false when state file exists"
    else
        pass "is_first_run returns false when state file exists"
    fi

    # Test 7: update_state writes correct format
    update_state "v1.0.0"
    result=$(get_last_notified_version)
    if [ "$result" = "v1.0.0" ]; then
        pass "update_state writes version correctly"
    else
        fail "update_state should write 'v1.0.0', got '$result'"
    fi

    # Test 8: Version with pre-release suffix (contains no extra colons)
    echo "1738300000:v0.3.0-beta.1" > "$STATE_FILE"
    result=$(get_last_notified_version)
    if [ "$result" = "v0.3.0-beta.1" ]; then
        pass "get_last_notified_version handles pre-release suffix"
    else
        fail "get_last_notified_version should return 'v0.3.0-beta.1', got '$result'"
    fi
}

# ============================================================
# Test: Interval checking logic (uses real function from script)
# ============================================================
test_interval_check() {
    echo ""
    echo "=== Testing interval check logic ==="

    # Source the actual script to get the real should_check function
    # shellcheck source=/dev/null
    source "$WORKSPACE/.sandbox/scripts/check-upstream-updates.sh"

    # Override STATE_FILE for testing
    STATE_FILE="$TEST_TMP_DIR/state"

    # Test 1: No state file - should check
    rm -f "$STATE_FILE"
    CHECK_INTERVAL_HOURS=24
    if should_check; then
        pass "should_check returns true when no state file"
    else
        fail "should_check should return true when no state file"
    fi

    # Test 2: Interval 0 - always check
    echo "$(date +%s):v0.1.0" > "$STATE_FILE"
    CHECK_INTERVAL_HOURS=0
    if should_check; then
        pass "should_check returns true when interval is 0"
    else
        fail "should_check should return true when interval is 0"
    fi

    # Test 3: Recent timestamp - should not check
    echo "$(date +%s):v0.1.0" > "$STATE_FILE"
    CHECK_INTERVAL_HOURS=24
    if should_check; then
        fail "should_check should return false when timestamp is recent"
    else
        pass "should_check returns false when timestamp is recent"
    fi

    # Test 4: Old timestamp - should check
    echo "$(($(date +%s) - 100000)):v0.1.0" > "$STATE_FILE"
    CHECK_INTERVAL_HOURS=24
    if should_check; then
        pass "should_check returns true when timestamp is old"
    else
        fail "should_check should return true when timestamp is old"
    fi

    # Test 5: Invalid (non-numeric) interval - should fallback to 24 and work
    echo "$(date +%s):v0.1.0" > "$STATE_FILE"
    CHECK_INTERVAL_HOURS="abc"
    if should_check; then
        fail "should_check should return false with invalid interval (fallback to 24)"
    else
        pass "should_check handles invalid interval by falling back to 24"
    fi

    # Test 6: Empty interval - should fallback to 24 and work
    echo "$(date +%s):v0.1.0" > "$STATE_FILE"
    CHECK_INTERVAL_HOURS=""
    if should_check; then
        fail "should_check should return false with empty interval (fallback to 24)"
    else
        pass "should_check handles empty interval by falling back to 24"
    fi
}

# ============================================================
# Test: build_api_url returns correct URL per channel
# ============================================================
test_build_api_url() {
    echo ""
    echo "=== Testing build_api_url ==="

    # Source the actual script
    # shellcheck source=/dev/null
    source "$WORKSPACE/.sandbox/scripts/check-upstream-updates.sh"

    local result

    # Test 1: channel "all" → releases?per_page=1
    CHECK_CHANNEL="all"
    result=$(build_api_url "owner/repo")
    if [ "$result" = "https://api.github.com/repos/owner/repo/releases?per_page=1" ]; then
        pass "build_api_url with channel 'all' returns releases?per_page=1"
    else
        fail "build_api_url with channel 'all' should return releases?per_page=1, got '$result'"
    fi

    # Test 2: channel "stable" → releases/latest
    CHECK_CHANNEL="stable"
    result=$(build_api_url "owner/repo")
    if [ "$result" = "https://api.github.com/repos/owner/repo/releases/latest" ]; then
        pass "build_api_url with channel 'stable' returns releases/latest"
    else
        fail "build_api_url with channel 'stable' should return releases/latest, got '$result'"
    fi

    # Test 3: unset/default channel → releases?per_page=1
    unset CHECK_CHANNEL
    result=$(build_api_url "owner/repo")
    if [ "$result" = "https://api.github.com/repos/owner/repo/releases?per_page=1" ]; then
        pass "build_api_url with unset channel defaults to releases?per_page=1"
    else
        fail "build_api_url with unset channel should default to releases?per_page=1, got '$result'"
    fi

    # Test 4: unknown channel value → treated as "all"
    CHECK_CHANNEL="unknown"
    result=$(build_api_url "owner/repo")
    if [ "$result" = "https://api.github.com/repos/owner/repo/releases?per_page=1" ]; then
        pass "build_api_url with unknown channel falls back to 'all'"
    else
        fail "build_api_url with unknown channel should fall back to 'all', got '$result'"
    fi
}

# ============================================================
# Test: extract_tag_from_json parses both array and object JSON
# ============================================================
test_extract_tag_from_json() {
    echo ""
    echo "=== Testing extract_tag_from_json ==="

    # Source the actual script
    # shellcheck source=/dev/null
    source "$WORKSPACE/.sandbox/scripts/check-upstream-updates.sh"

    # Skip if jq is not available (grep fallback is tested separately)
    if ! command -v jq &>/dev/null; then
        info "jq not available, testing grep fallback only"
    fi

    local result
    local json_file="$TEST_TMP_DIR/release.json"

    # Test 1: Array response (channel "all") — /releases?per_page=1 format
    cat > "$json_file" <<'JSONEOF'
[{"tag_name":"v0.2.0-beta.1","name":"Beta Release","prerelease":true}]
JSONEOF
    CHECK_CHANNEL="all"
    result=$(extract_tag_from_json "$json_file")
    if [ "$result" = "v0.2.0-beta.1" ]; then
        pass "extract_tag_from_json parses array response (channel=all)"
    else
        fail "extract_tag_from_json array should return 'v0.2.0-beta.1', got '$result'"
    fi

    # Test 2: Object response (channel "stable") — /releases/latest format
    cat > "$json_file" <<'JSONEOF'
{"tag_name":"v1.0.0","name":"Stable Release","prerelease":false}
JSONEOF
    CHECK_CHANNEL="stable"
    result=$(extract_tag_from_json "$json_file")
    if [ "$result" = "v1.0.0" ]; then
        pass "extract_tag_from_json parses object response (channel=stable)"
    else
        fail "extract_tag_from_json object should return 'v1.0.0', got '$result'"
    fi

    # Test 3: Empty array — no releases
    echo '[]' > "$json_file"
    CHECK_CHANNEL="all"
    result=$(extract_tag_from_json "$json_file")
    if [ -z "$result" ] || [ "$result" = "null" ]; then
        pass "extract_tag_from_json returns empty for empty array"
    else
        fail "extract_tag_from_json should return empty for '[]', got '$result'"
    fi

    # Test 4: Object with no tag_name
    echo '{"name":"No Tag"}' > "$json_file"
    CHECK_CHANNEL="stable"
    result=$(extract_tag_from_json "$json_file")
    if [ -z "$result" ] || [ "$result" = "null" ]; then
        pass "extract_tag_from_json returns empty when tag_name missing"
    else
        fail "extract_tag_from_json should return empty for missing tag_name, got '$result'"
    fi
}

# ============================================================
# Test: Debug mode outputs diagnostic info to stderr
# ============================================================
test_debug_mode() {
    echo ""
    echo "=== Testing debug mode ==="

    local script="$WORKSPACE/.sandbox/scripts/check-upstream-updates.sh"
    local stderr_output

    # テスト用の設定ファイルを作成（CHECK_UPDATES=false で即終了させる）
    local mock_config="$TEST_TMP_DIR/.sandbox/config/template-source.conf"
    cat > "$mock_config" <<'EOF'
TEMPLATE_REPO="YujiSuzuki/ai-sandbox-dkmcp"
CHECK_CHANNEL="all"
CHECK_UPDATES="false"
CHECK_INTERVAL_HOURS="0"
EOF

    # Test 1: --debug flag produces debug output on stderr
    stderr_output=$( (WORKSPACE="$TEST_TMP_DIR" "$script" --debug) 2>&1 1>/dev/null ) || true
    if echo "$stderr_output" | grep -q "^\[debug\]"; then
        pass "--debug flag produces [debug] output on stderr"
    else
        fail "--debug flag should produce [debug] output on stderr, got: '$stderr_output'"
    fi

    # Test 2: DEBUG_UPDATE_CHECK=1 environment variable also works
    stderr_output=$( (WORKSPACE="$TEST_TMP_DIR" DEBUG_UPDATE_CHECK=1 "$script") 2>&1 1>/dev/null ) || true
    if echo "$stderr_output" | grep -q "^\[debug\]"; then
        pass "DEBUG_UPDATE_CHECK=1 produces [debug] output on stderr"
    else
        fail "DEBUG_UPDATE_CHECK=1 should produce [debug] output on stderr, got: '$stderr_output'"
    fi

    # Test 3: Without debug, no [debug] output
    stderr_output=$( (WORKSPACE="$TEST_TMP_DIR" "$script") 2>&1 1>/dev/null ) || true
    if echo "$stderr_output" | grep -q "^\[debug\]"; then
        fail "Without debug flag, should not produce [debug] output"
    else
        pass "Without debug flag, no [debug] output on stderr"
    fi

    # Test 4: Debug shows config values when loaded
    stderr_output=$( (WORKSPACE="$TEST_TMP_DIR" DEBUG_UPDATE_CHECK=1 "$script") 2>&1 1>/dev/null ) || true
    if echo "$stderr_output" | grep -q "Config loaded:"; then
        pass "Debug output includes config values"
    else
        fail "Debug output should include 'Config loaded:', got: '$stderr_output'"
    fi

    # Test 5: Debug shows reason for exit (CHECK_UPDATES=false in mock config)
    if echo "$stderr_output" | grep -q "disabled, exit"; then
        pass "Debug output shows disabled reason"
    else
        fail "Debug output should show disabled reason, got: '$stderr_output'"
    fi

    # Test 6: Debug output goes to stderr, not stdout (stdout should be empty)
    local stdout_output
    stdout_output=$( (WORKSPACE="$TEST_TMP_DIR" DEBUG_UPDATE_CHECK=1 "$script") 2>/dev/null ) || true
    if [ -z "$stdout_output" ]; then
        pass "Debug output goes to stderr only, stdout is clean"
    else
        fail "Debug output should not appear on stdout, got: '$stdout_output'"
    fi
}

# ============================================================
# Test: First run records version without notification
# ============================================================
test_first_run_no_notification() {
    echo ""
    echo "=== Testing first run behavior ==="

    local script="$WORKSPACE/.sandbox/scripts/check-upstream-updates.sh"

    # テスト用の設定ファイル（CHECK_UPDATES=true, INTERVAL=0 で毎回チェック）
    local mock_config="$TEST_TMP_DIR/.sandbox/config/template-source.conf"
    cat > "$mock_config" <<'EOF'
TEMPLATE_REPO="YujiSuzuki/ai-sandbox-dkmcp"
CHECK_CHANNEL="all"
CHECK_UPDATES="true"
CHECK_INTERVAL_HOURS="0"
EOF

    # STATE_FILE を明示的に削除して初回状態にする
    local mock_state="$TEST_TMP_DIR/state"
    rm -f "$mock_state"

    # Test 1: 初回実行 → stdout に通知が出ない（記録のみ）
    local stdout_output
    stdout_output=$( (WORKSPACE="$TEST_TMP_DIR" STATE_FILE="$mock_state" "$script") 2>/dev/null ) || true
    if [ -z "$stdout_output" ]; then
        pass "First run produces no notification on stdout"
    else
        fail "First run should produce no notification, got: '$stdout_output'"
    fi

    # Test 2: 初回実行後、状態ファイルが作成されている
    if [ -f "$mock_state" ]; then
        pass "State file created after first run"
    else
        fail "State file should be created after first run"
    fi

    # Test 3: 状態ファイルにバージョンが記録されている
    local recorded_version
    recorded_version=$(cut -d: -f2- "$mock_state" 2>/dev/null || echo "")
    if [ -n "$recorded_version" ]; then
        pass "State file contains recorded version: $recorded_version"
    else
        fail "State file should contain a version"
    fi

    # Test 4: Debug で "First run" が出る
    rm -f "$mock_state"
    local stderr_output
    stderr_output=$( (WORKSPACE="$TEST_TMP_DIR" STATE_FILE="$mock_state" DEBUG_UPDATE_CHECK=1 "$script") 2>&1 1>/dev/null ) || true
    if echo "$stderr_output" | grep -q "First run"; then
        pass "Debug output shows 'First run' on first execution"
    else
        fail "Debug output should show 'First run', got: '$stderr_output'"
    fi
}

# ============================================================
# Test: Same version does not re-notify
# ============================================================
test_same_version_no_renotify() {
    echo ""
    echo "=== Testing same version dedup ==="

    local script="$WORKSPACE/.sandbox/scripts/check-upstream-updates.sh"

    # テスト用の設定ファイル
    local mock_config="$TEST_TMP_DIR/.sandbox/config/template-source.conf"
    cat > "$mock_config" <<'EOF'
TEMPLATE_REPO="YujiSuzuki/ai-sandbox-dkmcp"
CHECK_CHANNEL="all"
CHECK_UPDATES="true"
CHECK_INTERVAL_HOURS="0"
EOF

    # 初回実行（バージョン記録）
    local mock_state="$TEST_TMP_DIR/state"
    rm -f "$mock_state"
    (WORKSPACE="$TEST_TMP_DIR" STATE_FILE="$mock_state" "$script") >/dev/null 2>&1 || true

    # 2回目実行（同バージョン → 通知なし）
    local stdout_output
    stdout_output=$( (WORKSPACE="$TEST_TMP_DIR" STATE_FILE="$mock_state" "$script") 2>/dev/null ) || true
    if [ -z "$stdout_output" ]; then
        pass "Second run with same version produces no notification"
    else
        fail "Second run should produce no notification, got: '$stdout_output'"
    fi

    # Debug で "Same version" が出ることを確認
    local stderr_output
    stderr_output=$( (WORKSPACE="$TEST_TMP_DIR" STATE_FILE="$mock_state" DEBUG_UPDATE_CHECK=1 "$script") 2>&1 1>/dev/null ) || true
    if echo "$stderr_output" | grep -q "Same version"; then
        pass "Debug output shows 'Same version' on second run"
    else
        fail "Debug output should show 'Same version', got: '$stderr_output'"
    fi
}

# ============================================================
# Test: show_update_notification outputs correctly per verbosity
# 各詳細度レベルで通知が正しく表示されるか
# ============================================================
test_show_update_notification() {
    echo ""
    echo "=== Testing show_update_notification ==="

    # Source the actual script
    # shellcheck source=/dev/null
    source "$WORKSPACE/.sandbox/scripts/check-upstream-updates.sh"

    # Setup English messages
    setup_messages

    local output

    # Test 1: Quiet mode — single line with version transition
    STARTUP_VERBOSITY="quiet"
    output=$(show_update_notification "v0.1.0" "v0.2.0" "https://example.com/releases")
    if echo "$output" | grep -q "v0.1.0 → v0.2.0"; then
        pass "Quiet mode shows version transition"
    else
        fail "Quiet mode should show 'v0.1.0 → v0.2.0', got: '$output'"
    fi

    # Test 2: Quiet mode — single line only (no extra decoration)
    local line_count
    line_count=$(echo "$output" | wc -l)
    if [ "$line_count" -eq 1 ]; then
        pass "Quiet mode outputs single line"
    else
        fail "Quiet mode should output 1 line, got $line_count"
    fi

    # Test 3: Summary mode — includes current version, latest version, and URL on separate lines
    STARTUP_VERBOSITY="summary"
    output=$(show_update_notification "v0.1.0" "v0.2.0" "https://example.com/releases")
    if echo "$output" | grep -q "v0.1.0" && echo "$output" | grep -q "v0.2.0" && echo "$output" | grep -q "https://example.com/releases"; then
        pass "Summary mode shows current version, latest version, and URL"
    else
        fail "Summary mode should show both versions and URL, got: '$output'"
    fi

    # Test 4: Summary mode — includes separator lines
    if echo "$output" | grep -q "━━━━"; then
        pass "Summary mode includes separator lines"
    else
        fail "Summary mode should include separator lines"
    fi

    # Test 5a: Summary mode — shows current and latest on separate lines (like verbose)
    if echo "$output" | grep -q "$MSG_CURRENT" && echo "$output" | grep -q "$MSG_LATEST"; then
        pass "Summary mode shows current/latest labels on separate lines"
    else
        fail "Summary mode should show current/latest labels, got: '$output'"
    fi

    # Test 5b: Summary mode — output has enough lines (title + content + footer, not truncated by set -e)
    local summary_line_count
    summary_line_count=$(echo "$output" | grep -c . || true)
    if [ "$summary_line_count" -ge 6 ]; then
        pass "Summary mode outputs at least 6 non-empty lines (not truncated)"
    else
        fail "Summary mode should have at least 6 non-empty lines, got $summary_line_count: '$output'"
    fi

    # Test 5c: Summary mode — has release notes URL line
    if echo "$output" | grep -q "$MSG_RELEASE_NOTES"; then
        pass "Summary mode shows release notes label"
    else
        fail "Summary mode should show release notes label, got: '$output'"
    fi

    # Test 5d: Summary mode — has footer separator (not just title separator)
    local separator_count
    separator_count=$(echo "$output" | grep -c "━━━━" || true)
    if [ "$separator_count" -ge 3 ]; then
        pass "Summary mode has title + footer separators (at least 3 separator lines)"
    else
        fail "Summary mode should have at least 3 separator lines, got $separator_count"
    fi

    # Test 5e: Summary mode — does NOT include how-to-update (that's verbose only)
    if echo "$output" | grep -q "$MSG_HOW_TO_UPDATE:"; then
        fail "Summary mode should NOT show how-to-update section"
    else
        pass "Summary mode omits how-to-update section (verbose only)"
    fi

    # Test 5: Verbose mode — includes how-to-update instructions
    STARTUP_VERBOSITY="verbose"
    output=$(show_update_notification "v0.1.0" "v0.2.0" "https://example.com/releases")
    if echo "$output" | grep -q "$MSG_HOW_TO_UPDATE"; then
        pass "Verbose mode shows how-to-update section"
    else
        fail "Verbose mode should show how-to-update section, got: '$output'"
    fi

    # Test 6: Verbose mode — shows current and latest version
    if echo "$output" | grep -q "v0.1.0" && echo "$output" | grep -q "v0.2.0"; then
        pass "Verbose mode shows both current and latest version"
    else
        fail "Verbose mode should show both versions"
    fi

    # Test 7: No previous version — shows latest only (no arrow)
    STARTUP_VERBOSITY="quiet"
    output=$(show_update_notification "" "v0.3.0" "https://example.com/releases")
    if echo "$output" | grep -q "v0.3.0" && ! echo "$output" | grep -q "→"; then
        pass "No previous version shows latest only without arrow"
    else
        fail "No previous version should show 'v0.3.0' without '→', got: '$output'"
    fi
}

# ============================================================
# Test: Script is executable
# ============================================================
test_script_executable() {
    echo ""
    echo "=== Testing script is executable ==="

    local script="$WORKSPACE/.sandbox/scripts/check-upstream-updates.sh"

    if [ -f "$script" ]; then
        pass "check-upstream-updates.sh exists"
    else
        fail "check-upstream-updates.sh does not exist"
        return
    fi

    if [ -x "$script" ]; then
        pass "check-upstream-updates.sh is executable"
    else
        fail "check-upstream-updates.sh should be executable"
    fi

    # Check shebang
    if head -1 "$script" | grep -q "^#!/bin/bash"; then
        pass "check-upstream-updates.sh has correct shebang"
    else
        fail "check-upstream-updates.sh should have #!/bin/bash shebang"
    fi
}

# ============================================================
# Test: Script runs without error (with CHECK_UPDATES=false)
# ============================================================
test_script_runs() {
    echo ""
    echo "=== Testing script execution ==="

    local script="$WORKSPACE/.sandbox/scripts/check-upstream-updates.sh"
    local exit_code

    # Test 1: Run with CHECK_UPDATES=false via mock config
    local mock_config="$TEST_TMP_DIR/.sandbox/config/template-source.conf"
    cat > "$mock_config" <<'EOF'
TEMPLATE_REPO="YujiSuzuki/ai-sandbox-dkmcp"
CHECK_CHANNEL="all"
CHECK_UPDATES="false"
CHECK_INTERVAL_HOURS="0"
EOF

    (WORKSPACE="$TEST_TMP_DIR" "$script" >/dev/null 2>&1)
    exit_code=$?

    if [ $exit_code -eq 0 ]; then
        pass "check-upstream-updates.sh exits cleanly with CHECK_UPDATES=false"
    else
        fail "check-upstream-updates.sh should exit cleanly, got exit code $exit_code"
    fi

    # Test 2: Run with empty TEMPLATE_REPO
    cat > "$mock_config" <<'EOF'
TEMPLATE_REPO=""
CHECK_CHANNEL="all"
CHECK_UPDATES="true"
CHECK_INTERVAL_HOURS="0"
EOF

    (WORKSPACE="$TEST_TMP_DIR" "$script" >/dev/null 2>&1)
    exit_code=$?

    if [ $exit_code -eq 0 ]; then
        pass "check-upstream-updates.sh exits cleanly with empty TEMPLATE_REPO"
    else
        fail "check-upstream-updates.sh should exit cleanly with empty TEMPLATE_REPO, got exit code $exit_code"
    fi
}

# ============================================================
# Main
# ============================================================
main() {
    echo "========================================"
    echo "Update Check Tests"
    echo "========================================"

    setup

    test_config_file
    test_state_file
    test_interval_check
    test_build_api_url
    test_extract_tag_from_json
    test_debug_mode
    test_first_run_no_notification
    test_same_version_no_renotify
    test_show_update_notification
    test_script_executable
    test_script_runs

    teardown

    echo ""
    echo "========================================"
    echo "Test Results"
    echo "========================================"
    echo -e "Passed: ${GREEN}${TESTS_PASSED}${NC}"
    echo -e "Failed: ${RED}${TESTS_FAILED}${NC}"
    echo ""

    if [ $TESTS_FAILED -gt 0 ]; then
        exit 1
    fi
    exit 0
}

main "$@"

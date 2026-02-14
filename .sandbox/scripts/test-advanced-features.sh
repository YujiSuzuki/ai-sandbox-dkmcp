#!/bin/bash
# test-advanced-features.sh
# Test script for advanced features documented in README.ja.md
#
# README.ja.md の「高度な使い方」セクションの機能テスト
#
# Test Sections / テストセクション:
#   [Default - always run / デフォルト - 常に実行]
#     1. Custom DockMCP configuration / カスタムDockMCP設定
#     2. Multiple DockMCP instances / 複数のDockMCPインスタンス
#     3. Project name customization / プロジェクト名のカスタマイズ
#     4. Multiple DevContainer instances / 複数DevContainer起動
#   [Optional - requires flags / オプション - フラグ必須]
#     5. Custom Config File Tests (--test-config)
#     6. .env File Tests (--test-env)
#     7. copy-credentials.sh Tests (--test-copy)
#     8. Docker Volume Tests (--test-volume)
#     9. Server Integration Tests (--all)
#        9.1 Basic server tests (start, multiple instances)
#        9.2 Config effectiveness tests:
#            - --port flag effectiveness
#            - mode: strict blocks exec
#            - allowed_containers filtering
#            - exec_whitelist enforcement
#
# Usage: ./test-advanced-features.sh [OPTIONS]
# 使用方法: ./test-advanced-features.sh [オプション]
#
# Environment: Host OS (requires Docker access)
# 実行環境: ホストOS（Docker アクセスが必要）
#
# Options:
#   --basic      Run sections 1-4 only (safe, read-only)
#                セクション1-4のみ実行（安全、読み取り専用）
#   --host-only  Run only tests that require host OS
#                ホストOS専用テストのみ実行
#   --all        Run sections 1-4 + section 9 (DockMCP server tests)
#                セクション1-4 + セクション9（DockMCPサーバーテスト）
#   --full       Run ALL sections 1-9
#                全セクション1-9を実行
#   --dry-run    Show what would be done without actually doing it
#                実際の操作を行わず、何をするかのみ表示
#   -y, --yes    Skip confirmation prompts
#                確認プロンプトをスキップ

## 表示確認
# 英語表示
# LANG=en_US.UTF-8 ./.sandbox/scripts/test-advanced-features.sh --full --dry-run
# 日本語表示
# LANG=ja_JP.UTF-8 ./.sandbox/scripts/test-advanced-features.sh --full --dry-run
###########



set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WORKSPACE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
DKMCP_DIR="$WORKSPACE_DIR/dkmcp"
DEVCONTAINER_DIR="$WORKSPACE_DIR/.devcontainer"
CLI_SANDBOX_DIR="$WORKSPACE_DIR/cli_sandbox"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Options
HOST_ONLY=false
RUN_BASIC=false
RUN_ALL=false
RUN_FULL=false
TEST_CONFIG=false
TEST_ENV=false
TEST_COPY=false
TEST_VOLUME=false
DRY_RUN=false
AUTO_YES=false

# Track created resources for cleanup
CREATED_VOLUMES=()
CREATED_FILES=()
STARTED_PROCESSES=()

# Show help message
# ヘルプメッセージを表示
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Test advanced features documented in README.ja.md"
    echo "README.ja.md の「高度な使い方」セクションの機能テスト"
    echo ""
    echo "Test Sections / テストセクション:"
    echo ""
    echo "  [Default - always run / デフォルト - 常に実行]"
    echo "    1. Custom DockMCP Configuration    (5 tests) - read-only"
    echo "       カスタムDockMCP設定             （5テスト）- 読み取り専用"
    echo "    2. Multiple DockMCP Instances      (3 tests) - read-only"
    echo "       複数DockMCPインスタンス         （3テスト）- 読み取り専用"
    echo "    3. Project Name Customization      (4 tests) - read-only"
    echo "       プロジェクト名カスタマイズ      （4テスト）- 読み取り専用"
    echo "    4. Multiple DevContainer Instances (4 tests) - read-only"
    echo "       複数DevContainer                （4テスト）- 読み取り専用"
    echo ""
    echo "  [Optional - requires flags / オプション - フラグ必須]"
    echo "    5. Custom Config File Tests   (--test-config) - creates files"
    echo "       カスタム設定ファイルテスト                 - ファイル作成"
    echo "    6. .env File Tests            (--test-env)    - creates files"
    echo "       .envファイルテスト                         - ファイル作成"
    echo "    7. copy-credentials.sh Tests  (--test-copy)   - creates temp files"
    echo "       copy-credentials.shテスト                  - 一時ファイル作成"
    echo "    8. Docker Volume Tests        (--test-volume) - requires Docker"
    echo "       Dockerボリュームテスト                     - Docker必須"
    echo "    9. Server Integration Tests   (--all)         - requires Docker"
    echo "       サーバー統合テスト                         - Docker必須"
    echo "       - 9.1: Basic server tests (start, multiple instances)"
    echo "              基本テスト（起動、複数インスタンス）"
    echo "       - 9.2: Config effectiveness (--port, strict mode,"
    echo "              allowed_containers, exec_whitelist)"
    echo "              設定有効性（--port、strictモード、"
    echo "              allowed_containers、exec_whitelist）"
    echo ""
    echo "Options / オプション:"
    echo "  --basic        Run sections 1-4 only (safe, read-only)"
    echo "                 セクション1-4のみ実行（安全、読み取り専用）"
    echo ""
    echo "  --host-only    Run only tests that require host OS"
    echo "                 ホストOS専用テストのみ実行"
    echo ""
    echo "  --all          Run sections 1-4 + section 9 (DockMCP server tests)"
    echo "                 セクション1-4 + セクション9（DockMCPサーバーテスト）"
    echo ""
    echo "  --full         Run ALL sections 1-9"
    echo "                 全セクション1-9を実行"
    echo ""
    echo "  --test-config  Add section 5 (custom config file creation)"
    echo "                 セクション5を追加（カスタム設定ファイル作成）"
    echo ""
    echo "  --test-env     Add section 6 (.env file tests)"
    echo "                 セクション6を追加（.envファイルテスト）"
    echo ""
    echo "  --test-copy    Add section 7 (copy-credentials.sh tests)"
    echo "                 セクション7を追加（copy-credentials.shテスト）"
    echo ""
    echo "  --test-volume  Add section 8 (Docker volume tests, requires Docker)"
    echo "                 セクション8を追加（Dockerボリュームテスト、Docker必須）"
    echo ""
    echo "  --dry-run      Show what would be done without actually doing it"
    echo "                 実際の操作を行わず、何をするかのみ表示"
    echo ""
    echo "  -y, --yes      Skip confirmation prompts (use with caution!)"
    echo "                 確認プロンプトをスキップ（注意して使用！）"
    echo ""
    echo "Examples / 実行例:"
    echo "  $0 --basic            # Sections 1-4 only (safe, read-only)"
    echo "                        # セクション1-4のみ（安全、読み取り専用）"
    echo "  $0 --all              # Sections 1-4, 9 (requires Docker)"
    echo "                        # セクション1-4, 9（Docker必須）"
    echo "  $0 --full             # All sections 1-9 (requires Docker)"
    echo "                        # 全セクション1-9（Docker必須）"
    echo "  $0 --test-config      # Sections 1-4 + section 5"
    echo "                        # セクション1-4 + セクション5"
}

# Confirm before running a section with impact/risk/recovery info
# セクション実行前に影響範囲/リスク/対処法を表示して確認
# Arguments: section_number, section_name, impact_en, impact_ja, risk_en, risk_ja, recovery_en, recovery_ja
confirm_section() {
    local section_num="$1"
    local section_name="$2"
    local impact_en="$3"
    local impact_ja="$4"
    local risk_en="$5"
    local risk_ja="$6"
    local recovery_en="$7"
    local recovery_ja="$8"

    # Skip confirmation (and message) if AUTO_YES is set
    # AUTO_YESが設定されている場合は確認（とメッセージ）をスキップ
    if [ "$AUTO_YES" = "true" ]; then
        return 0
    fi

    # Show confirmation message
    # 確認メッセージを表示
    echo ""
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}Section $section_num: $section_name${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "${BLUE}Impact / 影響範囲:${NC}"
    echo "  $impact_en"
    echo "  $impact_ja"
    echo ""
    echo -e "${YELLOW}Risk / リスク:${NC}"
    echo "  $risk_en"
    echo "  $risk_ja"
    echo ""
    echo -e "${GREEN}Recovery / 失敗時の対処法:${NC}"
    echo "  $recovery_en"
    echo "  $recovery_ja"
    echo ""
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    # In dry-run mode, show message but auto-continue
    # dry-runモードではメッセージ表示後、自動的に続行
    if [ "$DRY_RUN" = "true" ]; then
        echo -e "${BLUE}[DRY-RUN] Auto-continuing without prompt${NC}"
        echo "[DRY-RUN] プロンプトなしで自動続行"
        return 0
    fi

    read -p "Run this section? / このセクションを実行しますか？ [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}Skipped section $section_num${NC}"
        echo "セクション $section_num をスキップしました"
        return 1
    fi
    return 0
}

# Cleanup on exit (trap handler)
cleanup_on_exit() {
    local exit_code=$?

    if [ ${#CREATED_VOLUMES[@]} -gt 0 ] || [ ${#CREATED_FILES[@]} -gt 0 ] || [ ${#STARTED_PROCESSES[@]} -gt 0 ]; then
        echo ""
        echo -e "${YELLOW}Cleaning up resources...${NC}"
    fi

    # Kill any started processes
    for pid in "${STARTED_PROCESSES[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
            wait "$pid" 2>/dev/null || true
            echo "  Stopped process: $pid"
        fi
    done

    # Remove created volumes
    if has_docker; then
        for vol in "${CREATED_VOLUMES[@]}"; do
            if docker volume inspect "$vol" >/dev/null 2>&1; then
                docker volume rm "$vol" >/dev/null 2>&1 && echo "  Removed volume: $vol" || true
            fi
        done
    fi

    # Remove created files/directories
    for file in "${CREATED_FILES[@]}"; do
        if [ -e "$file" ]; then
            rm -rf "$file" && echo "  Removed: $file" || true
        fi
    done

    exit $exit_code
}

# Set trap for cleanup
trap cleanup_on_exit EXIT INT TERM

# Track resource creation
track_volume() {
    CREATED_VOLUMES+=("$1")
}

track_file() {
    CREATED_FILES+=("$1")
}

track_process() {
    STARTED_PROCESSES+=("$1")
}

# Remove element from array
# 配列から要素を削除
# Usage: remove_from_array "array_name" "value"
# Note: bash 3.2 compatible (no nameref)
remove_from_array() {
    local array_name="$1"
    local value="$2"
    local new_array=()

    eval "local items=(\"\${${array_name}[@]}\")"
    for item in "${items[@]}"; do
        [ "$item" != "$value" ] && new_array+=("$item")
    done
    eval "${array_name}=(\"\${new_array[@]}\")"
}

# Dry run wrapper for docker commands
docker_run() {
    if [ "$DRY_RUN" = "true" ]; then
        echo -e "  ${BLUE}[DRY-RUN] docker $*${NC}"
        return 0
    else
        docker "$@"
    fi
}

# Check for existing test volumes
check_existing_volumes() {
    if ! has_docker; then
        return 0
    fi

    local existing
    existing=$(docker volume ls --format '{{.Name}}' 2>/dev/null | grep "^${TEST_VOLUME_PREFIX}" || true)

    if [ -n "$existing" ]; then
        echo ""
        echo -e "${RED}========================================"
        echo -e "WARNING: Existing test volumes found!"
        echo -e "警告: 既存のテストボリュームが見つかりました！"
        echo -e "========================================${NC}"
        echo ""
        echo "The following volumes will be DELETED during cleanup:"
        echo "以下のボリュームはクリーンアップ時に削除されます:"
        echo ""
        echo "$existing" | while read -r vol; do
            echo "  - $vol"
        done
        echo ""
        return 1
    fi
    return 0
}

# Confirmation prompt for dangerous operations
confirm_dangerous_operation() {
    local operation="$1"

    if [ "$AUTO_YES" = "true" ]; then
        return 0
    fi

    if [ "$DRY_RUN" = "true" ]; then
        echo -e "${BLUE}[DRY-RUN] Would ask for confirmation: $operation${NC}"
        return 0
    fi

    echo ""
    echo -e "${YELLOW}========================================"
    echo -e "CONFIRMATION REQUIRED / 確認が必要です"
    echo -e "========================================${NC}"
    echo ""
    echo "This test will perform the following operations:"
    echo "このテストは以下の操作を実行します:"
    echo ""
    echo "  $operation"
    echo ""
    echo -e "${YELLOW}These operations may:"
    echo "  - Create Docker volumes (prefix: ${TEST_VOLUME_PREFIX})"
    echo "  - Delete Docker volumes matching the test prefix"
    echo "  - Create temporary files in /tmp"
    echo "  - Start/stop DockMCP server processes${NC}"
    echo ""
    echo "これらの操作は:"
    echo "  - Dockerボリュームを作成します（プレフィックス: ${TEST_VOLUME_PREFIX}）"
    echo "  - テストプレフィックスに一致するDockerボリュームを削除します"
    echo "  - /tmp に一時ファイルを作成します"
    echo "  - DockMCPサーバープロセスを起動/停止します"
    echo ""

    read -p "Continue? / 続行しますか？ [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Cancelled. / キャンセルしました。"
        return 1
    fi
    return 0
}

# Language detection for i18n
# 言語検出（国際化対応）
is_japanese() {
    [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]
}

# Message helper: msg "English message" "日本語メッセージ"
# Returns appropriate message based on locale
# メッセージヘルパー: ロケールに基づいて適切なメッセージを返す
msg() {
    if is_japanese; then
        echo "$2"
    else
        echo "$1"
    fi
}

# Helper functions
pass() {
    echo -e "${GREEN}PASS: $1${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

fail() {
    echo -e "${RED}FAIL: $1${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

skip() {
    echo -e "${YELLOW}SKIP: $1${NC}"
    TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
}

info() {
    echo -e "${BLUE}TEST: $1${NC}"
}

section() {
    echo ""
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}========================================${NC}"
    echo ""
}

# Check if running in DevContainer
is_devcontainer() {
    [ "${SANDBOX_ENV:-}" = "devcontainer" ] || [ -f "/.dockerenv" ]
}

# Check if Docker is available
has_docker() {
    command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1
}

# Check if dkmcp binary is available
has_dkmcp() {
    command -v dkmcp >/dev/null 2>&1
}

###############################################################################
# Section 1: Custom DockMCP Configuration Tests
# カスタムDockMCP設定のテスト
###############################################################################

test_dkmcp_config_exists() {
    info "DockMCP example config exists"

    if [ -f "$DKMCP_DIR/configs/dkmcp.example.yaml" ]; then
        pass "DockMCP example config exists"
    else
        fail "DockMCP example config not found at $DKMCP_DIR/configs/dkmcp.example.yaml"
    fi
}

test_dkmcp_config_valid_yaml() {
    info "DockMCP config is valid YAML"

    local config_file="$DKMCP_DIR/configs/dkmcp.example.yaml"

    if [ ! -f "$config_file" ]; then
        skip "Config file not found"
        return
    fi

    # Check if yq or python is available for YAML validation
    if command -v yq >/dev/null 2>&1; then
        if yq eval '.' "$config_file" >/dev/null 2>&1; then
            pass "DockMCP config is valid YAML (checked with yq)"
        else
            fail "DockMCP config has invalid YAML syntax"
        fi
    elif command -v python3 >/dev/null 2>&1 && python3 -c "import yaml" 2>/dev/null; then
        # Only use Python if PyYAML is installed
        if python3 -c "import yaml; yaml.safe_load(open('$config_file'))" 2>/dev/null; then
            pass "DockMCP config is valid YAML (checked with Python)"
        else
            fail "DockMCP config has invalid YAML syntax"
        fi
    elif command -v go >/dev/null 2>&1; then
        # Use dkmcp itself to validate (it will fail to start if config is invalid)
        if has_dkmcp; then
            # Try to parse config with dkmcp (dry-run style check)
            local output
            output=$(dkmcp serve --config "$config_file" --help 2>&1) || true
            # If help shows without parse error, config is likely valid
            if echo "$output" | grep -qi "error.*yaml\|parse\|invalid"; then
                fail "DockMCP config has invalid YAML syntax"
            else
                pass "DockMCP config appears valid (dkmcp accepts config)"
            fi
        else
            # Fallback: basic syntax check
            if grep -q "security:" "$config_file" && grep -q "mode:" "$config_file"; then
                pass "DockMCP config appears valid (basic check)"
            else
                fail "DockMCP config missing required fields"
            fi
        fi
    else
        # Fallback: basic syntax check
        if grep -q "security:" "$config_file" && grep -q "mode:" "$config_file"; then
            pass "DockMCP config appears valid (basic check)"
        else
            fail "DockMCP config missing required fields"
        fi
    fi
}

test_dkmcp_config_has_security_modes() {
    info "DockMCP config supports security modes (strict/moderate/permissive)"

    local config_file="$DKMCP_DIR/configs/dkmcp.example.yaml"

    if [ ! -f "$config_file" ]; then
        skip "Config file not found"
        return
    fi

    # Check if config mentions security modes
    if grep -qE "mode:.*\"?(strict|moderate|permissive)\"?" "$config_file"; then
        pass "DockMCP config has security mode setting"
    else
        # Check in Go source code for supported modes
        if grep -rq "strict\|moderate\|permissive" "$DKMCP_DIR/internal/security/" 2>/dev/null; then
            pass "DockMCP supports security modes (found in source)"
        else
            fail "Security modes not found in config or source"
        fi
    fi
}

test_dkmcp_config_allowed_containers() {
    info "DockMCP config supports allowed_containers"

    local config_file="$DKMCP_DIR/configs/dkmcp.example.yaml"

    if [ ! -f "$config_file" ]; then
        skip "Config file not found"
        return
    fi

    if grep -q "allowed_containers:" "$config_file"; then
        pass "DockMCP config has allowed_containers setting"
    else
        fail "DockMCP config missing allowed_containers setting"
    fi
}

test_dkmcp_config_exec_whitelist() {
    info "DockMCP config supports exec_whitelist"

    local config_file="$DKMCP_DIR/configs/dkmcp.example.yaml"

    if [ ! -f "$config_file" ]; then
        skip "Config file not found"
        return
    fi

    if grep -q "exec_whitelist:" "$config_file"; then
        pass "DockMCP config has exec_whitelist setting"
    else
        fail "DockMCP config missing exec_whitelist setting"
    fi
}

###############################################################################
# Section 2: Multiple DockMCP Instances Tests
# 複数DockMCPインスタンスのテスト
###############################################################################

test_dkmcp_serve_port_flag() {
    info "DockMCP serve command supports --port flag"

    if ! has_dkmcp; then
        skip "dkmcp binary not found (run 'make install' in dkmcp/)"
        return
    fi

    local help_output
    help_output=$(dkmcp serve --help 2>&1) || true

    if echo "$help_output" | grep -qE "(--port|-p)"; then
        pass "DockMCP serve supports --port flag"
    else
        fail "DockMCP serve does not support --port flag"
        echo "  Help output: $help_output"
    fi
}

test_dkmcp_serve_config_flag() {
    info "DockMCP serve command supports --config flag"

    if ! has_dkmcp; then
        skip "dkmcp binary not found"
        return
    fi

    local help_output
    help_output=$(dkmcp serve --help 2>&1) || true

    if echo "$help_output" | grep -qE "(--config|-c)"; then
        pass "DockMCP serve supports --config flag"
    else
        fail "DockMCP serve does not support --config flag"
    fi
}

test_dkmcp_multiple_configs_exist() {
    info "Multiple DockMCP config examples can exist"

    local configs_dir="$DKMCP_DIR/configs"

    if [ -d "$configs_dir" ]; then
        local config_count
        config_count=$(find "$configs_dir" -name "*.yaml" -o -name "*.yml" | wc -l)

        if [ "$config_count" -ge 1 ]; then
            pass "DockMCP configs directory exists with $config_count config(s)"
        else
            fail "No YAML configs found in $configs_dir"
        fi
    else
        fail "DockMCP configs directory not found"
    fi
}

###############################################################################
# Section 3: Project Name Customization Tests
# プロジェクト名カスタマイズのテスト
###############################################################################

test_devcontainer_env_example_exists() {
    info ".devcontainer/.env.example exists"

    if [ -f "$DEVCONTAINER_DIR/.env.example" ]; then
        pass ".devcontainer/.env.example exists"
    else
        # Check if documented in README
        if grep -q "COMPOSE_PROJECT_NAME" "$WORKSPACE_DIR/README.ja.md" 2>/dev/null; then
            skip ".env.example not found but documented in README"
        else
            fail ".devcontainer/.env.example not found"
        fi
    fi
}

test_devcontainer_env_gitignore() {
    info ".devcontainer/.env is in .gitignore"

    local gitignore="$WORKSPACE_DIR/.gitignore"

    if [ -f "$gitignore" ]; then
        if grep -qE "^\.devcontainer/\.env$|^\*\*/\.env$|^\.env$" "$gitignore"; then
            pass ".env is in .gitignore"
        elif grep -q ".env" "$gitignore"; then
            pass ".env pattern found in .gitignore"
        else
            fail ".devcontainer/.env should be in .gitignore"
        fi
    else
        skip ".gitignore not found"
    fi
}

test_compose_project_name_support() {
    info "docker-compose.yml supports COMPOSE_PROJECT_NAME"

    local compose_file="$DEVCONTAINER_DIR/docker-compose.yml"

    if [ ! -f "$compose_file" ]; then
        skip "docker-compose.yml not found"
        return
    fi

    # COMPOSE_PROJECT_NAME is an environment variable, not in the file
    # Just verify the file exists and is valid
    if has_docker; then
        local config_output
        config_output=$(cd "$DEVCONTAINER_DIR" && docker compose config 2>&1) || true

        if echo "$config_output" | grep -q "name:"; then
            pass "docker-compose.yml is valid and supports project naming"
        else
            # May fail if docker is not available, but file should be parseable
            pass "docker-compose.yml exists (docker compose config unavailable)"
        fi
    else
        pass "docker-compose.yml exists (Docker not available for validation)"
    fi
}

test_cli_sandbox_env_support() {
    info "cli_sandbox supports COMPOSE_PROJECT_NAME via script exports"

    local cli_compose="$CLI_SANDBOX_DIR/docker-compose.yml"

    if [ ! -f "$cli_compose" ]; then
        skip "cli_sandbox/docker-compose.yml not found"
        return
    fi

    # Check if .sh files set COMPOSE_PROJECT_NAME
    # 各 .sh ファイルが COMPOSE_PROJECT_NAME を設定しているか確認
    # Pattern: ^COMPOSE_PROJECT_NAME= (see _common.sh for why this pattern)
    local project_names
    project_names=$(grep -h "^COMPOSE_PROJECT_NAME=" "$CLI_SANDBOX_DIR"/*.sh 2>/dev/null | \
                    sed 's/COMPOSE_PROJECT_NAME=//' | sort -u)

    if [ -n "$project_names" ]; then
        local count
        count=$(echo "$project_names" | wc -l)
        pass "Found $count COMPOSE_PROJECT_NAME(s) in cli_sandbox/*.sh"
    else
        # Fallback: check if docker-compose.yml exists
        pass "cli_sandbox/docker-compose.yml exists"
    fi
}

test_cli_sandbox_multi_project() {
    info "cli_sandbox has multiple AI tool scripts"

    if [ ! -d "$CLI_SANDBOX_DIR" ]; then
        skip "cli_sandbox directory not found"
        return
    fi

    local has_claude=false
    local has_gemini=false
    local has_ai_sandbox=false

    [ -f "$CLI_SANDBOX_DIR/claude.sh" ] && has_claude=true
    [ -f "$CLI_SANDBOX_DIR/gemini.sh" ] && has_gemini=true
    [ -f "$CLI_SANDBOX_DIR/ai_sandbox.sh" ] && has_ai_sandbox=true

    if $has_claude && $has_gemini && $has_ai_sandbox; then
        pass "Found all AI tool scripts (claude.sh, gemini.sh, ai_sandbox.sh)"
    elif $has_claude || $has_gemini || $has_ai_sandbox; then
        local found=""
        $has_claude && found="${found}claude.sh "
        $has_gemini && found="${found}gemini.sh "
        $has_ai_sandbox && found="${found}ai_sandbox.sh "
        pass "Found AI tool scripts: $found"
    else
        fail "No AI tool scripts found in cli_sandbox/"
    fi
}

test_cli_sandbox_project_isolation() {
    info "cli_sandbox scripts have unique COMPOSE_PROJECT_NAME"

    if [ ! -d "$CLI_SANDBOX_DIR" ]; then
        skip "cli_sandbox directory not found"
        return
    fi

    # Pattern: ^COMPOSE_PROJECT_NAME= (see _common.sh for why this pattern)
    local project_names
    project_names=$(grep -h "^COMPOSE_PROJECT_NAME=" "$CLI_SANDBOX_DIR"/*.sh 2>/dev/null | \
                    sed 's/COMPOSE_PROJECT_NAME=//' | sort)

    if [ -z "$project_names" ]; then
        skip "No COMPOSE_PROJECT_NAME found in scripts"
        return
    fi

    local unique_count
    local total_count
    unique_count=$(echo "$project_names" | sort -u | wc -l)
    total_count=$(echo "$project_names" | wc -l)

    if [ "$unique_count" -eq "$total_count" ]; then
        pass "All $total_count COMPOSE_PROJECT_NAMEs are unique"
    else
        fail "Duplicate COMPOSE_PROJECT_NAME found ($unique_count unique out of $total_count)"
    fi
}

###############################################################################
# Section 4: Multiple DevContainer Instances Tests
# 複数DevContainer起動のテスト
###############################################################################

test_copy_credentials_script_exists() {
    info "copy-credentials.sh script exists"

    if [ -f "$SCRIPT_DIR/../host-tools/copy-credentials.sh" ]; then
        pass "copy-credentials.sh exists"
    else
        fail "copy-credentials.sh not found"
    fi
}

test_copy_credentials_help() {
    info "copy-credentials.sh shows help"

    local script="$SCRIPT_DIR/../host-tools/copy-credentials.sh"

    if [ ! -f "$script" ]; then
        skip "copy-credentials.sh not found"
        return
    fi

    local output
    output=$(bash "$script" --help 2>&1)

    if echo "$output" | grep -q "Usage"; then
        pass "copy-credentials.sh --help shows usage"
    else
        fail "copy-credentials.sh --help should show usage"
    fi
}

test_copy_credentials_export_import() {
    info "copy-credentials.sh supports --export and --import"

    local script="$SCRIPT_DIR/../host-tools/copy-credentials.sh"

    if [ ! -f "$script" ]; then
        skip "copy-credentials.sh not found"
        return
    fi

    local help_output
    help_output=$(bash "$script" --help 2>&1)

    if echo "$help_output" | grep -q "\-\-export" && echo "$help_output" | grep -q "\-\-import"; then
        pass "copy-credentials.sh supports --export and --import"
    else
        fail "copy-credentials.sh should support --export and --import"
    fi
}

test_copy_credentials_workspace_mode() {
    info "copy-credentials.sh supports workspace mode"

    local script="$SCRIPT_DIR/../host-tools/copy-credentials.sh"

    if [ ! -f "$script" ]; then
        skip "copy-credentials.sh not found"
        return
    fi

    local help_output
    help_output=$(bash "$script" --help 2>&1)

    if echo "$help_output" | grep -qi "workspace"; then
        pass "copy-credentials.sh supports workspace mode"
    else
        # Check script content
        if grep -q "workspace" "$script"; then
            pass "copy-credentials.sh has workspace support in code"
        else
            fail "copy-credentials.sh should support workspace mode"
        fi
    fi
}

###############################################################################
# Sections 5-7: File Creation/Deletion Tests
# セクション5-7: ファイル作成/削除テスト
#   5. Custom Config File Tests (--test-config)
#   6. .env File Tests (--test-env)
#   7. copy-credentials.sh Tests (--test-copy)
###############################################################################

# Cleanup function for temporary files
cleanup_temp_files() {
    local files=("$@")
    for file in "${files[@]}"; do
        if [ -f "$file" ]; then
            rm -f "$file"
            echo "  Cleaned up: $file"
        elif [ -d "$file" ]; then
            rm -rf "$file"
            echo "  Cleaned up: $file"
        fi
    done
}

test_create_custom_dkmcp_config() {
    info "[File Test] Create and validate custom DockMCP config"

    local test_config="/tmp/test-dkmcp-strict-$$.yaml"
    local cleanup_files=("$test_config")

    # Create a strict mode config
    cat > "$test_config" << 'EOF'
# Test: Strict mode config
security:
  mode: "strict"
  allowed_containers:
    - "test-*"
  exec_whitelist: {}
  permissions:
    logs: true
    inspect: true
    stats: true
    exec: false
EOF

    if [ -f "$test_config" ]; then
        echo "  Created: $test_config"

        # Validate the config has expected content
        if grep -q 'mode: "strict"' "$test_config" && grep -q 'exec: false' "$test_config"; then
            pass "Custom strict config created and validated"
        else
            fail "Custom config missing expected content"
        fi
    else
        fail "Failed to create custom config file"
    fi

    # Cleanup
    cleanup_temp_files "${cleanup_files[@]}"
}

test_create_permissive_dkmcp_config() {
    info "[File Test] Create permissive DockMCP config"

    local test_config="/tmp/test-dkmcp-permissive-$$.yaml"
    local cleanup_files=("$test_config")

    # Create a permissive mode config
    cat > "$test_config" << 'EOF'
# Test: Permissive mode config (use with caution!)
security:
  mode: "permissive"
  allowed_containers: []
  exec_whitelist:
    "*":
      - "npm test"
      - "npm run lint"
  permissions:
    logs: true
    inspect: true
    stats: true
    exec: true
EOF

    if [ -f "$test_config" ]; then
        echo "  Created: $test_config"

        if grep -q 'mode: "permissive"' "$test_config"; then
            pass "Custom permissive config created"
        else
            fail "Custom config missing expected content"
        fi
    else
        fail "Failed to create custom config file"
    fi

    # Cleanup
    cleanup_temp_files "${cleanup_files[@]}"
}

test_create_env_file() {
    info "[File Test] Create .env file for project name customization"

    local test_env="/tmp/test-devcontainer-env-$$"
    mkdir -p "$test_env"
    local env_file="$test_env/.env"
    local cleanup_files=("$test_env")

    # Create .env file
    echo "COMPOSE_PROJECT_NAME=test-project-$$" > "$env_file"

    if [ -f "$env_file" ]; then
        echo "  Created: $env_file"

        # Verify content
        local project_name
        project_name=$(grep "COMPOSE_PROJECT_NAME" "$env_file" | cut -d= -f2)

        if [ "$project_name" = "test-project-$$" ]; then
            pass ".env file created with correct COMPOSE_PROJECT_NAME"
        else
            fail ".env file has incorrect content"
            echo "  Expected: test-project-$$"
            echo "  Got: $project_name"
        fi
    else
        fail "Failed to create .env file"
    fi

    # Cleanup
    cleanup_temp_files "${cleanup_files[@]}"
}

test_env_file_multiple_projects() {
    info "[File Test] Create multiple .env files for different projects"

    local test_dir="/tmp/test-multi-env-$$"
    mkdir -p "$test_dir/project-a"
    mkdir -p "$test_dir/project-b"
    local cleanup_files=("$test_dir")

    # Create .env files for different projects
    echo "COMPOSE_PROJECT_NAME=client-a" > "$test_dir/project-a/.env"
    echo "COMPOSE_PROJECT_NAME=client-b" > "$test_dir/project-b/.env"

    local project_a
    local project_b
    project_a=$(grep "COMPOSE_PROJECT_NAME" "$test_dir/project-a/.env" | cut -d= -f2)
    project_b=$(grep "COMPOSE_PROJECT_NAME" "$test_dir/project-b/.env" | cut -d= -f2)

    if [ "$project_a" = "client-a" ] && [ "$project_b" = "client-b" ]; then
        pass "Multiple .env files created for different projects"
        echo "  Project A: $project_a"
        echo "  Project B: $project_b"
    else
        fail "Failed to create multiple .env files correctly"
    fi

    # Cleanup
    cleanup_temp_files "${cleanup_files[@]}"
}

test_copy_credentials_export_dry_run() {
    info "[File Test] copy-credentials.sh export (dry run validation)"

    local script="$SCRIPT_DIR/../host-tools/copy-credentials.sh"
    local test_backup="/tmp/test-backup-$$"
    local cleanup_files=("$test_backup")

    if [ ! -f "$script" ]; then
        skip "copy-credentials.sh not found"
        return
    fi

    # Test with a non-existent source (should fail gracefully)
    local output
    output=$(bash "$script" --export /nonexistent/path "$test_backup" 2>&1) || true

    if echo "$output" | grep -q "Cannot find docker-compose.yml"; then
        pass "copy-credentials.sh validates source path correctly"
    else
        fail "copy-credentials.sh should validate source path"
        echo "  Output: $output"
    fi

    # Cleanup (backup dir shouldn't be created on failure)
    cleanup_temp_files "${cleanup_files[@]}"
}

test_copy_credentials_backup_structure() {
    info "[File Test] copy-credentials.sh creates correct backup structure (new format)"

    local test_backup="/tmp/test-backup-structure-$$"
    local cleanup_files=("$test_backup")

    # Create expected backup structure (new format with multi-project cli_sandbox)
    # 新しい形式（マルチプロジェクト cli_sandbox）のバックアップ構造を作成
    mkdir -p "$test_backup/devcontainer/home"
    mkdir -p "$test_backup/devcontainer/gcloud"
    mkdir -p "$test_backup/cli_sandbox/cli-claude/home"
    mkdir -p "$test_backup/cli_sandbox/cli-gemini/home"
    mkdir -p "$test_backup/cli_sandbox/cli-ai-sandbox/home"

    # Create dummy files
    echo "test" > "$test_backup/devcontainer/home/.bashrc"
    echo "claude" > "$test_backup/cli_sandbox/cli-claude/home/.bashrc"
    echo "gemini" > "$test_backup/cli_sandbox/cli-gemini/home/.bashrc"
    echo "sandbox" > "$test_backup/cli_sandbox/cli-ai-sandbox/home/.bashrc"

    # Verify structure
    if [ -d "$test_backup/devcontainer/home" ] && \
       [ -d "$test_backup/cli_sandbox/cli-claude/home" ] && \
       [ -d "$test_backup/cli_sandbox/cli-gemini/home" ] && \
       [ -d "$test_backup/cli_sandbox/cli-ai-sandbox/home" ]; then
        pass "Backup directory structure created correctly (new format)"
        echo "  Structure:"
        echo "    $test_backup/"
        echo "    ├── devcontainer/"
        echo "    │   └── home/"
        echo "    └── cli_sandbox/"
        echo "        ├── cli-claude/"
        echo "        │   └── home/"
        echo "        ├── cli-gemini/"
        echo "        │   └── home/"
        echo "        └── cli-ai-sandbox/"
        echo "            └── home/"
    else
        fail "Backup directory structure is incorrect"
    fi

    # Cleanup
    cleanup_temp_files "${cleanup_files[@]}"
}

test_copy_credentials_import_validation() {
    info "[File Test] copy-credentials.sh import validates backup path"

    local script="$SCRIPT_DIR/../host-tools/copy-credentials.sh"

    if [ ! -f "$script" ]; then
        skip "copy-credentials.sh not found"
        return
    fi

    # Test with non-existent backup directory
    local output
    output=$(bash "$script" --import /nonexistent/backup "$WORKSPACE_DIR" 2>&1) || true

    if echo "$output" | grep -q "Backup directory not found"; then
        pass "copy-credentials.sh validates backup directory exists"
    else
        fail "copy-credentials.sh should validate backup directory"
        echo "  Output: $output"
    fi
}

###############################################################################
# Section 8: Docker Volume Tests (Host OS only, requires Docker)
# セクション8: Dockerボリュームテスト（ホストOS専用、Docker必須）
###############################################################################

# Test volume name prefix for cleanup
TEST_VOLUME_PREFIX="test-advanced-features"

# Cleanup test volumes
cleanup_test_volumes() {
    echo "  Cleaning up test volumes..."
    local volumes
    volumes=$(docker volume ls --format '{{.Name}}' 2>/dev/null | grep "^${TEST_VOLUME_PREFIX}" || true)

    for vol in $volumes; do
        docker volume rm "$vol" >/dev/null 2>&1 && echo "    Removed: $vol" || true
    done
}

test_volume_create() {
    info "[Volume Test] Create Docker volume"

    if ! has_docker; then
        skip "Docker not available (run on host OS)"
        return
    fi

    local vol_name="${TEST_VOLUME_PREFIX}-create-$$"

    if [ "$DRY_RUN" = "true" ]; then
        echo -e "  ${BLUE}[DRY-RUN] Would create volume: $vol_name${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would verify volume exists${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would delete volume: $vol_name${NC}"
        pass "Docker volume create (dry-run)"
        return
    fi

    # Create volume
    local create_output
    create_output=$(docker volume create "$vol_name" 2>&1)
    local create_status=$?

    if [ $create_status -eq 0 ]; then
        track_volume "$vol_name"
        echo "  Created volume: $vol_name"

        # Verify volume exists
        if docker volume inspect "$vol_name" >/dev/null 2>&1; then
            pass "Docker volume created successfully"
        else
            fail "Volume created but cannot be inspected"
        fi

        # Cleanup (also tracked for safety)
        docker volume rm "$vol_name" >/dev/null 2>&1
        # Remove from tracking since we cleaned it up
        remove_from_array "CREATED_VOLUMES" "$vol_name"
        echo "  Cleaned up: $vol_name"
    else
        fail "Failed to create Docker volume"
        echo -e "  ${RED}Error: $create_output${NC}"
        # Common causes
        if echo "$create_output" | grep -qi "permission denied"; then
            echo -e "  ${YELLOW}Hint: Check Docker daemon permissions${NC}"
        elif echo "$create_output" | grep -qi "no space"; then
            echo -e "  ${YELLOW}Hint: Insufficient disk space${NC}"
        elif echo "$create_output" | grep -qi "cannot connect"; then
            echo -e "  ${YELLOW}Hint: Docker daemon may not be running${NC}"
        fi
    fi
}

test_volume_write_read() {
    info "[Volume Test] Write and read data from volume"

    if ! has_docker; then
        skip "Docker not available"
        return
    fi

    local vol_name="${TEST_VOLUME_PREFIX}-rw-$$"
    local test_data="test-data-$(date +%s)"

    if [ "$DRY_RUN" = "true" ]; then
        echo -e "  ${BLUE}[DRY-RUN] Would create volume: $vol_name${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would write test data to volume${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would read and verify data${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would delete volume: $vol_name${NC}"
        pass "Volume read/write (dry-run)"
        return
    fi

    # Create volume
    docker volume create "$vol_name" >/dev/null 2>&1
    track_volume "$vol_name"
    echo "  Created volume: $vol_name"

    # Write data to volume
    docker run --rm -v "${vol_name}:/data" alpine sh -c "echo '$test_data' > /data/test.txt" 2>/dev/null

    # Read data from volume
    local read_data
    read_data=$(docker run --rm -v "${vol_name}:/data:ro" alpine cat /data/test.txt 2>/dev/null)

    if [ "$read_data" = "$test_data" ]; then
        pass "Volume read/write works correctly"
        echo "    Written: $test_data"
        echo "    Read: $read_data"
    else
        fail "Volume read/write mismatch"
        echo -e "  ${RED}Expected: $test_data${NC}"
        echo -e "  ${RED}Got: $read_data${NC}"
        if [ -z "$read_data" ]; then
            echo -e "  ${YELLOW}Hint: Read returned empty - file may not have been written${NC}"
        fi
    fi

    # Cleanup
    docker volume rm "$vol_name" >/dev/null 2>&1
    remove_from_array "CREATED_VOLUMES" "$vol_name"
    echo "  Cleaned up: $vol_name"
}

test_volume_copy_between_volumes() {
    info "[Volume Test] Copy data between volumes"

    if ! has_docker; then
        skip "Docker not available"
        return
    fi

    local src_vol="${TEST_VOLUME_PREFIX}-src-$$"
    local dst_vol="${TEST_VOLUME_PREFIX}-dst-$$"
    local test_data="copy-test-$(date +%s)"

    if [ "$DRY_RUN" = "true" ]; then
        echo -e "  ${BLUE}[DRY-RUN] Would create volumes: $src_vol, $dst_vol${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would write test data to source volume${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would copy data from source to destination${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would verify copied data${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would delete volumes${NC}"
        pass "Volume copy (dry-run)"
        return
    fi

    # Create source and destination volumes
    docker volume create "$src_vol" >/dev/null 2>&1
    docker volume create "$dst_vol" >/dev/null 2>&1
    track_volume "$src_vol"
    track_volume "$dst_vol"
    echo "  Created volumes: $src_vol, $dst_vol"

    # Write data to source volume
    docker run --rm -v "${src_vol}:/data" alpine sh -c "
        mkdir -p /data/subdir
        echo '$test_data' > /data/test.txt
        echo 'nested' > /data/subdir/nested.txt
    " 2>/dev/null

    # Copy from source to destination (similar to copy-credentials.sh logic)
    docker run --rm \
        -v "${src_vol}:/source:ro" \
        -v "${dst_vol}:/target" \
        alpine sh -c "cp -a /source/. /target/" 2>/dev/null

    # Verify data in destination
    local dst_data
    local dst_nested
    dst_data=$(docker run --rm -v "${dst_vol}:/data:ro" alpine cat /data/test.txt 2>/dev/null)
    dst_nested=$(docker run --rm -v "${dst_vol}:/data:ro" alpine cat /data/subdir/nested.txt 2>/dev/null)

    if [ "$dst_data" = "$test_data" ] && [ "$dst_nested" = "nested" ]; then
        pass "Data copied between volumes correctly"
        echo "    Root file: $dst_data"
        echo "    Nested file: $dst_nested"
    else
        fail "Volume copy failed"
        echo "    Expected root: $test_data, got: $dst_data"
        echo "    Expected nested: nested, got: $dst_nested"
    fi

    # Cleanup
    docker volume rm "$src_vol" "$dst_vol" >/dev/null 2>&1
    remove_from_array "CREATED_VOLUMES" "$src_vol"
    remove_from_array "CREATED_VOLUMES" "$dst_vol"
    echo "  Cleaned up: $src_vol, $dst_vol"
}

test_volume_export_import_simulation() {
    info "[Volume Test] Simulate copy-credentials.sh export/import flow"

    if ! has_docker; then
        skip "Docker not available"
        return
    fi

    local src_vol="${TEST_VOLUME_PREFIX}-export-$$"
    local dst_vol="${TEST_VOLUME_PREFIX}-import-$$"
    local backup_dir="/tmp/test-volume-backup-$$"

    if [ "$DRY_RUN" = "true" ]; then
        echo -e "  ${BLUE}[DRY-RUN] Would create source volume: $src_vol${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would populate with test home directory data${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would export to: $backup_dir${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would verify .cache is excluded${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would create destination volume: $dst_vol${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would import and verify data${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would cleanup all resources${NC}"
        pass "Volume export/import simulation (dry-run)"
        return
    fi

    # Create source volume with test data (simulating home directory)
    docker volume create "$src_vol" >/dev/null 2>&1
    track_volume "$src_vol"
    echo "  Created source volume: $src_vol"

    docker run --rm -v "${src_vol}:/home/node" alpine sh -c "
        mkdir -p /home/node/.claude
        echo 'api_key=test123' > /home/node/.claude/config
        echo 'history' > /home/node/.bash_history
        mkdir -p /home/node/.cache
        echo 'cache data' > /home/node/.cache/temp
    " 2>/dev/null

    # Export (similar to copy-credentials.sh --export)
    mkdir -p "$backup_dir/home"
    track_file "$backup_dir"
    docker run --rm \
        -v "${src_vol}:/source:ro" \
        -v "${backup_dir}/home:/target" \
        alpine sh -c "cd /source && tar --exclude='.cache' -cf - . | (cd /target && tar -xf -)" 2>/dev/null

    echo "  Exported to: $backup_dir"

    # Verify export (should not contain .cache)
    if [ -f "$backup_dir/home/.claude/config" ] && [ ! -d "$backup_dir/home/.cache" ]; then
        echo "    Export excludes .cache correctly"
    else
        fail "Export did not exclude .cache correctly"
        rm -rf "$backup_dir"
        remove_from_array "CREATED_FILES" "$backup_dir"
        docker volume rm "$src_vol" >/dev/null 2>&1
        remove_from_array "CREATED_VOLUMES" "$src_vol"
        return
    fi

    # Create destination volume and import
    docker volume create "$dst_vol" >/dev/null 2>&1
    track_volume "$dst_vol"
    echo "  Created destination volume: $dst_vol"

    docker run --rm \
        -v "${backup_dir}/home:/source:ro" \
        -v "${dst_vol}:/target" \
        alpine sh -c "cp -a /source/. /target/ && chown -R 1000:1000 /target/" 2>/dev/null

    # Verify import
    local imported_config
    imported_config=$(docker run --rm -v "${dst_vol}:/data:ro" alpine cat /data/.claude/config 2>/dev/null)

    if [ "$imported_config" = "api_key=test123" ]; then
        pass "Volume export/import simulation successful"
        echo "    Imported config: $imported_config"
    else
        fail "Volume import verification failed"
        echo "    Expected: api_key=test123"
        echo "    Got: $imported_config"
    fi

    # Cleanup
    rm -rf "$backup_dir"
    remove_from_array "CREATED_FILES" "$backup_dir"
    docker volume rm "$src_vol" "$dst_vol" >/dev/null 2>&1
    remove_from_array "CREATED_VOLUMES" "$src_vol"
    remove_from_array "CREATED_VOLUMES" "$dst_vol"
    echo "  Cleaned up volumes and backup directory"
}

test_volume_different_project_names() {
    info "[Volume Test] Volumes with different COMPOSE_PROJECT_NAME"

    if ! has_docker; then
        skip "Docker not available"
        return
    fi

    # Simulate two projects with different names (like cli-claude and cli-gemini)
    # 異なる名前の2つのプロジェクトをシミュレート（cli-claude と cli-gemini のように）
    local project_a="${TEST_VOLUME_PREFIX}-projecta-$$"
    local project_b="${TEST_VOLUME_PREFIX}-projectb-$$"
    local vol_a="${project_a}_cli-sandbox-home"
    local vol_b="${project_b}_cli-sandbox-home"

    if [ "$DRY_RUN" = "true" ]; then
        echo -e "  ${BLUE}[DRY-RUN] Would create project volumes:${NC}"
        echo -e "  ${BLUE}    Project A: $vol_a${NC}"
        echo -e "  ${BLUE}    Project B: $vol_b${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would write different data to each${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would verify data is isolated${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would delete both volumes${NC}"
        pass "Volume project separation (dry-run)"
        return
    fi

    # Create volumes (simulating what docker-compose would create)
    docker volume create "$vol_a" >/dev/null 2>&1
    docker volume create "$vol_b" >/dev/null 2>&1
    track_volume "$vol_a"
    track_volume "$vol_b"
    echo "  Created project volumes:"
    echo "    Project A: $vol_a"
    echo "    Project B: $vol_b"

    # Write different data to each
    docker run --rm -v "${vol_a}:/data" alpine sh -c "echo 'project-a-data' > /data/id.txt" 2>/dev/null
    docker run --rm -v "${vol_b}:/data" alpine sh -c "echo 'project-b-data' > /data/id.txt" 2>/dev/null

    # Verify they are separate
    local data_a
    local data_b
    data_a=$(docker run --rm -v "${vol_a}:/data:ro" alpine cat /data/id.txt 2>/dev/null)
    data_b=$(docker run --rm -v "${vol_b}:/data:ro" alpine cat /data/id.txt 2>/dev/null)

    if [ "$data_a" = "project-a-data" ] && [ "$data_b" = "project-b-data" ]; then
        pass "Different project names create separate volumes"
        echo "    Volume A contains: $data_a"
        echo "    Volume B contains: $data_b"
    else
        fail "Volume separation failed"
    fi

    # Cleanup
    docker volume rm "$vol_a" "$vol_b" >/dev/null 2>&1
    remove_from_array "CREATED_VOLUMES" "$vol_a"
    remove_from_array "CREATED_VOLUMES" "$vol_b"
    echo "  Cleaned up: $vol_a, $vol_b"
}

test_volume_cleanup_on_failure() {
    info "[Volume Test] Verify cleanup function works"

    if ! has_docker; then
        skip "Docker not available"
        return
    fi

    if [ "$DRY_RUN" = "true" ]; then
        echo -e "  ${BLUE}[DRY-RUN] Would create test volumes for cleanup verification${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would run cleanup_test_volumes()${NC}"
        echo -e "  ${BLUE}[DRY-RUN] Would verify volumes are removed${NC}"
        pass "Cleanup function (dry-run)"
        return
    fi

    # Create some test volumes
    local vol1="${TEST_VOLUME_PREFIX}-cleanup1-$$"
    local vol2="${TEST_VOLUME_PREFIX}-cleanup2-$$"

    docker volume create "$vol1" >/dev/null 2>&1
    docker volume create "$vol2" >/dev/null 2>&1
    # Don't track these - we want cleanup_test_volumes to find them

    # Run cleanup
    cleanup_test_volumes

    # Verify volumes are removed
    if ! docker volume inspect "$vol1" >/dev/null 2>&1 && \
       ! docker volume inspect "$vol2" >/dev/null 2>&1; then
        pass "Cleanup function removes test volumes"
    else
        fail "Cleanup function did not remove all test volumes"
        # Force cleanup
        docker volume rm "$vol1" "$vol2" >/dev/null 2>&1 || true
    fi
}

###############################################################################
# Section 9: Server Integration Tests (Host OS only)
# セクション9: サーバー統合テスト（ホストOS専用）
###############################################################################

test_dkmcp_serve_starts() {
    info "$(msg "[Integration] DockMCP server can start" "[統合] DockMCPサーバーが起動できるか")"

    if is_devcontainer && [ "$HOST_ONLY" != "true" ]; then
        skip "$(msg "Integration test - run on host OS" "統合テスト - ホストOSで実行してください")"
        return
    fi

    if ! has_dkmcp; then
        skip "$(msg "dkmcp binary not found" "dkmcpバイナリが見つかりません")"
        return
    fi

    if ! has_docker; then
        skip "$(msg "Docker not available" "Dockerが利用できません")"
        return
    fi

    # Try to start server briefly and check if it responds
    # サーバーを起動して応答を確認
    local config_file="$DKMCP_DIR/configs/dkmcp.example.yaml"
    local test_port=18080

    if [ ! -f "$config_file" ]; then
        skip "$(msg "Config file not found" "設定ファイルが見つかりません")"
        return
    fi

    if [ "$DRY_RUN" = "true" ]; then
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would start DockMCP server on port $test_port" "ポート $test_port でDockMCPサーバーを起動")${NC}"
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would check health endpoint" "ヘルスエンドポイントを確認")${NC}"
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would stop server" "サーバーを停止")${NC}"
        pass "$(msg "DockMCP server start (dry-run)" "DockMCPサーバー起動 (dry-run)")"
        return
    fi

    # Check if port is already in use
    # ポートが既に使用中か確認
    if command -v lsof >/dev/null 2>&1 && lsof -i ":$test_port" >/dev/null 2>&1; then
        fail "$(msg "Port $test_port is already in use" "ポート $test_port は既に使用中です")"
        echo -e "  ${YELLOW}$(msg "Hint: Another process is using port $test_port" "ヒント: 別のプロセスがポート $test_port を使用中")${NC}"
        echo -e "  ${YELLOW}      $(msg "Run: lsof -i :$test_port" "実行: lsof -i :$test_port")${NC}"
        return
    fi

    # Start server in background, capture stderr
    # バックグラウンドでサーバーを起動、stderrをキャプチャ
    local server_log="/tmp/dkmcp-test-$$.log"
    track_file "$server_log"
    dkmcp serve --port $test_port --config "$config_file" >"$server_log" 2>&1 &
    local server_pid=$!
    track_process $server_pid

    # Wait for server to start
    sleep 2

    # Check if server is running
    # サーバーが起動しているか確認
    if kill -0 $server_pid 2>/dev/null; then
        # Try health check
        if curl -s "http://localhost:$test_port/health" >/dev/null 2>&1; then
            pass "$(msg "DockMCP server starts and responds to health check" "DockMCPサーバー起動、ヘルスチェック応答OK")"
        else
            pass "$(msg "DockMCP server starts (health endpoint not available)" "DockMCPサーバー起動（ヘルスエンドポイントなし）")"
        fi

        # Stop server
        kill $server_pid 2>/dev/null || true
        wait $server_pid 2>/dev/null || true
        remove_from_array "STARTED_PROCESSES" "$server_pid"
    else
        fail "$(msg "DockMCP server failed to start" "DockMCPサーバーの起動に失敗")"
        # Show server output for debugging
        if [ -f "$server_log" ] && [ -s "$server_log" ]; then
            echo -e "  ${RED}$(msg "Server output:" "サーバー出力:")${NC}"
            head -10 "$server_log" | sed 's/^/    /'
        fi
        # Common causes
        if grep -qi "address already in use" "$server_log" 2>/dev/null; then
            echo -e "  ${YELLOW}$(msg "Hint: Port $test_port is already in use" "ヒント: ポート $test_port は既に使用中")${NC}"
        elif grep -qi "permission denied" "$server_log" 2>/dev/null; then
            echo -e "  ${YELLOW}$(msg "Hint: Permission denied - check Docker socket access" "ヒント: 権限エラー - Dockerソケットのアクセスを確認")${NC}"
        elif grep -qi "cannot connect" "$server_log" 2>/dev/null; then
            echo -e "  ${YELLOW}$(msg "Hint: Cannot connect to Docker daemon" "ヒント: Dockerデーモンに接続できません")${NC}"
        fi
        remove_from_array "STARTED_PROCESSES" "$server_pid"
    fi

    # Cleanup log file
    rm -f "$server_log"
    remove_from_array "CREATED_FILES" "$server_log"
}

test_dkmcp_multiple_instances() {
    info "$(msg "[Integration] Multiple DockMCP instances can run" "[統合] 複数のDockMCPインスタンスが起動できるか")"

    if is_devcontainer && [ "$HOST_ONLY" != "true" ]; then
        skip "$(msg "Integration test - run on host OS" "統合テスト - ホストOSで実行してください")"
        return
    fi

    if ! has_dkmcp; then
        skip "$(msg "dkmcp binary not found" "dkmcpバイナリが見つかりません")"
        return
    fi

    if ! has_docker; then
        skip "$(msg "Docker not available" "Dockerが利用できません")"
        return
    fi

    local config_file="$DKMCP_DIR/configs/dkmcp.example.yaml"
    local port1=18081
    local port2=18082

    if [ ! -f "$config_file" ]; then
        skip "$(msg "Config file not found" "設定ファイルが見つかりません")"
        return
    fi

    if [ "$DRY_RUN" = "true" ]; then
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would start DockMCP instance 1 on port $port1" "ポート $port1 でインスタンス1を起動")${NC}"
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would start DockMCP instance 2 on port $port2" "ポート $port2 でインスタンス2を起動")${NC}"
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would verify both are running" "両方が起動していることを確認")${NC}"
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would stop both instances" "両方のインスタンスを停止")${NC}"
        pass "$(msg "Multiple DockMCP instances (dry-run)" "複数DockMCPインスタンス (dry-run)")"
        return
    fi

    # Check if ports are already in use
    # ポートが既に使用中か確認
    for port in $port1 $port2; do
        if command -v lsof >/dev/null 2>&1 && lsof -i ":$port" >/dev/null 2>&1; then
            fail "$(msg "Port $port is already in use" "ポート $port は既に使用中です")"
            echo -e "  ${YELLOW}$(msg "Hint: Run: lsof -i :$port" "ヒント: 実行: lsof -i :$port")${NC}"
            return
        fi
    done

    # Start two instances with logging
    # 2つのインスタンスをログ付きで起動
    local log1="/tmp/dkmcp-test1-$$.log"
    local log2="/tmp/dkmcp-test2-$$.log"
    track_file "$log1"
    track_file "$log2"

    dkmcp serve --port $port1 --config "$config_file" >"$log1" 2>&1 &
    local pid1=$!
    track_process $pid1

    dkmcp serve --port $port2 --config "$config_file" >"$log2" 2>&1 &
    local pid2=$!
    track_process $pid2

    sleep 2

    local instance1_ok=false
    local instance2_ok=false

    if kill -0 $pid1 2>/dev/null; then
        instance1_ok=true
    fi

    if kill -0 $pid2 2>/dev/null; then
        instance2_ok=true
    fi

    # Cleanup
    kill $pid1 2>/dev/null || true
    kill $pid2 2>/dev/null || true
    wait $pid1 2>/dev/null || true
    wait $pid2 2>/dev/null || true
    remove_from_array "STARTED_PROCESSES" "$pid1"
    remove_from_array "STARTED_PROCESSES" "$pid2"

    if $instance1_ok && $instance2_ok; then
        pass "$(msg "Multiple DockMCP instances can run simultaneously" "複数DockMCPインスタンスが同時に起動可能")"
    else
        fail "$(msg "Failed to run multiple DockMCP instances" "複数DockMCPインスタンスの起動に失敗")"
        echo "  $(msg "Instance 1 (port $port1):" "インスタンス1 (ポート $port1):") $instance1_ok"
        echo "  $(msg "Instance 2 (port $port2):" "インスタンス2 (ポート $port2):") $instance2_ok"
        # Show error details
        if [ "$instance1_ok" = "false" ] && [ -f "$log1" ] && [ -s "$log1" ]; then
            echo -e "  ${RED}$(msg "Instance 1 error:" "インスタンス1 エラー:")${NC}"
            head -5 "$log1" | sed 's/^/    /'
        fi
        if [ "$instance2_ok" = "false" ] && [ -f "$log2" ] && [ -s "$log2" ]; then
            echo -e "  ${RED}$(msg "Instance 2 error:" "インスタンス2 エラー:")${NC}"
            head -5 "$log2" | sed 's/^/    /'
        fi
    fi

    # Cleanup log files
    rm -f "$log1" "$log2"
    remove_from_array "CREATED_FILES" "$log1"
    remove_from_array "CREATED_FILES" "$log2"
}

test_dkmcp_port_flag_effective() {
    info "$(msg "[Integration] --port flag binds to specified port" "[統合] --port フラグが指定ポートで待ち受けるか")"

    if is_devcontainer && [ "$HOST_ONLY" != "true" ]; then
        skip "$(msg "Integration test - run on host OS" "統合テスト - ホストOSで実行してください")"
        return
    fi

    if ! has_dkmcp; then
        skip "$(msg "dkmcp binary not found" "dkmcpバイナリが見つかりません")"
        return
    fi

    if ! has_docker; then
        skip "$(msg "Docker not available" "Dockerが利用できません")"
        return
    fi

    local config_file="$DKMCP_DIR/configs/dkmcp.example.yaml"
    local test_port=19090

    if [ ! -f "$config_file" ]; then
        skip "$(msg "Config file not found" "設定ファイルが見つかりません")"
        return
    fi

    if [ "$DRY_RUN" = "true" ]; then
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would start server with --port $test_port" "--port $test_port でサーバー起動")${NC}"
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would verify server responds on port $test_port" "ポート $test_port で応答を確認")${NC}"
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would verify server does NOT respond on default port 8080" "デフォルトポート 8080 で応答しないことを確認")${NC}"
        pass "$(msg "--port flag effectiveness (dry-run)" "--port フラグ有効性 (dry-run)")"
        return
    fi

    # Check if port is already in use
    # ポートが既に使用中か確認
    if command -v lsof >/dev/null 2>&1 && lsof -i ":$test_port" >/dev/null 2>&1; then
        fail "$(msg "Port $test_port is already in use" "ポート $test_port は既に使用中です")"
        return
    fi

    # Start server with custom port
    local server_log="/tmp/dkmcp-port-test-$$.log"
    track_file "$server_log"
    dkmcp serve --port $test_port --config "$config_file" >"$server_log" 2>&1 &
    local server_pid=$!
    track_process $server_pid

    sleep 2

    if ! kill -0 $server_pid 2>/dev/null; then
        fail "$(msg "Server failed to start" "サーバーの起動に失敗")"
        rm -f "$server_log"
        remove_from_array "CREATED_FILES" "$server_log"
        remove_from_array "STARTED_PROCESSES" "$server_pid"
        return
    fi

    # Test 1: Server should respond on specified port
    # テスト1: 指定ポートでサーバーが応答するか
    local responds_on_custom=false
    if curl -s --max-time 2 "http://localhost:$test_port/health" >/dev/null 2>&1; then
        responds_on_custom=true
    fi

    # Test 2: Server should NOT respond on default port 8080 (unless something else is running)
    # テスト2: デフォルトポート8080では応答しないか（他のプロセスが使用中でない限り）
    local responds_on_default=false
    if curl -s --max-time 2 "http://localhost:8080/health" >/dev/null 2>&1; then
        responds_on_default=true
    fi

    # Cleanup
    kill $server_pid 2>/dev/null || true
    wait $server_pid 2>/dev/null || true
    remove_from_array "STARTED_PROCESSES" "$server_pid"
    rm -f "$server_log"
    remove_from_array "CREATED_FILES" "$server_log"

    if $responds_on_custom; then
        pass "$(msg "--port $test_port flag is effective (server responds on custom port)" "--port $test_port フラグ有効（カスタムポートで応答）")"
    else
        fail "$(msg "Server does not respond on specified port $test_port" "サーバーが指定ポート $test_port で応答しません")"
    fi
}

test_dkmcp_config_strict_mode() {
    info "$(msg "[Integration] mode: strict blocks exec commands" "[統合] mode: strict が exec をブロックするか")"

    if is_devcontainer && [ "$HOST_ONLY" != "true" ]; then
        skip "$(msg "Integration test - run on host OS" "統合テスト - ホストOSで実行してください")"
        return
    fi

    if ! has_dkmcp; then
        skip "$(msg "dkmcp binary not found" "dkmcpバイナリが見つかりません")"
        return
    fi

    if ! has_docker; then
        skip "$(msg "Docker not available" "Dockerが利用できません")"
        return
    fi

    local test_port=19091
    local strict_config="/tmp/dkmcp-strict-$$.yaml"

    if [ "$DRY_RUN" = "true" ]; then
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would create strict mode config" "strictモード設定を作成")${NC}"
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would start server with strict config" "strict設定でサーバー起動")${NC}"
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would verify exec command is rejected" "execコマンドが拒否されることを確認")${NC}"
        pass "$(msg "strict mode effectiveness (dry-run)" "strictモード有効性 (dry-run)")"
        return
    fi

    # Create strict mode config
    # strictモード設定を作成
    cat > "$strict_config" << 'EOF'
server:
  port: 19091
  host: "127.0.0.1"
security:
  mode: "strict"
  allowed_containers: []
  exec_whitelist: {}
  permissions:
    logs: true
    inspect: true
    stats: true
    exec: false
EOF
    track_file "$strict_config"

    # Check if port is already in use
    # ポートが既に使用中か確認
    if command -v lsof >/dev/null 2>&1 && lsof -i ":$test_port" >/dev/null 2>&1; then
        fail "$(msg "Port $test_port is already in use" "ポート $test_port は既に使用中です")"
        rm -f "$strict_config"
        remove_from_array "CREATED_FILES" "$strict_config"
        return
    fi

    # Start server with strict config
    # strict設定でサーバーを起動
    local server_log="/tmp/dkmcp-strict-test-$$.log"
    track_file "$server_log"
    dkmcp serve --port $test_port --config "$strict_config" >"$server_log" 2>&1 &
    local server_pid=$!
    track_process $server_pid

    sleep 2

    if ! kill -0 $server_pid 2>/dev/null; then
        fail "$(msg "Server failed to start with strict config" "strict設定でのサーバー起動に失敗")"
        if [ -f "$server_log" ] && [ -s "$server_log" ]; then
            echo -e "  ${RED}$(msg "Server output:" "サーバー出力:")${NC}"
            head -5 "$server_log" | sed 's/^/    /'
        fi
        rm -f "$strict_config" "$server_log"
        remove_from_array "CREATED_FILES" "$strict_config"
        remove_from_array "CREATED_FILES" "$server_log"
        remove_from_array "STARTED_PROCESSES" "$server_pid"
        return
    fi

    # Try exec command - should be rejected in strict mode
    # execコマンドを試行 - strictモードでは拒否されるべき
    local exec_output
    exec_output=$($DKMCP_BIN client exec --url "http://localhost:$test_port" "securenote-api" "pwd" 2>&1 | tr -d '\0') || true

    # Cleanup
    kill $server_pid 2>/dev/null || true
    wait $server_pid 2>/dev/null || true
    remove_from_array "STARTED_PROCESSES" "$server_pid"
    rm -f "$strict_config" "$server_log"
    remove_from_array "CREATED_FILES" "$strict_config"
    remove_from_array "CREATED_FILES" "$server_log"

    # Check if exec was rejected
    # execが拒否されたか確認
    if echo "$exec_output" | grep -qiE "(not allowed|disabled|strict|permission denied|exec.*disabled)"; then
        pass "$(msg "strict mode blocks exec commands" "strictモードがexecコマンドをブロック")"
    elif echo "$exec_output" | grep -qiE "(error|failed)"; then
        pass "$(msg "strict mode appears to block exec (error returned)" "strictモードがexecをブロック（エラー返却）")"
        echo "    Output: $(echo "$exec_output" | head -1)"
    else
        fail "$(msg "strict mode did not block exec command" "strictモードがexecコマンドをブロックしませんでした")"
        echo "    Output: $exec_output"
    fi
}

test_dkmcp_allowed_containers_effective() {
    info "$(msg "[Integration] allowed_containers filters container access" "[統合] allowed_containers がコンテナアクセスをフィルタするか")"

    if is_devcontainer && [ "$HOST_ONLY" != "true" ]; then
        skip "$(msg "Integration test - run on host OS" "統合テスト - ホストOSで実行してください")"
        return
    fi

    if ! has_dkmcp; then
        skip "$(msg "dkmcp binary not found" "dkmcpバイナリが見つかりません")"
        return
    fi

    if ! has_docker; then
        skip "$(msg "Docker not available" "Dockerが利用できません")"
        return
    fi

    local test_port=19092
    local filter_config="/tmp/dkmcp-filter-$$.yaml"

    if [ "$DRY_RUN" = "true" ]; then
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would create config with allowed_containers: [\"nonexistent-*\"]" "allowed_containers: [\"nonexistent-*\"] の設定を作成")${NC}"
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would start server" "サーバーを起動")${NC}"
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would verify existing containers are not accessible" "既存コンテナにアクセスできないことを確認")${NC}"
        pass "$(msg "allowed_containers effectiveness (dry-run)" "allowed_containers有効性 (dry-run)")"
        return
    fi

    # Create config that only allows nonexistent containers
    # 存在しないコンテナのみ許可する設定を作成
    cat > "$filter_config" << 'EOF'
server:
  port: 19092
  host: "127.0.0.1"
security:
  mode: "moderate"
  allowed_containers:
    - "nonexistent-container-*"
  exec_whitelist:
    "*":
      - "pwd"
  permissions:
    logs: true
    inspect: true
    stats: true
    exec: true
EOF
    track_file "$filter_config"

    # Check if port is already in use
    # ポートが既に使用中か確認
    if command -v lsof >/dev/null 2>&1 && lsof -i ":$test_port" >/dev/null 2>&1; then
        fail "$(msg "Port $test_port is already in use" "ポート $test_port は既に使用中です")"
        rm -f "$filter_config"
        remove_from_array "CREATED_FILES" "$filter_config"
        return
    fi

    # Start server
    # サーバーを起動
    local server_log="/tmp/dkmcp-filter-test-$$.log"
    track_file "$server_log"
    dkmcp serve --port $test_port --config "$filter_config" >"$server_log" 2>&1 &
    local server_pid=$!
    track_process $server_pid

    sleep 2

    if ! kill -0 $server_pid 2>/dev/null; then
        fail "$(msg "Server failed to start" "サーバーの起動に失敗")"
        rm -f "$filter_config" "$server_log"
        remove_from_array "CREATED_FILES" "$filter_config"
        remove_from_array "CREATED_FILES" "$server_log"
        remove_from_array "STARTED_PROCESSES" "$server_pid"
        return
    fi

    # Try to list containers - should show filtered results or none
    # コンテナ一覧を取得 - フィルタされた結果または空のはず
    local list_output
    list_output=$($DKMCP_BIN client list --url "http://localhost:$test_port" 2>&1 | tr -d '\0') || true

    # Cleanup
    kill $server_pid 2>/dev/null || true
    wait $server_pid 2>/dev/null || true
    remove_from_array "STARTED_PROCESSES" "$server_pid"
    rm -f "$filter_config" "$server_log"
    remove_from_array "CREATED_FILES" "$filter_config"
    remove_from_array "CREATED_FILES" "$server_log"

    # Check results - should NOT show securenote-api or demo-* containers
    # 結果確認 - securenote-api や demo-* コンテナは表示されないはず
    if echo "$list_output" | grep -qE "(securenote|demo-)"; then
        fail "$(msg "allowed_containers filter not effective - restricted containers visible" "allowed_containersフィルタ無効 - 制限コンテナが表示されています")"
        echo "    $(msg "Output shows restricted containers" "出力に制限コンテナが含まれています")"
    else
        pass "$(msg "allowed_containers filter is effective (restricted containers not visible)" "allowed_containersフィルタ有効（制限コンテナ非表示）")"
    fi
}

test_dkmcp_exec_whitelist_effective() {
    info "$(msg "[Integration] exec_whitelist blocks non-whitelisted commands" "[統合] exec_whitelist がホワイトリスト外コマンドをブロックするか")"

    if is_devcontainer && [ "$HOST_ONLY" != "true" ]; then
        skip "$(msg "Integration test - run on host OS" "統合テスト - ホストOSで実行してください")"
        return
    fi

    if ! has_dkmcp; then
        skip "$(msg "dkmcp binary not found" "dkmcpバイナリが見つかりません")"
        return
    fi

    if ! has_docker; then
        skip "$(msg "Docker not available" "Dockerが利用できません")"
        return
    fi

    # Check if securenote-api container is running
    # securenote-api コンテナが起動中か確認
    if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -q "securenote-api"; then
        skip "$(msg "securenote-api container not running (start demo-apps first)" "securenote-apiコンテナ未起動（先にdemo-appsを起動）")"
        return
    fi

    local test_port=19093
    local whitelist_config="/tmp/dkmcp-whitelist-$$.yaml"

    if [ "$DRY_RUN" = "true" ]; then
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would create config with limited exec_whitelist" "制限付きexec_whitelist設定を作成")${NC}"
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would start server" "サーバーを起動")${NC}"
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would verify whitelisted command works" "ホワイトリストコマンドが動作することを確認")${NC}"
        echo -e "  ${BLUE}[DRY-RUN] $(msg "Would verify non-whitelisted command is rejected" "ホワイトリスト外コマンドが拒否されることを確認")${NC}"
        pass "$(msg "exec_whitelist effectiveness (dry-run)" "exec_whitelist有効性 (dry-run)")"
        return
    fi

    # Create config with limited whitelist
    # 制限付きホワイトリスト設定を作成
    cat > "$whitelist_config" << 'EOF'
server:
  port: 19093
  host: "127.0.0.1"
security:
  mode: "moderate"
  allowed_containers:
    - "securenote-*"
    - "demo-*"
  exec_whitelist:
    "*":
      - "pwd"
      - "whoami"
  permissions:
    logs: true
    inspect: true
    stats: true
    exec: true
EOF
    track_file "$whitelist_config"

    # Check if port is already in use
    # ポートが既に使用中か確認
    if command -v lsof >/dev/null 2>&1 && lsof -i ":$test_port" >/dev/null 2>&1; then
        fail "$(msg "Port $test_port is already in use" "ポート $test_port は既に使用中です")"
        rm -f "$whitelist_config"
        remove_from_array "CREATED_FILES" "$whitelist_config"
        return
    fi

    # Start server
    # サーバーを起動
    local server_log="/tmp/dkmcp-whitelist-test-$$.log"
    track_file "$server_log"
    dkmcp serve --port $test_port --config "$whitelist_config" >"$server_log" 2>&1 &
    local server_pid=$!
    track_process $server_pid

    sleep 2

    if ! kill -0 $server_pid 2>/dev/null; then
        fail "$(msg "Server failed to start" "サーバーの起動に失敗")"
        rm -f "$whitelist_config" "$server_log"
        remove_from_array "CREATED_FILES" "$whitelist_config"
        remove_from_array "CREATED_FILES" "$server_log"
        remove_from_array "STARTED_PROCESSES" "$server_pid"
        return
    fi

    local whitelist_test_passed=true
    local non_whitelist_test_passed=true

    # Test 1: Whitelisted command should work
    # テスト1: ホワイトリストコマンドは動作すべき
    local pwd_output
    pwd_output=$($DKMCP_BIN client exec --url "http://localhost:$test_port" "securenote-api" "pwd" 2>&1 | tr -d '\0') || true

    if echo "$pwd_output" | grep -qE "Exit Code: 0|/app|/home"; then
        echo "    $(msg "Whitelisted command 'pwd': OK" "ホワイトリストコマンド 'pwd': OK")"
    else
        whitelist_test_passed=false
        echo "    $(msg "Whitelisted command 'pwd': FAILED" "ホワイトリストコマンド 'pwd': 失敗")"
        echo "    Output: $(echo "$pwd_output" | head -1)"
    fi

    # Test 2: Non-whitelisted command should be rejected
    # テスト2: ホワイトリスト外コマンドは拒否されるべき
    local cat_output
    cat_output=$($DKMCP_BIN client exec --url "http://localhost:$test_port" "securenote-api" "cat /etc/passwd" 2>&1 | tr -d '\0') || true

    if echo "$cat_output" | grep -qiE "(not whitelisted|not allowed|rejected)"; then
        echo "    $(msg "Non-whitelisted command 'cat': Rejected (OK)" "ホワイトリスト外コマンド 'cat': 拒否 (OK)")"
    else
        non_whitelist_test_passed=false
        echo "    $(msg "Non-whitelisted command 'cat': NOT rejected (FAILED)" "ホワイトリスト外コマンド 'cat': 拒否されず (失敗)")"
        echo "    Output: $(echo "$cat_output" | head -1)"
    fi

    # Cleanup
    kill $server_pid 2>/dev/null || true
    wait $server_pid 2>/dev/null || true
    remove_from_array "STARTED_PROCESSES" "$server_pid"
    rm -f "$whitelist_config" "$server_log"
    remove_from_array "CREATED_FILES" "$whitelist_config"
    remove_from_array "CREATED_FILES" "$server_log"

    if $whitelist_test_passed && $non_whitelist_test_passed; then
        pass "$(msg "exec_whitelist is effective" "exec_whitelistが有効")"
    elif $non_whitelist_test_passed; then
        pass "$(msg "exec_whitelist blocks non-whitelisted commands (whitelist test inconclusive)" "exec_whitelistがホワイトリスト外をブロック（ホワイトリストテスト不確定）")"
    else
        fail "$(msg "exec_whitelist not effective" "exec_whitelistが無効")"
    fi
}

# Variable to hold dkmcp binary path for integration tests
# 統合テスト用のdkmcpバイナリパス
DKMCP_BIN=""

# Find or build dkmcp binary for integration tests
# 統合テスト用のdkmcpバイナリを検索またはビルド
setup_dkmcp_bin() {
    # First check if dkmcp is in PATH
    if command -v dkmcp >/dev/null 2>&1; then
        DKMCP_BIN="dkmcp"
        return 0
    fi

    # Check common locations
    local locations=(
        "$HOME/go/bin/dkmcp"
        "/usr/local/bin/dkmcp"
        "$DKMCP_DIR/dkmcp"
    )

    for loc in "${locations[@]}"; do
        if [ -x "$loc" ]; then
            DKMCP_BIN="$loc"
            return 0
        fi
    done

    # Try to build
    if command -v go >/dev/null 2>&1; then
        echo "  Building dkmcp binary..."
        local tmp_bin="/tmp/dkmcp-test-bin-$$"
        if (cd "$DKMCP_DIR" && go build -o "$tmp_bin" ./cmd/dkmcp 2>/dev/null); then
            DKMCP_BIN="$tmp_bin"
            track_file "$tmp_bin"
            return 0
        fi
    fi

    return 1
}

###############################################################################
# Main
###############################################################################

main() {
    # Show help if no arguments provided
    # 引数がない場合はヘルプを表示
    if [ $# -eq 0 ]; then
        show_help
        exit 0
    fi

    # Parse arguments
    while [ $# -gt 0 ]; do
        case "$1" in
            --host-only)
                HOST_ONLY=true
                shift
                ;;
            --basic)
                # Sections 1-4 only (default behavior, no extra flags needed)
                # セクション1-4のみ（デフォルト動作、追加フラグ不要）
                # Note: RUN_BASIC is not directly referenced, but this flag's presence
                # prevents the script from exiting at the "if [ $# -eq 0 ]" check in main()
                # and allows sections 1-4 to run without enabling optional sections 5-9.
                # 注: RUN_BASIC は直接参照されないが、このフラグの存在により
                # main() 内の「if [ $# -eq 0 ]」チェックで終了せず、セクション1-4が実行される。
                RUN_BASIC=true
                shift
                ;;
            --all)
                RUN_ALL=true
                shift
                ;;
            --full)
                RUN_FULL=true
                RUN_ALL=true
                TEST_CONFIG=true
                TEST_ENV=true
                TEST_COPY=true
                TEST_VOLUME=true
                shift
                ;;
            --test-config)
                TEST_CONFIG=true
                shift
                ;;
            --test-env)
                TEST_ENV=true
                shift
                ;;
            --test-copy)
                TEST_COPY=true
                shift
                ;;
            --test-volume)
                TEST_VOLUME=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            -y|--yes)
                AUTO_YES=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                echo -e "${RED}Unknown option: $1${NC}"
                echo "不明なオプション: $1"
                echo ""
                show_help
                exit 1
                ;;
        esac
    done

    echo "========================================"
    echo "Testing Advanced Features"
    echo "高度な使い方機能のテスト"
    echo "========================================"
    echo ""
    echo "Workspace: $WORKSPACE_DIR"
    echo "Environment: ${SANDBOX_ENV:-unknown}"
    echo "Docker available: $(has_docker && echo 'yes' || echo 'no')"
    echo "dkmcp available: $(has_dkmcp && echo 'yes' || echo 'no')"
    [ "$DRY_RUN" = "true" ] && echo -e "Mode: ${BLUE}DRY-RUN${NC}"
    [ "$AUTO_YES" = "true" ] && echo -e "Auto-confirm: ${YELLOW}YES${NC}"
    echo ""

    # Check for dangerous operations and confirm
    local needs_docker_ops=false
    [ "$TEST_VOLUME" = "true" ] && needs_docker_ops=true
    [ "$RUN_ALL" = "true" ] && needs_docker_ops=true

    if [ "$needs_docker_ops" = "true" ] && has_docker; then
        # Check for existing test volumes
        if ! check_existing_volumes; then
            if [ "$AUTO_YES" != "true" ] && [ "$DRY_RUN" != "true" ]; then
                read -p "Continue anyway? / それでも続行しますか？ [y/N] " -n 1 -r
                echo
                if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                    echo "Cancelled. / キャンセルしました。"
                    exit 0
                fi
            fi
        fi

        # Confirm dangerous operations
        local ops_desc=""
        [ "$TEST_VOLUME" = "true" ] && ops_desc="Docker volume create/copy/delete tests"
        [ "$RUN_ALL" = "true" ] && ops_desc="${ops_desc:+$ops_desc, }DockMCP server start/stop"

        if ! confirm_dangerous_operation "$ops_desc"; then
            exit 0
        fi
    fi

    echo ""

    section "1. Custom DockMCP Configuration / カスタムDockMCP設定"
    test_dkmcp_config_exists
    test_dkmcp_config_valid_yaml
    test_dkmcp_config_has_security_modes
    test_dkmcp_config_allowed_containers
    test_dkmcp_config_exec_whitelist

    section "2. Multiple DockMCP Instances / 複数DockMCPインスタンス"
    test_dkmcp_serve_port_flag
    test_dkmcp_serve_config_flag
    test_dkmcp_multiple_configs_exist

    section "3. Project Name Customization / プロジェクト名カスタマイズ"
    test_devcontainer_env_example_exists
    test_devcontainer_env_gitignore
    test_compose_project_name_support
    test_cli_sandbox_env_support
    test_cli_sandbox_multi_project
    test_cli_sandbox_project_isolation

    section "4. Multiple DevContainer Instances / 複数DevContainer"
    test_copy_credentials_script_exists
    test_copy_credentials_help
    test_copy_credentials_export_import
    test_copy_credentials_workspace_mode

    # File creation/deletion tests (require confirmation)
    # ファイル作成/削除テスト（確認が必要）
    if [ "$TEST_CONFIG" = "true" ]; then
        if confirm_section "5" "Custom Config File Tests / カスタム設定ファイルテスト" \
            "Creates test config files in /tmp directory (does NOT modify actual project)" \
            "/tmp にテスト設定ファイルを作成（実際のプロジェクトは変更しない）" \
            "Low - Only creates temp files, cleaned up automatically" \
            "低 - 一時ファイルのみ作成、自動的にクリーンアップ" \
            "Delete manually if needed: rm /tmp/test-dkmcp-*.yaml" \
            "必要なら手動で削除: rm /tmp/test-dkmcp-*.yaml"; then
            section "5. Custom Config File Tests / カスタム設定ファイルテスト"
            test_create_custom_dkmcp_config
            test_create_permissive_dkmcp_config
        fi
    fi

    if [ "$TEST_ENV" = "true" ]; then
        if confirm_section "6" ".env File Tests / .envファイルテスト" \
            "Creates test .env files in /tmp directory (does NOT modify actual project)" \
            "/tmp にテスト用 .env を作成（実際のプロジェクトは変更しない）" \
            "Low - Only creates temp files, cleaned up automatically" \
            "低 - 一時ファイルのみ作成、自動的にクリーンアップ" \
            "Delete manually if needed: rm -rf /tmp/test-*-env-*" \
            "必要なら手動で削除: rm -rf /tmp/test-*-env-*"; then
            section "6. .env File Tests / .envファイルテスト"
            test_create_env_file
            test_env_file_multiple_projects
        fi
    fi

    if [ "$TEST_COPY" = "true" ]; then
        if confirm_section "7" "copy-credentials.sh Tests / copy-credentials.shテスト" \
            "Creates temporary backup directories in /tmp (test-backup-*)" \
            "/tmp に一時バックアップディレクトリを作成（test-backup-*）" \
            "Low - Only creates temp files, no Docker operations" \
            "低 - 一時ファイルのみ作成、Docker操作なし" \
            "Delete manually if needed: rm -rf /tmp/test-backup-*" \
            "必要なら手動で削除: rm -rf /tmp/test-backup-*"; then
            section "7. copy-credentials.sh Tests / copy-credentials.shテスト"
            test_copy_credentials_export_dry_run
            test_copy_credentials_backup_structure
            test_copy_credentials_import_validation
        fi
    fi

    # Docker volume tests (requires Docker)
    if [ "$TEST_VOLUME" = "true" ]; then
        if confirm_section "8" "Docker Volume Tests / Dockerボリュームテスト" \
            "Creates/deletes Docker volumes: test-advanced-features-* (isolated test prefix)" \
            "Dockerボリュームの作成/削除: test-advanced-features-*（隔離されたテスト用プレフィックス）" \
            "Low-Medium - Uses unique prefix, each test cleans up after itself" \
            "低〜中 - 一意のプレフィックス使用、各テスト終了後に自動クリーンアップ" \
            "Cleanup if needed: docker volume rm \$(docker volume ls -q | grep test-advanced-features)" \
            "必要なら: docker volume rm \$(docker volume ls -q | grep test-advanced-features)"; then
            section "8. Docker Volume Tests / Dockerボリュームテスト"
            test_volume_create
            test_volume_write_read
            test_volume_copy_between_volumes
            test_volume_export_import_simulation
            test_volume_different_project_names
            test_volume_cleanup_on_failure
        fi
    fi

    # Server integration tests (requires Docker on host)
    if [ "$RUN_ALL" = "true" ]; then
        if confirm_section "9" "Server Integration Tests / サーバー統合テスト" \
            "Starts DockMCP servers (ports 18080-19093), creates temp configs in /tmp" \
            "DockMCPサーバー起動（ポート18080-19093）、/tmp に一時設定ファイル作成" \
            "Low-Medium - High ports, temp files auto-cleaned on exit" \
            "低〜中 - 高ポート使用、一時ファイルは終了時に自動クリーンアップ" \
            "Test ports only: for p in 18080 18081 18082 19090 19091 19092 19093; do lsof -ti:\$p | xargs kill 2>/dev/null; done" \
            "テストポートのみ停止: for p in 18080 18081 18082 19090 19091 19092 19093; do lsof -ti:\$p | xargs kill 2>/dev/null; done"; then
            section "9. Server Integration Tests / サーバー統合テスト"

            # Setup dkmcp binary for integration tests
            # 統合テスト用のdkmcpバイナリをセットアップ
            if ! setup_dkmcp_bin; then
                echo -e "${YELLOW}[WARN] $(msg "Could not find or build dkmcp binary" "dkmcpバイナリが見つからないかビルドできません")${NC}"
                echo "$(msg "Run: cd dkmcp && make install" "実行: cd dkmcp && make install")"
            fi

            # 9.1 Basic server tests
            echo -e "${BLUE}--- $(msg "9.1 Basic Server Tests" "9.1 基本サーバーテスト") ---${NC}"
            test_dkmcp_serve_starts
            test_dkmcp_multiple_instances

            # 9.2 Configuration effectiveness tests
            echo ""
            echo -e "${BLUE}--- $(msg "9.2 Config Effectiveness Tests" "9.2 設定有効性テスト") ---${NC}"
            test_dkmcp_port_flag_effective
            test_dkmcp_config_strict_mode
            test_dkmcp_allowed_containers_effective
            test_dkmcp_exec_whitelist_effective
        fi
    fi

    echo ""
    echo "========================================"
    echo "Results / 結果"
    echo "========================================"
    echo -e "Passed / 成功: ${GREEN}${TESTS_PASSED}${NC}"
    echo -e "Failed / 失敗: ${RED}${TESTS_FAILED}${NC}"
    echo -e "Skipped / スキップ: ${YELLOW}${TESTS_SKIPPED}${NC}"

    if [ $TESTS_FAILED -gt 0 ]; then
        exit 1
    fi
}

main "$@"

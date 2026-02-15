#!/bin/bash
# setup-dkmcp.sh
# Register DockMCP as an MCP server for AI tools (Claude Code, Gemini CLI)
#
# Detects available AI tools and registers DockMCP as an SSE MCP server.
# Also checks registration status and connectivity. Designed for AI-driven setup:
# AI can run --check to detect missing registration, then offer to register.
#
# Usage:
#   .sandbox/scripts/setup-dkmcp.sh [options]
#
# Options:
#   --check       Silent check (exit code: 0=connected, 1=not registered, 2=registered but offline)
#   --status      Human-readable status report
#   --url <url>   Custom DockMCP URL (default: http://host.docker.internal:8080/sse)
#   --unregister  Remove DockMCP from all detected AI tools
#   --help, -h    Show this help
#
# Examples:
#   .sandbox/scripts/setup-dkmcp.sh              # Register if needed + verify connectivity
#   .sandbox/scripts/setup-dkmcp.sh --check      # Silent check (for AI/startup detection)
#   .sandbox/scripts/setup-dkmcp.sh --status     # Show detailed status
#   .sandbox/scripts/setup-dkmcp.sh --unregister # Remove DockMCP registration
# ---
# DockMCP を AI ツール（Claude Code, Gemini CLI）に MCP サーバーとして登録
#
# 利用可能な AI ツールを検出し、DockMCP を SSE MCP サーバーとして登録します。
# 登録状態と接続性のチェックも可能で、AI による自動セットアップに活用できます。
# AI が --check で未登録を検出し、「登録しましょうか？」と提案する想定です。
#
# 使用法:
#   .sandbox/scripts/setup-dkmcp.sh [options]
#
# オプション:
#   --check       サイレントチェック（終了コード: 0=接続済, 1=未登録, 2=登録済だがオフライン）
#   --status      人向けのステータスレポート
#   --url <url>   カスタム DockMCP URL（デフォルト: http://host.docker.internal:8080/sse）
#   --unregister  全 AI ツールから DockMCP を削除
#   --help, -h    ヘルプ表示

set -euo pipefail

# ─── Colors & helpers / カラー出力・ヘルパー関数 ────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

info()  { echo -e "${CYAN}ℹ️  $1${NC}"; }
ok()    { echo -e "${GREEN}✅ $1${NC}"; }
warn()  { echo -e "${YELLOW}⚠️  $1${NC}"; }
err()   { echo -e "${RED}❌ $1${NC}" >&2; }
die()   { err "$1"; exit 1; }

# ─── Language detection / 言語検出 ─────────────────────────────

if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
    MSG_TITLE="🔗 DockMCP セットアップ"
    MSG_REGISTERED="登録済み"
    MSG_NOT_REGISTERED="未登録"
    MSG_CLI_NOT_FOUND="CLI 未インストール"
    MSG_ALREADY_REGISTERED="登録済み"
    MSG_REGISTERING="登録中..."
    MSG_REGISTERED_OK="登録完了"
    MSG_REGISTERED_FALLBACK="登録完了（.mcp.json 直接編集）"
    MSG_REGISTER_FAILED="登録に失敗しました"
    MSG_CONNECTIVITY="接続状態"
    MSG_SERVER_RUNNING="DockMCP サーバー稼働中"
    MSG_SERVER_NOT_RUNNING="DockMCP サーバーに接続できません"
    MSG_START_HINT="ホスト OS で DockMCP を起動してください:"
    MSG_NEXT_STEPS="次のステップ"
    MSG_CLAUDE_RECONNECT="Claude Code: /mcp → Reconnect を実行"
    MSG_GEMINI_RESTART="Gemini CLI: セッションを再起動"
    MSG_NO_AI_TOOLS="AI ツールが見つかりません（claude / gemini どちらも未インストール）"
    MSG_UNREGISTER_TITLE="🔗 DockMCP 登録解除"
    MSG_UNREGISTERED="削除済み"
    MSG_NOT_FOUND="未登録のためスキップ"
    MSG_HELP_USAGE="使用法"
    MSG_HELP_OPTIONS="オプション"
    MSG_HELP_EXAMPLES="例"
else
    MSG_TITLE="🔗 DockMCP Setup"
    MSG_REGISTERED="Registered"
    MSG_NOT_REGISTERED="Not registered"
    MSG_CLI_NOT_FOUND="CLI not installed"
    MSG_ALREADY_REGISTERED="Already registered"
    MSG_REGISTERING="Registering..."
    MSG_REGISTERED_OK="Registered successfully"
    MSG_REGISTERED_FALLBACK="Registered via .mcp.json (fallback)"
    MSG_REGISTER_FAILED="Registration failed"
    MSG_CONNECTIVITY="Connectivity"
    MSG_SERVER_RUNNING="DockMCP server is running"
    MSG_SERVER_NOT_RUNNING="DockMCP server is not reachable"
    MSG_START_HINT="Start DockMCP on host OS:"
    MSG_NEXT_STEPS="Next Steps"
    MSG_CLAUDE_RECONNECT="Claude Code: Run /mcp -> Reconnect"
    MSG_GEMINI_RESTART="Gemini CLI: Restart the session"
    MSG_NO_AI_TOOLS="No AI tools found (neither claude nor gemini)"
    MSG_UNREGISTER_TITLE="🔗 DockMCP Unregister"
    MSG_UNREGISTERED="Removed"
    MSG_NOT_FOUND="Not registered, skipping"
    MSG_HELP_USAGE="Usage"
    MSG_HELP_OPTIONS="Options"
    MSG_HELP_EXAMPLES="Examples"
fi

# ─── Constants / 定数 ──────────────────────────────────────────

WORKSPACE="${WORKSPACE:-/workspace}"
DEFAULT_URL="http://host.docker.internal:8080/sse"
DKMCP_NAME="dkmcp"

# ─── Help / ヘルプ ─────────────────────────────────────────────

show_help() {
    echo ""
    echo "$MSG_HELP_USAGE:"
    echo "  .sandbox/scripts/setup-dkmcp.sh [options]"
    echo ""
    echo "$MSG_HELP_OPTIONS:"
    echo "  --check       Silent check (exit: 0=connected, 1=not registered, 2=offline)"
    echo "  --status      Human-readable status report"
    echo "  --url <url>   Custom DockMCP URL (default: $DEFAULT_URL)"
    echo "  --unregister  Remove DockMCP from all AI tools"
    echo "  --help, -h    Show this help"
    echo ""
    echo "$MSG_HELP_EXAMPLES:"
    echo "  .sandbox/scripts/setup-dkmcp.sh              # Register + verify"
    echo "  .sandbox/scripts/setup-dkmcp.sh --check      # Silent check"
    echo "  .sandbox/scripts/setup-dkmcp.sh --status     # Show status"
    echo "  .sandbox/scripts/setup-dkmcp.sh --unregister # Remove registration"
    echo ""
    exit 0
}

# ─── Argument parsing / 引数解析 ──────────────────────────────

MODE="default"
DKMCP_URL="$DEFAULT_URL"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --check)      MODE="check"; shift ;;
        --status)     MODE="status"; shift ;;
        --unregister) MODE="unregister"; shift ;;
        --url)
            [[ -z "${2:-}" ]] && die "--url requires a URL"
            DKMCP_URL="$2"; shift 2 ;;
        --help|-h)    show_help ;;
        -*)           die "Unknown option: $1" ;;
        *)            die "Unexpected argument: $1" ;;
    esac
done

# ─── Tool detection / ツール検出 ──────────────────────────────

has_claude() { command -v claude >/dev/null 2>&1; }
has_gemini() { command -v gemini >/dev/null 2>&1; }

# Claude registration is possible via .mcp.json or .mcp.json.example even without claude CLI
# claude CLI がなくても .mcp.json / .mcp.json.example 経由で登録可能
can_register_claude() {
    has_claude || \
    [[ -f "$WORKSPACE/.mcp.json" ]] || \
    [[ -f "$WORKSPACE/.mcp.json.example" ]]
}

# Gemini registration is possible via .gemini/settings.json even without gemini CLI
# gemini CLI がなくても .gemini/settings.json 経由で登録可能
can_register_gemini() {
    has_gemini || \
    [[ -f "$WORKSPACE/.gemini/settings.json" ]]
}

# ─── Registration detection / 登録検出 ────────────────────────

is_claude_registered() {
    # Check .mcp.json (project shared config)
    if [[ -f "$WORKSPACE/.mcp.json" ]] && \
       jq -e ".mcpServers[\"$DKMCP_NAME\"]" "$WORKSPACE/.mcp.json" >/dev/null 2>&1; then
        return 0
    fi
    # Check ~/.claude.json project scope
    if [[ -f "$HOME/.claude.json" ]] && \
       jq -e ".projects[\"$WORKSPACE\"].mcpServers[\"$DKMCP_NAME\"]" "$HOME/.claude.json" >/dev/null 2>&1; then
        return 0
    fi
    # Check ~/.claude.json user scope
    if [[ -f "$HOME/.claude.json" ]] && \
       jq -e ".mcpServers[\"$DKMCP_NAME\"]" "$HOME/.claude.json" >/dev/null 2>&1; then
        return 0
    fi
    return 1
}

is_gemini_registered() {
    # Check project-scope settings
    if [[ -f "$WORKSPACE/.gemini/settings.json" ]] && \
       jq -e ".mcpServers[\"$DKMCP_NAME\"]" "$WORKSPACE/.gemini/settings.json" >/dev/null 2>&1; then
        return 0
    fi
    # Check user-scope settings
    if [[ -f "$HOME/.gemini/settings.json" ]] && \
       jq -e ".mcpServers[\"$DKMCP_NAME\"]" "$HOME/.gemini/settings.json" >/dev/null 2>&1; then
        return 0
    fi
    return 1
}

# ─── Connectivity check / 接続確認 ────────────────────────────

check_connectivity() {
    local url="$1"
    local base_url="${url%/sse}"

    # Try base URL - even a 404 means the server is reachable
    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" \
        --connect-timeout 3 --max-time 5 "${base_url}/" 2>/dev/null) || true

    [[ "$http_code" != "000" && "$http_code" != "" ]]
}

# ─── Safe JSON write / 安全な JSON 書き込み ───────────────────

safe_write_json() {
    local target="$1"
    shift
    local tmp="${target}.tmp.$$"
    if jq "$@" > "$tmp"; then
        mv "$tmp" "$target"
    else
        rm -f "$tmp"
        return 1
    fi
}

# ─── Registration / 登録 ──────────────────────────────────────

register_claude() {
    local url="$1"

    # Primary: use claude CLI (official method)
    if has_claude; then
        (cd "$WORKSPACE" && claude mcp add --transport sse --scope user "$DKMCP_NAME" "$url" >/dev/null 2>&1)
        return $?
    fi

    # Fallback: write .mcp.json directly
    local mcp_json="$WORKSPACE/.mcp.json"
    if [[ -f "$mcp_json" ]]; then
        safe_write_json "$mcp_json" --arg url "$url" --arg name "$DKMCP_NAME" \
            '.mcpServers[$name] = {"type": "sse", "url": $url}' "$mcp_json"
    elif [[ -f "$WORKSPACE/.mcp.json.example" ]]; then
        safe_write_json "$mcp_json" --arg url "$url" --arg name "$DKMCP_NAME" \
            '.mcpServers[$name] = {"type": "sse", "url": $url}' "$WORKSPACE/.mcp.json.example"
    fi
}

register_gemini() {
    local url="$1"

    # Primary: use gemini CLI (official method)
    if has_gemini; then
        (cd "$WORKSPACE" && gemini mcp add --transport sse "$DKMCP_NAME" "$url" >/dev/null 2>&1)
        return $?
    fi

    # Fallback: write .gemini/settings.json directly
    local settings="$WORKSPACE/.gemini/settings.json"
    mkdir -p "$WORKSPACE/.gemini"
    if [[ -f "$settings" ]]; then
        safe_write_json "$settings" --arg url "$url" --arg name "$DKMCP_NAME" \
            '.mcpServers[$name] = {"url": $url, "type": "sse"}' "$settings"
    else
        safe_write_json "$settings" -n --arg url "$url" --arg name "$DKMCP_NAME" \
            '{"mcpServers":{($name):{"url":$url,"type":"sse"}}}'
    fi
}

# ─── Unregistration / 登録解除 ────────────────────────────────

unregister_claude() {
    local removed=false

    # Remove from .mcp.json
    local mcp_json="$WORKSPACE/.mcp.json"
    if [[ -f "$mcp_json" ]] && jq -e ".mcpServers[\"$DKMCP_NAME\"]" "$mcp_json" >/dev/null 2>&1; then
        if safe_write_json "$mcp_json" --arg name "$DKMCP_NAME" 'del(.mcpServers[$name])' "$mcp_json"; then
            removed=true
        fi
    fi

    # Remove via CLI (handles user/project scope in ~/.claude.json)
    if has_claude; then
        if (cd "$WORKSPACE" && claude mcp remove "$DKMCP_NAME" >/dev/null 2>&1); then
            removed=true
        fi
    fi

    [[ "$removed" == true ]]
}

unregister_gemini() {
    local removed=false

    # Remove from project settings
    local settings="$WORKSPACE/.gemini/settings.json"
    if [[ -f "$settings" ]] && jq -e ".mcpServers[\"$DKMCP_NAME\"]" "$settings" >/dev/null 2>&1; then
        if safe_write_json "$settings" --arg name "$DKMCP_NAME" 'del(.mcpServers[$name])' "$settings"; then
            removed=true
        fi
    fi

    # Remove via CLI (handles user scope)
    if has_gemini; then
        if (cd "$WORKSPACE" && gemini mcp remove "$DKMCP_NAME" >/dev/null 2>&1); then
            removed=true
        fi
    fi

    [[ "$removed" == true ]]
}

# ─── Mode: check / チェックモード ─────────────────────────────
# Returns: 0=registered+connected, 1=not registered, 2=registered but offline

mode_check() {
    local registered=false

    if can_register_claude; then
        is_claude_registered && registered=true
    fi
    if can_register_gemini; then
        is_gemini_registered && registered=true
    fi

    if [[ "$registered" == false ]]; then
        exit 1
    fi

    if check_connectivity "$DKMCP_URL"; then
        exit 0
    else
        exit 2
    fi
}

# ─── Mode: status / ステータスモード ──────────────────────────

mode_status() {
    echo ""
    echo -e "${BOLD}${MSG_TITLE}${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    # Per-tool status
    if can_register_claude; then
        if is_claude_registered; then
            ok "[Claude] $MSG_REGISTERED"
        else
            warn "[Claude] $MSG_NOT_REGISTERED"
        fi
    else
        echo -e "  ${DIM}[Claude] $MSG_CLI_NOT_FOUND${NC}"
    fi

    if can_register_gemini; then
        if is_gemini_registered; then
            ok "[Gemini] $MSG_REGISTERED"
        else
            warn "[Gemini] $MSG_NOT_REGISTERED"
        fi
    else
        echo -e "  ${DIM}[Gemini] $MSG_CLI_NOT_FOUND${NC}"
    fi

    echo ""

    # Connectivity
    echo -e "${BOLD}$MSG_CONNECTIVITY${NC}"
    echo "──────────────────────────────────────"
    if check_connectivity "$DKMCP_URL"; then
        ok "$MSG_SERVER_RUNNING ($DKMCP_URL)"
    else
        warn "$MSG_SERVER_NOT_RUNNING ($DKMCP_URL)"
        echo ""
        info "$MSG_START_HINT"
        echo -e "  ${CYAN}cd dkmcp && make install && dkmcp serve${NC}"
    fi
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# ─── Mode: default / デフォルトモード ─────────────────────────

mode_default() {
    local any_new=false
    local has_any_tool=false

    echo ""
    echo -e "${BOLD}${MSG_TITLE}${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    # Claude
    if can_register_claude; then
        has_any_tool=true
        if is_claude_registered; then
            ok "[Claude] $MSG_ALREADY_REGISTERED"
        else
            info "[Claude] $MSG_REGISTERING"
            if register_claude "$DKMCP_URL"; then
                if has_claude; then
                    ok "[Claude] $MSG_REGISTERED_OK"
                else
                    ok "[Claude] $MSG_REGISTERED_FALLBACK"
                fi
                any_new=true
            else
                err "[Claude] $MSG_REGISTER_FAILED"
            fi
        fi
    fi

    # Gemini
    if can_register_gemini; then
        has_any_tool=true
        if is_gemini_registered; then
            ok "[Gemini] $MSG_ALREADY_REGISTERED"
        else
            info "[Gemini] $MSG_REGISTERING"
            if register_gemini "$DKMCP_URL"; then
                if has_gemini; then
                    ok "[Gemini] $MSG_REGISTERED_OK"
                else
                    ok "[Gemini] $MSG_REGISTERED_FALLBACK"
                fi
                any_new=true
            else
                err "[Gemini] $MSG_REGISTER_FAILED"
            fi
        fi
    fi

    # No tools found
    if [[ "$has_any_tool" == false ]]; then
        warn "$MSG_NO_AI_TOOLS"
        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        exit 1
    fi

    echo ""

    # Connectivity check
    echo -e "${BOLD}$MSG_CONNECTIVITY${NC}"
    echo "──────────────────────────────────────"
    if check_connectivity "$DKMCP_URL"; then
        ok "$MSG_SERVER_RUNNING"
    else
        warn "$MSG_SERVER_NOT_RUNNING"
        echo ""
        info "$MSG_START_HINT"
        echo -e "  ${CYAN}cd dkmcp && make install && dkmcp serve${NC}"
    fi

    # Post-registration guidance
    if [[ "$any_new" == true ]]; then
        echo ""
        echo -e "${BOLD}$MSG_NEXT_STEPS${NC}"
        echo "──────────────────────────────────────"
        if has_claude; then
            info "$MSG_CLAUDE_RECONNECT"
        fi
        if has_gemini; then
            info "$MSG_GEMINI_RESTART"
        fi
    fi

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# ─── Mode: unregister / 登録解除モード ────────────────────────

mode_unregister() {
    echo ""
    echo -e "${BOLD}${MSG_UNREGISTER_TITLE}${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    if can_register_claude; then
        if unregister_claude; then
            ok "[Claude] $MSG_UNREGISTERED"
        else
            echo -e "  ${DIM}[Claude] $MSG_NOT_FOUND${NC}"
        fi
    fi

    if can_register_gemini; then
        if unregister_gemini; then
            ok "[Gemini] $MSG_UNREGISTERED"
        else
            echo -e "  ${DIM}[Gemini] $MSG_NOT_FOUND${NC}"
        fi
    fi

    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# ─── Main / メイン ────────────────────────────────────────────

case "$MODE" in
    check)      mode_check ;;
    status)     mode_status ;;
    unregister) mode_unregister ;;
    default)    mode_default ;;
esac

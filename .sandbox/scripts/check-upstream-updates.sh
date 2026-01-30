#!/bin/bash
# check-upstream-updates.sh
# Check for updates to the upstream repository
# アップストリームリポジトリの更新をチェック
#
# This script checks GitHub releases API for new versions
# and notifies the user if updates are available.
# このスクリプトはGitHub releases APIをチェックし、
# 更新があればユーザーに通知します。

set -e

WORKSPACE="${WORKSPACE:-/workspace}"

# Source common startup functions
# 共通起動関数を読み込み
# shellcheck source=/dev/null
source "${WORKSPACE}/.sandbox/scripts/_startup_common.sh"

# Configuration file path
# 設定ファイルのパス
TEMPLATE_CONFIG="${WORKSPACE}/.sandbox/config/template-source.conf"

# State file for check interval and last notified version
# チェック間隔と前回通知バージョン用の状態ファイル
# Format: <unix_timestamp>:<version>
# 形式: <UNIXタイムスタンプ>:<バージョン>
STATE_FILE="${STATE_FILE:-${WORKSPACE}/.sandbox/.state/update-check}"

# Debug mode: --debug flag or DEBUG_UPDATE_CHECK=1 environment variable
# デバッグモード: --debug フラグ または DEBUG_UPDATE_CHECK=1 環境変数
DEBUG_MODE="${DEBUG_UPDATE_CHECK:-0}"

# Parse --debug flag
# --debug フラグを解析
for arg in "$@"; do
    if [ "$arg" = "--debug" ]; then
        DEBUG_MODE=1
        break
    fi
done

# Output debug message to stderr
# デバッグメッセージを stderr に出力
debug_log() {
    if [ "$DEBUG_MODE" = "1" ]; then
        echo "[debug] $*" >&2
    fi
}

# Load template configuration
# テンプレート設定を読み込み
# Returns: 0 if config loaded, 1 if no config file
load_template_config() {
    if [ ! -f "$TEMPLATE_CONFIG" ]; then
        debug_log "Config not found: $TEMPLATE_CONFIG → skip"
        return 1
    fi
    # shellcheck source=/dev/null
    source "$TEMPLATE_CONFIG"
    debug_log "Config loaded: REPO=$TEMPLATE_REPO, CHANNEL=${CHECK_CHANNEL:-all}, UPDATES=$CHECK_UPDATES, INTERVAL=${CHECK_INTERVAL_HOURS}h"
    return 0
}

# Language detection based on locale
# ロケールに基づく言語検出
setup_messages() {
    if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
        MSG_TITLE="📦 更新チェック"
        MSG_UPDATE_AVAILABLE="更新があります"
        MSG_CURRENT="現在のバージョン"
        MSG_LATEST="最新バージョン"
        MSG_RELEASE_NOTES="リリースノート"
        MSG_HOW_TO_UPDATE="更新方法"
        MSG_HOW_TO_UPDATE_1="1. リリースノートで変更内容を確認"
        MSG_HOW_TO_UPDATE_2="2. 必要な変更を手動で適用"
    else
        MSG_TITLE="📦 Update Check"
        MSG_UPDATE_AVAILABLE="Update available"
        MSG_CURRENT="Current version"
        MSG_LATEST="Latest version"
        MSG_RELEASE_NOTES="Release notes"
        MSG_HOW_TO_UPDATE="How to update"
        MSG_HOW_TO_UPDATE_1="1. Check release notes for changes"
        MSG_HOW_TO_UPDATE_2="2. Manually apply relevant updates"
    fi
}

# Read timestamp from state file
# 状態ファイルからタイムスタンプを読み取り
read_state_timestamp() {
    if [ -f "$STATE_FILE" ]; then
        cut -d: -f1 "$STATE_FILE" 2>/dev/null || echo "0"
    else
        echo "0"
    fi
}

# Read last notified version from state file
# 状態ファイルから前回通知バージョンを読み取り
get_last_notified_version() {
    if [ -f "$STATE_FILE" ]; then
        cut -d: -f2- "$STATE_FILE" 2>/dev/null || echo ""
    else
        echo ""
    fi
}

# Check if this is the first run (no state file)
# 初回実行かどうか（状態ファイルがない）
is_first_run() {
    [ ! -f "$STATE_FILE" ]
}

# Check if enough time has passed since last check
# 前回のチェックから十分な時間が経過したか確認
should_check() {
    local interval_hours="${CHECK_INTERVAL_HOURS:-24}"

    # Validate: must be a non-negative integer, fallback to 24 if invalid
    # バリデーション: 非負整数でなければ24にフォールバック
    if ! [[ "$interval_hours" =~ ^[0-9]+$ ]]; then
        interval_hours=24
    fi

    # 0 means check every time
    # 0 は毎回チェック
    if [ "$interval_hours" -eq 0 ]; then
        debug_log "Interval: 0 (always check)"
        return 0
    fi

    local interval_seconds=$((interval_hours * 3600))
    local last_check
    last_check=$(read_state_timestamp)

    if [ "$last_check" != "0" ]; then
        local now
        now=$(date +%s)
        local elapsed=$((now - last_check))

        if [ $elapsed -lt $interval_seconds ]; then
            debug_log "Interval: ${elapsed}s elapsed < ${interval_seconds}s required → skip"
            return 1
        fi
        debug_log "Interval: ${elapsed}s elapsed >= ${interval_seconds}s required → check"
    else
        debug_log "Interval: no state file → first check"
    fi

    return 0
}

# Update state file with timestamp and version
# 状態ファイルをタイムスタンプとバージョンで更新
update_state() {
    local version="${1:-}"
    local state_dir
    state_dir=$(dirname "$STATE_FILE")
    mkdir -p "$state_dir" 2>/dev/null || true
    echo "$(date +%s):${version}" > "$STATE_FILE" 2>/dev/null || true
}

# Build GitHub API URL based on channel setting
# チャンネル設定に応じた GitHub API URL を構築
build_api_url() {
    local repo="$1"
    local channel="${CHECK_CHANNEL:-all}"

    case "$channel" in
        stable)
            # Official releases only (non-prerelease, non-draft)
            # 正式リリースのみ（プレリリース・ドラフト除外）
            echo "https://api.github.com/repos/${repo}/releases/latest"
            ;;
        *)
            # All releases including pre-releases (default)
            # プレリリースを含む全リリース（デフォルト）
            echo "https://api.github.com/repos/${repo}/releases?per_page=1"
            ;;
    esac
}

# Extract tag_name from API response JSON
# APIレスポンスJSONからtag_nameを抽出
extract_tag_from_json() {
    local json_file="$1"
    local channel="${CHECK_CHANNEL:-all}"

    # /releases?per_page=1 returns an array, /releases/latest returns an object
    # /releases?per_page=1 は配列、/releases/latest はオブジェクトを返す
    local jq_expr
    if [ "$channel" = "stable" ]; then
        jq_expr='.tag_name // empty'
    else
        jq_expr='.[0].tag_name // empty'
    fi

    if command -v jq &>/dev/null; then
        jq -r "$jq_expr" "$json_file" 2>/dev/null
    else
        # Fallback: grep for first tag_name (works for both array and object)
        # フォールバック: 最初の tag_name を grep（配列・オブジェクト両対応）
        grep -o '"tag_name"[[:space:]]*:[[:space:]]*"[^"]*"' "$json_file" 2>/dev/null | \
            sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/' | head -1
    fi
}

# Fetch latest release from GitHub API
# GitHub API から最新リリースを取得
fetch_latest_release() {
    local repo="$1"
    local api_url
    api_url=$(build_api_url "$repo")
    local tmp_file="/tmp/ai-sandbox-release-check.json"

    # Fetch with timeout, capture HTTP status
    # タイムアウト付きで取得、HTTPステータスをキャプチャ
    local http_code
    http_code=$(curl -s \
        --connect-timeout 1 \
        --max-time 3 \
        -w "%{http_code}" \
        -o "$tmp_file" \
        "$api_url" 2>/dev/null) || http_code="000"

    debug_log "API: $api_url → HTTP $http_code"

    # Check HTTP status
    case "$http_code" in
        200)
            # Success - extract tag_name
            # 成功 - tag_name を抽出
            local tag
            tag=$(extract_tag_from_json "$tmp_file")
            debug_log "API: tag_name=$tag"
            echo "$tag"
            rm -f "$tmp_file" 2>/dev/null
            return 0
            ;;
        *)
            # 404: No releases, 403: Rate limit, others: Network error
            # All cases: skip silently
            debug_log "API: failed (HTTP $http_code) → skip"
            rm -f "$tmp_file" 2>/dev/null
            return 1
            ;;
    esac
}

# Main
# メイン処理
main() {
    # Load config, exit if not found
    if ! load_template_config; then
        return 0
    fi
    setup_messages

    # Check if updates are enabled
    # 更新チェックが有効か確認
    if [ "${CHECK_UPDATES:-true}" != "true" ]; then
        debug_log "CHECK_UPDATES=${CHECK_UPDATES} → disabled, exit"
        exit 0
    fi

    # Check if template repo is configured
    # リポジトリが設定されているか確認
    if [ -z "${TEMPLATE_REPO:-}" ]; then
        debug_log "TEMPLATE_REPO is empty → exit"
        exit 0
    fi

    # Check interval
    # 間隔チェック
    if ! should_check; then
        exit 0
    fi

    # Fetch latest release
    # 最新リリースを取得
    local latest_version
    latest_version=$(fetch_latest_release "$TEMPLATE_REPO") || {
        debug_log "Fetch failed → exit"
        exit 0
    }

    # No release found
    if [ -z "$latest_version" ]; then
        debug_log "No release found → exit"
        update_state ""
        exit 0
    fi

    # First run: record version without notification
    # 初回実行: 通知せずバージョンを記録
    if is_first_run; then
        debug_log "First run → record $latest_version, no notification"
        update_state "$latest_version"
        exit 0
    fi

    # Compare with last notified version
    # 前回通知バージョンと比較
    local last_notified
    last_notified=$(get_last_notified_version)
    debug_log "Compare: last_notified=$last_notified, latest=$latest_version"

    if [ "$last_notified" = "$latest_version" ]; then
        debug_log "Same version → no notification"
        update_state "$latest_version"
        exit 0
    fi

    # New version available - show notification
    # 新バージョンあり - 通知表示
    local release_url="https://github.com/${TEMPLATE_REPO}/releases"
    debug_log "New version → notification"
    show_update_notification "$last_notified" "$latest_version" "$release_url"

    update_state "$latest_version"
}

# Show update notification based on verbosity
# 詳細度に応じて更新通知を表示
show_update_notification() {
    local previous="$1"
    local latest="$2"
    local url="$3"

    # Build version display
    # バージョン表示を構築
    local version_display
    if [ -n "$previous" ]; then
        version_display="$previous → $latest"
    else
        version_display="$latest"
    fi

    # ============================================================
    # Quiet mode: minimal output
    # ============================================================
    if is_quiet; then
        echo "📦 $MSG_UPDATE_AVAILABLE: $version_display"
        return
    fi

    # ============================================================
    # Default mode: summary with URL
    # ============================================================
    if is_default; then
        print_title "$MSG_TITLE"

        if [ -n "$previous" ]; then
            echo "  $MSG_CURRENT:  $previous"
        fi
        echo "  $MSG_LATEST:   $latest"
        echo "  $MSG_RELEASE_NOTES:"
        echo "    $url"

        print_footer
        return
    fi

    # ============================================================
    # Verbose mode: full details
    # ============================================================
    print_title "$MSG_TITLE"

    if [ -n "$previous" ]; then
        echo "  $MSG_CURRENT:  $previous"
    fi
    echo "  $MSG_LATEST:   $latest"
    echo ""
    echo "  $MSG_HOW_TO_UPDATE:"
    echo "    $MSG_HOW_TO_UPDATE_1"
    echo "    $MSG_HOW_TO_UPDATE_2"
    echo ""
    echo "  $MSG_RELEASE_NOTES:"
    echo "    $url"

    print_footer
}

# Only run main if script is executed directly (not sourced)
# スクリプトが直接実行された場合のみ main を実行（source された場合は実行しない）
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi

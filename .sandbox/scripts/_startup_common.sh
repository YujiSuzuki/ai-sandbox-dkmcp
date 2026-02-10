#!/bin/bash
# _startup_common.sh
# Common functions for startup scripts with verbosity support
#
# Usage: source this file at the beginning of startup scripts
#
#   source "${WORKSPACE:-/workspace}/.sandbox/scripts/_startup_common.sh"
# ---
# 詳細度サポート付き起動スクリプト用共通関数
# 使用法: 起動スクリプトの冒頭でこのファイルを source する

# Configuration paths
# 設定パス
WORKSPACE="${WORKSPACE:-/workspace}"
STARTUP_CONFIG="${WORKSPACE}/.sandbox/config/startup.conf"
SYNC_IGNORE_FILE="${WORKSPACE}/.sandbox/config/sync-ignore"

# Load configuration file
# 設定ファイルの読み込み
load_startup_config() {
    # Save environment variable values before sourcing config
    # 設定ファイル読み込み前に環境変数の値を保存
    local env_verbosity="${STARTUP_VERBOSITY:-}"
    local env_readme_url="${SANDBOX_README_URL:-}"
    local env_readme_url_ja="${SANDBOX_README_URL_JA:-}"
    local env_backup_keep="${BACKUP_KEEP_COUNT:-}"

    # Load config file if exists
    # 設定ファイルが存在すれば読み込み
    if [ -f "$STARTUP_CONFIG" ]; then
        # shellcheck source=/dev/null
        source "$STARTUP_CONFIG"
    fi

    # Environment variables take precedence over config file
    # 環境変数は設定ファイルより優先
    README_URL="${env_readme_url:-${README_URL:-README.md}}"
    README_URL_JA="${env_readme_url_ja:-${README_URL_JA:-README.ja.md}}"
    STARTUP_VERBOSITY="${env_verbosity:-${STARTUP_VERBOSITY:-verbose}}"
    BACKUP_KEEP_COUNT="${env_backup_keep:-${BACKUP_KEEP_COUNT:-0}}"

    export README_URL README_URL_JA STARTUP_VERBOSITY BACKUP_KEEP_COUNT
}

# ============================================================
# README URL Functions / README URL 関数
# ============================================================

# Get README URL based on locale
# ロケールに応じた README URL を取得
get_readme_url() {
    if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
        echo "${README_URL_JA:-README.ja.md}"
    else
        echo "${README_URL:-README.md}"
    fi
}

# Get "See README for details" message
# 「詳細はREADMEを参照」メッセージを取得
get_readme_reference_message() {
    local url
    url=$(get_readme_url)
    if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
        echo "詳細は ${url} を参照してください。"
    else
        echo "See ${url} for details."
    fi
}

# ============================================================
# Verbosity Helper Functions / 詳細度ヘルパー関数
# ============================================================

# Check verbosity level
# 詳細度レベルをチェック
is_quiet() { [[ "$STARTUP_VERBOSITY" == "quiet" ]]; }
is_verbose() { [[ "$STARTUP_VERBOSITY" == "verbose" ]]; }
is_summary() { [[ "$STARTUP_VERBOSITY" == "summary" ]]; }

# Print script title (thick separator)
# スクリプトタイトルを出力（太線セパレータ）
print_title() {
    is_quiet && return
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "$1"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    is_verbose && echo "" || true
}

# Print script footer (thick separator)
# スクリプトフッターを出力（太線セパレータ）
print_footer() {
    is_quiet && return
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
}

# Print summary line (for summary mode)
# サマリー行を出力（summary モード用）
# Usage: print_summary "emoji" "message" "OK|WARN|ERR"
print_summary() {
    local emoji="$1" msg="$2" result="$3"
    case "$result" in
        OK)
            is_quiet || echo "${emoji} ${msg}"
            ;;
        WARN)
            echo "⚠️  ${msg}"
            ;;
        ERR)
            echo "❌ ${msg}"
            ;;
    esac
}

# Print detail (verbose mode only)
# 詳細を出力（verbose モードのみ）
print_detail() {
    is_verbose && echo "$1" || true
}

# Print default (not in quiet mode)
# デフォルト出力（quiet モード以外）
print_default() {
    is_quiet || echo "$1"
    true  # Always return success for set -e compatibility
}

# Print always (warning/error)
# 常に出力（警告/エラー）
print_warning() {
    echo "⚠️  $1"
}

print_error() {
    echo "❌ $1" >&2
}

# ============================================================
# Sync-Ignore Functions / Sync-Ignore 関数
# ============================================================

# Load sync-ignore patterns
# sync-ignore パターンを読み込み
# Returns patterns one per line, comments and empty lines excluded
# パターンを1行ずつ返す（コメントと空行を除外）
load_sync_ignore_patterns() {
    [ -f "$SYNC_IGNORE_FILE" ] || return 0
    grep -v '^#' "$SYNC_IGNORE_FILE" | grep -v '^[[:space:]]*$' || true
}

# Check if a file matches any sync-ignore pattern
# ファイルが sync-ignore パターンにマッチするかチェック
# Usage: matches_sync_ignore "/workspace/path/to/file"
# Returns: 0 if matches (should ignore), 1 if not
matches_sync_ignore() {
    local file_path="$1"
    local rel_path="${file_path#$WORKSPACE/}"
    local filename
    filename=$(basename "$file_path")
    local pattern

    while IFS= read -r pattern; do
        [ -z "$pattern" ] && continue

        # Handle ** patterns (recursive matching)
        # ** パターンを処理（再帰マッチング）
        if [[ "$pattern" == "**/"* ]]; then
            # **/*.example -> matches any file ending with .example
            # **/*.sample -> matches any file ending with .sample
            local suffix="${pattern#\*\*/}"

            # If suffix contains *, use glob matching on filename
            # suffix に * が含まれる場合、ファイル名に対してグロブマッチング
            if [[ "$suffix" == "*"* ]]; then
                # Extract the extension part (e.g., ".example" from "*.example")
                local ext="${suffix#\*}"
                if [[ "$filename" == *"$ext" ]]; then
                    return 0
                fi
            elif [[ "$rel_path" == *"$suffix" ]]; then
                return 0
            fi
        elif [[ "$pattern" == *"/**" ]]; then
            # path/** -> matches anything under path/
            local prefix="${pattern%/\*\*}"
            if [[ "$rel_path" == "$prefix/"* ]]; then
                return 0
            fi
        elif [[ "$pattern" == *"*"* ]]; then
            # Simple wildcard matching on the full path
            # パス全体に対する単純なワイルドカードマッチング
            # shellcheck disable=SC2053
            if [[ "$rel_path" == $pattern ]]; then
                return 0
            fi
        else
            # Exact match
            # 完全一致
            if [[ "$rel_path" == "$pattern" ]]; then
                return 0
            fi
        fi
    done < <(load_sync_ignore_patterns)

    return 1
}

# ============================================================
# Backup Utility Functions / バックアップユーティリティ関数
# ============================================================

# Backup directory
# バックアップ保存先ディレクトリ
BACKUP_DIR="${WORKSPACE}/.sandbox/backups"

# Create a backup of a file in .sandbox/backups/
# .sandbox/backups/ にファイルのバックアップを作成
#
# Usage: backup_file "/path/to/file" "label"
# Example: backup_file "$COMPOSE_FILE" "devcontainer"
#   -> .sandbox/backups/devcontainer.docker-compose.yml.20260130123456
#
# Returns: backup file path via stdout
backup_file() {
    local file="$1"
    local label="${2:-}"
    local timestamp
    timestamp=$(date +%Y%m%d%H%M%S)

    mkdir -p "$BACKUP_DIR"

    local file_basename
    file_basename=$(basename "$file")
    local backup_name
    if [ -n "$label" ]; then
        backup_name="${label}.${file_basename}.${timestamp}"
    else
        backup_name="${file_basename}.${timestamp}"
    fi

    local backup_path="${BACKUP_DIR}/${backup_name}"
    cp "$file" "$backup_path"
    echo "$backup_path"
}

# Clean up old backups, keeping only the most recent N
# 古いバックアップを削除し、直近 N 件のみ保持
#
# Usage: cleanup_backups "label.docker-compose.yml.*" [count]
#   count defaults to BACKUP_KEEP_COUNT (0 = unlimited, no cleanup)
cleanup_backups() {
    local pattern="$1"
    local keep="${2:-$BACKUP_KEEP_COUNT}"

    # 0 or non-numeric means unlimited
    # 0 または数値以外は無制限
    if ! [[ "$keep" =~ ^[0-9]+$ ]] || [ "$keep" -le 0 ]; then
        return 0
    fi
    [ ! -d "$BACKUP_DIR" ] && return 0

    # List matching files sorted by modification time (newest first)
    # 更新日時の降順でマッチするファイルを一覧
    local count=0
    while IFS= read -r f; do
        [ -z "$f" ] && continue
        count=$((count + 1))
        if [ "$count" -gt "$keep" ]; then
            rm -f "$f"
        fi
    done < <(ls -1t "${BACKUP_DIR}"/${pattern} 2>/dev/null)
}

# ============================================================
# Initialization / 初期化
# ============================================================

# Auto-load configuration when sourced
# source 時に自動で設定を読み込み
load_startup_config

#!/bin/bash
# sync-secrets.sh
# Interactive script to sync secret files from .claude/settings.json to docker-compose.yml
#
# This script finds files blocked in Claude settings that are not hidden in docker-compose.yml,
# and offers to add them interactively. Updates both DevContainer and CLI Sandbox configs.
#
# IMPORTANT: Must run inside AI Sandbox container (not on host OS). Auto-detects which
# environment to use ($SANDBOX_ENV: devcontainer, cli_claude, cli_gemini, cli_ai_sandbox).
# ---
# .claude/settings.json から docker-compose.yml へ秘匿ファイルを同期する対話式スクリプト
# このスクリプトは Claude 設定でブロックされているが docker-compose.yml で隠蔽されていない
# ファイルを見つけ、対話式で追加を提案します。DevContainer と CLI Sandbox の両方を更新します。

set -e

# Check if running on host OS (not in container)
# ホストOSで実行されていないかチェック
if [[ -z "${SANDBOX_ENV:-}" ]] && [[ ! -f "/.dockerenv" ]]; then
    if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
        echo "❌ このスクリプトはホストOSでは実行できません。"
        echo ""
        echo "以下のいずれかの環境で実行してください："
        echo "  • AI Sandbox のターミナル"
        echo "  • cli_sandbox/ai_sandbox.sh"
        echo ""
        echo "または、手動で docker-compose.yml を編集してください。"
    else
        echo "❌ This script cannot be run on the host OS."
        echo ""
        echo "Please run in one of these environments:"
        echo "  • AI Sandbox terminal"
        echo "  • cli_sandbox/ai_sandbox.sh"
        echo ""
        echo "Or manually edit docker-compose.yml."
    fi
    exit 1
fi

WORKSPACE="${WORKSPACE:-/workspace}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common startup functions for sync-ignore support
# sync-ignore サポート用に共通起動関数を読み込み
# shellcheck source=/dev/null
source "${WORKSPACE}/.sandbox/scripts/_startup_common.sh"

# Both docker-compose.yml files
# 両方の docker-compose.yml
DEVCONTAINER_COMPOSE="$WORKSPACE/.devcontainer/docker-compose.yml"
CLI_SANDBOX_COMPOSE="$WORKSPACE/cli_sandbox/docker-compose.yml"

CLAUDE_SETTINGS="$WORKSPACE/.claude/settings.json"

# Short labels for compose files
# compose ファイルの短縮ラベル
LABEL_DC="DevContainer"
LABEL_CLI="CLI Sandbox"

# Language detection based on locale
# ロケールに基づく言語検出
if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
    MSG_TITLE="🔧 シークレット設定同期ツール"
    MSG_CHECKING="チェック中..."
    MSG_NO_SETTINGS="Claude 設定ファイルが見つかりません"
    MSG_NO_COMPOSE="docker-compose.yml が見つかりません（両方とも）"
    MSG_ALL_SYNCED="✅ すべての秘匿ファイルが同期されています。追加は不要です。"
    MSG_FOUND_HEADER="以下のファイルが docker-compose.yml に未設定です:"
    MSG_MISSING_FROM="未設定:"
    MSG_PROMPT_ALL="これらすべてを docker-compose.yml に追加しますか？"
    MSG_YES_ALL="すべて追加"
    MSG_YES_EACH="個別確認"
    MSG_NO="追加しない"
    MSG_PREVIEW="プレビュー表示（ドライラン）"
    MSG_CONFIRM_FILE="追加しますか？"
    MSG_ADDING="追加中:"
    MSG_ADDED="✅ 追加しました"
    MSG_SKIPPED="⏭️  スキップしました"
    MSG_DONE_HEADER="完了！"
    MSG_DONE_ADDED="追加されたファイル:"
    MSG_DONE_NONE="追加されたファイルはありません"
    MSG_REBUILD="変更を反映するにはコンテナをリビルドしてください:"
    MSG_REBUILD_CMD="  VS Code: Ctrl+Shift+P → 'Dev Containers: Rebuild Container'"
    MSG_REBUILD_CLI="  CLI: ./cli_sandbox/build.sh"
    MSG_NO_DENY="deny 設定にファイルパターンがありません"
    MSG_NO_FILES="該当するファイルが見つかりませんでした"
    MSG_BACKUP="バックアップを作成しました:"
    MSG_FILE_TYPE="ファイル"
    MSG_DIR_TYPE="ディレクトリ"
    MSG_PREVIEW_HEADER="以下を docker-compose.yml に追加してください:"
    MSG_PREVIEW_VOLUMES="📄 volumes セクションに追加:"
    MSG_PREVIEW_TMPFS="📁 tmpfs セクションに追加:"
    MSG_PREVIEW_FOOTER="上記をコピーして docker-compose.yml に貼り付けてください"
    MSG_TARGET_FILES="対象ファイル:"
    MSG_COMPOSE_FOUND="検出された docker-compose.yml:"
else
    MSG_TITLE="🔧 Secret Config Sync Tool"
    MSG_CHECKING="Checking..."
    MSG_NO_SETTINGS="Claude settings file not found"
    MSG_NO_COMPOSE="docker-compose.yml not found (neither file exists)"
    MSG_ALL_SYNCED="✅ All secret files are synced. No additions needed."
    MSG_FOUND_HEADER="The following files are NOT configured in docker-compose.yml:"
    MSG_MISSING_FROM="Missing from:"
    MSG_PROMPT_ALL="Add all of these to docker-compose.yml?"
    MSG_YES_ALL="Add all"
    MSG_YES_EACH="Review each"
    MSG_NO="Don't add"
    MSG_PREVIEW="Preview (dry-run)"
    MSG_CONFIRM_FILE="Add this file?"
    MSG_ADDING="Adding:"
    MSG_ADDED="✅ Added"
    MSG_SKIPPED="⏭️  Skipped"
    MSG_DONE_HEADER="Done!"
    MSG_DONE_ADDED="Files added:"
    MSG_DONE_NONE="No files were added"
    MSG_REBUILD="Rebuild containers to apply changes:"
    MSG_REBUILD_CMD="  VS Code: Ctrl+Shift+P → 'Dev Containers: Rebuild Container'"
    MSG_REBUILD_CLI="  CLI: ./cli_sandbox/build.sh"
    MSG_NO_DENY="No file patterns in deny settings"
    MSG_NO_FILES="No matching files found"
    MSG_BACKUP="Backup created:"
    MSG_FILE_TYPE="File"
    MSG_DIR_TYPE="Directory"
    MSG_PREVIEW_HEADER="Add the following to docker-compose.yml:"
    MSG_PREVIEW_VOLUMES="📄 Add to volumes section:"
    MSG_PREVIEW_TMPFS="📁 Add to tmpfs section:"
    MSG_PREVIEW_FOOTER="Copy and paste the above into your docker-compose.yml"
    MSG_TARGET_FILES="Target files:"
    MSG_COMPOSE_FOUND="Detected docker-compose.yml:"
fi

# Collect existing compose files
# 存在する compose ファイルを収集
COMPOSE_FILES=()
COMPOSE_LABELS=()
if [ -f "$DEVCONTAINER_COMPOSE" ]; then
    COMPOSE_FILES+=("$DEVCONTAINER_COMPOSE")
    COMPOSE_LABELS+=("$LABEL_DC")
fi
if [ -f "$CLI_SANDBOX_COMPOSE" ]; then
    COMPOSE_FILES+=("$CLI_SANDBOX_COMPOSE")
    COMPOSE_LABELS+=("$LABEL_CLI")
fi

# Directories to ignore during file search
# ファイル検索時に無視するディレクトリ
IGNORE_PATTERNS=(
    "*/node_modules/*"
    "*/.git/*"
    "*/.sandbox/*"
)

# Build find ignore options
# find の除外オプションを構築
build_ignore_opts() {
    local opts=()
    for p in "${IGNORE_PATTERNS[@]}"; do
        opts+=("!" "-path" "$p")
    done
    echo "${opts[@]}"
}

# Extract Read() patterns from .claude/settings.json
# .claude/settings.json から Read() パターンを抽出
extract_deny_patterns() {
    local settings_file="$1"

    if [ ! -f "$settings_file" ]; then
        return
    fi

    jq -r '.permissions.deny[]' "$settings_file" 2>/dev/null | \
        grep -E '^Read\(' | \
        sed -E 's/^Read\(([^)]+)\)$/\1/' | \
        sort -u
}

# Find files matching a pattern
# パターンに一致するファイルを検索
find_matching_files() {
    local pattern="$1"
    local ignore_opts
    read -ra ignore_opts <<< "$(build_ignore_opts)"

    if [[ "$pattern" == **/* ]]; then
        local search_pattern="${pattern//\*\*\//*}"
        search_pattern="${search_pattern//\*\*/*}"

        if [[ "$pattern" == *"/**" ]]; then
            local dir_name="${pattern%/**}"
            dir_name="${dir_name##**/}"
            find "$WORKSPACE" -type d -name "$dir_name" "${ignore_opts[@]}" 2>/dev/null | while read -r dir; do
                find "$dir" -type f "${ignore_opts[@]}" 2>/dev/null
            done
        else
            local file_pattern="${pattern##**/}"
            find "$WORKSPACE" -name "$file_pattern" -type f "${ignore_opts[@]}" 2>/dev/null
        fi
    else
        local full_path="$WORKSPACE/$pattern"
        if [[ "$pattern" == *"*"* ]]; then
            # shellcheck disable=SC2086
            ls -1 $full_path 2>/dev/null || true
        elif [ -f "$full_path" ]; then
            echo "$full_path"
        elif [ -d "$full_path" ]; then
            find "$full_path" -type f "${ignore_opts[@]}" 2>/dev/null
        fi
    fi
}

# Check if a file is configured in docker-compose.yml
# ファイルが docker-compose.yml に設定されているかチェック
is_file_in_compose() {
    local file_path="$1"
    local compose_file="$2"

    if grep -qE "^\s*-\s*/dev/null:${file_path}(:ro)?$" "$compose_file" 2>/dev/null; then
        return 0
    fi

    local dir_path
    dir_path=$(dirname "$file_path")
    while [ "$dir_path" != "$WORKSPACE" ] && [ "$dir_path" != "/" ]; do
        if grep -qE "^\s*-\s*${dir_path}:ro$" "$compose_file" 2>/dev/null; then
            return 0
        fi
        dir_path=$(dirname "$dir_path")
    done

    return 1
}

# Add a file to docker-compose.yml volumes section
# ファイルを docker-compose.yml の volumes セクションに追加
add_file_to_compose() {
    local file_path="$1"
    local compose_file="$2"

    # Find the line number of the last /dev/null mount in volumes
    # volumes 内の最後の /dev/null マウントの行番号を見つける
    local last_devnull_line
    last_devnull_line=$(grep -n '/dev/null:' "$compose_file" | tail -1 | cut -d: -f1)

    if [ -n "$last_devnull_line" ]; then
        # Insert after the last /dev/null line
        # 最後の /dev/null 行の後に挿入
        local indent="      "  # Match existing indentation
        sed -i "${last_devnull_line}a\\${indent}- /dev/null:${file_path}:ro" "$compose_file"
    else
        # No /dev/null mounts found, find volumes section and add
        # /dev/null マウントがない場合、volumes セクションを見つけて追加
        echo "Warning: Could not find existing /dev/null mounts in $compose_file"
        echo "Please add manually: - /dev/null:${file_path}:ro"
        return 1
    fi
}

# Add a directory to docker-compose.yml tmpfs section
# ディレクトリを docker-compose.yml の tmpfs セクションに追加
add_dir_to_compose() {
    local dir_path="$1"
    local compose_file="$2"

    # Find the line number of the last tmpfs entry
    # tmpfs セクションの最後のエントリの行番号を見つける
    local in_tmpfs=false
    local last_tmpfs_line=0
    local line_num=0

    while IFS= read -r line; do
        ((line_num++))
        if [[ "$line" =~ ^[[:space:]]*tmpfs: ]]; then
            in_tmpfs=true
            continue
        fi
        if [[ "$in_tmpfs" == true && "$line" =~ ^[[:space:]]*-[[:space:]]*/workspace ]]; then
            last_tmpfs_line=$line_num
        fi
        if [[ "$in_tmpfs" == true && "$line" =~ ^[[:space:]]*[a-z_]+: && ! "$line" =~ ^[[:space:]]*- ]]; then
            in_tmpfs=false
        fi
    done < "$compose_file"

    if [ "$last_tmpfs_line" -gt 0 ]; then
        local indent="      "
        sed -i "${last_tmpfs_line}a\\${indent}- ${dir_path}:ro" "$compose_file"
    else
        echo "Warning: Could not find tmpfs section in $compose_file"
        echo "Please add manually under tmpfs: - ${dir_path}:ro"
        return 1
    fi
}

# Determine if path should be added as file or directory
# パスをファイルとして追加すべきかディレクトリとして追加すべきか判断
get_path_type() {
    local path="$1"
    if [ -d "$path" ]; then
        echo "dir"
    else
        echo "file"
    fi
}

# Get label for a compose file path
# compose ファイルパスのラベルを取得
get_compose_label() {
    local compose_file="$1"
    if [ "$compose_file" = "$DEVCONTAINER_COMPOSE" ]; then
        echo "$LABEL_DC"
    elif [ "$compose_file" = "$CLI_SANDBOX_COMPOSE" ]; then
        echo "$LABEL_CLI"
    else
        echo "$compose_file"
    fi
}

# Add a secret file to all compose files where it's missing
# 不足している全 compose ファイルに秘匿ファイルを追加
add_to_missing_composes() {
    local file="$1"
    local path_type
    path_type=$(get_path_type "$file")
    local success=false

    for compose_file in "${COMPOSE_FILES[@]}"; do
        if ! is_file_in_compose "$file" "$compose_file"; then
            local label
            label=$(get_compose_label "$compose_file")
            if [ "$path_type" = "dir" ]; then
                if add_dir_to_compose "$file" "$compose_file"; then
                    echo "   $MSG_ADDED ($label)"
                    success=true
                fi
            else
                if add_file_to_compose "$file" "$compose_file"; then
                    echo "   $MSG_ADDED ($label)"
                    success=true
                fi
            fi
        fi
    done

    if [ "$success" = true ]; then
        return 0
    else
        return 1
    fi
}

# Create backups for all compose files in .sandbox/backups/
# 全 compose ファイルのバックアップを .sandbox/backups/ に作成
create_backups() {
    echo ""
    for i in "${!COMPOSE_FILES[@]}"; do
        local compose_file="${COMPOSE_FILES[$i]}"
        local label
        label=$(get_compose_label "$compose_file")
        # Use lowercase label without spaces as backup prefix
        # スペースなしの小文字ラベルをバックアッププレフィックスに使用
        local backup_label
        backup_label=$(echo "$label" | tr '[:upper:] ' '[:lower:]_')
        local backup_path
        backup_path=$(backup_file "$compose_file" "$backup_label")
        echo "$MSG_BACKUP $label"
        echo "   $backup_path"
        cleanup_backups "${backup_label}.docker-compose.yml.*"
    done
    echo ""
}

# Main
# メイン処理
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "$MSG_TITLE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check prerequisites
# 前提条件の確認
if [ ! -f "$CLAUDE_SETTINGS" ]; then
    echo "$MSG_NO_SETTINGS: $CLAUDE_SETTINGS"
    exit 1
fi

if [ ${#COMPOSE_FILES[@]} -eq 0 ]; then
    echo "$MSG_NO_COMPOSE"
    exit 1
fi

# Show target compose files
# 対象の compose ファイルを表示
echo "$MSG_COMPOSE_FOUND"
for i in "${!COMPOSE_FILES[@]}"; do
    echo "   📄 ${COMPOSE_LABELS[$i]}: ${COMPOSE_FILES[$i]}"
done
echo ""

echo "$MSG_CHECKING"
echo ""

# Get deny patterns
# deny パターンを取得
patterns=$(extract_deny_patterns "$CLAUDE_SETTINGS")

if [ -z "$patterns" ]; then
    echo "$MSG_NO_DENY"
    exit 0
fi

# Find all files matching deny patterns
# deny パターンに一致するすべてのファイルを検索
all_matching_files=$(
    while IFS= read -r pattern; do
        [ -n "$pattern" ] && find_matching_files "$pattern"
    done <<< "$patterns" | sort -u
)

if [ -z "$all_matching_files" ]; then
    echo "$MSG_NO_FILES"
    exit 0
fi

# Check which files are NOT in any docker-compose.yml
# Also filter out files matching sync-ignore patterns
# いずれかの docker-compose.yml に設定されていないファイルを確認
# sync-ignore パターンにマッチするファイルも除外
missing_files=()
ignored_files=()
declare -A missing_labels  # file -> "DC, CLI" etc.

while IFS= read -r file; do
    [ -z "$file" ] && continue

    # Check if file matches sync-ignore patterns
    # sync-ignore パターンにマッチするかチェック
    if matches_sync_ignore "$file"; then
        ignored_files+=("$file")
        continue
    fi

    local_missing=()
    for i in "${!COMPOSE_FILES[@]}"; do
        if ! is_file_in_compose "$file" "${COMPOSE_FILES[$i]}"; then
            local_missing+=("${COMPOSE_LABELS[$i]}")
        fi
    done
    if [ ${#local_missing[@]} -gt 0 ]; then
        missing_files+=("$file")
        missing_labels["$file"]=$(IFS=", "; echo "${local_missing[*]}")
    fi
done <<< "$all_matching_files"

# Show info about ignored files
# 無視されたファイルの情報を表示
if [ ${#ignored_files[@]} -gt 0 ]; then
    echo "ℹ️  ${#ignored_files[@]} file(s) ignored (matched sync-ignore patterns)"
    echo "   無視されたファイル (sync-ignore パターンにマッチ): ${#ignored_files[@]} 件"
    echo ""
fi

# If all files are synced, exit
# すべて同期済みなら終了
if [ ${#missing_files[@]} -eq 0 ]; then
    echo "$MSG_ALL_SYNCED"
    exit 0
fi

# Show missing files
# 未設定ファイルを表示
echo "$MSG_FOUND_HEADER"
echo ""
for file in "${missing_files[@]}"; do
    rel_path="${file#$WORKSPACE/}"
    if [ -d "$file" ]; then
        type_label="[$MSG_DIR_TYPE]"
    else
        type_label="[$MSG_FILE_TYPE]"
    fi
    echo "   📄 $rel_path $type_label"
    echo "      $MSG_MISSING_FROM ${missing_labels[$file]}"
done
echo ""

# Prompt user
# ユーザーに確認
echo "───────────────────────────────────────────────────────────"
echo "$MSG_PROMPT_ALL"
echo ""
echo "  1) $MSG_YES_ALL"
echo "  2) $MSG_YES_EACH"
echo "  3) $MSG_NO"
echo "  4) $MSG_PREVIEW"
echo ""
read -rp "Select [1/2/3/4]: " choice

added_files=()

case "$choice" in
    1)
        # Add all files
        # すべて追加
        create_backups

        for file in "${missing_files[@]}"; do
            rel_path="${file#$WORKSPACE/}"
            echo "$MSG_ADDING $rel_path"
            if add_to_missing_composes "$file"; then
                added_files+=("$file")
            fi
        done
        ;;
    2)
        # Review each file
        # 個別確認
        create_backups

        for file in "${missing_files[@]}"; do
            rel_path="${file#$WORKSPACE/}"
            echo ""
            echo "📄 $rel_path"
            echo "   $MSG_MISSING_FROM ${missing_labels[$file]}"
            read -rp "   $MSG_CONFIRM_FILE [y/N]: " confirm
            if [[ "$confirm" =~ ^[Yy]$ ]]; then
                if add_to_missing_composes "$file"; then
                    added_files+=("$file")
                fi
            else
                echo "   $MSG_SKIPPED"
            fi
        done
        ;;
    3)
        # Don't add
        # 追加しない
        echo ""
        echo "$MSG_SKIPPED"
        exit 0
        ;;
    4)
        # Preview / Dry-run
        # プレビュー / ドライラン
        for i in "${!COMPOSE_FILES[@]}"; do
            local_volumes=()
            local_tmpfs=()

            for file in "${missing_files[@]}"; do
                if ! is_file_in_compose "$file" "${COMPOSE_FILES[$i]}"; then
                    if [ -d "$file" ]; then
                        local_tmpfs+=("$file")
                    else
                        local_volumes+=("$file")
                    fi
                fi
            done

            if [ ${#local_volumes[@]} -eq 0 ] && [ ${#local_tmpfs[@]} -eq 0 ]; then
                continue
            fi

            echo ""
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            echo "$MSG_PREVIEW_HEADER ${COMPOSE_LABELS[$i]}"
            echo "   ${COMPOSE_FILES[$i]}"
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            echo ""

            if [ ${#local_volumes[@]} -gt 0 ]; then
                echo "$MSG_PREVIEW_VOLUMES"
                echo ""
                for file in "${local_volumes[@]}"; do
                    echo "      - /dev/null:${file}:ro"
                done
                echo ""
            fi

            if [ ${#local_tmpfs[@]} -gt 0 ]; then
                echo "$MSG_PREVIEW_TMPFS"
                echo ""
                for dir in "${local_tmpfs[@]}"; do
                    echo "      - ${dir}:ro"
                done
                echo ""
            fi
        done

        echo "───────────────────────────────────────────────────────────"
        echo "$MSG_PREVIEW_FOOTER"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
        exit 0
        ;;
    *)
        # Invalid option - treat as don't add
        # 無効なオプション - 追加しないとして扱う
        echo ""
        echo "$MSG_SKIPPED"
        exit 0
        ;;
esac

# Summary
# サマリー
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "$MSG_DONE_HEADER"
echo ""

if [ ${#added_files[@]} -gt 0 ]; then
    echo "$MSG_DONE_ADDED"
    for file in "${added_files[@]}"; do
        rel_path="${file#$WORKSPACE/}"
        echo "   ✅ $rel_path"
    done
    echo ""
    echo "───────────────────────────────────────────────────────────"
    echo "$MSG_REBUILD"
    echo "$MSG_REBUILD_CMD"
    echo "$MSG_REBUILD_CLI"
else
    echo "$MSG_DONE_NONE"
fi
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

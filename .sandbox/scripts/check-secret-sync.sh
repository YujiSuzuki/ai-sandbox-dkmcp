#!/bin/bash
# check-secret-sync.sh
# Check if files blocked in AI settings are also hidden in docker-compose.yml
# AI設定でブロックされたファイルが docker-compose.yml でも隠蔽されているかチェック
#
# This script runs at DevContainer startup and warns if there are files that should
# be hidden from AI but are not configured in docker-compose.yml volume mounts.
# このスクリプトは DevContainer 起動時に実行され、AI から隠すべきファイルが
# docker-compose.yml のボリュームマウントに設定されていない場合に警告します。
#
# Supported AI settings files / 対応するAI設定ファイル:
#   - .claude/settings.json  (Claude Code)
#   - .aiexclude             (Gemini Code Assist)
#   - .geminiignore          (Gemini CLI)
#
# NOTE: .gitignore is intentionally NOT supported.
# 注意: .gitignore は意図的にサポートしていません。
#
# Reason / 理由:
#   .gitignore contains many non-secret patterns (node_modules/, dist/, *.log, etc.)
#   that would create noise in the sync check. AI exclusion files should explicitly
#   list only secrets, keeping the intent clear and maintenance easy.
#
#   .gitignore には秘匿情報以外のパターン（node_modules/, dist/, *.log 等）が
#   多く含まれ、同期チェックでノイズになります。AI除外ファイルには秘匿情報のみを
#   明示的に記載することで、意図が明確になりメンテナンスも容易になります。

set -e

WORKSPACE="${WORKSPACE:-/workspace}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common startup functions
# 共通起動関数を読み込み
# shellcheck source=/dev/null
source "${WORKSPACE}/.sandbox/scripts/_startup_common.sh"

# Determine which docker-compose.yml to use based on environment
# 環境に応じて使用する docker-compose.yml を決定
if [[ "$SANDBOX_ENV" == cli_* ]]; then
    COMPOSE_FILE="$WORKSPACE/cli_sandbox/docker-compose.yml"
else
    COMPOSE_FILE="$WORKSPACE/.devcontainer/docker-compose.yml"
fi

CLAUDE_SETTINGS="$WORKSPACE/.claude/settings.json"

# Language detection based on locale
# ロケールに基づく言語検出
if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
    MSG_TITLE="🔄 シークレット設定同期チェック"
    MSG_CHECKING="チェック中..."
    MSG_NO_SETTINGS="Claude 設定ファイルが見つかりません"
    MSG_NO_COMPOSE="docker-compose.yml が見つかりません"
    MSG_ALL_SYNCED="✅ すべての秘匿ファイルが docker-compose.yml に設定されています"
    MSG_MISSING_HEADER="⚠️  以下のファイルが docker-compose.yml に未設定です:"
    MSG_MISSING_FOOTER="これらのファイルはいずれかの AI設定でブロックされていますが、"
    MSG_MISSING_FOOTER2="docker-compose.yml のボリュームマウントに設定されていません。"
    MSG_MISSING_FOOTER3="DevContainer や CLI Sandbox 内では AI がこれらのファイルを読める可能性があります。"
    MSG_ACTION="対処方法:"
    MSG_ACTION1="  手動で docker-compose.yml を編集する（ホストOS側で）"
    MSG_ACTION2="  または: .sandbox/scripts/sync-secrets.sh を実行（シェル環境で）"
    MSG_ACTION3="  秘匿不要なら: .sandbox/config/sync-ignore にパターンを追加"
    MSG_NO_DENY="AI設定にファイルパターンがありません"
    MSG_NO_FILES="該当するファイルが見つかりませんでした"
else
    MSG_TITLE="🔄 Secret Config Sync Check"
    MSG_CHECKING="Checking..."
    MSG_NO_SETTINGS="Claude settings file not found"
    MSG_NO_COMPOSE="docker-compose.yml not found"
    MSG_ALL_SYNCED="✅ All secret files are configured in docker-compose.yml"
    MSG_MISSING_HEADER="⚠️  The following files are NOT configured in docker-compose.yml:"
    MSG_MISSING_FOOTER="These files are blocked in one or more AI settings but"
    MSG_MISSING_FOOTER2="not configured in docker-compose.yml volume mounts."
    MSG_MISSING_FOOTER3="AI may be able to read these files inside DevContainer or CLI Sandbox."
    MSG_ACTION="Action required:"
    MSG_ACTION1="  Manually edit docker-compose.yml (on host OS)"
    MSG_ACTION2="  Or run: .sandbox/scripts/sync-secrets.sh (in shell environment)"
    MSG_ACTION3="  If not secret: add pattern to .sandbox/config/sync-ignore"
    MSG_NO_DENY="No file patterns in AI settings"
    MSG_NO_FILES="No matching files found"
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
extract_claude_patterns() {
    local settings_file="$1"

    if [ ! -f "$settings_file" ]; then
        return
    fi

    # Extract Read() patterns and convert to search-friendly format
    # Read() パターンを抽出し、検索しやすい形式に変換
    jq -r '.permissions.deny[]' "$settings_file" 2>/dev/null | \
        grep -E '^Read\(' | \
        sed -E 's/^Read\(([^)]+)\)$/\1/' | \
        sort -u
}

# Extract patterns from .aiexclude files (Gemini Code Assist)
# .aiexclude ファイルからパターンを抽出（Gemini Code Assist用）
extract_aiexclude_patterns() {
    local aiexclude_file="$1"

    if [ ! -f "$aiexclude_file" ]; then
        return
    fi

    # gitignore-style patterns, filter out comments and empty lines
    # gitignore形式のパターン、コメントと空行を除外
    grep -v '^#' "$aiexclude_file" | grep -v '^[[:space:]]*$' | sort -u
}

# Find all Gemini ignore files in workspace (.aiexclude, .geminiignore)
# ワークスペース内のすべての Gemini 除外ファイルを検索
find_gemini_ignore_files() {
    find "$WORKSPACE" \( -name ".aiexclude" -o -name ".geminiignore" \) -type f \
        ! -path "*/node_modules/*" \
        ! -path "*/.git/*" \
        2>/dev/null
}

# Find files matching a pattern
# パターンに一致するファイルを検索
find_matching_files() {
    local pattern="$1"
    local ignore_opts
    read -ra ignore_opts <<< "$(build_ignore_opts)"

    # Convert glob pattern to find-compatible format
    # グロブパターンを find 互換形式に変換
    # **/ -> recursive, * -> single level

    if [[ "$pattern" == **/* ]]; then
        # Pattern like **/*.env or **/secrets/**
        # **/*.env や **/secrets/** のようなパターン
        local search_pattern="${pattern//\*\*\//*}"
        search_pattern="${search_pattern//\*\*/*}"

        if [[ "$pattern" == *"/**" ]]; then
            # Directory pattern like **/secrets/**
            # **/secrets/** のようなディレクトリパターン
            local dir_name="${pattern%/**}"
            dir_name="${dir_name##**/}"
            find "$WORKSPACE" -type d -name "$dir_name" "${ignore_opts[@]}" 2>/dev/null | while read -r dir; do
                find "$dir" -type f "${ignore_opts[@]}" 2>/dev/null
            done
        else
            # File pattern like **/*.env
            # **/*.env のようなファイルパターン
            local file_pattern="${pattern##**/}"
            find "$WORKSPACE" -name "$file_pattern" -type f "${ignore_opts[@]}" 2>/dev/null
        fi
    else
        # Specific path pattern like securenote-api/.env
        # securenote-api/.env のような具体的パスパターン
        local full_path="$WORKSPACE/$pattern"
        # Handle wildcards in specific paths
        # 具体的パスのワイルドカードを処理
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

    # Check /dev/null mounts
    # /dev/null マウントをチェック
    if grep -qE "^\s*-\s*/dev/null:${file_path}(:ro)?$" "$compose_file" 2>/dev/null; then
        return 0
    fi

    # Check tmpfs mounts (for directories)
    # tmpfs マウントをチェック（ディレクトリ用）
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

# Main
# メイン処理

# Check if settings file exists
# 設定ファイルの存在確認
if [ ! -f "$CLAUDE_SETTINGS" ]; then
    print_default "✓ Secret sync: $MSG_NO_SETTINGS"
    exit 0
fi

# Check if compose file exists
# compose ファイルの存在確認
if [ ! -f "$COMPOSE_FILE" ]; then
    print_default "✓ Secret sync: $MSG_NO_COMPOSE"
    exit 0
fi

# Get deny patterns from Claude settings
# Claude 設定から deny パターンを取得
claude_patterns=$(extract_claude_patterns "$CLAUDE_SETTINGS")

# Get patterns from all Gemini ignore files (.aiexclude, .geminiignore)
# すべての Gemini 除外ファイルからパターンを取得
gemini_patterns=""
while IFS= read -r ignore_file; do
    [ -z "$ignore_file" ] && continue
    file_patterns=$(extract_aiexclude_patterns "$ignore_file")
    if [ -n "$file_patterns" ]; then
        gemini_patterns="${gemini_patterns}${file_patterns}"$'\n'
    fi
done < <(find_gemini_ignore_files)

# Combine all patterns
# すべてのパターンを結合
patterns=$(echo -e "${claude_patterns}\n${gemini_patterns}" | grep -v '^$' | sort -u)

if [ -z "$patterns" ]; then
    print_default "✓ Secret sync: $MSG_NO_DENY"
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
    print_default "✓ Secret sync: $MSG_NO_FILES"
    exit 0
fi

# Check which files are NOT in docker-compose.yml
# Also filter out files matching sync-ignore patterns
# docker-compose.yml に設定されていないファイルを確認
# sync-ignore パターンにマッチするファイルも除外
missing_files=()
ignored_files=()
while IFS= read -r file; do
    [ -z "$file" ] && continue

    # Check if file matches sync-ignore patterns
    # sync-ignore パターンにマッチするかチェック
    if matches_sync_ignore "$file"; then
        ignored_files+=("$file")
        continue
    fi

    if ! is_file_in_compose "$file" "$COMPOSE_FILE"; then
        missing_files+=("$file")
    fi
done <<< "$all_matching_files"

# ============================================================
# Quiet mode: only show if missing files
# ============================================================
if is_quiet; then
    if [ ${#missing_files[@]} -gt 0 ]; then
        echo "⚠️  ${#missing_files[@]} files missing from docker-compose.yml"
        for file in "${missing_files[@]}"; do
            rel_path="${file#$WORKSPACE/}"
            echo "   📄 $rel_path"
        done
    fi
    exit 0
fi

# ============================================================
# Summary mode: problem explanation + action required
# ============================================================
if is_summary; then
    if [ ${#missing_files[@]} -gt 0 ]; then
        echo ""
        echo "$MSG_MISSING_HEADER"
        echo ""
        for file in "${missing_files[@]}"; do
            rel_path="${file#$WORKSPACE/}"
            echo "   📄 $rel_path"
        done
        echo ""
        echo "$MSG_MISSING_FOOTER"
        echo "$MSG_MISSING_FOOTER2"
        echo "$MSG_MISSING_FOOTER3"
        echo ""
        echo "$MSG_ACTION"
        echo "$MSG_ACTION1"
        echo "$MSG_ACTION2"
        echo "$MSG_ACTION3"
        echo ""
    else
        total_checked=$(echo "$all_matching_files" | grep -c . || true)
        echo "✓ Secret sync: all configured (${total_checked} checked, ${#ignored_files[@]} ignored)"
    fi
    exit 0
fi

# ============================================================
# Verbose mode: full output
# ============================================================
print_title "$MSG_TITLE"

# Report results
# 結果を報告
if [ ${#missing_files[@]} -eq 0 ]; then
    echo "$MSG_ALL_SYNCED"
    if [ ${#ignored_files[@]} -gt 0 ]; then
        echo ""
        echo "Ignored files (matched sync-ignore patterns):"
        echo "無視されたファイル (sync-ignore パターンにマッチ):"
        for file in "${ignored_files[@]}"; do
            rel_path="${file#$WORKSPACE/}"
            echo "   📄 $rel_path"
        done
    fi
else
    echo "$MSG_MISSING_HEADER"
    echo ""
    for file in "${missing_files[@]}"; do
        # Show relative path from workspace
        # ワークスペースからの相対パスを表示
        rel_path="${file#$WORKSPACE/}"
        echo "   📄 $rel_path"
    done
    echo ""
    echo "$MSG_MISSING_FOOTER"
    echo "$MSG_MISSING_FOOTER2"
    echo "$MSG_MISSING_FOOTER3"
    echo ""
    echo "$MSG_ACTION"
    echo "$MSG_ACTION1"
    echo "$MSG_ACTION2"
    echo "$MSG_ACTION3"

    if [ ${#ignored_files[@]} -gt 0 ]; then
        echo ""
        echo "Ignored files (matched sync-ignore patterns):"
        echo "無視されたファイル (sync-ignore パターンにマッチ):"
        for file in "${ignored_files[@]}"; do
            rel_path="${file#$WORKSPACE/}"
            echo "   📄 $rel_path"
        done
    fi
fi
print_footer

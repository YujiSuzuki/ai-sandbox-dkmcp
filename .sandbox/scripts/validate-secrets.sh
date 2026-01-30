#!/bin/bash
# validate-secrets.sh
# Validate that secret files are properly hidden from AI
# シークレットファイルがAIから適切に隠蔽されているか検証
#
# This script automatically reads secret paths from docker-compose.yml
# and checks if they are actually inaccessible (empty, /dev/null mounted, or tmpfs mounted)
# このスクリプトは docker-compose.yml から秘匿パスを自動で読み込み、
# 実際にアクセス不可（空、/dev/nullマウント、tmpfsマウント）であることを確認します

set -e

# Check if running on host OS (not in container)
# ホストOSで実行されていないかチェック
if [[ -z "${SANDBOX_ENV:-}" ]] && [[ ! -f "/.dockerenv" ]]; then
    if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
        echo "❌ このスクリプトはホストOSでは実行できません。"
        echo ""
        echo "以下のいずれかの環境で実行してください："
        echo "  • DevContainer のターミナル"
        echo "  • cli_sandbox/ai_sandbox.sh"
    else
        echo "❌ This script cannot be run on the host OS."
        echo ""
        echo "Please run in one of these environments:"
        echo "  • DevContainer terminal"
        echo "  • cli_sandbox/ai_sandbox.sh"
    fi
    exit 1
fi

WORKSPACE="${WORKSPACE:-/workspace}"

# Source common startup functions
# 共通起動関数を読み込み
# shellcheck source=/dev/null
source "${WORKSPACE}/.sandbox/scripts/_startup_common.sh"
EXIT_CODE=0
ERRORS=()

# Determine which docker-compose.yml to use based on environment
# 環境に応じて使用する docker-compose.yml を決定
# cli_sandbox environments: cli_claude, cli_gemini, cli_ai_sandbox
if [[ "$SANDBOX_ENV" == cli_* ]]; then
    COMPOSE_FILE="$WORKSPACE/cli_sandbox/docker-compose.yml"
else
    COMPOSE_FILE="$WORKSPACE/.devcontainer/docker-compose.yml"
fi

# Language detection based on locale
# ロケールに基づく言語検出
if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
    MSG_TITLE="🔍 シークレット隠蔽検証"
    MSG_SOURCE="設定ファイル:"
    MSG_CHECKING="検証中..."
    MSG_OK="✅ 正常に隠蔽されています"
    MSG_ERROR="❌ エラー"
    MSG_FILE_READABLE="ファイルが読み取り可能です（内容あり）"
    MSG_DIR_NOT_EMPTY="ディレクトリが空ではありません"
    MSG_EMPTY_OK="空または存在しない（OK）"
    MSG_ALL_OK="すべてのシークレットが正常に隠蔽されています"
    MSG_HAS_ERRORS="エラーがあります - 対応が必要です"
    MSG_NO_SECRETS="秘匿設定が見つかりませんでした"
    MSG_FILES_SECTION="📄 ファイル（/dev/null マウント）"
    MSG_DIRS_SECTION="📁 ディレクトリ（tmpfs マウント）"
    MSG_CHECK_CONFIG="docker-compose.yml の volumes/tmpfs 設定を確認してください"
else
    MSG_TITLE="🔍 Secret Hiding Validation"
    MSG_SOURCE="Config file:"
    MSG_CHECKING="Checking..."
    MSG_OK="✅ Properly hidden"
    MSG_ERROR="❌ Error"
    MSG_FILE_READABLE="File is readable (has content)"
    MSG_DIR_NOT_EMPTY="Directory is not empty"
    MSG_EMPTY_OK="Empty or does not exist (OK)"
    MSG_ALL_OK="All secrets are properly hidden"
    MSG_HAS_ERRORS="Errors found - action required"
    MSG_NO_SECRETS="No secret hiding configuration found"
    MSG_FILES_SECTION="📄 Files (/dev/null mounts)"
    MSG_DIRS_SECTION="📁 Directories (tmpfs mounts)"
    MSG_CHECK_CONFIG="Check your docker-compose.yml volumes/tmpfs configuration"
fi

# Extract /dev/null volume mounts (secret files)
# Format in docker-compose.yml: - /dev/null:/workspace/path/.env:ro
# /dev/null マウントを抽出（秘匿ファイル）
extract_secret_files() {
    local file="$1"
    grep -E '^\s*-\s*/dev/null:' "$file" 2>/dev/null | \
        sed -E 's/^[[:space:]]*-[[:space:]]*//' | \
        sed -E 's|^/dev/null:||' | \
        sed -E 's/:ro$//' | \
        sort -u || true
}

# Extract tmpfs mounts for secrets (directories)
# Only /workspace paths with :ro are considered secrets
# Format in docker-compose.yml: - /workspace/path/secrets:ro
# tmpfs マウントを抽出（秘匿ディレクトリ）
# /workspace で始まり :ro で終わるもののみを秘匿とみなす
extract_secret_dirs() {
    local file="$1"
    local in_tmpfs=false

    while IFS= read -r line; do
        # Check if we're entering tmpfs section
        # tmpfs セクションに入るかチェック
        if [[ "$line" =~ ^[[:space:]]*tmpfs: ]]; then
            in_tmpfs=true
            continue
        fi

        # Check if we're leaving tmpfs section (new top-level key)
        # tmpfs セクションを抜けるかチェック（新しいトップレベルキー）
        if [[ "$in_tmpfs" == true && "$line" =~ ^[[:space:]]*[a-z_]+: && ! "$line" =~ ^[[:space:]]*- ]]; then
            in_tmpfs=false
            continue
        fi

        # If in tmpfs section, extract /workspace paths with :ro (read-only = secrets)
        # tmpfs セクション内で /workspace パスを :ro 付きで抽出（読み取り専用 = 秘匿）
        # Must start with /workspace and end with :ro
        # /workspace で始まり :ro で終わる必要がある
        if [[ "$in_tmpfs" == true && "$line" =~ ^[[:space:]]*-[[:space:]]*/workspace && "$line" =~ :ro($|[[:space:]]) ]]; then
            echo "$line" | sed -E 's/^[[:space:]]*-[[:space:]]*//' | sed -E 's/:ro$//'
        fi
    done < "$file" | sort -u
}

# Validate a file path (should be empty or non-existent)
# ファイルパスを検証（空または存在しないべき）
# Sets VALIDATED_COUNT and populates ERRORS array
validate_file() {
    local path="$1"

    if [ -f "$path" ]; then
        if [ -s "$path" ]; then
            # File has content - ERROR
            # ファイルに内容あり - エラー
            ERRORS+=("$path: $MSG_FILE_READABLE")
            EXIT_CODE=1
            if is_verbose; then
                echo "   $path"
                echo "      $MSG_ERROR: $MSG_FILE_READABLE"
            fi
        else
            # File is empty (likely /dev/null mount)
            # ファイルが空（おそらく /dev/null マウント）
            ((VALIDATED_COUNT++)) || true
            if is_verbose; then
                echo "   $path"
                echo "      $MSG_OK"
            fi
        fi
    else
        # File doesn't exist
        # ファイルが存在しない
        ((VALIDATED_COUNT++)) || true
        if is_verbose; then
            echo "   $path"
            echo "      $MSG_EMPTY_OK"
        fi
    fi
}

# Validate a directory path (should be empty or non-existent)
# ディレクトリパスを検証（空または存在しないべき）
# Sets VALIDATED_COUNT and populates ERRORS array
validate_dir() {
    local path="$1"

    if [ -d "$path" ]; then
        if [ -z "$(ls -A "$path" 2>/dev/null)" ]; then
            # Directory is empty (likely tmpfs mount)
            # ディレクトリが空（おそらく tmpfs マウント）
            ((VALIDATED_COUNT++)) || true
            if is_verbose; then
                echo "   $path"
                echo "      $MSG_OK"
            fi
        else
            # Directory has files - ERROR
            # ディレクトリにファイルあり - エラー
            local file_count
            file_count=$(ls -1 "$path" 2>/dev/null | wc -l)
            ERRORS+=("$path: $MSG_DIR_NOT_EMPTY ($file_count files)")
            EXIT_CODE=1
            if is_verbose; then
                echo "   $path"
                echo "      $MSG_ERROR: $MSG_DIR_NOT_EMPTY ($file_count files)"
            fi
        fi
    else
        # Directory doesn't exist
        # ディレクトリが存在しない
        ((VALIDATED_COUNT++)) || true
        if is_verbose; then
            echo "   $path"
            echo "      $MSG_EMPTY_OK"
        fi
    fi
}

# Main
# メイン処理

# Check if compose file exists
# compose ファイルの存在確認
if [ ! -f "$COMPOSE_FILE" ]; then
    print_error "$MSG_ERROR: $COMPOSE_FILE not found"
    exit 1
fi

# Extract secret paths from docker-compose.yml
# docker-compose.yml から秘匿パスを抽出
secret_files=$(extract_secret_files "$COMPOSE_FILE")
secret_dirs=$(extract_secret_dirs "$COMPOSE_FILE")

# Initialize counter
VALIDATED_COUNT=0

# Validate secret files
# 秘匿ファイルを検証
if [ -n "$secret_files" ]; then
    if is_verbose; then
        echo "$MSG_FILES_SECTION"
        echo ""
    fi
    while IFS= read -r path; do
        [ -n "$path" ] && validate_file "$path"
    done <<< "$secret_files"
    is_verbose && echo ""
fi

# Validate secret directories
# 秘匿ディレクトリを検証
if [ -n "$secret_dirs" ]; then
    if is_verbose; then
        echo "$MSG_DIRS_SECTION"
        echo ""
    fi
    while IFS= read -r path; do
        [ -n "$path" ] && validate_dir "$path"
    done <<< "$secret_dirs"
    is_verbose && echo ""
fi

# Count total secrets
total_secrets=0
[ -n "$secret_files" ] && total_secrets=$((total_secrets + $(echo "$secret_files" | grep -c . || true)))
[ -n "$secret_dirs" ] && total_secrets=$((total_secrets + $(echo "$secret_dirs" | grep -c . || true)))

# ============================================================
# Quiet mode: only show errors
# ============================================================
if is_quiet; then
    if [ ${#ERRORS[@]} -gt 0 ]; then
        echo "❌ $MSG_HAS_ERRORS"
        for err in "${ERRORS[@]}"; do
            echo "   $err"
        done
    fi
    exit $EXIT_CODE
fi

# ============================================================
# Summary mode: show errors + action required
# ============================================================
if is_summary; then
    if [ ${#ERRORS[@]} -gt 0 ]; then
        echo ""
        echo "❌ $MSG_HAS_ERRORS (${#ERRORS[@]}/${total_secrets})"
        echo ""
        for err in "${ERRORS[@]}"; do
            echo "   ❌ $err"
        done
        echo ""
        echo "$MSG_CHECK_CONFIG"
        echo ""
    elif [ "$total_secrets" -eq 0 ]; then
        echo "✓ Secret hiding: $MSG_NO_SECRETS"
    else
        echo "✓ Secret hiding: ${VALIDATED_COUNT}/${total_secrets} validated"
    fi
    exit $EXIT_CODE
fi

# ============================================================
# Verbose mode: full output
# ============================================================
print_title "$MSG_TITLE"

echo "$MSG_SOURCE $COMPOSE_FILE"
echo ""

# Re-run validation with verbose output (already done above, so just show summary)
# Verbose output is already shown via validate_file/validate_dir functions

# No secrets configured
# 秘匿設定がない場合
if [ "$total_secrets" -eq 0 ]; then
    echo "$MSG_NO_SECRETS"
    echo ""
fi

# Summary (no mid-section separator)
# 結果サマリー（中間罫線なし）
if [ ${#ERRORS[@]} -gt 0 ]; then
    echo "$MSG_HAS_ERRORS"
    echo ""
    for err in "${ERRORS[@]}"; do
        echo "  ❌ $err"
    done
    echo ""
    echo "$MSG_CHECK_CONFIG"
else
    echo "$MSG_ALL_OK"
fi
print_footer

exit $EXIT_CODE

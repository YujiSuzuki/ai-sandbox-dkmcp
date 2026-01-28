#!/bin/bash
# compare-secret-config.sh
# Compare secret hiding configuration between DevContainer and CLI Sandbox
# DevContainer と CLI Sandbox の秘匿設定を比較
#
# This script checks if both docker-compose.yml files have the same
# secret hiding configuration (volumes with /dev/null and tmpfs mounts)
# 両方の docker-compose.yml で秘匿設定（/dev/null volumes と tmpfs マウント）が
# 同じであることを確認します

set -e

WORKSPACE="${WORKSPACE:-/workspace}"

# Source common startup functions
# 共通起動関数を読み込み
# shellcheck source=/dev/null
source "${WORKSPACE}/.sandbox/scripts/_startup_common.sh"
DEVCONTAINER_COMPOSE="$WORKSPACE/.devcontainer/docker-compose.yml"
CLI_SANDBOX_COMPOSE="$WORKSPACE/cli_sandbox/docker-compose.yml"

# Short display paths for mismatch messages
# 差異表示用の短いパス
DEVCONTAINER_COMPOSE_SHORT=".devcontainer/docker-compose.yml"
CLI_SANDBOX_COMPOSE_SHORT="cli_sandbox/docker-compose.yml"

# Language detection based on locale
# ロケールに基づく言語検出
if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
    MSG_TITLE="🔍 秘匿設定の整合性チェック"
    MSG_CHECKING="チェック中..."
    MSG_MATCH="✅ 両環境の秘匿設定は一致しています"
    MSG_MISMATCH="⚠️  秘匿設定に差異があります"
    MSG_DEVCONTAINER="DevContainer"
    MSG_CLI_SANDBOX="CLI Sandbox"
    MSG_VOLUMES="/dev/null マウント (volumes)"
    MSG_TMPFS="tmpfs マウント"
    MSG_ONLY_IN="のみに存在:"
    MSG_HINT="両方の docker-compose.yml を同期してください:"
    MSG_FILE_NOT_FOUND="ファイルが見つかりません:"
    MSG_ACTION="対処方法:"
    MSG_ACTION1="  手動で docker-compose.yml を編集する（ホストOS側で）"
    MSG_ACTION2="  または: .sandbox/scripts/sync-compose-secrets.sh を実行（この環境内で）"
else
    MSG_TITLE="🔍 Secret Config Consistency Check"
    MSG_CHECKING="Checking..."
    MSG_MATCH="✅ Secret hiding config matches in both environments"
    MSG_MISMATCH="⚠️  Secret hiding config mismatch detected"
    MSG_DEVCONTAINER="DevContainer"
    MSG_CLI_SANDBOX="CLI Sandbox"
    MSG_VOLUMES="/dev/null mounts (volumes)"
    MSG_TMPFS="tmpfs mounts"
    MSG_ONLY_IN="only in:"
    MSG_HINT="Please sync both docker-compose.yml files:"
    MSG_FILE_NOT_FOUND="File not found:"
    MSG_ACTION="How to fix:"
    MSG_ACTION1="  Manually edit docker-compose.yml (on host OS)"
    MSG_ACTION2="  Or run: .sandbox/scripts/sync-compose-secrets.sh (inside this environment)"
fi

# Check if files exist
# ファイルの存在確認
if [ ! -f "$DEVCONTAINER_COMPOSE" ]; then
    echo "$MSG_FILE_NOT_FOUND $DEVCONTAINER_COMPOSE"
    exit 1
fi

if [ ! -f "$CLI_SANDBOX_COMPOSE" ]; then
    echo "$MSG_FILE_NOT_FOUND $CLI_SANDBOX_COMPOSE"
    exit 1
fi

# Extract /dev/null volume mounts (secret hiding)
# Format: /dev/null:/path:ro
# /dev/null マウントを抽出（秘匿ファイル）
extract_devnull_mounts() {
    local file="$1"
    grep -E '^\s*-\s*/dev/null:' "$file" 2>/dev/null | \
        sed 's/^[[:space:]]*-[[:space:]]*//' | \
        sort || true
}

# Extract tmpfs mounts (secret directory hiding)
# Only /workspace paths with :ro are considered secrets
# tmpfs マウントを抽出（秘匿ディレクトリ）
# /workspace で始まり :ro で終わるもののみを秘匿とみなす
extract_tmpfs_mounts() {
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
            echo "$line" | sed -E 's/^[[:space:]]*-[[:space:]]*//'
        fi
    done < "$file" | sort -u
}

# Extract mounts from both files
# 両ファイルからマウント設定を抽出
devcontainer_volumes=$(extract_devnull_mounts "$DEVCONTAINER_COMPOSE")
cli_sandbox_volumes=$(extract_devnull_mounts "$CLI_SANDBOX_COMPOSE")
devcontainer_tmpfs=$(extract_tmpfs_mounts "$DEVCONTAINER_COMPOSE")
cli_sandbox_tmpfs=$(extract_tmpfs_mounts "$CLI_SANDBOX_COMPOSE")

# Check for mismatches
# 不一致をチェック
volumes_match=true
tmpfs_match=true

if [ "$devcontainer_volumes" != "$cli_sandbox_volumes" ]; then
    volumes_match=false
fi
if [ "$devcontainer_tmpfs" != "$cli_sandbox_tmpfs" ]; then
    tmpfs_match=false
fi

has_mismatch=false
if [ "$volumes_match" = false ] || [ "$tmpfs_match" = false ]; then
    has_mismatch=true
fi

# ============================================================
# Quiet mode: only show on mismatch
# ============================================================
if is_quiet; then
    if [ "$has_mismatch" = true ]; then
        echo "⚠️  $MSG_MISMATCH"
        [ "$volumes_match" = false ] && echo "   - $MSG_VOLUMES"
        [ "$tmpfs_match" = false ] && echo "   - $MSG_TMPFS"
        exit 1
    fi
    exit 0
fi

# ============================================================
# Default mode: show differences + action required
# ============================================================
if is_default; then
    if [ "$has_mismatch" = true ]; then
        echo ""
        echo "$MSG_MISMATCH"
        echo ""

        # Show volume differences
        # ボリューム差分を表示
        if [ "$volumes_match" = false ]; then
            echo "📁 $MSG_VOLUMES"
            only_in_devcontainer=$(comm -23 <(echo "$devcontainer_volumes") <(echo "$cli_sandbox_volumes") 2>/dev/null || true)
            only_in_cli=$(comm -13 <(echo "$devcontainer_volumes") <(echo "$cli_sandbox_volumes") 2>/dev/null || true)

            if [ -n "$only_in_devcontainer" ]; then
                echo "   $MSG_DEVCONTAINER $MSG_ONLY_IN ($DEVCONTAINER_COMPOSE_SHORT)"
                echo "$only_in_devcontainer" | while read -r line; do
                    [ -n "$line" ] && echo "      - $line"
                done
            fi
            if [ -n "$only_in_cli" ]; then
                echo "   $MSG_CLI_SANDBOX $MSG_ONLY_IN ($CLI_SANDBOX_COMPOSE_SHORT)"
                echo "$only_in_cli" | while read -r line; do
                    [ -n "$line" ] && echo "      - $line"
                done
            fi
            echo ""
        fi

        # Show tmpfs differences
        # tmpfs 差分を表示
        if [ "$tmpfs_match" = false ]; then
            echo "📁 $MSG_TMPFS"
            only_in_devcontainer=$(comm -23 <(echo "$devcontainer_tmpfs") <(echo "$cli_sandbox_tmpfs") 2>/dev/null || true)
            only_in_cli=$(comm -13 <(echo "$devcontainer_tmpfs") <(echo "$cli_sandbox_tmpfs") 2>/dev/null || true)

            if [ -n "$only_in_devcontainer" ]; then
                echo "   $MSG_DEVCONTAINER $MSG_ONLY_IN ($DEVCONTAINER_COMPOSE_SHORT)"
                echo "$only_in_devcontainer" | while read -r line; do
                    [ -n "$line" ] && echo "      - $line"
                done
            fi
            if [ -n "$only_in_cli" ]; then
                echo "   $MSG_CLI_SANDBOX $MSG_ONLY_IN ($CLI_SANDBOX_COMPOSE_SHORT)"
                echo "$only_in_cli" | while read -r line; do
                    [ -n "$line" ] && echo "      - $line"
                done
            fi
            echo ""
        fi

        echo "$MSG_ACTION"
        echo "$MSG_ACTION1"
        echo "$MSG_ACTION2"
        echo ""
        exit 1
    else
        echo "✓ $MSG_MATCH"
    fi
    exit 0
fi

# ============================================================
# Verbose mode: full output
# ============================================================
print_title "$MSG_TITLE"

# Compare /dev/null volumes
# /dev/null ボリュームを比較
echo "📁 $MSG_VOLUMES"

if [ "$volumes_match" = true ]; then
    echo "   ✅ Match"
else
    echo "   ⚠️  Mismatch"

    # Show differences
    # 差分を表示
    only_in_devcontainer=$(comm -23 <(echo "$devcontainer_volumes") <(echo "$cli_sandbox_volumes") 2>/dev/null || true)
    only_in_cli=$(comm -13 <(echo "$devcontainer_volumes") <(echo "$cli_sandbox_volumes") 2>/dev/null || true)

    if [ -n "$only_in_devcontainer" ]; then
        echo ""
        echo "   $MSG_DEVCONTAINER $MSG_ONLY_IN ($DEVCONTAINER_COMPOSE_SHORT)"
        echo "$only_in_devcontainer" | while read -r line; do
            [ -n "$line" ] && echo "      - $line"
        done
    fi

    if [ -n "$only_in_cli" ]; then
        echo ""
        echo "   $MSG_CLI_SANDBOX $MSG_ONLY_IN ($CLI_SANDBOX_COMPOSE_SHORT)"
        echo "$only_in_cli" | while read -r line; do
            [ -n "$line" ] && echo "      - $line"
        done
    fi
fi
echo ""

# Compare tmpfs mounts
# tmpfs マウントを比較
echo "📁 $MSG_TMPFS"

if [ "$tmpfs_match" = true ]; then
    echo "   ✅ Match"
else
    echo "   ⚠️  Mismatch"

    # Show differences
    # 差分を表示
    only_in_devcontainer=$(comm -23 <(echo "$devcontainer_tmpfs") <(echo "$cli_sandbox_tmpfs") 2>/dev/null || true)
    only_in_cli=$(comm -13 <(echo "$devcontainer_tmpfs") <(echo "$cli_sandbox_tmpfs") 2>/dev/null || true)

    if [ -n "$only_in_devcontainer" ]; then
        echo ""
        echo "   $MSG_DEVCONTAINER $MSG_ONLY_IN ($DEVCONTAINER_COMPOSE_SHORT)"
        echo "$only_in_devcontainer" | while read -r line; do
            [ -n "$line" ] && echo "      - $line"
        done
    fi

    if [ -n "$only_in_cli" ]; then
        echo ""
        echo "   $MSG_CLI_SANDBOX $MSG_ONLY_IN ($CLI_SANDBOX_COMPOSE_SHORT)"
        echo "$only_in_cli" | while read -r line; do
            [ -n "$line" ] && echo "      - $line"
        done
    fi
fi
echo ""

# Summary (no mid-section separator)
# 結果サマリー（中間罫線なし）
if [ "$has_mismatch" = true ]; then
    echo "$MSG_MISMATCH"
    echo ""
    echo "$MSG_HINT"
    echo "  📄 $DEVCONTAINER_COMPOSE"
    echo "  📄 $CLI_SANDBOX_COMPOSE"
    echo ""
    echo "$MSG_ACTION"
    echo "$MSG_ACTION1"
    echo "$MSG_ACTION2"
else
    echo "$MSG_MATCH"
fi
print_footer

# Return non-zero exit code if mismatch detected
# 差異がある場合は非ゼロの終了コードを返す
if [ "$has_mismatch" = true ]; then
    exit 1
fi
exit 0

#!/bin/bash
# help.sh
# Display description of all scripts in .sandbox/scripts/
# .sandbox/scripts/ 内の全スクリプトの説明を表示
#
# Usage: .sandbox/scripts/help.sh [--all]
#   --all: Include test scripts (default: utility scripts only)
#
# 使用法: .sandbox/scripts/help.sh [--all]
#   --all: テストスクリプトも表示（デフォルト: ユーティリティのみ）

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Language detection
if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
    LANG_JA=true
    MSG_TITLE="📚 .sandbox/scripts/ スクリプト一覧"
    MSG_UTILITY="ユーティリティスクリプト"
    MSG_TEST="テストスクリプト"
    MSG_ENV_CONTAINER="コンテナ内で実行"
    MSG_ENV_HOST="ホストOSで実行"
    MSG_ENV_BOTH="どちらでも実行可"
    MSG_SHOW_TESTS="テストスクリプトも表示するには: $0 --all"
else
    LANG_JA=false
    MSG_TITLE="📚 .sandbox/scripts/ Script List"
    MSG_UTILITY="Utility Scripts"
    MSG_TEST="Test Scripts"
    MSG_ENV_CONTAINER="Run in container"
    MSG_ENV_HOST="Run on host OS"
    MSG_ENV_BOTH="Run anywhere"
    MSG_SHOW_TESTS="To show test scripts: $0 --all"
fi

# Parse arguments
SHOW_ALL=false
if [[ "${1:-}" == "--all" ]]; then
    SHOW_ALL=true
fi

# Scripts that must run on host OS
HOST_ONLY_SCRIPTS="copy-credentials.sh"

# Scripts that must run in container
CONTAINER_ONLY_SCRIPTS="sync-secrets.sh validate-secrets.sh sync-compose-secrets.sh"

# Get environment indicator
get_env_indicator() {
    local script="$1"
    if [[ " $HOST_ONLY_SCRIPTS " == *" $script "* ]]; then
        echo "🖥️"
    elif [[ " $CONTAINER_ONLY_SCRIPTS " == *" $script "* ]]; then
        echo "🐳"
    else
        echo "  "
    fi
}

# Extract description from script header
get_description() {
    local script="$1"
    local desc_en desc_ja

    # Read lines 3-4 (after shebang and script name)
    desc_en=$(sed -n '3p' "$script" | sed 's/^# *//')
    desc_ja=$(sed -n '4p' "$script" | sed 's/^# *//')

    # Return appropriate language
    if [[ "$LANG_JA" == true ]] && [[ -n "$desc_ja" ]] && [[ "$desc_ja" != "#"* ]]; then
        echo "$desc_ja"
    else
        echo "$desc_en"
    fi
}

# Print header
echo ""
echo "$MSG_TITLE"
echo ""
echo "  🐳 = $MSG_ENV_CONTAINER"
echo "  🖥️  = $MSG_ENV_HOST"
echo ""

# Print utility scripts
echo "━━━ $MSG_UTILITY ━━━"
echo ""

for script in "$SCRIPT_DIR"/*.sh; do
    name=$(basename "$script")

    # Skip test scripts, help.sh itself, and _startup_common.sh
    [[ "$name" == test-* ]] && continue
    [[ "$name" == "help.sh" ]] && continue
    [[ "$name" == "_startup_common.sh" ]] && continue

    env_icon=$(get_env_indicator "$name")
    desc=$(get_description "$script")

    printf "  %s %-28s %s\n" "$env_icon" "$name" "$desc"
done

echo ""

# Print test scripts if requested
if [[ "$SHOW_ALL" == true ]]; then
    echo "━━━ $MSG_TEST ━━━"
    echo ""

    for script in "$SCRIPT_DIR"/test-*.sh; do
        [[ ! -f "$script" ]] && continue
        name=$(basename "$script")
        desc=$(get_description "$script")

        printf "     %-28s %s\n" "$name" "$desc"
    done
    echo ""
else
    echo "$MSG_SHOW_TESTS"
    echo ""
fi

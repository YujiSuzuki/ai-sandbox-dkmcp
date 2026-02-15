#!/bin/bash
# show-sponsor.sh
# Show GitHub Sponsors message at startup
#
# Usage:
#   show-sponsor.sh              # Show sponsor message
#   show-sponsor.sh --no-thanks  # Disable sponsor message
# ---
# 起動時に GitHub Sponsors メッセージを表示
#
# 使用法:
#   show-sponsor.sh              # スポンサーメッセージを表示
#   show-sponsor.sh --no-thanks  # スポンサーメッセージを無効化

set -e

WORKSPACE="${WORKSPACE:-/workspace}"

# Source common startup functions
# 共通起動関数を読み込み
# shellcheck source=/dev/null
source "${WORKSPACE}/.sandbox/scripts/_startup_common.sh"

# Sponsor URL
SPONSOR_URL="https://github.com/sponsors/YujiSuzuki"

# Handle --no-thanks flag
# --no-thanks フラグの処理
if [ "${1:-}" = "--no-thanks" ]; then
    if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
        echo "💡 スポンサーメッセージを無効にするには:"
        echo "   DevContainer: devcontainer.json の postStartCommand に --no-sponsor を追加"
        echo "     例: \"postStartCommand\": \"/workspace/.sandbox/scripts/startup.sh --no-sponsor\""
        echo "   CLI: cli_sandbox/_common.sh の startup.sh 呼び出しに --no-sponsor を追加"
    else
        echo "💡 To disable the sponsor message:"
        echo "   DevContainer: Add --no-sponsor to postStartCommand in devcontainer.json"
        echo "     e.g. \"postStartCommand\": \"/workspace/.sandbox/scripts/startup.sh --no-sponsor\""
        echo "   CLI: Add --no-sponsor to the startup.sh call in cli_sandbox/_common.sh"
    fi
    exit 0
fi

# Language detection based on locale
# ロケールに基づく言語検出
if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
    MSG_TITLE="💖 このプロジェクトを応援"
    MSG_BODY="AI Sandbox が役に立ったら、スポンサーになって応援してください！"
    MSG_HIDE="非表示にするには: .sandbox/scripts/show-sponsor.sh --no-thanks"
else
    MSG_TITLE="💖 Support this project"
    MSG_BODY="If you find AI Sandbox useful, consider sponsoring!"
    MSG_HIDE="To hide this message: .sandbox/scripts/show-sponsor.sh --no-thanks"
fi

# Show message based on verbosity
# 詳細度に応じてメッセージを表示

if is_quiet; then
    echo "💖 Sponsor: $SPONSOR_URL"
    exit 0
fi

print_title "$MSG_TITLE"
echo "  $MSG_BODY"
echo "  $SPONSOR_URL"
echo ""
echo "  $MSG_HIDE"
print_footer

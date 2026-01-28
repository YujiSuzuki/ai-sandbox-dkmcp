#!/bin/bash
# claude.sh - Run Claude Code in CLI sandbox
# Claude Code を CLI サンドボックスで実行

# Script configuration
# スクリプト設定
SCRIPT_NAME="claude.sh"
COMPOSE_PROJECT_NAME="cli-claude"
SANDBOX_ENV="cli_claude"

# Load common functions
# 共通関数を読み込み
source "$(dirname "$0")/_common.sh"

# Run startup scripts
# 起動時スクリプトを実行
run_startup_scripts || {
    confirm_continue_after_failure || exit 1
    # Validation failed: enter shell only (do NOT start Claude)
    # 検証失敗: シェルのみ起動（Claude は起動しない）
    run_in_container "$@"
    exit $?
}

# Check for warnings (only if startup succeeded but warnings were detected)
# 警告チェック（起動成功したが警告が検出された場合のみ）
if [ "$HAS_WARNINGS" = true ]; then
    confirm_continue_with_warnings || exit 0
fi

# Run Claude with environment message
# 環境メッセージ付きで Claude を実行
ENV_MSG=$(get_env_message "Claude")
run_in_container claude "$ENV_MSG" "$@"

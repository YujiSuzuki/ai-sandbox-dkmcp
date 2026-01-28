#!/bin/bash
# gemini.sh - Run Gemini CLI in CLI sandbox
# Gemini CLI を CLI サンドボックスで実行

# Script configuration
# スクリプト設定
SCRIPT_NAME="gemini.sh"
COMPOSE_PROJECT_NAME="cli-gemini"
SANDBOX_ENV="cli_gemini"

# Load common functions
# 共通関数を読み込み
source "$(dirname "$0")/_common.sh"

# Run startup scripts
# 起動時スクリプトを実行
run_startup_scripts || {
    confirm_continue_after_failure || exit 1
    # Validation failed: enter shell only (do NOT start Gemini)
    # 検証失敗: シェルのみ起動（Gemini は起動しない）
    run_in_container "$@"
    exit $?
}

# Check for warnings (only if startup succeeded but warnings were detected)
# 警告チェック（起動成功したが警告が検出された場合のみ）
if [ "$HAS_WARNINGS" = true ]; then
    confirm_continue_with_warnings || exit 0
fi

# Run Gemini
# Gemini を実行
run_in_container gemini "$@"

#!/bin/bash
# ai_sandbox.sh - Run interactive shell in CLI sandbox
# CLI サンドボックスで対話シェルを実行

# Script configuration
# スクリプト設定
SCRIPT_NAME="ai_sandbox.sh"
COMPOSE_PROJECT_NAME="cli-ai-sandbox"
SANDBOX_ENV="cli_ai_sandbox"

# Load common functions
# 共通関数を読み込み
source "$(dirname "$0")/_common.sh"

# Run startup scripts
# 起動時スクリプトを実行
run_startup_scripts || {
    confirm_continue_after_failure || exit 1
}

# Run interactive shell
# 対話シェルを実行
run_in_container "$@"

#!/bin/bash
# startup.sh
# Orchestrate all startup scripts for AI Sandbox
# ---
# AI Sandbox の起動スクリプトを統合管理

set -e  # Exit on error

# Import common functions from _startup_common.sh if available
if [[ -f "/workspace/.sandbox/scripts/_startup_common.sh" ]]; then
    source "/workspace/.sandbox/scripts/_startup_common.sh"
fi

# Language detection based on locale
# ロケールに基づく言語検出
if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
    MSG_TITLE="🚀 AI Sandbox 起動"
    MSG_MERGE_FAILED="⚠️  設定マージに失敗しましたが、続行します..."
    MSG_COMPARE_FAILED="⚠️  設定比較に失敗しましたが、続行します..."
    MSG_VALIDATE_FAILED="⚠️  秘匿検証に失敗しました"
    MSG_SYNC_CHECK_FAILED="⚠️  秘匿同期チェックに失敗しましたが、続行します..."
    MSG_REGISTERING="📦 SandboxMCP 登録"
    MSG_REGISTER_FAILED="⚠️  SandboxMCP 登録に失敗しましたが、続行します..."
    MSG_NO_GO="⚠️  Go がインストールされていないため、SandboxMCP 登録をスキップします"
    MSG_COMPLETE="✅ 起動完了"
else
    MSG_TITLE="🚀 AI Sandbox Startup"
    MSG_MERGE_FAILED="⚠️  Settings merge failed, but continuing..."
    MSG_COMPARE_FAILED="⚠️  Config comparison failed, but continuing..."
    MSG_VALIDATE_FAILED="⚠️  Secret validation failed"
    MSG_SYNC_CHECK_FAILED="⚠️  Secret sync check failed, but continuing..."
    MSG_REGISTERING="📦 Registering SandboxMCP"
    MSG_REGISTER_FAILED="⚠️  SandboxMCP registration failed, but continuing..."
    MSG_NO_GO="⚠️  Go not installed, skipping SandboxMCP registration"
    MSG_COMPLETE="✅ Startup complete"
fi

# Run startup scripts in order
# 起動スクリプトを順番に実行

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "$MSG_TITLE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 1. Merge Claude settings (low-failure, essential)
# Claude 設定のマージ（失敗しにくい、必須）
/workspace/.sandbox/scripts/merge-claude-settings.sh || {
    echo "$MSG_MERGE_FAILED"
    echo ""
}

# 2. Compare secret config consistency (report mismatches first)
# 秘匿設定の整合性チェック（不一致を先に報告）
/workspace/.sandbox/scripts/compare-secret-config.sh || {
    echo "$MSG_COMPARE_FAILED"
    echo ""
}

# 3. Validate secrets (critical check)
# 秘匿検証（重要チェック）
/workspace/.sandbox/scripts/validate-secrets.sh || {
    echo "$MSG_VALIDATE_FAILED"
    echo ""
}

# 4. Check secret sync (warning only)
# 秘匿同期チェック（警告のみ）
/workspace/.sandbox/scripts/check-secret-sync.sh || {
    echo "$MSG_SYNC_CHECK_FAILED"
    echo ""
}

# 5. Check for upstream updates (informational only)
# 上流更新チェック（情報提供のみ）
/workspace/.sandbox/scripts/check-upstream-updates.sh || true

# 6. Register SandboxMCP (if Go is available)
# SandboxMCP 登録（Go がある場合）
if command -v go >/dev/null 2>&1; then
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "$MSG_REGISTERING"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    make -C /workspace/.sandbox/sandbox-mcp register || {
        echo "$MSG_REGISTER_FAILED"
    }
else
    echo ""
    echo "$MSG_NO_GO"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "$MSG_COMPLETE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

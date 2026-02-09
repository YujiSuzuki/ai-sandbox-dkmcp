#!/bin/bash
# startup.sh
# Orchestrate all startup scripts for AI Sandbox
# AI Sandbox の起動スクリプトを統合管理

set -e  # Exit on error

# Import common functions from _startup_common.sh if available
if [[ -f "/workspace/.sandbox/scripts/_startup_common.sh" ]]; then
    source "/workspace/.sandbox/scripts/_startup_common.sh"
fi

# Run startup scripts in order
# 起動スクリプトを順番に実行

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🚀 AI Sandbox Startup"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 1. Merge Claude settings (low-failure, essential)
# Claude 設定のマージ（失敗しにくい、必須）
/workspace/.sandbox/scripts/merge-claude-settings.sh || {
    echo "⚠️  Settings merge failed, but continuing..."
    echo ""
}

# 2. Compare secret config consistency (report mismatches first)
# 秘匿設定の整合性チェック（不一致を先に報告）
/workspace/.sandbox/scripts/compare-secret-config.sh || {
    echo "⚠️  Config comparison failed, but continuing..."
    echo ""
}

# 3. Validate secrets (critical check)
# 秘匿検証（重要チェック）
/workspace/.sandbox/scripts/validate-secrets.sh || {
    echo "⚠️  Secret validation failed"
    echo ""
}

# 4. Check secret sync (warning only)
# 秘匿同期チェック（警告のみ）
/workspace/.sandbox/scripts/check-secret-sync.sh || {
    echo "⚠️  Secret sync check failed, but continuing..."
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
    echo "📦 Registering SandboxMCP"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    make -C /workspace/.sandbox/sandbox-mcp register || {
        echo "⚠️  SandboxMCP registration failed, but continuing..."
    }
else
    echo ""
    echo "⚠️  Go not installed, skipping SandboxMCP registration"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Startup complete"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

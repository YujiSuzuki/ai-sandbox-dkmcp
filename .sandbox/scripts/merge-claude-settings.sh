#!/bin/bash
# merge-claude-settings.sh
# Merge .claude/settings.json from subprojects into workspace root
#
# Merge logic (4 cases):
#   1) No workspace settings → Create by merging all subproject permissions
#   2) Settings exist, no changes → Re-merge from subprojects and update backup
#   3) Settings exist with manual changes → Disable auto-merge, preserve manual edits
#   4) Settings exist without backup → Assume manual creation, skip merge
# ---
# サブプロジェクトの .claude/settings.json を workspace 直下にマージ

set -e

WORKSPACE_ROOT="${WORKSPACE_ROOT:-/workspace}"

# Source common startup functions
# 共通起動関数を読み込み
# shellcheck source=/dev/null
source "${WORKSPACE_ROOT}/.sandbox/scripts/_startup_common.sh"
WORKSPACE_SETTINGS="$WORKSPACE_ROOT/.claude/settings.json"
BACKUP_FILE="$HOME/.claude-settings-backup.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Language detection based on locale
# ロケールに基づく言語検出
if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
    MSG_JQ_MISSING="警告: jq がインストールされていません。設定のマージをスキップします。"
    MSG_NO_SETTINGS="workspace の .claude/settings.json が見つかりません。"
    MSG_NO_PROJECT="マージするプロジェクト設定がありません。"
    MSG_CREATED="プロジェクトの permissions をマージして workspace 設定を作成しました。"
    MSG_BACKUP="  バックアップ保存先:"
    MSG_NO_BACKUP="workspace の .claude/settings.json がバックアップなしで存在します。"
    MSG_SKIP_MERGE="手動作成とみなし、マージをスキップします。"
    MSG_REMERGED="プロジェクトの permissions を再マージしました（手動変更なし）。"
    MSG_CHANGES_DETECTED="workspace の .claude/settings.json に手動変更が検出されました"
    MSG_FOUND_IN="以下のプロジェクトに設定があります："
    MSG_MERGE_MANUALLY="必要に応じて手動でマージしてください。"
    MSG_PRESERVED="手動変更を保護するため、自動マージを無効にしました。"
    MSG_REENABLE="自動マージを再開するには: %s を削除してコンテナを再起動してください。"
else
    MSG_JQ_MISSING="Warning: jq is not installed. Skipping settings merge."
    MSG_NO_SETTINGS="No workspace .claude/settings.json found."
    MSG_NO_PROJECT="No project settings to merge."
    MSG_CREATED="Created workspace settings by merging project permissions."
    MSG_BACKUP="  Backup saved to:"
    MSG_NO_BACKUP="Workspace .claude/settings.json exists without backup."
    MSG_SKIP_MERGE="Assuming manually created. Skipping merge."
    MSG_REMERGED="Re-merged project permissions (no manual changes detected)."
    MSG_CHANGES_DETECTED="Manual changes detected in workspace .claude/settings.json"
    MSG_FOUND_IN="Project settings found in:"
    MSG_MERGE_MANUALLY="Please merge manually if needed."
    MSG_PRESERVED="Your manual changes are preserved. Auto-merge has been disabled to avoid overwriting them."
    MSG_REENABLE="To re-enable auto-merge: delete %s and restart the container."
fi

# Check if jq is available
if ! command -v jq &> /dev/null; then
    print_warning "${MSG_JQ_MISSING}"
    exit 0
fi

# Find all .claude/settings.json in subprojects (max depth 2)
find_project_settings() {
    find "$WORKSPACE_ROOT" -maxdepth 3 -path "*/.claude/settings.json" -type f 2>/dev/null | \
        grep -v "^$WORKSPACE_ROOT/.claude/settings.json$" | sort || true
}

# Merge permissions from all project settings
merge_permissions() {
    local merged='{"permissions":{}}'
    local project_settings

    project_settings=$(find_project_settings)

    if [ -z "$project_settings" ]; then
        echo ""
        return
    fi

    while IFS= read -r settings_file; do
        if [ -f "$settings_file" ]; then
            # Extract permissions and merge (arrays are concatenated and deduplicated)
            local perms
            perms=$(jq -c '.permissions // {}' "$settings_file" 2>/dev/null || true)
            if [ -n "$perms" ] && [ "$perms" != "{}" ] && [ "$perms" != "null" ]; then
                merged=$(echo "$merged" | jq --argjson new "$perms" '
                    .permissions = {
                        deny: ((.permissions.deny // []) + ($new.deny // []) | unique),
                        allow: ((.permissions.allow // []) + ($new.allow // []) | unique)
                    } | with_entries(select(.value | length > 0))
                ')
            fi
        fi
    done <<< "$project_settings"

    echo "$merged"
}

# Check if workspace settings exists
if [ ! -f "$WORKSPACE_SETTINGS" ]; then
    # Case 1: No workspace settings - create by merging
    print_detail "${MSG_NO_SETTINGS}"

    merged=$(merge_permissions)

    if [ -z "$merged" ] || [ "$merged" = '{"permissions":{}}' ]; then
        print_detail "${MSG_NO_PROJECT}"
        exit 0
    fi

    # Create directory and save
    mkdir -p "$(dirname "$WORKSPACE_SETTINGS")"
    echo "$merged" | jq '.' > "$WORKSPACE_SETTINGS"

    # Create backup
    cp "$WORKSPACE_SETTINGS" "$BACKUP_FILE"

    # Count sources for summary
    source_count=$(find_project_settings | wc -l)
    if is_verbose; then
        echo -e "${GREEN}${MSG_CREATED}${NC}"
        echo -e "${MSG_BACKUP} $BACKUP_FILE"
    else
        print_default "✓ Claude settings: created (${source_count} sources)"
    fi
    exit 0
fi

# Workspace settings exists
if [ ! -f "$BACKUP_FILE" ]; then
    # Case 4: Settings exist but no backup - assume manually created
    if is_verbose; then
        echo -e "${YELLOW}${MSG_NO_BACKUP}${NC}"
        echo -e "${YELLOW}${MSG_SKIP_MERGE}${NC}"
    else
        print_default "✓ Claude settings: manual (skip merge)"
    fi
    exit 0
fi

# Both workspace settings and backup exist - check for changes
if diff -q "$WORKSPACE_SETTINGS" "$BACKUP_FILE" > /dev/null 2>&1; then
    # Case 2: No changes - re-merge and update backup
    merged=$(merge_permissions)

    if [ -z "$merged" ] || [ "$merged" = '{"permissions":{}}' ]; then
        exit 0
    fi

    echo "$merged" | jq '.' > "$WORKSPACE_SETTINGS"
    cp "$WORKSPACE_SETTINGS" "$BACKUP_FILE"

    # Count sources for summary
    source_count=$(find_project_settings | wc -l)
    if is_verbose; then
        echo -e "${GREEN}${MSG_REMERGED}${NC}"
    else
        print_default "✓ Claude settings: merged (${source_count} sources)"
    fi
else
    # Case 3: Changes detected - don't merge, prompt manual merge
    # Remove backup to disable auto-merge (protect manual changes)
    rm -f "$BACKUP_FILE"

    if is_verbose; then
        echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${YELLOW}${MSG_CHANGES_DETECTED}${NC}"
        echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
        echo ""
        echo -e "${MSG_FOUND_IN}"
        find_project_settings | while read -r f; do
            echo -e "  - $f"
        done
        echo ""
        echo -e "${YELLOW}${MSG_MERGE_MANUALLY}${NC}"
        echo ""
        echo -e "${MSG_PRESERVED}"
        # shellcheck disable=SC2059
        printf "${MSG_REENABLE}\n" "$WORKSPACE_SETTINGS"
    else
        # Quiet mode: minimal warning
        if is_quiet; then
            print_warning "Claude settings: ${MSG_CHANGES_DETECTED}"
            echo "   ${MSG_MERGE_MANUALLY}"
        else
            # Default mode: show details + action required
            echo ""
            print_warning "${MSG_CHANGES_DETECTED}"
            echo ""
            echo "${MSG_FOUND_IN}"
            find_project_settings | while read -r f; do
                echo "  - $f"
            done
            echo ""
            echo "${MSG_MERGE_MANUALLY}"
            echo ""
            echo "${MSG_PRESERVED}"
            # shellcheck disable=SC2059
            printf "${MSG_REENABLE}\n" "$WORKSPACE_SETTINGS"
            echo ""
        fi
    fi
fi

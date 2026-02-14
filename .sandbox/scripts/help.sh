#!/bin/bash
# help.sh
# Display one-line summary of all scripts in .sandbox/scripts/ (for shell users)
# For detailed information, see the header comments in each script or use SandboxMCP tools
#
# Usage: .sandbox/scripts/help.sh [--list]
#   --list: Show raw script list (for developers)
# ---
# .sandbox/scripts/ 内の全スクリプトの1行サマリーを表示（シェルユーザー向け）
# 詳細は各スクリプトの冒頭コメントまたは SandboxMCP ツールを参照
#
# 使用法: .sandbox/scripts/help.sh [--list]
#   --list: スクリプト一覧を表示（開発者向け）

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Language detection
if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
    LANG_JA=true
else
    LANG_JA=false
fi

# ─── Raw script list mode (--list) ───────────────────────────────

show_script_list() {
    local msg_title msg_utility msg_test msg_container msg_host msg_show_tests

    if [[ "$LANG_JA" == true ]]; then
        msg_title="📚 .sandbox/scripts/ スクリプト一覧"
        msg_utility="ユーティリティスクリプト"
        msg_test="テストスクリプト"
        msg_container="コンテナ内で実行"
        msg_host="ホストOSで実行"
    else
        msg_title="📚 .sandbox/scripts/ Script List"
        msg_utility="Utility Scripts"
        msg_test="Test Scripts"
        msg_container="Run in container"
        msg_host="Run on host OS"
    fi

    # Scripts that must run on host OS
    local host_only="init-host-env.sh"
    # Scripts that must run in container
    local container_only="sync-secrets.sh validate-secrets.sh sync-compose-secrets.sh"

    get_env_icon() {
        local s="$1"
        if [[ " $host_only " == *" $s "* ]]; then echo "🖥️"
        elif [[ " $container_only " == *" $s "* ]]; then echo "🐳"
        else echo "  "; fi
    }

    get_desc() {
        local script="$1"
        local desc_lines=()
        local line_num=0

        # Read script and parse description (first line only for --list view)
        while IFS= read -r line; do
            ((line_num++))

            # Skip shebang and filename lines
            [[ $line_num -le 2 ]] && continue

            # Stop at non-comment lines
            [[ ! "$line" =~ ^# ]] && break

            # Extract content after '#'
            local content="${line#\#}"
            content="${content# }"

            # Stop at # --- separator
            [[ "$content" =~ ^--- ]] && break

            # Collect first non-empty line only
            if [[ -n "$content" ]] && [[ ${#desc_lines[@]} -eq 0 ]]; then
                desc_lines+=("$content")
                break
            fi
        done < "$script"

        echo "${desc_lines[*]}"
    }

    echo ""
    echo "$msg_title"
    echo ""
    echo "  🐳 = $msg_container    🖥️  = $msg_host"
    echo ""
    echo "━━━ $msg_utility ━━━"
    echo ""

    for script in "$SCRIPT_DIR"/*.sh; do
        local name
        name=$(basename "$script")
        [[ "$name" == test-* ]] && continue
        [[ "$name" == "help.sh" ]] && continue
        [[ "$name" == "_startup_common.sh" ]] && continue

        printf "  %s %-32s %s\n" "$(get_env_icon "$name")" "$name" "$(get_desc "$script")"
    done

    echo ""
    echo "━━━ $msg_test ━━━"
    echo ""

    for script in "$SCRIPT_DIR"/test-*.sh; do
        [[ ! -f "$script" ]] && continue
        local name
        name=$(basename "$script")
        printf "     %-32s %s\n" "$name" "$(get_desc "$script")"
    done
    echo ""

    # Footer message
    if [[ "$LANG_JA" == true ]]; then
        echo "💡 詳細は各スクリプトの冒頭コメントを参照してください"
    else
        echo "💡 For detailed information, see the header comments in each script"
    fi
    echo ""
}

# ─── Default: workflow guide ─────────────────────────────────────

show_workflow_guide() {
    if [[ "$LANG_JA" == true ]]; then
        cat <<'GUIDE_JA'

🚀 AI Sandbox ヘルプ
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

■ はじめる
  DevContainer または CLI Sandbox を起動すれば準備完了。
  シークレットの隠蔽は自動で適用されます。

■ 起動時に自動実行（手動で実行する必要はありません）

  シークレットが正しく隠れているか確認:
    .sandbox/scripts/validate-secrets.sh

  AI 設定ファイルと docker-compose の同期チェック:
    .sandbox/scripts/check-secret-sync.sh

■ 必要に応じて手動実行（上記の結果に応じて実行を提案されます）

  同期のズレを対話的に修正:
    .sandbox/scripts/sync-secrets.sh

■ DockMCP（他コンテナとの連携）

  ホスト OS で DockMCP サーバーを起動:
    cd dkmcp && make install && dkmcp serve

  AI Sandbox 内から接続:
    claude mcp add --transport sse --scope user dkmcp http://host.docker.internal:8080/sse

  接続後は AI がログ確認・テスト実行などを自動で行います。

■ 困ったとき

  README を確認:
    README.md（英語） / README.ja.md（日本語）

  全スクリプトの一覧を見る:
    .sandbox/scripts/help.sh --list

GUIDE_JA
    else
        cat <<'GUIDE_EN'

🚀 AI Sandbox Help
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

■ Getting Started
  Open DevContainer or start CLI Sandbox. That's it.
  Secret hiding is applied automatically.

■ Auto-run on startup (no need to run manually)

  Verify secrets are properly hidden:
    .sandbox/scripts/validate-secrets.sh

  Check if AI config and docker-compose are in sync:
    .sandbox/scripts/check-secret-sync.sh

■ Run manually when needed (suggested based on results above)

  Interactively fix sync issues:
    .sandbox/scripts/sync-secrets.sh

■ DockMCP (Cross-Container Access)

  Start DockMCP server on host OS:
    cd dkmcp && make install && dkmcp serve

  Connect from AI Sandbox:
    claude mcp add --transport sse --scope user dkmcp http://host.docker.internal:8080/sse

  Once connected, AI can check logs, run tests, etc. automatically.

■ Need Help?

  See the docs:
    README.md (English) / README.ja.md (Japanese)

  Show all scripts:
    .sandbox/scripts/help.sh --list

GUIDE_EN
    fi
}

# ─── Main ────────────────────────────────────────────────────────

case "${1:-}" in
    --list)
        show_script_list
        ;;
    --help|-h)
        if [[ "$LANG_JA" == true ]]; then
            echo "使用法: .sandbox/scripts/help.sh [--list]"
            echo "  (引数なし)  ワークフローガイドを表示"
            echo "  --list      全スクリプトの一覧を表示"
        else
            echo "Usage: .sandbox/scripts/help.sh [--list]"
            echo "  (no args)   Show workflow guide"
            echo "  --list      Show all scripts"
        fi
        ;;
    *)
        show_workflow_guide
        ;;
esac

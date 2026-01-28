#!/bin/bash
# init-env-files.sh
# Auto-create env files from .example templates on first setup
# 初回セットアップ時に .example テンプレートから環境変数ファイルを自動生成
#
# Usage:
#   Automatic (startup):  init-env-files.sh [project_root]
#   Manual (interactive): init-env-files.sh -i [project_root]
#                         init-env-files.sh --interactive [project_root]
# 使用法:
#   自動（起動時）:       init-env-files.sh [project_root]
#   手動（対話式）:       init-env-files.sh -i [project_root]
#                         init-env-files.sh --interactive [project_root]
#
# This script is called from:
#   - cli_sandbox/_common.sh (CLI sandbox startup)
#   - .devcontainer/devcontainer.json initializeCommand (DevContainer startup)

set -euo pipefail

# Parse arguments / 引数のパース
INTERACTIVE=false
PROJECT_ROOT="."

while [[ $# -gt 0 ]]; do
    case "$1" in
        -i|--interactive)
            INTERACTIVE=true
            shift
            ;;
        -h|--help)
            echo "Usage: init-env-files.sh [-i|--interactive] [project_root]"
            echo "  -i, --interactive  Enable interactive mode (prompt for language)"
            echo "                     対話モードを有効化（言語を選択できます）"
            echo "  project_root       Project root directory (default: current directory)"
            echo "                     プロジェクトルート（デフォルト: カレントディレクトリ）"
            exit 0
            ;;
        *)
            PROJECT_ROOT="$1"
            shift
            ;;
    esac
done

created=0
SELECTED_LANG=""

# Interactive language selection / 対話式の言語選択
select_language() {
    echo ""
    echo "Select language / 言語を選択してください:"
    echo "  1) English (default)"
    echo "  2) 日本語"
    echo ""
    read -r -p "Enter 1 or 2 [1]: " choice
    case "$choice" in
        2)
            SELECTED_LANG="ja_JP.UTF-8"
            echo "→ 日本語 (ja_JP.UTF-8) を選択しました"
            ;;
        *)
            SELECTED_LANG="C.UTF-8"
            echo "→ Selected English (C.UTF-8)"
            ;;
    esac
    echo ""
}

# Apply language setting to .env.sandbox / .env.sandbox に言語設定を適用
apply_language_setting() {
    local env_file="$1"
    if [ -n "$SELECTED_LANG" ] && [ -f "$env_file" ]; then
        # Replace LANG= line with selected language
        if grep -q "^LANG=" "$env_file"; then
            # Use temp file for portability (BSD sed vs GNU sed)
            local tmp_file
            tmp_file=$(mktemp)
            sed "s/^LANG=.*/LANG=$SELECTED_LANG/" "$env_file" > "$tmp_file"
            mv "$tmp_file" "$env_file"
        else
            echo "LANG=$SELECTED_LANG" >> "$env_file"
        fi
    fi
}

# In interactive mode, prompt for language selection
# 対話モードの場合、言語選択を行う
if [ "$INTERACTIVE" = true ]; then
    select_language
fi

# --- .env.sandbox ---
env_sandbox_created=false
if [ ! -f "$PROJECT_ROOT/.env.sandbox" ]; then
    if [ -f "$PROJECT_ROOT/.env.sandbox.example" ]; then
        cp "$PROJECT_ROOT/.env.sandbox.example" "$PROJECT_ROOT/.env.sandbox"
        echo "Created .env.sandbox from .env.sandbox.example (first-time setup)"
        echo "  .env.sandbox.example から .env.sandbox を作成しました（初回セットアップ）"
        created=$((created + 1))
        env_sandbox_created=true
    else
        touch "$PROJECT_ROOT/.env.sandbox"
        echo "Created empty .env.sandbox (.env.sandbox.example not found)"
        echo "  .env.sandbox.example が見つからないため、空ファイルを作成しました"
        created=$((created + 1))
        env_sandbox_created=true
    fi
elif [ "$INTERACTIVE" = true ]; then
    echo ".env.sandbox already exists. / .env.sandbox は既に存在します。"
    read -r -p "Update language setting? / 言語設定を更新しますか? [y/N]: " update_lang
    if [[ "$update_lang" =~ ^[Yy] ]]; then
        env_sandbox_created=true
    fi
fi

# Apply language setting if selected / 言語設定を適用
if [ "$env_sandbox_created" = true ] && [ -n "$SELECTED_LANG" ]; then
    apply_language_setting "$PROJECT_ROOT/.env.sandbox"
    echo "  Language set to: $SELECTED_LANG"
    echo "  言語を設定しました: $SELECTED_LANG"
fi
if [ "$env_sandbox_created" = true ]; then
    echo "  Edit .env.sandbox to customize. / 設定変更は .env.sandbox を編集してください。"
    echo ""
fi

# --- cli_sandbox/.env ---
if [ -d "$PROJECT_ROOT/cli_sandbox" ] && [ ! -f "$PROJECT_ROOT/cli_sandbox/.env" ]; then
    if [ -f "$PROJECT_ROOT/cli_sandbox/.env.example" ]; then
        cp "$PROJECT_ROOT/cli_sandbox/.env.example" "$PROJECT_ROOT/cli_sandbox/.env"
        echo "Created cli_sandbox/.env from cli_sandbox/.env.example (first-time setup)"
        echo "  cli_sandbox/.env.example から cli_sandbox/.env を作成しました（初回セットアップ）"
        echo "  Edit cli_sandbox/.env to customize. / 設定変更は cli_sandbox/.env を編集してください。"
        echo ""
        created=$((created + 1))
    else
        touch "$PROJECT_ROOT/cli_sandbox/.env"
        echo "Created empty cli_sandbox/.env (cli_sandbox/.env.example not found)"
        echo "  cli_sandbox/.env.example が見つからないため、空ファイルを作成しました"
        echo ""
        created=$((created + 1))
    fi
fi

if [ "$created" -gt 0 ]; then
    echo "--- $created env file(s) initialized. These files are git-ignored. ---"
    echo "--- $created 個の環境変数ファイルを初期化しました。これらのファイルは git 管理対象外です。 ---"
    echo ""
fi

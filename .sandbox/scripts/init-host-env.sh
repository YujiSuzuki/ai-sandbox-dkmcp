#!/bin/bash
# init-host-env.sh
# Host-side initialization: create env files from templates and write host OS info
#
# Timezone behavior: Only offered when Japanese is selected (Asia/Tokyo default)
# Architecture conversion: x86_64→amd64, aarch64→arm64 for cross-build compatibility
#
# Usage:
#   Automatic (startup):  init-host-env.sh [project_root]
#   Manual (interactive): init-host-env.sh -i [project_root]
#                         init-host-env.sh --interactive [project_root]
#
# This script is called from:
#   - cli_sandbox/_common.sh (CLI sandbox startup)
#   - .devcontainer/devcontainer.json initializeCommand (DevContainer startup)
#
# Writes host OS info to .sandbox/.host-os for cross-build support (used by dkmcp/Makefile build-host)
# ---
# ホスト側の初期化: テンプレートからenvファイル作成、ホストOS情報の書き出し
#
# 使用法:
#   自動（起動時）:       init-host-env.sh [project_root]
#   手動（対話式）:       init-host-env.sh -i [project_root]
#                         init-host-env.sh --interactive [project_root]
#
# クロスビルド用にホストOS情報を .sandbox/.host-os に書き出す

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
            echo "Usage: init-host-env.sh [-i|--interactive] [project_root]"
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
SELECTED_TZ=""

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
            echo ""
            # Prompt for timezone when Japanese is selected
            # 日本語選択時にタイムゾーンを確認
            select_timezone_for_japanese
            ;;
        *)
            SELECTED_LANG="C.UTF-8"
            echo "→ Selected English (C.UTF-8)"
            ;;
    esac
    echo ""
}

# Interactive timezone selection for Japanese users / 日本語ユーザー向けタイムゾーン選択
select_timezone_for_japanese() {
    echo "タイムゾーンを Asia/Tokyo に設定しますか?"
    echo "  1) はい (default)"
    echo "  2) いいえ"
    echo ""
    read -r -p "1 または 2 を入力 [1]: " tz_choice
    case "$tz_choice" in
        2)
            echo "→ タイムゾーンは変更しません"
            ;;
        *)
            SELECTED_TZ="Asia/Tokyo"
            echo "→ TZ=Asia/Tokyo を設定します"
            ;;
    esac
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

# Apply timezone setting to .env.sandbox / .env.sandbox にタイムゾーン設定を適用
apply_timezone_setting() {
    local env_file="$1"
    if [ -n "$SELECTED_TZ" ] && [ -f "$env_file" ]; then
        if grep -q "^# *TZ=" "$env_file"; then
            # Uncomment and set TZ line / コメントアウトされた TZ 行を有効化
            local tmp_file
            tmp_file=$(mktemp)
            sed "s|^# *TZ=.*|TZ=$SELECTED_TZ|" "$env_file" > "$tmp_file"
            mv "$tmp_file" "$env_file"
        elif grep -q "^TZ=" "$env_file"; then
            # Replace existing TZ line / 既存の TZ 行を置換
            local tmp_file
            tmp_file=$(mktemp)
            sed "s|^TZ=.*|TZ=$SELECTED_TZ|" "$env_file" > "$tmp_file"
            mv "$tmp_file" "$env_file"
        else
            # Append TZ line / TZ 行を追加
            echo "TZ=$SELECTED_TZ" >> "$env_file"
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
# Apply timezone setting if selected / タイムゾーン設定を適用
if [ "$env_sandbox_created" = true ] && [ -n "$SELECTED_TZ" ]; then
    apply_timezone_setting "$PROJECT_ROOT/.env.sandbox"
    echo "  Timezone set to: $SELECTED_TZ"
    echo "  タイムゾーンを設定しました: $SELECTED_TZ"
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

# Write host OS info for cross-build (used by dkmcp/Makefile build-host)
# クロスビルド用にホストOS情報を書き出し（dkmcp/Makefile build-host で使用）
HOST_OS_FILE="$PROJECT_ROOT/.sandbox/.host-os"
mkdir -p "$(dirname "$HOST_OS_FILE")"
uname -s | tr '[:upper:]' '[:lower:]' > "$HOST_OS_FILE"
uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/' >> "$HOST_OS_FILE"

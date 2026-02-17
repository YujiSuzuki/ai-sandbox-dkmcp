#!/bin/bash
# install-commands.sh
# Install custom slash commands from .sandbox/commands/ into .claude/commands/
#
# Lists available custom commands and installs selected ones (or all) as
# Claude Code slash commands. Installed commands appear as /command-name.
#
# Usage:
#   .sandbox/scripts/install-commands.sh [options] [command-name...]
#
# Options:
#   --list         List available commands without installing
#   --all          Install all available commands
#   --uninstall    Remove installed commands (that originated from .sandbox/commands/)
#   --help, -h     Show this help
#
# Arguments:
#   command-name   Name(s) of commands to install (without .md extension)
#                  If omitted and --all not specified, shows interactive selection
#
# Examples:
#   .sandbox/scripts/install-commands.sh --list           # List available commands
#   .sandbox/scripts/install-commands.sh ais-local-review     # Install ais-local-review command
#   .sandbox/scripts/install-commands.sh --all            # Install all commands
#   .sandbox/scripts/install-commands.sh --uninstall      # Remove installed commands
#
# AI Workflow:
#   1. Run install-commands.sh --list to show available commands
#   2. Run install-commands.sh --all or install-commands.sh <name> to install
#   3. Restart Claude Code to recognize the new commands
# ---
# .sandbox/commands/ のカスタムスラッシュコマンドを .claude/commands/ にインストール
#
# 利用可能なカスタムコマンドを一覧表示し、選択したもの（または全て）を
# Claude Code のスラッシュコマンドとしてインストールします。
# インストール後は /command-name として使えます。
#
# 使用法:
#   .sandbox/scripts/install-commands.sh [options] [command-name...]
#
# オプション:
#   --list         インストールせずに利用可能なコマンドを一覧表示
#   --all          全コマンドをインストール
#   --uninstall    インストール済みコマンド（.sandbox/commands/ 由来）を削除
#   --help, -h     ヘルプ表示
#
# 引数:
#   command-name   インストールするコマンド名（.md 拡張子なし）
#                  省略かつ --all 未指定の場合、対話的に選択
#
# 例:
#   .sandbox/scripts/install-commands.sh --list           # 一覧表示
#   .sandbox/scripts/install-commands.sh ais-local-review     # ais-local-review をインストール
#   .sandbox/scripts/install-commands.sh --all            # 全コマンドをインストール
#   .sandbox/scripts/install-commands.sh --uninstall      # インストール済みを削除
#
# AI ワークフロー:
#   1. install-commands.sh --list で利用可能なコマンドを確認
#   2. install-commands.sh --all または install-commands.sh <name> でインストール
#   3. Claude Code を再起動して新しいコマンドを認識させる

set -euo pipefail

WORKSPACE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMMANDS_SRC_DIR="$WORKSPACE_ROOT/.sandbox/commands"
COMMANDS_DIR="$WORKSPACE_ROOT/.claude/commands"

# Language detection
if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
    LANG_JA=true
else
    LANG_JA=false
fi

msg() {
    local en="$1" ja="$2"
    if [[ "$LANG_JA" == true ]]; then
        echo "$ja"
    else
        echo "$en"
    fi
}

# ─── Help ─────────────────────────────────────────────────────────

show_help() {
    if [[ "$LANG_JA" == true ]]; then
        cat <<'HELP_JA'
使用法: install-commands.sh [options] [command-name...]

.sandbox/commands/ のカスタムコマンドを .claude/commands/ にインストールします。

オプション:
  --list         利用可能なコマンドを一覧表示
  --all          全コマンドをインストール
  --uninstall    インストール済みコマンドを削除
  --help, -h     このヘルプを表示

例:
  install-commands.sh --list           # 一覧表示
  install-commands.sh ais-local-review     # ais-local-review をインストール
  install-commands.sh --all            # 全コマンドをインストール
HELP_JA
    else
        cat <<'HELP_EN'
Usage: install-commands.sh [options] [command-name...]

Install custom commands from .sandbox/commands/ into .claude/commands/.

Options:
  --list         List available commands without installing
  --all          Install all available commands
  --uninstall    Remove installed commands
  --help, -h     Show this help

Examples:
  install-commands.sh --list           # List available commands
  install-commands.sh ais-local-review     # Install ais-local-review command
  install-commands.sh --all            # Install all commands
HELP_EN
    fi
}

# ─── List available commands ──────────────────────────────────────

list_commands() {
    if [[ ! -d "$COMMANDS_SRC_DIR" ]]; then
        msg "No commands directory found at $COMMANDS_SRC_DIR" \
            "コマンドディレクトリが見つかりません: $COMMANDS_SRC_DIR"
        exit 1
    fi

    local files=()
    while IFS= read -r -d '' f; do
        files+=("$f")
    done < <(find "$COMMANDS_SRC_DIR" -maxdepth 1 -name "*.md" -print0 2>/dev/null | sort -z)

    if [[ ${#files[@]} -eq 0 ]]; then
        msg "No commands available in $COMMANDS_SRC_DIR" \
            "利用可能なコマンドがありません: $COMMANDS_SRC_DIR"
        exit 0
    fi

    msg "Available commands:" "利用可能なコマンド:"
    echo ""

    for f in "${files[@]}"; do
        local name
        name="$(basename "$f" .md)"
        local desc
        desc="$(get_description "$f")"
        # Check if already installed
        local status=""
        if [[ -f "$COMMANDS_DIR/$name.md" ]]; then
            status=$(msg " [installed]" " [インストール済]")
        fi
        printf "  %-20s %s%s\n" "/$name" "$desc" "$status"
    done
    echo ""
    msg "Install with: .sandbox/scripts/install-commands.sh <command-name>" \
        "インストール: .sandbox/scripts/install-commands.sh <コマンド名>"
    msg "Install all:  .sandbox/scripts/install-commands.sh --all" \
        "全てインストール: .sandbox/scripts/install-commands.sh --all"
}

# ─── Front matter helpers ─────────────────────────────────────────

# Extract a field from YAML front matter
# Usage: extract_field "file.md" "description"
extract_field() {
    local file="$1" field="$2"
    if head -1 "$file" | grep -q '^---'; then
        sed -n "/^---$/,/^---$/{ /^${field}:/{ s/^${field}: *//; p; } }" "$file"
    fi
}

# Get description, preferring localized version if available
get_description() {
    local file="$1"
    if [[ "$LANG_JA" == true ]]; then
        local ja_desc
        ja_desc="$(extract_field "$file" "description-ja")"
        if [[ -n "$ja_desc" ]]; then
            echo "$ja_desc"
            return
        fi
    fi
    extract_field "$file" "description"
}

# Localize a command file for installation (writes to stdout)
# If description-ja exists and LANG_JA=true, replaces description and removes description-ja line
localize_file() {
    local file="$1"
    if [[ "$LANG_JA" == true ]]; then
        local ja_desc
        ja_desc="$(extract_field "$file" "description-ja")"
        if [[ -n "$ja_desc" ]]; then
            sed -e "s/^description: .*/description: ${ja_desc}/" -e '/^description-ja:/d' "$file"
            return
        fi
    fi
    # Remove description-ja line (not needed in installed file)
    sed '/^description-ja:/d' "$file"
}

# ─── Install commands ─────────────────────────────────────────────

install_command() {
    local name="$1"
    local src="$COMMANDS_SRC_DIR/$name.md"

    if [[ ! -f "$src" ]]; then
        msg "Command not found: $name (no file at $src)" \
            "コマンドが見つかりません: $name ($src が存在しません)"
        return 1
    fi

    mkdir -p "$COMMANDS_DIR"

    # Generate localized content for comparison and installation
    local translated
    translated="$(localize_file "$src")"

    if [[ -f "$COMMANDS_DIR/$name.md" ]]; then
        if echo "$translated" | diff -q - "$COMMANDS_DIR/$name.md" > /dev/null 2>&1; then
            msg "  $name: already up to date" \
                "  $name: 最新です"
            return 2
        else
            msg "  $name: updating (overwriting existing)" \
                "  $name: 更新（既存を上書き）"
        fi
    else
        msg "  $name: installing" \
            "  $name: インストール中"
    fi

    echo "$translated" > "$COMMANDS_DIR/$name.md"
    return 0
}

install_all() {
    local files=()
    while IFS= read -r -d '' f; do
        files+=("$f")
    done < <(find "$COMMANDS_SRC_DIR" -maxdepth 1 -name "*.md" -print0 2>/dev/null | sort -z)

    if [[ ${#files[@]} -eq 0 ]]; then
        msg "No commands available to install" \
            "インストール可能なコマンドがありません"
        exit 0
    fi

    local count=0
    for f in "${files[@]}"; do
        local name
        name="$(basename "$f" .md)"
        local rc=0
        install_command "$name" || rc=$?
        if [[ $rc -eq 0 ]]; then
            ((count++)) || true
        fi
    done

    echo ""
    msg "Installed $count command(s) to $COMMANDS_DIR" \
        "$count 個のコマンドを $COMMANDS_DIR にインストールしました"
    echo ""
    msg "Restart Claude Code to use the new commands." \
        "新しいコマンドを使うには Claude Code を再起動してください。"
}

# ─── Uninstall ────────────────────────────────────────────────────

uninstall_commands() {
    if [[ ! -d "$COMMANDS_DIR" ]]; then
        msg "No commands directory found" "コマンドディレクトリがありません"
        exit 0
    fi

    local count=0
    for f in "$COMMANDS_SRC_DIR"/*.md; do
        [[ -f "$f" ]] || continue
        local name
        name="$(basename "$f" .md)"
        if [[ -f "$COMMANDS_DIR/$name.md" ]]; then
            rm "$COMMANDS_DIR/$name.md"
            msg "  Removed: $name" "  削除: $name"
            ((count++)) || true
        fi
    done

    if [[ $count -eq 0 ]]; then
        msg "No installed commands to remove" \
            "削除するインストール済みコマンドがありません"
    else
        echo ""
        msg "Removed $count command(s)" "$count 個のコマンドを削除しました"
    fi

    # Clean up empty directory
    if [[ -d "$COMMANDS_DIR" ]] && [[ -z "$(ls -A "$COMMANDS_DIR" 2>/dev/null)" ]]; then
        rmdir "$COMMANDS_DIR"
    fi
}

# ─── Interactive selection ────────────────────────────────────────

interactive_select() {
    local files=()
    while IFS= read -r -d '' f; do
        files+=("$f")
    done < <(find "$COMMANDS_SRC_DIR" -maxdepth 1 -name "*.md" -print0 2>/dev/null | sort -z)

    if [[ ${#files[@]} -eq 0 ]]; then
        msg "No commands available to install" \
            "インストール可能なコマンドがありません"
        exit 0
    fi

    echo ""
    msg "Available commands:" "利用可能なコマンド:"
    echo ""

    local i=1
    local names=()
    for f in "${files[@]}"; do
        local name
        name="$(basename "$f" .md)"
        names+=("$name")
        local desc
        desc="$(get_description "$f")"
        local status=""
        if [[ -f "$COMMANDS_DIR/$name.md" ]]; then
            status=$(msg " [installed]" " [インストール済]")
        fi
        printf "  %d) %-20s %s%s\n" "$i" "/$name" "$desc" "$status"
        ((i++))
    done

    printf "  %d) %-20s\n" "$i" "$(msg 'All' '全て')"
    echo ""

    if [[ "$LANG_JA" == true ]]; then
        printf "インストールするコマンドを選択（番号）: "
    else
        printf "Select command to install (number): "
    fi
    read -r selection

    if [[ "$selection" == "$i" ]]; then
        install_all
    elif [[ "$selection" =~ ^[0-9]+$ ]] && (( selection >= 1 && selection < i )); then
        install_command "${names[$((selection-1))]}"
        echo ""
        msg "Restart Claude Code to use the new command." \
            "新しいコマンドを使うには Claude Code を再起動してください。"
    else
        msg "Invalid selection" "無効な選択です"
        exit 1
    fi
}

# ─── Main ─────────────────────────────────────────────────────────

main() {
    if [[ $# -eq 0 ]]; then
        interactive_select
        exit 0
    fi

    case "${1:-}" in
        --help|-h)
            show_help
            exit 0
            ;;
        --list)
            list_commands
            exit 0
            ;;
        --all)
            install_all
            exit 0
            ;;
        --uninstall)
            uninstall_commands
            exit 0
            ;;
        --*)
            msg "Unknown option: $1" "不明なオプション: $1"
            echo ""
            show_help
            exit 1
            ;;
        *)
            # Install specific commands by name
            local count=0
            local errors=0
            for name in "$@"; do
                name="${name%.md}"  # Strip .md if provided
                local rc=0
                install_command "$name" || rc=$?
                if [[ $rc -eq 0 ]]; then
                    ((count++)) || true
                elif [[ $rc -eq 1 ]]; then
                    ((errors++)) || true
                fi
                # rc=2 means "already up to date" — skip counting
            done
            echo ""
            if [[ $count -gt 0 ]]; then
                msg "Installed $count command(s). Restart Claude Code to use them." \
                    "$count 個のコマンドをインストールしました。使うには Claude Code を再起動してください。"
            fi
            if [[ $errors -gt 0 ]]; then
                exit 1
            fi
            ;;
    esac
}

main "$@"

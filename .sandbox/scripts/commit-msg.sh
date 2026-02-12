#!/bin/bash
# commit-msg.sh
# Generate commit message draft from staged changes for AI-assisted refinement, then commit
#
# Usage:
#   .sandbox/scripts/commit-msg.sh [options]
#
# Options:
#   --msg-file <file>  Use refined message file to commit
#   --log [n]          Show recent n commit messages for style reference (default: 10)
#   --style <style>    Subject style: "verb" (Add ...) or "cc" (feat: ...) (default: verb)
#   --amend            Amend the previous commit (use with --msg-file)
#   --help, -h         Show this help
#
# Environment:
#   COMMIT_MSG_STYLE   Default style ("verb" or "cc"). Overridden by --style flag.
#
# Examples:
#   .sandbox/scripts/commit-msg.sh                              # Generate draft
#   .sandbox/scripts/commit-msg.sh --style cc                   # Conventional Commits style
#   .sandbox/scripts/commit-msg.sh --log                        # Show recent commits
#   .sandbox/scripts/commit-msg.sh --msg-file CommitMsg-draft.md  # Commit
# ---
# ステージ済み変更からコミットメッセージのドラフトを生成し、AI と推敲してからコミットする
#
# 使用法:
#   .sandbox/scripts/commit-msg.sh [options]
#
# オプション:
#   --msg-file <file>  推敲済みメッセージファイルを指定してコミット
#   --log [n]          直近 n 件のコミットメッセージをスタイル参考用に表示（デフォルト: 10）
#   --style <style>    サブジェクトのスタイル: "verb" (Add ...) or "cc" (feat: ...) (デフォルト: verb)
#   --amend            直前のコミットを修正（--msg-file と併用）
#   --help, -h         ヘルプ表示
#
# 環境変数:
#   COMMIT_MSG_STYLE   デフォルトスタイル ("verb" or "cc")。--style フラグで上書き可能。

set -euo pipefail

# ─── Colors & helpers / カラー出力・ヘルパー関数 ────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

info()  { echo -e "${CYAN}ℹ️  $1${NC}"; }
ok()    { echo -e "${GREEN}✅ $1${NC}"; }
warn()  { echo -e "${YELLOW}⚠️  $1${NC}"; }
err()   { echo -e "${RED}❌ $1${NC}" >&2; }
die()   { err "$1"; exit 1; }

# ─── Argument parsing / 引数のパース ────────────────────────────

MSG_FILE=""
SHOW_LOG=false
LOG_COUNT=10
AMEND=false
STYLE="${COMMIT_MSG_STYLE:-verb}"  # "verb" or "cc"
DRAFT_FILE="CommitMsg-draft.md"

show_help() {
    cat <<'EOF'
Usage: .sandbox/scripts/commit-msg.sh [options]

Options:
  --msg-file <file>  Use refined message file to commit
  --log [n]          Show recent n commit messages for style reference (default: 10)
  --style <style>    Subject style: "verb" (Add ...) or "cc" (feat: ...)
  --amend            Amend the previous commit (use with --msg-file)
  --help, -h         Show this help

Environment:
  COMMIT_MSG_STYLE   Default style (default: verb). Overridden by --style.

Styles:
  verb  - Imperative verb start: "Add feature", "Fix bug", "Update docs"
  cc    - Conventional Commits: "feat: add feature", "fix: resolve bug"

Workflow:
  1. git add <files>                                          # Stage changes
  2. .sandbox/scripts/commit-msg.sh                           # Generate draft
  3. .sandbox/scripts/commit-msg.sh --log                     # Check style
  4. Refine CommitMsg-draft.md with AI                        # Collaborate
  5. .sandbox/scripts/commit-msg.sh --msg-file CommitMsg-draft.md  # Commit
EOF
    exit 0
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --msg-file)
            [[ -z "${2:-}" ]] && die "--msg-file requires a file path"
            MSG_FILE="$2"; shift 2 ;;
        --log)
            SHOW_LOG=true
            # Next arg is optional count (numeric)
            if [[ -n "${2:-}" ]] && [[ "$2" =~ ^[0-9]+$ ]]; then
                LOG_COUNT="$2"; shift
            fi
            shift ;;
        --style)
            [[ -z "${2:-}" ]] && die "--style requires 'verb' or 'cc'"
            STYLE="$2"
            [[ "$STYLE" != "verb" && "$STYLE" != "cc" ]] && die "Unknown style: $STYLE (use 'verb' or 'cc')"
            shift 2 ;;
        --amend)    AMEND=true; shift ;;
        --help|-h)  show_help ;;
        -*)         die "Unknown option: $1" ;;
        *)          die "Unexpected argument: $1" ;;
    esac
done

# ─── Show recent commits / 直近のコミット履歴表示 ───────────────

if [[ "$SHOW_LOG" == true ]]; then
    echo ""
    echo -e "${BOLD}📜 Recent ${LOG_COUNT} commits${NC}"
    echo "──────────────────────────────────────"
    echo ""

    # Show commits with full message (subject + body) for style reference
    # スタイル参考用にコミットメッセージの全文を表示
    git log -n "$LOG_COUNT" --format="  %C(dim)%h%C(reset) %s%n%w(0,4,4)%+b" 2>/dev/null || warn "No commits found."

    echo "──────────────────────────────────────"
    echo ""
    exit 0
fi

# ─── Pre-flight checks / 事前チェック ─────────────────────────

echo ""
echo -e "${BOLD}📝 Commit Message Draft${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check for staged changes / ステージ済み変更があるか確認
if [[ -z "$MSG_FILE" ]]; then
    # Draft mode: must have staged changes
    STAGED_COUNT=$(git diff --cached --name-only | wc -l)
    if [[ "$STAGED_COUNT" -eq 0 ]]; then
        die "No staged changes. Run 'git add <files>' first."
    fi
    ok "Staged files: ${STAGED_COUNT}"
    echo ""
fi

# Validate message file if specified / msg-file の検証
if [[ -n "$MSG_FILE" ]]; then
    if [[ ! -f "$MSG_FILE" ]]; then
        die "Message file not found: $MSG_FILE"
    fi
    if [[ ! -s "$MSG_FILE" ]]; then
        die "Message file is empty: $MSG_FILE"
    fi
fi

# ─── Analyze staged changes / ステージ済み変更の分析 ───────────

analyze_changes() {
    local files_added=0 files_modified=0 files_deleted=0 files_renamed=0
    local lines_added=0 lines_removed=0
    local file_list=()
    local ext_counts=""

    # Count file operations / ファイル操作のカウント
    while IFS=$'\t' read -r status file rest; do
        case "$status" in
            A)  files_added=$((files_added + 1))   ;;
            M)  files_modified=$((files_modified + 1)) ;;
            D)  files_deleted=$((files_deleted + 1))  ;;
            R*) files_renamed=$((files_renamed + 1))  ;;
        esac
        # Use the destination file for renames
        local target="${rest:-$file}"
        file_list+=("$target")
    done < <(git diff --cached --name-status)

    # Count line changes / 行数の変更をカウント
    while read -r added removed _file; do
        [[ "$added" == "-" ]] && continue  # binary
        lines_added=$((lines_added + added))
        lines_removed=$((lines_removed + removed))
    done < <(git diff --cached --numstat)

    # Detect file extensions / ファイル拡張子の集計
    ext_counts=$(printf '%s\n' "${file_list[@]}" | sed 's/.*\.//' | sort | uniq -c | sort -rn | head -5)

    # Output analysis as structured text / 分析結果を構造化テキストで出力
    echo "### Staged Changes Summary"
    echo ""
    echo "| Type | Count |"
    echo "|------|-------|"
    [[ $files_added -gt 0 ]]    && echo "| Added | $files_added |"
    [[ $files_modified -gt 0 ]] && echo "| Modified | $files_modified |"
    [[ $files_deleted -gt 0 ]]  && echo "| Deleted | $files_deleted |"
    [[ $files_renamed -gt 0 ]]  && echo "| Renamed | $files_renamed |"
    echo ""
    echo "**Lines:** +${lines_added} / -${lines_removed}"
    echo ""

    # File list / ファイル一覧
    echo "### Files"
    echo ""
    git diff --cached --name-status | while IFS=$'\t' read -r status file rest; do
        local icon
        case "$status" in
            A)  icon="+" ;;
            M)  icon="~" ;;
            D)  icon="-" ;;
            R*) icon="→" ;;
            *)  icon="?" ;;
        esac
        if [[ -n "${rest:-}" ]]; then
            echo "  ${icon} ${file} → ${rest}"
        else
            echo "  ${icon} ${file}"
        fi
    done
    echo ""

    # Top file types / 主要なファイルタイプ
    if [[ -n "$ext_counts" ]]; then
        echo "### Top File Types"
        echo ""
        echo "$ext_counts" | while read -r count ext; do
            echo "  ${count}x .${ext}"
        done
        echo ""
    fi
}

classify_changes() {
    # Classify the nature of changes from staged diff / ステージ済み差分から変更の性質を分類
    local status_list
    status_list=$(git diff --cached --name-status)
    local file_list
    file_list=$(git diff --cached --name-only)

    local categories=()

    # Check for documentation changes / ドキュメント変更の判定
    if echo "$file_list" | grep -qiE '(README|CLAUDE\.md|GEMINI\.md|\.md$|docs/)'; then
        categories+=("docs")
    fi

    # Check for test changes / テスト変更の判定
    if echo "$file_list" | grep -qiE '(_test\.go|\.test\.|test-|spec\.|__tests__)'; then
        categories+=("test")
    fi

    # Check for config changes / 設定変更の判定
    if echo "$file_list" | grep -qiE '(\.yaml$|\.yml$|\.json$|\.toml$|\.conf$|Makefile|Dockerfile|docker-compose)'; then
        categories+=("config")
    fi

    # Check for new files / 新規ファイルの判定
    if echo "$status_list" | grep -q '^A'; then
        categories+=("add")
    fi

    # Check for deletions / 削除の判定
    if echo "$status_list" | grep -q '^D'; then
        categories+=("remove")
    fi

    # Check for renames / リネームの判定
    if echo "$status_list" | grep -q '^R'; then
        categories+=("rename")
    fi

    # Check for bug fix indicators / バグ修正の手がかり
    local diff_content
    diff_content=$(git diff --cached --unified=0 2>/dev/null || echo "")
    if echo "$diff_content" | grep -qiE '(fix|bug|patch|hotfix|correct|resolve)'; then
        categories+=("fix")
    fi

    # Check for refactoring indicators / リファクタリングの手がかり
    if echo "$diff_content" | grep -qiE '(refactor|cleanup|reorganize|simplify|extract|inline)'; then
        categories+=("refactor")
    fi

    # Default: feature or update / デフォルト: 機能追加または更新
    if [[ ${#categories[@]} -eq 0 ]]; then
        categories+=("update")
    fi

    printf '%s\n' "${categories[@]}" | sort -u
}

generate_draft() {
    local categories
    categories=$(classify_changes)

    # Get common directory / 共通ディレクトリの推定
    local common_scope
    common_scope=$(git diff --cached --name-only | sed 's|/[^/]*$||' | sort | uniq -c | sort -rn | head -1 | awk '{print $2}')

    # Build bullet hints from file list / ファイル一覧から箇条書きヒントを生成
    local bullet_hints=""
    while IFS=$'\t' read -r status file rest; do
        local target="${rest:-$file}"
        case "$status" in
            A)  bullet_hints+="- Add ${target}"$'\n' ;;
            D)  bullet_hints+="- Remove ${target}"$'\n' ;;
            R*) bullet_hints+="- Rename ${file} to ${rest}"$'\n' ;;
            M)  bullet_hints+="- Update ${target}"$'\n' ;;
        esac
    done < <(git diff --cached --name-status)

    local subject_hint style_comment

    if [[ "$STYLE" == "cc" ]]; then
        # ─── Conventional Commits style / CC スタイル ───
        local prefix_suggestions=()
        while IFS= read -r cat; do
            case "$cat" in
                add)      prefix_suggestions+=("feat") ;;
                fix)      prefix_suggestions+=("fix") ;;
                docs)     prefix_suggestions+=("docs") ;;
                test)     prefix_suggestions+=("test") ;;
                refactor) prefix_suggestions+=("refactor") ;;
                config)   prefix_suggestions+=("chore") ;;
                remove)   prefix_suggestions+=("chore") ;;
                rename)   prefix_suggestions+=("refactor") ;;
                *)        prefix_suggestions+=("feat") ;;
            esac
        done <<< "$categories"

        local unique_prefixes
        unique_prefixes=$(printf '%s\n' "${prefix_suggestions[@]}" | awk '!seen[$0]++' | tr '\n' ', ' | sed 's/,$//')

        local primary_prefix="${prefix_suggestions[0]}"
        local scope_part=""
        if [[ -n "$common_scope" ]]; then
            local scope_name
            scope_name=$(basename "$common_scope")
            if [[ "$scope_name" != "$primary_prefix" ]]; then
                scope_part="(${scope_name})"
            fi
        fi

        subject_hint="${primary_prefix}${scope_part}: <describe change>"
        style_comment="<!-- Style: cc (Conventional Commits) | Prefixes: ${unique_prefixes} -->
<!-- Format: <type>(<scope>): <description>  (scope is optional) -->"
    else
        # ─── Verb style (default) / 動詞スタイル ───
        local verb_suggestions=()
        while IFS= read -r cat; do
            case "$cat" in
                add)      verb_suggestions+=("Add") ;;
                fix)      verb_suggestions+=("Fix") ;;
                docs)     verb_suggestions+=("Update" "Add") ;;
                test)     verb_suggestions+=("Add" "Fix") ;;
                refactor) verb_suggestions+=("Refactor" "Simplify") ;;
                config)   verb_suggestions+=("Update" "Configure") ;;
                remove)   verb_suggestions+=("Remove") ;;
                rename)   verb_suggestions+=("Rename") ;;
                *)        verb_suggestions+=("Update" "Improve") ;;
            esac
        done <<< "$categories"

        local unique_verbs
        unique_verbs=$(printf '%s\n' "${verb_suggestions[@]}" | awk '!seen[$0]++' | tr '\n' ', ' | sed 's/,$//')
        local primary_verb="${verb_suggestions[0]}"

        local file_count
        file_count=$(git diff --cached --name-only | wc -l | tr -d ' ')
        if [[ "$file_count" -eq 1 ]]; then
            local single_file basename_file
            single_file=$(git diff --cached --name-only)
            basename_file=$(basename "$single_file")
            subject_hint="${primary_verb} <describe change> in ${basename_file}"
        else
            subject_hint="${primary_verb} <describe change>${common_scope:+ in ${common_scope}}"
        fi

        style_comment="<!-- Style: verb (imperative) | Verbs: ${unique_verbs} -->"
    fi

    cat <<EOF
# Commit Message Draft

<!-- Generated by commit-msg.sh -->
<!-- Lines starting with # or <!-- are stripped when committing -->
${style_comment}
<!-- Scope hint: ${common_scope:-project root} -->
<!-- To commit: .sandbox/scripts/commit-msg.sh --msg-file ${DRAFT_FILE} -->

${subject_hint}

${bullet_hints}
<日本語サブジェクト>

<日本語の説明>
EOF
}

# ─── Draft mode (default) / ドラフトモード（デフォルト） ────────

if [[ -z "$MSG_FILE" ]]; then
    # Show change analysis / 変更分析の表示
    echo -e "${BOLD}📊 Change Analysis${NC}"
    echo "──────────────────────────────────────"
    echo ""
    analyze_changes
    echo "──────────────────────────────────────"
    echo ""

    # Show classification and style / 分類結果とスタイルの表示
    CATEGORIES=$(classify_changes)
    echo -e "${DIM}Detected categories: $(echo "$CATEGORIES" | tr '\n' ', ' | sed 's/,$//')${NC}"
    echo -e "${DIM}Style: ${STYLE}${NC}"
    echo ""

    # Show recent commits for style reference / スタイル参考の直近コミット
    echo -e "${BOLD}📜 Recent commits (for style reference)${NC}"
    echo "──────────────────────────────────────"
    echo ""
    git log -n 5 --format="  %C(dim)%h%C(reset) %s" 2>/dev/null || true
    echo ""
    echo "──────────────────────────────────────"
    echo ""

    # Generate and write draft / ドラフト生成・書き出し
    DRAFT=$(generate_draft)

    echo -e "${BOLD}📋 Draft${NC}"
    echo "──────────────────────────────────────"
    echo ""
    echo "$DRAFT"
    echo "──────────────────────────────────────"

    echo "$DRAFT" > "$DRAFT_FILE"

    echo ""
    ok "${DRAFT_FILE} を出力しました。"
    echo ""
    echo -e "  直近のコミットスタイルをもっと見たい場合:"
    echo -e "    ${CYAN}.sandbox/scripts/commit-msg.sh --log${NC}"
    echo ""
    echo -e "  AI と相談してコミットメッセージを推敲することもできます。"
    echo -e "  完了したら:"
    echo -e "    ${CYAN}.sandbox/scripts/commit-msg.sh --msg-file ${DRAFT_FILE}${NC}"
    echo ""
    exit 0
fi

# ─── Commit mode (--msg-file) / コミット実行モード ──────────────

# Parse the message file: strip markdown scaffolding, return clean commit message
# メッセージファイルを解析: マークダウンの足場を除去し、クリーンなコミットメッセージを返す
parse_message() {
    local file="$1"
    local result=""

    while IFS= read -r line; do
        # Skip HTML comments / HTMLコメントをスキップ
        [[ "$line" =~ ^\<\!-- ]] && continue

        # Skip markdown headers (# or ##) / マークダウンヘッダーをスキップ
        [[ "$line" =~ ^##?\  ]] && continue

        result+="${line}"$'\n'
    done < "$file"

    # Trim leading/trailing blank lines / 前後の空行を除去
    result=$(printf '%s\n' "$result" | awk '
        !started && /^[[:space:]]*$/ { next }
        { started=1; lines[++n]=$0 }
        /[^[:space:]]/ { last=n }
        END { for(i=1;i<=last;i++) print lines[i] }
    ')

    echo "$result"
}

COMMIT_MSG=$(parse_message "$MSG_FILE")

if [[ -z "$COMMIT_MSG" ]]; then
    die "Could not extract commit message from: $MSG_FILE"
fi

# Show the message / メッセージの表示
echo -e "${BOLD}📋 Commit Message${NC}"
echo "──────────────────────────────────────"
echo ""
echo "$COMMIT_MSG"
echo ""
echo "──────────────────────────────────────"

# Show staged files / ステージ済みファイルの表示
STAGED=$(git diff --cached --name-status)
if [[ -n "$STAGED" ]]; then
    echo ""
    echo -e "${DIM}Staged files:${NC}"
    echo "$STAGED" | while IFS=$'\t' read -r status file rest; do
        echo -e "  ${DIM}${status}  ${rest:+${file} → ${rest}}${rest:-${file}}${NC}"
    done
fi

# ─── Confirmation / 実行確認 ─────────────────────────────────────

echo ""
AMEND_LABEL=""
if [[ "$AMEND" == true ]]; then
    AMEND_LABEL=" (amend)"
fi
echo -ne "${YELLOW}Commit${AMEND_LABEL}? [y/N]: ${NC}"
read -r confirm
if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    info "Cancelled."
    exit 0
fi

# ─── Create commit / コミット実行 ────────────────────────────────

# Write parsed message to a temp file (strips markdown scaffolding)
# パース済みメッセージを一時ファイルに書き出し（マークダウンの足場を除去）
TEMP_MSG=$(mktemp)
trap 'rm -f "$TEMP_MSG"' EXIT
echo "$COMMIT_MSG" > "$TEMP_MSG"

COMMIT_ARGS=(-F "$TEMP_MSG")
if [[ "$AMEND" == true ]]; then
    COMMIT_ARGS+=(--amend)
fi

git commit "${COMMIT_ARGS[@]}"

echo ""
ok "Committed successfully!"
echo ""

# Show the result / 結果表示
git log -1 --format="  %C(dim)%h%C(reset) %s" 2>/dev/null || true
echo ""

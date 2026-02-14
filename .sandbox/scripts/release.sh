#!/bin/bash
# release.sh
# Generate release notes draft for AI-assisted refinement, then publish
#
# Usage:
#   .sandbox/scripts/release.sh <version> [options]
#
# Arguments:
#   <version>     Release version (e.g. v0.4.0). Must be semver with v prefix.
#
# Options:
#   --notes-file <file>  Use refined release notes file to create tag + GitHub Release
#   --prev               Show the latest GitHub Release notes for reference
#   --help, -h           Show this help
#
# Examples:
#   .sandbox/scripts/release.sh v0.4.0                              # Generate draft
#   .sandbox/scripts/release.sh --prev                               # Show previous release
#   .sandbox/scripts/release.sh v0.4.0 --notes-file notes.md        # Publish release
# ---
# リリースノートのドラフトを生成し、AI と推敲してからリリースする
#
# 使用法:
#   .sandbox/scripts/release.sh <version> [options]
#
# 引数:
#   <version>     リリースバージョン（例: v0.4.0）。v付き semver 形式。
#
# オプション:
#   --notes-file <file>  推敲済みリリースノートを指定してタグ + GitHub Release を作成
#   --prev               直近の GitHub Release のリリースノートを表示
#   --help, -h           ヘルプ表示

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

# ─── Language detection / 言語検出 ─────────────────────────────

if [[ "${LANG:-}" == ja_JP* ]] || [[ "${LC_ALL:-}" == ja_JP* ]]; then
    MSG_RELEASE_TITLE="🚀 リリース:"
    MSG_VERSION_FORMAT="バージョンは v付き semver 形式で指定してください（例: v0.4.0）。指定値:"
    MSG_NOT_MAIN="'main' ブランチで実行してください。現在のブランチ:"
    MSG_NOT_CLEAN="ワーキングツリーがクリーンではありません。先にコミットまたは stash してください。"
    MSG_NOTES_NOT_FOUND="ノートファイルが見つかりません:"
    MSG_NOTES_EMPTY="ノートファイルが空です:"
    MSG_TAG_EXISTS="タグ %s はすでに存在します。"
    MSG_NO_PREV_TAG="前回のタグが見つかりません。最初のタグは手動で作成してください。"
    MSG_PREFLIGHT="事前チェック通過"
    MSG_NO_COMMITS="前回のタグ %s 以降のコミットがありません。リリースするものがありません。"
    MSG_DRAFT_TITLE="📋 リリースノート ドラフト"
    MSG_WROTE="を出力しました。"
    MSG_NEXT_STEPS="次のステップ:"
    MSG_STEP1="1. 前回のリリースノートのトーンを確認:"
    MSG_STEP2="2. ドラフトをトーンに合わせて推敲"
    MSG_STEP3="3. 推敲が完了したらリリース実行:"
    MSG_NOTES_TITLE="📋 リリースノート"
    MSG_CONFIRM_TAG="タグ %s を作成して push しますか？"
    MSG_CANCELLED="キャンセルしました。"
    MSG_TAG_CREATED="タグ %s を作成しました"
    MSG_TAG_PUSHED="タグ %s を origin に push しました"
    MSG_GH_CREATED="GitHub Release を作成しました"
    MSG_GH_FAILED="gh release create に失敗しました。"
    MSG_GH_NOT_FOUND="gh CLI が見つかりません。"
    MSG_MANUAL_RELEASE="手動でリリースを作成してください:"
    MSG_PASTE_NOTES="リリースノートを貼り付けてください:"
    MSG_RELEASE_COMPLETE="リリース %s 完了！ 🎉"
    MSG_LATEST_RELEASE="📌 最新リリース:"
    MSG_NO_RELEASES="リリースが見つかりません。"
    MSG_VERSION_REQUIRED="バージョン引数が必要です。使用法: release.sh <version> [--notes-file <file>]"
    MSG_REQUIRES_GH="gh CLI または curl + jq が必要です。"
    MSG_NO_REPO="git remote から GitHub リポジトリを検出できません。"
else
    MSG_RELEASE_TITLE="🚀 Release:"
    MSG_VERSION_FORMAT="Version must be semver with v prefix (e.g. v0.4.0). Got:"
    MSG_NOT_MAIN="Must be on 'main' branch. Currently on:"
    MSG_NOT_CLEAN="Working tree is not clean. Commit or stash changes first."
    MSG_NOTES_NOT_FOUND="Notes file not found:"
    MSG_NOTES_EMPTY="Notes file is empty:"
    MSG_TAG_EXISTS="Tag %s already exists."
    MSG_NO_PREV_TAG="No previous tag found. Create the first tag manually."
    MSG_PREFLIGHT="Pre-flight checks passed"
    MSG_NO_COMMITS="No commits since %s. Nothing to release."
    MSG_DRAFT_TITLE="📋 Release Notes Draft"
    MSG_WROTE="written."
    MSG_NEXT_STEPS="Next steps:"
    MSG_STEP1="1. Check the previous release tone:"
    MSG_STEP2="2. Refine the draft to match the tone"
    MSG_STEP3="3. When refined, publish the release:"
    MSG_NOTES_TITLE="📋 Release Notes"
    MSG_CONFIRM_TAG="Create tag %s and push?"
    MSG_CANCELLED="Cancelled."
    MSG_TAG_CREATED="Tag %s created"
    MSG_TAG_PUSHED="Tag %s pushed to origin"
    MSG_GH_CREATED="GitHub Release created"
    MSG_GH_FAILED="gh release create failed."
    MSG_GH_NOT_FOUND="gh CLI not found."
    MSG_MANUAL_RELEASE="Create the release manually:"
    MSG_PASTE_NOTES="Paste the release notes from:"
    MSG_RELEASE_COMPLETE="Release %s complete! 🎉"
    MSG_LATEST_RELEASE="📌 Latest Release:"
    MSG_NO_RELEASES="No releases found."
    MSG_VERSION_REQUIRED="Version argument required. Usage: release.sh <version> [--notes-file <file>]"
    MSG_REQUIRES_GH="Requires gh CLI or curl + jq."
    MSG_NO_REPO="Could not detect GitHub repository from git remote."
fi

# ─── Argument parsing / 引数のパース ────────────────────────────

VERSION=""
NOTES_FILE=""
SHOW_PREV=false
DRAFT_FILE="ReleaseNotes-draft.md"

# Get GitHub API repo path from git remote / git remote から GitHub API 用のリポジトリパスを取得
get_github_repo() {
    local remote_url
    remote_url=$(git remote get-url origin 2>/dev/null || echo "")
    echo "$remote_url" | sed -E 's|.*github\.com[:/]||;s|\.git$||'
}

show_help() {
    cat <<'EOF'
Usage: .sandbox/scripts/release.sh <version> [options]

Arguments:
  <version>     Release version (e.g. v0.4.0)

Options:
  --notes-file <file>  Use refined release notes to create tag + GitHub Release
  --prev               Show the latest GitHub Release notes for reference
  --help, -h           Show this help

Workflow:
  1. release.sh v0.4.0                          # Generate draft
  2. release.sh --prev                          # Check previous release
  3. Refine ReleaseNotes-draft.md with AI       # Collaborate
  4. release.sh v0.4.0 --notes-file ReleaseNotes-draft.md  # Publish
EOF
    exit 0
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --notes-file)
            [[ -z "${2:-}" ]] && die "--notes-file requires a file path"
            NOTES_FILE="$2"; shift 2 ;;
        --prev)     SHOW_PREV=true; shift ;;
        --help|-h)  show_help ;;
        -*)         die "Unknown option: $1" ;;
        *)
            if [[ -z "$VERSION" ]]; then
                VERSION="$1"
            else
                die "Unexpected argument: $1"
            fi
            shift
            ;;
    esac
done

# ─── Show previous release / 前回のリリースノート表示 ───────────

if [[ "$SHOW_PREV" == true ]]; then
    echo ""
    if command -v gh >/dev/null 2>&1; then
        LATEST=$(gh release view --json tagName,name,body 2>/dev/null || echo "")
        if [[ -n "$LATEST" ]]; then
            TAG=$(echo "$LATEST" | jq -r '.tagName')
            NAME=$(echo "$LATEST" | jq -r '.name')
            BODY=$(echo "$LATEST" | jq -r '.body')
            echo -e "${BOLD}${MSG_LATEST_RELEASE} ${TAG} — ${NAME}${NC}"
            echo "──────────────────────────────────────"
            echo ""
            echo "$BODY"
            echo ""
            echo "──────────────────────────────────────"
        else
            warn "$MSG_NO_RELEASES"
        fi
    elif command -v curl >/dev/null 2>&1 && command -v jq >/dev/null 2>&1; then
        REPO=$(get_github_repo)
        if [[ -n "$REPO" ]]; then
            LATEST=$(curl -s "https://api.github.com/repos/${REPO}/releases" | jq -r '.[0]' 2>/dev/null || echo "")
            if [[ -n "$LATEST" && "$LATEST" != "null" ]]; then
                TAG=$(echo "$LATEST" | jq -r '.tag_name')
                NAME=$(echo "$LATEST" | jq -r '.name')
                BODY=$(echo "$LATEST" | jq -r '.body')
                echo -e "${BOLD}${MSG_LATEST_RELEASE} ${TAG} — ${NAME}${NC}"
                echo "──────────────────────────────────────"
                echo ""
                echo "$BODY"
                echo ""
                echo "──────────────────────────────────────"
            else
                warn "$MSG_NO_RELEASES"
            fi
        else
            die "$MSG_NO_REPO"
        fi
    else
        die "$MSG_REQUIRES_GH"
    fi
    echo ""
    exit 0
fi

[[ -z "$VERSION" ]] && die "$MSG_VERSION_REQUIRED"

# ─── Pre-flight checks / 事前チェック ───────────────────────────

echo ""
echo -e "${BOLD}${MSG_RELEASE_TITLE} ${VERSION}${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Semver format check / semver 形式チェック
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    die "$MSG_VERSION_FORMAT $VERSION"
fi

# Must be on main branch / main ブランチであること
BRANCH=$(git branch --show-current)
if [[ "$BRANCH" != "main" ]]; then
    die "$MSG_NOT_MAIN $BRANCH"
fi

# Working tree must be clean (only for publish mode) / ワーキングツリーがクリーンであること（publish時のみ）
if [[ -n "$NOTES_FILE" ]] && [[ -n "$(git status --porcelain)" ]]; then
    die "$MSG_NOT_CLEAN"
fi

# Validate notes file if specified / notes-file が指定されていれば存在チェック
if [[ -n "$NOTES_FILE" ]]; then
    if [[ ! -f "$NOTES_FILE" ]]; then
        die "$MSG_NOTES_NOT_FOUND $NOTES_FILE"
    fi
    if [[ ! -s "$NOTES_FILE" ]]; then
        die "$MSG_NOTES_EMPTY $NOTES_FILE"
    fi
fi

# Tag must not exist / 同名タグが存在しないこと
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    # shellcheck disable=SC2059
    die "$(printf "$MSG_TAG_EXISTS" "$VERSION")"
fi

# Find previous tag / 直前のタグを取得
PREV_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [[ -z "$PREV_TAG" ]]; then
    die "$MSG_NO_PREV_TAG"
fi

ok "$MSG_PREFLIGHT"
echo -e "  ${DIM}Branch: $BRANCH | Previous: $PREV_TAG | Target: $VERSION${NC}"
echo ""

# ─── Generate release notes / リリースノート生成 ────────────────

generate_notes() {
    local features=() fixes=() docs=() other=()

    while IFS= read -r line; do
        # Extract hash and message / ハッシュとメッセージを分離
        local hash="${line%% *}"
        local msg="${line#* }"

        local entry="- ${msg} (${hash})"

        # Classify: docs first (more specific), then fixes, then features
        # 分類: docs を先に判定（より具体的）、次に fixes、最後に features
        case "$msg" in
            *README*|*doc*|*Doc*|*CLAUDE.md*|*GEMINI.md*|*documentation*)
                docs+=("$entry") ;;
            Fix*|Resolve*|Correct*)
                fixes+=("$entry") ;;
            Add*|Implement*|Support*|Enable*)
                features+=("$entry") ;;
            *)
                other+=("$entry") ;;
        esac
    done < <(git log "${PREV_TAG}..HEAD" --oneline --no-merges)

    echo "## What's Changed"
    echo ""

    if [[ ${#features[@]} -gt 0 ]]; then
        echo "### Features"
        printf '%s\n' "${features[@]}"
        echo ""
    fi

    if [[ ${#fixes[@]} -gt 0 ]]; then
        echo "### Fixes"
        printf '%s\n' "${fixes[@]}"
        echo ""
    fi

    if [[ ${#docs[@]} -gt 0 ]]; then
        echo "### Documentation"
        printf '%s\n' "${docs[@]}"
        echo ""
    fi

    if [[ ${#other[@]} -gt 0 ]]; then
        echo "### Other"
        printf '%s\n' "${other[@]}"
        echo ""
    fi

    # Detect GitHub repo URL for Full Changelog link / GitHub リポジトリ URL から変更履歴リンクを生成
    local remote_url
    remote_url=$(git remote get-url origin 2>/dev/null || echo "")
    if [[ -n "$remote_url" ]]; then
        # Convert SSH or HTTPS URL to web URL / SSH・HTTPS の URL を Web URL に変換
        local web_url
        web_url=$(echo "$remote_url" | sed -E 's|git@github\.com:|https://github.com/|;s|\.git$||')
        echo "**Full Changelog**: ${web_url}/compare/${PREV_TAG}...${VERSION}"
    fi
}

if [[ -z "$(git log "${PREV_TAG}..HEAD" --oneline --no-merges)" ]]; then
    # shellcheck disable=SC2059
    die "$(printf "$MSG_NO_COMMITS" "$PREV_TAG")"
fi

NOTES=$(generate_notes)

# ─── Draft mode (default) / ドラフトモード（デフォルト） ────────

if [[ -z "$NOTES_FILE" ]]; then
    echo -e "${BOLD}${MSG_DRAFT_TITLE}${NC}"
    echo "──────────────────────────────────────"
    echo ""
    echo "$NOTES"
    echo ""
    echo "──────────────────────────────────────"

    # Write draft file / ドラフトファイルに書き出し
    echo "$NOTES" > "$DRAFT_FILE"

    echo ""
    ok "${DRAFT_FILE} ${MSG_WROTE}"
    echo ""
    echo -e "  ${BOLD}${MSG_NEXT_STEPS}${NC}"
    echo -e "    ${MSG_STEP1}"
    echo -e "      ${CYAN}.sandbox/scripts/release.sh --prev${NC}"
    echo -e "    ${MSG_STEP2}"
    echo -e "    ${MSG_STEP3}"
    echo -e "      ${CYAN}.sandbox/scripts/release.sh ${VERSION} --notes-file ${DRAFT_FILE}${NC}"
    echo ""
    exit 0
fi

# ─── Publish mode (--notes-file) / リリース実行モード ───────────

NOTES=$(cat "$NOTES_FILE")

echo -e "${BOLD}${MSG_NOTES_TITLE}${NC}"
echo "──────────────────────────────────────"
echo ""
echo "$NOTES"
echo ""
echo "──────────────────────────────────────"

# ─── Confirmation / 実行確認 ─────────────────────────────────────

echo ""
# shellcheck disable=SC2059
printf -v confirm_msg "$MSG_CONFIRM_TAG" "$VERSION"
echo -ne "${YELLOW}${confirm_msg} [y/N]: ${NC}"
read -r confirm
if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    info "$MSG_CANCELLED"
    exit 0
fi

# ─── Create tag / タグ作成 ───────────────────────────────────────

# Tag message: first line of notes as summary, full notes as body
# タグメッセージ: ノートの1行目を要約、全文を本文に
TAG_MSG="$NOTES"

git tag -a "$VERSION" -m "$TAG_MSG"
# shellcheck disable=SC2059
ok "$(printf "$MSG_TAG_CREATED" "$VERSION")"

# ─── Push tag / タグを push ──────────────────────────────────────

git push origin "$VERSION"
# shellcheck disable=SC2059
ok "$(printf "$MSG_TAG_PUSHED" "$VERSION")"

# ─── GitHub Release / GitHub Release 作成 ───────────────────────

show_manual_release_url() {
    local remote_url
    remote_url=$(git remote get-url origin 2>/dev/null || echo "")
    local web_url
    web_url=$(echo "$remote_url" | sed -E 's|git@github\.com:|https://github.com/|;s|\.git$||')

    info "$MSG_MANUAL_RELEASE"
    echo ""
    echo -e "  ${CYAN}${web_url}/releases/new?tag=${VERSION}${NC}"
    echo ""
    echo "  $MSG_PASTE_NOTES ${NOTES_FILE}"
}

echo ""

if command -v gh >/dev/null 2>&1; then
    if gh release create "$VERSION" --title "$VERSION" --notes-file "$NOTES_FILE"; then
        ok "$MSG_GH_CREATED"
        echo ""
        RELEASE_URL=$(gh release view "$VERSION" --json url -q '.url' 2>/dev/null || echo "")
        if [[ -n "$RELEASE_URL" ]]; then
            echo -e "  ${CYAN}${RELEASE_URL}${NC}"
        fi
    else
        warn "$MSG_GH_FAILED"
        echo ""
        show_manual_release_url
    fi
else
    info "$MSG_GH_NOT_FOUND"
    show_manual_release_url
fi

echo ""
# shellcheck disable=SC2059
ok "$(printf "$MSG_RELEASE_COMPLETE" "$VERSION")"
echo ""

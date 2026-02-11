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
            echo -e "${BOLD}📌 Latest Release: ${TAG} — ${NAME}${NC}"
            echo "──────────────────────────────────────"
            echo ""
            echo "$BODY"
            echo ""
            echo "──────────────────────────────────────"
        else
            warn "No releases found."
        fi
    elif command -v curl >/dev/null 2>&1 && command -v jq >/dev/null 2>&1; then
        REPO=$(get_github_repo)
        if [[ -n "$REPO" ]]; then
            LATEST=$(curl -s "https://api.github.com/repos/${REPO}/releases" | jq -r '.[0]' 2>/dev/null || echo "")
            if [[ -n "$LATEST" && "$LATEST" != "null" ]]; then
                TAG=$(echo "$LATEST" | jq -r '.tag_name')
                NAME=$(echo "$LATEST" | jq -r '.name')
                BODY=$(echo "$LATEST" | jq -r '.body')
                echo -e "${BOLD}📌 Latest Release: ${TAG} — ${NAME}${NC}"
                echo "──────────────────────────────────────"
                echo ""
                echo "$BODY"
                echo ""
                echo "──────────────────────────────────────"
            else
                warn "No releases found."
            fi
        else
            die "Could not detect GitHub repository from git remote."
        fi
    else
        die "Requires gh CLI or curl + jq."
    fi
    echo ""
    exit 0
fi

[[ -z "$VERSION" ]] && die "Version argument required. Usage: release.sh <version> [--notes-file <file>]"

# ─── Pre-flight checks / 事前チェック ───────────────────────────

echo ""
echo -e "${BOLD}🚀 Release: ${VERSION}${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Semver format check / semver 形式チェック
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    die "Version must be semver with v prefix (e.g. v0.4.0). Got: $VERSION"
fi

# Must be on main branch / main ブランチであること
BRANCH=$(git branch --show-current)
if [[ "$BRANCH" != "main" ]]; then
    die "Must be on 'main' branch. Currently on: $BRANCH"
fi

# Working tree must be clean (only for publish mode) / ワーキングツリーがクリーンであること（publish時のみ）
if [[ -n "$NOTES_FILE" ]] && [[ -n "$(git status --porcelain)" ]]; then
    die "Working tree is not clean. Commit or stash changes first."
fi

# Validate notes file if specified / notes-file が指定されていれば存在チェック
if [[ -n "$NOTES_FILE" ]]; then
    if [[ ! -f "$NOTES_FILE" ]]; then
        die "Notes file not found: $NOTES_FILE"
    fi
    if [[ ! -s "$NOTES_FILE" ]]; then
        die "Notes file is empty: $NOTES_FILE"
    fi
fi

# Tag must not exist / 同名タグが存在しないこと
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    die "Tag $VERSION already exists."
fi

# Find previous tag / 直前のタグを取得
PREV_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [[ -z "$PREV_TAG" ]]; then
    die "No previous tag found. Create the first tag manually."
fi

ok "Pre-flight checks passed"
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

NOTES=$(generate_notes)

if [[ -z "$(git log "${PREV_TAG}..HEAD" --oneline --no-merges)" ]]; then
    die "No commits since $PREV_TAG. Nothing to release."
fi

# ─── Draft mode (default) / ドラフトモード（デフォルト） ────────

if [[ -z "$NOTES_FILE" ]]; then
    echo -e "${BOLD}📋 Release Notes Draft${NC}"
    echo "──────────────────────────────────────"
    echo ""
    echo "$NOTES"
    echo ""
    echo "──────────────────────────────────────"

    # Write draft file / ドラフトファイルに書き出し
    echo "$NOTES" > "$DRAFT_FILE"

    echo ""
    ok "${DRAFT_FILE} を出力しました。"
    echo ""
    echo -e "  前回のトーンを確認したい場合:"
    echo -e "    ${CYAN}.sandbox/scripts/release.sh --prev${NC}"
    echo ""
    echo -e "  AI と相談してリリースノートを推敲することもできます。"
    echo -e "  完了したら:"
    echo -e "    ${CYAN}.sandbox/scripts/release.sh ${VERSION} --notes-file ${DRAFT_FILE}${NC}"
    echo ""
    exit 0
fi

# ─── Publish mode (--notes-file) / リリース実行モード ───────────

NOTES=$(cat "$NOTES_FILE")

echo -e "${BOLD}📋 Release Notes${NC}"
echo "──────────────────────────────────────"
echo ""
echo "$NOTES"
echo ""
echo "──────────────────────────────────────"

# ─── Confirmation / 実行確認 ─────────────────────────────────────

echo ""
echo -ne "${YELLOW}Create tag ${VERSION} and push? [y/N]: ${NC}"
read -r confirm
if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    info "Cancelled."
    exit 0
fi

# ─── Create tag / タグ作成 ───────────────────────────────────────

# Tag message: first line of notes as summary, full notes as body
# タグメッセージ: ノートの1行目を要約、全文を本文に
TAG_MSG="$NOTES"

git tag -a "$VERSION" -m "$TAG_MSG"
ok "Tag $VERSION created"

# ─── Push tag / タグを push ──────────────────────────────────────

git push origin "$VERSION"
ok "Tag $VERSION pushed to origin"

# ─── GitHub Release / GitHub Release 作成 ───────────────────────

show_manual_release_url() {
    local remote_url
    remote_url=$(git remote get-url origin 2>/dev/null || echo "")
    local web_url
    web_url=$(echo "$remote_url" | sed -E 's|git@github\.com:|https://github.com/|;s|\.git$||')

    info "Create the release manually:"
    echo ""
    echo -e "  ${CYAN}${web_url}/releases/new?tag=${VERSION}${NC}"
    echo ""
    echo "  Paste the release notes from: ${NOTES_FILE}"
}

echo ""

if command -v gh >/dev/null 2>&1; then
    if gh release create "$VERSION" --title "$VERSION" --notes-file "$NOTES_FILE"; then
        ok "GitHub Release created"
        echo ""
        RELEASE_URL=$(gh release view "$VERSION" --json url -q '.url' 2>/dev/null || echo "")
        if [[ -n "$RELEASE_URL" ]]; then
            echo -e "  ${CYAN}${RELEASE_URL}${NC}"
        fi
    else
        warn "gh release create failed."
        echo ""
        show_manual_release_url
    fi
else
    info "gh CLI not found."
    show_manual_release_url
fi

echo ""
ok "Release $VERSION complete! 🎉"
echo ""

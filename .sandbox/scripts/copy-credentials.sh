#!/bin/bash
# copy-credentials.sh
# Export/Import home directory (credentials, settings, history) using docker-compose.yml
#
# docker-compose.yml を使用してホームディレクトリ（認証情報・設定・履歴）を
# エクスポート/インポートします。
#
# Usage:
#   Export:
#     ./copy-credentials.sh --export <workspace-or-yaml> <backup-path>
#   Import:
#     ./copy-credentials.sh --import <backup-path> <workspace-or-yaml>
#
# Manual testing (run on host OS / ホストOSで実行):
#
#   1. Help display / ヘルプ表示:
#      ./.sandbox/scripts/copy-credentials.sh
#      → Usage message, exit code 1
#
#   2. Missing args / 引数不足:
#      ./.sandbox/scripts/copy-credentials.sh only-one-arg
#      → Usage message, exit code 1
#
#   3. Non-existent volume / 存在しないボリューム:
#      ./.sandbox/scripts/copy-credentials.sh --export nonexistent-project /tmp/test
#      → "volume ... not found" error, exit code 1
#
#   4. Actual copy / 実際のコピー (optional):
#      ./.sandbox/scripts/copy-credentials.sh --export /path/to/workspace ~/backup
#      docker volume ls | grep backup
#      # Cleanup: docker volume rm <created-volumes>
#
#   5. Overwrite prompt / 上書き確認:
#      Run export twice to same target → "already exists" warning, [y/N] prompt

set -e

# Debug mode
DEBUG=${DEBUG:-false}

debug() {
    if [ "$DEBUG" = "true" ]; then
        echo -e "\033[0;36m[DEBUG] $1\033[0m" >&2
    fi
}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Show usage
show_usage() {
    echo "Usage:"
    echo "  $0 --export <workspace-or-yaml> <backup-path>"
    echo "  $0 --import <backup-path> <workspace-or-yaml>"
    echo ""
    echo "Export/Import home directory (credentials, settings, history)."
    echo "ホームディレクトリ（認証情報・設定・履歴）をエクスポート/インポートします。"
    echo ""
    echo "Arguments / 引数:"
    echo "  <workspace-or-yaml>  Workspace directory or docker-compose.yml file"
    echo "                       ワークスペースディレクトリまたはdocker-compose.ymlファイル"
    echo ""
    echo "  <backup-path>        Backup directory path (absolute path required)"
    echo "                       バックアップディレクトリパス（絶対パス必須）"
    echo ""
    echo "Commands / コマンド:"
    echo "  --export     Export from Docker volumes to backup directory"
    echo "               Dockerボリュームからバックアップディレクトリへエクスポート"
    echo ""
    echo "  --import     Import from backup directory to Docker volumes"
    echo "               バックアップディレクトリからDockerボリュームへインポート"
    echo ""
    echo "Examples / 例:"
    echo "  # Export from workspace (both devcontainer and cli_sandbox)"
    echo "  $0 --export /path/to/workspace ~/backup"
    echo ""
    echo "  # Export from specific docker-compose.yml"
    echo "  $0 --export /path/to/.devcontainer/docker-compose.yml ~/backup"
    echo ""
    echo "  # Import to workspace"
    echo "  $0 --import ~/backup /path/to/workspace"
    echo ""
    echo "Workspace mode / ワークスペースモード:"
    echo "  When a workspace directory is specified, the script will automatically"
    echo "  detect and process .devcontainer/ and cli_sandbox/ subdirectories."
    echo "  ワークスペースディレクトリを指定すると、.devcontainer/ と cli_sandbox/"
    echo "  サブディレクトリを自動検出して処理します。"
    echo ""
    echo "Multiple AI tools / 複数のAIツール:"
    echo "  For cli_sandbox/, the script detects COMPOSE_PROJECT_NAME from all .sh files"
    echo "  (claude.sh, gemini.sh, ai_sandbox.sh, etc.) and exports/imports each separately."
    echo "  cli_sandbox/ では、全 .sh ファイル（claude.sh, gemini.sh, ai_sandbox.sh 等）から"
    echo "  COMPOSE_PROJECT_NAME を検出し、それぞれ個別にエクスポート/インポートします。"
    echo ""
    echo "  Backup structure / バックアップ構造:"
    echo "    backup/"
    echo "      ├── devcontainer/"
    echo "      │   └── home/"
    echo "      └── cli_sandbox/"
    echo "          ├── cli-claude/    # from claude.sh"
    echo "          │   └── home/"
    echo "          ├── cli-gemini/    # from gemini.sh"
    echo "          │   └── home/"
    echo "          └── cli-ai-sandbox/ # from ai_sandbox.sh"
    echo "              └── home/"
}

# Get COMPOSE_PROJECT_NAME values from cli_sandbox/*.sh files
# cli_sandbox/*.sh ファイルから COMPOSE_PROJECT_NAME の値を取得
# Returns: list of project names (one per line)
get_cli_sandbox_projects() {
    local workspace_dir="$1"
    local cli_sandbox_dir="$workspace_dir/cli_sandbox"

    debug "get_cli_sandbox_projects: workspace_dir=$workspace_dir"

    if [ ! -d "$cli_sandbox_dir" ]; then
        debug "get_cli_sandbox_projects: cli_sandbox directory not found"
        return 0
    fi

    # Extract COMPOSE_PROJECT_NAME from all .sh files in cli_sandbox/
    # cli_sandbox/ 内の全 .sh ファイルから COMPOSE_PROJECT_NAME を抽出
    # Pattern: ^COMPOSE_PROJECT_NAME= (see _common.sh for why this pattern)
    # パターン: ^COMPOSE_PROJECT_NAME= （このパターンの理由は _common.sh を参照）
    local projects
    projects=$(grep -h "^COMPOSE_PROJECT_NAME=" "$cli_sandbox_dir"/*.sh 2>/dev/null | \
               sed 's/COMPOSE_PROJECT_NAME=//' | \
               sort -u)

    debug "get_cli_sandbox_projects: found projects: $projects"
    echo "$projects"
}

# Get volume names from docker-compose.yml
# docker-compose.yml からボリューム名を取得
# Returns full volume name including project prefix (e.g., workspace_devcontainer_claude-home-vol)
#
# Strategy: Search existing Docker volumes for matching patterns
# 戦略: 既存のDockerボリュームからパターンにマッチするものを検索
get_home_volume() {
    local yaml_file="$1"
    local project_dir
    project_dir="$(dirname "$yaml_file")"

    debug "get_home_volume: yaml_file=$yaml_file"
    debug "get_home_volume: project_dir=$project_dir"

    # For cli_sandbox, run from parent directory (workspace) because build context is relative
    # cli_sandbox の場合、build context が相対パスのため親ディレクトリから実行
    local compose_dir="$project_dir"
    local compose_file="$yaml_file"
    if [[ "$yaml_file" == */cli_sandbox/* ]]; then
        compose_dir="$(dirname "$project_dir")"
        compose_file="cli_sandbox/docker-compose.yml"
        debug "get_home_volume: detected cli_sandbox, adjusted compose_dir=$compose_dir, compose_file=$compose_file"
    fi

    # Get volume name pattern from config
    local volume_pattern
    debug "get_home_volume: running: cd $compose_dir && docker compose -f $compose_file config --volumes"
    volume_pattern=$(cd "$compose_dir" && docker compose -f "$compose_file" config --volumes 2>/dev/null | grep -E '(claude-home|cli-sandbox-home)' | head -1)
    debug "get_home_volume: volume_pattern=$volume_pattern"

    if [ -z "$volume_pattern" ]; then
        debug "get_home_volume: volume_pattern is empty, returning 1"
        return 1
    fi

    # First, check if exact volume name exists (external volume case)
    # まず、完全一致するボリュームがあるか確認（外部ボリュームの場合）
    debug "get_home_volume: checking exact volume: $volume_pattern"
    if docker volume inspect "$volume_pattern" >/dev/null 2>&1; then
        debug "get_home_volume: exact match found"
        echo "$volume_pattern"
        return 0
    fi

    # Search existing volumes for pattern match
    # 既存ボリュームからパターンマッチで検索
    debug "get_home_volume: searching for volumes ending with _${volume_pattern}"
    local matching_volume
    matching_volume=$(docker volume ls --format '{{.Name}}' 2>/dev/null | grep -E "_${volume_pattern}$" | head -1)
    debug "get_home_volume: matching_volume=$matching_volume"

    if [ -n "$matching_volume" ]; then
        debug "get_home_volume: returning matching_volume"
        echo "$matching_volume"
        return 0
    fi

    # Fallback: construct from project name
    # フォールバック: プロジェクト名から構築
    debug "get_home_volume: falling back to project name construction"
    local project_name
    project_name=$(cd "$compose_dir" && docker compose -f "$compose_file" config 2>/dev/null | grep -E "^name:" | head -1 | sed 's/^name: *//')
    debug "get_home_volume: project_name=$project_name"

    if [ -n "$project_name" ]; then
        debug "get_home_volume: returning ${project_name}_${volume_pattern}"
        echo "${project_name}_${volume_pattern}"
    else
        debug "get_home_volume: returning $volume_pattern"
        echo "$volume_pattern"
    fi
}

get_gcloud_volume() {
    local yaml_file="$1"
    local project_dir
    project_dir="$(dirname "$yaml_file")"

    # For cli_sandbox, run from parent directory (workspace) because build context is relative
    # cli_sandbox の場合、build context が相対パスのため親ディレクトリから実行
    local compose_dir="$project_dir"
    local compose_file="$yaml_file"
    if [[ "$yaml_file" == */cli_sandbox/* ]]; then
        compose_dir="$(dirname "$project_dir")"
        compose_file="cli_sandbox/docker-compose.yml"
    fi

    # Get volume name pattern from config
    local volume_pattern
    volume_pattern=$(cd "$compose_dir" && docker compose -f "$compose_file" config --volumes 2>/dev/null | grep -E 'gcloud-config' | head -1)

    if [ -z "$volume_pattern" ]; then
        return 1
    fi

    # First, check if exact volume name exists
    if docker volume inspect "$volume_pattern" >/dev/null 2>&1; then
        echo "$volume_pattern"
        return 0
    fi

    # Search existing volumes for pattern match
    local matching_volume
    matching_volume=$(docker volume ls --format '{{.Name}}' 2>/dev/null | grep -E "_${volume_pattern}$" | head -1)

    if [ -n "$matching_volume" ]; then
        echo "$matching_volume"
        return 0
    fi

    # Fallback: construct from project name
    local project_name
    project_name=$(cd "$compose_dir" && docker compose -f "$compose_file" config 2>/dev/null | grep -E "^name:" | head -1 | sed 's/^name: *//')

    if [ -n "$project_name" ]; then
        echo "${project_name}_${volume_pattern}"
    else
        echo "$volume_pattern"
    fi
}

# Get project name from docker-compose.yml
# docker-compose.yml からプロジェクト名を取得
get_project_name() {
    local yaml_file="$1"
    local project_dir
    project_dir="$(dirname "$yaml_file")"

    # For cli_sandbox, run from parent directory (workspace) because build context is relative
    # cli_sandbox の場合、build context が相対パスのため親ディレクトリから実行
    local compose_dir="$project_dir"
    local compose_file="$yaml_file"
    local env_file=""
    local is_devcontainer=false

    if [[ "$yaml_file" == */cli_sandbox/* ]]; then
        compose_dir="$(dirname "$project_dir")"
        compose_file="cli_sandbox/docker-compose.yml"
        # cli_sandbox uses cli_sandbox/.env for COMPOSE_PROJECT_NAME (sourced by sandbox.sh)
        # cli_sandbox は cli_sandbox/.env から COMPOSE_PROJECT_NAME を読む（sandbox.sh で source される）
        env_file="$compose_dir/cli_sandbox/.env"
    elif [[ "$yaml_file" == */.devcontainer/* ]]; then
        is_devcontainer=true
        # devcontainer uses .devcontainer/.env
        env_file="$project_dir/.env"
    else
        env_file="$project_dir/.env"
    fi

    # Source the env file if it exists to get COMPOSE_PROJECT_NAME
    # .env ファイルが存在すれば source して COMPOSE_PROJECT_NAME を取得
    local env_project_name=""
    if [ -f "$env_file" ]; then
        env_project_name=$(grep -E "^COMPOSE_PROJECT_NAME=" "$env_file" 2>/dev/null | tail -1 | sed 's/^COMPOSE_PROJECT_NAME=//')
    fi

    # If COMPOSE_PROJECT_NAME is set in .env, use it directly
    # .env に COMPOSE_PROJECT_NAME が設定されていれば、それを使用
    if [ -n "$env_project_name" ]; then
        echo "$env_project_name"
        return 0
    fi

    # For VS Code DevContainers, use workspace_devcontainer naming convention
    # VS Code DevContainer の場合、workspace_devcontainer の命名規則を使用
    if [ "$is_devcontainer" = true ]; then
        # Get workspace directory name (parent of .devcontainer)
        local workspace_dir
        workspace_dir=$(basename "$(dirname "$project_dir")")
        # VS Code uses <workspace>_devcontainer as project name
        local vscode_project_name="${workspace_dir}_devcontainer"
        # Convert to lowercase and remove special characters
        vscode_project_name=$(echo "$vscode_project_name" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9_]//g')
        debug "get_project_name: VS Code DevContainer detected, using: $vscode_project_name"
        echo "$vscode_project_name"
        return 0
    fi

    # Try using docker compose config with name extraction
    # docker compose config からプロジェクト名を抽出
    local project_name

    # Method 1: Try to get from config output (handles both "name:" at start and with indentation)
    # 方法1: config出力から取得（行頭でもインデントありでも対応）
    project_name=$(cd "$compose_dir" && docker compose -f "$compose_file" config 2>/dev/null | grep -E "^name:" | head -1 | sed 's/^name: *//')

    # Method 2: If empty, try docker compose config --format json
    # 方法2: 空の場合、docker compose config --format json を試す
    if [ -z "$project_name" ]; then
        # Use docker compose config name field from JSON format if available
        project_name=$(cd "$compose_dir" && docker compose -f "$compose_file" config --format json 2>/dev/null | grep -o '"name"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
    fi

    # Method 3: Fallback to directory-based project name (docker compose default behavior)
    # 方法3: フォールバックとしてディレクトリ名ベースのプロジェクト名を使用
    if [ -z "$project_name" ]; then
        # Docker compose uses the directory name, converted to lowercase with special chars removed
        local dir_name
        dir_name=$(basename "$compose_dir")
        # Convert to lowercase and remove special characters (docker compose behavior)
        project_name=$(echo "$dir_name" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]//g')
    fi

    echo "$project_name"
}

# Save export metadata
# エクスポートメタデータを保存
save_export_metadata() {
    local backup_dir="$1"
    local yaml_file="$2"
    local project_name="$3"

    local metadata_file="$backup_dir/.export-metadata"
    local export_time
    export_time=$(date -Iseconds 2>/dev/null || date '+%Y-%m-%dT%H:%M:%S%z')

    cat > "$metadata_file" << EOF
{
  "source_project": "$project_name",
  "export_time": "$export_time",
  "source_path": "$yaml_file"
}
EOF
}

# Check project name and warn if same (single environment)
# プロジェクト名をチェックし、同じ場合は警告（単一環境用）
check_project_name_conflict() {
    local backup_dir="$1"
    local yaml_file="$2"
    local env_type="$3"  # "devcontainer" or "cli_sandbox" or "environment"

    local metadata_file="$backup_dir/.export-metadata"

    # メタデータファイルがなければスキップ
    if [ ! -f "$metadata_file" ]; then
        return 0
    fi

    # ソースプロジェクト名を取得
    local source_project
    source_project=$(grep '"source_project"' "$metadata_file" | sed 's/.*: *"\([^"]*\)".*/\1/')

    # ターゲットプロジェクト名を取得
    local target_project
    target_project=$(get_project_name "$yaml_file")

    # 同じ場合は警告
    if [ -n "$source_project" ] && [ -n "$target_project" ] && [ "$source_project" = "$target_project" ]; then
        echo ""
        echo -e "${YELLOW}========================================"
        echo -e "WARNING: Same project name detected!"
        echo -e "警告: 同じプロジェクト名が検出されました！"
        echo -e "========================================${NC}"
        echo ""
        echo "  Source / エクスポート元: $source_project"
        echo "  Target / インポート先:   $target_project"
        echo ""
        echo "This will overwrite the same volumes."
        echo "同じボリュームを上書きします。"
        echo ""
        echo "If this is a copied workspace, set COMPOSE_PROJECT_NAME:"
        echo "コピーしたワークスペースの場合は、COMPOSE_PROJECT_NAME を設定してください:"
        echo ""

        # 環境タイプに応じたパスを表示
        local yaml_dir
        yaml_dir="$(dirname "$yaml_file")"
        echo "  echo 'COMPOSE_PROJECT_NAME=new-project-name' >> $yaml_dir/.env"
        echo ""

        read -p "Continue anyway? / それでも続行しますか？ [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Cancelled. / キャンセルしました。"
            return 1
        fi
    fi

    return 0
}

# Check project name conflicts for workspace mode (multiple environments)
# ワークスペースモードで複数環境のプロジェクト名競合をチェック
check_workspace_project_conflicts() {
    local backup_path="$1"
    local devcontainer_yaml="$2"
    local cli_yaml="$3"

    local conflicts=()
    local env_paths=()

    # Check devcontainer
    if [ -n "$devcontainer_yaml" ] && [ -d "$backup_path/devcontainer" ]; then
        local metadata_file="$backup_path/devcontainer/.export-metadata"
        if [ -f "$metadata_file" ]; then
            local source_project
            source_project=$(grep '"source_project"' "$metadata_file" | sed 's/.*: *"\([^"]*\)".*/\1/')
            local target_project
            target_project=$(get_project_name "$devcontainer_yaml")

            if [ -n "$source_project" ] && [ -n "$target_project" ] && [ "$source_project" = "$target_project" ]; then
                conflicts+=("devcontainer: $source_project -> $target_project")
                env_paths+=("$(dirname "$devcontainer_yaml")/.env")
            fi
        fi
    fi

    # Check cli_sandbox
    if [ -n "$cli_yaml" ] && [ -d "$backup_path/cli_sandbox" ]; then
        local metadata_file="$backup_path/cli_sandbox/.export-metadata"
        if [ -f "$metadata_file" ]; then
            local source_project
            source_project=$(grep '"source_project"' "$metadata_file" | sed 's/.*: *"\([^"]*\)".*/\1/')
            local target_project
            target_project=$(get_project_name "$cli_yaml")

            if [ -n "$source_project" ] && [ -n "$target_project" ] && [ "$source_project" = "$target_project" ]; then
                conflicts+=("cli_sandbox: $source_project -> $target_project")
                # cli_sandbox uses cli_sandbox/.env (sourced by sandbox.sh before docker-compose)
                # cli_sandbox は cli_sandbox/.env を使用（sandbox.sh が docker-compose 前に source する）
                env_paths+=("$(dirname "$cli_yaml")/.env")
            fi
        fi
    fi

    # No conflicts
    if [ ${#conflicts[@]} -eq 0 ]; then
        return 0
    fi

    # Show warning
    echo ""
    echo -e "${YELLOW}========================================"
    echo -e "WARNING: Same project name detected!"
    echo -e "警告: 同じプロジェクト名が検出されました！"
    echo -e "========================================${NC}"
    echo ""
    echo "The following environments have the same project name:"
    echo "以下の環境で同じプロジェクト名が検出されました："
    echo ""
    for conflict in "${conflicts[@]}"; do
        echo "  - $conflict"
    done
    echo ""
    echo "This will overwrite the same volumes."
    echo "同じボリュームを上書きします。"
    echo ""
    echo "If this is a copied workspace, set COMPOSE_PROJECT_NAME:"
    echo "コピーしたワークスペースの場合は、COMPOSE_PROJECT_NAME を設定してください:"
    echo ""
    for env_path in "${env_paths[@]}"; do
        echo "  echo 'COMPOSE_PROJECT_NAME=new-project-name' >> $env_path"
    done
    echo ""

    read -p "Continue anyway? / それでも続行しますか？ [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Cancelled. / キャンセルしました。"
        return 1
    fi

    return 0
}

# Resolve input path to yaml file(s)
# 入力パスをYAMLファイルに解決
# Returns: "single:<yaml-path>" or "workspace:<devcontainer-yaml>:<cli-yaml>"
resolve_yaml_path() {
    local input_path="$1"

    # Remove trailing slash to avoid double slashes
    # 末尾のスラッシュを削除して二重スラッシュを防ぐ
    input_path="${input_path%/}"

    debug "resolve_yaml_path: input_path=$input_path"

    # Direct YAML file
    if [ -f "$input_path" ]; then
        debug "resolve_yaml_path: found direct YAML file"
        echo "single:$input_path"
        return 0
    fi

    # Directory - check for workspace structure
    if [ -d "$input_path" ]; then
        local devcontainer_yaml="$input_path/.devcontainer/docker-compose.yml"
        local cli_yaml="$input_path/cli_sandbox/docker-compose.yml"

        local has_devcontainer=false
        local has_cli=false

        [ -f "$devcontainer_yaml" ] && has_devcontainer=true
        [ -f "$cli_yaml" ] && has_cli=true

        debug "resolve_yaml_path: devcontainer_yaml=$devcontainer_yaml (exists: $has_devcontainer)"
        debug "resolve_yaml_path: cli_yaml=$cli_yaml (exists: $has_cli)"

        if $has_devcontainer || $has_cli; then
            # Return workspace format
            local dc_path=""
            local cli_path=""
            $has_devcontainer && dc_path="$devcontainer_yaml"
            $has_cli && cli_path="$cli_yaml"
            debug "resolve_yaml_path: returning workspace:$dc_path:$cli_path"
            echo "workspace:$dc_path:$cli_path"
            return 0
        fi

        # Check if it's a directory containing docker-compose.yml directly
        if [ -f "$input_path/docker-compose.yml" ]; then
            echo "single:$input_path/docker-compose.yml"
            return 0
        fi
    fi

    echo "Error: Cannot find docker-compose.yml in '$input_path'" >&2
    echo "エラー: '$input_path' に docker-compose.yml が見つかりません" >&2
    return 1
}

# Check if environment is running
# 環境が実行中か確認
is_environment_running() {
    local yaml_file="$1"
    local project_dir
    project_dir="$(dirname "$yaml_file")"

    # Run in subshell to prevent cd from affecting caller's working directory
    # サブシェルで実行し、cd が呼び出し元のカレントディレクトリに影響しないようにする
    (cd "$project_dir" && docker compose -f "$yaml_file" ps -q 2>/dev/null | grep -q .)
}

# Get volume name by constructing from project name (for both export and import)
# プロジェクト名からボリューム名を構築（エクスポート・インポート両方用）
get_volume_by_project() {
    local yaml_file="$1"
    local volume_type="$2"  # "home" or "gcloud"

    local project_dir
    project_dir="$(dirname "$yaml_file")"

    local compose_dir="$project_dir"
    local compose_file="$yaml_file"
    if [[ "$yaml_file" == */cli_sandbox/* ]]; then
        compose_dir="$(dirname "$project_dir")"
        compose_file="cli_sandbox/docker-compose.yml"
    fi

    # Get volume pattern from config
    local volume_pattern
    if [ "$volume_type" = "home" ]; then
        volume_pattern=$(cd "$compose_dir" && docker compose -f "$compose_file" config --volumes 2>/dev/null | grep -E '(claude-home|cli-sandbox-home)' | head -1)
    else
        volume_pattern=$(cd "$compose_dir" && docker compose -f "$compose_file" config --volumes 2>/dev/null | grep -E 'gcloud-config' | head -1)
    fi

    if [ -z "$volume_pattern" ]; then
        return 1
    fi

    # Get project name (this respects COMPOSE_PROJECT_NAME from .env)
    local project_name
    project_name=$(get_project_name "$yaml_file")

    debug "get_volume_by_project: volume_type=$volume_type, volume_pattern=$volume_pattern, project_name=$project_name"

    # Construct volume name: project_name + volume_pattern
    echo "${project_name}_${volume_pattern}"
}

# Get volume name directly from project name (for cli_sandbox projects extracted from .sh files)
# プロジェクト名から直接ボリューム名を取得（.sh ファイルから抽出した cli_sandbox プロジェクト用）
get_volume_by_project_name() {
    local project_name="$1"
    local volume_type="$2"  # "home" or "gcloud"

    debug "get_volume_by_project_name: project_name=$project_name, volume_type=$volume_type"

    # cli_sandbox uses cli-sandbox-home as volume name
    # cli_sandbox は cli-sandbox-home をボリューム名として使用
    local volume_pattern
    if [ "$volume_type" = "home" ]; then
        volume_pattern="cli-sandbox-home"
    else
        volume_pattern="gcloud-config"
    fi

    echo "${project_name}_${volume_pattern}"
}

# Export from a single yaml file
# 単一のYAMLファイルからエクスポート
export_from_yaml() {
    local yaml_file="$1"
    local backup_dir="$2"
    local env_name="$3"

    debug "export_from_yaml: yaml_file=$yaml_file, backup_dir=$backup_dir, env_name=$env_name"

    local home_vol
    local gcloud_vol
    local project_name
    # Use project-name-based volume detection (not searching existing volumes)
    # プロジェクト名ベースのボリューム検出を使用（既存ボリューム検索ではなく）
    home_vol=$(get_volume_by_project "$yaml_file" "home")
    debug "export_from_yaml: home_vol=$home_vol"
    gcloud_vol=$(get_volume_by_project "$yaml_file" "gcloud" || true)
    debug "export_from_yaml: gcloud_vol=$gcloud_vol"
    project_name=$(get_project_name "$yaml_file")
    debug "export_from_yaml: project_name=$project_name"

    if [ -z "$home_vol" ]; then
        echo -e "${RED}Error: No home volume found in $yaml_file${NC}"
        echo -e "${RED}エラー: $yaml_file にホームボリュームが見つかりません${NC}"
        return 1
    fi

    # Check if volume exists
    debug "export_from_yaml: checking if volume exists: $home_vol"
    if ! docker volume inspect "$home_vol" >/dev/null 2>&1; then
        echo -e "${YELLOW}Warning: Volume '$home_vol' not found (skipping)${NC}"
        echo -e "${YELLOW}警告: ボリューム '$home_vol' が見つかりません（スキップ）${NC}"
        return 0
    fi
    debug "export_from_yaml: volume exists, proceeding with export"

    mkdir -p "$backup_dir/home"
    mkdir -p "$backup_dir/gcloud"

    echo "  Exporting $env_name..."
    echo "    - Project: $project_name"
    echo "    - $home_vol -> home/ (excluding .cache, .vscode-server, .npm)"

    docker run --rm \
        -v "${home_vol}:/source:ro" \
        -v "${backup_dir}/home:/target" \
        alpine sh -c "cd /source && tar --exclude='.cache' --exclude='.vscode-server' --exclude='.npm' -cf - . | (cd /target && tar -xf -)"

    if [ -n "$gcloud_vol" ] && docker volume inspect "$gcloud_vol" >/dev/null 2>&1; then
        echo "    - $gcloud_vol -> gcloud/"
        docker run --rm \
            -v "${gcloud_vol}:/source:ro" \
            -v "${backup_dir}/gcloud:/target" \
            alpine sh -c "cp -a /source/. /target/"
    fi

    # Save metadata for import verification
    # インポート時の検証用にメタデータを保存
    save_export_metadata "$backup_dir" "$yaml_file" "$project_name"

    echo -e "  ${GREEN}Done${NC}"
}

# Export from a cli_sandbox project (using project name directly)
# cli_sandbox プロジェクトからエクスポート（プロジェクト名を直接使用）
export_from_cli_project() {
    local project_name="$1"
    local backup_dir="$2"

    debug "export_from_cli_project: project_name=$project_name, backup_dir=$backup_dir"

    local home_vol
    local gcloud_vol
    home_vol=$(get_volume_by_project_name "$project_name" "home")
    gcloud_vol=$(get_volume_by_project_name "$project_name" "gcloud")

    debug "export_from_cli_project: home_vol=$home_vol, gcloud_vol=$gcloud_vol"

    # Check if volume exists
    if ! docker volume inspect "$home_vol" >/dev/null 2>&1; then
        echo -e "${YELLOW}  Warning: Volume '$home_vol' not found (skipping)${NC}"
        echo -e "${YELLOW}  警告: ボリューム '$home_vol' が見つかりません（スキップ）${NC}"
        return 0
    fi

    mkdir -p "$backup_dir/home"
    mkdir -p "$backup_dir/gcloud"

    echo "  Exporting cli_sandbox/$project_name..."
    echo "    - Project: $project_name"
    echo "    - $home_vol -> home/ (excluding .cache, .vscode-server, .npm)"

    docker run --rm \
        -v "${home_vol}:/source:ro" \
        -v "${backup_dir}/home:/target" \
        alpine sh -c "cd /source && tar --exclude='.cache' --exclude='.vscode-server' --exclude='.npm' -cf - . | (cd /target && tar -xf -)"

    if [ -n "$gcloud_vol" ] && docker volume inspect "$gcloud_vol" >/dev/null 2>&1; then
        echo "    - $gcloud_vol -> gcloud/"
        docker run --rm \
            -v "${gcloud_vol}:/source:ro" \
            -v "${backup_dir}/gcloud:/target" \
            alpine sh -c "cp -a /source/. /target/"
    fi

    # Save metadata for import verification
    local metadata_file="$backup_dir/.export-metadata"
    local export_time
    export_time=$(date -Iseconds 2>/dev/null || date '+%Y-%m-%dT%H:%M:%S%z')

    cat > "$metadata_file" << EOF
{
  "source_project": "$project_name",
  "export_time": "$export_time",
  "source_type": "cli_sandbox"
}
EOF

    echo -e "  ${GREEN}Done${NC}"
}

# Import to a single yaml file
# 単一のYAMLファイルへインポート
# $4 (optional): "skip_conflict_check" to skip conflict check (already done in workspace mode)
import_to_yaml() {
    local backup_dir="$1"
    local yaml_file="$2"
    local env_name="$3"
    local skip_conflict_check="${4:-}"

    # Construct volume names from project name (don't search existing volumes)
    # プロジェクト名からボリューム名を構築（既存ボリュームを検索しない）
    local home_vol
    local gcloud_vol
    home_vol=$(get_volume_by_project "$yaml_file" "home")
    gcloud_vol=$(get_volume_by_project "$yaml_file" "gcloud" || true)

    if [ -z "$home_vol" ]; then
        echo -e "${RED}Error: No home volume found in $yaml_file${NC}"
        echo -e "${RED}エラー: $yaml_file にホームボリュームが見つかりません${NC}"
        return 1
    fi

    if [ ! -d "$backup_dir/home" ]; then
        echo -e "${YELLOW}Warning: No backup found at $backup_dir/home (skipping)${NC}"
        echo -e "${YELLOW}警告: $backup_dir/home にバックアップが見つかりません（スキップ）${NC}"
        return 0
    fi

    # Check for project name conflict (skip if already checked in workspace mode)
    # プロジェクト名の競合をチェック（ワークスペースモードで既にチェック済みの場合はスキップ）
    if [ "$skip_conflict_check" != "skip_conflict_check" ]; then
        if ! check_project_name_conflict "$backup_dir" "$yaml_file" "$env_name"; then
            return 1
        fi
    fi

    # Check if volume exists
    # ボリュームが存在するか確認
    if ! docker volume inspect "$home_vol" >/dev/null 2>&1; then
        # Volume doesn't exist - ask user to start environment first
        # ボリュームが存在しない - ユーザーに先に環境を起動してもらう
        echo ""
        echo -e "${RED}Error: Volume '$home_vol' not found.${NC}"
        echo -e "${RED}エラー: ボリューム '$home_vol' が見つかりません。${NC}"
        echo ""
        echo "Please start the environment once first to create volumes:"
        echo "ボリュームを作成するため、先に環境を一度起動してください："
        echo ""

        local yaml_dir
        yaml_dir="$(dirname "$yaml_file")"
        if [[ "$yaml_file" == */cli_sandbox/* ]]; then
            local workspace_dir
            workspace_dir="$(dirname "$yaml_dir")"
            echo "  cd $workspace_dir && ./cli_sandbox/sandbox.sh true"
        else
            echo "  Option 1 (CLI):"
            echo "  方法1（コマンドライン）:"
            echo "    cd $yaml_dir && docker compose up -d && docker compose down"
            echo ""
            echo "  Option 2 (VS Code):"
            echo "  方法2（VS Code）:"
            echo "    Open the workspace in VS Code and select 'Reopen in Container'"
            echo "    VS Code でワークスペースを開き「コンテナーで再度開く」を選択"
            echo "    Then close the DevContainer / その後 DevContainer を閉じる"
        fi
        echo ""
        echo "Then run the import again."
        echo "その後、再度インポートを実行してください。"
        return 1
    fi

    echo -e "${YELLOW}Note: Volume '$home_vol' exists. Data will be overwritten.${NC}"
    echo -e "${YELLOW}注意: ボリューム '$home_vol' は存在します。データは上書きされます。${NC}"

    echo "  Importing to $env_name..."
    echo "    - home/ -> $home_vol"

    # Default UID:GID for container user (node user in node:*-slim images)
    # コンテナユーザーのデフォルト UID:GID（node:*-slim イメージの node ユーザー）
    # Can be overridden with SANDBOX_UID and SANDBOX_GID environment variables
    # SANDBOX_UID と SANDBOX_GID 環境変数で上書き可能
    local uid="${SANDBOX_UID:-1000}"
    local gid="${SANDBOX_GID:-1000}"

    docker run --rm \
        -v "${backup_dir}/home:/source:ro" \
        -v "${home_vol}:/target" \
        alpine sh -c "cp -a /source/. /target/ && chown -R ${uid}:${gid} /target/"

    if [ -d "$backup_dir/gcloud" ] && [ -n "$gcloud_vol" ]; then
        echo "    - gcloud/ -> $gcloud_vol"
        docker run --rm \
            -v "${backup_dir}/gcloud:/source:ro" \
            -v "${gcloud_vol}:/target" \
            alpine sh -c "cp -a /source/. /target/ && chown -R ${uid}:${gid} /target/"
    fi

    echo -e "  ${GREEN}Done${NC}"

    # Show environment status
    # 環境のステータスを表示
    echo ""

    # For cli_sandbox, compose commands must run from parent directory
    # cli_sandbox の場合、compose コマンドは親ディレクトリから実行する必要がある
    local compose_cmd_dir
    local compose_cmd_file=""
    if [[ "$yaml_file" == */cli_sandbox/* ]]; then
        compose_cmd_dir="$(dirname "$(dirname "$yaml_file")")"
        compose_cmd_file=" -f cli_sandbox/docker-compose.yml"
    else
        compose_cmd_dir="$(dirname "$yaml_file")"
    fi

    if is_environment_running "$yaml_file"; then
        echo -e "${YELLOW}Note: Environment is running. Restart to apply changes:${NC}"
        echo -e "${YELLOW}注意: 環境は実行中です。変更を適用するには再起動してください:${NC}"
        echo "  cd $compose_cmd_dir && docker compose${compose_cmd_file} restart"
    else
        echo -e "${YELLOW}Note: Environment is not running. Start it with:${NC}"
        echo -e "${YELLOW}注意: 環境は実行中ではありません。起動するには:${NC}"
        echo "  cd $compose_cmd_dir && docker compose${compose_cmd_file} up -d"
    fi
}

# Import to a cli_sandbox project (using project name directly)
# cli_sandbox プロジェクトへインポート（プロジェクト名を直接使用）
import_to_cli_project() {
    local backup_dir="$1"
    local project_name="$2"
    local workspace_dir="$3"

    debug "import_to_cli_project: backup_dir=$backup_dir, project_name=$project_name, workspace_dir=$workspace_dir"

    local home_vol
    home_vol=$(get_volume_by_project_name "$project_name" "home")

    if [ ! -d "$backup_dir/home" ]; then
        echo -e "${YELLOW}  Warning: No backup found at $backup_dir/home (skipping)${NC}"
        echo -e "${YELLOW}  警告: $backup_dir/home にバックアップが見つかりません（スキップ）${NC}"
        return 0
    fi

    # Check if volume exists
    if ! docker volume inspect "$home_vol" >/dev/null 2>&1; then
        echo ""
        echo -e "${RED}  Error: Volume '$home_vol' not found.${NC}"
        echo -e "${RED}  エラー: ボリューム '$home_vol' が見つかりません。${NC}"
        echo ""
        echo "  Please start the environment once first to create volumes:"
        echo "  ボリュームを作成するため、先に環境を一度起動してください："
        echo ""

        # Find the script that uses this project name
        local script_file
        script_file=$(grep -l "COMPOSE_PROJECT_NAME=$project_name" "$workspace_dir/cli_sandbox"/*.sh 2>/dev/null | head -1)
        if [ -n "$script_file" ]; then
            echo "    cd $workspace_dir && ./cli_sandbox/$(basename "$script_file") true"
        else
            echo "    # Could not find script for $project_name"
        fi
        return 1
    fi

    echo -e "${YELLOW}  Note: Volume '$home_vol' exists. Data will be overwritten.${NC}"
    echo -e "${YELLOW}  注意: ボリューム '$home_vol' は存在します。データは上書きされます。${NC}"

    echo "  Importing to cli_sandbox/$project_name..."
    echo "    - home/ -> $home_vol"

    local uid="${SANDBOX_UID:-1000}"
    local gid="${SANDBOX_GID:-1000}"

    docker run --rm \
        -v "${backup_dir}/home:/source:ro" \
        -v "${home_vol}:/target" \
        alpine sh -c "cp -a /source/. /target/ && chown -R ${uid}:${gid} /target/"

    local gcloud_vol
    gcloud_vol=$(get_volume_by_project_name "$project_name" "gcloud")

    if [ -d "$backup_dir/gcloud" ] && docker volume inspect "$gcloud_vol" >/dev/null 2>&1; then
        echo "    - gcloud/ -> $gcloud_vol"
        docker run --rm \
            -v "${backup_dir}/gcloud:/source:ro" \
            -v "${gcloud_vol}:/target" \
            alpine sh -c "cp -a /source/. /target/ && chown -R ${uid}:${gid} /target/"
    fi

    echo -e "  ${GREEN}Done${NC}"
}

# Main export function
# メインエクスポート関数
do_export() {
    local yaml_path="$1"
    local backup_path="$2"

    debug "do_export: yaml_path=$yaml_path, backup_path=$backup_path"

    # Validate backup path is absolute
    backup_path="${backup_path/#\~/$HOME}"
    if [[ "$backup_path" != /* ]]; then
        echo -e "${RED}Error: Backup path must be absolute (starting with / or ~)${NC}"
        echo -e "${RED}エラー: バックアップパスは絶対パス（/ または ~ で始まる）である必要があります${NC}"
        exit 1
    fi

    local resolved
    resolved=$(resolve_yaml_path "$yaml_path") || exit 1

    local type="${resolved%%:*}"
    local paths="${resolved#*:}"

    debug "do_export: resolved=$resolved"
    debug "do_export: type=$type, paths=$paths"

    echo "========================================"
    echo "Exporting credentials / 認証情報をエクスポート"
    echo "========================================"
    echo ""

    if [ "$type" = "single" ]; then
        mkdir -p "$backup_path"
        export_from_yaml "$paths" "$backup_path" "environment"
    else
        # Workspace mode
        local devcontainer_yaml="${paths%%:*}"
        local cli_yaml="${paths#*:}"
        local workspace_dir="$yaml_path"

        debug "do_export: devcontainer_yaml='$devcontainer_yaml'"
        debug "do_export: cli_yaml='$cli_yaml'"
        debug "do_export: workspace_dir='$workspace_dir'"

        if [ -n "$devcontainer_yaml" ]; then
            mkdir -p "$backup_path/devcontainer"
            export_from_yaml "$devcontainer_yaml" "$backup_path/devcontainer" "devcontainer"
        fi

        # Export cli_sandbox projects (detect from .sh files)
        # cli_sandbox プロジェクトをエクスポート（.sh ファイルから検出）
        if [ -n "$cli_yaml" ]; then
            local cli_projects
            cli_projects=$(get_cli_sandbox_projects "$workspace_dir")

            if [ -n "$cli_projects" ]; then
                debug "do_export: found cli_sandbox projects: $cli_projects"
                echo "$cli_projects" | while read -r project_name; do
                    if [ -n "$project_name" ]; then
                        mkdir -p "$backup_path/cli_sandbox/$project_name"
                        export_from_cli_project "$project_name" "$backup_path/cli_sandbox/$project_name"
                    fi
                done
            else
                # Fallback: use old behavior (single cli_sandbox export)
                # フォールバック: 従来の動作（単一の cli_sandbox エクスポート）
                debug "do_export: no projects found in .sh files, using fallback"
                mkdir -p "$backup_path/cli_sandbox"
                export_from_yaml "$cli_yaml" "$backup_path/cli_sandbox" "cli_sandbox"
            fi
        else
            debug "do_export: cli_yaml is empty, skipping"
        fi
    fi

    echo ""
    echo "========================================"
    echo -e "${GREEN}Export complete! / エクスポート完了！${NC}"
    echo "Backup location / バックアップ先: $backup_path"
    echo "========================================"
}

# Main import function
# メインインポート関数
do_import() {
    local backup_path="$1"
    local yaml_path="$2"

    # Validate backup path is absolute
    backup_path="${backup_path/#\~/$HOME}"
    if [[ "$backup_path" != /* ]]; then
        echo -e "${RED}Error: Backup path must be absolute (starting with / or ~)${NC}"
        echo -e "${RED}エラー: バックアップパスは絶対パス（/ または ~ で始まる）である必要があります${NC}"
        exit 1
    fi

    if [ ! -d "$backup_path" ]; then
        echo -e "${RED}Error: Backup directory not found: $backup_path${NC}"
        echo -e "${RED}エラー: バックアップディレクトリが見つかりません: $backup_path${NC}"
        exit 1
    fi

    local resolved
    resolved=$(resolve_yaml_path "$yaml_path") || exit 1

    local type="${resolved%%:*}"
    local paths="${resolved#*:}"

    echo "========================================"
    echo "Importing credentials / 認証情報をインポート"
    echo "========================================"
    echo ""

    if [ "$type" = "single" ]; then
        import_to_yaml "$backup_path" "$paths" "environment"
    else
        # Workspace mode
        local devcontainer_yaml="${paths%%:*}"
        local cli_yaml="${paths#*:}"
        local workspace_dir="$yaml_path"

        # Check all project name conflicts upfront (only for devcontainer)
        # 先にプロジェクト名競合をチェック（devcontainer のみ）
        # Note: cli_sandbox conflicts are now handled per-project
        if [ -n "$devcontainer_yaml" ] && [ -d "$backup_path/devcontainer" ]; then
            local dc_metadata="$backup_path/devcontainer/.export-metadata"
            if [ -f "$dc_metadata" ]; then
                local source_project
                source_project=$(grep '"source_project"' "$dc_metadata" | sed 's/.*: *"\([^"]*\)".*/\1/')
                local target_project
                target_project=$(get_project_name "$devcontainer_yaml")

                if [ -n "$source_project" ] && [ -n "$target_project" ] && [ "$source_project" = "$target_project" ]; then
                    echo ""
                    echo -e "${YELLOW}========================================"
                    echo -e "WARNING: Same project name detected for devcontainer!"
                    echo -e "警告: devcontainer で同じプロジェクト名が検出されました！"
                    echo -e "========================================${NC}"
                    echo ""
                    echo "  Source / エクスポート元: $source_project"
                    echo "  Target / インポート先:   $target_project"
                    echo ""

                    read -p "Continue anyway? / それでも続行しますか？ [y/N] " -n 1 -r
                    echo
                    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                        echo "Cancelled. / キャンセルしました。"
                        exit 1
                    fi
                fi
            fi
        fi

        # Check devcontainer volume exists
        # devcontainer ボリュームの存在確認
        local missing_volumes=()
        local missing_commands=()

        if [ -n "$devcontainer_yaml" ] && [ -d "$backup_path/devcontainer" ]; then
            local dc_home_vol
            dc_home_vol=$(get_volume_by_project "$devcontainer_yaml" "home")
            if ! docker volume inspect "$dc_home_vol" >/dev/null 2>&1; then
                missing_volumes+=("devcontainer: $dc_home_vol")
                local dc_yaml_dir
                dc_yaml_dir="$(dirname "$devcontainer_yaml")"
                missing_commands+=("  # DevContainer:")
                missing_commands+=("  cd $dc_yaml_dir && docker compose up -d && docker compose down")
                missing_commands+=("  # Or open in VS Code and select 'Reopen in Container'")
                missing_commands+=("")
            fi
        fi

        if [ ${#missing_volumes[@]} -gt 0 ]; then
            echo ""
            echo -e "${RED}Error: The following volumes are not found:${NC}"
            echo -e "${RED}エラー: 以下のボリュームが見つかりません:${NC}"
            echo ""
            for vol in "${missing_volumes[@]}"; do
                echo "  - $vol"
            done
            echo ""
            echo "Please start the environments first to create volumes:"
            echo "ボリュームを作成するため、先に環境を起動してください："
            echo ""
            for cmd in "${missing_commands[@]}"; do
                echo "$cmd"
            done
            echo "Then run the import again."
            echo "その後、再度インポートを実行してください。"
            exit 1
        fi

        local imported_any=false

        if [ -n "$devcontainer_yaml" ] && [ -d "$backup_path/devcontainer" ]; then
            import_to_yaml "$backup_path/devcontainer" "$devcontainer_yaml" "devcontainer" "skip_conflict_check"
            imported_any=true
            echo ""
        fi

        # Import cli_sandbox projects (each project in its own subdirectory)
        # cli_sandbox プロジェクトをインポート（各プロジェクトは個別のサブディレクトリ）
        if [ -n "$cli_yaml" ] && [ -d "$backup_path/cli_sandbox" ]; then
            local import_errors=()

            for project_dir in "$backup_path/cli_sandbox"/*/; do
                if [ -d "$project_dir" ]; then
                    local project_name
                    project_name=$(basename "$project_dir")
                    debug "do_import: importing cli_sandbox project: $project_name"

                    if [ -d "$project_dir/home" ]; then
                        if import_to_cli_project "$project_dir" "$project_name" "$workspace_dir"; then
                            imported_any=true
                        else
                            import_errors+=("$project_name")
                        fi
                        echo ""
                    fi
                fi
            done

            if [ ${#import_errors[@]} -gt 0 ]; then
                echo -e "${YELLOW}Note: Some cli_sandbox projects could not be imported:${NC}"
                echo -e "${YELLOW}注意: 一部の cli_sandbox プロジェクトがインポートできませんでした:${NC}"
                for err in "${import_errors[@]}"; do
                    echo "  - $err"
                done
            fi
        fi

        if [ "$imported_any" = false ]; then
            echo -e "${RED}Error: No matching backup found for workspace environments${NC}"
            echo -e "${RED}エラー: ワークスペース環境に対応するバックアップが見つかりません${NC}"
            echo ""
            echo "Expected structure / 期待される構造:"
            echo "  $backup_path/"
            echo "    ├── devcontainer/"
            echo "    │   ├── home/"
            echo "    │   └── gcloud/"
            echo "    └── cli_sandbox/"
            echo "        ├── cli-claude/"
            echo "        │   └── home/"
            echo "        ├── cli-gemini/"
            echo "        │   └── home/"
            echo "        └── cli-ai-sandbox/"
            echo "            └── home/"
            exit 1
        fi
    fi

    echo ""
    echo "========================================"
    echo -e "${GREEN}Import complete! / インポート完了！${NC}"
    echo "========================================"
}

# Parse arguments
# 引数をパース
main() {
    local command=""
    local arg1=""
    local arg2=""

    while [ $# -gt 0 ]; do
        case "$1" in
            --export)
                command="export"
                shift
                ;;
            --import)
                command="import"
                shift
                ;;
            --help|-h)
                show_usage
                exit 0
                ;;
            *)
                # Positional arguments
                if [ -z "$arg1" ]; then
                    arg1="$1"
                elif [ -z "$arg2" ]; then
                    arg2="$1"
                fi
                shift
                ;;
        esac
    done

    # Validate arguments
    if [ -z "$command" ]; then
        echo -e "${RED}Error: Must specify --export or --import${NC}"
        echo -e "${RED}エラー: --export または --import を指定してください${NC}"
        echo ""
        show_usage
        exit 1
    fi

    if [ -z "$arg1" ] || [ -z "$arg2" ]; then
        echo -e "${RED}Error: Two arguments required${NC}"
        echo -e "${RED}エラー: 2つの引数が必要です${NC}"
        echo ""
        show_usage
        exit 1
    fi

    case "$command" in
        export)
            # --export <workspace-or-yaml> <backup-path>
            do_export "$arg1" "$arg2"
            ;;
        import)
            # --import <backup-path> <workspace-or-yaml>
            do_import "$arg1" "$arg2"
            ;;
    esac
}

main "$@"

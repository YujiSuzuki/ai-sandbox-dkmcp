#!/bin/bash
# demo-down.sh
# Stop demo applications (securenote-api, securenote-web)
#
# Usage:
#   demo-down.sh [workspace-root]
#
# Examples:
#   demo-down.sh /path/to/workspace
# ---
# デモアプリケーション（securenote-api, securenote-web）を停止します。

set -e

WORKSPACE="${1:-.}"

COMPOSE_FILE="$WORKSPACE/demo-apps/docker-compose.demo.yml"

if [ ! -f "$COMPOSE_FILE" ]; then
    echo "Error: $COMPOSE_FILE not found" >&2
    echo "Usage: demo-down.sh [workspace-root]" >&2
    exit 1
fi

echo "Stopping demo apps..."
echo "  Compose file: $COMPOSE_FILE"
docker compose -f "$COMPOSE_FILE" down

echo ""
echo "Demo apps stopped."

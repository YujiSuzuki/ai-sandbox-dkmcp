#!/bin/bash
# demo-up.sh
# Start demo applications (securenote-api, securenote-web)
#
# Usage:
#   demo-up.sh [workspace-root]
#
# Examples:
#   demo-up.sh /path/to/workspace
# ---
# デモアプリケーション（securenote-api, securenote-web）を起動します。

set -e

WORKSPACE="${1:-.}"

COMPOSE_FILE="$WORKSPACE/demo-apps/docker-compose.demo.yml"

if [ ! -f "$COMPOSE_FILE" ]; then
    echo "Error: $COMPOSE_FILE not found" >&2
    echo "Usage: demo-up.sh [workspace-root]" >&2
    exit 1
fi

echo "Starting demo apps..."
echo "  Compose file: $COMPOSE_FILE"
docker compose -f "$COMPOSE_FILE" up -d

echo ""
echo "Status:"
docker compose -f "$COMPOSE_FILE" ps

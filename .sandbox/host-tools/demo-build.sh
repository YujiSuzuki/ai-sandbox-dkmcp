#!/bin/bash
# demo-build.sh
# Build demo application images (securenote-api, securenote-web)
#
# Usage:
#   demo-build.sh [workspace-root]
#
# Examples:
#   demo-build.sh /path/to/workspace
# ---
# デモアプリケーションのDockerイメージをビルドします。

set -e

WORKSPACE="${1:-.}"

COMPOSE_FILE="$WORKSPACE/demo-apps/docker-compose.demo.yml"

if [ ! -f "$COMPOSE_FILE" ]; then
    echo "Error: $COMPOSE_FILE not found" >&2
    echo "Usage: demo-build.sh [workspace-root]" >&2
    exit 1
fi

echo "Building demo app images..."
echo "  Compose file: $COMPOSE_FILE"
docker compose -f "$COMPOSE_FILE" build

echo ""
echo "Build complete. Images:"
docker compose -f "$COMPOSE_FILE" images

#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> Stopping fake-platform-api stack"
docker compose -f "$REPO_ROOT/docker-compose.dev.yml" down

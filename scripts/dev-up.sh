#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> Starting fake-platform-api stack"
docker compose -f "$REPO_ROOT/docker-compose.dev.yml" up -d --build

echo "==> Waiting for fake-platform-api to be ready..."
for i in $(seq 1 20); do
  if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
    echo "==> fake-platform-api is up"
    exit 0
  fi
  sleep 1
done

echo "ERROR: fake-platform-api did not become healthy in time" >&2
exit 1

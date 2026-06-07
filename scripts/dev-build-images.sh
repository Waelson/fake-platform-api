#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> Building fake-api:v1 (healthy)"
docker build \
  --build-arg APP_MODE=healthy \
  -t fake-api:v1 \
  -f "$REPO_ROOT/test/fake-app/Dockerfile" \
  "$REPO_ROOT"

echo "==> Building fake-api:v2 (healthy, version 2)"
docker build \
  --build-arg APP_MODE=healthy \
  -t fake-api:v2 \
  -f "$REPO_ROOT/test/fake-app/Dockerfile" \
  "$REPO_ROOT"

echo "==> Building fake-api:broken"
docker build \
  --build-arg APP_MODE=broken \
  -t fake-api:broken \
  -f "$REPO_ROOT/test/fake-app/Dockerfile" \
  "$REPO_ROOT"

echo "==> Building fake-api:slow"
docker build \
  --build-arg APP_MODE=slow \
  -t fake-api:slow \
  -f "$REPO_ROOT/test/fake-app/Dockerfile" \
  "$REPO_ROOT"

echo "==> Building fake-worker:v1"
docker build \
  -t fake-worker:v1 \
  -f "$REPO_ROOT/test/fake-worker/Dockerfile" \
  "$REPO_ROOT"

echo "==> Done. Images built:"
docker images | grep -E "^(fake-api|fake-worker)"

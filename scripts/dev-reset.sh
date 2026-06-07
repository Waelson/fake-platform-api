#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${FAKE_API_URL:-http://localhost:8080}"

echo "==> Resetting fake-platform-api state"
curl -sf -X POST "${BASE_URL}/testing/reset" | jq .
echo "==> Done"

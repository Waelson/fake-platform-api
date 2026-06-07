#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${FAKE_API_URL:-http://localhost:8080}"
ENVIRONMENT="${DEVEX_FAKE_ENVIRONMENT:-dev}"
HOST="${TEST_HOST:-billing-api.${ENVIRONMENT}.local}"

echo "==> Testing route for host=${HOST}"
echo ""

echo "--- Desired state ---"
curl -sf "${BASE_URL}/testing/desired-state?environment=${ENVIRONMENT}" | jq .
echo ""

echo "--- Deployments ---"
curl -sf "${BASE_URL}/testing/deployments" | jq .
echo ""

echo "--- HTTP test (requires Caddy running and route active) ---"
curl -sv "http://${HOST}/" 2>&1 | tail -20

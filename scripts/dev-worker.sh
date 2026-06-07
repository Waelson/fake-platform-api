#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${FAKE_API_URL:-http://localhost:8080}"
ENVIRONMENT="${DEVEX_FAKE_ENVIRONMENT:-dev}"

echo "==> Deploying invoice-worker to environment=${ENVIRONMENT} (requires_route=false)"

curl -sf -X POST "${BASE_URL}/testing/commands/deploy" \
  -H "Content-Type: application/json" \
  -d "{
    \"target_agent_role\": \"worker\",
    \"application\": \"invoice-worker\",
    \"environment\": \"${ENVIRONMENT}\",
    \"image\": \"fake-worker:v1\",
    \"container_internal_port\": 0,
    \"requires_route\": false,
    \"environment_variables\": {
      \"WORKER_NAME\": \"invoice-worker\",
      \"WORKER_VERSION\": \"v1\"
    }
  }" | jq .

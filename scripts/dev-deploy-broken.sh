#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${FAKE_API_URL:-http://localhost:8080}"
ENVIRONMENT="${DEVEX_FAKE_ENVIRONMENT:-dev}"

echo "==> Deploying billing-api broken to environment=${ENVIRONMENT}"

curl -sf -X POST "${BASE_URL}/testing/commands/deploy" \
  -H "Content-Type: application/json" \
  -d "{
    \"target_agent_role\": \"api\",
    \"application\": \"billing-api\",
    \"environment\": \"${ENVIRONMENT}\",
    \"image\": \"fake-api:broken\",
    \"container_internal_port\": 3000,
    \"health_check_path\": \"/health\",
    \"requires_route\": true,
    \"host\": \"billing-api.${ENVIRONMENT}.local\",
    \"environment_variables\": {
      \"APP_NAME\": \"billing-api\",
      \"APP_VERSION\": \"broken\",
      \"APP_MODE\": \"broken\",
      \"PORT\": \"3000\"
    }
  }" | jq .

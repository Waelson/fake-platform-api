# 10 — Estratégia de Testes

## Unit tests

Testar:

- register agent
- re-register agent
- agent_id generation
- heartbeat stores last_heartbeat
- create deploy command
- claim pending -> claimed
- claim same agent idempotent 200
- claim different agent 409
- start claimed -> running
- report succeeded requires_route=true
- report succeeded requires_route=false
- report failed
- desired state by environment
- stale desired-state report
- future desired-state report 409
- remove route on remove succeeded
- reset clears counters

## Handler tests

Testar:

```text
POST /api/agents/register
POST /api/agents/{id}/heartbeat
GET /api/agents/{id}/commands/pending
POST /api/agents/{id}/commands/{cmd}/claim
POST /api/agents/{id}/commands/{cmd}/start
POST /api/agents/{id}/commands/{cmd}/report
GET /api/agents/{id}/desired-state
POST /api/agents/{id}/desired-state/report
POST /testing/commands/deploy
POST /testing/commands/stop
POST /testing/commands/remove
POST /testing/commands/cleanup-draining
GET /testing/commands?status=pending&type=DEPLOY_APPLICATION
POST /testing/reset
```

## Manual end-to-end

```bash
make dev-up
make dev-build-images
make run-runtime-agent
make run-gateway-agent
make dev-deploy-v1
make dev-test-route
make dev-deploy-v2
make dev-test-route
make dev-deploy-broken
make dev-worker
```

---

## Testes adicionais obrigatórios

```text
GET /health sem auth quando auth enabled
GET /testing/desired-state sem environment retorna todos
GET /testing/desired-state com environment retorna apenas o environment informado
GET /testing/debug retorna dump completo
DEPLOY_APPLICATION inclui labels automáticas
container_internal_port=0 é aceito para worker
Command.Payload preserva JSON específico por tipo de comando
STOP_APPLICATION report sem deployment_id usa Command.DeploymentID como fallback
```

---

## Testes de labels obrigatórias

```text
DEPLOY_APPLICATION pending command inclui as 5 labels obrigatórias:
  devex.managed=true
  devex.application={application}
  devex.environment={environment}
  devex.deployment_id={deployment_id}
  devex.command_id={command_id}
labels não podem ser omitidas no payload retornado por GET /commands/pending
```

---

## Testes de desired-state com environment inexistente

```text
GET /testing/desired-state?environment=prod quando nenhum deploy foi feito para "prod"
  → HTTP 200
  → body: {"version": 0, "type": "gateway_routes", "environment": "prod", "routes": []}
  → não retorna 404
```

---

## Testes de CleanupDrainingPayload

```text
cleanup-draining com environment explícito → payload usa environment informado
cleanup-draining sem environment → payload usa DEVEX_FAKE_ENVIRONMENT
cleanup-draining com older_than_seconds=0 → payload usa default 300
cleanup-draining com older_than_seconds=600 → payload usa 600
```

---

## Testes de TimeoutSeconds padrão

```text
DEPLOY_APPLICATION criado sem timeout_seconds → command.timeout_seconds = 600
STOP_APPLICATION criado sem timeout_seconds → command.timeout_seconds = 60
REMOVE_DEPLOYMENT criado sem timeout_seconds → command.timeout_seconds = 120
CLEANUP_DRAINING criado sem timeout_seconds → command.timeout_seconds = 300
comando criado com timeout_seconds=120 → command.timeout_seconds = 120 (valor informado vence)
timeout_seconds aparece no response de GET /commands/pending
```

---

## Testes de report com result flexível

```text
STOP_APPLICATION report succeeded com result {"container_name": "...", "stopped": true}
    → command → succeeded
    → deployment → removed
    → report armazenado com result intacto

REMOVE_DEPLOYMENT report succeeded com result {"removed": true, "port_released": true}
    → command → succeeded
    → deployment → removed
    → se deployment tinha rota: rota removida, desired state version incrementado
    → report armazenado com result intacto

CLEANUP_DRAINING report succeeded com result {"cleaned": 2}
    → command → succeeded
    → nenhum deployment alterado
    → report armazenado com result intacto

qualquer tipo report failed → command → failed; deployment sem alteração de desired state
```

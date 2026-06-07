# 02 — Contratos da API

## Convenções

Todos os endpoints retornam JSON.

Erro padrão:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Mensagem sanitizada",
    "retryable": false
  }
}
```

---

## POST /api/agents/register

### Request

```json
{
  "mode": "runtime",
  "environment": "dev",
  "role": "api",
  "hostname": "local-runtime",
  "instance_id": "local-runtime-api-001",
  "private_ip": "127.0.0.1",
  "public_ip": null,
  "version": "0.1.0",
  "capabilities": {
    "workload_types": ["api"],
    "max_active_containers": 10,
    "port_range": {
      "from": 4100,
      "to": 4114
    }
  }
}
```

### Response

```json
{
  "agent_id": "agent-dev-api-001",
  "status": "registered"
}
```

### Regras

- `agent_id = agent-{environment}-{role}-{NNN}`.
- Counter por `environment + role`.
- Re-register preserva `agent_id`.
- Atualizar agent data e `last_seen_at`.

---

## POST /api/agents/{agent_id}/heartbeat

### Request Runtime Agent

```json
{
  "status": "online",
  "mode": "runtime",
  "environment": "dev",
  "role": "api",
  "version": "0.1.0",
  "private_ip": "127.0.0.1",
  "running_containers": 2,
  "active_deployments": 2,
  "allocated_ports": 2,
  "last_command_id": "cmd-001",
  "last_successful_command_id": "cmd-001"
}
```

### Request Gateway Agent

```json
{
  "status": "online",
  "mode": "gateway",
  "environment": "dev",
  "role": "gateway",
  "version": "0.1.0",
  "caddy_status": "healthy",
  "routes_total": 1,
  "last_applied_desired_state_version": 1,
  "last_successful_desired_state_version": 1
}
```

### Response

```json
{
  "status": "accepted",
  "server_time": "2026-06-05T18:00:00Z"
}
```

### Regra

Armazenar o payload completo em `Agent.LastHeartbeat`.

---

## GET /api/agents/{agent_id}/commands/pending

Retorna comandos `pending` do agent.

### Response

```json
[
  {
    "id": "cmd-001",
    "type": "DEPLOY_APPLICATION",
    "deployment_id": "dep-001",
    "target_agent_id": "agent-dev-api-001",
    "status": "pending",
    "timeout_seconds": 600,
    "created_at": "2026-06-05T18:00:00Z",
    "payload": {
      "application": "billing-api",
      "environment": "dev",
      "image": "fake-api:v1",
      "container_name": "billing-api-dev-dep-001",
      "container_internal_port": 3000,
      "health_check_path": "/health",
      "requires_route": true,
      "environment_variables": {
        "APP_NAME": "billing-api",
        "APP_VERSION": "v1",
        "APP_MODE": "healthy",
        "PORT": "3000"
      },
      "labels": {
        "devex.managed": "true",
        "devex.application": "billing-api",
        "devex.environment": "dev",
        "devex.deployment_id": "dep-001",
        "devex.command_id": "cmd-001"
      }
    }
  }
]
```

---

## POST /api/agents/{agent_id}/commands/{command_id}/claim

### Request

```json
{
  "status": "claimed"
}
```

### Idempotência final

- Se status é `pending`: transicionar para `claimed` e retornar 200.
- Se status é `claimed` e `claimed_by == agent_id`: retornar 200 com estado atual.
- Se status é `claimed` e `claimed_by != agent_id`: retornar 409.
- Se status é `running`, `succeeded` ou `failed`: retornar 409.

### Response 200

```json
{
  "id": "cmd-001",
  "status": "claimed",
  "claimed_by": "agent-dev-api-001",
  "claimed_at": "2026-06-05T18:00:05Z"
}
```

---

## POST /api/agents/{agent_id}/commands/{command_id}/start

### Request

```json
{
  "status": "running"
}
```

### Regras

- `claimed -> running`.
- Se já está `running` pelo mesmo agent, retornar 200 idempotente.
- Se finalizado, retornar 409.

### Response

```json
{
  "id": "cmd-001",
  "status": "running",
  "started_at": "2026-06-05T18:00:06Z"
}
```

---

## POST /api/agents/{agent_id}/commands/{command_id}/report

### Request succeeded

```json
{
  "status": "succeeded",
  "deployment_id": "dep-001",
  "result": {
    "application": "billing-api",
    "environment": "dev",
    "container_name": "billing-api-dev-dep-001",
    "image": "fake-api:v1",
    "runtime_private_ip": "127.0.0.1",
    "host_port": 4100,
    "container_internal_port": 3000,
    "health": "healthy",
    "requires_route": true
  }
}
```

### Request failed

```json
{
  "status": "failed",
  "deployment_id": "dep-001",
  "error": {
    "code": "HEALTH_CHECK_FAILED",
    "message": "Application did not return a successful health response",
    "operation": "health.http",
    "retryable": false
  }
}
```

### Campo result como JSON flexível

O campo `result` é aceito como JSON flexível (`json.RawMessage`).

A Fake API não valida o conteúdo de `result` para comandos diferentes de `DEPLOY_APPLICATION`.

Ela apenas armazena o report e aplica transições de estado com base em:

- `command.type`
- `status`
- `command.deployment_id` e `report.deployment_id` (com fallback)

Exemplos de result por tipo:

**STOP_APPLICATION succeeded:**

```json
{
  "container_name": "billing-api-dev-dep-001",
  "stopped": true
}
```

**REMOVE_DEPLOYMENT succeeded:**

```json
{
  "container_name": "billing-api-dev-dep-001",
  "removed": true,
  "port_released": true
}
```

**CLEANUP_DRAINING succeeded:**

```json
{
  "cleaned": 2
}
```

### Regras por command.type no report

| command.type | status | Ação |
|---|---|---|
| `DEPLOY_APPLICATION` | `succeeded` | Ver regras succeeded acima |
| `STOP_APPLICATION` | `succeeded` | Deployment efetivo → `removed` |
| `REMOVE_DEPLOYMENT` | `succeeded` | Deployment efetivo → `removed`; se rota associada, remover rota e incrementar desired state |
| `CLEANUP_DRAINING` | `succeeded` | Apenas armazenar report (MVP) |
| qualquer | `failed` | Armazenar erro; não alterar desired state |

O deployment efetivo é resolvido por:

```text
effective_deployment_id = report.deployment_id
if report.deployment_id == "":
    effective_deployment_id = command.deployment_id
```

### Regras succeeded

Se `requires_route=true`:

1. `Command -> succeeded`.
2. `Deployment -> healthy`.
3. Criar/atualizar rota por `{environment, host}`.
4. `Deployment -> route_pending`.
5. Incrementar desired state version.

Se `requires_route=false`:

1. `Command -> succeeded`.
2. `Deployment -> healthy`.
3. Não criar rota.
4. Não incrementar desired state.

### Regras failed

1. `Command -> failed`.
2. `Deployment -> failed`.
3. Não alterar rota.
4. Não incrementar desired state.

---

## GET /api/agents/{agent_id}/desired-state

Retorna desired state do `environment` do agent.

### Response — com rotas

```json
{
  "version": 1,
  "type": "gateway_routes",
  "environment": "dev",
  "routes": [
    {
      "id": "route-001",
      "host": "billing-api.dev.useclarus.local",
      "path": "/",
      "upstream": "10.0.0.42:4100",
      "deployment_id": "dep-001",
      "health_check_path": "/health"
    }
  ]
}
```

### Response — environment sem rotas

Se nenhum desired state existir para o environment do agent, retornar HTTP 200 com estado vazio.

Não retornar 404.

```json
{
  "version": 0,
  "type": "gateway_routes",
  "environment": "dev",
  "routes": []
}
```

Ausência de rotas não é condição de erro. O Gateway Agent deve interpretar `routes: []` como instrução para não configurar nenhuma rota.

---

## POST /api/agents/{agent_id}/desired-state/report

### Request applied

```json
{
  "status": "applied",
  "desired_state_version": 1,
  "type": "gateway_routes",
  "environment": "dev",
  "routes_total": 1,
  "validated_routes": 1,
  "failed_routes": 0,
  "applied_at": "2026-06-05T18:00:30Z"
}
```

### Request failed

```json
{
  "status": "failed",
  "desired_state_version": 1,
  "type": "gateway_routes",
  "environment": "dev",
  "error": {
    "code": "CADDY_ROUTE_VALIDATION_FAILED",
    "message": "Route billing-api.dev.useclarus.local did not pass health check",
    "operation": "caddy.validate_route",
    "retryable": false
  }
}
```

### Regra de versão

- Se `desired_state_version == current version`: processar.
- Se `desired_state_version < current version`: registrar report como stale e não alterar deployments.
- Se `desired_state_version > current version`: registrar report como invalid e retornar 409.

### Regras applied atual

Para cada rota do environment/version atual:

- deployment em `route_pending` vira `route_active`.

### Regras failed atual

Para deployments relacionados em `route_pending`:

- mudar para `route_failed`.

---

## Status HTTP

- 200: sucesso.
- 201: criado.
- 400: payload inválido.
- 401: auth inválida.
- 403: agent sem permissão.
- 404: recurso não encontrado.
- 409: conflito de estado.
- 500: erro interno.

---

## Contrato adicional: GET /health

Endpoint público de health check da Fake Platform API.

```http
GET /health
```

Response:

```json
{
  "status": "ok"
}
```

Regras:

- Não exige autenticação.
- Deve funcionar mesmo quando `DEVEX_FAKE_AUTH_ENABLED=true`.

---

## Payloads e json.RawMessage

O campo `payload` de `Command` deve ser tratado como JSON flexível.

A implementação recomendada em Go é:

```go
Payload json.RawMessage `json:"payload"`
```

Cada tipo de comando deve possuir uma struct específica para montar ou interpretar o payload.

---

## Labels geradas automaticamente

Ao criar um comando `DEPLOY_APPLICATION`, a Fake API deve gerar as labels obrigatórias:

```json
{
  "devex.managed": "true",
  "devex.application": "billing-api",
  "devex.environment": "dev",
  "devex.deployment_id": "dep-001",
  "devex.command_id": "cmd-001"
}
```

Essas labels devem aparecer no payload retornado em:

```http
GET /api/agents/{agent_id}/commands/pending
```

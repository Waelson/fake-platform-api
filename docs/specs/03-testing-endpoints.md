# 03 — Endpoints de Teste

## POST /testing/commands/deploy

Cria deployment e comando `DEPLOY_APPLICATION`.

### Request

```json
{
  "target_agent_role": "api",
  "application": "billing-api",
  "environment": "dev",
  "image": "fake-api:v1",
  "container_name": "",
  "container_internal_port": 3000,
  "health_check_path": "/health",
  "requires_route": true,
  "host": "billing-api.dev.useclarus.local",
  "environment_variables": {
    "APP_NAME": "billing-api",
    "APP_VERSION": "v1",
    "APP_MODE": "healthy",
    "PORT": "3000"
  }
}
```

### Defaults

Se `environment` estiver vazio:

```text
environment = DEVEX_FAKE_ENVIRONMENT
```

Se `container_name` estiver vazio:

```text
container_name = {application}-{environment}-{deployment_id}
```

Se `requires_route=false`, `host` pode ser vazio.

### Response

```json
{
  "deployment_id": "dep-001",
  "command_id": "cmd-001",
  "target_agent_id": "agent-dev-api-001",
  "status": "pending"
}
```

---

## POST /testing/commands/stop

Cria comando `STOP_APPLICATION`.

### Request

```json
{
  "target_agent_id": "agent-dev-api-001",
  "deployment_id": "dep-001",
  "container_name": "billing-api-dev-dep-001",
  "stop_timeout_seconds": 30
}
```

### timeout_seconds

Default: `60` segundos se não informado.

### Store behavior

- Criar command `STOP_APPLICATION` com status `pending`.
- Não alterar deployment imediatamente.
- Quando command reportar `succeeded`, deployment efetivo → `removed`.
- Quando command reportar `failed`, deployment permanece no estado atual.

---

## POST /testing/commands/remove

Cria comando `REMOVE_DEPLOYMENT`.

### Request

```json
{
  "target_agent_id": "agent-dev-api-001",
  "deployment_id": "dep-001",
  "container_name": "billing-api-dev-dep-001",
  "release_port": true
}
```

### timeout_seconds

Default: `120` segundos se não informado.

### Store behavior

- Criar command `REMOVE_DEPLOYMENT` pending.
- Quando reportar `succeeded`, deployment efetivo → `removed`.
- Se deployment tinha rota associada, remover rota do desired state.
- Incrementar desired state version se rota for removida.

---

## POST /testing/commands/cleanup-draining

Cria comando `CLEANUP_DRAINING`.

### Request

```json
{
  "target_agent_role": "api",
  "environment": "dev",
  "older_than_seconds": 300
}
```

### Defaults

- `environment`: se vazio, usar `DEVEX_FAKE_ENVIRONMENT`.
- `older_than_seconds`: se zero ou ausente, usar `300`.

### Payload gerado

```go
CleanupDrainingPayload{
    Environment:      environment,
    OlderThanSeconds: older_than_seconds,
}
```

### timeout_seconds

Default: `300` segundos se não informado.

### Store behavior

- Escolher Runtime Agent por role/environment.
- Criar command `CLEANUP_DRAINING` pending.
- No MVP, não alterar deployments automaticamente.
- Quando report recebido, apenas armazenar `CommandReport`.

---

## GET /testing/agents

Lista agents.

---

## GET /testing/commands

Lista comandos.

Query params:

```text
status
type
agent_id
```

Todos opcionais.

---

## GET /testing/reports

Lista command reports.

---

## GET /testing/deployments

Lista deployments.

---

## GET /testing/desired-state

Query param opcional: `environment`.

### Com `?environment=dev`

Retorna o `DesiredState` daquele environment diretamente:

```json
{
  "version": 1,
  "type": "gateway_routes",
  "environment": "dev",
  "routes": []
}
```

Se o environment ainda não existir na store, retornar HTTP 200 com desired state vazio:

```json
{
  "version": 0,
  "type": "gateway_routes",
  "environment": "prod",
  "routes": []
}
```

Não retornar 404. Ausência de rotas não é um erro.

### Sem `environment`

Retorna todos os desired states agrupados por environment:

```json
{
  "desired_states": {
    "dev": {
      "version": 1,
      "type": "gateway_routes",
      "environment": "dev",
      "routes": []
    },
    "stage": {
      "version": 1,
      "type": "gateway_routes",
      "environment": "stage",
      "routes": []
    }
  }
}
```

O formato diferente por presença/ausência do query param é intencional para facilitar inspeção manual.

---

## GET /testing/desired-state/reports

Lista desired-state reports.

---

## GET /testing/debug

Dump completo da store.

---

## POST /testing/reset

Limpa tudo:

- agents
- agent indexes
- commands
- deployments
- reports
- desired states
- desired state reports
- counters

Response:

```json
{
  "status": "reset"
}
```

## GET /testing/debug formato

`GET /testing/debug` deve retornar dump completo da store:

```json
{
  "agents": {},
  "agent_index": {},
  "commands": {},
  "deployments": {},
  "command_reports": [],
  "desired_states": {},
  "desired_state_reports": [],
  "counters": {}
}
```

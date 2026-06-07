# 99 — Decisões e Clarificações

Este documento resolve ambiguidades encontradas nas specs.

Se houver conflito entre este documento e qualquer outro, este documento prevalece.

---

## 1. Deployment `running`

Decisão: **remover `running` como estado público de Deployment**.

Motivo: ele seria transitório demais e não agregaria valor em `/testing/deployments`.

Estados públicos finais:

```text
requested
command_created
deploying
healthy
failed
route_pending
route_active
route_failed
removed
```

---

## 2. Claim idempotente

Decisão:

```text
pending + target agent -> claimed, 200
claimed + same agent -> 200
claimed + different agent -> 409
running/succeeded/failed -> 409
```

Essa regra substitui qualquer spec anterior que diga "se status não for pending, sempre 409".

---

## 3. DEVEX_FAKE_ENVIRONMENT

Decisão: `DEVEX_FAKE_ENVIRONMENT` define o environment padrão para endpoints `/testing/*`.

Se o request informar `environment`, usar o valor do request.

Se não informar, usar `DEVEX_FAKE_ENVIRONMENT`.

Default:

```text
dev
```

---

## 4. Desired-state report desatualizado

Decisão:

```text
if reported_version == current_version:
    processar normalmente

if reported_version < current_version:
    registrar report como stale=true
    não alterar deployments
    retornar 200

if reported_version > current_version:
    registrar report como invalid/future
    não alterar deployments
    retornar 409
```

---

## 5. Stop, remove e cleanup-draining

### STOP_APPLICATION

- Criado via `/testing/commands/stop`.
- Gera command pending.
- Não altera deployment imediatamente.
- Quando report `succeeded`, deployment -> `removed`.
- Não altera desired state automaticamente, a menos que também remova rota em evolução futura.
- No MVP, preferir `REMOVE_DEPLOYMENT` para remover rota.

### REMOVE_DEPLOYMENT

- Criado via `/testing/commands/remove`.
- Gera command pending.
- Quando report `succeeded`, deployment -> `removed`.
- Se deployment possui rota ativa, remover rota do desired state.
- Incrementar desired state version.

### CLEANUP_DRAINING

- Criado via `/testing/commands/cleanup-draining`.
- Gera command pending.
- No MVP, não precisa alterar deployments automaticamente.
- Apenas registra report.

---

## 6. Route update

Decisão: rota é única por:

```text
environment + host
```

Novo deploy para o mesmo host atualiza a rota existente.

Não criar rota duplicada.

---

## 7. Heartbeat data

Decisão: não descartar os campos do heartbeat.

Armazenar payload completo em:

```text
Agent.LastHeartbeat
```

Também atualizar:

```text
Agent.Status
Agent.LastSeenAt
Agent.Version
Agent.PrivateIP, se enviado
```

---

## 8. GET /testing/commands com filtros

Implementar query params opcionais:

```text
status
type
agent_id
```

Adicionar testes para esses filtros.

---

## 9. Fake Worker

Fake worker testa `requires_route=false`.

Fluxo esperado:

```text
deploy worker
Runtime Agent report succeeded
Deployment -> healthy
Desired state não muda
Gateway Agent não recebe nova rota
```

---

## 10. /testing/reset

Reset deve limpar tudo:

```text
agents
agent index
commands
deployments
reports
desired states
desired state reports
counters
```

Após reset, IDs voltam a `001`.

---

## 11. runtime_private_ip vs DEVEX_FAKE_UPSTREAM_HOST

Regra final:

```text
if DEVEX_FAKE_UPSTREAM_HOST != "":
    upstream = DEVEX_FAKE_UPSTREAM_HOST + ":" + host_port
else:
    upstream = runtime_private_ip + ":" + host_port
```

---

## 12. container_name

Se `container_name` não vier em `/testing/commands/deploy`:

```text
container_name = {application}-{environment}-{deployment_id}
```

Exemplo:

```text
billing-api-dev-dep-001
```

---

## 13. GET /testing/desired-state sem environment

Decisão: se `GET /testing/desired-state` for chamado sem query param `environment`, retornar todos os desired states agrupados por environment.

Exemplo:

```json
{
  "desired_states": {
    "dev": {
      "version": 2,
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

Se `environment=dev` for informado, retornar apenas o desired state daquele environment:

```json
{
  "version": 2,
  "type": "gateway_routes",
  "environment": "dev",
  "routes": []
}
```

---

## 14. GET /testing/debug

Decisão: `GET /testing/debug` deve retornar um dump completo da store em memória, em JSON, com finalidade exclusivamente diagnóstica.

Formato esperado:

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

O formato pode seguir diretamente os modelos internos da store, desde que seja JSON válido e legível.

---

## 15. /health e autenticação

Decisão: `GET /health` é sempre público.

Ele deve ignorar autenticação, mesmo quando `DEVEX_FAKE_AUTH_ENABLED=true`.

Resposta esperada:

```json
{
  "status": "ok"
}
```

---

## 16. Labels automáticas

Decisão: a Fake API deve sempre gerar labels padrão no payload de `DEPLOY_APPLICATION`.

Labels obrigatórias:

```text
devex.managed=true
devex.application={application}
devex.environment={environment}
devex.deployment_id={deployment_id}
devex.command_id={command_id}
```

Exemplo:

```json
{
  "labels": {
    "devex.managed": "true",
    "devex.application": "billing-api",
    "devex.environment": "dev",
    "devex.deployment_id": "dep-001",
    "devex.command_id": "cmd-001"
  }
}
```

Se o request de `/testing/commands/deploy` trouxer labels customizadas no futuro, elas podem ser mescladas, mas as labels obrigatórias não podem ser sobrescritas.

---

## 17. container_internal_port igual a zero

Decisão: a Fake API deve aceitar `container_internal_port=0`.

Esse valor significa que o workload não precisa publicar porta.

Uso típico:

```text
worker
requires_route=false
container_internal_port=0
```

A Fake API apenas repassa esse valor no payload do comando. A interpretação operacional é responsabilidade do Runtime Agent.

---

## 18. Command.Payload

Decisão: o modelo `Command` deve usar `json.RawMessage` para o campo `Payload`.

Motivo:

- `DEPLOY_APPLICATION`, `STOP_APPLICATION`, `REMOVE_DEPLOYMENT` e `CLEANUP_DRAINING` têm payloads diferentes.
- `json.RawMessage` evita uma struct gigante cheia de campos opcionais.
- Handlers `/testing/*` podem usar structs tipadas para construir os payloads.
- O endpoint `GET /commands/pending` retorna o payload exatamente como foi montado.

Modelo recomendado:

```go
type Command struct {
    ID            string          `json:"id"`
    Type          string          `json:"type"`
    DeploymentID  string          `json:"deployment_id,omitempty"`
    TargetAgentID string          `json:"target_agent_id"`
    Status        string          `json:"status"`
    TimeoutSeconds int            `json:"timeout_seconds"`
    Payload       json.RawMessage `json:"payload"`
    CreatedAt     time.Time       `json:"created_at"`
    ClaimedAt      *time.Time      `json:"claimed_at,omitempty"`
    StartedAt      *time.Time      `json:"started_at,omitempty"`
    FinishedAt     *time.Time      `json:"finished_at,omitempty"`
    ClaimedBy      string          `json:"claimed_by,omitempty"`
}
```

Payloads específicos devem ter structs próprias, por exemplo:

```go
type DeployApplicationPayload struct {
    Application           string            `json:"application"`
    Environment           string            `json:"environment"`
    Image                 string            `json:"image"`
    ContainerName         string            `json:"container_name"`
    ContainerInternalPort int               `json:"container_internal_port"`
    HealthCheckPath       string            `json:"health_check_path"`
    RequiresRoute         bool              `json:"requires_route"`
    EnvironmentVariables  map[string]string `json:"environment_variables,omitempty"`
    Labels                map[string]string `json:"labels,omitempty"`
}
```

---

## 19. STOP_APPLICATION report e deployment_id

Decisão: para `STOP_APPLICATION`, o comando deve conter `deployment_id` tanto no campo raiz `Command.DeploymentID` quanto no payload.

Se o Runtime Agent reportar sem `deployment_id` no body, a Fake API deve usar `Command.DeploymentID` como fallback.

Regra:

```text
effective_deployment_id = report.deployment_id
if report.deployment_id == "":
    effective_deployment_id = command.deployment_id
```

Isso evita falha caso o Runtime Agent não envie `deployment_id` no report de STOP/REMOVE.

---

## 20. CleanupDrainingPayload

Decisão: formalizar `CleanupDrainingPayload` como struct tipada, equivalente às outras três structs de payload.

```go
type CleanupDrainingPayload struct {
    Environment      string `json:"environment"`
    OlderThanSeconds int    `json:"older_than_seconds"`
}
```

Regras:

- `environment` é opcional no request `/testing/commands/cleanup-draining`.
- Se `environment` vier vazio, usar `DEVEX_FAKE_ENVIRONMENT`.
- `older_than_seconds` tem default `300` se não informado ou se vier zero.
- O command criado tem type `CLEANUP_DRAINING`.
- No MVP, o report de CLEANUP_DRAINING apenas é armazenado; não altera deployments automaticamente.

---

## 21. result como json.RawMessage para todos os command types

Decisão: o campo `result` de `CommandReport` é armazenado como `json.RawMessage`.

A Fake API não valida o conteúdo de `result` para comandos diferentes de `DEPLOY_APPLICATION`.

Transições de estado por `command.type` e `status`:

| command.type | status | Ação |
|---|---|---|
| `DEPLOY_APPLICATION` | `succeeded` | Regras completas de route/desired state |
| `STOP_APPLICATION` | `succeeded` | Deployment efetivo → `removed` |
| `REMOVE_DEPLOYMENT` | `succeeded` | Deployment efetivo → `removed`; se rota associada, remover rota + incrementar desired state |
| `CLEANUP_DRAINING` | `succeeded` | Apenas armazenar report (MVP) |
| qualquer | `failed` | Armazenar erro; não alterar desired state |

O deployment efetivo é resolvido por:

```text
effective_deployment_id = report.deployment_id
if report.deployment_id == "":
    effective_deployment_id = command.deployment_id
```

---

## 22. TimeoutSeconds padrão por command type

Decisão: aplicar defaults quando `timeout_seconds` não for informado ou vier zero.

| Command Type | Default |
|---|---|
| `DEPLOY_APPLICATION` | 600s |
| `STOP_APPLICATION` | 60s |
| `REMOVE_DEPLOYMENT` | 120s |
| `CLEANUP_DRAINING` | 300s |

Regra de precedência:

```text
if request.timeout_seconds > 0:
    command.timeout_seconds = request.timeout_seconds
else:
    command.timeout_seconds = default_por_type
```

O campo `timeout_seconds` deve sempre aparecer no command retornado por `GET /commands/pending`.

---

## 23. GET /testing/desired-state — dois formatos intencionais

Decisão: o endpoint `GET /testing/desired-state` retorna formatos diferentes conforme a presença ou ausência do query param `environment`.

**Com `?environment=dev`:**

```json
{
  "version": 1,
  "type": "gateway_routes",
  "environment": "dev",
  "routes": []
}
```

**Sem `environment`:**

```json
{
  "desired_states": {
    "dev": { "version": 1, "type": "gateway_routes", "environment": "dev", "routes": [] },
    "stage": { "version": 1, "type": "gateway_routes", "environment": "stage", "routes": [] }
  }
}
```

Esse comportamento diferente na mesma rota é intencional: facilita tanto a inspeção de um environment específico quanto a visão global do estado de todos os environments.

---

## 24. GET /testing/desired-state com environment inexistente

Decisão: se `GET /testing/desired-state?environment=X` for chamado e nenhum desired state existir para esse environment, retornar HTTP 200 com desired state vazio.

```json
{
  "version": 0,
  "type": "gateway_routes",
  "environment": "X",
  "routes": []
}
```

Não retornar 404.

Motivo:
- Facilita inspeção manual antes de qualquer deploy.
- Mantém consistência com `GET /api/agents/{agent_id}/desired-state`, que também retorna estado vazio para environment sem rotas.
- Ausência de rota não é condição de erro.

---

## 25. Pronto para implementação

Com estas decisões (1–25), a documentação está considerada pronta para implementação.

Novas ambiguidades devem ser resolvidas preferencialmente em `99-decisions-and-clarifications.md`.

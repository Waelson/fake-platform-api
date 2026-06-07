# 04 — Modelo de Estado

## Store

```go
type Store struct {
    mu sync.RWMutex

    Agents map[string]*Agent
    AgentIndex map[string]string

    Commands map[string]*Command
    Deployments map[string]*Deployment

    CommandReports []CommandReport

    DesiredStates map[string]*DesiredState
    DesiredStateReports []DesiredStateReport

    Counters Counters
}
```

---

## Counters

```go
type Counters struct {
    AgentByEnvironmentRole map[string]int
    Command int
    Deployment int
    Route int
    DesiredStateVersionByEnvironment map[string]int
}
```

Reset limpa counters.

---

## Agent

```json
{
  "id": "agent-dev-api-001",
  "mode": "runtime",
  "environment": "dev",
  "role": "api",
  "hostname": "local-runtime",
  "instance_id": "local-runtime-api-001",
  "private_ip": "127.0.0.1",
  "public_ip": null,
  "version": "0.1.0",
  "status": "online",
  "capabilities": {},
  "last_heartbeat": {},
  "registered_at": "2026-06-05T18:00:00Z",
  "last_seen_at": "2026-06-05T18:00:00Z"
}
```

`last_heartbeat` armazena o payload completo mais recente de heartbeat.

---

## AgentIndex

Chave:

```text
{instance_id}:{mode}:{environment}:{role}
```

Valor:

```text
agent_id
```

---

## Command

```json
{
  "id": "cmd-001",
  "type": "DEPLOY_APPLICATION",
  "deployment_id": "dep-001",
  "target_agent_id": "agent-dev-api-001",
  "status": "pending",
  "timeout_seconds": 600,
  "payload": {},
  "created_at": "2026-06-05T18:00:00Z",
  "claimed_at": null,
  "started_at": null,
  "finished_at": null,
  "claimed_by": ""
}
```

---

## Deployment

Estados públicos:

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

Não existe `running` como estado público.

```json
{
  "id": "dep-001",
  "application": "billing-api",
  "environment": "dev",
  "image": "fake-api:v1",
  "host": "billing-api.dev.useclarus.local",
  "status": "command_created",
  "target_agent_id": "agent-dev-api-001",
  "command_id": "cmd-001",
  "container_name": "billing-api-dev-dep-001",
  "container_internal_port": 3000,
  "health_check_path": "/health",
  "requires_route": true,
  "runtime_private_ip": "",
  "host_port": 0,
  "route_id": "",
  "created_at": "2026-06-05T18:00:00Z",
  "updated_at": "2026-06-05T18:00:00Z"
}
```

---

## Route

Rota é única por:

```text
environment + host
```

Update v2 deve atualizar rota existente.

```json
{
  "id": "route-001",
  "environment": "dev",
  "host": "billing-api.dev.useclarus.local",
  "path": "/",
  "upstream": "host.docker.internal:4100",
  "deployment_id": "dep-001",
  "health_check_path": "/health"
}
```

---

## DesiredState

Um por environment.

```json
{
  "version": 1,
  "type": "gateway_routes",
  "environment": "dev",
  "routes": []
}
```

---

## CommandReport

```json
{
  "command_id": "cmd-001",
  "deployment_id": "dep-001",
  "agent_id": "agent-dev-api-001",
  "status": "succeeded",
  "result": {},
  "error": null,
  "received_at": "2026-06-05T18:00:00Z"
}
```

---

## DesiredStateReport

```json
{
  "agent_id": "agent-dev-gateway-001",
  "desired_state_version": 1,
  "current_desired_state_version": 2,
  "stale": true,
  "environment": "dev",
  "type": "gateway_routes",
  "status": "applied",
  "routes_total": 1,
  "validated_routes": 1,
  "failed_routes": 0,
  "error": null,
  "received_at": "2026-06-05T18:00:30Z"
}
```

---

## Command.Payload como json.RawMessage

O campo `payload` de `Command` deve ser flexível.

Implementação recomendada:

```go
Payload json.RawMessage `json:"payload"`
```

Motivo:

- Cada command type possui payload diferente.
- Evita struct única com muitos campos opcionais.
- Mantém fidelidade ao contrato retornado aos agents.

Payloads específicos devem ser modelados separadamente para criação e validação.

Structs de payload:

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

```go
type StopApplicationPayload struct {
    DeploymentID       string `json:"deployment_id"`
    ContainerName      string `json:"container_name"`
    StopTimeoutSeconds int    `json:"stop_timeout_seconds"`
}
```

```go
type RemoveDeploymentPayload struct {
    DeploymentID  string `json:"deployment_id"`
    ContainerName string `json:"container_name"`
    ReleasePort   bool   `json:"release_port"`
}
```

```go
type CleanupDrainingPayload struct {
    Environment      string `json:"environment"`
    OlderThanSeconds int    `json:"older_than_seconds"`
}
```

---

## TimeoutSeconds padrão por tipo de comando

| Command Type | Default (segundos) |
|---|---|
| `DEPLOY_APPLICATION` | 600 |
| `STOP_APPLICATION` | 60 |
| `REMOVE_DEPLOYMENT` | 120 |
| `CLEANUP_DRAINING` | 300 |

Regras:

- Se o request de criação trouxer `timeout_seconds > 0`, usar o valor informado.
- Se não trouxer ou vier zero, aplicar o default por tipo.
- O campo `timeout_seconds` deve sempre aparecer no comando retornado por `GET /api/agents/{agent_id}/commands/pending`.

---

## CommandReport — result como json.RawMessage

O campo `result` de `CommandReport` deve ser armazenado como JSON flexível:

```go
type CommandReport struct {
    CommandID    string          `json:"command_id"`
    DeploymentID string          `json:"deployment_id"`
    AgentID      string          `json:"agent_id"`
    Status       string          `json:"status"`
    Result       json.RawMessage `json:"result,omitempty"`
    Error        *ReportError    `json:"error,omitempty"`
    ReceivedAt   time.Time       `json:"received_at"`
}
```

A Fake API não valida o conteúdo de `result` para comandos diferentes de `DEPLOY_APPLICATION`. Apenas armazena e aplica as transições de estado baseadas em `command.type`.

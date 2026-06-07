# 11 — Roadmap de Implementação

## Milestone 0 — Fundação

- go.mod
- Makefile
- .gitignore
- Dockerfile
- estrutura de diretórios
- GET /health
- go test ./...

## Milestone 1 — Config e modelos

- config por env vars
- response helpers
- modelos de dados completos
- error response padrão
- Counters

## Milestone 2 — Store

- store com RWMutex
- register/upsert agent
- heartbeat com LastHeartbeat
- ID generation
- deployment creation
- command lifecycle
- deployment lifecycle
- desired state por environment

## Milestone 3 — API oficial

- register
- heartbeat
- pending
- claim
- start
- report
- desired-state
- desired-state/report

## Milestone 4 — Testing endpoints

- deploy
- stop
- remove
- cleanup-draining
- agents
- commands com filtros
- reports
- deployments
- desired-state
- debug
- reset

## Milestone 5 — Fake App/Worker

- fake-app
- fake-worker
- Dockerfiles
- scripts build images

## Milestone 6 — Compose e scripts

- docker-compose.dev.yml
- dev scripts
- Makefile targets

## Milestone 7 — Testes

- unit tests
- handler tests
- fluxo completo simulado

---

## Decisão de implementação para Command.Payload

Implementar `Command.Payload` como `json.RawMessage`.

Durante os milestones, criar structs específicas para payloads:

```text
DeployApplicationPayload
StopApplicationPayload
RemoveDeploymentPayload
CleanupDrainingPayload
```

Handlers de `/testing/*` devem montar payloads tipados e serializar para `json.RawMessage` antes de salvar no command.

`CleanupDrainingPayload`:

```go
type CleanupDrainingPayload struct {
    Environment      string `json:"environment"`
    OlderThanSeconds int    `json:"older_than_seconds"`
}
```

Defaults: `environment = DEVEX_FAKE_ENVIRONMENT` se vazio; `older_than_seconds = 300` se zero.

---

## TimeoutSeconds padrão por tipo de comando

Aplicar ao criar o command quando o request não informar `timeout_seconds`:

| Command Type | Default |
|---|---|
| `DEPLOY_APPLICATION` | 600s |
| `STOP_APPLICATION` | 60s |
| `REMOVE_DEPLOYMENT` | 120s |
| `CLEANUP_DRAINING` | 300s |

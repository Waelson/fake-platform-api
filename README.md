# Fake DevEx Platform API

Aplicação local para testar o `devex-agent` end-to-end.

Ela simula a Platform API esperada pelo Runtime Agent e pelo Gateway Agent.

---

## Fluxo testado

```text
Fake Platform API
  -> Runtime Agent real
  -> Docker real
  -> Gateway Agent real
  -> Caddy real
  -> Fake App / Fake Worker
```

---

## Endpoints oficiais

```http
POST /api/agents/register
POST /api/agents/{agent_id}/heartbeat
GET  /api/agents/{agent_id}/commands/pending
POST /api/agents/{agent_id}/commands/{command_id}/claim
POST /api/agents/{agent_id}/commands/{command_id}/start
POST /api/agents/{agent_id}/commands/{command_id}/report
GET  /api/agents/{agent_id}/desired-state
POST /api/agents/{agent_id}/desired-state/report
```

---

## Endpoints de teste

```http
POST /testing/commands/deploy
POST /testing/commands/stop
POST /testing/commands/remove
POST /testing/commands/cleanup-draining
GET  /testing/agents
GET  /testing/commands
GET  /testing/reports
GET  /testing/deployments
GET  /testing/desired-state
GET  /testing/desired-state/reports
GET  /testing/debug
POST /testing/reset
```

---

## Configuração

```text
DEVEX_FAKE_PORT=8080
DEVEX_FAKE_AUTH_ENABLED=false
DEVEX_FAKE_TOKEN=dev-token
DEVEX_FAKE_ENVIRONMENT=dev
DEVEX_FAKE_UPSTREAM_HOST=host.docker.internal
DEVEX_FAKE_STATE_FILE=
DEVEX_FAKE_STATE_SAVE_INTERVAL_SECONDS=2
```

### DEVEX_FAKE_ENVIRONMENT

Define o environment padrão usado pelos endpoints `/testing/*` quando o request não informar `environment`.

### DEVEX_FAKE_STATE_FILE / DEVEX_FAKE_STATE_SAVE_INTERVAL_SECONDS

Persistência opcional em arquivo, para sobreviver a restarts do processo/container
sem precisar de banco de dados:

- `DEVEX_FAKE_STATE_FILE`: caminho de um snapshot JSON (ex.: `/data/state.json`).
  Vazio (default) = a API roda 100% em memória, igual ao MVP original.
- `DEVEX_FAKE_STATE_SAVE_INTERVAL_SECONDS`: intervalo do snapshot periódico em
  segundos (default `2`).

Quando habilitada, o estado (agents, commands, deployments, desired states,
reports e counters) é restaurado no boot, salvo periodicamente e ao receber
SIGTERM/SIGINT, e removido por `POST /testing/reset`. Veja
`docs/specs/04-state-model.md` e `docs/specs/09-docker-compose-dev.md` para
detalhes.

### DEVEX_FAKE_UPSTREAM_HOST

Controla o host usado no upstream enviado ao Caddy pelo desired state.

Mac/Docker Desktop:

```text
host.docker.internal
```

Linux local:

```text
127.0.0.1
```

EC2:

```text
IP privado da instância
```

---

## Exemplo de deploy API

```bash
curl -X POST http://127.0.0.1:8080/testing/commands/deploy \
  -H "Content-Type: application/json" \
  -d '{
    "target_agent_role": "api",
    "application": "billing-api",
    "environment": "dev",
    "image": "fake-api:v1",
    "container_internal_port": 3000,
    "health_check_path": "/health",
    "requires_route": true,
    "host": "billing-api.dev.useclarus.local"
  }'
```

---

## Exemplo de worker

```bash
curl -X POST http://127.0.0.1:8080/testing/commands/deploy \
  -H "Content-Type: application/json" \
  -d '{
    "target_agent_role": "worker",
    "application": "invoice-worker",
    "environment": "dev",
    "image": "fake-worker:v1",
    "container_internal_port": 0,
    "requires_route": false
  }'
```

---

## Teste de rota

```bash
curl -H "Host: billing-api.dev.useclarus.local" http://127.0.0.1/health
```

---

## Decisões finais

Leia:

```text
docs/specs/99-decisions-and-clarifications.md
```

---

## Decisões finais de implementação

- `GET /health` é público.
- `GET /testing/desired-state` sem `environment` retorna todos os environments.
- `GET /testing/debug` retorna dump completo da store.
- Commands usam `payload` flexível via `json.RawMessage`.
- Labels DevEx são geradas automaticamente em deploys.
- `container_internal_port=0` é válido para workers sem rota.

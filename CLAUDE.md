# CLAUDE.md

## Projeto: Fake DevEx Platform API

Este projeto implementa uma **Fake DevEx Platform API** para testar o `devex-agent` em ambiente local/desenvolvimento.

A aplicação deve simular os contratos esperados por:

- Runtime Agent
- Gateway Agent

O objetivo é permitir testes end-to-end com:

```text
Fake Platform API
  -> Runtime Agent real
  -> Docker real
  -> Gateway Agent real
  -> Caddy real
  -> Fake App
```

---

## Stack

Implementar em **Go**.

Preferência:

```text
Go 1.22+
net/http ou github.com/go-chi/chi/v5
encoding/json
sync.RWMutex
httptest
```

---

## Escopo do MVP

Implementar:

- Config por env vars.
- Store em memória.
- Registro/upsert de agents.
- Heartbeat.
- Command lifecycle.
- Deployment lifecycle simplificado e explícito.
- Desired state de `gateway_routes` por environment.
- Endpoints oficiais `/api/*`.
- Endpoints auxiliares `/testing/*`.
- Fake app HTTP.
- Fake worker.
- Docker Compose local.
- Makefile e scripts.
- Testes unitários e de handlers.

---

## Decisões importantes

As decisões de arquitetura e contrato estão consolidadas em:

```text
docs/specs/99-decisions-and-clarifications.md
```

Se houver conflito entre qualquer spec e esse arquivo, **prevalece `99-decisions-and-clarifications.md`**.

---

## Documentos obrigatórios

Leia antes de implementar:

```text
docs/specs/00-product-overview.md
docs/specs/01-architecture.md
docs/specs/02-api-contracts.md
docs/specs/03-testing-endpoints.md
docs/specs/04-state-model.md
docs/specs/05-deployment-simulation-flow.md
docs/specs/06-command-and-deployment-lifecycle.md
docs/specs/07-authentication.md
docs/specs/08-fake-app-and-worker.md
docs/specs/09-docker-compose-dev.md
docs/specs/10-testing-strategy.md
docs/specs/11-implementation-roadmap.md
docs/specs/99-decisions-and-clarifications.md
```

---

## Estrutura esperada

```text
fake-platform-api/
├── CLAUDE.md
├── README.md
├── Makefile
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.dev.yml
├── cmd/
│   └── fake-platform-api/
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── router.go
│   │   ├── middleware.go
│   │   ├── handlers_agents.go
│   │   ├── handlers_commands.go
│   │   ├── handlers_desired_state.go
│   │   └── handlers_testing.go
│   ├── store/
│   │   ├── store.go
│   │   ├── models.go
│   │   ├── agents.go
│   │   ├── commands.go
│   │   ├── deployments.go
│   │   ├── desired_state.go
│   │   └── testing.go
│   ├── config/
│   │   └── config.go
│   ├── response/
│   │   └── response.go
│   └── ids/
│       └── ids.go
├── test/
│   ├── fake-app/
│   │   ├── main.go
│   │   └── Dockerfile
│   └── fake-worker/
│       ├── main.go
│       └── Dockerfile
├── scripts/
│   ├── dev-build-images.sh
│   ├── dev-up.sh
│   ├── dev-down.sh
│   ├── dev-deploy-v1.sh
│   ├── dev-deploy-v2.sh
│   ├── dev-deploy-broken.sh
│   ├── dev-worker.sh
│   ├── dev-test-route.sh
│   └── dev-reset.sh
└── docs/
    └── specs/
```

---

## Regras essenciais

1. A Fake API não executa Docker.
2. A Fake API não chama Caddy.
3. Runtime Agent e Gateway Agent reais executam as ações reais.
4. Claim deve ser atômico.
5. Claim repetido pelo mesmo agent deve retornar 200 idempotente.
6. Claim por agent diferente deve retornar 409.
7. Desired state deve ser separado por environment.
8. Desired-state report antigo deve ser registrado, mas não deve alterar deployments.
9. Deployment `running` foi removido do lifecycle público.
10. Heartbeat data deve ser armazenado em `Agent.LastHeartbeat`.
11. `DEVEX_FAKE_ENVIRONMENT` define o environment padrão para endpoints de teste quando request não informar environment.
12. `container_name` deve ser gerado no formato `{application}-{environment}-{deployment_id}` se não vier no request.
13. Para route update, procurar rota existente por `{environment, host}` e atualizar, não duplicar.
14. `/testing/reset` limpa tudo, inclusive counters.

---

## Primeira tarefa do Claude Code

Antes de codar:

```text
1. Leia CLAUDE.md.
2. Leia todos os arquivos em docs/specs.
3. Explique as decisões consolidadas em 99-decisions-and-clarifications.md.
4. Liste todos os endpoints.
5. Liste todos os modelos de dados.
6. Liste regras de lifecycle.
7. Proponha plano de implementação por milestone.
8. Aguarde aprovação.
```

Não implemente código antes da aprovação.

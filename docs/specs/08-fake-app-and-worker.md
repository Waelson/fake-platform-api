# 08 — Fake App e Fake Worker

## Fake App

Endpoints:

```http
GET /
GET /health
GET /version
GET /env
```

Variáveis:

```text
APP_NAME
APP_VERSION
APP_MODE
PORT
```

Modos:

- `healthy`: `/health` retorna 200.
- `broken`: `/health` retorna 500.
- `slow`: `/health` demora para responder.

Imagens:

```text
fake-api:v1
fake-api:v2
fake-api:broken
fake-api:slow
```

---

## Fake Worker

Objetivo: testar `requires_route=false`.

O fake worker deve ser um processo simples que:

- inicia
- escreve logs periodicamente
- não expõe rota pública
- pode opcionalmente expor `/health` se o agent exigir container_internal_port

Imagem:

```text
fake-worker:v1
```

Endpoint de deploy recomendado:

```json
{
  "target_agent_role": "worker",
  "application": "invoice-worker",
  "environment": "dev",
  "image": "fake-worker:v1",
  "container_internal_port": 0,
  "requires_route": false
}
```

Resultado esperado:

- Runtime Agent executa container.
- Report succeeded.
- Deployment fica `healthy`.
- Desired state não muda.
- Gateway Agent não recebe nova rota.

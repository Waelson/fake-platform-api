# 07 — Autenticação

## Configuração

```text
DEVEX_FAKE_AUTH_ENABLED=false
DEVEX_FAKE_TOKEN=dev-token
```

## Auth disabled

Se `DEVEX_FAKE_AUTH_ENABLED=false`:

- Ignorar `Authorization`.
- Aceitar todos os requests.

## Auth enabled

Se `DEVEX_FAKE_AUTH_ENABLED=true`:

Endpoints `/api/*` exigem:

```http
Authorization: Bearer {DEVEX_FAKE_TOKEN}
```

## Falha

```http
HTTP 401
```

```json
{
  "error": {
    "code": "AUTHENTICATION_FAILED",
    "message": "Invalid or missing token",
    "retryable": false
  }
}
```

## /testing/*

No MVP, `/testing/*` ignora autenticação.

---

## GET /health

`GET /health` é sempre público.

Mesmo com `DEVEX_FAKE_AUTH_ENABLED=true`, esse endpoint não deve exigir token.

# 01 — Arquitetura

## Visão geral

```text
Fake Platform API :8080
        ↑
        │ HTTP
        ↓
Runtime Agent real        Gateway Agent real
        │                         │
        ▼                         ▼
Docker real                 Caddy real
        │                         │
        ▼                         ▼
Fake App / Worker      Reverse Proxy
```

---

## Store em memória

A store é única, em memória, thread-safe.

Usar `sync.RWMutex`.

---

## Separação por environment

Agents, deployments, routes e desired state devem considerar `environment`.

O desired state retornado ao Gateway Agent deve ser filtrado por `agent.environment`.

---

## Re-registro

Agent re-register é upsert por:

```text
instance_id + mode + environment + role
```

Preserva `agent_id`.

---

## Heartbeat

Heartbeat atualiza:

- `last_seen_at`
- `status`
- `last_heartbeat`

Os campos do heartbeat **não devem ser descartados**. Devem ser armazenados em `Agent.LastHeartbeat` como objeto flexível.

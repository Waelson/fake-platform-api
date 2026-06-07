# 05 — Fluxo de Simulação de Deploy

## Deploy com rota

```text
1. POST /testing/commands/deploy.
2. Fake API escolhe Runtime Agent.
3. Deployment = requested.
4. Command = pending.
5. Deployment = command_created.
6. Runtime Agent busca command.
7. Runtime Agent claim.
8. Command = claimed.
9. Runtime Agent start.
10. Command = running.
11. Deployment = deploying.
12. Runtime Agent executa Docker.
13. Runtime Agent report succeeded.
14. Command = succeeded.
15. Deployment = healthy.
16. Fake API cria/atualiza rota.
17. Deployment = route_pending.
18. DesiredState.version++.
19. Gateway Agent busca desired state.
20. Gateway Agent report applied da versão atual.
21. Deployment = route_active.
```

---

## Deploy sem rota

```text
1. Report succeeded.
2. Command = succeeded.
3. Deployment = healthy.
4. Não cria rota.
5. Não incrementa desired state.
```

---

## Deploy failed

```text
1. Runtime Agent report failed.
2. Command = failed.
3. Deployment = failed.
4. Não altera rota.
5. Não incrementa desired state.
```

---

## Update v2

Para mesmo `environment + host`:

```text
1. Criar novo deployment.
2. Report succeeded.
3. Procurar rota existente por environment+host.
4. Atualizar upstream e deployment_id da rota.
5. Incrementar desired state version.
```

Não duplicar rota.

---

## Desired-state report desatualizado

Se current version é 3 e Gateway reporta version 1:

```text
1. Registrar report como stale=true.
2. Não alterar deployments.
3. Retornar 200 accepted.
```

Se Gateway reporta version 5 e current é 3:

```text
1. Registrar report como invalid/future version.
2. Retornar 409.
3. Não alterar deployments.
```

# 06 — Lifecycle de Commands e Deployments

## Command states

```text
pending
claimed
running
succeeded
failed
cancelled
expired
```

## Command transitions

```text
pending -> claimed
claimed -> running
running -> succeeded
running -> failed
pending -> cancelled
pending -> expired
```

## Claim idempotente

```text
pending + same/target agent -> claimed, 200
claimed + same agent -> 200
claimed + different agent -> 409
running/succeeded/failed -> 409
```

---

## Deployment states públicos

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

`running` não é estado público. Foi removido para evitar estado transitório sem utilidade externa.

---

## Deployment transitions

```text
requested -> command_created
command_created -> deploying
deploying -> healthy
deploying -> failed
healthy -> route_pending
route_pending -> route_active
route_pending -> route_failed
healthy -> removed
route_active -> removed
route_failed -> removed
failed -> removed
```

---

## Evento: create deploy

```text
Deployment: requested -> command_created
Command: pending
```

## Evento: command start

```text
Command: claimed -> running
Deployment: command_created -> deploying
```

## Evento: command report succeeded requires_route=false

```text
Command: running -> succeeded
Deployment: deploying -> healthy
```

## Evento: command report succeeded requires_route=true

```text
Command: running -> succeeded
Deployment: deploying -> healthy -> route_pending
```

## Evento: command report failed

```text
Command: running -> failed
Deployment: deploying -> failed
```

## Evento: desired-state report applied current version

```text
Deployment: route_pending -> route_active
```

## Evento: desired-state report failed current version

```text
Deployment: route_pending -> route_failed
```

## Evento: remove succeeded

```text
Deployment: healthy/route_active/route_failed/failed -> removed
```

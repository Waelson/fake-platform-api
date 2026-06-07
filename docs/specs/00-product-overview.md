# 00 — Visão Geral do Produto

## Objetivo

A **Fake DevEx Platform API** é um simulador local da DevEx Platform API real.

Ela existe para permitir testes end-to-end do `devex-agent`.

---

## Cenários que deve suportar

- Deploy inicial de API com rota.
- Update de API v1 -> v2 com atualização de rota existente.
- Deploy com health check falhando sem alterar desired state.
- Deploy de worker sem rota.
- Gateway Agent aplicando desired state.
- Gateway Agent reportando versão aplicada.
- Desired-state report desatualizado sem alterar deployments.
- Stop/remove/cleanup como comandos simulados.
- Reset completo do ambiente de teste.

---

## Princípio

A Fake API simula a plataforma, mas não executa infraestrutura.

```text
Fake API cria comandos e desired state.
Runtime Agent executa Docker.
Gateway Agent executa Caddy.
```

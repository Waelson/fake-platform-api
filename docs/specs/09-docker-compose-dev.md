# 09 — Docker Compose de Desenvolvimento

## docker-compose.dev.yml

```yaml
services:
  fake-platform-api:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: fake-platform-api
    ports:
      - "8080:8080"
    environment:
      DEVEX_FAKE_PORT: "8080"
      DEVEX_FAKE_AUTH_ENABLED: "false"
      DEVEX_FAKE_TOKEN: "dev-token"
      DEVEX_FAKE_ENVIRONMENT: "dev"
      DEVEX_FAKE_STATE_FILE: "/data/state.json"
      DEVEX_FAKE_STATE_SAVE_INTERVAL_SECONDS: "2"
    volumes:
      - ./tmp/fake-platform-api-data:/data

  caddy:
    image: caddy:latest
    container_name: caddy-devex-test
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
      - "443:443/udp"
      - "127.0.0.1:2019:2019"
    volumes:
      - ./tmp/caddy-data:/data
      - ./tmp/caddy-config:/config
```

## DEVEX_FAKE_ENVIRONMENT

Define environment padrão para `/testing/*`.

Se request informar `environment`, o request vence.

Se não informar, usar `DEVEX_FAKE_ENVIRONMENT`.

## Persistência opcional em arquivo (DEVEX_FAKE_STATE_FILE)

Por padrão a Fake API mantém todo o estado apenas em memória (perdido a cada
restart). Definir `DEVEX_FAKE_STATE_FILE` ativa persistência opt-in em um
snapshot JSON local — sem banco de dados:

- `DEVEX_FAKE_STATE_FILE`: caminho do arquivo de snapshot (ex.: `/data/state.json`).
  Vazio (default) = persistência desabilitada, comportamento 100% em memória.
- `DEVEX_FAKE_STATE_SAVE_INTERVAL_SECONDS`: intervalo do snapshot periódico em
  segundos (default `2`).

Quando habilitada:

- Na inicialização, se o arquivo existir, o estado é restaurado.
- Uma goroutine salva o snapshot periodicamente, e um save final é feito no
  shutdown gracioso (SIGTERM/SIGINT).
- A escrita é atômica (`*.tmp` + `rename`) para não corromper o arquivo em
  caso de crash durante a gravação.
- `POST /testing/reset` remove o arquivo de estado, para que um restart
  subsequente não restaure dados anteriores ao reset.

No `docker-compose.dev.yml` isso é viabilizado por um volume:

```yaml
    environment:
      DEVEX_FAKE_STATE_FILE: "/data/state.json"
      DEVEX_FAKE_STATE_SAVE_INTERVAL_SECONDS: "2"
    volumes:
      - ./tmp/fake-platform-api-data:/data
```

## Makefile targets

```text
test
run
dev-up
dev-down
dev-build-images
dev-deploy-v1
dev-deploy-v2
dev-deploy-broken
dev-worker
dev-test-route
dev-reset
```

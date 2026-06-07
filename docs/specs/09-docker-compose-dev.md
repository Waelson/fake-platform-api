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
      DEVEX_FAKE_UPSTREAM_HOST: "host.docker.internal"

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

.PHONY: build run test dev-up dev-down dev-build-images dev-deploy-v1 dev-deploy-v2 dev-deploy-broken dev-worker dev-test-route dev-reset

build:
	go build -o bin/fake-platform-api ./cmd/fake-platform-api

run:
	go run ./cmd/fake-platform-api

test:
	go test ./...

dev-up:
	docker compose -f docker-compose.dev.yml up -d

dev-down:
	docker compose -f docker-compose.dev.yml down

dev-build-images:
	./scripts/dev-build-images.sh

dev-deploy-v1:
	./scripts/dev-deploy-v1.sh

dev-deploy-v2:
	./scripts/dev-deploy-v2.sh

dev-deploy-broken:
	./scripts/dev-deploy-broken.sh

dev-worker:
	./scripts/dev-worker.sh

dev-test-route:
	./scripts/dev-test-route.sh

dev-reset:
	./scripts/dev-reset.sh

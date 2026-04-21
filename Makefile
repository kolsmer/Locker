.PHONY: help db-up db-down db-migrate db-reset run dev test build clean smoke

help:
	@echo "Available commands:"
	@echo "  make db-up      - Start PostgreSQL container"
	@echo "  make db-down    - Stop PostgreSQL container"
	@echo "  make db-migrate - Apply SQL migrations"
	@echo "  make db-reset   - Recreate DB schema from migrations"
	@echo "  make run        - Run API server (without DEBUG)"
	@echo "  make dev        - Run API server with DEBUG=1"
	@echo "  make test       - Run tests"
	@echo "  make build      - Build locker-api binary"
	@echo "  make smoke      - Run API smoke test"
	@echo "  make clean      - Remove local binaries and test artifacts"

db-up:
	docker compose up -d postgres
	@echo "PostgreSQL is running. Waiting for health check..."
	sleep 5

db-down:
	docker compose down

db-migrate:
	docker compose build migrate
	docker compose run --rm migrate

db-reset:
	docker compose down -v
	docker compose up -d postgres
	sleep 5
	docker compose build migrate
	docker compose run --rm migrate

run:
	go run ./cmd/api/main.go

dev:
	DEBUG=1 go run ./cmd/api/main.go

test:
	go test -v ./...

build:
	go build -o locker-api ./cmd/api/main.go

smoke:
	bash ./scripts/smoke-test.sh

clean:
	rm -f locker-api main
	find . -name "*.out" -delete

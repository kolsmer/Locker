.PHONY: help db-up db-down db-migrate db-reset run dev test build clean smoke obs-up obs-down k8s-build-images k8s-check k8s-up k8s-down k8s-status k8s-render

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
	@echo "  make obs-up     - Start observability stack (Prometheus/Grafana/Loki/Promtail)"
	@echo "  make obs-down   - Stop observability stack"
	@echo "  make k8s-build-images - Build Docker images used by Kubernetes manifests"
	@echo "  make k8s-check  - Verify kubectl can reach the current cluster"
	@echo "  make k8s-up     - Deploy stack to Kubernetes namespace locker"
	@echo "  make k8s-down   - Remove Kubernetes resources from namespace locker"
	@echo "  make k8s-status - Show Kubernetes resource status"
	@echo "  make k8s-render - Render Kubernetes manifests with kustomize"
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

obs-up:
	docker compose --profile observability up -d

obs-down:
	docker compose --profile observability down

k8s-build-images:
	docker build -t locker-backend -f Dockerfile.backend .
	docker build -t locker-migrate -f Dockerfile.migrate .
	docker build -t locker-frontend ./frontend

k8s-check:
	@if kubectl version --request-timeout=5s >/dev/null 2>&1; then \
		echo "kubectl cluster is reachable"; \
	else \
		echo "kubectl cannot reach the current cluster."; \
		echo "Current context: $$(kubectl config current-context 2>/dev/null || echo unknown)"; \
		echo "If you use kind with Docker Desktop, start Docker Desktop first or recreate the cluster:"; \
		echo "  kind delete cluster --name kind"; \
		echo "  kind create cluster --name kind"; \
		exit 1; \
	fi

k8s-up: k8s-check
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -n locker -f k8s/configmap.yaml -f k8s/secret.yaml -f k8s/postgres-pvc.yaml -f k8s/postgres-service.yaml -f k8s/postgres-deployment.yaml
	kubectl rollout status deployment/postgres -n locker --timeout=180s
	kubectl delete job locker-migrate -n locker --ignore-not-found=true
	kubectl apply -n locker -f k8s/migrate-job.yaml
	kubectl wait --for=condition=complete job/locker-migrate -n locker --timeout=180s
	kubectl apply -n locker -f k8s/backend-service.yaml -f k8s/backend-deployment.yaml -f k8s/frontend-service.yaml -f k8s/frontend-deployment.yaml
	kubectl rollout status deployment/backend -n locker --timeout=180s
	kubectl rollout status deployment/frontend -n locker --timeout=180s

k8s-down:
	kubectl delete -k k8s --ignore-not-found=true

k8s-status:
	kubectl get pods,svc,pvc,job -n locker

k8s-render:
	kubectl kustomize k8s

clean:
	rm -f locker-api main
	find . -name "*.out" -delete

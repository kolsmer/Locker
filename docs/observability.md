# Observability: quick start

В проект добавлены:
- структурированные JSON-логи backend
- `request_id` middleware (`X-Request-Id`)
- HTTP metrics Prometheus на `/metrics`
- readiness endpoint `/readyz`
- стек наблюдаемости в Docker Compose: Prometheus + Grafana + Loki + Promtail

## Что уже есть в backend

- `GET /healthz` — liveness
- `GET /readyz` — readiness (проверка подключения к БД)
- `GET /metrics` — Prometheus метрики
- access logs в JSON со стандартными полями:
  - `request_id`
  - `method`
  - `path`
  - `route`
  - `status`
  - `duration_ms`
  - `bytes`

## Запуск приложения

```bash
docker compose up -d --build
```

## Запуск observability-стека

Используется профиль `observability`:

```bash
docker compose --profile observability up -d
```

Сервисы:
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000
- Loki API: http://localhost:3100

Логин Grafana по умолчанию:
- user: `admin`
- password: `admin`

Можно переопределить через `.env`:
- `GRAFANA_ADMIN_USER`
- `GRAFANA_ADMIN_PASSWORD`

## Базовая проверка

```bash
curl -i http://localhost:8080/healthz
curl -i http://localhost:8080/readyz
curl -s http://localhost:8080/metrics | head
```

## Что смотреть в Grafana

1. Data Sources уже провиженятся автоматически:
   - Prometheus
   - Loki
2. Можно создать дашборд по метрикам:
   - `locker_http_requests_total`
   - `locker_http_request_duration_seconds_bucket`
3. В Explore выбрать Loki и фильтровать по label `compose_service="backend"`.

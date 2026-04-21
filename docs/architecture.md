# Locker Backend Architecture

Актуальная архитектура backend части проекта на Go + PostgreSQL.

## Current Project Layout

```
.
├── cmd/
│   └── api/main.go
├── internal/
│   ├── auth/
│   ├── config/
│   ├── domain/
│   ├── observability/
│   ├── repository/
│   ├── service/
│   └── transport/
│       ├── device/
│       └── http/
├── migrations/
├── docs/
│   └── architecture.md
├── docker-compose.yml
├── Makefile
├── go.mod
└── go.sum
```

## Runtime Entry Point

- Active entrypoint: `cmd/api/main.go`
- HTTP only (not HTTPS) by default
- App connects to an already running PostgreSQL instance

## Active API (Implemented in Runtime)

Маршруты, которые реально зарегистрированы в текущем runtime:

### Service
- `GET /`
- `GET /healthz`

### v1
- `GET /api/v1/lockers`
- `POST /api/v1/lockers/{lockerId}/cell-selection`
- `POST /api/v1/lockers/{lockerId}/bookings`
- `POST /api/v1/lockers/{lockerId}/access-code/check`
- `GET /api/v1/payments/{paymentId}`
- `POST /api/v1/rentals/{rentalId}/open`
- `POST /api/v1/rentals/{rentalId}/finish`
- `GET /api/v1/rentals/{rentalId}`

Основная реализация:
- handler: `internal/transport/http/locker_handler.go`
- service: `internal/service/rental_flow_service.go`
- repository: `internal/repository/rental_flow_repository.go`

## Domain Model (Core)

- Locker: состояние ячейки (`free`, `reserved`, `occupied`, `locked`, `maintenance`, ...)
- StorageSession/Rental: жизненный цикл аренды
- Payment: `pending`, `paid`, `failed`, `refunded`
- DeviceCommand + DeviceEvent: интеграция с постоматом
- Admin + Audit entities: админ-действия и журналирование

## Database

Основная БД: PostgreSQL.

Ключевые таблицы:
- `locations`
- `lockers`
- `storage_sessions`
- `payments`
- `admins`
- `locker_events`
- `audit_logs`
- `device_commands`
- `device_events`

Runtime таблицы пользовательского сценария:
- `cell_selections`
- `locker_rentals`
- `rental_payments`

Схема накатывается из `migrations/`.

## Startup Flow

1. Поднять Postgres (например, через `docker-compose.yml`)
2. Применить миграции (`goose`)
3. Запустить API (`go run ./cmd/api/main.go`)

Приложение не поднимает базу само, только подключается к ней.

## Layer Responsibilities

- `internal/domain`: типы, enum-статусы, инварианты домена
- `internal/repository`: SQL доступ к данным
- `internal/service`: бизнес-правила и оркестрация
- `internal/transport/http`: HTTP handlers и API envelope
- `internal/transport/device`: транспорт для postomat/device интеграции

Текущий runtime соблюдает слойность: `handler -> service -> repository`.

## Status And Next Steps

Готово:
- Единый entrypoint
- PostgreSQL-backed MVP flow
- Основные пользовательские endpoints

В процессе:
- Полная admin auth (JWT/RBAC)
- Device command polling/reporting в runtime
- Реальный payment provider webhook
- Тесты и более полная observability

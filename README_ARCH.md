# LOCK'IT Backend

Go + PostgreSQL backend для MVP системы камер хранения.

## Текущее состояние

- Рабочий entrypoint: `cmd/api/main.go`
- База данных: PostgreSQL
- Основной runtime handler: `internal/transport/http/locker_handler.go`
- Транспорт по умолчанию: HTTP (`http://localhost:8080`)

## Технологии

- Go 1.21+
- PostgreSQL
- gorilla/mux
- lib/pq
- goose (миграции)

## Структура проекта

```text
.
├── cmd/
│   └── api/main.go
├── internal/
├── migrations/
├── docs/
│   └── architecture.md
├── docker-compose.yml
├── Makefile
├── QUICKSTART.md
└── go.mod
```

## Быстрый запуск

```bash
make db-up
make db-migrate
go run ./cmd/api/main.go
```

Если `make` не используется:

```bash
docker-compose up -d postgres
goose -dir ./migrations postgres "postgres://postgres:postgres@localhost:5432/locker?sslmode=disable" up
go run ./cmd/api/main.go
```

## Активные API эндпоинты

- `GET /`
- `GET /healthz`
- `GET /api/v1/lockers`
- `POST /api/v1/lockers/{lockerId}/cell-selection`
- `POST /api/v1/lockers/{lockerId}/bookings`
- `POST /api/v1/lockers/{lockerId}/access-code/check`
- `GET /api/v1/payments/{paymentId}`
- `POST /api/v1/rentals/{rentalId}/open`
- `POST /api/v1/rentals/{rentalId}/finish`
- `GET /api/v1/rentals/{rentalId}`

## Документация

- Архитектура: `docs/architecture.md`
- Практический запуск и curl-примеры: `QUICKSTART.md`
- Контракт API: `backend-api-endpoints.md`

## Важно

- Приложение не поднимает PostgreSQL автоматически, только подключается к уже запущенной БД.
- Часть admin/device функциональности пока в статусе scaffolding и не подключена в runtime.


# Development Quick Start

## 1. Start database
```bash
docker-compose up -d postgres
```

## 2. Run migrations
```bash
goose -dir ./migrations postgres "postgres://postgres:postgres@localhost:5432/locker?sslmode=disable" up
```

## 3. Run server
```bash
go run ./cmd/api/main.go
```

## 4. Test endpoints

### Service check
```bash
curl -X GET http://localhost:8080/healthz
```

### Client: Get lockers
```bash
curl -X GET http://localhost:8080/api/v1/lockers
```

### Select cell
```bash
curl -X POST http://localhost:8080/api/v1/lockers/123/cell-selection \
  -H "Content-Type: application/json" \
  -d '{"size": "m"}'
```

### Create booking
```bash
curl -X POST http://localhost:8080/api/v1/lockers/123/bookings \
  -H "Content-Type: application/json" \
  -d '{"selectionId": "sel_xxxxxx", "phone": "+79991234567"}'
```

### Check access code
```bash
curl -X POST http://localhost:8080/api/v1/lockers/123/access-code/check \
  -H "Content-Type: application/json" \
  -d '{"accessCode": "1A2B3C"}'
```

### Payment status
```bash
curl -X GET http://localhost:8080/api/v1/payments/pay_xxxxxx
```

### Open rental
```bash
curl -X POST http://localhost:8080/api/v1/rentals/rent_xxxxxx/open
```

### Finish rental
```bash
curl -X POST http://localhost:8080/api/v1/rentals/rent_xxxxxx/finish
```

### Get rental state
```bash
curl -X GET http://localhost:8080/api/v1/rentals/rent_xxxxxx
```

## Database

### Connect to DB
```bash
psql -U postgres -d locker
```

### Check tables
```sql
\dt
SELECT * FROM lockers;
```

## Debugging

Enable debug mode:
```bash
DEBUG=1 go run ./cmd/api/main.go
```

Check active port listener:
```bash
ss -ltnp | grep ':8080' || true
```

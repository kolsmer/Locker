# Implementation TODO

Актуальный список задач после запуска PostgreSQL-backed MVP API.

## Что уже сделано

- Активный entrypoint: `cmd/api/main.go`
- Реализован runtime флоу из API-спеки в `internal/transport/http/locker_handler.go`
- Добавлены миграции `001..009`, runtime поля folded into `003/004`
- Описана архитектура в `docs/architecture.md`

## Priority 1 (блокеры для прод-подобного режима)

### 1. Security: admin auth
- [ ] Добавить `internal/auth/jwt.go` (JWT issue/validate)
- [ ] Добавить `internal/auth/password.go` (bcrypt hash/verify)
- [ ] Подключить middleware на admin endpoints

### 2. Исправить заглушки времени в service/repository
- [ ] Заменить все `nowUnix() -> 0` на `time.Now().Unix()`
- [ ] Удалить временные заглушки, влияющие на бизнес-логику

### 3. Статусы и валидация
- [ ] Добавить общий пакет валидации (`pkg/validator`)
- [ ] Унифицировать коды ошибок по `backend-api-endpoints.md`

## Priority 2 (MVP+)

### 1. Payment integration
- [ ] Выделить payment provider abstraction
- [ ] Реализовать webhook/callback endpoint
- [ ] Проверять подпись callback и идемпотентность

### 2. Device integration
- [ ] Подключить runtime маршруты для device transport
- [ ] Реализовать polling команд и отчет о выполнении
- [ ] Добавить heartbeat обработку

### 3. Нормализация service layer
- [ ] Дробить `rental_flow_service` на более узкие сервисы по bounded context
- [ ] Сохранить текущий API контракт без breaking changes

## Priority 3 (качество и эксплуатация)

### 1. Tests
- [ ] Unit тесты для service
- [ ] Repository тесты на тестовой БД
- [ ] Набор smoke-интеграций для MVP endpoints

### 2. Observability
- [ ] Добавить request logging с `requestId`
- [ ] Добавить базовые метрики API
- [ ] Подготовить health/readiness checks

### 3. CI hygiene
- [ ] Проверки `go test ./...` и `go vet ./...` в CI
- [ ] Проверка миграций в CI (dry-run)

## Infra / DX

- [ ] Добавить `.env.example` с актуальными переменными
- [ ] Проверить, что `Makefile` отражает реальный workflow
- [ ] Держать docs синхронно с runtime

## Удалять ли этот файл

Не удалять. Файл нужен как roadmap, но в актуальном виде (без старых импортов и путей).

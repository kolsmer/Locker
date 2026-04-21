# LOCK'IT Admin Panel API Spec (v1)

## 1. Цель

Документ описывает backend API для админской панели LOCK'IT:

- дэшборд со статусом всех камер хранения и ячеек;
- просмотр детальной информации по ячейке;
- изменение статуса ячейки и ручное открытие;
- выгрузка выручки в Excel;
- авторизация администратора без регистрации.

Документ ориентирован на постановку задачи backend-команде.

---

## 2. Общие правила

- Base URL: `/api/v1/admin`
- Формат запросов/ответов: `application/json; charset=utf-8`
- Авторизация для всех admin endpoint (кроме login): `Authorization: Bearer <token>`
- Версия API: v1

Рекомендуемый единый envelope:

```json
{
  "ok": true,
  "data": {},
  "meta": {},
  "requestId": "req_xxx"
}
```

Ошибки:

```json
{
  "ok": false,
  "error": {
    "code": "FORBIDDEN",
    "message": "Недостаточно прав",
    "details": {}
  },
  "requestId": "req_xxx"
}
```

---

## 3. Роли и права (RBAC)

Поддерживаемые роли (из текущей доменной модели):

- `admin`
- `operator`
- `support`

Матрица доступа:

- `admin`: полный доступ, включая выручку/экспорт.
- `operator`: дэшборд, карточка ячейки, смена статуса, ручное открытие.
- `support`: read-only дэшборд/карточка без изменения статусов и без выручки.

---

## 4. Авторизация (без регистрации)

Регистрация endpoint не нужен. Админы создаются только backend-ом (миграцией/скриптом/ручным seed).

## 4.1 POST /api/v1/admin/login

Назначение:

- вход администратора по логину/паролю.

Request:

```json
{
  "login": "admin",
  "password": "secret"
}
```

Response 200:

```json
{
  "ok": true,
  "data": {
    "accessToken": "<jwt>",
    "tokenType": "Bearer",
    "expiresIn": 3600,
    "admin": {
      "id": 1,
      "login": "admin",
      "role": "admin"
    }
  }
}
```

Ошибки:

- `401 INVALID_CREDENTIALS`
- `403 ADMIN_DISABLED`

## 4.2 GET /api/v1/admin/me

Назначение:

- получить профиль текущего админа по токену.

Response 200:

```json
{
  "ok": true,
  "data": {
    "id": 1,
    "login": "admin",
    "role": "admin",
    "isActive": true
  }
}
```

## 4.3 (Опционально) POST /api/v1/admin/logout

Назначение:

- инвалидировать refresh token / server session (если реализуется stateful auth).

## 4.4 (Опционально) POST /api/v1/admin/refresh

Назначение:

- обновление access token.

---

## 5. Дэшборд: камеры хранения и ячейки

## 5.1 GET /api/v1/admin/locations

Назначение:

- список точек (камер хранения) для левой панели/таблицы.
- сразу возвращает агрегаты по статусам ячеек.

Query params:

- `search` (string, optional)
- `isActive` (bool, optional)
- `limit` (int, default 50)
- `offset` (int, default 0)

Response 200:

```json
{
  "ok": true,
  "data": [
    {
      "locationId": 123,
      "name": "LOCK'IT Demo",
      "address": "ULITSA PUSHKINA, 14",
      "isActive": true,
      "cellsTotal": 7,
      "cellsByStatus": {
        "free": 4,
        "reserved": 0,
        "occupied": 2,
        "locked": 0,
        "open": 0,
        "maintenance": 1,
        "out_of_service": 0
      },
      "updatedAt": "2026-04-20T12:30:00Z"
    }
  ],
  "meta": {
    "total": 1
  }
}
```

## 5.2 GET /api/v1/admin/locations/{locationId}/lockers

Назначение:

- список всех ячеек внутри выбранной камеры хранения (таблица на дэшборде).

Query params:

- `status` (multi, optional)
- `size` (multi: `S|M|L|XL`, optional)
- `isActive` (bool, optional)
- `limit`/`offset` (pagination)

Response 200:

```json
{
  "ok": true,
  "data": [
    {
      "lockerId": 987,
      "lockerNo": 201,
      "size": "M",
      "status": "free",
      "isActive": true,
      "price": 900,
      "hardwareId": "hw-201",
      "lastEventAt": 1713601111,
      "updatedAt": 1713601111
    }
  ],
  "meta": {
    "total": 1
  }
}
```

## 5.3 GET /api/v1/admin/lockers/{lockerId}

Назначение:

- детальная карточка ячейки (просмотр инфо по ячейке + текущая аренда + платеж + события).

Response 200:

```json
{
  "ok": true,
  "data": {
    "locker": {
      "lockerId": 987,
      "locationId": 123,
      "lockerNo": 201,
      "size": "M",
      "status": "occupied",
      "isActive": true,
      "price": 900,
      "hardwareId": "hw-201"
    },
    "activeRental": {
      "rentalId": "rent_abc123",
      "state": "active",
      "phoneMasked": "+79******33",
      "openedAt": "2026-04-20T11:10:00Z",
      "finishedAt": null
    },
    "lastPayment": {
      "paymentId": "pay_abc123",
      "status": "paid",
      "amount": 900,
      "currency": "RUB",
      "paidAt": "2026-04-20T11:09:50Z"
    },
    "recentEvents": [
      {
        "id": 1001,
        "eventType": "locker_opened",
        "payload": {},
        "createdAt": 1713601111
      }
    ]
  }
}
```

---

## 6. Управление статусом ячеек

Статусы из доменной модели:

- `free`
- `reserved`
- `occupied`
- `locked`
- `open`
- `maintenance`
- `out_of_service`

## 6.1 PATCH /api/v1/admin/lockers/{lockerId}/status

Назначение:

- вручную изменить статус ячейки из админки.

Request:

```json
{
  "status": "maintenance",
  "reason": "door sensor issue"
}
```

Response 200:

```json
{
  "ok": true,
  "data": {
    "lockerId": 987,
    "previousStatus": "free",
    "newStatus": "maintenance",
    "updatedAt": "2026-04-20T12:35:00Z"
  }
}
```

Ошибки:

- `404 LOCKER_NOT_FOUND`
- `409 INVALID_STATUS_TRANSITION`
- `409 ACTIVE_RENTAL_EXISTS` (если запрещено менять на `free`/`maintenance` при активной аренде)
- `403 FORBIDDEN`

Требования:

- логировать изменение в `audit_logs`;
- добавлять запись в `locker_events`.

## 6.2 POST /api/v1/admin/lockers/{lockerId}/open

Назначение:

- ручное открытие ячейки админом/оператором (через device command).

Request:

```json
{
  "reason": "support_manual_open"
}
```

Response 200:

```json
{
  "ok": true,
  "data": {
    "lockerId": 987,
    "commandId": 543,
    "status": "pending"
  }
}
```

Ошибки:

- `404 LOCKER_NOT_FOUND`
- `409 LOCKER_NOT_FUNCTIONAL`
- `403 FORBIDDEN`

---

## 7. Сессии/аренды для админки

## 7.1 GET /api/v1/admin/sessions

Назначение:

- таблица сессий/аренд для админов (поиск инцидентов и текущих пользователей).

Query params:

- `locationId` (optional)
- `lockerId` (optional)
- `status` (multi, optional)
- `phone` (optional, partial)
- `from`/`to` (unix или ISO-8601)
- `limit`/`offset`

Response 200:

```json
{
  "ok": true,
  "data": [
    {
      "sessionId": 445,
      "lockerId": 987,
      "lockerNo": 201,
      "locationId": 123,
      "phoneMasked": "+79******33",
      "status": "active",
      "startedAt": 1713601111,
      "paidUntil": 1713604711,
      "closedAt": null
    }
  ],
  "meta": {
    "total": 1
  }
}
```

---

## 8. Выручка: выгрузка Excel

Требование бизнеса: на фронте только кнопка "Скачать", файл генерируется backend-ом.

## 8.1 GET /api/v1/admin/revenue/export

Назначение:

- сформировать и отдать Excel-файл по фильтрам.

Query params:

- `from` (required, ISO date: `YYYY-MM-DD`)
- `to` (required, ISO date: `YYYY-MM-DD`)
- `locationId` (optional)
- `groupBy` (optional: `location` | `day`, default `location`)
- `tz` (optional, default `UTC`)

Response 200:

- `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- `Content-Disposition: attachment; filename="revenue_2026-04-01_2026-04-20.xlsx"`
- Body: binary `.xlsx`

Ошибки:

- `422 INVALID_DATE_RANGE`
- `403 FORBIDDEN` (роль не `admin`)
- `500 EXPORT_GENERATION_FAILED`

Минимальный состав колонок в Excel:

- `location_id`
- `location_name`
- `address`
- `payments_count`
- `revenue_rub`
- `avg_check_rub`
- `first_payment_at`
- `last_payment_at`

Примечание по источнику данных:

- учитывать только успешные платежи (`paid`/`confirmed`), в зависимости от активной схемы таблиц runtime.

## 8.2 (Опционально) GET /api/v1/admin/revenue/summary

Назначение:

- быстрый JSON-превью KPI перед загрузкой файла.

---

## 9. Коды ошибок (рекомендуемый каталог)

- `UNAUTHORIZED`
- `FORBIDDEN`
- `INVALID_CREDENTIALS`
- `ADMIN_DISABLED`
- `LOCATION_NOT_FOUND`
- `LOCKER_NOT_FOUND`
- `SESSION_NOT_FOUND`
- `INVALID_STATUS`
- `INVALID_STATUS_TRANSITION`
- `ACTIVE_RENTAL_EXISTS`
- `LOCKER_NOT_FUNCTIONAL`
- `INVALID_DATE_RANGE`
- `EXPORT_GENERATION_FAILED`
- `INTERNAL_ERROR`

---

## 10. Минимальный backend scope (MVP) для старта фронта

Обязательные endpoint, чтобы фронт админки можно было начать делать сразу:

1. `POST /api/v1/admin/login`
2. `GET /api/v1/admin/me`
3. `GET /api/v1/admin/locations`
4. `GET /api/v1/admin/locations/{locationId}/lockers`
5. `GET /api/v1/admin/lockers/{lockerId}`
6. `PATCH /api/v1/admin/lockers/{lockerId}/status`
7. `POST /api/v1/admin/lockers/{lockerId}/open`
8. `GET /api/v1/admin/sessions`
9. `GET /api/v1/admin/revenue/export`

---

## 11. Что важно зафиксировать до реализации

1. Формат времени для фильтров (`ISO` или `unix`) и единый timezone.
2. Политика masking телефона в админке.
3. Разрешенные переходы статусов ячеек (state machine).
4. Единая бизнес-логика "успешного платежа" для расчета выручки.
5. TTL JWT и стратегия refresh/logout.
6. Требования по аудиту: какие действия логировать обязательно.

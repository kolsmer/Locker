# LOCK'IT Backend API Spec

## 1. Назначение документа

Документ описывает API для MVP системы камер хранения LOCK'IT:

- публичная страница со списком камер хранения и количеством свободных ячеек;
- интерфейс локера (киоск) с внутренним флоу бронирования/открытия;
- обработка доступа по коду и оплаты;
- завершение аренды.

Документ ориентирован на бэкенд-разработку и согласование контрактов с фронтендом.

---

## 2. Базовые правила API

### 2.1 Base URL и версия

- Base URL: `/api`
- Версия: `/v1`
- Полные пути: `/api/v1/...`

### 2.2 Формат данных

- `Content-Type: application/json; charset=utf-8`
- Все даты и время в `ISO-8601 UTC` (пример: `2026-04-17T12:30:00Z`)

### 2.3 Общая структура ошибок

```json
{
  "ok": false,
  "error": {
    "code": "INVALID_PHONE",
    "message": "Введите корректный номер телефона",
    "details": {
      "field": "phone"
    }
  },
  "requestId": "d6e7f8aa-4bb3-4a24-9f4e-6a8f2487f2d1"
}
```

### 2.4 Успешный ответ

Рекомендуемый формат:

```json
{
  "ok": true,
  "data": {},
  "meta": {}
}
```

---

## 3. Модели данных

## 3.1 LockerSummary

```json
{
  "id": 123,
  "street": "ULITSA PUSHKINA, 14",
  "freeCells": {
    "s": 4,
    "m": 2,
    "l": 1,
    "xl": 0
  },
  "updatedAt": "2026-04-17T12:30:00Z"
}
```

## 3.2 Selection

```json
{
  "selectionId": "sel_8f5bc4",
  "lockerId": 123,
  "size": "m",
  "cellNumber": 202,
  "holdExpiresAt": "2026-04-17T12:32:00Z"
}
```

## 3.3 Booking/Rental

```json
{
  "bookingId": "book_11fa6f",
  "rentalId": "rent_0f9db8",
  "lockerId": 123,
  "cellNumber": 202,
  "phone": "+79999999999",
  "accessCode": "1A2B3C",
  "state": "active",
  "openedAt": "2026-04-17T12:31:05Z"
}
```

## 3.4 Payment

```json
{
  "paymentId": "pay_84fbd3",
  "rentalId": "rent_0f9db8",
  "amount": 900,
  "currency": "RUB",
  "status": "pending",
  "qrPayload": "<string for qr generation>",
  "paymentExpiresAt": "2026-04-17T12:36:00Z"
}
```

---

## 4. Эндпоинты MVP (обязательные)

## 4.1 Получить список камер хранения

### `GET /api/v1/lockers`

Назначение:

- Главная пользовательская страница;
- Автообновление каждые 20 секунд.

Query params (опционально):

- `city` (string)
- `limit` (int)
- `offset` (int)

Ответ `200`:

```json
{
  "ok": true,
  "data": [
    {
      "id": 123,
      "street": "ULITSA PUSHKINA, 14",
      "freeCells": {
        "s": 4,
        "m": 2,
        "l": 1,
        "xl": 0
      },
      "updatedAt": "2026-04-17T12:30:00Z"
    }
  ],
  "meta": {
    "total": 1
  }
}
```

---

## 4.2 Выбрать ячейку (по размеру или габаритам)

### `POST /api/v1/lockers/{lockerId}/cell-selection`

Назначение:

- На экране локера определить конкретную ячейку.

Тело запроса (вариант A, по размеру):

```json
{
  "size": "m"
}
```

Тело запроса (вариант B, по габаритам):

```json
{
  "dimensions": {
    "length": 45,
    "width": 35,
    "height": 30,
    "unit": "cm"
  }
}
```

Ответ `200` (ячейка найдена):

```json
{
  "ok": true,
  "data": {
    "selectionId": "sel_8f5bc4",
    "lockerId": 123,
    "size": "m",
    "cellNumber": 202,
    "holdExpiresAt": "2026-04-17T12:32:00Z"
  }
}
```

Ответ `409` (нет свободных):

```json
{
  "ok": false,
  "error": {
    "code": "NO_CELLS_AVAILABLE",
    "message": "Свободных ячеек этого размера нет"
  }
}
```

Ответ `422` (невалидные данные): `INVALID_DIMENSIONS` или `INVALID_SIZE`.

---

## 4.3 Подтвердить бронь по номеру телефона

### `POST /api/v1/lockers/{lockerId}/bookings`

Назначение:

- После выбора ячейки пользователь вводит телефон;
- Создается аренда и код доступа.

Тело запроса:

```json
{
  "selectionId": "sel_8f5bc4",
  "phone": "+79999999999"
}
```

Ответ `201`:

```json
{
  "ok": true,
  "data": {
    "bookingId": "book_11fa6f",
    "rentalId": "rent_0f9db8",
    "lockerId": 123,
    "cellNumber": 202,
    "phone": "+79999999999",
    "accessCode": "1A2B3C",
    "state": "active",
    "openedAt": "2026-04-17T12:31:05Z"
  }
}
```

Ошибки:

- `422 INVALID_PHONE`
- `410 SELECTION_EXPIRED`
- `409 CELL_ALREADY_TAKEN`

---

## 4.4 Проверить код доступа (ветка "Есть код?")

### `POST /api/v1/lockers/{lockerId}/access-code/check`

Назначение:

- Пользователь вводит код доступа на стартовом экране локера;
- Система определяет, нужна ли оплата.

Тело запроса:

```json
{
  "accessCode": "1A2B3C"
}
```

Ответ `200` (оплата не нужна):

```json
{
  "ok": true,
  "data": {
    "rentalId": "rent_0f9db8",
    "lockerId": 123,
    "cellNumber": 202,
    "phone": "+79999999999",
    "accessCode": "1A2B3C",
    "paymentRequired": false,
    "state": "active"
  }
}
```

Ответ `200` (оплата нужна):

```json
{
  "ok": true,
  "data": {
    "rentalId": "rent_0f9db8",
    "lockerId": 123,
    "cellNumber": 202,
    "phone": "+79999999999",
    "accessCode": "1A2B3C",
    "paymentRequired": true,
    "payment": {
      "paymentId": "pay_84fbd3",
      "amount": 900,
      "currency": "RUB",
      "status": "pending",
      "qrPayload": "<string>",
      "paymentExpiresAt": "2026-04-17T12:36:00Z"
    }
  }
}
```

Ошибка `404 INVALID_ACCESS_CODE`.

---

## 4.5 Статус оплаты

### `GET /api/v1/payments/{paymentId}`

Назначение:

- Polling со стороны фронта на экране оплаты;
- Для текущего UI можно переходить в `paid` через ~5 секунд (мок/стаб).

Ответ `200`:

```json
{
  "ok": true,
  "data": {
    "paymentId": "pay_84fbd3",
    "status": "pending",
    "amount": 900,
    "currency": "RUB",
    "paidAt": null
  }
}
```

`status`:

- `pending`
- `paid`
- `failed`
- `expired`

---

## 4.6 Открыть ячейку после оплаты (если требуется отдельной командой)

### `POST /api/v1/rentals/{rentalId}/open`

Назначение:

- Явно открыть замок после `payment.status=paid`.

Ответ `200`:

```json
{
  "ok": true,
  "data": {
    "rentalId": "rent_0f9db8",
    "cellNumber": 202,
    "opened": true,
    "openedAt": "2026-04-17T12:35:15Z"
  }
}
```

Примечание:

- Если открытие выполняется автоматически на стороне бэка/железа, этот эндпоинт можно не делать.

---

## 4.7 Завершить аренду

### `POST /api/v1/rentals/{rentalId}/finish`

Назначение:

- Кнопка "Завершить аренду".

Ответ `200`:

```json
{
  "ok": true,
  "data": {
    "rentalId": "rent_0f9db8",
    "state": "closed",
    "finishedAt": "2026-04-17T12:36:10Z"
  }
}
```

---

## 5. Эндпоинты (желательно, но можно после MVP)

## 5.1 Получить текущее состояние аренды

### `GET /api/v1/rentals/{rentalId}`

Назначение:

- Восстановление состояния после перезагрузки интерфейса локера.

---

## 6. Коды ошибок (каталог)

- `INVALID_PHONE`
- `INVALID_SIZE`
- `INVALID_DIMENSIONS`
- `NO_CELLS_AVAILABLE`
- `SELECTION_EXPIRED`
- `CELL_ALREADY_TAKEN`
- `INVALID_ACCESS_CODE`
- `PAYMENT_REQUIRED`
- `PAYMENT_FAILED`
- `PAYMENT_EXPIRED`
- `RENTAL_NOT_FOUND`
- `LOCKER_NOT_FOUND`
- `INTERNAL_ERROR`

---

## 7. Бизнес-правила и валидация

1. Размеры: enum `s | m | l | xl`.
2. Телефон:
   - принимать 11 цифр;
   - поддержка ввода с `7`/`8`;
   - хранить в нормализованном формате `+7XXXXXXXXXX`.
3. Код доступа:
   - uppercase;
   - формат `A-Z0-9`;
   - длина 6 (рекомендуется).
4. Selection имеет TTL (рекомендуется 60-120 секунд).
5. Резерв/выдача ячейки должны быть атомарными (избежать гонки).
6. Все мутационные эндпоинты логировать с `requestId`.

---

## 8. Маппинг к текущему UI-флоу

### Ветка 1 (новая аренда)

1. `POST /cell-selection` (размер или габариты)
2. `POST /bookings` (телефон)
3. UI показывает `cellNumber + phone + accessCode`

### Ветка 2 (вход по коду)

1. `POST /access-code/check`
2. если `paymentRequired=true` -> экран оплаты
3. `GET /payments/{paymentId}` polling
4. после `paid` -> `POST /rentals/{rentalId}/open` (если нужно)
5. UI показывает активную аренду и кнопку завершения

### Завершение

1. `POST /rentals/{rentalId}/finish`

---

## 9. Нефункциональные требования (рекомендации)

1. p95 latency:
   - read endpoints <= 300ms
   - write endpoints <= 500ms
2. Временное отсутствие платежного провайдера не должно ломать киоск-флоу.
3. Контракты API стабилизировать через OpenAPI 3.1.
4. Для киосков рекомендуется service-to-service auth (например, `X-Kiosk-Token`).

---

## 10. Что можно сделать следующим шагом

- Подготовить OpenAPI YAML по этим контрактам.
- Добавить JSON Schema для валидации запросов/ответов.
- Согласовать интеграцию с железом (open/close lock command).

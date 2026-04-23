# Электронная камера хранения багажа

Система для автоматического подбора ячейки хранения на основе габаритов багажа.

## О проекте

Проект позволяет пользователям:
- Ввести размеры своего багажа (высота, ширина, глубина)
- Получить рекомендацию по подходящему типу ячейки (S/M/L/XL)
- Увидеть стоимость аренды
- Забронировать ячейку с получением QR-кода

## Технологический стек

### Backend
- **Go 1.21+** - основной язык
- **Gorilla Mux** - роутинг
- **PostgreSQL** - основная база данных
- **JWT** - аутентификация

### Frontend
- **React 19** - UI библиотека
- **TypeScript** - типизация
- **Axios** - HTTP клиент
- **React Router v7** - маршрутизация
- **React Query** - управление состоянием серверных данных

### DevOps
- **Docker** - контейнеризация
- **Docker Compose** - оркестрация
- **Kubernetes** - оркестрация для cluster deployment
- **GitHub Actions** - CI/CD
- **Render / Yandex Cloud** - деплой

## Быстрый старт

### Предварительные требования
- Go 1.21+
- Node.js 18+
- Docker & Docker Compose
- PostgreSQL (или используйте Docker)

### Установка

1. **Клонировать репозиторий**
```bash
git clone https://github.com/kolsmer/Locker.git
cd Locker
```

2. **Запустить проект через Docker Compose**
```bash
docker compose up -d --build
```

3. **Проверить доступность**
```bash
curl http://localhost:8081/
curl http://localhost:8080/healthz
```

## Kubernetes

В репозитории добавлены Kubernetes-манифесты в директории [`k8s`](./k8s) для следующих компонентов:
- PostgreSQL с `PersistentVolumeClaim`
- backend API
- frontend nginx
- отдельный `Job` для миграций

Перед первым деплоем проверьте значения в [`k8s/secret.yaml`](./k8s/secret.yaml) и замените как минимум `JWT_SECRET` и пароль БД.

Собрать образы:
```bash
make k8s-build-images
```

Если ваш кластер не видит локальные Docker-образы, загрузите их в runtime кластера отдельной командой вашей среды, например `kind load docker-image ...` или `minikube image load ...`.

Развернуть в Kubernetes:
```bash
make k8s-up
```

Проверить статус:
```bash
make k8s-status
```

Открыть frontend локально:
```bash
kubectl port-forward svc/frontend 8081:80 -n locker
```

После этого приложение будет доступно на `http://localhost:8081`.

Удалить ресурсы:
```bash
make k8s-down
```

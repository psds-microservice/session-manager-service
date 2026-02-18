# session-manager-service

Микросервис управления бизнес-логикой сессий консультаций PSDS: matchmaking, приглашения операторов, права, метаданные, PIN. Шаблон — **user-service** (helpy, internal/command, Cobra).

## Назначение

- Сопоставление клиентов со свободными операторами (matchmaking).
- Приглашение операторов в сессию.
- Управление правами (ведущий оператор, режим).
- Метаданные сессии (участники, статусы, таймстампы).
- Генерация уникальных PIN-кодов для быстрого доступа операторов.

## API (REST)

- **POST /session** — создать сессию консультации (body: `client_id`, опционально `stream_session_id`). Возвращает `id`, `status`, `pin`.
- **GET /session/:id** — метаданные сессии.
- **GET /session/:id/participants** — список участников.
- **POST /session/join** — оператор присоединяется по `session_id` или `pin` (body: `session_id` или `pin`, `user_id`).
- **POST /session/:id/invite** — пригласить оператора (body: `operator_id`).
- **POST /session/:id/control** — изменить параметры (body: `action`).

## Health

- **GET /health**, **GET /ready** — пути из helpy.

## Конфигурация

`.env` (см. `.env.example`): `APP_HOST`, `APP_PORT` (8091), `DB_*`. Конфиг без godotenv внутри — загрузка `.env` в cmd. Валидация `Validate()` при старте.

## Запуск

PostgreSQL должен быть запущен. Из корня сервиса:

```bash
cp .env.example .env
go run ./cmd/session-manager-service api
# или
go run ./cmd/session-manager-service migrate up
go run ./cmd/session-manager-service api
```

Docker: `cd deployments && docker compose up -d`.

## Шаблон (user-service)

- **helpy:** `db.Open(cfg.DSN())`, `paths.PathHealth`, `PathReady`.
- **internal/command:** `MigrateUp(databaseURL)`, `Seed(db)` — вызываются из cmd.
- **internal/database:** `Open`, `MigrateUp`, `RunSeeds`.
- **cmd:** godotenv в api/migrate/seed; Cobra: api, migrate up, seed.
- **config:** `Load()` без godotenv, `Validate()`, `DSN()`, `DatabaseURL()`, вложенный `DB`.

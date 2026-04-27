@../edium-claude/claude/shared/CLAUDE.md
@../edium-claude/claude/shared/PRODUCT_FLOWS.md
@../edium-claude/claude/shared/api-conventions.md
@../edium-claude/claude/shared/security-rules.md
@../edium-claude/claude/shared/project-context.md
@../edium-claude/claude/backend/CLAUDE.md

# CLAUDE.md — Edium Backend

Образовательная платформа Edium — серверная часть. Go-микросервисы + Python-сервис генерации.

## О проекте

Edium — платформа для проведения квизов, контрольных и домашних работ. Автоматизирует создание учебных материалов с помощью LLM.

**Роли:** student (проходит квизы) и teacher (создаёт квизы, управляет классами)

## Сервисы

| Сервис | Ответственность |
|--------|----------------|
| **Doorman** | Auth: OTP по телефону, JWT (access + refresh), регистрация |
| **Caesar** | Пользователи, классы, участники классов, курсы, модули, элементы курса, ведомость |
| **Riddler** | Квизы, тестовые и live-сессии, результаты, real-time через WebSocket |
| **Herald** | Уведомления: OTP через Telegram-бот, push (Firebase) |
| **Charon** | Прокси к DeepSeek API (LLM) через NATS |
| **Sphinx** | Генерация вопросов квиза по тексту (Python + LLM, NATS-воркер) |
| **Louvre** | Загрузка и хранение изображений (MinIO) |

## Стек

- **Go 1.25+** — все сервисы кроме Sphinx
- **Python** — Sphinx (генерация вопросов) + `ml/` (офлайн-пайплайн датасета)
- **PostgreSQL** — основные данные (миграции через Flyway)
- **Redis** — кэш, OTP, rate limiting, WebSocket-сессии
- **MinIO** — файловое хранилище (изображения к вопросам)
- **NATS** — брокер сообщений, Transactional Outbox
- **Docker Compose** — оркестрация всего стека

## Структура проекта

```
edium-backend/
├── doorman/        # Auth: OTP, JWT
├── caesar/         # Пользователи, классы, курсы, ведомость
├── riddler/        # Квизы + WebSocket live
├── herald/         # Уведомления (Telegram, Firebase)
├── charon/         # LLM-прокси (DeepSeek)
├── sphinx/         # Генерация вопросов по тексту (Python + LLM)
├── louvre/         # Изображения (MinIO)
├── ml/             # Офлайн-пайплайн генерации квиз-датасета (не production)
│   ├── main.py
│   └── books/
├── docker-compose.yaml
└── Makefile
```

## Архитектура Go-сервисов

Каждый сервис следует стандартной раскладке:

```
<service>/
├── cmd/main.go
└── internal/
    ├── domain/          # Бизнес-объекты (нет зависимостей от БД/HTTP)
    ├── service/         # Бизнес-логика, зависит от интерфейсов repository
    ├── handler/         # HTTP-обработчики
    ├── repository/      # Работа с БД
    ├── transport/dto/   # Request/Response структуры
    ├── infra/           # Инфраструктурные зависимости
    ├── app/             # Сборка приложения (DI)
    └── config/          # Конфигурация из env
```

## Doorman — аутентификация

**Домен:** `Identity` (id, status: active/blocked/deleted, phone)

**Эндпоинты:**
- `POST doorman/v1/otp/send` — отправка OTP (канал: `tg` или `sms`; повтор через 3 минуты)
- `POST doorman/v1/otp/verify` — проверка OTP (6-значный код); возвращает токены или `registration_token`
- `POST doorman/v1/auth/register` — регистрация нового пользователя (заголовок `X-Reg-Token`)
- `POST doorman/v1/auth/refresh` — обновление токенов
- `POST doorman/v1/auth/logout` — выход (отзыв refresh-токена)
- `GET doorman/.well-known/jwks.json` — публичные ключи (без `/v1/`)

**Телефон:** regex `^\+7\d{10}$` (только +7)

**Воркеры (Transactional Outbox):**

Входящие NATS-события сначала сохраняются в таблицу `task`, затем обрабатываются отдельным воркером-процессором. Это гарантирует exactly-once при перезапусках. Сквозной `trace_id` передаётся через W3C `traceparent` в колонке `task.trace_ctx`.

| Воркер | Тип | NATS / DB | Действие |
|--------|-----|-----------|---------|
| `OTPRequestConsumer` | consumer | `herald.otp.requested` → DB `otp_request` | сохраняет задачу |
| `OTPRequestProcessor` | processor | DB `otp_request` | вызывает `SendOTP` |
| `OTPSentPublisher` | publisher | DB `otp_sent` → `doorman.otp.sent` | публикует в NATS |
| `UserCreatedPublisher` | publisher | DB `user_created` → `doorman.user.created` | публикует в NATS |
| `UserDeletedConsumer` | consumer | `caesar.user.deleted` → DB `user_deleted` | сохраняет задачу |
| `UserDeletedProcessor` | processor | DB `user_deleted` | `status=deleted` + удаляет JWT-токены |

**Payload задач:**
- `otp_request`: `{phone, channel}`
- `otp_sent`: `{phone, otp, channel}`
- `user_created`: `{user_id, phone, name, surname}`
- `user_deleted`: `{user_id}`

## Herald — уведомления

**Задача:** Telegram-бот принимает номер телефона от пользователя и связывает его с `chat_id`. Когда Doorman отправляет OTP, Herald доставляет его в нужный чат. Дополнительно отправляет push-уведомления через Firebase Cloud Messaging.

**Флоу:**
1. Пользователь отправляет контакт боту → `handleContact` → `RequestOTP(chatID, phone)` → сохраняет `PendingOTP{phone, chat_id}` в Redis + создаёт задачу `otp_request` в outbox
2. Doorman генерирует OTP и публикует в `doorman.otp.sent`
3. Herald доставляет OTP в Telegram по `chat_id` из `pending_otp`

**Воркеры:**

| Воркер | Тип | NATS / DB | Действие |
|--------|-----|-----------|---------|
| `OTPRequestPublisher` | publisher | DB `otp_request` → `herald.otp.requested` | публикует запрос в NATS |
| `OTPSentConsumer` | consumer | `doorman.otp.sent` → DB `otp_sent` | сохраняет задачу |
| `OTPSentProcessor` | processor | DB `otp_sent` | смотрит `pending_otp` по `phone`, шлёт Telegram, удаляет `pending_otp` |

**Хранилище `pending_otp`:** Redis, ключ — номер телефона, содержит `chat_id` и TTL 10 минут.

**Сквозной trace_id:** root span создаётся в `handleContact` → передаётся через `task.trace_ctx` → восстанавливается в каждом процессоре.

## Riddler — квизы и WebSocket

**Типы сессий:**
- **test** — асинхронный, каждый в своём темпе, таймер на весь тест (`total_time_limit_sec`)
- **live** — синхронный, учитель управляет вопросами, таймер на каждый вопрос (`question_time_limit_sec`)

**Типы вопросов:** `single_choice`, `multiple_choice`, `with_given_answer`, `with_free_answer`, `drag`, `connection`

**Live-фазы:** `pending` → `lobby` → `question_active` → `question_locked` → `completed`

**Источники сессий:**
- `course` — привязана к модулю курса; только ученики класса, требуется JWT
- `library` — публичная; анонимный доступ по 6-значному числовому коду (`join_code`)

**WebSocket-события (server → client):**
- `state.snapshot` — полное состояние при (ре)коннекте
- `lobby.participant_joined` / `lobby.participant_left` — лобби
- `quiz.started` — старт квиза
- `question.started` — показан очередной вопрос
- `participant.answered` — ученик ответил (только учителю)
- `question.stats_tick` — промежуточная статистика (только учителю)
- `question.locked` — вопрос закрыт, правильный ответ
- `quiz.completed` — квиз завершён
- `participant.kicked` / `you_were_kicked` — кик участника

**WebSocket-команды (client → server):**
- `teacher.start_lobby`, `teacher.start_quiz`, `teacher.next_question`, `teacher.kick_participant`
- `student.submit_answer`

**Настройки квиза:** таймер, дедлайн (`started_at`/`finished_at`), перемешивание вопросов. Вход в live — по 6-значному числовому `join_code` (только в фазе `lobby`).

## ML-пайплайн (ml/) — офлайн-инструмент

Не является частью production-системы. Используется для подготовки датасета квиз-вопросов.

**Флоу:** `PDF + chapters.txt → PyMuPDF → GPT-4o-mini → dataset.jsonl`

**Запуск:** `cd ml && python main.py --books_dir books/ --output dataset.jsonl`

**Env:** `OPENAI_API_KEY` в `ml/.env` или `.env` в корне

## Команды

```bash
make run          # docker-compose up -d --build
make down         # docker-compose down
make clean        # docker-compose down --volumes
make genrsa       # RSA-ключи для JWT

go test ./...     # тесты (в директории сервиса)
gofmt -w .
goimports -w .

cd ml && pytest
cd ml && ruff format .
```

## Безопасность

- Все эндпоинты (кроме auth и `GET /doorman/.well-known/jwks.json`) требуют JWT
- Для live-сессий ученик аутентифицируется одноразовым `ws_token` (TTL 5 минут, не JWT)
- Не логировать персональные данные (телефоны, имена)
- Не хардкодить секреты — только через env
- Текст вопросов квиза не передавать клиенту до старта сессии / попытки

## OpenAPI-спецификации

Каждый сервис обязан иметь спецификацию в формате OpenAPI 3.0 YAML:

```
<service>/
└── api/
    └── openapi.yaml
```

**Правила:**
- Все эндпоинты сервиса описаны в одном файле `openapi.yaml`
- Схемы request/response совпадают с `transport/dto/` структурами
- При добавлении или изменении эндпоинта — обновлять `openapi.yaml` в том же коммите
- WebSocket-события (Riddler) описывать в `x-websocket-events` extension или отдельным `asyncapi.yaml`

**Структура файла:**
```yaml
openapi: "3.0.3"
info:
  title: Edium <ServiceName>
  version: "1.0.0"
servers:
  - url: /api/v1
paths:
  /endpoint:
    post:
      summary: Краткое описание на русском
      requestBody: ...
      responses:
        "200": ...
        "400": { $ref: "#/components/responses/BadRequest" }
        "401": { $ref: "#/components/responses/Unauthorized" }
```

## Деплой

`docker compose up -d` **не перезапускает** контейнеры, у которых изменился только смонтированный файл конфига (volume). Для таких сервисов нужен явный `docker compose restart <service>` в deploy-скрипте.

Сервисы с файловым конфигом (всегда рестартовать при деплое):
- `grafana` — provisioning datasources/dashboards (`./grafana/`)
- `vector` — `vector.yaml`

OTLP gRPC endpoint (`OTEL_ENDPOINT`) должен быть в формате `host:port` без схемы: `jaeger:4317`, **не** `http://jaeger:4317`.

## Соглашения

- **Не добавлять Claude или AI-инструменты в список авторов/contributors** в коде, коммитах или документации
- **Язык:** комментарии, git-сообщения, документация — на русском
- **Ошибки Go:** `fmt.Errorf("контекст: %w", err)`
- **HTTP-коды:** 200, 204, 400, 401, 403, 404, 409, 410, 422, 429, 500
- **ID:** UUID (пользователи, классы, курсы, квизы); 6-значный числовой — `join_code` live-сессии
- **Миграции:** Flyway, `V{N}__{описание}.sql`, только append-only
- **OTEL endpoint:** формат `host:port` без схемы — `jaeger:4317`, не `http://jaeger:4317`
- **Телефон:** только `^\+7\d{10}$`

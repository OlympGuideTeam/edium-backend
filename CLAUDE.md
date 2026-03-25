# CLAUDE.md — Edium Backend

Образовательная платформа Edium — серверная часть. Go-микросервисы + Python ML-пайплайн.

## О проекте

Edium — платформа для проведения квизов, контрольных и домашних работ. Автоматизирует создание учебных материалов с помощью OCR и LLM.

**Роли:** student (проходит квизы) и teacher (создаёт квизы, управляет классами)

> **Продуктовые флоу:** карта экранов → сервисы → API-контракт описана в
> `edium-claude/claude/shared/PRODUCT_FLOWS.md` — читай перед проектированием эндпоинтов.

## Сервисы

| Сервис | Ответственность |
|--------|----------------|
| **Doorman** | Auth: OTP по телефону, JWT (access + refresh), регистрация |
| **Caesar** | Пользователи, классы, участники классов |
| **Tate** | Курсы (вложены в классы), модули курсов |
| **Riddler** | Квизы, сессии прохождения, результаты, real-time через WebSocket |
| **Yoda** | Проверочные работы, сабмиты, оценки *(v2)* |
| **Herald** | Уведомления: push (Firebase) + in-app |
| **Charon** | Прокси к внешним LLM API (OpenAI и др.) |
| **Hawkeye** | OCR: распознавание рукописного текста с фото *(Python, v2)* |

## Стек

- **Go 1.22+** — все сервисы кроме Hawkeye
- **Python** — Hawkeye (OCR) + ML-пайплайн генерации датасета
- **PostgreSQL** — основные данные (миграции через Flyway)
- **Redis** — кэш, OTP, rate limiting, WebSocket-сессии
- **ClickHouse** — аналитика, логи
- **MinIO** — файловое хранилище (фото работ, аватары)
- **Kafka** — брокер сообщений, Transactional Outbox
- **Docker Compose** — оркестрация всего стека

## Структура проекта

```
edium-backend/
├── doorman/        # Auth: OTP, JWT
├── caesar/         # Пользователи, классы
├── tate/           # Курсы
├── riddler/        # Квизы + WebSocket
├── yoda/           # Работы (v2)
├── herald/         # Уведомления
├── charon/         # LLM-прокси
├── hawkeye/        # OCR (Python, v2)
├── ml/             # Пайплайн генерации квиз-датасета
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
- `POST doorman/v1/otp/send` — отправка OTP (повтор через 1 минуту)
- `POST doorman/v1/otp/verify` — проверка OTP (6-значный код)
- `POST doorman/v1/auth/register` — регистрация
- `POST doorman/v1/auth/refresh` — обновление токенов
- `POST doorman/v1/auth/logout` — выход

**Телефон:** regex `^\+[1-9]\d{1,14}$`

## Riddler — квизы и WebSocket

**Типы квизов:**
- **Синхронный** — все на одном вопросе, учитель видит прогресс, после ответов всех — статистика
- **Асинхронный** — каждый в своём темпе, учитель видит real-time прогресс

**WebSocket-события:**
- `participant_joined` / `participant_left` — лобби
- `quiz_started` — старт квиза
- `answer_submitted` — ученик ответил
- `question_stats` — статистика по вопросу (синхронный режим)
- `participant_disqualified` — кик участника
- `quiz_completed` — квиз завершён

**Настройки квиза:** таймер, дедлайн, лимит участников, отложенный старт, вход по QR/коду (6-значный числовой)

## ML-пайплайн (ml/)

**Флоу:** `PDF + chapters.txt → Surya OCR / PyMuPDF → GPT-4o-mini → dataset.jsonl`

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

- Все эндпоинты (кроме auth) требуют JWT — проверять каждый запрос на сервере
- Не логировать персональные данные (телефоны, имена)
- Не хардкодить секреты — только через env
- Текст вопросов квиза не передавать клиенту до старта сессии

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

## Соглашения

- **Язык:** комментарии, git-сообщения, документация — на русском
- **Ошибки Go:** `fmt.Errorf("контекст: %w", err)`
- **HTTP-коды:** 200, 400, 401, 422, 500
- **ID:** 8-значный hex (пользователи, классы), 6-значный числовой (быстрые квизы)
- **Миграции:** Flyway, `V{N}__{описание}.sql`, только append-only

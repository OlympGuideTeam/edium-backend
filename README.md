# edium-backend

Серверная часть образовательной платформы Edium — Go-микросервисы + Python ML-пайплайн.

## Архитектура

```mermaid
flowchart TD
    Client(["Клиент\n(iOS / Android)"])

    subgraph ingress ["Ingress"]
        Caddy["Caddy"]
    end

    subgraph services ["Go-сервисы"]
        Doorman["Doorman"]
        Caesar["Caesar"]
        Riddler["Riddler"]
        Herald["Herald"]
        Charon["Charon"]
        Sphinx["Sphinx"]
    end

    subgraph infra ["Инфраструктура"]
        NATS(["NATS JetStream"])
        PG[("PostgreSQL")]
        Redis[("Redis")]
    end

    TG["Telegram Bot API"]
    DS["DeepSeek API"]

    Client --> Caddy
    Caddy --> Doorman & Caesar & Riddler

    Caesar & Riddler -.-> Doorman

    Doorman & Caesar & Riddler & Herald & Charon --> PG
    Doorman & Charon --> Redis

    Doorman & Herald & Caesar & Riddler & Charon & Sphinx <-.-> NATS

    Herald --> TG
    Charon --> DS
```

> Пунктир — асинхронные события через NATS. Детали топиков — в `<service>/SCHEMA.md`.

## Сервисы

| Сервис | Описание                                    |
|--------|---------------------------------------------|
| **Doorman** | Аутентификация: OTP, JWT (access + refresh) |
| **Caesar** | Пользователи, классы, курсы, модули         |
| **Riddler** | Квизы, сессии прохождения, результаты       |
| **Herald** | Уведомления, Telegram bot, SMS              |
| **Charon** | Прокси к LLM (DeepSeek / OpenAI)            |
| **Sphinx** | Генерация квизов по тексту (Python + LLM)   |

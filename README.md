# edium-backend

Серверная часть образовательной платформы Edium — Go-микросервисы + Python ML-пайплайн.

## Архитектура

```mermaid
flowchart TD
    Client(["Клиент\n(iOS / Android)"])
    NATS(["NATS JetStream"])
    TG["Telegram Bot API"]
    DS["DeepSeek API"]

    subgraph ingress ["Ingress"]
        Caddy["Caddy"]
    end

    subgraph services ["Go-сервисы"]
        Doorman["Doorman\nPG · Redis"]
        Caesar["Caesar\nPG"]
        Riddler["Riddler\nPG · Redis"]
        Herald["Herald\nPG"]
        Charon["Charon\nPG · Redis"]
        Sphinx["Sphinx"]
    end

    Client --> Caddy
    Caddy --> Doorman & Caesar & Riddler

    Caesar & Riddler -.-> Doorman

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

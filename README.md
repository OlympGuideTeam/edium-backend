# edium-backend

Серверная часть образовательной платформы Edium — Go-микросервисы + Python ML-пайплайн.

## Архитектура

```mermaid
flowchart TD
    Client(["Клиент\n(iOS / Android)"])

    subgraph ingress ["Ingress"]
        Caddy["Caddy\nTLS · роутинг"]
    end

    subgraph services ["Go-сервисы"]
        Doorman["Doorman\nOTP · JWT · Auth"]
        Caesar["Caesar\nКурсы · Классы · Пользователи"]
        Riddler["Riddler\nКвизы · Сессии · Попытки"]
        Herald["Herald\nУведомления"]
        Charon["Charon\nLLM Proxy"]
        Sphinx["Sphinx\nГенерация квизов (Python)"]
    end

    subgraph infra ["Инфраструктура"]
        NATS(["NATS JetStream"])
        PG[("PostgreSQL")]
        Redis[("Redis")]
    end

    subgraph external ["Внешние"]
        TG["Telegram Bot API"]
        VK["VK Bot API"]
        DS["DeepSeek API"]
    end

    Client --> Caddy
    Caddy -->|"/doorman/v1"| Doorman
    Caddy -->|"/caesar/v1"| Caesar
    Caddy -->|"/riddler/v1"| Riddler

    Caesar & Riddler -.->|"JWKS (HTTP)"| Doorman

    Doorman & Caesar & Riddler & Herald & Charon --> PG
    Doorman & Charon --> Redis

    Doorman <-.->|"otp · user"| NATS
    Herald  <-.->|"otp"| NATS
    Caesar  <-.->|"user · quiz progress"| NATS
    Riddler <-.->|"sessions · attempts · grading"| NATS
    Charon  <-.->|"grading"| NATS
    Sphinx  <-.->|"generation"| NATS

    Herald --> TG & VK
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

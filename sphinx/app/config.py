from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    # HuggingFace
    model_id: str = "mistralai/Mistral-7B-Instruct-v0.3"
    # TODO: раскомментировать после обучения адаптеров
    # extraction_adapter: str = "edium/sphinx-extraction-lora"
    # generation_adapter: str = "edium/sphinx-generation-lora"
    hf_token: str = ""

    # Inference
    max_new_tokens: int = 2048
    temperature: float = 0.3
    top_p: float = 0.9
    max_retries: int = 2

    # NATS (transactional outbox/inbox с Riddler)
    nats_url: str = "nats://sphinx:password@nats:4222"
    # Путь к CA-сертификату для TLS (stunnel на платформенной VM).
    # Оставить пустым, если TLS не используется (dev / прямое подключение).
    nats_tls_ca_path: str = ""

    # PostgreSQL для outbox/inbox таблиц
    postgres_dsn: str = "postgresql://sphinx:sphinx@sphinx-db:5432/sphinx"

    # Интервал опроса воркеров (секунды)
    worker_poll_interval: float = 2.0

    # Восстановление зависших задач: таймаут обработки и максимум попыток
    processing_timeout_minutes: int = 30
    max_task_attempts: int = 3

    model_config = {"env_file": ".env", "case_sensitive": False}


settings = Settings()

from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    # HuggingFace
    model_id: str = "Qwen/Qwen2.5-14B-Instruct"
    # TODO: раскомментировать после обучения адаптеров
    # extraction_adapter: str = "edium/sphinx-extraction-lora"
    # generation_adapter: str = "edium/sphinx-generation-lora"
    hf_token: str = ""

    # Путь для кэша квантизованной модели (пусто — не кэшировать)
    quantized_model_cache_dir: str = "/root/.cache/huggingface/quantized"

    # Inference
    max_new_tokens: int = 2048
    temperature: float = 0.6
    top_p: float = 0.9
    max_retries: int = 2

    # NATS prod (transactional outbox/inbox с Riddler)
    nats_url: str = "nats://sphinx:password@nats:4222"
    nats_tls_ca_path: str = ""

    # NATS test — отдельный брокер; оставить пустым, чтобы не подключаться
    nats_url_test: str = ""
    nats_tls_ca_path_test: str = ""

    # PostgreSQL для outbox/inbox таблиц
    postgres_dsn: str = "postgresql://sphinx:sphinx@sphinx-db:5432/sphinx"

    # Интервал опроса воркеров (секунды)
    worker_poll_interval: float = 2.0

    # Восстановление зависших задач: таймаут обработки и максимум попыток
    processing_timeout_minutes: int = 30
    max_task_attempts: int = 3

    model_config = {"env_file": ".env", "case_sensitive": False}


settings = Settings()

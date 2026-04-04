from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    grpc_port: int = 50051
    max_workers: int = 4

    # HuggingFace
    model_id: str = "mistralai/Mistral-7B-Instruct-v0.3"
    extraction_adapter: str = "edium/sphinx-extraction-lora"
    generation_adapter: str = "edium/sphinx-generation-lora"
    hf_token: str = ""

    # Auth — API-ключ для межсервисного доступа
    api_key: str = ""

    # Inference
    max_new_tokens: int = 2048
    temperature: float = 0.3
    top_p: float = 0.9
    max_retries: int = 2

    model_config = {"env_file": ".env", "case_sensitive": False}


settings = Settings()

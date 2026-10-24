from functools import lru_cache
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )

    APP_NAME: str = "ReqSentinel ML Service"
    APP_VERSION: str = "0.2.0"

    # Token nội bộ: API service (Go) gửi qua header X-Internal-Token.
    # Để trống = không kiểm tra (chỉ nên dùng khi chạy local).
    INTERNAL_TOKEN: str = ""
    # Load model ngay khi start để request đầu tiên không bị cold-start.
    WARMUP_ON_STARTUP: bool = True
    # Ghi đè nhãn phiên bản model (để trống = tự sinh từ extractor + checkpoint + SBERT)
    MODEL_VERSION: str = ""

    DEBERTA_MODEL_NAME: str = "microsoft/deberta-v3-base"
    DEBERTA_MODEL_DIR: str = "data/models/deberta-extractor"
    SBERT_MODEL_NAME: str = "sentence-transformers/all-MiniLM-L6-v2"
    MODEL_CACHE_DIR: str = "data/models"

    ALIGNMENT_THRESHOLD: float = 0.55
    ROLE_WEIGHT: float = 0.35
    ACTION_WEIGHT: float = 0.35
    RESOURCE_WEIGHT: float = 0.30

    LOG_LEVEL: str = "INFO"


@lru_cache
def get_settings() -> Settings:
    return Settings()

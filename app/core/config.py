from functools import lru_cache
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
    )

    APP_NAME: str = "Hybrid-ACCheck"
    APP_VERSION: str = "0.1.0"
    DEBUG: bool = True
    API_V1_PREFIX: str = "/api/v1"

    DATABASE_URL: str = "postgresql+asyncpg://postgres:postgres@localhost:5432/hybrid_accheck"
    DATABASE_ECHO: bool = False

    DEBERTA_MODEL_NAME: str = "microsoft/deberta-v3-base"
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
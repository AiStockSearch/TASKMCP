"""Environment-backed settings."""

from functools import lru_cache
from typing import Optional

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    database_url: str = Field(validation_alias="DATABASE_URL")
    redis_url: str = Field(default="redis://127.0.0.1:6379/0", validation_alias="REDIS_URL")
    webhook_secret: Optional[str] = Field(default=None, validation_alias="WEBHOOK_SECRET")
    api_host: str = Field(default="0.0.0.0", validation_alias="API_HOST")
    api_port: int = Field(default=8080, validation_alias="API_PORT")
    arq_queue_name: Optional[str] = Field(default=None, validation_alias="ARQ_QUEUE_NAME")


@lru_cache
def get_settings() -> Settings:
    return Settings()

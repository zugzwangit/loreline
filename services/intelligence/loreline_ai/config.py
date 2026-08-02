from __future__ import annotations

from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_prefix="LORELINE_", extra="ignore")
    database_url: str = ""
    ai_addr: str = "0.0.0.0:8090"
    worker_id: str = "intelligence-1"
    worker_poll_seconds: float = 1.0
    object_endpoint: str = ""
    object_bucket: str = "loreline-documents"
    object_access_key: str = ""
    object_secret_key: str = ""
    provider: str = "extractive"
    provider_base_url: str = "https://api.openai.com/v1"
    provider_api_key: str = ""
    provider_model: str = ""
    max_question_chars: int = 4000
    max_documents: int = 25
    chunk_chars: int = 1200
    chunk_overlap: int = 160
    connector_allowed_hosts: str = ""
    connector_allow_http: bool = False
    allow_database_only: bool = False

    @property
    def allowed_connector_hosts(self) -> set[str]:
        return {host.strip().lower() for host in self.connector_allowed_hosts.split(",") if host.strip()}

    @property
    def host_port(self) -> tuple[str, int]:
        host, port = self.ai_addr.rsplit(":", 1)
        return host, int(port)


@lru_cache
def settings() -> Settings:
    return Settings()

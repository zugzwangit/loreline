from __future__ import annotations

from typing import Any

from loreline_ai.config import Settings, settings

from .base import Connector
from .confluence import ConfluenceConnector
from .http_json import HTTPJSONConnector
from .zendesk import ZendeskConnector


def build_connector(config: dict[str, Any], cfg: Settings | None = None) -> Connector:
    runtime = cfg or settings()
    provider = config.get("provider", "http_json")
    if provider == "http_json":
        return HTTPJSONConnector(config, runtime)
    if provider == "zendesk":
        return ZendeskConnector(config, runtime)
    if provider == "confluence":
        return ConfluenceConnector(config, runtime)
    raise ValueError(f"unsupported connector provider: {provider}")


__all__ = ["build_connector", "Connector"]

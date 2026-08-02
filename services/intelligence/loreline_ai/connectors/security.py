from __future__ import annotations

import os
from typing import Any
from urllib.parse import urlsplit

from loreline_ai.config import Settings


def validate_url(value: str, cfg: Settings) -> str:
    parsed = urlsplit(value)
    schemes = {"https", "http"} if cfg.connector_allow_http else {"https"}
    host = (parsed.hostname or "").lower()
    if parsed.scheme not in schemes or not host or parsed.username or parsed.password:
        raise ValueError("connector URL must use an approved scheme and cannot contain credentials")
    if not cfg.allowed_connector_hosts or host not in cfg.allowed_connector_hosts:
        raise ValueError(f"connector host {host!r} is not in LORELINE_CONNECTOR_ALLOWED_HOSTS")
    return value


def credential(config: dict[str, Any]) -> str:
    name = str(config.get("credential_env", ""))
    if not name.startswith("LORELINE_CONNECTOR_") or len(name) > 128:
        raise ValueError("connector credential_env must use the LORELINE_CONNECTOR_ prefix")
    value = os.getenv(name, "")
    if not value:
        raise ValueError(f"connector credential {name!r} is not configured")
    return value

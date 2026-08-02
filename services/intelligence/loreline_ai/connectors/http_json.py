from __future__ import annotations

from typing import Any

import httpx

from loreline_ai.config import Settings

from .base import SourceRecord
from .security import credential, validate_url


def nested(value: Any, path: str, default: Any = "") -> Any:
    for part in path.split("."):
        if not isinstance(value, dict):
            return default
        value = value.get(part, default)
    return value


class HTTPJSONConnector:
    def __init__(self, config: dict[str, Any], cfg: Settings):
        self.config = config
        self.url = validate_url(str(config["url"]), cfg)
        self.records_path = str(config.get("records_path", "items"))
        self.id_path = str(config.get("id_path", "id"))
        self.title_path = str(config.get("title_path", "title"))
        self.content_path = str(config.get("content_path", "content"))
        self.cursor_path = str(config.get("cursor_path", "next_cursor"))
        self.credential = credential(config)

    def fetch(self, cursor: dict[str, Any]) -> tuple[list[SourceRecord], dict[str, Any]]:
        headers = {"Accept": "application/json"}
        if self.credential:
            headers["Authorization"] = "Bearer " + self.credential
        params = {"cursor": cursor["value"]} if cursor.get("value") else {}
        response = httpx.get(self.url, headers=headers, params=params, timeout=30, follow_redirects=False)
        response.raise_for_status()
        data = response.json()
        raw = nested(data, self.records_path, [])
        if not isinstance(raw, list):
            raise ValueError("connector records_path did not resolve to a list")
        records = [
            SourceRecord(
                str(nested(item, self.id_path)),
                str(nested(item, self.title_path, "Untitled")),
                str(nested(item, self.content_path)),
                metadata={"connector": "http_json"},
            )
            for item in raw
        ]
        next_value = nested(data, self.cursor_path, "")
        return records, {"value": next_value} if next_value else cursor

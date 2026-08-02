from __future__ import annotations

from typing import Any

import httpx

from loreline_ai.config import Settings

from .base import SourceRecord
from .security import credential, validate_url


class ConfluenceConnector:
    def __init__(self, config: dict[str, Any], cfg: Settings):
        self.base = validate_url(str(config["base_url"]).rstrip("/"), cfg)
        self.space = str(config.get("space_key", ""))
        self.token = credential(config)

    def fetch(self, cursor: dict[str, Any]) -> tuple[list[SourceRecord], dict[str, Any]]:
        start = int(cursor.get("start", 0))
        params = {"start": start, "limit": 100, "expand": "body.storage,version", "type": "page"}
        if self.space:
            params["spaceKey"] = self.space
        response = httpx.get(
            self.base + "/rest/api/content",
            params=params,
            headers={"Authorization": "Bearer " + self.token, "Accept": "application/json"},
            timeout=30,
            follow_redirects=False,
        )
        response.raise_for_status()
        data = response.json()
        records = [
            SourceRecord(
                str(page["id"]),
                str(page.get("title") or "Wiki page"),
                str(page.get("body", {}).get("storage", {}).get("value", "")),
                "text/html",
                {"version": page.get("version", {}).get("number"), "connector": "confluence"},
            )
            for page in data.get("results", [])
        ]
        next_start = start + len(records)
        return records, {"start": next_start} if data.get("_links", {}).get("next") else {}

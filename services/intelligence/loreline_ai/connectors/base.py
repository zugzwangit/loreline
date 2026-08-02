from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Protocol


@dataclass(frozen=True)
class SourceRecord:
    external_id: str
    title: str
    content: str
    media_type: str = "text/plain"
    metadata: dict[str, Any] | None = None


class Connector(Protocol):
    def fetch(self, cursor: dict[str, Any]) -> tuple[list[SourceRecord], dict[str, Any]]: ...

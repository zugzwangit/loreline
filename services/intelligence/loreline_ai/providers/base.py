from __future__ import annotations

from typing import Protocol

from loreline_ai.models import Answer, Document


class Provider(Protocol):
    name: str

    async def answer(self, question: str, documents: list[Document]) -> Answer: ...

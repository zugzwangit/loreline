from __future__ import annotations

from typing import Any
from pydantic import BaseModel, Field, field_validator


class Document(BaseModel):
    id: str
    title: str
    content: str
    source: str = "Unknown"

class RetrieveRequest(BaseModel):
    question: str = Field(min_length=1, max_length=4000)
    documents: list[Document] = Field(max_length=1000)

    @field_validator("question")
    @classmethod
    def clean_question(cls, value: str) -> str:
        value = value.strip()
        if not value:
            raise ValueError("question cannot be blank")
        return value

class Citation(BaseModel):
    id: str
    title: str
    source: str

class Answer(BaseModel):
    answer: str
    confidence: float = Field(ge=0, le=1)
    citations: list[Citation]
    grounded: bool
    provider: str = "extractive"
    metadata: dict[str, Any] = Field(default_factory=dict)

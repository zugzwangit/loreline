from __future__ import annotations
from loreline_ai.engine import answer_question
from loreline_ai.models import Answer, Document

class ExtractiveProvider:
    name = "extractive"
    async def answer(self, question: str, documents: list[Document]) -> Answer:
        result = answer_question(question, [doc.model_dump() for doc in documents])
        result["provider"] = self.name
        return Answer.model_validate(result)

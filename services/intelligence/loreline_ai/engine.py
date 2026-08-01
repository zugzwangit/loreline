from __future__ import annotations

import math
import re
from collections import Counter
from typing import Any

TOKEN = re.compile(r"[a-z0-9]+")
STOPWORDS = {"a", "an", "and", "are", "do", "for", "how", "i", "in", "is", "of", "or", "the", "to", "what", "with"}


def tokenize(value: str) -> list[str]:
    return [word for word in TOKEN.findall(value.lower()) if word not in STOPWORDS]


def score(question: str, document: dict[str, Any]) -> float:
    query = Counter(tokenize(question))
    text = Counter(tokenize(f"{document.get('title', '')} {document.get('content', '')}"))
    if not query or not text:
        return 0.0
    overlap = sum(min(count, text.get(term, 0)) for term, count in query.items())
    title_terms = set(tokenize(document.get("title", "")))
    title_bonus = sum(0.7 for term in query if term in title_terms)
    norm = math.sqrt(sum(v * v for v in query.values()) * sum(v * v for v in text.values()))
    return (overlap + title_bonus) / norm if norm else 0.0


def answer_question(question: str, documents: list[dict[str, Any]], limit: int = 3) -> dict[str, Any]:
    ranked = sorted(((score(question, doc), doc) for doc in documents), key=lambda row: row[0], reverse=True)
    selected = [(value, doc) for value, doc in ranked if value > 0][:limit]
    if not selected:
        return {
            "answer": "I could not find an approved knowledge source that answers that question. Try adding more detail or send it to the review team.",
            "confidence": 0.0,
            "citations": [],
            "grounded": False,
        }
    statements = [doc.get("content", "").strip() for _, doc in selected if doc.get("content", "").strip()]
    answer = " ".join(statements)
    confidence = min(0.99, 0.72 + sum(value for value, _ in selected) / (4 * len(selected)))
    citations = [{"id": doc.get("id", ""), "title": doc.get("title", "Untitled"), "source": doc.get("source", "Unknown")} for _, doc in selected]
    return {"answer": answer, "confidence": round(confidence, 2), "citations": citations, "grounded": True}

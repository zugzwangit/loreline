from __future__ import annotations

import hashlib
import html
import json
import re
from dataclasses import dataclass

TAG = re.compile(r"<[^>]+>")
SPACE = re.compile(r"\s+")


def extract_text(content: str, media_type: str) -> str:
    if media_type in {"application/json", "application/x-ndjson"}:
        try:
            value = json.loads(content)
            content = json.dumps(value, ensure_ascii=False, indent=2)
        except json.JSONDecodeError as exc:
            raise ValueError("invalid JSON document") from exc
    elif media_type in {"text/html", "application/xhtml+xml"}:
        content = html.unescape(TAG.sub(" ", content))
    return SPACE.sub(" ", content.replace("\x00", " ")).strip()


def chunks(text: str, size: int = 1200, overlap: int = 160) -> list[str]:
    if size < 200 or overlap < 0 or overlap >= size:
        raise ValueError("invalid chunk configuration")
    out: list[str] = []
    start = 0
    while start < len(text):
        end = min(len(text), start + size)
        if end < len(text):
            boundary = max(text.rfind(". ", start, end), text.rfind("\n", start, end))
            if boundary > start + size // 2:
                end = boundary + 1
        part = text[start:end].strip()
        if part:
            out.append(part)
        if end >= len(text):
            break
        start = max(start + 1, end - overlap)
    return out


def fingerprint(value: str) -> str:
    return hashlib.sha256(value.encode("utf-8")).hexdigest()


def embedding(value: str, dimensions: int = 64) -> list[float]:
    vector = [0.0] * dimensions
    terms = re.findall(r"[a-z0-9]+", value.lower())
    for term in terms:
        digest = hashlib.blake2b(term.encode(), digest_size=8).digest()
        slot = int.from_bytes(digest[:4], "big") % dimensions
        vector[slot] += 1.0 if digest[4] & 1 else -1.0
    norm = sum(v * v for v in vector) ** 0.5
    return [round(v / norm, 8) for v in vector] if norm else vector


@dataclass(frozen=True)
class CandidateResult:
    title: str
    content: str
    confidence: float
    chunks: list[str]


def normalize(title: str, content: str, media_type: str, chunk_size: int, overlap: int) -> CandidateResult:
    text = extract_text(content, media_type)
    if len(text) < 20:
        raise ValueError("document has insufficient extractable text")
    parts = chunks(text, chunk_size, overlap)
    confidence = min(0.99, 0.72 + min(len(text), 5000) / 25000 + (0.05 if len(parts) > 1 else 0))
    return CandidateResult(title=SPACE.sub(" ", title).strip(), content=text, confidence=round(confidence, 3), chunks=parts)

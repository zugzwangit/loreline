from __future__ import annotations
import json
import httpx
from loreline_ai.models import Answer, Citation, Document

class OpenAICompatibleProvider:
    name = "openai-compatible"
    def __init__(self, base_url: str, api_key: str, model: str):
        if not api_key or not model:
            raise ValueError("provider_api_key and provider_model are required")
        self.base_url, self.api_key, self.model = base_url.rstrip("/"), api_key, model

    async def answer(self, question: str, documents: list[Document]) -> Answer:
        evidence = "\n\n".join(f"[{i+1}] {d.title}\n{d.content}" for i,d in enumerate(documents[:8]))
        prompt = "Answer only from the supplied evidence. If evidence is insufficient, say so. Return JSON with answer and citation_numbers.\n\nQUESTION:\n"+question+"\n\nEVIDENCE:\n"+evidence
        async with httpx.AsyncClient(timeout=30) as client:
            response = await client.post(self.base_url+"/chat/completions",headers={"Authorization":"Bearer "+self.api_key},json={"model":self.model,"temperature":0,"response_format":{"type":"json_object"},"messages":[{"role":"system","content":"You are a grounded enterprise knowledge assistant."},{"role":"user","content":prompt}]})
            response.raise_for_status(); data=response.json()
        parsed=json.loads(data["choices"][0]["message"]["content"]); numbers=[n for n in parsed.get("citation_numbers",[]) if isinstance(n,int) and 1<=n<=len(documents[:8])]
        citations=[Citation(id=documents[n-1].id,title=documents[n-1].title,source=documents[n-1].source) for n in numbers]
        grounded=bool(citations); return Answer(answer=str(parsed.get("answer","")),confidence=0.9 if grounded else 0.0,citations=citations,grounded=grounded,provider=self.name,metadata={"model":self.model})

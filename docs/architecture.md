# Loreline architecture

Loreline separates the fast public API path from retrieval logic and the browser experience.

```text
Browser console
    │
    ▼
Go gateway ───────► Python intelligence service
    │                  │
    ▼                  ▼
review state       retrieval + grounded answers
```

## Packages

- `app/` is the responsive operations console and interactive assistant demo.
- `services/gateway/` is a Go HTTP API for dashboard data, knowledge lifecycle decisions, CORS, validation, and request routing.
- `services/intelligence/` is a dependency-light Python package for deterministic ranking, grounded answer composition, and citations.

The demo store is intentionally in memory, behind a small `Store` boundary. A production adapter can replace it with PostgreSQL without changing the handlers. The deterministic retrieval engine also provides a safe local default; a provider-backed generator can be added behind `answer_question` while preserving the same response contract.

## API surface

| Method | Route | Purpose |
|---|---|---|
| GET | `/health` | Gateway health |
| GET | `/v1/dashboard` | Operational metrics |
| GET | `/v1/knowledge?status=review` | Filtered knowledge list |
| GET | `/v1/knowledge/{id}` | Knowledge detail |
| POST | `/v1/knowledge/{id}/decision` | Approve, edit, or reject |
| POST | `/v1/assistant/ask` | Retrieve a grounded, cited answer |

Every answer is built only from approved records supplied by the gateway. When retrieval finds no relevant source, the service returns an explicit ungrounded response with no citations.

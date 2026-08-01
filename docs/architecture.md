# Loreline architecture

```text
Signed-in web console
        │ server-side authenticated proxy
        ▼
Go control plane ─────────────► Python intelligence API
   │       │                         │
   │       └── approved records ─────┘
   ▼
PostgreSQL ◄──── Python workers ────► encrypted object storage
   │                 │
   ├─ tenants        ├─ normalize / chunk / deduplicate
   ├─ RBAC + keys    ├─ deterministic embeddings
   ├─ sources/docs   ├─ index / retire / retry
   ├─ durable jobs   └─ provider abstraction
   ├─ review state
   ├─ messages
   └─ audit/feedback/outbox
```

The Go control plane owns identity, tenant isolation, authorization, request validation, transactional lifecycle changes, audit, idempotency, conversations, feedback, and service orchestration. It is stateless outside PostgreSQL.

The Python worker claims durable jobs with row locks. Ingestion extracts supported text, normalizes whitespace, generates overlapping semantic chunks, calculates content fingerprints and deterministic embeddings, detects exact duplicates, stores the raw source in encrypted object storage, and sends the candidate to human review. Approval activates chunks and enqueues indexing; retirement deactivates them without erasing audit history.

The intelligence API accepts only approved documents selected by the control plane. Its local provider uses weighted lexical and semantic retrieval and never invents an answer without evidence. An OpenAI-compatible provider can be enabled through configuration without changing the gateway contract. Answers, citations, confidence, latency, conversations, and feedback are persisted.

The web console calls only its same-origin server proxy. Production credentials remain server-side. Without a configured gateway the UI explicitly identifies demo data instead of presenting it as durable state.

See [the OpenAPI contract](../api/openapi.yaml), [production operations](production.md), and [security model](security.md).

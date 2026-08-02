# Production operations

## Service guarantees

Loreline targets 99.9% monthly availability for reads and answers, a 99.5% ingestion success rate, p95 answer time below 3 seconds excluding an external generation provider, and a recovery point objective of 15 minutes with a recovery time objective of 60 minutes.

The gateway fails closed when the database or credentials are absent. Every `/v1` request is tenant-scoped, authenticated, rate limited, assigned a request ID, and logged. Admin and review writes are authorized by role and written to the audit log in the same transaction as the state change.

## Production prerequisites

- Managed PostgreSQL 16 with point-in-time recovery, TLS required, encryption at rest, and at least one standby
- S3-compatible object storage with versioning, server-side encryption, lifecycle rules, and public access disabled
- A secret manager supplying database, object-store, service API, and optional generation-provider credentials
- TLS termination, a private network between services, DNS, and an ingress/WAF
- a private Sites access policy plus `LORELINE_ALLOWED_USER_EMAILS` for console authorization
- Prometheus-compatible metrics collection and centralized JSON logs

Do not deploy the example secrets or local Compose passwords. Run `lorelinectl migrate` as a single pre-deployment job, then deploy the API and workers. Run `lorelinectl bootstrap <slug> <name>` once to create the first tenant and owner API key; capture the key at creation because only its SHA-256 digest is stored.

Set `LORELINE_CONNECTOR_ALLOWED_HOSTS` to the exact external hostnames workers may contact. Connector credentials must be injected under names beginning with `LORELINE_CONNECTOR_`; source records store only those environment-variable references. Workers fail startup when object storage is absent unless the explicit test-only database mode is enabled.

## Scaling

The gateway is stateless and horizontally scalable. Workers claim jobs using `FOR UPDATE SKIP LOCKED`, so replicas safely share a queue. Intelligence API replicas are stateless. Scale gateway and intelligence on CPU and latency; scale workers on queue age and queued job count. PostgreSQL pool capacity is 20 connections per gateway replica, so cap replicas against the database connection budget or insert a transaction pooler.

## Deployment safety

1. Back up the database and confirm object versioning.
2. Run the forward-only migration job once.
3. Deploy workers, intelligence, then gateway with a rolling strategy.
4. Require `/livez` and `/readyz` before receiving traffic.
5. Run the smoke flow: source creation, idempotent ingest, candidate approval, grounded answer, feedback, audit lookup.
6. Monitor 5xx rate, p95 latency, database saturation, job age, dead jobs, and ungrounded answer rate for at least 15 minutes.

Database migrations are additive in normal releases. Destructive changes require an expand/migrate/contract sequence across separate releases.

## Retention

Recommended defaults: audit events 400 days, conversations 180 days, rejected candidates 90 days, retired document metadata 400 days, and raw source objects according to the source system’s legal policy. Apply retention with scheduled jobs; never delete active approved chunks before their replacement is indexed.

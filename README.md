# Loreline

Loreline is a production-oriented, multi-tenant knowledge operations platform. It ingests support tickets, policies, wiki pages, uploads, and API records; normalizes and deduplicates them in durable workers; routes candidates through human approval; indexes approved knowledge; and answers employee questions with traceable citations and feedback.

## System packages

- `services/gateway`: Go control plane, tenant isolation, hashed API-key authentication, RBAC, PostgreSQL migrations, idempotent writes, audit, rate limits, health, metrics, and graceful shutdown
- `services/intelligence`: Python FastAPI retrieval service, provider abstraction, SSE streaming, text extraction, chunking, fingerprinting, embeddings, and durable job worker
- `app`: responsive operations console with a server-only authenticated API proxy; it clearly reports whether it is connected to durable services
- `api/openapi.yaml`: versioned HTTP contract
- `deploy`: Compose stack, TLS edge, Prometheus/Grafana configuration, and hardened Kubernetes resources
- `docs`: architecture, production, security, backup/restore, and incident response documentation

## Local operational stack

Requirements: Docker Compose, or Node 22 + pnpm 11 + Go 1.23 + Python 3.12 + PostgreSQL 16.

```bash
cp .env.example .env
# Replace every example secret, then:
docker compose up --build -d
docker compose exec gateway /lorelinectl bootstrap acme "Acme Labs"
```

The bootstrap command prints the only copy of the first owner API key. Store it in a secret manager. Configure the console proxy with `LORELINE_GATEWAY_URL` and `LORELINE_GATEWAY_TOKEN`; never expose the token through a `NEXT_PUBLIC_` variable.

For UI development:

```bash
pnpm install
LORELINE_GATEWAY_URL=http://localhost:8088 LORELINE_GATEWAY_TOKEN=ll_live_xxx LORELINE_ALLOW_ANONYMOUS_PROXY=true pnpm dev
```

Anonymous proxy mode is for local development only. The hosted private console uses the platform-authenticated viewer headers.

## Verification

```bash
pnpm test
cd services/intelligence && pip install -e '.[test]' && pytest --cov=loreline_ai
cd ../gateway && go test -race ./...
LORELINE_TEST_PYTHON=python go test -tags=integration ./internal/controlplane -run TestProductionLifecycle -v
```

The integration test launches an isolated PostgreSQL instance and exercises real migrations, source creation, durable ingestion, the Python worker, candidate review, answer persistence, feedback, and audit retrieval.

## Production deployment

Use managed PostgreSQL and S3-compatible storage, a secret manager, TLS ingress/WAF, and centralized logs. Apply [deploy/kubernetes.yaml](deploy/kubernetes.yaml) after replacing image names, hostname, and the example secret. Run the migration Job before rolling out application pods.

Read these before deployment:

- [Architecture](docs/architecture.md)
- [Production operations and SLOs](docs/production.md)
- [Security model](docs/security.md)
- [Backup and restore](docs/runbooks/backup-restore.md)
- [Incident response](docs/runbooks/incident-response.md)
- [OpenAPI contract](api/openapi.yaml)

The repository is initialized on `main`, contains no application credentials, and includes CI, dependency updates, CodeQL, immutable migrations, container builds, and full-stack integration coverage.

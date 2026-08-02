# Loreline gateway

The gateway is the authoritative Go control plane. It owns authentication, RBAC, tenant isolation, source and document lifecycles, human-review decisions, assistant persistence, feedback, audit events, and durable job orchestration.

## Package map

- `cmd/server`: HTTP process and graceful lifecycle
- `cmd/lorelinectl`: migration and tenant bootstrap administration
- `internal/controlplane`: handlers, middleware, configuration, repository transactions, and tests
- `migrations`: immutable PostgreSQL migrations embedded into the administrative binary

## Verification

```bash
go test ./...
go vet ./...
govulncheck ./...
LORELINE_TEST_PYTHON=python go test -tags=integration ./internal/controlplane -run TestProductionLifecycle -v
```

Production startup requires `DATABASE_URL`. Development header authentication is rejected unless `LORELINE_ENVIRONMENT` is explicitly `development` or `test`.

# Repository layout

Loreline is a small polyglot monorepo. Each runtime has one clear owner and communicates through versioned HTTP and PostgreSQL contracts.

| Path | Ownership | Deployable artifact |
| --- | --- | --- |
| `app/` | TypeScript console, same-origin proxy, and Sites authentication | Cloudflare Worker bundle in `dist/` |
| `app/features/` | Feature-specific console screens | Included in the console bundle |
| `app/lib/` | Shared console domain types and explicit demo fixtures | Included in the console bundle |
| `worker/` | Console Worker entry point | Included in the console bundle |
| `tooling/` | Build-time Sites integration only | Not shipped as application code |
| `services/gateway/` | Go control plane, database migrations, authorization, and durable workflows | `loreline-gateway` and `lorelinectl` binaries |
| `services/intelligence/` | Python retrieval API, connectors, ingestion pipeline, and job worker | `loreline-intelligence` container |
| `api/` | Versioned OpenAPI contract between clients and the gateway | Documentation and contract validation |
| `deploy/` | Runtime-neutral production deployment manifests and observability configuration | Kubernetes, Caddy, Prometheus, and Grafana assets |
| `scripts/` | Operator-run smoke, backup, and restore commands | Operational tooling |
| `docs/` | Architecture, security, SLO, and runbook documentation | Operator documentation |

Generated directories such as `dist/`, `.vinext/`, `.wrangler/`, `node_modules/`, and `work/` are ignored and must never be committed.

## Dependency boundaries

- The browser calls only `app/api/loreline/`; it never receives a gateway service key.
- The Go gateway owns tenant identity, authorization, state transitions, and the public HTTP contract.
- Python workers own connector execution, normalization, chunking, object storage, and durable job execution.
- The intelligence HTTP service accepts only bounded evidence selected by the gateway.
- PostgreSQL migrations live only beside the Go gateway and are applied by `lorelinectl`.
- Demo fixtures live only in `app/lib/demo-data.ts` and are never returned by production APIs.

## Adding code

Place UI code in the closest `app/features/<feature>` directory, reusable UI domain code in `app/lib`, gateway behavior in `internal/controlplane`, and intelligence behavior in the corresponding Python subpackage. New cross-service fields must be added to `api/openapi.yaml` and covered by an integration test.

# Loreline

Trusted knowledge, ready to answer.

Loreline turns resolved support tickets, policy documents, and wiki pages into reviewed, searchable knowledge. It gives operations teams a human approval queue and gives employees fast answers with traceable citations.

## What is included

- A polished responsive console with overview, knowledge library, review queue, and cited assistant flows
- A Go gateway with knowledge lifecycle APIs, validation, CORS, health checks, and service orchestration
- A Python retrieval service with deterministic ranking, grounded answer generation, and citation contracts
- Unit, handler, integration, and server-render tests
- Dockerfiles, Compose, GitHub Actions, environment examples, and architecture documentation

## Quick start

Prerequisites: Node 22+, pnpm 11+, Go 1.23+, and Python 3.11+.

```bash
pnpm install
pnpm dev
```

The console opens at `http://localhost:3000`.

Run the services in two terminals:

```bash
cd services/intelligence
python -m loreline_ai.server
```

```bash
cd services/gateway
go run ./cmd/server
```

The gateway listens at `http://localhost:8080` and the intelligence service at `http://localhost:8090`. You can also start both with `docker compose up --build`.

## Verify everything

```bash
pnpm test
cd services/gateway && go test ./...
cd ../intelligence && python -m unittest discover -s tests -v
```

See [docs/architecture.md](docs/architecture.md) for service boundaries and API contracts.

## Repository status

The repository is initialized on `main`, includes a focused `.gitignore`, and has a CI workflow ready for GitHub. No credentials are required for the local demo.

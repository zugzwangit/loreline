# Loreline intelligence

This Python package contains two independently scalable processes:

- `loreline-intelligence`: bounded retrieval and grounded answer generation
- `loreline-worker`: durable ingestion, connector synchronization, indexing, retirement, retries, and worker heartbeats

The `connectors` package is an egress boundary. Connector URLs must use an allowed hostname and credentials must be referenced through dedicated `LORELINE_CONNECTOR_*` environment variables.

## Verification

```bash
pip install -r requirements.lock
pip install -e '.[test]'
ruff format --check loreline_ai tests
ruff check loreline_ai tests
pytest --cov=loreline_ai --cov-fail-under=60
pip-audit -r requirements.lock
```

`requirements.lock` is the hash-pinned production dependency set. Test and formatting tools remain optional development dependencies.

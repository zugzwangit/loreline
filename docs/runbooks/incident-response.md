# Incident response runbook

## Triage

Declare severity based on affected tenants, data integrity, and duration. Capture the first failing request ID, deployment version, job queue age, database health, provider health, and recent audit events. Freeze deployments for severity 1 or suspected data exposure.

## Containment

- Authentication failure: revoke the affected API key and rotate the UI service key.
- Cross-tenant concern: remove public traffic immediately, preserve logs, and verify every implicated query includes its tenant predicate.
- Bad answers: switch to the extractive provider, pause the affected source, and retire compromised chunks.
- Queue backlog: pause new sync jobs, scale workers, and inspect repeated `last_error` values before replaying dead jobs.
- Database saturation: shed nonessential audit-list and dashboard traffic, reduce replica pool counts, and activate the pooler.

## Recovery

Restore service in dependency order: PostgreSQL and object storage, intelligence, workers, gateway, UI proxy. Run the full smoke flow for an unaffected and affected tenant. Monitor for 30 minutes. Publish an incident timeline and corrective actions; never delete forensic audit or access logs during cleanup.

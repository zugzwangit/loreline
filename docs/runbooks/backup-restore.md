# Backup and restore runbook

1. Verify automated PostgreSQL snapshots and continuous WAL archiving every day. Alert if the newest recoverable point is older than 15 minutes.
2. Enable object-store versioning and cross-region replication. Keep database and object retention aligned.
3. Quarterly, restore both systems into an isolated environment using a timestamp at least 24 hours old.
4. Run migrations only if the restored schema predates the application image.
5. Validate tenant, document, candidate, chunk, conversation, feedback, audit, and job counts; sample object hashes against `documents.content_sha256`.
6. Start intelligence and workers, then gateway. Execute the end-to-end smoke flow and record actual recovery time and recovery point.

During an incident, stop ingestion before taking the final recovery point. Restore the database first, then the object bucket to the same timestamp. Jobs are at-least-once and idempotent: reset jobs left `running` longer than the lock timeout to `failed`, then allow workers to retry them.

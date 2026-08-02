DROP INDEX IF EXISTS jobs_active_source_sync_idx;
DROP INDEX IF EXISTS jobs_tenant_claim_idx;
ALTER TABLE candidates DROP COLUMN IF EXISTS reviewer_actor_id;
DROP TABLE IF EXISTS worker_heartbeats;
DELETE FROM schema_migrations WHERE version=2;

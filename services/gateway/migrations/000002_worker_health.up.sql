CREATE TABLE worker_heartbeats (
  worker_id text PRIMARY KEY,
  hostname text NOT NULL,
  started_at timestamptz NOT NULL DEFAULT now(),
  heartbeat_at timestamptz NOT NULL DEFAULT now(),
  version text NOT NULL
);

ALTER TABLE candidates ADD COLUMN reviewer_actor_id text;

CREATE INDEX jobs_tenant_claim_idx
  ON jobs (tenant_id,status,run_after,created_at)
  WHERE status IN ('queued','failed');
CREATE UNIQUE INDEX jobs_active_source_sync_idx
  ON jobs ((payload->>'source_id'))
  WHERE kind='sync' AND status IN ('queued','running','failed');

INSERT INTO schema_migrations(version) VALUES (2) ON CONFLICT DO NOTHING;

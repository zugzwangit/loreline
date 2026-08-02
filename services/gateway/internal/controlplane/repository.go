package controlplane

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")
var ErrValidation = errors.New("validation")

type Repository struct{ Pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{Pool: pool} }

func (r *Repository) ResolveAPIKey(ctx context.Context, raw string) (Principal, error) {
	sum := sha256.Sum256([]byte(raw))
	var p Principal
	err := r.Pool.QueryRow(ctx, `UPDATE api_keys SET last_used_at=now() WHERE key_hash=$1 AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at>now()) RETURNING tenant_id::text,id::text,role`, sum[:]).Scan(&p.TenantID, &p.ActorID, &p.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return Principal{}, ErrNotFound
	}
	return p, err
}

func (r *Repository) ListAPIKeys(ctx context.Context, tenant string) ([]map[string]any, error) {
	rows, err := r.Pool.Query(ctx, `SELECT id::text,name,key_prefix,role,expires_at,last_used_at,created_at FROM api_keys WHERE tenant_id=$1 AND revoked_at IS NULL ORDER BY created_at DESC`, tenant)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, name, prefix, role string
		var expires, last *time.Time
		var created time.Time
		if err = rows.Scan(&id, &name, &prefix, &role, &expires, &last, &created); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "name": name, "key_prefix": prefix, "role": role, "expires_at": expires, "last_used_at": last, "created_at": created})
	}
	return out, rows.Err()
}
func (r *Repository) CreateAPIKey(ctx context.Context, p Principal, name, role string, expires *time.Time, requestID string) (map[string]any, error) {
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, err
	}
	raw := "ll_live_" + base64.RawURLEncoding.EncodeToString(rawBytes)
	hash := sha256.Sum256([]byte(raw))
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO api_keys(tenant_id,name,key_prefix,key_hash,role,expires_at) VALUES($1,$2,$3,$4,$5,$6) RETURNING id::text`, p.TenantID, name, raw[:16], hash[:], role, expires).Scan(&id)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events(tenant_id,actor_id,action,resource_type,resource_id,request_id,after_state) VALUES($1,$2,'api_key.created','api_key',$3,$4,jsonb_build_object('name',$5::text,'role',$6::text))`, p.TenantID, p.ActorID, id, requestID, name, role); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "name": name, "role": role, "api_key": raw, "expires_at": expires}, nil
}
func (r *Repository) RevokeAPIKey(ctx context.Context, p Principal, id, requestID string) error {
	tag, err := r.Pool.Exec(ctx, `UPDATE api_keys SET revoked_at=now() WHERE id=$1 AND tenant_id=$2 AND revoked_at IS NULL`, id, p.TenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	_, err = r.Pool.Exec(ctx, `INSERT INTO audit_events(tenant_id,actor_id,action,resource_type,resource_id,request_id) VALUES($1,$2,'api_key.revoked','api_key',$3,$4)`, p.TenantID, p.ActorID, id, requestID)
	return err
}

func (r *Repository) Dashboard(ctx context.Context, tenant string) (map[string]any, error) {
	out := map[string]any{}
	var total, approved, review, jobs, workers, resolved int
	var quality, freshness float64
	var workspace string
	err := r.Pool.QueryRow(ctx, `SELECT
      (SELECT name FROM tenants WHERE id=$1),
      (SELECT count(*) FROM documents WHERE tenant_id=$1),
      (SELECT count(*) FROM candidates WHERE tenant_id=$1 AND status='approved'),
      (SELECT count(*) FROM candidates WHERE tenant_id=$1 AND status='pending'),
      (SELECT count(*) FROM jobs WHERE tenant_id=$1 AND status IN ('queued','running','failed')),
      COALESCE((SELECT avg(CASE rating WHEN 1 THEN 1.0 ELSE 0.0 END)*100 FROM feedback WHERE tenant_id=$1),100),
      (SELECT count(*) FROM worker_heartbeats WHERE heartbeat_at>now()-interval '45 seconds'),
      (SELECT count(*) FROM messages WHERE tenant_id=$1 AND role='assistant'),
      COALESCE((SELECT avg(CASE WHEN processed_at>now()-interval '90 days' THEN 100.0 ELSE 0.0 END) FROM documents WHERE tenant_id=$1 AND state='approved'),100)`, tenant).Scan(&workspace, &total, &approved, &review, &jobs, &quality, &workers, &resolved, &freshness)
	if err != nil {
		return nil, err
	}
	out["documents"] = total
	out["approved_knowledge"] = approved
	out["needs_review"] = review
	out["active_jobs"] = jobs
	out["answer_quality"] = quality
	out["workers_online"] = workers
	out["questions_resolved"] = resolved
	out["freshness"] = freshness
	out["workspace_name"] = workspace
	return out, nil
}

func (r *Repository) ListSources(ctx context.Context, tenant string) ([]Source, error) {
	rows, err := r.Pool.Query(ctx, `SELECT id::text,kind,name,status,config,last_synced_at,last_error,created_at FROM sources WHERE tenant_id=$1 AND status!='deleted' ORDER BY created_at DESC`, tenant)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Source{}
	for rows.Next() {
		var s Source
		if err = rows.Scan(&s.ID, &s.Kind, &s.Name, &s.Status, &s.Config, &s.LastSyncedAt, &s.LastError, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
func (r *Repository) CreateSource(ctx context.Context, p Principal, kind, name string, config json.RawMessage, requestID string) (Source, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return Source{}, err
	}
	defer tx.Rollback(ctx)
	var s Source
	err = tx.QueryRow(ctx, `INSERT INTO sources(tenant_id,kind,name,config) VALUES($1,$2,$3,$4) RETURNING id::text,kind,name,status,config,last_synced_at,last_error,created_at`, p.TenantID, kind, name, config).Scan(&s.ID, &s.Kind, &s.Name, &s.Status, &s.Config, &s.LastSyncedAt, &s.LastError, &s.CreatedAt)
	if err != nil {
		return Source{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events(tenant_id,actor_id,action,resource_type,resource_id,request_id,after_state) VALUES($1,$2,'source.created','source',$3,$4,$5)`, p.TenantID, p.ActorID, s.ID, requestID, toJSON(s)); err != nil {
		return Source{}, err
	}
	return s, tx.Commit(ctx)
}

func (r *Repository) EnqueueSync(ctx context.Context, p Principal, id, requestID string) (string, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var exists bool
	err = tx.QueryRow(ctx, `SELECT true FROM sources WHERE id=$1 AND tenant_id=$2 AND status='active' FOR UPDATE`, id, p.TenantID).Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	var activeJob string
	err = tx.QueryRow(ctx, `SELECT id::text FROM jobs WHERE tenant_id=$1 AND kind='sync' AND payload->>'source_id'=$2 AND status IN ('queued','running','failed') LIMIT 1`, p.TenantID, id).Scan(&activeJob)
	if err == nil {
		return "", fmt.Errorf("%w: source synchronization is already active", ErrConflict)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	var job string
	err = tx.QueryRow(ctx, `INSERT INTO jobs(tenant_id,kind,payload) VALUES($1,'sync',jsonb_build_object('source_id',$2::text)) RETURNING id::text`, p.TenantID, id).Scan(&job)
	if err != nil {
		return "", err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events(tenant_id,actor_id,action,resource_type,resource_id,request_id,after_state) VALUES($1,$2,'source.sync_queued','source',$3,$4,jsonb_build_object('job_id',$5::text))`, p.TenantID, p.ActorID, id, requestID, job); err != nil {
		return "", err
	}
	return job, tx.Commit(ctx)
}

type IngestInput struct {
	SourceID   string          `json:"source_id"`
	ExternalID string          `json:"external_id"`
	Title      string          `json:"title"`
	MediaType  string          `json:"media_type"`
	Content    string          `json:"content"`
	Metadata   json.RawMessage `json:"metadata"`
}

func (r *Repository) Ingest(ctx context.Context, p Principal, in IngestInput, idempotency, requestID string) (map[string]any, int, error) {
	contentSum := sha256.Sum256([]byte(in.Content))
	contentHash := hex.EncodeToString(contentSum[:])
	canonicalRequest, err := json.Marshal(in)
	if err != nil {
		return nil, 0, err
	}
	requestSum := sha256.Sum256(canonicalRequest)
	reqHash := hex.EncodeToString(requestSum[:])
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback(ctx)
	if in.SourceID != "" {
		var exists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sources WHERE id=$1 AND tenant_id=$2 AND status='active')`, in.SourceID, p.TenantID).Scan(&exists); err != nil {
			return nil, 0, err
		}
		if !exists {
			return nil, 0, ErrNotFound
		}
	}
	if idempotency != "" {
		var saved json.RawMessage
		var status int
		var stored string
		err = tx.QueryRow(ctx, `SELECT request_hash,status_code,response FROM idempotency_keys WHERE tenant_id=$1 AND key=$2 AND expires_at>now()`, p.TenantID, idempotency).Scan(&stored, &status, &saved)
		if err == nil {
			if stored != reqHash {
				return nil, 0, fmt.Errorf("%w: idempotency key reused with different content", ErrConflict)
			}
			var out map[string]any
			json.Unmarshal(saved, &out)
			return out, status, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, err
		}
	}
	var docID, jobID string
	err = tx.QueryRow(ctx, `INSERT INTO documents(tenant_id,source_id,external_id,title,media_type,content_sha256,metadata) VALUES($1,NULLIF($2,'')::uuid,NULLIF($3,''),$4,$5,$6,$7) RETURNING id::text`, p.TenantID, in.SourceID, in.ExternalID, in.Title, in.MediaType, contentHash, in.Metadata).Scan(&docID)
	if err != nil {
		return nil, 0, err
	}
	payload, _ := json.Marshal(map[string]any{"document_id": docID, "content": in.Content})
	err = tx.QueryRow(ctx, `INSERT INTO jobs(tenant_id,kind,payload) VALUES($1,'ingest',$2) RETURNING id::text`, p.TenantID, payload).Scan(&jobID)
	if err != nil {
		return nil, 0, err
	}
	out := map[string]any{"document_id": docID, "job_id": jobID, "status": "queued"}
	if idempotency != "" {
		_, err = tx.Exec(ctx, `INSERT INTO idempotency_keys(tenant_id,key,request_hash,status_code,response,expires_at) VALUES($1,$2,$3,202,$4,now()+interval '24 hours')`, p.TenantID, idempotency, reqHash, toJSON(out))
		if err != nil {
			return nil, 0, err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events(tenant_id,actor_id,action,resource_type,resource_id,request_id,after_state) VALUES($1,$2,'document.queued','document',$3,$4,$5)`, p.TenantID, p.ActorID, docID, requestID, toJSON(out))
	if err != nil {
		return nil, 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, 0, err
	}
	return out, 202, nil
}

func (r *Repository) ListCandidates(ctx context.Context, tenant, status string, limit int) ([]Candidate, error) {
	rows, err := r.Pool.Query(ctx, `SELECT c.id::text,c.document_id::text,c.title,c.content,c.status,c.confidence,COALESCE(s.name,'Direct upload'),c.created_at FROM candidates c JOIN documents d ON d.id=c.document_id LEFT JOIN sources s ON s.id=d.source_id WHERE c.tenant_id=$1 AND ($2='' OR c.status=$2) ORDER BY c.created_at DESC LIMIT $3`, tenant, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Candidate{}
	for rows.Next() {
		var c Candidate
		if err = rows.Scan(&c.ID, &c.DocumentID, &c.Title, &c.Content, &c.Status, &c.Confidence, &c.Source, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) Decide(ctx context.Context, p Principal, id, action, content, note, requestID string) (Candidate, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return Candidate{}, err
	}
	defer tx.Rollback(ctx)
	var before Candidate
	err = tx.QueryRow(ctx, `SELECT id::text,document_id::text,title,content,status,confidence,'',created_at FROM candidates WHERE id=$1 AND tenant_id=$2 FOR UPDATE`, id, p.TenantID).Scan(&before.ID, &before.DocumentID, &before.Title, &before.Content, &before.Status, &before.Confidence, &before.Source, &before.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Candidate{}, ErrNotFound
	}
	if err != nil {
		return Candidate{}, err
	}
	if before.Status != "pending" {
		return Candidate{}, fmt.Errorf("%w: candidate already decided", ErrConflict)
	}
	status := map[string]string{"approve": "approved", "edit": "approved", "reject": "rejected"}[action]
	if status == "" {
		return Candidate{}, fmt.Errorf("%w: invalid decision", ErrValidation)
	}
	if action == "edit" && content == "" {
		return Candidate{}, fmt.Errorf("%w: content required for edit", ErrValidation)
	}
	if content == "" {
		content = before.Content
	}
	var after Candidate
	err = tx.QueryRow(ctx, `UPDATE candidates SET content=$1,status=$2,reviewer_id=NULL,reviewer_actor_id=NULLIF($3,''),reviewed_at=now(),review_note=$4 WHERE id=$5 RETURNING id::text,document_id::text,title,content,status,confidence,'',created_at`, content, status, p.ActorID, note, id).Scan(&after.ID, &after.DocumentID, &after.Title, &after.Content, &after.Status, &after.Confidence, &after.Source, &after.CreatedAt)
	if err != nil {
		return Candidate{}, err
	}
	active := status == "approved"
	if _, err = tx.Exec(ctx, `UPDATE chunks SET active=$1 WHERE candidate_id=$2`, active, id); err != nil {
		return Candidate{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE documents SET state=$1,processed_at=now() WHERE id=$2`, status, after.DocumentID); err != nil {
		return Candidate{}, err
	}
	if active {
		_, err = tx.Exec(ctx, `INSERT INTO jobs(tenant_id,kind,payload) VALUES($1,'index',jsonb_build_object('candidate_id',$2::text))`, p.TenantID, id)
		if err != nil {
			return Candidate{}, err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events(tenant_id,actor_id,action,resource_type,resource_id,request_id,before_state,after_state) VALUES($1,$2,$3,'candidate',$4,$5,$6,$7)`, p.TenantID, p.ActorID, "candidate."+action, id, requestID, toJSON(before), toJSON(after))
	if err != nil {
		return Candidate{}, err
	}
	return after, tx.Commit(ctx)
}

func (r *Repository) SearchApprovedDocuments(ctx context.Context, tenant, question string, limit int) ([]map[string]any, error) {
	rows, err := r.Pool.Query(ctx, `
WITH q AS (SELECT websearch_to_tsquery('english', $2) AS query),
ranked AS (
  SELECT c.id::text AS id,c.title,left(c.content,20000) AS content,COALESCE(s.name,'Direct upload') AS source,
         MAX(ts_rank_cd(ch.search_vector,q.query)) AS score,c.updated_at
  FROM candidates c
  JOIN documents d ON d.id=c.document_id
  JOIN chunks ch ON ch.candidate_id=c.id AND ch.active
  LEFT JOIN sources s ON s.id=d.source_id
  CROSS JOIN q
  WHERE c.tenant_id=$1 AND c.status='approved' AND ch.search_vector @@ q.query
  GROUP BY c.id,c.title,c.content,s.name,c.updated_at
), fallback AS (
  SELECT c.id::text AS id,c.title,left(c.content,20000) AS content,COALESCE(s.name,'Direct upload') AS source,
         0::real AS score,c.updated_at
  FROM candidates c
  JOIN documents d ON d.id=c.document_id
  LEFT JOIN sources s ON s.id=d.source_id
  WHERE c.tenant_id=$1 AND c.status='approved'
    AND NOT EXISTS (SELECT 1 FROM ranked r WHERE r.id=c.id::text)
  ORDER BY c.updated_at DESC
  LIMIT $3
)
SELECT id,title,content,source FROM (
  SELECT * FROM ranked UNION ALL SELECT * FROM fallback
) candidates ORDER BY score DESC,updated_at DESC LIMIT $3`, tenant, question, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, title, content, source string
		if err = rows.Scan(&id, &title, &content, &source); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "title": title, "content": content, "source": source})
	}
	return out, rows.Err()
}
func (r *Repository) SaveExchange(ctx context.Context, p Principal, question string, a Answer, latency int) (Answer, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return a, err
	}
	defer tx.Rollback(ctx)
	var conv string
	err = tx.QueryRow(ctx, `INSERT INTO conversations(tenant_id,user_id) VALUES($1,NULL) RETURNING id::text`, p.TenantID).Scan(&conv)
	if err != nil {
		return a, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO messages(tenant_id,conversation_id,role,content) VALUES($1,$2,'user',$3)`, p.TenantID, conv, question); err != nil {
		return a, err
	}
	var msg string
	err = tx.QueryRow(ctx, `INSERT INTO messages(tenant_id,conversation_id,role,content,citations,confidence,latency_ms) VALUES($1,$2,'assistant',$3,$4,$5,$6) RETURNING id::text`, p.TenantID, conv, a.Answer, toJSON(a.Citations), a.Confidence, latency).Scan(&msg)
	if err != nil {
		return a, err
	}
	a.ConversationID = conv
	a.MessageID = msg
	return a, tx.Commit(ctx)
}
func (r *Repository) Feedback(ctx context.Context, p Principal, messageID string, rating int, correction string) (string, error) {
	var id string
	err := r.Pool.QueryRow(ctx, `INSERT INTO feedback(tenant_id,message_id,user_id,rating,correction) SELECT $1,id,NULL,$3,NULLIF($4,'') FROM messages WHERE id=$2 AND tenant_id=$1 RETURNING id::text`, p.TenantID, messageID, rating, correction).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return id, err
}
func (r *Repository) Audit(ctx context.Context, tenant string, limit int) ([]map[string]any, error) {
	rows, err := r.Pool.Query(ctx, `SELECT id,actor_id,action,resource_type,resource_id,request_id,created_at FROM audit_events WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT $2`, tenant, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var actor, action, typ, res, req string
		var at time.Time
		if err = rows.Scan(&id, &actor, &action, &typ, &res, &req, &at); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "actor_id": actor, "action": action, "resource_type": typ, "resource_id": res, "request_id": req, "created_at": at})
	}
	return out, rows.Err()
}
func (r *Repository) ListJobs(ctx context.Context, tenant, status string, limit int) ([]map[string]any, error) {
	rows, err := r.Pool.Query(ctx, `SELECT id::text,kind,status,attempts,max_attempts,run_after,locked_at,last_error,created_at,finished_at FROM jobs WHERE tenant_id=$1 AND ($2='' OR status=$2) ORDER BY created_at DESC LIMIT $3`, tenant, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, kind, state string
		var attempts, max int
		var run, created time.Time
		var locked, finished *time.Time
		var last *string
		if err = rows.Scan(&id, &kind, &state, &attempts, &max, &run, &locked, &last, &created, &finished); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "kind": kind, "status": state, "attempts": attempts, "max_attempts": max, "run_after": run, "locked_at": locked, "last_error": last, "created_at": created, "finished_at": finished})
	}
	return out, rows.Err()
}
func (r *Repository) ReplayJob(ctx context.Context, p Principal, id, requestID string) error {
	tag, err := r.Pool.Exec(ctx, `UPDATE jobs SET status='queued',attempts=0,run_after=now(),last_error=NULL,finished_at=NULL WHERE id=$1 AND tenant_id=$2 AND status IN ('failed','dead')`, id, p.TenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	_, err = r.Pool.Exec(ctx, `INSERT INTO audit_events(tenant_id,actor_id,action,resource_type,resource_id,request_id) VALUES($1,$2,'job.replayed','job',$3,$4)`, p.TenantID, p.ActorID, id, requestID)
	return err
}

func toJSON(v any) []byte { b, _ := json.Marshal(v); return b }
func validateRole(actual, required string) bool {
	rank := map[string]int{"viewer": 1, "reviewer": 2, "admin": 3, "owner": 4}
	return rank[actual] >= rank[required]
}
func wrap(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

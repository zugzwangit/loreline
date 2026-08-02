//go:build integration

package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	postgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestProductionLifecycle(t *testing.T) {
	temp := t.TempDir()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := uint32(listener.Addr().(*net.TCPAddr).Port)
	_ = listener.Close()
	db := postgres.NewDatabase(postgres.DefaultConfig().Version(postgres.V16).Port(port).Database("loreline").Username("postgres").Password("postgres").RuntimePath(filepath.Join(temp, "runtime")).DataPath(filepath.Join(temp, "data")).BinariesPath(filepath.Join(temp, "bin")))
	if err := db.Start(); err != nil {
		t.Fatal(err)
	}
	defer db.Stop()
	url := fmt.Sprintf("postgres://postgres:postgres@localhost:%d/loreline?sslmode=disable", port)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err = Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var migrationVersion int
	if err = pool.QueryRow(ctx, "SELECT max(version) FROM schema_migrations").Scan(&migrationVersion); err != nil || migrationVersion != 2 {
		t.Fatalf("expected migration version 2, got %d (%v)", migrationVersion, err)
	}
	var tenant string
	if err = pool.QueryRow(ctx, "INSERT INTO tenants(slug,name) VALUES('acme','Acme') RETURNING id::text").Scan(&tenant); err != nil {
		t.Fatal(err)
	}
	ai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, Answer{Answer: "Reset access after 15 minutes.", Confidence: .94, Grounded: true, Citations: []Citation{{ID: "one", Title: "SSO guide", Source: "Wiki"}}})
	}))
	defer ai.Close()
	api := NewAPI(pool, Config{AIURL: ai.URL, AllowedOrigins: []string{"http://localhost"}, AllowDevAuth: true, MaxBodyBytes: 2 << 20, RatePerMinute: 100, ShutdownGrace: 1})
	callWithHeaders := func(method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Loreline-Tenant", tenant)
		req.Header.Set("X-Loreline-Role", "owner")
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)
		return w
	}
	call := func(method, path, body string) *httptest.ResponseRecorder {
		return callWithHeaders(method, path, body, nil)
	}
	unsafeSource := call("POST", "/v1/sources", `{"kind":"wiki","name":"Unsafe","config":{"provider":"http_json","url":"http://metadata.internal","credential_env":"DATABASE_URL"}}`)
	if unsafeSource.Code != 422 {
		t.Fatalf("unsafe source config: %d %s", unsafeSource.Code, unsafeSource.Body.String())
	}
	source := call("POST", "/v1/sources", `{"kind":"wiki","name":"Company wiki","config":{}}`)
	if source.Code != 201 {
		t.Fatalf("source: %d %s", source.Code, source.Body.String())
	}
	var s Source
	json.NewDecoder(source.Body).Decode(&s)
	var otherTenant, otherSource string
	if err = pool.QueryRow(ctx, "INSERT INTO tenants(slug,name) VALUES('other','Other') RETURNING id::text").Scan(&otherTenant); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, "INSERT INTO sources(tenant_id,kind,name) VALUES($1,'wiki','Other wiki') RETURNING id::text", otherTenant).Scan(&otherSource); err != nil {
		t.Fatal(err)
	}
	crossTenantIngest := call("POST", "/v1/documents", fmt.Sprintf(`{"source_id":%q,"external_id":"forbidden","title":"Forbidden","media_type":"text/plain","content":"This source belongs to another tenant.","metadata":{}}`, otherSource))
	if crossTenantIngest.Code != 404 {
		t.Fatalf("cross-tenant source was visible: %d %s", crossTenantIngest.Code, crossTenantIngest.Body.String())
	}
	ingest := call("POST", "/v1/documents", fmt.Sprintf(`{"source_id":%q,"external_id":"page-1","title":"SSO reset","media_type":"text/plain","content":"Wait fifteen minutes and retry the identity portal after an SSO lockout.","metadata":{}}`, s.ID))
	if ingest.Code != 202 {
		t.Fatalf("ingest: %d %s", ingest.Code, ingest.Body.String())
	}
	var queued map[string]any
	json.NewDecoder(ingest.Body).Decode(&queued)
	doc := queued["document_id"].(string)
	idempotentBody := fmt.Sprintf(`{"source_id":%q,"external_id":"page-2","title":"Second page","media_type":"text/plain","content":"A stable body for retry testing.","metadata":{}}`, s.ID)
	firstIdempotent := callWithHeaders("POST", "/v1/documents", idempotentBody, map[string]string{"Idempotency-Key": "retry-key"})
	secondIdempotent := callWithHeaders("POST", "/v1/documents", idempotentBody, map[string]string{"Idempotency-Key": "retry-key"})
	if firstIdempotent.Code != 202 || secondIdempotent.Code != 202 || firstIdempotent.Body.String() != secondIdempotent.Body.String() {
		t.Fatalf("idempotent retry mismatch: %d/%d %s/%s", firstIdempotent.Code, secondIdempotent.Code, firstIdempotent.Body.String(), secondIdempotent.Body.String())
	}
	conflictingIdempotent := callWithHeaders("POST", "/v1/documents", strings.Replace(idempotentBody, "Second page", "Different title", 1), map[string]string{"Idempotency-Key": "retry-key"})
	if conflictingIdempotent.Code != 409 {
		t.Fatalf("idempotency conflict: %d %s", conflictingIdempotent.Code, conflictingIdempotent.Body.String())
	}
	var candidate string
	if python := os.Getenv("LORELINE_TEST_PYTHON"); python != "" {
		workerCtx, cancelWorker := context.WithCancel(ctx)
		defer cancelWorker()
		cwd, _ := os.Getwd()
		cmd := exec.CommandContext(workerCtx, python, "-m", "loreline_ai.worker")
		cmd.Dir = filepath.Clean(filepath.Join(cwd, "../../../intelligence"))
		cmd.Env = append(os.Environ(), "LORELINE_DATABASE_URL="+url, "LORELINE_WORKER_POLL_SECONDS=0.05", "LORELINE_ALLOW_DATABASE_ONLY=true")
		if err = cmd.Start(); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(12 * time.Second)
		for time.Now().Before(deadline) {
			err = pool.QueryRow(ctx, "SELECT id::text FROM candidates WHERE document_id=$1", doc).Scan(&candidate)
			if err == nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		cancelWorker()
		_ = cmd.Wait()
		if candidate == "" {
			t.Fatal("worker did not create candidate")
		}
		var onlineWorkers int
		if err = pool.QueryRow(ctx, "SELECT count(*) FROM worker_heartbeats WHERE heartbeat_at > now()-interval '45 seconds'").Scan(&onlineWorkers); err != nil || onlineWorkers < 1 {
			t.Fatalf("worker heartbeat missing: %d (%v)", onlineWorkers, err)
		}
	} else {
		if err = pool.QueryRow(ctx, `INSERT INTO candidates(tenant_id,document_id,title,content,confidence) VALUES($1,$2,'SSO reset','Wait fifteen minutes and retry.',.95) RETURNING id::text`, tenant, doc).Scan(&candidate); err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, `INSERT INTO chunks(tenant_id,document_id,candidate_id,ordinal,content,content_sha256) VALUES($1,$2,$3,0,'Wait fifteen minutes and retry.','hash')`, tenant, doc, candidate); err != nil {
			t.Fatal(err)
		}
	}
	productionKeyResponse := call("POST", "/v1/api-keys", `{"name":"production console","role":"owner"}`)
	if productionKeyResponse.Code != 201 {
		t.Fatalf("production key: %d %s", productionKeyResponse.Code, productionKeyResponse.Body.String())
	}
	var productionKey map[string]any
	json.NewDecoder(productionKeyResponse.Body).Decode(&productionKey)
	bearerCall := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+productionKey["api_key"].(string))
		w := httptest.NewRecorder()
		api.ServeHTTP(w, req)
		return w
	}
	decision := bearerCall("POST", "/v1/candidates/"+candidate+"/decision", `{"action":"approve","note":"verified"}`)
	if decision.Code != 200 {
		t.Fatalf("decision: %d %s", decision.Code, decision.Body.String())
	}
	var reviewerActor string
	var reviewerID *string
	if err = pool.QueryRow(ctx, "SELECT reviewer_actor_id,reviewer_id::text FROM candidates WHERE id=$1", candidate).Scan(&reviewerActor, &reviewerID); err != nil {
		t.Fatal(err)
	}
	if reviewerActor != productionKey["id"].(string) || reviewerID != nil {
		t.Fatalf("unexpected reviewer attribution: actor=%q user=%v", reviewerActor, reviewerID)
	}
	duplicateDecision := bearerCall("POST", "/v1/candidates/"+candidate+"/decision", `{"action":"approve","note":"duplicate"}`)
	if duplicateDecision.Code != 409 {
		t.Fatalf("duplicate decision: %d %s", duplicateDecision.Code, duplicateDecision.Body.String())
	}
	answer := call("POST", "/v1/assistant/ask", `{"question":"How do I reset SSO?"}`)
	if answer.Code != 200 {
		t.Fatalf("answer: %d %s", answer.Code, answer.Body.String())
	}
	var a Answer
	json.NewDecoder(answer.Body).Decode(&a)
	if !a.Grounded || a.MessageID == "" {
		t.Fatalf("answer not persisted: %+v", a)
	}
	fb := call("POST", "/v1/feedback", fmt.Sprintf(`{"message_id":%q,"rating":1}`, a.MessageID))
	if fb.Code != 201 {
		t.Fatalf("feedback: %d %s", fb.Code, fb.Body.String())
	}
	var otherConversation, otherMessage string
	if err = pool.QueryRow(ctx, "INSERT INTO conversations(tenant_id) VALUES($1) RETURNING id::text", otherTenant).Scan(&otherConversation); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, "INSERT INTO messages(tenant_id,conversation_id,role,content) VALUES($1,$2,'assistant','private') RETURNING id::text", otherTenant, otherConversation).Scan(&otherMessage); err != nil {
		t.Fatal(err)
	}
	crossTenantFeedback := call("POST", "/v1/feedback", fmt.Sprintf(`{"message_id":%q,"rating":-1}`, otherMessage))
	if crossTenantFeedback.Code != 404 {
		t.Fatalf("cross-tenant message was visible: %d %s", crossTenantFeedback.Code, crossTenantFeedback.Body.String())
	}
	audit := call("GET", "/v1/audit", "")
	if audit.Code != 200 {
		t.Fatalf("audit: %d %s", audit.Code, audit.Body.String())
	}
	keyResponse := call("POST", "/v1/api-keys", `{"name":"automation","role":"viewer"}`)
	if keyResponse.Code != 201 {
		t.Fatalf("api key: %d %s", keyResponse.Code, keyResponse.Body.String())
	}
	var key map[string]any
	json.NewDecoder(keyResponse.Body).Decode(&key)
	if !strings.HasPrefix(key["api_key"].(string), "ll_live_") {
		t.Fatal("key secret missing")
	}
	revoked := call("DELETE", "/v1/api-keys/"+key["id"].(string), "")
	if revoked.Code != 204 {
		t.Fatalf("revoke: %d %s", revoked.Code, revoked.Body.String())
	}
	var failedJob string
	if err = pool.QueryRow(ctx, `INSERT INTO jobs(tenant_id,kind,payload,status,attempts,last_error) VALUES($1,'archive','{}','dead',5,'test') RETURNING id::text`, tenant).Scan(&failedJob); err != nil {
		t.Fatal(err)
	}
	replayed := call("POST", "/v1/jobs/"+failedJob+"/replay", "")
	if replayed.Code != 202 {
		t.Fatalf("replay: %d %s", replayed.Code, replayed.Body.String())
	}
	firstSync := call("POST", "/v1/sources/"+s.ID+"/sync", "")
	duplicateSync := call("POST", "/v1/sources/"+s.ID+"/sync", "")
	if firstSync.Code != 202 || duplicateSync.Code != 409 {
		t.Fatalf("sync de-duplication: %d/%d %s/%s", firstSync.Code, duplicateSync.Code, firstSync.Body.String(), duplicateSync.Body.String())
	}
}

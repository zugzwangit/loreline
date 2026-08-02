package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type API struct {
	repo       *Repository
	pool       *pgxpool.Pool
	cfg        Config
	client     *http.Client
	middleware *Middleware
}

func NewAPI(pool *pgxpool.Pool, cfg Config) http.Handler {
	repo := NewRepository(pool)
	a := &API{repo: repo, pool: pool, cfg: cfg, client: &http.Client{Timeout: 30 * time.Second}}
	a.middleware = NewMiddleware(repo, cfg)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", a.live)
	mux.HandleFunc("GET /readyz", a.ready)
	mux.HandleFunc("GET /metrics", a.metrics)
	mux.HandleFunc("GET /v1/dashboard", a.dashboard)
	mux.HandleFunc("GET /v1/sources", a.sources)
	mux.HandleFunc("POST /v1/sources", a.createSource)
	mux.HandleFunc("POST /v1/sources/{id}/sync", a.syncSource)
	mux.HandleFunc("POST /v1/documents", a.ingest)
	mux.HandleFunc("GET /v1/candidates", a.candidates)
	mux.HandleFunc("POST /v1/candidates/{id}/decision", a.decide)
	mux.HandleFunc("POST /v1/assistant/ask", a.ask)
	mux.HandleFunc("POST /v1/assistant/stream", a.stream)
	mux.HandleFunc("POST /v1/feedback", a.feedback)
	mux.HandleFunc("GET /v1/audit", a.audit)
	mux.HandleFunc("GET /v1/api-keys", a.apiKeys)
	mux.HandleFunc("POST /v1/api-keys", a.createAPIKey)
	mux.HandleFunc("DELETE /v1/api-keys/{id}", a.revokeAPIKey)
	mux.HandleFunc("GET /v1/jobs", a.jobs)
	mux.HandleFunc("POST /v1/jobs/{id}/replay", a.replayJob)
	return a.middleware.Wrap(mux)
}
func (a *API) live(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"status": "ok", "service": "loreline-gateway", "version": "1.1.0"})
}
func (a *API) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.pool.Ping(ctx); err != nil {
		writeProblem(w, 503, "database_unavailable", "Database readiness check failed", requestID(r))
		return
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, a.cfg.AIURL+"/readyz", nil)
	resp, err := a.client.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		writeProblem(w, 503, "intelligence_unavailable", "Intelligence readiness check failed", requestID(r))
		return
	}
	resp.Body.Close()
	writeJSON(w, 200, map[string]string{"status": "ready"})
}
func (a *API) metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	io.WriteString(w, a.middleware.Metrics())
}
func (a *API) dashboard(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	out, err := a.repo.Dashboard(r.Context(), p.TenantID)
	respond(w, r, out, err)
}
func (a *API) sources(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	out, err := a.repo.ListSources(r.Context(), p.TenantID)
	respond(w, r, map[string]any{"items": out}, err)
}
func (a *API) createSource(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	if !authorize(w, r, p, "admin") {
		return
	}
	var in struct {
		Kind   string          `json:"kind"`
		Name   string          `json:"name"`
		Config json.RawMessage `json:"config"`
	}
	if err := decode(w, r, a.cfg.MaxBodyBytes, &in); err != nil {
		return
	}
	if !oneOf(in.Kind, "ticketing", "wiki", "drive", "webhook", "upload", "api") || strings.TrimSpace(in.Name) == "" || len(in.Name) > 160 {
		writeProblem(w, 422, "validation_error", "kind and name are required", requestID(r))
		return
	}
	if len(in.Config) == 0 {
		in.Config = json.RawMessage(`{}`)
	}
	lowerConfig := strings.ToLower(string(in.Config))
	for _, blocked := range []string{`"password"`, `"secret"`, `"token"`, `"api_key"`} {
		if strings.Contains(lowerConfig, blocked) {
			writeProblem(w, 422, "secret_in_config", "Store credentials in the secret manager and reference them by credential_env", requestID(r))
			return
		}
	}
	if err := validateSourceConfig(in.Config); err != nil {
		writeProblem(w, 422, "invalid_source_config", err.Error(), requestID(r))
		return
	}
	out, err := a.repo.CreateSource(r.Context(), p, in.Kind, in.Name, in.Config, requestID(r))
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	writeJSON(w, 201, out)
}
func (a *API) syncSource(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	if !authorize(w, r, p, "admin") {
		return
	}
	job, err := a.repo.EnqueueSync(r.Context(), p, r.PathValue("id"), requestID(r))
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	writeJSON(w, 202, map[string]string{"job_id": job, "status": "queued"})
}
func (a *API) ingest(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	if !authorize(w, r, p, "admin") {
		return
	}
	var in IngestInput
	if err := decode(w, r, a.cfg.MaxBodyBytes, &in); err != nil {
		return
	}
	if strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.Content) == "" {
		writeProblem(w, 422, "validation_error", "title and content are required", requestID(r))
		return
	}
	if in.MediaType == "" {
		in.MediaType = "text/plain"
	}
	if len(in.Metadata) == 0 {
		in.Metadata = json.RawMessage(`{}`)
	}
	out, status, err := a.repo.Ingest(r.Context(), p, in, r.Header.Get("Idempotency-Key"), requestID(r))
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	writeJSON(w, status, out)
}
func (a *API) candidates(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	limit := bounded(r.URL.Query().Get("limit"), 50, 1, 200)
	out, err := a.repo.ListCandidates(r.Context(), p.TenantID, r.URL.Query().Get("status"), limit)
	respond(w, r, map[string]any{"items": out}, err)
}
func (a *API) decide(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	if !authorize(w, r, p, "reviewer") {
		return
	}
	var in struct {
		Action  string `json:"action"`
		Content string `json:"content"`
		Note    string `json:"note"`
	}
	if err := decode(w, r, a.cfg.MaxBodyBytes, &in); err != nil {
		return
	}
	out, err := a.repo.Decide(r.Context(), p, r.PathValue("id"), in.Action, in.Content, in.Note, requestID(r))
	respond(w, r, out, err)
}
func (a *API) ask(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	var in struct {
		Question string `json:"question"`
	}
	if err := decode(w, r, a.cfg.MaxBodyBytes, &in); err != nil {
		return
	}
	if strings.TrimSpace(in.Question) == "" {
		writeProblem(w, 422, "validation_error", "question is required", requestID(r))
		return
	}
	docs, err := a.repo.SearchApprovedDocuments(r.Context(), p.TenantID, in.Question, 25)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	payload, _ := json.Marshal(map[string]any{"question": in.Question, "documents": docs})
	start := time.Now()
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, a.cfg.AIURL+"/v1/retrieve", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		writeProblem(w, 502, "intelligence_unavailable", "Answer service is unavailable", requestID(r))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		proxyProblem(w, resp)
		return
	}
	var answer Answer
	if err = json.NewDecoder(io.LimitReader(resp.Body, a.cfg.MaxBodyBytes)).Decode(&answer); err != nil {
		respond(w, r, nil, err)
		return
	}
	answer, err = a.repo.SaveExchange(r.Context(), p, in.Question, answer, int(time.Since(start).Milliseconds()))
	respond(w, r, answer, err)
}
func (a *API) stream(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	var in struct {
		Question string `json:"question"`
	}
	if err := decode(w, r, a.cfg.MaxBodyBytes, &in); err != nil {
		return
	}
	if strings.TrimSpace(in.Question) == "" {
		writeProblem(w, 422, "validation_error", "question is required", requestID(r))
		return
	}
	docs, err := a.repo.SearchApprovedDocuments(r.Context(), p.TenantID, in.Question, 25)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	payload, _ := json.Marshal(map[string]any{"question": in.Question, "documents": docs})
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, a.cfg.AIURL+"/v1/stream", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		writeProblem(w, 502, "intelligence_unavailable", "Answer stream is unavailable", requestID(r))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		proxyProblem(w, resp)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(200)
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 2048)
	for {
		n, e := resp.Body.Read(buf)
		if n > 0 {
			if _, err = w.Write(buf[:n]); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if e != nil {
			return
		}
	}
}
func (a *API) feedback(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	var in struct {
		MessageID  string `json:"message_id"`
		Rating     int    `json:"rating"`
		Correction string `json:"correction"`
	}
	if err := decode(w, r, a.cfg.MaxBodyBytes, &in); err != nil {
		return
	}
	if in.MessageID == "" || (in.Rating != -1 && in.Rating != 1) {
		writeProblem(w, 422, "validation_error", "message_id and rating (-1 or 1) are required", requestID(r))
		return
	}
	id, err := a.repo.Feedback(r.Context(), p, in.MessageID, in.Rating, in.Correction)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	writeJSON(w, 201, map[string]string{"id": id, "status": "open"})
}
func (a *API) audit(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	if !authorize(w, r, p, "admin") {
		return
	}
	out, err := a.repo.Audit(r.Context(), p.TenantID, bounded(r.URL.Query().Get("limit"), 100, 1, 500))
	respond(w, r, map[string]any{"items": out}, err)
}
func (a *API) apiKeys(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	if !authorize(w, r, p, "owner") {
		return
	}
	out, err := a.repo.ListAPIKeys(r.Context(), p.TenantID)
	respond(w, r, map[string]any{"items": out}, err)
}
func (a *API) createAPIKey(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	if !authorize(w, r, p, "owner") {
		return
	}
	var in struct {
		Name      string     `json:"name"`
		Role      string     `json:"role"`
		ExpiresAt *time.Time `json:"expires_at"`
	}
	if err := decode(w, r, a.cfg.MaxBodyBytes, &in); err != nil {
		return
	}
	if strings.TrimSpace(in.Name) == "" || !oneOf(in.Role, "viewer", "reviewer", "admin", "owner") {
		writeProblem(w, 422, "validation_error", "name and a valid role are required", requestID(r))
		return
	}
	out, err := a.repo.CreateAPIKey(r.Context(), p, in.Name, in.Role, in.ExpiresAt, requestID(r))
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	writeJSON(w, 201, out)
}
func (a *API) revokeAPIKey(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	if !authorize(w, r, p, "owner") {
		return
	}
	if r.PathValue("id") == p.ActorID {
		writeProblem(w, 409, "self_revocation", "Create and test a replacement key before revoking the current key", requestID(r))
		return
	}
	err := a.repo.RevokeAPIKey(r.Context(), p, r.PathValue("id"), requestID(r))
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	w.WriteHeader(204)
}

func (a *API) jobs(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	if !authorize(w, r, p, "admin") {
		return
	}
	out, err := a.repo.ListJobs(r.Context(), p.TenantID, r.URL.Query().Get("status"), bounded(r.URL.Query().Get("limit"), 100, 1, 500))
	respond(w, r, map[string]any{"items": out}, err)
}
func (a *API) replayJob(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFrom(r.Context())
	if !authorize(w, r, p, "admin") {
		return
	}
	err := a.repo.ReplayJob(r.Context(), p, r.PathValue("id"), requestID(r))
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	writeJSON(w, 202, map[string]string{"id": r.PathValue("id"), "status": "queued"})
}

func decode(w http.ResponseWriter, r *http.Request, max int64, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, max)
	defer r.Body.Close()
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		writeProblem(w, 400, "invalid_json", err.Error(), requestID(r))
		return err
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		writeProblem(w, 400, "invalid_json", "Only one JSON value is allowed", requestID(r))
		return errors.New("trailing json")
	}
	return nil
}
func respond(w http.ResponseWriter, r *http.Request, v any, err error) {
	if err == nil {
		writeJSON(w, 200, v)
		return
	}
	if errors.Is(err, ErrNotFound) {
		writeProblem(w, 404, "not_found", "Resource not found", requestID(r))
		return
	}
	if errors.Is(err, ErrConflict) {
		writeProblem(w, 409, "conflict", err.Error(), requestID(r))
		return
	}
	if errors.Is(err, ErrValidation) {
		writeProblem(w, 422, "validation_error", err.Error(), requestID(r))
		return
	}
	slog.Error("request failed", "request_id", requestID(r), "error", err)
	writeProblem(w, 500, "internal_error", "The operation could not be completed", requestID(r))
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeProblem(w http.ResponseWriter, status int, code, detail, request string) {
	writeJSON(w, status, map[string]any{"type": "https://docs.loreline.dev/problems/" + code, "title": http.StatusText(status), "status": status, "code": code, "detail": detail, "request_id": request})
}
func proxyProblem(w http.ResponseWriter, resp *http.Response) {
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(resp.Body, 1<<20))
}
func authorize(w http.ResponseWriter, r *http.Request, p Principal, role string) bool {
	if validateRole(p.Role, role) {
		return true
	}
	writeProblem(w, 403, "forbidden", "This operation requires the "+role+" role", requestID(r))
	return false
}
func bounded(raw string, fallback, minV, maxV int) int {
	n, e := strconv.Atoi(raw)
	if e != nil {
		return fallback
	}
	if n < minV {
		return minV
	}
	if n > maxV {
		return maxV
	}
	return n
}
func oneOf(v string, values ...string) bool {
	for _, x := range values {
		if v == x {
			return true
		}
	}
	return false
}

func validateSourceConfig(raw json.RawMessage) error {
	var cfg struct {
		Provider      string `json:"provider"`
		URL           string `json:"url"`
		BaseURL       string `json:"base_url"`
		CredentialEnv string `json:"credential_env"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return errors.New("config must be a JSON object")
	}
	if cfg.Provider == "" {
		return nil
	}
	if !oneOf(cfg.Provider, "confluence", "zendesk", "http_json") {
		return errors.New("provider must be confluence, zendesk, or http_json")
	}
	endpoint := cfg.BaseURL
	if cfg.Provider == "http_json" {
		endpoint = cfg.URL
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		return errors.New("connector endpoint must be an HTTPS URL without embedded credentials")
	}
	if !strings.HasPrefix(cfg.CredentialEnv, "LORELINE_CONNECTOR_") || len(cfg.CredentialEnv) > 128 {
		return errors.New("credential_env must use the LORELINE_CONNECTOR_ prefix")
	}
	return nil
}

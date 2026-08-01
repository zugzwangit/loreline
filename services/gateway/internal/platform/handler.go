package platform

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Handler struct {
	store  *Store
	aiURL  string
	client *http.Client
}

func NewHandler(store *Store, aiURL string) http.Handler {
	h := &Handler{store: store, aiURL: strings.TrimRight(aiURL, "/"), client: &http.Client{Timeout: 5 * time.Second}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /v1/dashboard", h.dashboard)
	mux.HandleFunc("GET /v1/knowledge", h.list)
	mux.HandleFunc("GET /v1/knowledge/{id}", h.get)
	mux.HandleFunc("POST /v1/knowledge/{id}/decision", h.decide)
	mux.HandleFunc("POST /v1/assistant/ask", h.ask)
	return withCORS(withRequestID(mux))
}
func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"status": "ok", "service": "loreline-gateway", "time": time.Now().UTC()})
}
func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"knowledge": h.store.Stats(), "questions_resolved": 1284, "answer_quality": 96.8, "freshness": 94})
}
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"items": h.store.List(r.URL.Query().Get("status"))})
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	v, ok := h.store.Get(r.PathValue("id"))
	if !ok {
		writeError(w, 404, "not_found", "knowledge not found")
		return
	}
	writeJSON(w, 200, v)
}
func (h *Handler) decide(w http.ResponseWriter, r *http.Request) {
	var d Decision
	if err := decode(r, &d); err != nil {
		writeError(w, 400, "invalid_request", err.Error())
		return
	}
	v, err := h.store.Decide(r.PathValue("id"), d)
	if err != nil {
		writeError(w, 400, "invalid_decision", err.Error())
		return
	}
	writeJSON(w, 200, v)
}
func (h *Handler) ask(w http.ResponseWriter, r *http.Request) {
	var q AskRequest
	if err := decode(r, &q); err != nil || strings.TrimSpace(q.Question) == "" {
		writeError(w, 400, "invalid_request", "question is required")
		return
	}
	payload, _ := json.Marshal(map[string]any{"question": q.Question, "documents": h.store.List("approved")})
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, h.aiURL+"/v1/retrieve", bytes.NewReader(payload))
	if err != nil {
		writeError(w, 500, "internal_error", err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		writeError(w, 502, "intelligence_unavailable", "answer service is unavailable")
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
func decode(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := os.Getenv("LORELINE_ALLOWED_ORIGIN")
		if origin == "" {
			origin = "http://localhost:3000"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,X-Request-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = time.Now().UTC().Format("20060102T150405.000000000")
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}

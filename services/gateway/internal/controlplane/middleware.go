package controlplane

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Middleware struct {
	repo             *Repository
	cfg              Config
	limiter          *rateLimiter
	requests, errors atomic.Uint64
}

func NewMiddleware(repo *Repository, cfg Config) *Middleware {
	return &Middleware{repo: repo, cfg: cfg, limiter: newRateLimiter(cfg.RatePerMinute)}
}
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return m.recover(m.observe(m.cors(m.authenticate(next))))
}
func (m *Middleware) Metrics() string {
	return "# HELP loreline_http_requests_total Total HTTP requests.\n# TYPE loreline_http_requests_total counter\nloreline_http_requests_total " + itoa(m.requests.Load()) + "\n# HELP loreline_http_errors_total Total HTTP 5xx responses.\n# TYPE loreline_http_errors_total counter\nloreline_http_errors_total " + itoa(m.errors.Load()) + "\n"
}
func (m *Middleware) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/v1/") {
			next.ServeHTTP(w, r)
			return
		}
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		var p Principal
		var err error
		if raw != "" {
			p, err = m.repo.ResolveAPIKey(r.Context(), raw)
		} else if m.cfg.AllowDevAuth {
			p = Principal{TenantID: r.Header.Get("X-Loreline-Tenant"), ActorID: r.Header.Get("X-Loreline-Actor"), Role: r.Header.Get("X-Loreline-Role")}
			if p.Role == "" {
				p.Role = "owner"
			}
			if p.TenantID == "" {
				err = ErrNotFound
			}
		} else {
			err = ErrNotFound
		}
		if err != nil {
			writeProblem(w, 401, "unauthorized", "A valid bearer API key is required", requestID(r))
			return
		}
		key := p.TenantID + ":" + p.ActorID
		if !m.limiter.Allow(key) {
			w.Header().Set("Retry-After", "60")
			writeProblem(w, 429, "rate_limited", "Request rate exceeded", requestID(r))
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey, p)))
	})
}
func (m *Middleware) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := origin == ""
		for _, x := range m.cfg.AllowedOrigins {
			if x == origin {
				allowed = true
				break
			}
		}
		if origin != "" && !allowed {
			writeProblem(w, 403, "origin_forbidden", "Origin is not allowed", requestID(r))
			return
		}
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,Idempotency-Key,X-Request-ID,X-Loreline-Tenant,X-Loreline-Actor,X-Loreline-Role")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (m *Middleware) observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = randomID()
		}
		r.Header.Set("X-Request-ID", id)
		w.Header().Set("X-Request-ID", id)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		rec := &statusWriter{ResponseWriter: w, status: 200}
		m.requests.Add(1)
		next.ServeHTTP(rec, r)
		if rec.status >= 500 {
			m.errors.Add(1)
		}
		slog.Info("request", "request_id", id, "method", r.Method, "path", r.URL.Path, "status", rec.status, "latency_ms", time.Since(start).Milliseconds())
	})
}
func (m *Middleware) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				slog.Error("panic", "request_id", requestID(r), "error", v)
				writeProblem(w, 500, "internal_error", "An internal error occurred", requestID(r))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(v int) { s.status = v; s.ResponseWriter.WriteHeader(v) }
func (s *statusWriter) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
func requestID(r *http.Request) string { return r.Header.Get("X-Request-ID") }
func randomID() string                 { b := make([]byte, 12); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func itoa(v uint64) string {
	const digits = "0123456789"
	if v == 0 {
		return "0"
	}
	b := make([]byte, 0, 20)
	for v > 0 {
		b = append(b, digits[v%10])
		v /= 10
	}
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

type bucket struct {
	tokens float64
	at     time.Time
}
type rateLimiter struct {
	mu    sync.Mutex
	rate  float64
	burst float64
	items map[string]bucket
}

func newRateLimiter(perMinute int) *rateLimiter {
	return &rateLimiter{rate: float64(perMinute) / 60, burst: float64(perMinute), items: map[string]bucket{}}
}
func (l *rateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	b, ok := l.items[key]
	if !ok {
		b = bucket{tokens: l.burst, at: now}
	}
	b.tokens = min(l.burst, b.tokens+now.Sub(b.at).Seconds()*l.rate)
	b.at = now
	if b.tokens < 1 {
		l.items[key] = b
		return false
	}
	b.tokens--
	l.items[key] = b
	return true
}

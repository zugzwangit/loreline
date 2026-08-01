package platform

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKnowledgeLifecycle(t *testing.T) {
	h := NewHandler(NewStore(), "http://127.0.0.1:1")
	r := httptest.NewRequest("GET", "/v1/knowledge?status=review", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("list status=%d", w.Code)
	}
	var body struct {
		Items []Knowledge `json:"items"`
	}
	json.NewDecoder(w.Body).Decode(&body)
	if len(body.Items) != 2 {
		t.Fatalf("want 2 review items, got %d", len(body.Items))
	}
	decision := bytes.NewBufferString(`{"action":"approve"}`)
	r = httptest.NewRequest("POST", "/v1/knowledge/KB-2046/decision", decision)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("decision status=%d body=%s", w.Code, w.Body.String())
	}
	var item Knowledge
	json.NewDecoder(w.Body).Decode(&item)
	if item.Status != "approved" {
		t.Fatalf("want approved, got %s", item.Status)
	}
}
func TestAskValidation(t *testing.T) {
	h := NewHandler(NewStore(), "http://127.0.0.1:1")
	r := httptest.NewRequest("POST", "/v1/assistant/ask", bytes.NewBufferString(`{"question":""}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}
func TestAssistantIntegration(t *testing.T) {
	ai := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, Answer{Answer: "Grounded answer", Confidence: .92, Citations: []Citation{{ID: "KB-1", Title: "Guide", Source: "Wiki"}}})
	}))
	defer ai.Close()
	h := NewHandler(NewStore(), ai.URL)
	r := httptest.NewRequest("POST", "/v1/assistant/ask", bytes.NewBufferString(`{"question":"How do I sign in?"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("Grounded answer")) {
		t.Fatalf("unexpected body %s", w.Body.String())
	}
}

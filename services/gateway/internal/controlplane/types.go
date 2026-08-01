package controlplane

import (
	"context"
	"encoding/json"
	"net"
	"time"
)

type Principal struct{ TenantID, ActorID, Role string }
type contextKey string

const principalKey contextKey = "principal"

func principalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey).(Principal)
	return p, ok
}

type Source struct {
	ID           string          `json:"id"`
	Kind         string          `json:"kind"`
	Name         string          `json:"name"`
	Status       string          `json:"status"`
	Config       json.RawMessage `json:"config"`
	LastSyncedAt *time.Time      `json:"last_synced_at"`
	LastError    *string         `json:"last_error"`
	CreatedAt    time.Time       `json:"created_at"`
}
type Candidate struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Status     string    `json:"status"`
	Confidence float32   `json:"confidence"`
	Source     string    `json:"source"`
	CreatedAt  time.Time `json:"created_at"`
}
type Citation struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Source string `json:"source"`
}
type Answer struct {
	Answer         string     `json:"answer"`
	Confidence     float64    `json:"confidence"`
	Grounded       bool       `json:"grounded"`
	Citations      []Citation `json:"citations"`
	ConversationID string     `json:"conversation_id,omitempty"`
	MessageID      string     `json:"message_id,omitempty"`
}
type AuditInput struct {
	TenantID, ActorID, Action, ResourceType, ResourceID, RequestID, UserAgent string
	IP                                                                        net.IP
	Before, After                                                             any
}

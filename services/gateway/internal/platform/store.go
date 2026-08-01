package platform

import (
	"errors"
	"sort"
	"sync"
	"time"
)

type Store struct {
	mu    sync.RWMutex
	items map[string]Knowledge
}

func NewStore() *Store {
	now := time.Now().UTC()
	seed := []Knowledge{
		{ID: "KB-2048", Title: "Restoring access after an SSO lockout", Content: "Wait 15 minutes for automatic reset. If access remains blocked, open an Identity and Access request and include the last successful login time.", Source: "Resolved ticket", Owner: "IT Operations", Status: "approved", Confidence: 98, UpdatedAt: now.Add(-8 * time.Minute)},
		{ID: "KB-2047", Title: "2026 parental leave policy", Content: "Eligible employees receive sixteen weeks of paid parental leave and may begin leave up to two weeks before the expected arrival date.", Source: "Policy document", Owner: "People Ops", Status: "approved", Confidence: 96, UpdatedAt: now.Add(-22 * time.Minute)},
		{ID: "KB-2046", Title: "Corporate card: international travel", Content: "Notify Finance before international business travel and confirm international transactions are enabled.", Source: "Wiki page", Owner: "Finance", Status: "review", Confidence: 87, UpdatedAt: now.Add(-34 * time.Minute)},
		{ID: "KB-2045", Title: "Requesting production database access", Content: "Production database access requires manager approval and an active security training certificate.", Source: "Resolved ticket", Owner: "Engineering", Status: "review", Confidence: 91, UpdatedAt: now.Add(-time.Hour)},
	}
	s := &Store{items: map[string]Knowledge{}}
	for _, item := range seed {
		s.items[item.ID] = item
	}
	return s
}

func (s *Store) List(status string) []Knowledge {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Knowledge, 0, len(s.items))
	for _, v := range s.items {
		if status == "" || v.Status == status {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out
}
func (s *Store) Get(id string) (Knowledge, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[id]
	return v, ok
}
func (s *Store) Decide(id string, d Decision) (Knowledge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return Knowledge{}, errors.New("knowledge candidate not found")
	}
	switch d.Action {
	case "approve":
		item.Status = "approved"
	case "reject":
		item.Status = "rejected"
	case "edit":
		if d.Content == "" {
			return Knowledge{}, errors.New("content is required for edit")
		}
		item.Content = d.Content
		item.Status = "approved"
	default:
		return Knowledge{}, errors.New("action must be approve, edit, or reject")
	}
	item.UpdatedAt = time.Now().UTC()
	s.items[id] = item
	return item, nil
}
func (s *Store) Stats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := map[string]int{"total": len(s.items), "approved": 0, "review": 0, "rejected": 0}
	for _, v := range s.items {
		out[v.Status]++
	}
	return out
}

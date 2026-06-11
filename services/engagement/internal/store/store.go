// Package store provides the in-memory engagement repository.
// In production this would be backed by PostgreSQL via sqlc/pgx.
package store

import (
	"sync"

	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/engagement/internal/domain"
)

// EngagementStore is a thread-safe in-memory store for engagements.
type EngagementStore struct {
	mu   sync.RWMutex
	data map[uuid.UUID]*domain.Engagement
}

// NewEngagementStore creates an empty store.
func NewEngagementStore() *EngagementStore {
	return &EngagementStore{
		data: make(map[uuid.UUID]*domain.Engagement),
	}
}

// Create saves a new engagement.
func (s *EngagementStore) Create(e *domain.Engagement) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// defensive copy
	copy := *e
	s.data[e.ID] = &copy
	return nil
}

// Get retrieves an engagement by ID scoped to an org.
func (s *EngagementStore) Get(orgID, id uuid.UUID) (*domain.Engagement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[id]
	if !ok || e.OrgID != orgID {
		return nil, domain.ErrEngagementNotFound
	}
	copy := *e
	return &copy, nil
}

// List returns all engagements for an org.
func (s *EngagementStore) List(orgID uuid.UUID) []*domain.Engagement {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domain.Engagement
	for _, e := range s.data {
		if e.OrgID == orgID {
			copy := *e
			out = append(out, &copy)
		}
	}
	return out
}

// Update persists changes to an existing engagement.
func (s *EngagementStore) Update(e *domain.Engagement) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[e.ID]; !ok {
		return domain.ErrEngagementNotFound
	}
	copy := *e
	s.data[e.ID] = &copy
	return nil
}


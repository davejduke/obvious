// Package store provides the in-memory approval workflow repository.
// In production this would be backed by PostgreSQL via sqlc/pgx.
package store

import (
	"sync"

	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/approval/internal/domain"
)

// Store is a thread-safe in-memory store for approval workflows.
type Store struct {
	mu   sync.RWMutex
	data map[uuid.UUID]*domain.Workflow
}

// New creates an empty store.
func New() *Store {
	return &Store{
		data: make(map[uuid.UUID]*domain.Workflow),
	}
}

// Create saves a new workflow.
func (s *Store) Create(w *domain.Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := copyWorkflow(w)
	s.data[w.ID] = cp
	return nil
}

// Get retrieves a workflow by ID, scoped to an org.
func (s *Store) Get(orgID, id uuid.UUID) (*domain.Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.data[id]
	if !ok || w.OrgID != orgID {
		return nil, domain.ErrWorkflowNotFound
	}
	cp := copyWorkflow(w)
	return cp, nil
}

// List returns all workflows for an org, optionally filtered by resource.
func (s *Store) List(orgID uuid.UUID, resourceType string, resourceID *uuid.UUID) []*domain.Workflow {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domain.Workflow
	for _, w := range s.data {
		if w.OrgID != orgID {
			continue
		}
		if resourceType != "" && w.ResourceType != resourceType {
			continue
		}
		if resourceID != nil && w.ResourceID != *resourceID {
			continue
		}
		cp := copyWorkflow(w)
		out = append(out, cp)
	}
	return out
}

// Update persists changes to an existing workflow.
func (s *Store) Update(w *domain.Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[w.ID]; !ok {
		return domain.ErrWorkflowNotFound
	}
	cp := copyWorkflow(w)
	s.data[w.ID] = cp
	return nil
}

// copyWorkflow performs a shallow copy of a Workflow, deep-copying the
// History slice so mutations on the returned value do not affect stored state.
func copyWorkflow(w *domain.Workflow) *domain.Workflow {
	cp := *w
	if w.History != nil {
		hist := make([]domain.HistoryEntry, len(w.History))
		copy(hist, w.History)
		cp.History = hist
	}
	return &cp
}

// Package store provides thread-safe in-memory repositories for the planning service.
// In production these would be backed by PostgreSQL.
package store

import (
	"sync"

	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/planning/internal/domain"
)

// ---------------------------------------------------------------------------
// StrategicPlanStore
// ---------------------------------------------------------------------------

// StrategicPlanStore is a thread-safe in-memory store for strategic plans.
type StrategicPlanStore struct {
	mu   sync.RWMutex
	data map[uuid.UUID]*domain.StrategicPlan
}

// NewStrategicPlanStore creates an empty store.
func NewStrategicPlanStore() *StrategicPlanStore {
	return &StrategicPlanStore{data: make(map[uuid.UUID]*domain.StrategicPlan)}
}

// Create persists a new strategic plan.
func (s *StrategicPlanStore) Create(p *domain.StrategicPlan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *p
	s.data[p.ID] = &cp
	return nil
}

// Get retrieves a strategic plan by ID scoped to an org.
func (s *StrategicPlanStore) Get(orgID, id uuid.UUID) (*domain.StrategicPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.data[id]
	if !ok || p.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

// List returns all strategic plans for an org.
func (s *StrategicPlanStore) List(orgID uuid.UUID) []*domain.StrategicPlan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domain.StrategicPlan
	for _, p := range s.data {
		if p.OrgID == orgID {
			cp := *p
			out = append(out, &cp)
		}
	}
	return out
}

// Update persists changes to an existing strategic plan.
func (s *StrategicPlanStore) Update(p *domain.StrategicPlan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[p.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *p
	s.data[p.ID] = &cp
	return nil
}

// ---------------------------------------------------------------------------
// AnnualPlanStore
// ---------------------------------------------------------------------------

// AnnualPlanStore is a thread-safe in-memory store for annual plans.
type AnnualPlanStore struct {
	mu   sync.RWMutex
	data map[uuid.UUID]*domain.AnnualPlan
}

// NewAnnualPlanStore creates an empty store.
func NewAnnualPlanStore() *AnnualPlanStore {
	return &AnnualPlanStore{data: make(map[uuid.UUID]*domain.AnnualPlan)}
}

// Create persists a new annual plan.
func (s *AnnualPlanStore) Create(p *domain.AnnualPlan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Enforce uniqueness: one active/draft plan per year per org.
	for _, existing := range s.data {
		if existing.OrgID == p.OrgID && existing.Year == p.Year &&
			existing.Status != domain.PlanStatusArchived {
			return domain.ErrDuplicateYear
		}
	}
	cp := *p
	s.data[p.ID] = &cp
	return nil
}

// Get retrieves an annual plan by ID scoped to an org.
func (s *AnnualPlanStore) Get(orgID, id uuid.UUID) (*domain.AnnualPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.data[id]
	if !ok || p.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

// List returns all annual plans for an org, optionally filtered by year.
func (s *AnnualPlanStore) List(orgID uuid.UUID, year int) []*domain.AnnualPlan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domain.AnnualPlan
	for _, p := range s.data {
		if p.OrgID != orgID {
			continue
		}
		if year != 0 && p.Year != year {
			continue
		}
		cp := *p
		out = append(out, &cp)
	}
	return out
}

// Update persists changes to an existing annual plan.
func (s *AnnualPlanStore) Update(p *domain.AnnualPlan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[p.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *p
	s.data[p.ID] = &cp
	return nil
}

// ---------------------------------------------------------------------------
// AssuranceMapStore
// ---------------------------------------------------------------------------

// AssuranceMapStore is a thread-safe in-memory store for assurance maps.
type AssuranceMapStore struct {
	mu   sync.RWMutex
	data map[uuid.UUID]*domain.AssuranceMap
}

// NewAssuranceMapStore creates an empty store.
func NewAssuranceMapStore() *AssuranceMapStore {
	return &AssuranceMapStore{data: make(map[uuid.UUID]*domain.AssuranceMap)}
}

// Create persists a new assurance map.
func (s *AssuranceMapStore) Create(m *domain.AssuranceMap) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *m
	s.data[m.ID] = &cp
	return nil
}

// Get retrieves an assurance map by ID scoped to an org.
func (s *AssuranceMapStore) Get(orgID, id uuid.UUID) (*domain.AssuranceMap, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.data[id]
	if !ok || m.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	cp := *m
	return &cp, nil
}

// List returns all assurance maps for an org.
func (s *AssuranceMapStore) List(orgID uuid.UUID) []*domain.AssuranceMap {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domain.AssuranceMap
	for _, m := range s.data {
		if m.OrgID == orgID {
			cp := *m
			out = append(out, &cp)
		}
	}
	return out
}

// Update persists changes to an existing assurance map.
func (s *AssuranceMapStore) Update(m *domain.AssuranceMap) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[m.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *m
	s.data[m.ID] = &cp
	return nil
}

// ---------------------------------------------------------------------------
// ResourceCalendarStore
// ---------------------------------------------------------------------------

// ResourceCalendarStore is a thread-safe in-memory store for resource calendars.
type ResourceCalendarStore struct {
	mu   sync.RWMutex
	data map[uuid.UUID]*domain.ResourceCalendar
}

// NewResourceCalendarStore creates an empty store.
func NewResourceCalendarStore() *ResourceCalendarStore {
	return &ResourceCalendarStore{data: make(map[uuid.UUID]*domain.ResourceCalendar)}
}

// Create persists a new resource calendar.
func (s *ResourceCalendarStore) Create(rc *domain.ResourceCalendar) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *rc
	s.data[rc.ID] = &cp
	return nil
}

// Get retrieves a resource calendar by ID scoped to an org.
func (s *ResourceCalendarStore) Get(orgID, id uuid.UUID) (*domain.ResourceCalendar, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rc, ok := s.data[id]
	if !ok || rc.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	cp := *rc
	return &cp, nil
}

// List returns all resource calendars for an org.
func (s *ResourceCalendarStore) List(orgID uuid.UUID) []*domain.ResourceCalendar {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domain.ResourceCalendar
	for _, rc := range s.data {
		if rc.OrgID == orgID {
			cp := *rc
			out = append(out, &cp)
		}
	}
	return out
}

// Update persists changes to an existing resource calendar.
func (s *ResourceCalendarStore) Update(rc *domain.ResourceCalendar) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[rc.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *rc
	s.data[rc.ID] = &cp
	return nil
}

// ---------------------------------------------------------------------------
// Stores — convenience bundle
// ---------------------------------------------------------------------------

// Stores bundles all planning stores together.
type Stores struct {
	StrategicPlans   *StrategicPlanStore
	AnnualPlans      *AnnualPlanStore
	AssuranceMaps    *AssuranceMapStore
	ResourceCalendars *ResourceCalendarStore
}

// New creates a bundle of all planning stores.
func New() *Stores {
	return &Stores{
		StrategicPlans:    NewStrategicPlanStore(),
		AnnualPlans:       NewAnnualPlanStore(),
		AssuranceMaps:     NewAssuranceMapStore(),
		ResourceCalendars: NewResourceCalendarStore(),
	}
}

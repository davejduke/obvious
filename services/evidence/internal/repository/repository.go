// Package repository provides evidence storage with in-memory and database backends.
package repository

import (
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/evidence/internal/models"
)

// Repository defines the evidence storage contract.
type Repository interface {
	Save(ev *models.Evidence) error
	FindByID(id uuid.UUID) (*models.Evidence, error)
	FindByControlID(controlID uuid.UUID) ([]*models.Evidence, error)
	SaveClassification(c *models.Classification) error
	FindClassificationByEvidenceID(evidenceID uuid.UUID) (*models.Classification, error)
	SaveQualityScore(qs *models.QualityScore) error
	FindQualityScoreByEvidenceID(evidenceID uuid.UUID) (*models.QualityScore, error)
	FindQualityScoresByControlID(controlID uuid.UUID) ([]*models.QualityScore, error)
}

// MemoryRepository is a thread-safe in-memory implementation used in tests.
type MemoryRepository struct {
	mu              sync.RWMutex
	evidence        map[uuid.UUID]*models.Evidence
	classifications map[uuid.UUID]*models.Classification // keyed by evidence_id
	qualityScores   map[uuid.UUID]*models.QualityScore   // keyed by evidence_id
}

// NewMemory creates a new in-memory repository.
func NewMemory() *MemoryRepository {
	return &MemoryRepository{
		evidence:        make(map[uuid.UUID]*models.Evidence),
		classifications: make(map[uuid.UUID]*models.Classification),
		qualityScores:   make(map[uuid.UUID]*models.QualityScore),
	}
}

// Save stores an evidence item.
func (r *MemoryRepository) Save(ev *models.Evidence) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.evidence[ev.ID] = ev
	return nil
}

// FindByID retrieves evidence by its ID.
func (r *MemoryRepository) FindByID(id uuid.UUID) (*models.Evidence, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ev, ok := r.evidence[id]
	if !ok {
		return nil, fmt.Errorf("evidence %s not found", id)
	}
	return ev, nil
}

// FindByControlID returns all evidence for a given control.
func (r *MemoryRepository) FindByControlID(controlID uuid.UUID) ([]*models.Evidence, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*models.Evidence
	for _, ev := range r.evidence {
		if ev.ControlID == controlID {
			out = append(out, ev)
		}
	}
	return out, nil
}

// SaveClassification stores a classification result (keyed by evidence ID).
func (r *MemoryRepository) SaveClassification(c *models.Classification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.classifications[c.EvidenceID] = c
	return nil
}

// FindClassificationByEvidenceID retrieves the classification for an evidence item.
func (r *MemoryRepository) FindClassificationByEvidenceID(evidenceID uuid.UUID) (*models.Classification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.classifications[evidenceID]
	if !ok {
		return nil, fmt.Errorf("classification for evidence %s not found", evidenceID)
	}
	return c, nil
}

// SaveQualityScore stores a quality score (keyed by evidence ID).
func (r *MemoryRepository) SaveQualityScore(qs *models.QualityScore) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.qualityScores[qs.EvidenceID] = qs
	return nil
}

// FindQualityScoreByEvidenceID retrieves the quality score for an evidence item.
func (r *MemoryRepository) FindQualityScoreByEvidenceID(evidenceID uuid.UUID) (*models.QualityScore, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	qs, ok := r.qualityScores[evidenceID]
	if !ok {
		return nil, fmt.Errorf("quality score for evidence %s not found", evidenceID)
	}
	return qs, nil
}

// FindQualityScoresByControlID returns all quality scores for evidence under a control.
func (r *MemoryRepository) FindQualityScoresByControlID(controlID uuid.UUID) ([]*models.QualityScore, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	evidence := make([]*models.Evidence, 0)
	for _, ev := range r.evidence {
		if ev.ControlID == controlID {
			evidence = append(evidence, ev)
		}
	}
	var scores []*models.QualityScore
	for _, ev := range evidence {
		if qs, ok := r.qualityScores[ev.ID]; ok {
			scores = append(scores, qs)
		}
	}
	return scores, nil
}


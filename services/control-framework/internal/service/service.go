// Package service implements the business logic for the Control Framework Service.
package service

import (
	"context"
	"fmt"

	"github.com/davejduke/obvious/services/control-framework/internal/domain"
	"github.com/davejduke/obvious/services/control-framework/internal/repository"
	"github.com/google/uuid"
)

// Service provides business logic for managing control frameworks and assessments.
type Service struct {
	repo repository.Repository
}

// New creates a new Service with the given repository.
func New(repo repository.Repository) *Service {
	return &Service{repo: repo}
}

// --- Framework operations ---

// ListFrameworks returns all frameworks for an org.
func (s *Service) ListFrameworks(ctx context.Context, orgID uuid.UUID) ([]domain.Framework, error) {
	return s.repo.ListFrameworks(ctx, orgID)
}

// GetFramework returns a framework by ID, or nil if not found.
func (s *Service) GetFramework(ctx context.Context, id uuid.UUID) (*domain.Framework, error) {
	return s.repo.GetFramework(ctx, id)
}

// CreateFramework creates a new control framework.
func (s *Service) CreateFramework(ctx context.Context, orgID uuid.UUID, req domain.CreateFrameworkRequest) (*domain.Framework, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.Version == "" {
		return nil, fmt.Errorf("version is required")
	}
	if req.Authority == "" {
		return nil, fmt.Errorf("authority is required")
	}
	return s.repo.CreateFramework(ctx, orgID, req)
}

// --- Control operations ---

// ListControls returns controls with optional filtering.
func (s *Service) ListControls(ctx context.Context, orgID uuid.UUID, filter domain.ListControlsFilter) (*domain.PaginatedControls, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}
	return s.repo.ListControls(ctx, orgID, filter)
}

// GetControl returns a control by ID.
func (s *Service) GetControl(ctx context.Context, id uuid.UUID) (*domain.Control, error) {
	return s.repo.GetControl(ctx, id)
}

// CreateControl creates a new control in a framework.
func (s *Service) CreateControl(ctx context.Context, orgID uuid.UUID, req domain.CreateControlRequest) (*domain.Control, error) {
	if req.ControlID == "" {
		return nil, fmt.Errorf("control_id is required")
	}
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if req.FrameworkID == uuid.Nil {
		return nil, fmt.Errorf("framework_id is required")
	}
	if req.RiskWeight < 0 || req.RiskWeight > 5 {
		return nil, fmt.Errorf("risk_weight must be between 0 and 5")
	}
	return s.repo.CreateControl(ctx, orgID, req)
}

// UpdateControl applies partial updates to a control.
func (s *Service) UpdateControl(ctx context.Context, id uuid.UUID, req domain.UpdateControlRequest) (*domain.Control, error) {
	if req.RiskWeight != nil && (*req.RiskWeight < 0 || *req.RiskWeight > 5) {
		return nil, fmt.Errorf("risk_weight must be between 0 and 5")
	}
	c, err := s.repo.UpdateControl(ctx, id, req)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("control not found")
	}
	return c, nil
}

// --- Mapping operations ---

// GetControlMappings returns all cross-framework mappings for a control.
func (s *Service) GetControlMappings(ctx context.Context, controlID uuid.UUID) ([]domain.FrameworkMapping, error) {
	return s.repo.GetControlMappings(ctx, controlID)
}

// --- Evidence requirements ---

// GetEvidenceRequirements returns the evidence requirements for a control.
func (s *Service) GetEvidenceRequirements(ctx context.Context, controlID uuid.UUID) ([]domain.EvidenceRequirement, error) {
	c, err := s.repo.GetControl(ctx, controlID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("control not found")
	}
	if c.EvidenceRequirements == nil {
		return []domain.EvidenceRequirement{}, nil
	}
	return c.EvidenceRequirements, nil
}

// --- Assessment operations ---

// AssessControl transitions a control assessment to a new status.
// The transition is validated against the allowed state machine before persisting.
func (s *Service) AssessControl(ctx context.Context, controlID uuid.UUID, orgID uuid.UUID, req domain.AssessControlRequest) (*domain.ControlAssessment, error) {
	if req.EngagementID == uuid.Nil {
		return nil, fmt.Errorf("engagement_id is required")
	}

	// Validate status
	switch req.Status {
	case domain.AssessmentStatusNotStarted,
		domain.AssessmentStatusInProgress,
		domain.AssessmentStatusAssessed,
		domain.AssessmentStatusRemediation:
	default:
		return nil, fmt.Errorf("invalid status %q: must be one of not_started, in_progress, assessed, remediation", req.Status)
	}

	// Get or create the assessment record
	current, err := s.repo.GetOrCreateAssessment(ctx, req.EngagementID, controlID, orgID)
	if err != nil {
		return nil, fmt.Errorf("get assessment: %w", err)
	}

	// Enforce state machine transition (skip validation if current == target, treat as idempotent update)
	if current.Status != req.Status && !current.Status.CanTransitionTo(req.Status) {
		return nil, fmt.Errorf(
			"invalid transition: cannot move from %q to %q",
			current.Status, req.Status,
		)
	}

	// Persist the transition
	updated, err := s.repo.UpdateAssessmentStatus(ctx, current.ID, req)
	if err != nil {
		return nil, fmt.Errorf("update assessment: %w", err)
	}
	return updated, nil
}

// GetAssessment returns the current assessment state for a control in an engagement.
func (s *Service) GetAssessment(ctx context.Context, engagementID, controlID uuid.UUID) (*domain.ControlAssessment, error) {
	a, err := s.repo.GetAssessment(ctx, engagementID, controlID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		// Return a synthetic not_started assessment if none exists
		return &domain.ControlAssessment{
			EngagementID: engagementID,
			ControlID:    controlID,
			Status:       domain.AssessmentStatusNotStarted,
			EvidenceIDs:  []uuid.UUID{},
		}, nil
	}
	return a, nil
}

// --- Audit objective generation ---

// GenerateAuditObjective generates an audit objective for a control based on its article reference.
func (s *Service) GenerateAuditObjective(ctx context.Context, controlID uuid.UUID) (*domain.AuditObjective, error) {
	c, err := s.repo.GetControl(ctx, controlID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("control not found")
	}

	var testApproaches []string
	for _, tp := range c.TestProcedures {
		testApproaches = append(testApproaches, tp.Description)
	}
	if len(testApproaches) == 0 {
		testApproaches = []string{"Review documentation", "Interview responsible personnel", "Inspect configuration"}
	}

	riskFocus := fmt.Sprintf(
		"Assess whether the organization has implemented effective controls for %s (risk weight: %.1f/5.0)",
		c.Title, c.RiskWeight,
	)

	return &domain.AuditObjective{
		ControlID:      c.ID,
		ControlRef:     c.ControlID,
		ArticleRef:     c.ArticleRef,
		Title:          fmt.Sprintf("Audit Objective: %s", c.Title),
		Objective:      c.Objective,
		TestApproaches: testApproaches,
		RiskFocus:      riskFocus,
	}, nil
}

// --- NIS 2 ontology helpers ---

// GetNIS2Domains returns the 10 NIS 2 Article 21 domain article references.
func (s *Service) GetNIS2Domains() map[string]string {
	return domain.ArticleRefNames
}


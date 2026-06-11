// Package repository defines the data access interface for the Control Framework Service.
package repository

import (
	"context"

	"github.com/davejduke/obvious/services/control-framework/internal/domain"
	"github.com/google/uuid"
)

// Repository is the data access interface for the Control Framework Service.
// Implementations: PostgresRepository (production), MemoryRepository (testing).
type Repository interface {
	// Framework operations
	ListFrameworks(ctx context.Context, orgID uuid.UUID) ([]domain.Framework, error)
	GetFramework(ctx context.Context, id uuid.UUID) (*domain.Framework, error)
	CreateFramework(ctx context.Context, orgID uuid.UUID, req domain.CreateFrameworkRequest) (*domain.Framework, error)

	// Control operations
	ListControls(ctx context.Context, orgID uuid.UUID, filter domain.ListControlsFilter) (*domain.PaginatedControls, error)
	GetControl(ctx context.Context, id uuid.UUID) (*domain.Control, error)
	CreateControl(ctx context.Context, orgID uuid.UUID, req domain.CreateControlRequest) (*domain.Control, error)
	UpdateControl(ctx context.Context, id uuid.UUID, req domain.UpdateControlRequest) (*domain.Control, error)

	// Mapping operations
	GetControlMappings(ctx context.Context, controlID uuid.UUID) ([]domain.FrameworkMapping, error)

	// Assessment operations
	GetOrCreateAssessment(ctx context.Context, engagementID, controlID, orgID uuid.UUID) (*domain.ControlAssessment, error)
	UpdateAssessmentStatus(ctx context.Context, assessmentID uuid.UUID, req domain.AssessControlRequest) (*domain.ControlAssessment, error)
	GetAssessment(ctx context.Context, engagementID, controlID uuid.UUID) (*domain.ControlAssessment, error)
}


// Package domain contains the audit planning domain model:
// StrategicPlan (3-year), AnnualPlan, AssuranceMap, and ResourceCalendar.
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Shared enumerations
// ---------------------------------------------------------------------------

// PlanStatus models the plan lifecycle state machine:
//
//	draft → approved → active → archived
type PlanStatus string

const (
	PlanStatusDraft    PlanStatus = "draft"
	PlanStatusApproved PlanStatus = "approved"
	PlanStatusActive   PlanStatus = "active"
	PlanStatusArchived PlanStatus = "archived"
)

// RiskRating is the risk-based prioritisation tier for auditable entities.
type RiskRating string

const (
	RiskCritical RiskRating = "critical"
	RiskHigh     RiskRating = "high"
	RiskMedium   RiskRating = "medium"
	RiskLow      RiskRating = "low"
)

// CoverageLevel describes assurance coverage for a business-unit/control-domain cell.
type CoverageLevel string

const (
	CoverageNone    CoverageLevel = "none"
	CoveragePartial CoverageLevel = "partial"
	CoverageFull    CoverageLevel = "full"
)

// EngagementPlanStatus is the scheduling status of a planned engagement.
type EngagementPlanStatus string

const (
	EngPlanPlanned    EngagementPlanStatus = "planned"
	EngPlanInProgress EngagementPlanStatus = "in_progress"
	EngPlanCompleted  EngagementPlanStatus = "completed"
	EngPlanDeferred   EngagementPlanStatus = "deferred"
)

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var (
	ErrNotFound         = errors.New("record not found")
	ErrInvalidRequest   = errors.New("invalid request")
	ErrInvalidTransition = errors.New("invalid plan status transition")
	ErrDuplicateYear    = errors.New("an annual plan for this year already exists")
)

// ---------------------------------------------------------------------------
// Strategic Plan
// ---------------------------------------------------------------------------

// AuditableEntity is a single auditable unit within a strategic plan,
// prioritised by risk rating and assigned to a year of the planning cycle.
type AuditableEntity struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	BusinessUnit   string     `json:"business_unit"`
	RiskRating     RiskRating `json:"risk_rating"`
	Priority       int        `json:"priority"` // lower = higher priority
	PlannedYear    int        `json:"planned_year"`
	ControlDomains []string   `json:"control_domains"`
	Notes          string     `json:"notes,omitempty"`
}

// StrategicPlan is the 3-year rolling audit plan for an organisation.
type StrategicPlan struct {
	ID          uuid.UUID         `json:"id"`
	OrgID       uuid.UUID         `json:"org_id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	StartYear   int               `json:"start_year"`
	EndYear     int               `json:"end_year"`
	Status      PlanStatus        `json:"status"`
	Version     int               `json:"version"`
	Entities    []AuditableEntity `json:"entities"`
	ApprovalID  *uuid.UUID        `json:"approval_id,omitempty"`
	ApprovedAt  *time.Time        `json:"approved_at,omitempty"`
	ApprovedBy  string            `json:"approved_by,omitempty"`
	Metadata    map[string]any    `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// CreateStrategicPlanRequest holds the data for creating a new strategic plan.
type CreateStrategicPlanRequest struct {
	OrgID       uuid.UUID         `json:"org_id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	StartYear   int               `json:"start_year"`
	EndYear     int               `json:"end_year"`
	Entities    []AuditableEntity `json:"entities,omitempty"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
}

// NewStrategicPlan creates a new StrategicPlan in draft status at version 1.
func NewStrategicPlan(req CreateStrategicPlanRequest) (*StrategicPlan, error) {
	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	if req.OrgID == uuid.Nil {
		return nil, errors.New("org_id is required")
	}
	if req.StartYear == 0 || req.EndYear == 0 || req.EndYear < req.StartYear {
		return nil, errors.New("valid start_year and end_year are required")
	}
	entities := req.Entities
	if entities == nil {
		entities = []AuditableEntity{}
	}
	// Assign IDs to any entities that are missing one.
	for i := range entities {
		if entities[i].ID == uuid.Nil {
			entities[i].ID = uuid.New()
		}
	}
	now := time.Now().UTC()
	return &StrategicPlan{
		ID:          uuid.New(),
		OrgID:       req.OrgID,
		Name:        req.Name,
		Description: req.Description,
		StartYear:   req.StartYear,
		EndYear:     req.EndYear,
		Status:      PlanStatusDraft,
		Version:     1,
		Entities:    entities,
		Metadata:    orEmptyMap(req.Metadata),
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Approve transitions a strategic plan to approved status.
func (p *StrategicPlan) Approve(byEmail string, approvalID *uuid.UUID) error {
	if p.Status != PlanStatusDraft {
		return ErrInvalidTransition
	}
	now := time.Now().UTC()
	p.Status = PlanStatusApproved
	p.ApprovedAt = &now
	p.ApprovedBy = byEmail
	p.ApprovalID = approvalID
	p.UpdatedAt = now
	return nil
}

// Activate transitions an approved plan to active.
func (p *StrategicPlan) Activate() error {
	if p.Status != PlanStatusApproved {
		return ErrInvalidTransition
	}
	p.Status = PlanStatusActive
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Archive moves a plan to archived (terminal).
func (p *StrategicPlan) Archive() error {
	if p.Status == PlanStatusArchived {
		return ErrInvalidTransition
	}
	p.Status = PlanStatusArchived
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateStrategicPlanRequest holds mutable fields.
type UpdateStrategicPlanRequest struct {
	Name        *string            `json:"name,omitempty"`
	Description *string            `json:"description,omitempty"`
	Entities    *[]AuditableEntity `json:"entities,omitempty"`
	Metadata    map[string]any     `json:"metadata,omitempty"`
}

// Apply applies a partial update and bumps the version.
func (p *StrategicPlan) Apply(req UpdateStrategicPlanRequest) {
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if req.Entities != nil {
		for i := range *req.Entities {
			if (*req.Entities)[i].ID == uuid.Nil {
				(*req.Entities)[i].ID = uuid.New()
			}
		}
		p.Entities = *req.Entities
	}
	p.Version++
	p.UpdatedAt = time.Now().UTC()
}

// ---------------------------------------------------------------------------
// Annual Plan
// ---------------------------------------------------------------------------

// PlannedEngagement is a scheduled audit engagement within an annual plan.
type PlannedEngagement struct {
	ID              uuid.UUID            `json:"id"`
	Name            string               `json:"name"`
	AuditableEntity string               `json:"auditable_entity"`
	AssignedTeam    []string             `json:"assigned_team"`
	LeadAuditorID   string               `json:"lead_auditor_id,omitempty"`
	Quarter         int                  `json:"quarter"` // 1–4
	StartDate       string               `json:"start_date"`
	EndDate         string               `json:"end_date"`
	BudgetDays      int                  `json:"budget_days"`
	Status          EngagementPlanStatus `json:"status"`
	EngagementID    *uuid.UUID           `json:"engagement_id,omitempty"`
}

// AnnualPlan schedules engagements across a single calendar year.
type AnnualPlan struct {
	ID               uuid.UUID           `json:"id"`
	OrgID            uuid.UUID           `json:"org_id"`
	StrategicPlanID  *uuid.UUID          `json:"strategic_plan_id,omitempty"`
	Year             int                 `json:"year"`
	Name             string              `json:"name"`
	Status           PlanStatus          `json:"status"`
	Version          int                 `json:"version"`
	Engagements      []PlannedEngagement `json:"engagements"`
	ApprovalID       *uuid.UUID          `json:"approval_id,omitempty"`
	Metadata         map[string]any      `json:"metadata"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

// CreateAnnualPlanRequest holds creation data for an annual plan.
type CreateAnnualPlanRequest struct {
	OrgID           uuid.UUID           `json:"org_id"`
	StrategicPlanID *uuid.UUID          `json:"strategic_plan_id,omitempty"`
	Year            int                 `json:"year"`
	Name            string              `json:"name"`
	Engagements     []PlannedEngagement `json:"engagements,omitempty"`
	Metadata        map[string]any      `json:"metadata,omitempty"`
}

// NewAnnualPlan creates a new AnnualPlan in draft status.
func NewAnnualPlan(req CreateAnnualPlanRequest) (*AnnualPlan, error) {
	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	if req.OrgID == uuid.Nil {
		return nil, errors.New("org_id is required")
	}
	if req.Year < 2000 || req.Year > 2100 {
		return nil, errors.New("valid year is required")
	}
	engs := req.Engagements
	if engs == nil {
		engs = []PlannedEngagement{}
	}
	for i := range engs {
		if engs[i].ID == uuid.Nil {
			engs[i].ID = uuid.New()
		}
		if engs[i].Status == "" {
			engs[i].Status = EngPlanPlanned
		}
	}
	now := time.Now().UTC()
	return &AnnualPlan{
		ID:              uuid.New(),
		OrgID:           req.OrgID,
		StrategicPlanID: req.StrategicPlanID,
		Year:            req.Year,
		Name:            req.Name,
		Status:          PlanStatusDraft,
		Version:         1,
		Engagements:     engs,
		Metadata:        orEmptyMap(req.Metadata),
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// Approve transitions an annual plan to approved.
func (p *AnnualPlan) Approve(approvalID *uuid.UUID) error {
	if p.Status != PlanStatusDraft {
		return ErrInvalidTransition
	}
	p.Status = PlanStatusApproved
	p.ApprovalID = approvalID
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Activate transitions an approved annual plan to active.
func (p *AnnualPlan) Activate() error {
	if p.Status != PlanStatusApproved {
		return ErrInvalidTransition
	}
	p.Status = PlanStatusActive
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Archive archives an annual plan.
func (p *AnnualPlan) Archive() error {
	if p.Status == PlanStatusArchived {
		return ErrInvalidTransition
	}
	p.Status = PlanStatusArchived
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateAnnualPlanRequest holds mutable fields.
type UpdateAnnualPlanRequest struct {
	Name        *string              `json:"name,omitempty"`
	Engagements *[]PlannedEngagement `json:"engagements,omitempty"`
	Metadata    map[string]any       `json:"metadata,omitempty"`
}

// Apply applies a partial update and bumps the version.
func (p *AnnualPlan) Apply(req UpdateAnnualPlanRequest) {
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Engagements != nil {
		for i := range *req.Engagements {
			if (*req.Engagements)[i].ID == uuid.Nil {
				(*req.Engagements)[i].ID = uuid.New()
			}
			if (*req.Engagements)[i].Status == "" {
				(*req.Engagements)[i].Status = EngPlanPlanned
			}
		}
		p.Engagements = *req.Engagements
	}
	p.Version++
	p.UpdatedAt = time.Now().UTC()
}

// ---------------------------------------------------------------------------
// Assurance Map
// ---------------------------------------------------------------------------

// AssuranceCell is a single cell in the assurance map matrix.
type AssuranceCell struct {
	BusinessUnit  string        `json:"business_unit"`
	ControlDomain string        `json:"control_domain"`
	Coverage      CoverageLevel `json:"coverage"`
	EngagementIDs []uuid.UUID   `json:"engagement_ids,omitempty"`
	Notes         string        `json:"notes,omitempty"`
}

// AssuranceMap is the visual coverage matrix: business units × control domains.
type AssuranceMap struct {
	ID             uuid.UUID       `json:"id"`
	OrgID          uuid.UUID       `json:"org_id"`
	AnnualPlanID   *uuid.UUID      `json:"annual_plan_id,omitempty"`
	Year           int             `json:"year"`
	Name           string          `json:"name"`
	BusinessUnits  []string        `json:"business_units"`
	ControlDomains []string        `json:"control_domains"`
	Matrix         []AssuranceCell `json:"matrix"`
	Metadata       map[string]any  `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// CreateAssuranceMapRequest holds creation data.
type CreateAssuranceMapRequest struct {
	OrgID          uuid.UUID       `json:"org_id"`
	AnnualPlanID   *uuid.UUID      `json:"annual_plan_id,omitempty"`
	Year           int             `json:"year"`
	Name           string          `json:"name"`
	BusinessUnits  []string        `json:"business_units"`
	ControlDomains []string        `json:"control_domains"`
	Matrix         []AssuranceCell `json:"matrix,omitempty"`
	Metadata       map[string]any  `json:"metadata,omitempty"`
}

// NewAssuranceMap creates a new AssuranceMap with an initialised matrix.
func NewAssuranceMap(req CreateAssuranceMapRequest) (*AssuranceMap, error) {
	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	if req.OrgID == uuid.Nil {
		return nil, errors.New("org_id is required")
	}
	matrix := req.Matrix
	if matrix == nil {
		// Auto-initialise matrix cells to CoverageNone.
		matrix = make([]AssuranceCell, 0, len(req.BusinessUnits)*len(req.ControlDomains))
		for _, bu := range req.BusinessUnits {
			for _, cd := range req.ControlDomains {
				matrix = append(matrix, AssuranceCell{
					BusinessUnit:  bu,
					ControlDomain: cd,
					Coverage:      CoverageNone,
				})
			}
		}
	}
	now := time.Now().UTC()
	return &AssuranceMap{
		ID:             uuid.New(),
		OrgID:          req.OrgID,
		AnnualPlanID:   req.AnnualPlanID,
		Year:           req.Year,
		Name:           req.Name,
		BusinessUnits:  req.BusinessUnits,
		ControlDomains: req.ControlDomains,
		Matrix:         matrix,
		Metadata:       orEmptyMap(req.Metadata),
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// UpdateAssuranceMapRequest allows matrix cells to be updated.
type UpdateAssuranceMapRequest struct {
	Name   *string         `json:"name,omitempty"`
	Matrix *[]AssuranceCell `json:"matrix,omitempty"`
}

// Apply applies partial updates to an assurance map.
func (m *AssuranceMap) Apply(req UpdateAssuranceMapRequest) {
	if req.Name != nil {
		m.Name = *req.Name
	}
	if req.Matrix != nil {
		m.Matrix = *req.Matrix
	}
	m.UpdatedAt = time.Now().UTC()
}

// ---------------------------------------------------------------------------
// Resource Calendar
// ---------------------------------------------------------------------------

// Assignment links an auditor to a specific planned engagement.
type Assignment struct {
	ID             uuid.UUID `json:"id"`
	EngagementName string    `json:"engagement_name"`
	StartDate      string    `json:"start_date"`
	EndDate        string    `json:"end_date"`
	AllocatedDays  int       `json:"allocated_days"`
	Quarter        int       `json:"quarter"`
}

// AuditorAllocation captures an auditor's availability and their assignments.
type AuditorAllocation struct {
	AuditorID    uuid.UUID    `json:"auditor_id"`
	AuditorName  string       `json:"auditor_name"`
	AuditorEmail string       `json:"auditor_email"`
	Role         string       `json:"role"` // auditor | senior_auditor | manager
	AvailableDays int         `json:"available_days"`
	Assignments  []Assignment `json:"assignments"`
}

// AllocatedDays returns the total days allocated to engagements.
func (a *AuditorAllocation) AllocatedDays() int {
	total := 0
	for _, asgn := range a.Assignments {
		total += asgn.AllocatedDays
	}
	return total
}

// UtilisationPct returns utilisation as 0–100 (capped at 100).
func (a *AuditorAllocation) UtilisationPct() int {
	if a.AvailableDays == 0 {
		return 0
	}
	pct := a.AllocatedDays() * 100 / a.AvailableDays
	if pct > 100 {
		return 100
	}
	return pct
}

// ResourceCalendar holds auditor availability and assignments for a year.
type ResourceCalendar struct {
	ID           uuid.UUID           `json:"id"`
	OrgID        uuid.UUID           `json:"org_id"`
	AnnualPlanID *uuid.UUID          `json:"annual_plan_id,omitempty"`
	Year         int                 `json:"year"`
	Name         string              `json:"name"`
	Auditors     []AuditorAllocation `json:"auditors"`
	Metadata     map[string]any      `json:"metadata"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

// CreateResourceCalendarRequest holds creation data.
type CreateResourceCalendarRequest struct {
	OrgID        uuid.UUID           `json:"org_id"`
	AnnualPlanID *uuid.UUID          `json:"annual_plan_id,omitempty"`
	Year         int                 `json:"year"`
	Name         string              `json:"name"`
	Auditors     []AuditorAllocation `json:"auditors,omitempty"`
	Metadata     map[string]any      `json:"metadata,omitempty"`
}

// NewResourceCalendar creates a new ResourceCalendar.
func NewResourceCalendar(req CreateResourceCalendarRequest) (*ResourceCalendar, error) {
	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	if req.OrgID == uuid.Nil {
		return nil, errors.New("org_id is required")
	}
	if req.Year < 2000 || req.Year > 2100 {
		return nil, errors.New("valid year is required")
	}
	auditors := req.Auditors
	if auditors == nil {
		auditors = []AuditorAllocation{}
	}
	// Ensure every assignment has an ID.
	for i := range auditors {
		if auditors[i].AuditorID == uuid.Nil {
			auditors[i].AuditorID = uuid.New()
		}
		for j := range auditors[i].Assignments {
			if auditors[i].Assignments[j].ID == uuid.Nil {
				auditors[i].Assignments[j].ID = uuid.New()
			}
		}
	}
	now := time.Now().UTC()
	return &ResourceCalendar{
		ID:           uuid.New(),
		OrgID:        req.OrgID,
		AnnualPlanID: req.AnnualPlanID,
		Year:         req.Year,
		Name:         req.Name,
		Auditors:     auditors,
		Metadata:     orEmptyMap(req.Metadata),
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// UpdateResourceCalendarRequest allows auditor allocations to be updated.
type UpdateResourceCalendarRequest struct {
	Name     *string              `json:"name,omitempty"`
	Auditors *[]AuditorAllocation `json:"auditors,omitempty"`
}

// Apply applies partial updates to a resource calendar.
func (rc *ResourceCalendar) Apply(req UpdateResourceCalendarRequest) {
	if req.Name != nil {
		rc.Name = *req.Name
	}
	if req.Auditors != nil {
		rc.Auditors = *req.Auditors
	}
	rc.UpdatedAt = time.Now().UTC()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func orEmptyMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

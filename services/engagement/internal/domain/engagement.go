// Package domain contains the engagement lifecycle state machine.
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Phase represents an engagement lifecycle phase.
type Phase string

const (
	PhasePlanning   Phase = "planning"
	PhaseFieldwork  Phase = "fieldwork"
	PhaseReporting  Phase = "reporting"
	PhaseMonitoring Phase = "monitoring"
	PhaseCompleted  Phase = "completed"
	PhaseCancelled  Phase = "cancelled"
)

// validTransitions defines the allowed state machine transitions.
// Each phase maps to the phases it can transition INTO.
var validTransitions = map[Phase][]Phase{
	PhasePlanning:   {PhaseFieldwork, PhaseCancelled},
	PhaseFieldwork:  {PhaseReporting, PhaseCancelled},
	PhaseReporting:  {PhaseMonitoring, PhaseCancelled},
	PhaseMonitoring: {PhaseCompleted, PhaseCancelled},
	PhaseCompleted:  {},
	PhaseCancelled:  {},
}

// ErrInvalidTransition is returned when a state transition is not allowed.
var ErrInvalidTransition = errors.New("invalid lifecycle transition")

// ErrEngagementNotFound is returned when an engagement does not exist.
var ErrEngagementNotFound = errors.New("engagement not found")

// ErrEngagementTerminal is returned when attempting to modify a terminal engagement.
var ErrEngagementTerminal = errors.New("engagement is in a terminal state")

// Engagement represents an audit engagement entity.
type Engagement struct {
	ID              uuid.UUID              `json:"id"`
	OrgID           uuid.UUID              `json:"org_id"`
	FrameworkID     uuid.UUID              `json:"framework_id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description,omitempty"`
	Phase           Phase                  `json:"phase"`
	Scope           map[string]interface{} `json:"scope"`
	LeadAuditorID   *uuid.UUID             `json:"lead_auditor_id,omitempty"`
	TargetStartDate *time.Time             `json:"target_start_date,omitempty"`
	TargetEndDate   *time.Time             `json:"target_end_date,omitempty"`
	ActualStartDate *time.Time             `json:"actual_start_date,omitempty"`
	ActualEndDate   *time.Time             `json:"actual_end_date,omitempty"`
	PhaseHistory    []PhaseEvent           `json:"phase_history,omitempty"`
	Metadata        map[string]interface{} `json:"metadata"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// PhaseEvent records a lifecycle phase transition.
type PhaseEvent struct {
	ID             uuid.UUID `json:"id"`
	EngagementID   uuid.UUID `json:"engagement_id"`
	FromPhase      Phase     `json:"from_phase"`
	ToPhase        Phase     `json:"to_phase"`
	TransitionedBy uuid.UUID `json:"transitioned_by"`
	Notes          string    `json:"notes,omitempty"`
	TransitionedAt time.Time `json:"transitioned_at"`
}

// CreateEngagementRequest holds data for creating a new engagement.
type CreateEngagementRequest struct {
	OrgID           uuid.UUID              `json:"org_id"`
	FrameworkID     uuid.UUID              `json:"framework_id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Scope           map[string]interface{} `json:"scope"`
	LeadAuditorID   *uuid.UUID             `json:"lead_auditor_id"`
	TargetStartDate *time.Time             `json:"target_start_date"`
	TargetEndDate   *time.Time             `json:"target_end_date"`
}

// UpdateEngagementRequest holds fields that can be updated.
type UpdateEngagementRequest struct {
	Name            *string                `json:"name"`
	Description     *string                `json:"description"`
	Scope           map[string]interface{} `json:"scope"`
	LeadAuditorID   *uuid.UUID             `json:"lead_auditor_id"`
	TargetStartDate *time.Time             `json:"target_start_date"`
	TargetEndDate   *time.Time             `json:"target_end_date"`
}

// TransitionRequest holds data for a phase transition.
type TransitionRequest struct {
	ToPhase        Phase     `json:"to_phase"`
	TransitionedBy uuid.UUID `json:"transitioned_by"`
	Notes          string    `json:"notes"`
}

// CanTransition returns true if the engagement can move to the target phase.
func (e *Engagement) CanTransition(to Phase) bool {
	allowed, ok := validTransitions[e.Phase]
	if !ok {
		return false
	}
	for _, p := range allowed {
		if p == to {
			return true
		}
	}
	return false
}

// IsTerminal returns true if the engagement is in a terminal state.
func (e *Engagement) IsTerminal() bool {
	return e.Phase == PhaseCompleted || e.Phase == PhaseCancelled
}

// Transition applies a lifecycle state transition.
// Returns ErrInvalidTransition if the transition is not allowed.
func (e *Engagement) Transition(req TransitionRequest) (PhaseEvent, error) {
	if e.IsTerminal() {
		return PhaseEvent{}, ErrEngagementTerminal
	}
	if !e.CanTransition(req.ToPhase) {
		return PhaseEvent{}, ErrInvalidTransition
	}

	now := time.Now().UTC()
	event := PhaseEvent{
		ID:             uuid.New(),
		EngagementID:   e.ID,
		FromPhase:      e.Phase,
		ToPhase:        req.ToPhase,
		TransitionedBy: req.TransitionedBy,
		Notes:          req.Notes,
		TransitionedAt: now,
	}

	e.Phase = req.ToPhase
	e.UpdatedAt = now

	if req.ToPhase == PhaseFieldwork && e.ActualStartDate == nil {
		e.ActualStartDate = &now
	}
	if (req.ToPhase == PhaseCompleted || req.ToPhase == PhaseCancelled) && e.ActualEndDate == nil {
		e.ActualEndDate = &now
	}

	e.PhaseHistory = append(e.PhaseHistory, event)
	return event, nil
}

// NewEngagement creates a new Engagement in the Planning phase.
func NewEngagement(req CreateEngagementRequest) *Engagement {
	now := time.Now().UTC()
	e := &Engagement{
		ID:              uuid.New(),
		OrgID:           req.OrgID,
		FrameworkID:     req.FrameworkID,
		Name:            req.Name,
		Description:     req.Description,
		Phase:           PhasePlanning,
		Scope:           req.Scope,
		LeadAuditorID:   req.LeadAuditorID,
		TargetStartDate: req.TargetStartDate,
		TargetEndDate:   req.TargetEndDate,
		Metadata:        map[string]interface{}{},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if e.Scope == nil {
		e.Scope = map[string]interface{}{}
	}
	return e
}

// AllowedNextPhases returns the list of phases this engagement can move to.
func (e *Engagement) AllowedNextPhases() []Phase {
	return validTransitions[e.Phase]
}


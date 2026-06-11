// Package domain_test tests the engagement lifecycle state machine.
package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/engagement/internal/domain"
)

func TestNewEngagement_StartsInPlanningPhase(t *testing.T) {
	req := domain.CreateEngagementRequest{
		OrgID:       uuid.New(),
		FrameworkID: uuid.New(),
		Name:        "NIS2 Audit 2024",
	}
	eng := domain.NewEngagement(req)

	if eng.Phase != domain.PhasePlanning {
		t.Errorf("expected Planning phase, got %s", eng.Phase)
	}
	if eng.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}
	if eng.Scope == nil {
		t.Error("expected non-nil scope map")
	}
}

func TestEngagement_ValidTransitions(t *testing.T) {
	tests := []struct {
		name  string
		from  domain.Phase
		to    domain.Phase
		wantOK bool
	}{
		{"planning->fieldwork", domain.PhasePlanning, domain.PhaseFieldwork, true},
		{"planning->reporting", domain.PhasePlanning, domain.PhaseReporting, false},
		{"planning->monitoring", domain.PhasePlanning, domain.PhaseMonitoring, false},
		{"planning->completed", domain.PhasePlanning, domain.PhaseCompleted, false},
		{"planning->cancelled", domain.PhasePlanning, domain.PhaseCancelled, true},
		{"fieldwork->reporting", domain.PhaseFieldwork, domain.PhaseReporting, true},
		{"fieldwork->monitoring", domain.PhaseFieldwork, domain.PhaseMonitoring, false},
		{"fieldwork->planning", domain.PhaseFieldwork, domain.PhasePlanning, false},
		{"fieldwork->cancelled", domain.PhaseFieldwork, domain.PhaseCancelled, true},
		{"reporting->monitoring", domain.PhaseReporting, domain.PhaseMonitoring, true},
		{"reporting->completed", domain.PhaseReporting, domain.PhaseCompleted, false},
		{"reporting->cancelled", domain.PhaseReporting, domain.PhaseCancelled, true},
		{"monitoring->completed", domain.PhaseMonitoring, domain.PhaseCompleted, true},
		{"monitoring->cancelled", domain.PhaseMonitoring, domain.PhaseCancelled, true},
		{"monitoring->fieldwork", domain.PhaseMonitoring, domain.PhaseFieldwork, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := makeEngagementInPhase(tt.from)
			got := eng.CanTransition(tt.to)
			if got != tt.wantOK {
				t.Errorf("CanTransition(%s->%s) = %v, want %v", tt.from, tt.to, got, tt.wantOK)
			}
		})
	}
}

func TestEngagement_Transition_Success(t *testing.T) {
	eng := makeEngagementInPhase(domain.PhasePlanning)
	actor := uuid.New()

	event, err := eng.Transition(domain.TransitionRequest{
		ToPhase:        domain.PhaseFieldwork,
		TransitionedBy: actor,
		Notes:          "fieldwork begins",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eng.Phase != domain.PhaseFieldwork {
		t.Errorf("expected Fieldwork phase, got %s", eng.Phase)
	}
	if event.FromPhase != domain.PhasePlanning {
		t.Errorf("expected FromPhase=Planning, got %s", event.FromPhase)
	}
	if event.ToPhase != domain.PhaseFieldwork {
		t.Errorf("expected ToPhase=Fieldwork, got %s", event.ToPhase)
	}
	if eng.ActualStartDate == nil {
		t.Error("expected ActualStartDate set when transitioning to Fieldwork")
	}
	if len(eng.PhaseHistory) != 1 {
		t.Errorf("expected 1 phase history entry, got %d", len(eng.PhaseHistory))
	}
}

func TestEngagement_Transition_InvalidTransition(t *testing.T) {
	eng := makeEngagementInPhase(domain.PhasePlanning)

	_, err := eng.Transition(domain.TransitionRequest{
		ToPhase:        domain.PhaseMonitoring,
		TransitionedBy: uuid.New(),
	})

	if err != domain.ErrInvalidTransition {
		t.Errorf("expected ErrInvalidTransition, got %v", err)
	}
	if eng.Phase != domain.PhasePlanning {
		t.Error("phase should not change on invalid transition")
	}
}

func TestEngagement_Transition_TerminalState(t *testing.T) {
	for _, phase := range []domain.Phase{domain.PhaseCompleted, domain.PhaseCancelled} {
		t.Run(string(phase), func(t *testing.T) {
			eng := makeEngagementInPhase(phase)
			_, err := eng.Transition(domain.TransitionRequest{
				ToPhase:        domain.PhasePlanning,
				TransitionedBy: uuid.New(),
			})
			if err != domain.ErrEngagementTerminal {
				t.Errorf("expected ErrEngagementTerminal, got %v", err)
			}
		})
	}
}

func TestEngagement_FullLifecycle(t *testing.T) {
	eng := domain.NewEngagement(domain.CreateEngagementRequest{
		OrgID:       uuid.New(),
		FrameworkID: uuid.New(),
		Name:        "Full Lifecycle Test",
	})
	actor := uuid.New()

	sequence := []domain.Phase{
		domain.PhaseFieldwork,
		domain.PhaseReporting,
		domain.PhaseMonitoring,
		domain.PhaseCompleted,
	}

	for _, phase := range sequence {
		_, err := eng.Transition(domain.TransitionRequest{
			ToPhase:        phase,
			TransitionedBy: actor,
		})
		if err != nil {
			t.Fatalf("unexpected error transitioning to %s: %v", phase, err)
		}
	}

	if eng.Phase != domain.PhaseCompleted {
		t.Errorf("expected Completed, got %s", eng.Phase)
	}
	if eng.ActualStartDate == nil {
		t.Error("ActualStartDate should be set")
	}
	if eng.ActualEndDate == nil {
		t.Error("ActualEndDate should be set")
	}
	if len(eng.PhaseHistory) != 4 {
		t.Errorf("expected 4 history entries, got %d", len(eng.PhaseHistory))
	}
	if eng.IsTerminal() != true {
		t.Error("completed engagement should be terminal")
	}
}

func TestEngagement_CancelFromAnyPhase(t *testing.T) {
	cancellable := []domain.Phase{
		domain.PhasePlanning,
		domain.PhaseFieldwork,
		domain.PhaseReporting,
		domain.PhaseMonitoring,
	}
	for _, phase := range cancellable {
		t.Run(string(phase), func(t *testing.T) {
			eng := makeEngagementInPhase(phase)
			_, err := eng.Transition(domain.TransitionRequest{
				ToPhase:        domain.PhaseCancelled,
				TransitionedBy: uuid.New(),
			})
			if err != nil {
				t.Errorf("expected cancel to succeed from %s, got %v", phase, err)
			}
		})
	}
}

func TestEngagement_AllowedNextPhases(t *testing.T) {
	eng := makeEngagementInPhase(domain.PhasePlanning)
	next := eng.AllowedNextPhases()
	if len(next) != 2 {
		t.Errorf("expected 2 allowed phases from Planning, got %d", len(next))
	}
}

// makeEngagementInPhase creates an Engagement directly in the specified phase
// without going through transitions (for test setup convenience).
func makeEngagementInPhase(phase domain.Phase) *domain.Engagement {
	eng := domain.NewEngagement(domain.CreateEngagementRequest{
		OrgID:       uuid.New(),
		FrameworkID: uuid.New(),
		Name:        "Test Engagement",
	})
	eng.Phase = phase
	eng.UpdatedAt = time.Now().UTC()
	return eng
}


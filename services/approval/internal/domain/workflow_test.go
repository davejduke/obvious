// Package domain — unit tests for the approval workflow state machine.
package domain

import (
	"testing"

	"github.com/google/uuid"
)

var (
	orgID    = uuid.New()
	engID    = uuid.New()
	resID    = uuid.New()
	elUserID = uuid.New()  // engagement_lead actor
	caeUserID = uuid.New() // chief_audit_executive actor
)

func newPlanWorkflow() *Workflow {
	return NewWorkflow(CreateWorkflowRequest{
		OrgID:        orgID,
		WorkflowType: WorkflowTypeAuditPlan,
		ResourceType: "engagement",
		ResourceID:   resID,
		EngagementID: &engID,
	})
}

func newFindingWorkflow() *Workflow {
	return NewWorkflow(CreateWorkflowRequest{
		OrgID:        orgID,
		WorkflowType: WorkflowTypeFindingSignoff,
		ResourceType: "finding",
		ResourceID:   resID,
		EngagementID: &engID,
	})
}

func newReportWorkflow() *Workflow {
	return NewWorkflow(CreateWorkflowRequest{
		OrgID:        orgID,
		WorkflowType: WorkflowTypeReportRelease,
		ResourceType: "report",
		ResourceID:   resID,
		EngagementID: &engID,
	})
}

// ------------------------------------------------------------
// Audit Plan Approval
// ------------------------------------------------------------

func TestAuditPlan_HappyPath(t *testing.T) {
	w := newPlanWorkflow()
	if w.Status != StatusDraft {
		t.Fatalf("expected draft, got %s", w.Status)
	}

	// EL submits
	_, err := w.Submit(SubmitRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if w.Status != StatusPendingApproval {
		t.Fatalf("expected pending_approval, got %s", w.Status)
	}

	// CAE approves
	_, err = w.Approve(ApproveRequest{ActorID: caeUserID, ActorEmail: "cae@test.com", ActorRole: RoleChiefAuditExecutive})
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if w.Status != StatusApproved {
		t.Fatalf("expected approved, got %s", w.Status)
	}

	if len(w.History) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(w.History))
	}
}

func TestAuditPlan_OnlyELCanSubmit(t *testing.T) {
	w := newPlanWorkflow()
	_, err := w.Submit(SubmitRequest{ActorID: caeUserID, ActorEmail: "cae@test.com", ActorRole: RoleChiefAuditExecutive})
	if err != ErrUnauthorizedRole {
		t.Fatalf("expected ErrUnauthorizedRole, got %v", err)
	}
}

func TestAuditPlan_OnlyCAECanApprove(t *testing.T) {
	w := newPlanWorkflow()
	_, _ = w.Submit(SubmitRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead})

	// EL should NOT be able to approve their own plan
	_, err := w.Approve(ApproveRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead})
	if err != ErrUnauthorizedRole {
		t.Fatalf("expected ErrUnauthorizedRole, got %v", err)
	}
}

func TestAuditPlan_RejectionReturnsToDraft(t *testing.T) {
	w := newPlanWorkflow()
	_, _ = w.Submit(SubmitRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead})
	_, err := w.Reject(RejectRequest{ActorID: caeUserID, ActorEmail: "cae@test.com", ActorRole: RoleChiefAuditExecutive, RejectionReason: "missing risk coverage"})
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if w.Status != StatusRejected {
		t.Fatalf("expected rejected, got %s", w.Status)
	}
	if w.RejectionReason == "" {
		t.Fatal("expected rejection_reason to be set")
	}

	// EL acknowledges and returns to draft for rework
	_, err = w.ReturnToDraft(SubmitRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead, Comment: "addressed feedback"})
	if err != nil {
		t.Fatalf("ReturnToDraft: %v", err)
	}
	if w.Status != StatusDraft {
		t.Fatalf("expected draft, got %s", w.Status)
	}
	if w.RejectionReason != "" {
		t.Fatal("expected rejection_reason to be cleared")
	}
}

func TestAuditPlan_RejectRequiresReason(t *testing.T) {
	w := newPlanWorkflow()
	_, _ = w.Submit(SubmitRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead})
	_, err := w.Reject(RejectRequest{ActorID: caeUserID, ActorEmail: "cae@test.com", ActorRole: RoleChiefAuditExecutive})
	if err == nil {
		t.Fatal("expected error for empty rejection_reason")
	}
}

// ------------------------------------------------------------
// Finding Sign-Off
// ------------------------------------------------------------

func TestFindingSignoff_HappyPath(t *testing.T) {
	w := newFindingWorkflow()

	_, err := w.Submit(SubmitRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// CAE sign-off
	_, err = w.Approve(ApproveRequest{ActorID: caeUserID, ActorEmail: "cae@test.com", ActorRole: RoleChiefAuditExecutive})
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if w.Status != StatusApproved {
		t.Fatalf("expected approved, got %s", w.Status)
	}
}

func TestFindingSignoff_ELCanApprove(t *testing.T) {
	w := newFindingWorkflow()
	_, _ = w.Submit(SubmitRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead})
	_, err := w.Approve(ApproveRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead})
	if err != nil {
		t.Fatalf("EL should be allowed to approve finding signoff, got %v", err)
	}
}

// ------------------------------------------------------------
// Report Release
// ------------------------------------------------------------

func TestReportRelease_HappyPath(t *testing.T) {
	w := newReportWorkflow()

	_, err := w.Submit(SubmitRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	_, err = w.Approve(ApproveRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead})
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if w.Status != StatusApproved {
		t.Fatalf("expected approved, got %s", w.Status)
	}

	// Password re-entry gate before locking
	_, err = w.Lock(LockRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead, PasswordConfirm: "secret"})
	if err != nil {
		t.Fatalf("Lock: %v", err)
	}
	if w.Status != StatusLocked {
		t.Fatalf("expected locked, got %s", w.Status)
	}
}

func TestReportRelease_LockRequiresPassword(t *testing.T) {
	w := newReportWorkflow()
	_, _ = w.Submit(SubmitRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead})
	_, _ = w.Approve(ApproveRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead})

	// Attempt lock without password
	_, err := w.Lock(LockRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead})
	if err != ErrPasswordRequired {
		t.Fatalf("expected ErrPasswordRequired, got %v", err)
	}
}

func TestReportRelease_LockedWorkflowIsImmutable(t *testing.T) {
	w := newReportWorkflow()
	_, _ = w.Submit(SubmitRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead})
	_, _ = w.Approve(ApproveRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead})
	_, _ = w.Lock(LockRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead, PasswordConfirm: "x"})

	// Any further action should be blocked
	_, err := w.Submit(SubmitRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead})
	if err != ErrWorkflowLocked {
		t.Fatalf("expected ErrWorkflowLocked, got %v", err)
	}
}

// ------------------------------------------------------------
// Invalid transitions
// ------------------------------------------------------------

func TestInvalidTransitions(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*Workflow) error
	}{
		{
			name: "approve_from_draft",
			fn: func(w *Workflow) error {
				_, err := w.Approve(ApproveRequest{ActorID: caeUserID, ActorEmail: "cae@test.com", ActorRole: RoleChiefAuditExecutive})
				return err
			},
		},
		{
			name: "reject_from_draft",
			fn: func(w *Workflow) error {
				_, err := w.Reject(RejectRequest{ActorID: caeUserID, ActorEmail: "cae@test.com", ActorRole: RoleChiefAuditExecutive, RejectionReason: "x"})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newPlanWorkflow()
			err := tt.fn(w)
			if err != ErrInvalidTransition {
				t.Fatalf("expected ErrInvalidTransition, got %v", err)
			}
		})
	}
}

// ------------------------------------------------------------
// History audit trail
// ------------------------------------------------------------

func TestHistoryIsAppendOnly(t *testing.T) {
	w := newPlanWorkflow()
	_, _ = w.Submit(SubmitRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead, Comment: "plan ready"})
	_, _ = w.Reject(RejectRequest{ActorID: caeUserID, ActorEmail: "cae@test.com", ActorRole: RoleChiefAuditExecutive, RejectionReason: "scope too narrow"})
	_, _ = w.ReturnToDraft(SubmitRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead})
	_, _ = w.Submit(SubmitRequest{ActorID: elUserID, ActorEmail: "el@test.com", ActorRole: RoleEngagementLead, Comment: "scope expanded"})
	_, _ = w.Approve(ApproveRequest{ActorID: caeUserID, ActorEmail: "cae@test.com", ActorRole: RoleChiefAuditExecutive})

	if len(w.History) != 5 {
		t.Fatalf("expected 5 history entries, got %d", len(w.History))
	}
	if w.History[0].Action != "submitted" {
		t.Errorf("first entry should be 'submitted'")
	}
	if w.History[4].Action != "approved" {
		t.Errorf("last entry should be 'approved'")
	}
}

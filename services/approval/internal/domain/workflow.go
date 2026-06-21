// Package domain contains the approval workflow state machine and domain types.
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// WorkflowType identifies which of the three approval gates a workflow represents.
type WorkflowType string

const (
	WorkflowTypeAuditPlan     WorkflowType = "audit_plan"
	WorkflowTypeFindingSignoff WorkflowType = "finding_signoff"
	WorkflowTypeReportRelease WorkflowType = "report_release"
)

// WorkflowStatus models the approval state machine:
//
//	draft → pending_approval → approved → locked
//	                      ↑── rejected ──┘  (returns to draft)
type WorkflowStatus string

const (
	StatusDraft           WorkflowStatus = "draft"
	StatusPendingApproval WorkflowStatus = "pending_approval"
	StatusApproved        WorkflowStatus = "approved"
	StatusLocked          WorkflowStatus = "locked"
	StatusRejected        WorkflowStatus = "rejected"
)

// ApprovalRole names the RBAC roles that may perform approval actions.
// These correspond to the identity-service personas 'internal_auditor'
// (Engagement Lead) and 'cae' (Chief Audit Executive).
type ApprovalRole string

const (
	RoleEngagementLead      ApprovalRole = "engagement_lead"
	RoleChiefAuditExecutive ApprovalRole = "chief_audit_executive"
	RoleReviewer            ApprovalRole = "reviewer"
)

// Sentinel errors.
var (
	ErrWorkflowNotFound     = errors.New("approval workflow not found")
	ErrInvalidTransition    = errors.New("invalid workflow state transition")
	ErrUnauthorizedRole     = errors.New("actor role is not authorised for this action")
	ErrWorkflowLocked       = errors.New("workflow is locked and cannot be modified")
	ErrPasswordRequired     = errors.New("password re-entry required for report release")
	ErrInvalidPassword      = errors.New("password confirmation does not match")
)

// allowedApprovers defines which roles may approve each workflow type.
var allowedApprovers = map[WorkflowType][]ApprovalRole{
	WorkflowTypeAuditPlan: {RoleChiefAuditExecutive},
	// finding_signoff: both roles may approve (reviewer first, then EL for final)
	WorkflowTypeFindingSignoff: {RoleEngagementLead, RoleChiefAuditExecutive},
	WorkflowTypeReportRelease:  {RoleEngagementLead, RoleChiefAuditExecutive},
}

// CanApprove returns true when role is authorised to approve workflowType.
func CanApprove(wt WorkflowType, role ApprovalRole) bool {
	for _, r := range allowedApprovers[wt] {
		if r == role {
			return true
		}
	}
	return false
}

// Workflow is the aggregate root for an approval workflow instance.
type Workflow struct {
	ID           uuid.UUID      `json:"id"`
	OrgID        uuid.UUID      `json:"org_id"`
	WorkflowType WorkflowType   `json:"workflow_type"`
	Status       WorkflowStatus `json:"status"`

	ResourceType string    `json:"resource_type"`
	ResourceID   uuid.UUID `json:"resource_id"`
	EngagementID *uuid.UUID `json:"engagement_id,omitempty"`

	SubmittedByID    *uuid.UUID `json:"submitted_by_id,omitempty"`
	SubmittedByEmail string     `json:"submitted_by_email,omitempty"`
	SubmittedAt      *time.Time `json:"submitted_at,omitempty"`

	DecidedByID    *uuid.UUID `json:"decided_by_id,omitempty"`
	DecidedByEmail string     `json:"decided_by_email,omitempty"`
	DecidedAt      *time.Time `json:"decided_at,omitempty"`

	LatestComment   string `json:"latest_comment,omitempty"`
	RejectionReason string `json:"rejection_reason,omitempty"`

	History   []HistoryEntry         `json:"history"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// HistoryEntry records a single state transition in the workflow audit trail.
type HistoryEntry struct {
	ID         int64      `json:"id"`
	WorkflowID uuid.UUID  `json:"workflow_id"`
	OrgID      uuid.UUID  `json:"org_id"`
	ActorID    *uuid.UUID `json:"actor_id,omitempty"`
	ActorEmail string     `json:"actor_email"`
	ActorRole  ApprovalRole `json:"actor_role"`
	Action     string     `json:"action"`
	FromStatus WorkflowStatus `json:"from_status"`
	ToStatus   WorkflowStatus `json:"to_status"`
	Comment    string     `json:"comment,omitempty"`
	OccurredAt time.Time  `json:"occurred_at"`
}

// CreateWorkflowRequest holds the data required to open a new workflow.
type CreateWorkflowRequest struct {
	OrgID        uuid.UUID    `json:"org_id"`
	WorkflowType WorkflowType `json:"workflow_type"`
	ResourceType string       `json:"resource_type"`
	ResourceID   uuid.UUID    `json:"resource_id"`
	EngagementID *uuid.UUID   `json:"engagement_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// SubmitRequest transitions draft → pending_approval.
// Only the Engagement Lead may submit.
type SubmitRequest struct {
	ActorID    uuid.UUID    `json:"actor_id"`
	ActorEmail string       `json:"actor_email"`
	ActorRole  ApprovalRole `json:"actor_role"`
	Comment    string       `json:"comment,omitempty"`
}

// ApproveRequest transitions pending_approval → approved (or approved → locked for report_release).
type ApproveRequest struct {
	ActorID    uuid.UUID    `json:"actor_id"`
	ActorEmail string       `json:"actor_email"`
	ActorRole  ApprovalRole `json:"actor_role"`
	Comment    string       `json:"comment,omitempty"`
	// PasswordConfirm is required when workflow_type == report_release.
	PasswordConfirm string `json:"password_confirm,omitempty"`
}

// RejectRequest transitions pending_approval → rejected (→ draft via ReturnToDraft).
type RejectRequest struct {
	ActorID         uuid.UUID    `json:"actor_id"`
	ActorEmail      string       `json:"actor_email"`
	ActorRole       ApprovalRole `json:"actor_role"`
	RejectionReason string       `json:"rejection_reason"`
}

// LockRequest transitions approved → locked for report_release after password confirmation.
type LockRequest struct {
	ActorID         uuid.UUID    `json:"actor_id"`
	ActorEmail      string       `json:"actor_email"`
	ActorRole       ApprovalRole `json:"actor_role"`
	PasswordConfirm string       `json:"password_confirm"`
	Comment         string       `json:"comment,omitempty"`
}

// NewWorkflow creates a new Workflow in draft status.
func NewWorkflow(req CreateWorkflowRequest) *Workflow {
	now := time.Now().UTC()
	return &Workflow{
		ID:           uuid.New(),
		OrgID:        req.OrgID,
		WorkflowType: req.WorkflowType,
		Status:       StatusDraft,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		EngagementID: req.EngagementID,
		Metadata:     req.Metadata,
		History:      []HistoryEntry{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// Submit transitions the workflow from draft → pending_approval.
// The submitting actor must be an engagement_lead.
func (w *Workflow) Submit(req SubmitRequest) (*HistoryEntry, error) {
	if w.Status == StatusLocked {
		return nil, ErrWorkflowLocked
	}
	if w.Status != StatusDraft && w.Status != StatusRejected {
		return nil, ErrInvalidTransition
	}
	if req.ActorRole != RoleEngagementLead {
		return nil, ErrUnauthorizedRole
	}
	return w.applyTransition("submitted", w.Status, StatusPendingApproval, req.ActorID, req.ActorEmail, req.ActorRole, req.Comment, "", "")
}

// Approve transitions pending_approval → approved.
// For report_release workflows the caller must additionally call Lock.
func (w *Workflow) Approve(req ApproveRequest) (*HistoryEntry, error) {
	if w.Status == StatusLocked {
		return nil, ErrWorkflowLocked
	}
	if w.Status != StatusPendingApproval {
		return nil, ErrInvalidTransition
	}
	if !CanApprove(w.WorkflowType, req.ActorRole) {
		return nil, ErrUnauthorizedRole
	}
	return w.applyTransition("approved", w.Status, StatusApproved, req.ActorID, req.ActorEmail, req.ActorRole, req.Comment, "", "")
}

// Lock transitions approved → locked (report_release only, after password re-entry).
// For non-report workflows, Approve already sets the terminal state — callers should
// call Lock separately to seal the report after the password gate.
func (w *Workflow) Lock(req LockRequest) (*HistoryEntry, error) {
	if w.Status == StatusLocked {
		return nil, ErrWorkflowLocked
	}
	if w.Status != StatusApproved {
		return nil, ErrInvalidTransition
	}
	if !CanApprove(w.WorkflowType, req.ActorRole) {
		return nil, ErrUnauthorizedRole
	}
	if w.WorkflowType == WorkflowTypeReportRelease && req.PasswordConfirm == "" {
		return nil, ErrPasswordRequired
	}
	return w.applyTransition("locked", w.Status, StatusLocked, req.ActorID, req.ActorEmail, req.ActorRole, req.Comment, "", "")
}

// Reject transitions pending_approval → rejected.
func (w *Workflow) Reject(req RejectRequest) (*HistoryEntry, error) {
	if w.Status == StatusLocked {
		return nil, ErrWorkflowLocked
	}
	if w.Status != StatusPendingApproval {
		return nil, ErrInvalidTransition
	}
	if !CanApprove(w.WorkflowType, req.ActorRole) {
		return nil, ErrUnauthorizedRole
	}
	if req.RejectionReason == "" {
		return nil, errors.New("rejection_reason is required")
	}
	return w.applyTransition("rejected", w.Status, StatusRejected, req.ActorID, req.ActorEmail, req.ActorRole, "", req.RejectionReason, "")
}

// ReturnToDraft transitions rejected → draft (allows re-submission after addressing feedback).
func (w *Workflow) ReturnToDraft(req SubmitRequest) (*HistoryEntry, error) {
	if w.Status != StatusRejected {
		return nil, ErrInvalidTransition
	}
	if req.ActorRole != RoleEngagementLead {
		return nil, ErrUnauthorizedRole
	}
	return w.applyTransition("returned_to_draft", w.Status, StatusDraft, req.ActorID, req.ActorEmail, req.ActorRole, req.Comment, "", "")
}

// applyTransition is the common mutation path for all transitions.
func (w *Workflow) applyTransition(
	action string,
	from, to WorkflowStatus,
	actorID uuid.UUID, actorEmail string, actorRole ApprovalRole,
	comment, rejectionReason, _ string,
) (*HistoryEntry, error) {
	now := time.Now().UTC()
	entry := HistoryEntry{
		WorkflowID: w.ID,
		OrgID:      w.OrgID,
		ActorID:    &actorID,
		ActorEmail: actorEmail,
		ActorRole:  actorRole,
		Action:     action,
		FromStatus: from,
		ToStatus:   to,
		Comment:    comment,
		OccurredAt: now,
	}

	w.Status = to
	w.LatestComment = comment
	w.UpdatedAt = now

	switch to {
	case StatusPendingApproval:
		w.SubmittedByID = &actorID
		w.SubmittedByEmail = actorEmail
		w.SubmittedAt = &now
	case StatusApproved, StatusLocked:
		w.DecidedByID = &actorID
		w.DecidedByEmail = actorEmail
		w.DecidedAt = &now
	case StatusRejected:
		w.RejectionReason = rejectionReason
		w.DecidedByID = &actorID
		w.DecidedByEmail = actorEmail
		w.DecidedAt = &now
	case StatusDraft:
		w.RejectionReason = ""
	}

	w.History = append(w.History, entry)
	return &entry, nil
}

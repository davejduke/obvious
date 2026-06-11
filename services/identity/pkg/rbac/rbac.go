// Package rbac implements the 7-persona RBAC permission matrix for AIAUDITOR.
package rbac

import "github.com/davejduke/obvious/services/identity/internal/models"

// Action represents an operation on a resource.
type Action string

const (
	ActionRead   Action = "read"
	ActionWrite  Action = "write"
	ActionDelete Action = "delete"
	ActionManage Action = "manage" // superset: read+write+delete
)

// Resource represents a protected resource class.
type Resource string

const (
	ResourceEngagement  Resource = "engagement"
	ResourceFinding     Resource = "finding"
	ResourceControl     Resource = "control"
	ResourceEvidence    Resource = "evidence"
	ResourceReport      Resource = "report"
	ResourceUser        Resource = "user"
	ResourceOrg         Resource = "org"
	ResourceAuditTrail  Resource = "audit_trail"
	ResourceWorkPaper   Resource = "work_paper"
	ResourceDashboard   Resource = "dashboard"
)

// Permission combines a resource and action.
type Permission struct {
	Resource Resource
	Action   Action
}

// personaPermissions defines what each persona is allowed to do.
// More permissive personas (CAE) are defined explicitly rather than inheriting
// so that the matrix is easy to audit and modify.
var personaPermissions = map[models.Persona][]Permission{
	// internal_auditor: create/manage engagements, findings, work papers
	models.PersonaInternalAuditor: {
		{ResourceEngagement, ActionRead},
		{ResourceEngagement, ActionWrite},
		{ResourceFinding, ActionRead},
		{ResourceFinding, ActionWrite},
		{ResourceControl, ActionRead},
		{ResourceEvidence, ActionRead},
		{ResourceEvidence, ActionWrite},
		{ResourceWorkPaper, ActionRead},
		{ResourceWorkPaper, ActionWrite},
		{ResourceReport, ActionRead},
		{ResourceDashboard, ActionRead},
		{ResourceAuditTrail, ActionRead},
	},
	// cae: Chief Audit Executive — all internal_auditor + org management + user management
	models.PersonaCAE: {
		{ResourceEngagement, ActionManage},
		{ResourceFinding, ActionManage},
		{ResourceControl, ActionManage},
		{ResourceEvidence, ActionManage},
		{ResourceWorkPaper, ActionManage},
		{ResourceReport, ActionManage},
		{ResourceDashboard, ActionManage},
		{ResourceAuditTrail, ActionRead},
		{ResourceUser, ActionManage},
		{ResourceOrg, ActionRead},
		{ResourceOrg, ActionWrite},
	},
	// audit_committee: read-only on reports and dashboards
	models.PersonaAuditCommittee: {
		{ResourceReport, ActionRead},
		{ResourceDashboard, ActionRead},
		{ResourceFinding, ActionRead},
		{ResourceEngagement, ActionRead},
	},
	// auditee_ciso: read findings, respond (write) on their own org
	models.PersonaAuditeeCISO: {
		{ResourceFinding, ActionRead},
		{ResourceFinding, ActionWrite},
		{ResourceEngagement, ActionRead},
		{ResourceControl, ActionRead},
		{ResourceEvidence, ActionRead},
		{ResourceEvidence, ActionWrite},
		{ResourceDashboard, ActionRead},
	},
	// cosourced_provider: limited — read/write in assigned engagements only
	models.PersonaCosourcedProvider: {
		{ResourceEngagement, ActionRead},
		{ResourceFinding, ActionRead},
		{ResourceFinding, ActionWrite},
		{ResourceEvidence, ActionRead},
		{ResourceEvidence, ActionWrite},
		{ResourceWorkPaper, ActionRead},
		{ResourceWorkPaper, ActionWrite},
	},
	// ptwg_member: Peer Technical Working Group — read work papers, submit comments
	models.PersonaPTWGMember: {
		{ResourceWorkPaper, ActionRead},
		{ResourceWorkPaper, ActionWrite},
		{ResourceControl, ActionRead},
		{ResourceFinding, ActionRead},
		{ResourceReport, ActionRead},
	},
	// beta_tester: broad read access, no writes to core audit data
	models.PersonaBetaTester: {
		{ResourceEngagement, ActionRead},
		{ResourceFinding, ActionRead},
		{ResourceControl, ActionRead},
		{ResourceEvidence, ActionRead},
		{ResourceReport, ActionRead},
		{ResourceDashboard, ActionRead},
		{ResourceWorkPaper, ActionRead},
	},
}

// Checker evaluates RBAC permissions.
type Checker struct{}

// NewChecker creates a new Checker.
func NewChecker() *Checker { return &Checker{} }

// Can returns true if the persona is permitted to perform action on resource.
func (c *Checker) Can(persona models.Persona, resource Resource, action Action) bool {
	perms, ok := personaPermissions[persona]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p.Resource == resource {
			if p.Action == action || p.Action == ActionManage {
				return true
			}
		}
	}
	return false
}

// Permissions returns all permissions for a given persona.
func (c *Checker) Permissions(persona models.Persona) []Permission {
	return personaPermissions[persona]
}

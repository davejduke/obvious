package rbac_test

import (
	"testing"

	"github.com/davejduke/obvious/services/identity/internal/models"
	"github.com/davejduke/obvious/services/identity/pkg/rbac"
)

func TestRBACPermissions(t *testing.T) {
	checker := rbac.NewChecker()

	tests := []struct {
		persona  models.Persona
		resource rbac.Resource
		action   rbac.Action
		want     bool
		desc     string
	}{
		// internal_auditor CAN read findings
		{models.PersonaInternalAuditor, rbac.ResourceFinding, rbac.ActionRead, true, "auditor can read findings"},
		// internal_auditor CAN write evidence
		{models.PersonaInternalAuditor, rbac.ResourceEvidence, rbac.ActionWrite, true, "auditor can write evidence"},
		// internal_auditor CANNOT manage users
		{models.PersonaInternalAuditor, rbac.ResourceUser, rbac.ActionManage, false, "auditor cannot manage users"},

		// cae CAN manage engagements
		{models.PersonaCAE, rbac.ResourceEngagement, rbac.ActionManage, true, "cae can manage engagements"},
		// cae CAN manage users
		{models.PersonaCAE, rbac.ResourceUser, rbac.ActionManage, true, "cae can manage users"},

		// audit_committee read-only
		{models.PersonaAuditCommittee, rbac.ResourceReport, rbac.ActionRead, true, "committee can read reports"},
		{models.PersonaAuditCommittee, rbac.ResourceReport, rbac.ActionWrite, false, "committee cannot write reports"},
		{models.PersonaAuditCommittee, rbac.ResourceUser, rbac.ActionRead, false, "committee cannot read users"},

		// auditee_ciso CAN write findings (respond)
		{models.PersonaAuditeeCISO, rbac.ResourceFinding, rbac.ActionWrite, true, "auditee_ciso can write findings"},
		// auditee_ciso CANNOT manage users
		{models.PersonaAuditeeCISO, rbac.ResourceUser, rbac.ActionManage, false, "auditee_ciso cannot manage users"},

		// cosourced_provider CAN write work papers
		{models.PersonaCosourcedProvider, rbac.ResourceWorkPaper, rbac.ActionWrite, true, "cosourced can write work papers"},
		// cosourced_provider CANNOT manage engagements
		{models.PersonaCosourcedProvider, rbac.ResourceEngagement, rbac.ActionManage, false, "cosourced cannot manage engagements"},

		// ptwg_member CAN read controls
		{models.PersonaPTWGMember, rbac.ResourceControl, rbac.ActionRead, true, "ptwg can read controls"},
		// ptwg_member CANNOT write engagements
		{models.PersonaPTWGMember, rbac.ResourceEngagement, rbac.ActionWrite, false, "ptwg cannot write engagements"},

		// beta_tester broad read
		{models.PersonaBetaTester, rbac.ResourceEngagement, rbac.ActionRead, true, "beta_tester can read engagements"},
		// beta_tester CANNOT write anything
		{models.PersonaBetaTester, rbac.ResourceEngagement, rbac.ActionWrite, false, "beta_tester cannot write engagements"},
		{models.PersonaBetaTester, rbac.ResourceFinding, rbac.ActionWrite, false, "beta_tester cannot write findings"},

		// unknown persona gets nothing
		{"unknown_persona", rbac.ResourceEngagement, rbac.ActionRead, false, "unknown persona denied"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := checker.Can(tt.persona, tt.resource, tt.action)
			if got != tt.want {
				t.Errorf("Can(%s, %s, %s) = %v, want %v", tt.persona, tt.resource, tt.action, got, tt.want)
			}
		})
	}
}

func TestAllPersonasHavePermissions(t *testing.T) {
	checker := rbac.NewChecker()
	for _, p := range models.AllPersonas {
		perms := checker.Permissions(p)
		if len(perms) == 0 {
			t.Errorf("persona %s has no permissions defined", p)
		}
	}
}

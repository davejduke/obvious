// Package domain_test tests assessment state machine transitions.
package domain

import "testing"

func TestAssessmentStatusTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    AssessmentStatus
		to      AssessmentStatus
		allowed bool
	}{
		// Valid transitions
		{"not_started -> in_progress", AssessmentStatusNotStarted, AssessmentStatusInProgress, true},
		{"in_progress -> assessed", AssessmentStatusInProgress, AssessmentStatusAssessed, true},
		{"in_progress -> remediation", AssessmentStatusInProgress, AssessmentStatusRemediation, true},
		{"assessed -> remediation", AssessmentStatusAssessed, AssessmentStatusRemediation, true},
		{"assessed -> in_progress", AssessmentStatusAssessed, AssessmentStatusInProgress, true},
		{"remediation -> in_progress", AssessmentStatusRemediation, AssessmentStatusInProgress, true},
		{"remediation -> assessed", AssessmentStatusRemediation, AssessmentStatusAssessed, true},

		// Invalid transitions
		{"not_started -> assessed", AssessmentStatusNotStarted, AssessmentStatusAssessed, false},
		{"not_started -> remediation", AssessmentStatusNotStarted, AssessmentStatusRemediation, false},
		{"in_progress -> not_started", AssessmentStatusInProgress, AssessmentStatusNotStarted, false},
		{"assessed -> not_started", AssessmentStatusAssessed, AssessmentStatusNotStarted, false},
		{"remediation -> not_started", AssessmentStatusRemediation, AssessmentStatusNotStarted, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.from.CanTransitionTo(tc.to)
			if got != tc.allowed {
				t.Errorf("CanTransitionTo(%q -> %q) = %v, want %v",
					tc.from, tc.to, got, tc.allowed)
			}
		})
	}
}

func TestNIS2DomainsComplete(t *testing.T) {
	// Verify all 10 NIS 2 Article 21 domains are defined
	expectedDomains := []string{
		"Art. 21(2)(a)",
		"Art. 21(2)(b)",
		"Art. 21(2)(c)",
		"Art. 21(2)(d)",
		"Art. 21(2)(e)",
		"Art. 21(2)(f)",
		"Art. 21(2)(g)",
		"Art. 21(2)(h)",
		"Art. 21(2)(i)",
		"Art. 21(2)(j)",
	}

	if len(NIS2Domains) != 10 {
		t.Errorf("expected 10 NIS2 domains, got %d", len(NIS2Domains))
	}

	domainSet := make(map[string]bool)
	for _, d := range NIS2Domains {
		domainSet[d] = true
	}

	for _, expected := range expectedDomains {
		if !domainSet[expected] {
			t.Errorf("missing NIS2 domain: %s", expected)
		}
		if _, ok := ArticleRefNames[expected]; !ok {
			t.Errorf("ArticleRefNames missing entry for: %s", expected)
		}
	}
}

func TestMappingTypes(t *testing.T) {
	validTypes := []MappingType{
		MappingTypeEquivalent,
		MappingTypeSubset,
		MappingTypeSuperset,
		MappingTypePartial,
		MappingTypeNone,
	}

	for _, mt := range validTypes {
		if mt == "" {
			t.Errorf("empty MappingType constant")
		}
	}

	if len(validTypes) != 5 {
		t.Errorf("expected 5 mapping types, got %d", len(validTypes))
	}
}


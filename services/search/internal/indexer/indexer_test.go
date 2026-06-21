package indexer

import (
	"testing"
)

// TestParseTextArray validates the PostgreSQL text[] parser.
func TestParseTextArray(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"{}", []string{}},
		{"{foo}", []string{"foo"}},
		{"{foo,bar,baz}", []string{"foo", "bar", "baz"}},
		{`{"hello world","foo bar"}`, []string{"hello world", "foo bar"}},
		{"", []string{}},
	}

	for _, tt := range tests {
		got := parseTextArray([]byte(tt.input))
		if len(got) != len(tt.expected) {
			t.Errorf("parseTextArray(%q): got %v, want %v", tt.input, got, tt.expected)
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("parseTextArray(%q)[%d]: got %q, want %q", tt.input, i, got[i], tt.expected[i])
			}
		}
	}
}

// TestIndexNames validates the index name constants.
func TestIndexNames(t *testing.T) {
	if IndexFindings != "aiauditor-findings" {
		t.Errorf("unexpected IndexFindings: %s", IndexFindings)
	}
	if IndexEvidence != "aiauditor-evidence" {
		t.Errorf("unexpected IndexEvidence: %s", IndexEvidence)
	}
	if IndexControls != "aiauditor-controls" {
		t.Errorf("unexpected IndexControls: %s", IndexControls)
	}
}


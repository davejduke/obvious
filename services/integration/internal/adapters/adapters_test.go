package adapters_test

import (
	"context"
	"testing"

	"github.com/davejduke/obvious/services/integration/internal/adapters"
	"github.com/davejduke/obvious/services/integration/internal/connector"
)

func TestSentinelAdapter_MockFetchLogs(t *testing.T) {
	adapter := adapters.NewSentinelAdapter(adapters.SentinelConfig{
		WorkspaceID: "test-workspace-id",
		MockMode:    true,
	})

	logs, err := adapter.FetchLogs(context.Background(), connector.QueryOptions{Limit: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 5 {
		t.Errorf("expected 5 logs, got %d", len(logs))
	}
	for _, l := range logs {
		if l.Source != "sentinel" {
			t.Errorf("expected source=sentinel, got %s", l.Source)
		}
		if l.EventID == "" {
			t.Error("expected non-empty EventID")
		}
		if l.Severity == "" {
			t.Error("expected non-empty Severity")
		}
	}
}

func TestSentinelAdapter_MockHealth(t *testing.T) {
	adapter := adapters.NewSentinelAdapter(adapters.SentinelConfig{
		MockMode: true,
	})
	status := adapter.Health(context.Background())
	if !status.Healthy {
		t.Errorf("expected healthy in mock mode, got: %s", status.Message)
	}
	if status.Connector != "sentinel" {
		t.Errorf("expected connector=sentinel, got %s", status.Connector)
	}
}

func TestSentinelAdapter_DefaultLimit(t *testing.T) {
	adapter := adapters.NewSentinelAdapter(adapters.SentinelConfig{MockMode: true})
	logs, _ := adapter.FetchLogs(context.Background(), connector.QueryOptions{})
	if len(logs) == 0 {
		t.Error("expected default mock logs")
	}
}

func TestSplunkAdapter_MockFetchLogs(t *testing.T) {
	adapter := adapters.NewSplunkAdapter(adapters.SplunkConfig{
		BaseURL:     "http://localhost:8089",
		Token:       "test-token",
		SavedSearch: "NIS2 Security Events",
		MockMode:    true,
	})

	logs, err := adapter.FetchLogs(context.Background(), connector.QueryOptions{Limit: 7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 7 {
		t.Errorf("expected 7 logs, got %d", len(logs))
	}
	for _, l := range logs {
		if l.Source != "splunk" {
			t.Errorf("expected source=splunk, got %s", l.Source)
		}
		if l.EventID == "" {
			t.Error("expected non-empty EventID")
		}
	}
}

func TestSplunkAdapter_MockHealth(t *testing.T) {
	adapter := adapters.NewSplunkAdapter(adapters.SplunkConfig{MockMode: true})
	status := adapter.Health(context.Background())
	if !status.Healthy {
		t.Errorf("expected healthy in mock mode, got: %s", status.Message)
	}
	if status.Connector != "splunk" {
		t.Errorf("expected connector=splunk, got %s", status.Connector)
	}
}

func TestSplunkAdapter_Name(t *testing.T) {
	adapter := adapters.NewSplunkAdapter(adapters.SplunkConfig{})
	if adapter.Name() != "splunk" {
		t.Errorf("expected name=splunk, got %s", adapter.Name())
	}
}

func TestSentinelAdapter_Name(t *testing.T) {
	adapter := adapters.NewSentinelAdapter(adapters.SentinelConfig{})
	if adapter.Name() != "sentinel" {
		t.Errorf("expected name=sentinel, got %s", adapter.Name())
	}
}


// Package connector defines the pluggable connector interface for SIEM integrations.
package connector

import (
	"context"
	"time"
)

// LogEntry represents a single log record returned from a SIEM.
type LogEntry struct {
	Timestamp   time.Time         `json:"timestamp"`
	Source      string            `json:"source"`
	Severity    string            `json:"severity"`
	EventID     string            `json:"event_id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	RawData     map[string]string `json:"raw_data,omitempty"`
}

// HealthStatus represents the health of a connector.
type HealthStatus struct {
	Healthy     bool      `json:"healthy"`
	Connector   string    `json:"connector"`
	LastChecked time.Time `json:"last_checked"`
	Message     string    `json:"message,omitempty"`
}

// QueryOptions holds filtering options for log queries.
type QueryOptions struct {
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Query     string     `json:"query,omitempty"`
}

// Connector is the interface all SIEM adapters must implement.
// New connectors are registered by implementing this interface.
type Connector interface {
	// Name returns the unique connector identifier.
	Name() string

	// FetchLogs retrieves log entries matching the given options.
	FetchLogs(ctx context.Context, opts QueryOptions) ([]LogEntry, error)

	// Health reports the connectivity status of this connector.
	Health(ctx context.Context) HealthStatus
}

// Registry holds all registered connectors.
type Registry struct {
	connectors map[string]Connector
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{connectors: make(map[string]Connector)}
}

// Register adds a connector to the registry.
func (r *Registry) Register(c Connector) {
	r.connectors[c.Name()] = c
}

// Get returns a connector by name.
func (r *Registry) Get(name string) (Connector, bool) {
	c, ok := r.connectors[name]
	return c, ok
}

// List returns all registered connector names.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.connectors))
	for n := range r.connectors {
		names = append(names, n)
	}
	return names
}


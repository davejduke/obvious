// Package connector defines the Connector SDK for AIAUDITOR third-party integrations.
//
// The SDK provides:
//   - A clean Connector interface: Connect, Sync, Transform, Healthcheck
//   - Auth abstraction (API key, OAuth 2.0 client credentials, certificate)
//   - Token bucket rate limiter with configurable burst and refill
//   - Circuit breaker (closed/open/half-open) protecting against cascading failures
//   - Retry with exponential backoff
//   - Test harness with a mock HTTP server, request recording, and response fixtures
//
// Quick start:
//
//	import sdk "github.com/davejduke/obvious/sdk/connector"
//
//	type MyConnector struct{ ... }
//	func (c *MyConnector) Connect(ctx context.Context) error       { ... }
//	func (c *MyConnector) Sync(ctx context.Context, opts sdk.SyncOptions) (<-chan sdk.SyncRecord, error) { ... }
//	func (c *MyConnector) Transform(ctx context.Context, req sdk.TransformRequest) (sdk.TransformResult, error) { ... }
//	func (c *MyConnector) Healthcheck(ctx context.Context) sdk.HealthStatus { ... }
package connector

import (
	"context"
	"time"
)

// ---------------------------------------------------------------------------
// Core types
// ---------------------------------------------------------------------------

// SyncOptions controls what data is fetched during a Sync call.
type SyncOptions struct {
	// Since filters records modified after this timestamp.
	// A zero value returns all records.
	Since time.Time

	// Limit caps the number of records returned. 0 means no limit.
	Limit int

	// Cursor is an opaque pagination token from a previous Sync response.
	Cursor string

	// Filter is an optional provider-specific query expression.
	Filter string
}

// SyncRecord represents a single data record returned from a remote system.
type SyncRecord struct {
	// ID is the external system's unique identifier.
	ID string `json:"id"`

	// Source identifies the connector that produced this record.
	Source string `json:"source"`

	// Type is the record category (e.g. "finding", "vulnerability", "user").
	Type string `json:"type"`

	// Timestamp is when this record was last modified in the source system.
	Timestamp time.Time `json:"timestamp"`

	// Payload holds the raw, provider-specific record data.
	Payload map[string]any `json:"payload"`

	// NextCursor is the pagination token to pass in the next SyncOptions.Cursor.
	// Empty string means no more pages.
	NextCursor string `json:"next_cursor,omitempty"`
}

// TransformRequest holds input data to be normalised by the connector.
type TransformRequest struct {
	// RecordType names the target AIAUDITOR schema (e.g. "finding", "control").
	RecordType string `json:"record_type"`

	// Raw is the provider-native payload to transform.
	Raw map[string]any `json:"raw"`
}

// TransformResult holds the normalised output produced by Transform.
type TransformResult struct {
	// RecordType mirrors TransformRequest.RecordType.
	RecordType string `json:"record_type"`

	// Normalised is the AIAUDITOR-canonical representation of the record.
	Normalised map[string]any `json:"normalised"`

	// Warnings lists non-fatal issues encountered during transformation.
	Warnings []string `json:"warnings,omitempty"`
}

// HealthStatus reports the operational health of a connector instance.
type HealthStatus struct {
	// Healthy is true when the connector can reach its data source.
	Healthy bool `json:"healthy"`

	// Connector is the connector's registered name.
	Connector string `json:"connector"`

	// LastChecked is the UTC time the check was performed.
	LastChecked time.Time `json:"last_checked"`

	// Latency is the round-trip time to the data source (0 if not measured).
	Latency time.Duration `json:"latency_ms"`

	// Message carries additional human-readable detail.
	Message string `json:"message,omitempty"`
}

// ---------------------------------------------------------------------------
// Connector interface
// ---------------------------------------------------------------------------

// Connector is the interface every AIAUDITOR integration must implement.
//
// Lifecycle:
//
//	Connect()     → called once on startup; establish session, verify credentials
//	Sync()        → called periodically to pull new records (streaming channel)
//	Transform()   → called per record to normalise raw data to AIAUDITOR schema
//	Healthcheck() → called on demand or by the platform health probe
type Connector interface {
	// Connect initialises the connection to the remote system.
	// Implementations must validate credentials and surface connection errors here
	// rather than deferring them until Sync.
	Connect(ctx context.Context) error

	// Sync returns a channel of SyncRecords fetched according to opts.
	// The channel is closed when all records have been sent or the context is
	// cancelled. Implementations must not block indefinitely; honour ctx.
	Sync(ctx context.Context, opts SyncOptions) (<-chan SyncRecord, error)

	// Transform converts a raw provider record into the AIAUDITOR canonical schema.
	// It is called per-record and must be stateless.
	Transform(ctx context.Context, req TransformRequest) (TransformResult, error)

	// Healthcheck reports whether the connector can reach its data source.
	// Implementations should perform a lightweight probe (e.g. API ping),
	// not a full Sync.
	Healthcheck(ctx context.Context) HealthStatus
}

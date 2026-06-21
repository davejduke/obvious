// Package logging provides structured JSON logging for AIAUDITOR services.
// Log format matches tech spec §10.1:
//
//	{timestamp, level, service, trace_id, span_id, org_id, engagement_id, event, metadata}
//
// Usage:
//
//	logger := logging.New("my-service")
//	logger.Info(ctx, "user.login", map[string]any{"user_id": id})
package logging

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"time"
)

// Level is a log severity level.
type Level string

const (
	LevelInfo     Level = "INFO"
	LevelWarn     Level = "WARN"
	LevelError    Level = "ERROR"
	LevelCritical Level = "CRITICAL"
)

// contextKey is an unexported type for context keys owned by this package.
type contextKey string

const (
	keyTraceID      contextKey = "trace_id"
	keySpanID       contextKey = "span_id"
	keyOrgID        contextKey = "org_id"
	keyEngagementID contextKey = "engagement_id"
	keyRequestID    contextKey = "request_id"
)

// Entry is a single structured log entry matching tech spec §10.1.
type Entry struct {
	Timestamp    string         `json:"timestamp"`
	Level        Level          `json:"level"`
	Service      string         `json:"service"`
	TraceID      string         `json:"trace_id,omitempty"`
	SpanID       string         `json:"span_id,omitempty"`
	OrgID        string         `json:"org_id,omitempty"`
	EngagementID string         `json:"engagement_id,omitempty"`
	Event        string         `json:"event"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// Logger emits structured JSON log entries to an io.Writer (default: os.Stdout).
type Logger struct {
	service string
	out     io.Writer
}

// New creates a Logger for the named service.
func New(service string) *Logger {
	return &Logger{service: service, out: os.Stdout}
}

// WithWriter returns a new Logger that writes to w. Useful for testing.
func (l *Logger) WithWriter(w io.Writer) *Logger {
	return &Logger{service: l.service, out: w}
}

// emit serialises and writes one log entry.
func (l *Logger) emit(ctx context.Context, level Level, event string, metadata map[string]any) {
	entry := Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level,
		Service:   l.service,
		Event:     event,
		Metadata:  metadata,
	}
	if ctx != nil {
		if v, ok := ctx.Value(keyTraceID).(string); ok {
			entry.TraceID = v
		}
		if v, ok := ctx.Value(keySpanID).(string); ok {
			entry.SpanID = v
		}
		if v, ok := ctx.Value(keyOrgID).(string); ok {
			entry.OrgID = v
		}
		if v, ok := ctx.Value(keyEngagementID).(string); ok {
			entry.EngagementID = v
		}
	}
	b, err := json.Marshal(entry)
	if err != nil {
		return
	}
	b = append(b, '\n')
	_, _ = l.out.Write(b)
}

// Info logs at INFO level.
func (l *Logger) Info(ctx context.Context, event string, metadata map[string]any) {
	l.emit(ctx, LevelInfo, event, metadata)
}

// Warn logs at WARN level.
func (l *Logger) Warn(ctx context.Context, event string, metadata map[string]any) {
	l.emit(ctx, LevelWarn, event, metadata)
}

// Error logs at ERROR level.
func (l *Logger) Error(ctx context.Context, event string, metadata map[string]any) {
	l.emit(ctx, LevelError, event, metadata)
}

// Critical logs at CRITICAL level.
func (l *Logger) Critical(ctx context.Context, event string, metadata map[string]any) {
	l.emit(ctx, LevelCritical, event, metadata)
}

// --- Context helpers --------------------------------------------------------

// WithTraceID stores a trace ID in the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, keyTraceID, traceID)
}

// WithSpanID stores a span ID in the context.
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, keySpanID, spanID)
}

// WithOrgID stores an org ID in the context.
func WithOrgID(ctx context.Context, orgID string) context.Context {
	return context.WithValue(ctx, keyOrgID, orgID)
}

// WithEngagementID stores an engagement ID in the context.
func WithEngagementID(ctx context.Context, engagementID string) context.Context {
	return context.WithValue(ctx, keyEngagementID, engagementID)
}

// TraceIDFromContext extracts the trace ID from ctx, returning "" if absent.
func TraceIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(keyTraceID).(string)
	return v
}

// SpanIDFromContext extracts the span ID from ctx, returning "" if absent.
func SpanIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(keySpanID).(string)
	return v
}

// OrgIDFromContext extracts the org ID from ctx.
func OrgIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(keyOrgID).(string)
	return v
}

// EngagementIDFromContext extracts the engagement ID from ctx.
func EngagementIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(keyEngagementID).(string)
	return v
}

// RequestIDFromContext extracts the request ID from ctx.
func RequestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(keyRequestID).(string)
	return v
}


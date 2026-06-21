package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestLogFormat verifies the JSON output matches tech spec \u00a710.1 field names.
func TestLogFormat(t *testing.T) {
	var buf bytes.Buffer
	l := New("test-service").WithWriter(&buf)
	ctx := WithTraceID(context.Background(), "4bf92f3577b34da6a3ce929d0e0e4736")
	ctx = WithSpanID(ctx, "00f067aa0ba902b7")
	ctx = WithOrgID(ctx, "org-1")
	ctx = WithEngagementID(ctx, "eng-1")

	l.Info(ctx, "user.login", map[string]any{"user_id": "u-1"})

	var e Entry
	if err := json.NewDecoder(&buf).Decode(&e); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if e.Level != LevelInfo {
		t.Errorf("level: got %q want %q", e.Level, LevelInfo)
	}
	if e.Service != "test-service" {
		t.Errorf("service: got %q want %q", e.Service, "test-service")
	}
	if e.TraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("trace_id: got %q", e.TraceID)
	}
	if e.SpanID != "00f067aa0ba902b7" {
		t.Errorf("span_id: got %q", e.SpanID)
	}
	if e.OrgID != "org-1" {
		t.Errorf("org_id: got %q", e.OrgID)
	}
	if e.EngagementID != "eng-1" {
		t.Errorf("engagement_id: got %q", e.EngagementID)
	}
	if e.Event != "user.login" {
		t.Errorf("event: got %q", e.Event)
	}
	if e.Timestamp == "" {
		t.Error("timestamp must not be empty")
	}
}

// TestLogLevels verifies each log method emits the correct level string.
func TestLogLevels(t *testing.T) {
	tests := []struct {
		name    string
		fn      func(*Logger, context.Context, string, map[string]any)
		wantLvl Level
	}{
		{"info", (*Logger).Info, LevelInfo},
		{"warn", (*Logger).Warn, LevelWarn},
		{"error", (*Logger).Error, LevelError},
		{"critical", (*Logger).Critical, LevelCritical},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := New("svc").WithWriter(&buf)
			tt.fn(l, context.Background(), "evt", nil)
			var e Entry
			if err := json.NewDecoder(&buf).Decode(&e); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if e.Level != tt.wantLvl {
				t.Errorf("level: got %q want %q", e.Level, tt.wantLvl)
			}
		})
	}
}

// TestParseTraceparent covers valid and invalid header values.
func TestParseTraceparent(t *testing.T) {
	tests := []struct {
		header  string
		wantOK  bool
		traceID string
		spanID  string
	}{
		{
			header:  "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			wantOK:  true,
			traceID: "4bf92f3577b34da6a3ce929d0e0e4736",
			spanID:  "00f067aa0ba902b7",
		},
		{header: "invalid", wantOK: false},
		{header: "", wantOK: false},
		{header: "01-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", wantOK: false}, // wrong version
	}
	for _, tt := range tests {
		tc, ok := ParseTraceparent(tt.header)
		if ok != tt.wantOK {
			t.Errorf("ParseTraceparent(%q) ok=%v want %v", tt.header, ok, tt.wantOK)
			continue
		}
		if tt.wantOK {
			if tc.TraceID != tt.traceID {
				t.Errorf("trace_id: got %q want %q", tc.TraceID, tt.traceID)
			}
			if tc.SpanID != tt.spanID {
				t.Errorf("span_id: got %q want %q", tc.SpanID, tt.spanID)
			}
			if !tc.Sampled {
				t.Error("sampled should be true for flags=01")
			}
		}
	}
}

// TestTraceContextMiddleware verifies propagation of existing and new trace IDs.
func TestTraceContextMiddleware(t *testing.T) {
	t.Run("propagates existing traceparent", func(t *testing.T) {
		var capturedTraceID, capturedSpanID string
		h := TraceContextMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedTraceID = TraceIDFromContext(r.Context())
			capturedSpanID = SpanIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set(HeaderTraceparent, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
		h.ServeHTTP(httptest.NewRecorder(), req)
		if capturedTraceID != "4bf92f3577b34da6a3ce929d0e0e4736" {
			t.Errorf("trace_id: got %q", capturedTraceID)
		}
		if capturedSpanID != "00f067aa0ba902b7" {
			t.Errorf("span_id: got %q", capturedSpanID)
		}
	})

	t.Run("generates new trace context", func(t *testing.T) {
		var capturedTraceID, capturedSpanID string
		h := TraceContextMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedTraceID = TraceIDFromContext(r.Context())
			capturedSpanID = SpanIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest("GET", "/", nil)
		h.ServeHTTP(httptest.NewRecorder(), req)
		if len(capturedTraceID) != 32 {
			t.Errorf("generated trace_id len: got %d want 32", len(capturedTraceID))
		}
		if len(capturedSpanID) != 16 {
			t.Errorf("generated span_id len: got %d want 16", len(capturedSpanID))
		}
	})
}

// TestRequestIDMiddleware verifies request ID generation and preservation.
func TestRequestIDMiddleware(t *testing.T) {
	t.Run("generates request ID", func(t *testing.T) {
		var capturedID string
		h := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedID = RequestIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		if capturedID == "" {
			t.Error("request ID must be generated")
		}
		if rec.Header().Get(HeaderRequestID) == "" {
			t.Error("X-Request-ID must appear on response")
		}
	})

	t.Run("preserves existing request ID", func(t *testing.T) {
		var capturedID string
		h := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedID = RequestIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set(HeaderRequestID, "my-req-id")
		h.ServeHTTP(httptest.NewRecorder(), req)
		if capturedID != "my-req-id" {
			t.Errorf("request ID: got %q want %q", capturedID, "my-req-id")
		}
	})
}


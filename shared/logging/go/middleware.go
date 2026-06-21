package logging

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"strings"
)

const (
	// HeaderTraceparent is the W3C Trace Context traceparent header.
	HeaderTraceparent = "traceparent"
	// HeaderTracestate is the W3C Trace Context tracestate header.
	HeaderTracestate = "tracestate"
	// HeaderRequestID carries a unique request ID across services.
	HeaderRequestID = "X-Request-ID"
)

// TraceContext holds values parsed from a W3C traceparent header.
type TraceContext struct {
	TraceID string
	SpanID  string
	Flags   string
	Sampled bool
}

// ParseTraceparent parses a W3C traceparent header.
// Format: 00-<trace-id:32hex>-<parent-id:16hex>-<flags:2hex>
func ParseTraceparent(header string) (*TraceContext, bool) {
	parts := strings.Split(header, "-")
	if len(parts) != 4 {
		return nil, false
	}
	version, traceID, parentID, flags := parts[0], parts[1], parts[2], parts[3]
	if version != "00" || len(traceID) != 32 || len(parentID) != 16 || len(flags) != 2 {
		return nil, false
	}
	return &TraceContext{
		TraceID: traceID,
		SpanID:  parentID,
		Flags:   flags,
		Sampled: flags == "01",
	}, true
}

// FormatTraceparent formats a W3C traceparent header value.
func FormatTraceparent(traceID, spanID string) string {
	return fmt.Sprintf("00-%s-%s-01", traceID, spanID)
}

// newTraceID generates a random 128-bit trace ID as 32 lowercase hex chars.
func newTraceID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%x", b)
}

// newSpanID generates a random 64-bit span ID as 16 lowercase hex chars.
func newSpanID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%x", b)
}

// newRequestID generates a UUID v4-style request ID.
func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// TraceContextMiddleware extracts or generates W3C Trace Context (traceparent)
// and injects trace_id and span_id into the request context.
func TraceContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		traceparent := r.Header.Get(HeaderTraceparent)
		if tc, ok := ParseTraceparent(traceparent); ok {
			ctx = WithTraceID(ctx, tc.TraceID)
			ctx = WithSpanID(ctx, tc.SpanID)
		} else {
			traceID := newTraceID()
			spanID := newSpanID()
			ctx = WithTraceID(ctx, traceID)
			ctx = WithSpanID(ctx, spanID)
			r = r.Clone(ctx)
			r.Header.Set(HeaderTraceparent, FormatTraceparent(traceID, spanID))
		}
		// Propagate tracestate unchanged.
		if ts := r.Header.Get(HeaderTracestate); ts != "" {
			w.Header().Set(HeaderTracestate, ts)
		}
		// Echo traceparent on response for downstream correlation.
		w.Header().Set(HeaderTraceparent, r.Header.Get(HeaderTraceparent))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDMiddleware ensures every request has an X-Request-ID, generating
// one if absent. The ID is echoed on the response and stored in the context.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get(HeaderRequestID)
		if reqID == "" {
			reqID = newRequestID()
			r = r.Clone(r.Context())
			r.Header.Set(HeaderRequestID, reqID)
		}
		w.Header().Set(HeaderRequestID, reqID)
		ctx := context.WithValue(r.Context(), keyRequestID, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}


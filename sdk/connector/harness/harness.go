// Package harness provides a test harness for the Connector SDK.
//
// The harness spins up an in-process httptest.Server and lets tests:
//   - Register response fixtures (status + body) per path
//   - Record all inbound requests for assertion
//   - Simulate errors, delays, and sequences of responses
//
// Usage:
//
//	h := harness.New(t)
//	h.AddResponse("/data", http.StatusOK, `{"items":[]}`)
//	h.AddResponse("/health", http.StatusOK, `{"status":"ok"}`)
//
//	connector := myconnector.New(myconnector.Config{BaseURL: h.URL()})
//	err := connector.Connect(ctx)
//	require.NoError(t, err)
//
//	reqs := h.Requests("/data")
//	require.Len(t, reqs, 1)
package harness

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// Response is a pre-canned HTTP response returned for a matching request.
type Response struct {
	// StatusCode defaults to 200 if zero.
	StatusCode int

	// Body is the response body bytes.
	Body []byte

	// Headers are additional headers to include in the response.
	Headers map[string]string

	// DelayMS is an optional artificial delay in milliseconds (for timeout tests).
	// Use carefully — it slows test execution.
	DelayMS int
}

// RecordedRequest captures a request observed by the harness.
type RecordedRequest struct {
	// Method is the HTTP method (GET, POST, etc.).
	Method string

	// Path is the URL path (without query string).
	Path string

	// Query is the raw query string.
	Query string

	// Headers are the request headers.
	Headers http.Header

	// Body is the full request body (read and buffered).
	Body []byte
}

// Harness wraps an httptest.Server with response queuing and request recording.
type Harness struct {
	t       testing.TB
	srv     *httptest.Server
	mu      sync.Mutex
	queues  map[string][]Response // path → FIFO queue of responses
	records map[string][]RecordedRequest
}

// New creates a new Harness backed by an httptest.Server.
// The server is automatically closed when the test ends.
func New(t testing.TB) *Harness {
	t.Helper()
	h := &Harness{
		t:       t,
		queues:  make(map[string][]Response),
		records: make(map[string][]RecordedRequest),
	}
	h.srv = httptest.NewServer(http.HandlerFunc(h.handle))
	t.Cleanup(h.srv.Close)
	return h
}

// NewTLS creates a Harness using an httptest.NewTLSServer.
// Use this to test TLS-enabled connectors.
func NewTLS(t testing.TB) *Harness {
	t.Helper()
	h := &Harness{
		t:       t,
		queues:  make(map[string][]Response),
		records: make(map[string][]RecordedRequest),
	}
	h.srv = httptest.NewTLSServer(http.HandlerFunc(h.handle))
	t.Cleanup(h.srv.Close)
	return h
}

// URL returns the base URL of the test server (e.g. "http://127.0.0.1:54321").
func (h *Harness) URL() string {
	return h.srv.URL
}

// Client returns the httptest.Server's client (pre-configured for TLS when
// created with NewTLS).
func (h *Harness) Client() *http.Client {
	return h.srv.Client()
}

// AddResponse enqueues a response for the given path.
// Responses are dequeued FIFO; if the queue empties, the last response is
// reused (sticky). This lets tests register a single response and have it
// returned for all matching requests.
func (h *Harness) AddResponse(path string, statusCode int, body string) {
	h.AddResponseRaw(path, Response{StatusCode: statusCode, Body: []byte(body)})
}

// AddJSONResponse enqueues a JSON-serialised response for the given path.
func (h *Harness) AddJSONResponse(path string, statusCode int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		h.t.Fatalf("harness: AddJSONResponse: marshal: %v", err)
	}
	h.AddResponseRaw(path, Response{
		StatusCode: statusCode,
		Body:       b,
		Headers:    map[string]string{"Content-Type": "application/json"},
	})
}

// AddResponseRaw enqueues a fully-specified Response for the given path.
func (h *Harness) AddResponseRaw(path string, resp Response) {
	if resp.StatusCode == 0 {
		resp.StatusCode = http.StatusOK
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.queues[path] = append(h.queues[path], resp)
}

// AddErrorResponse enqueues an error response (default 500) for path.
func (h *Harness) AddErrorResponse(path string, statusCode int, message string) {
	body := fmt.Sprintf(`{"error":%q}`, message)
	h.AddResponseRaw(path, Response{StatusCode: statusCode, Body: []byte(body),
		Headers: map[string]string{"Content-Type": "application/json"}})
}

// Requests returns all recorded requests for the given path, in order.
func (h *Harness) Requests(path string) []RecordedRequest {
	h.mu.Lock()
	defer h.mu.Unlock()
	recs := h.records[path]
	out := make([]RecordedRequest, len(recs))
	copy(out, recs)
	return out
}

// AllRequests returns every recorded request, flattened in arrival order.
func (h *Harness) AllRequests() []RecordedRequest {
	h.mu.Lock()
	defer h.mu.Unlock()
	var all []RecordedRequest
	for _, recs := range h.records {
		all = append(all, recs...)
	}
	return all
}

// RequestCount returns the total number of requests received for path.
func (h *Harness) RequestCount(path string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.records[path])
}

// Reset clears all queued responses and recorded requests.
func (h *Harness) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.queues = make(map[string][]Response)
	h.records = make(map[string][]RecordedRequest)
}

// handle is the httptest.Server handler.
func (h *Harness) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()

	h.mu.Lock()
	h.records[r.URL.Path] = append(h.records[r.URL.Path], RecordedRequest{
		Method:  r.Method,
		Path:    r.URL.Path,
		Query:   r.URL.RawQuery,
		Headers: r.Header.Clone(),
		Body:    body,
	})

	resp := h.dequeue(r.URL.Path)
	h.mu.Unlock()

	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(resp.Body)
}

// dequeue returns the next response for path.
// MUST be called with h.mu held.
func (h *Harness) dequeue(path string) Response {
	q := h.queues[path]
	if len(q) == 0 {
		// No fixture registered — return a generic 404.
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       []byte(fmt.Sprintf(`{"error":"no fixture registered for %s"}`, path)),
			Headers:    map[string]string{"Content-Type": "application/json"},
		}
	}
	resp := q[0]
	if len(q) > 1 {
		h.queues[path] = q[1:]
	}
	// else: leave the last response in place (sticky).
	return resp
}

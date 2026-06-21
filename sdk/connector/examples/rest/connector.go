// Package rest provides a generic REST connector that demonstrates the full
// Connector SDK lifecycle: Connect, Sync, Transform, Healthcheck, auth, rate
// limiting, retry, and circuit breaker.
//
// This implementation is intentionally simple so it can be used as a starting
// point for real connectors. Copy this package and adapt the SyncRecord mapping
// in Transform to match your target AIAUDITOR schema.
package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	connector "github.com/davejduke/obvious/sdk/connector"
	"github.com/davejduke/obvious/sdk/connector/auth"
	"github.com/davejduke/obvious/sdk/connector/circuit"
	"github.com/davejduke/obvious/sdk/connector/ratelimit"
	"github.com/davejduke/obvious/sdk/connector/retry"
)

// Config holds all settings for the generic REST connector.
type Config struct {
	// Name is the unique connector identifier used in the registry.
	Name string

	// BaseURL is the root URL of the remote REST API.
	// Example: "https://api.example.com/v1"
	BaseURL string

	// DataPath is the API path that returns a list of records.
	// Example: "/findings"
	DataPath string

	// HealthPath is the API path used for the Healthcheck probe.
	// Example: "/health"
	HealthPath string

	// Auth is the authentication provider. Required.
	Auth auth.AuthProvider

	// RateLimit configures token bucket rate limiting.
	// Zero values use ratelimit.DefaultConfig().
	RateLimit ratelimit.Config

	// CircuitBreaker configures the circuit breaker.
	// Zero values use circuit.DefaultConfig().
	CircuitBreaker circuit.Config

	// Retry configures exponential backoff.
	// Zero values use retry.DefaultConfig().
	Retry retry.Config

	// RecordType is the AIAUDITOR schema type to populate in Transform.
	// Example: "finding"
	RecordType string

	// HTTPClient is the HTTP client to use. Defaults to a 30s timeout client.
	HTTPClient *http.Client
}

// RESTConnector is a generic REST API connector that implements connector.Connector.
type RESTConnector struct {
	cfg     Config
	client  *http.Client
	bucket  *ratelimit.TokenBucket
	breaker *circuit.Breaker
}

// New returns a RESTConnector ready to be registered with the SDK.
func New(cfg Config) (*RESTConnector, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("rest connector: Name is required")
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("rest connector: BaseURL is required")
	}
	if cfg.Auth == nil {
		return nil, fmt.Errorf("rest connector: Auth is required")
	}

	// Apply defaults for zero configs.
	if cfg.RateLimit.Rate == 0 {
		cfg.RateLimit = ratelimit.DefaultConfig()
	}
	if cfg.CircuitBreaker.FailureThreshold == 0 {
		cfg.CircuitBreaker = circuit.DefaultConfig()
	}
	if cfg.Retry.MaxAttempts == 0 {
		cfg.Retry = retry.DefaultConfig()
	}
	if cfg.DataPath == "" {
		cfg.DataPath = "/data"
	}
	if cfg.HealthPath == "" {
		cfg.HealthPath = "/health"
	}
	if cfg.RecordType == "" {
		cfg.RecordType = "record"
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	// Configure mTLS if the auth provider is CertificateAuth.
	if certAuth, ok := cfg.Auth.(*auth.CertificateAuth); ok {
		certAuth.ConfigureClient(httpClient)
	}

	bucket, err := ratelimit.New(cfg.RateLimit)
	if err != nil {
		return nil, fmt.Errorf("rest connector: rate limit config: %w", err)
	}

	return &RESTConnector{
		cfg:     cfg,
		client:  httpClient,
		bucket:  bucket,
		breaker: circuit.New(cfg.CircuitBreaker),
	}, nil
}

// Connect verifies that the remote API is reachable and credentials are valid.
// It hits the health endpoint and fails fast on auth errors.
func (c *RESTConnector) Connect(ctx context.Context) error {
	status := c.Healthcheck(ctx)
	if !status.Healthy {
		return fmt.Errorf("rest connector %s: connect failed: %s", c.cfg.Name, status.Message)
	}
	return nil
}

// Sync fetches records from the remote API and sends them on the returned
// channel. The channel is closed when all pages have been sent or ctx is
// cancelled. Pagination is controlled by opts.Cursor.
func (c *RESTConnector) Sync(ctx context.Context, opts connector.SyncOptions) (<-chan connector.SyncRecord, error) {
	ch := make(chan connector.SyncRecord, 64)

	go func() {
		defer close(ch)

		cursor := opts.Cursor
		pagesSent := 0

		for {
			if err := ctx.Err(); err != nil {
				return
			}

			// Rate limit — block until a token is available.
			if err := c.bucket.Acquire(ctx); err != nil {
				return
			}

			var page apiPage
			fetchErr := c.breaker.Do(func() error {
				return retry.Do(ctx, c.cfg.Retry, func() error {
					var err error
					page, err = c.fetchPage(ctx, cursor, opts)
					return err
				})
			})
			if fetchErr != nil {
				// Log and stop; in a real connector you might send an error record.
				_ = fetchErr
				return
			}

			for _, item := range page.Items {
				rec := connector.SyncRecord{
					ID:        item.ID,
					Source:    c.cfg.Name,
					Type:      c.cfg.RecordType,
					Timestamp: item.UpdatedAt,
					Payload:   item.Raw,
				}
				select {
				case ch <- rec:
				case <-ctx.Done():
					return
				}
			}

			pagesSent++
			if opts.Limit > 0 && pagesSent*len(page.Items) >= opts.Limit {
				return
			}
			if page.NextCursor == "" {
				return
			}
			cursor = page.NextCursor
		}
	}()

	return ch, nil
}

// Transform maps a raw provider record to the AIAUDITOR canonical schema.
// Adapt the field mapping to your target system's data model.
func (c *RESTConnector) Transform(_ context.Context, req connector.TransformRequest) (connector.TransformResult, error) {
	normalised := make(map[string]any)
	var warnings []string

	// Copy all raw fields as the baseline.
	for k, v := range req.Raw {
		normalised[k] = v
	}

	// Promote common fields to canonical positions.
	if id, ok := req.Raw["id"].(string); ok {
		normalised["external_id"] = id
	} else {
		warnings = append(warnings, "missing 'id' field in raw record")
	}

	if title, ok := req.Raw["title"].(string); ok {
		normalised["title"] = title
	} else if name, ok := req.Raw["name"].(string); ok {
		normalised["title"] = name
	}

	if severity, ok := req.Raw["severity"].(string); ok {
		normalised["severity"] = normaliseSeverity(severity)
	}

	normalised["source"] = c.cfg.Name
	normalised["record_type"] = req.RecordType

	return connector.TransformResult{
		RecordType: req.RecordType,
		Normalised: normalised,
		Warnings:   warnings,
	}, nil
}

// Healthcheck probes the health endpoint of the remote API.
func (c *RESTConnector) Healthcheck(ctx context.Context) connector.HealthStatus {
	status := connector.HealthStatus{
		Connector:   c.cfg.Name,
		LastChecked: time.Now().UTC(),
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.cfg.BaseURL+c.cfg.HealthPath, nil)
	if err != nil {
		status.Message = "build request: " + err.Error()
		return status
	}
	if err := c.cfg.Auth.AddAuth(ctx, req); err != nil {
		status.Message = "add auth: " + err.Error()
		return status
	}

	resp, err := c.client.Do(req)
	status.Latency = time.Since(start)
	if err != nil {
		status.Message = "request: " + err.Error()
		return status
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	status.Healthy = resp.StatusCode >= 200 && resp.StatusCode < 300
	status.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)

	// Annotate with circuit state.
	if state := c.breaker.State(); state != circuit.StateClosed {
		status.Message += fmt.Sprintf("; circuit %s", state)
	}
	return status
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// apiPage is the generic paginated response shape assumed by this connector.
type apiPage struct {
	Items      []apiItem `json:"items"`
	NextCursor string    `json:"next_cursor,omitempty"`
	Total      int       `json:"total,omitempty"`
}

// apiItem is a single record in an apiPage.
type apiItem struct {
	ID        string         `json:"id"`
	UpdatedAt time.Time      `json:"updated_at"`
	Raw       map[string]any `json:"-"`
}

// UnmarshalJSON captures the raw map alongside the typed fields.
func (i *apiItem) UnmarshalJSON(data []byte) error {
	// Unmarshal to a raw map first.
	if err := json.Unmarshal(data, &i.Raw); err != nil {
		return err
	}
	// Then populate the typed fields.
	type alias apiItem
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	i.ID = a.ID
	i.UpdatedAt = a.UpdatedAt
	return nil
}

// fetchPage calls the data endpoint and unmarshals one page.
func (c *RESTConnector) fetchPage(ctx context.Context, cursor string, opts connector.SyncOptions) (apiPage, error) {
	url := c.cfg.BaseURL + c.cfg.DataPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return apiPage{}, fmt.Errorf("build request: %w", err)
	}

	// Append pagination and filter params.
	q := req.URL.Query()
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if !opts.Since.IsZero() {
		q.Set("since", opts.Since.UTC().Format(time.RFC3339))
	}
	if opts.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", opts.Limit))
	}
	if opts.Filter != "" {
		q.Set("filter", opts.Filter)
	}
	req.URL.RawQuery = q.Encode()

	if err := c.cfg.Auth.AddAuth(ctx, req); err != nil {
		return apiPage{}, fmt.Errorf("add auth: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return apiPage{}, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return apiPage{}, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return apiPage{}, retry.Permanent(fmt.Errorf("auth error: HTTP %d", resp.StatusCode))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiPage{}, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var page apiPage
	if err := json.Unmarshal(body, &page); err != nil {
		return apiPage{}, fmt.Errorf("parse response: %w", err)
	}
	return page, nil
}

// normaliseSeverity maps provider-specific severity strings to the canonical set.
func normaliseSeverity(raw string) string {
	switch raw {
	case "CRITICAL", "critical", "crit":
		return "critical"
	case "HIGH", "high":
		return "high"
	case "MEDIUM", "medium", "med":
		return "medium"
	case "LOW", "low":
		return "low"
	case "INFO", "info", "informational":
		return "info"
	default:
		return raw
	}
}

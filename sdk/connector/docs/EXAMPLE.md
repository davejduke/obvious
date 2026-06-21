# Example Connector Walkthrough

This guide walks through building a production-ready connector for a hypothetical
"Acme Vulnerability Scanner" REST API, using every component of the SDK.

## 1. Define Your Config

```go
package acme

type Config struct {
    BaseURL string
    APIKey  string
}
```

## 2. Create the Connector Struct

```go
import (
    sdk    "github.com/davejduke/obvious/sdk/connector"
    "github.com/davejduke/obvious/sdk/connector/auth"
    "github.com/davejduke/obvious/sdk/connector/circuit"
    "github.com/davejduke/obvious/sdk/connector/ratelimit"
    "github.com/davejduke/obvious/sdk/connector/retry"
)

type AcmeConnector struct {
    cfg     Config
    client  *http.Client
    bucket  *ratelimit.TokenBucket
    breaker *circuit.Breaker
    auth    auth.AuthProvider
}

func New(cfg Config) (*AcmeConnector, error) {
    bucket, err := ratelimit.New(ratelimit.Config{Rate: 5, Burst: 10})
    if err != nil {
        return nil, err
    }
    return &AcmeConnector{
        cfg:     cfg,
        client:  &http.Client{Timeout: 30 * time.Second},
        bucket:  bucket,
        breaker: circuit.New(circuit.DefaultConfig()),
        auth:    auth.NewAPIKeyAuth(auth.APIKeyConfig{Key: cfg.APIKey}),
    }, nil
}
```

## 3. Implement Connect

```go
func (c *AcmeConnector) Connect(ctx context.Context) error {
    s := c.Healthcheck(ctx)
    if !s.Healthy {
        return fmt.Errorf("acme: connect: %s", s.Message)
    }
    return nil
}
```

## 4. Implement Sync

```go
func (c *AcmeConnector) Sync(ctx context.Context, opts sdk.SyncOptions) (<-chan sdk.SyncRecord, error) {
    ch := make(chan sdk.SyncRecord, 32)
    go func() {
        defer close(ch)
        cursor := opts.Cursor
        for {
            if err := c.bucket.Acquire(ctx); err != nil { return }
            var page Page
            err := c.breaker.Do(func() error {
                return retry.Do(ctx, retry.DefaultConfig(), func() error {
                    var e error
                    page, e = c.fetchPage(ctx, cursor)
                    return e
                })
            })
            if err != nil { return }
            for _, item := range page.Items {
                ch <- sdk.SyncRecord{
                    ID: item.ID, Source: "acme", Type: "vulnerability",
                    Timestamp: item.UpdatedAt, Payload: item.Raw,
                }
            }
            cursor = page.NextCursor
            if cursor == "" { return }
        }
    }()
    return ch, nil
}
```

## 5. Implement Transform

```go
func (c *AcmeConnector) Transform(_ context.Context, req sdk.TransformRequest) (sdk.TransformResult, error) {
    var warnings []string
    out := make(map[string]any)

    out["external_id"] = req.Raw["id"]
    out["title"]       = req.Raw["vulnerability_name"]
    out["description"] = req.Raw["description"]
    out["source"]      = "acme"

    if cvss, ok := req.Raw["cvss_score"].(float64); ok {
        out["severity"] = cvssToSeverity(cvss)
    } else {
        warnings = append(warnings, "missing cvss_score")
    }

    return sdk.TransformResult{RecordType: req.RecordType, Normalised: out, Warnings: warnings}, nil
}
```

## 6. Implement Healthcheck

```go
func (c *AcmeConnector) Healthcheck(ctx context.Context) sdk.HealthStatus {
    s := sdk.HealthStatus{Connector: "acme", LastChecked: time.Now()}
    start := time.Now()
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+"/ping", nil)
    c.auth.AddAuth(ctx, req)
    resp, err := c.client.Do(req)
    s.Latency = time.Since(start)
    if err != nil { s.Message = err.Error(); return s }
    defer resp.Body.Close()
    s.Healthy = resp.StatusCode == 200
    s.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
    return s
}
```

## 7. Register and Use

```go
func main() {
    c, _ := acme.New(acme.Config{BaseURL: "https://api.acme.io/v2", APIKey: os.Getenv("ACME_KEY")})
    sdk.DefaultRegistry.MustRegister("acme", c)
    c.Connect(context.Background())

    ch, _ := c.Sync(context.Background(), sdk.SyncOptions{Since: time.Now().Add(-24*time.Hour)})
    for record := range ch {
        result, _ := c.Transform(context.Background(), sdk.TransformRequest{
            RecordType: "vulnerability",
            Raw:        record.Payload,
        })
        ingestIntoAIAUDITOR(result)
    }
}
```

## 8. Test with the Harness

```go
func TestAcmeConnector_Sync(t *testing.T) {
    h := harness.New(t)
    h.AddJSONResponse("/vulnerabilities", http.StatusOK, map[string]any{
        "items": []map[string]any{
            {"id": "CVE-2024-001", "vulnerability_name": "Log4Shell", "cvss_score": 10.0,
             "updated_at": time.Now().UTC().Format(time.RFC3339)},
        },
        "next_cursor": "",
    })
    h.AddResponse("/ping", http.StatusOK, `{}`)

    c, _ := acme.New(acme.Config{BaseURL: h.URL(), APIKey: "test"})
    c.Connect(t.Context())

    ch, _ := c.Sync(t.Context(), sdk.SyncOptions{})
    var records []sdk.SyncRecord
    for r := range ch { records = append(records, r) }

    if len(records) != 1 { t.Fatalf("want 1, got %d", len(records)) }

    // Verify the auth header was sent.
    req := h.Requests("/vulnerabilities")[0]
    if req.Headers.Get("Authorization") != "Bearer test" {
        t.Errorf("missing auth header")
    }
}
```

## Common Patterns

### Permanent errors (no retry)

```go
if resp.StatusCode == 401 || resp.StatusCode == 403 {
    return retry.Permanent(fmt.Errorf("auth error: HTTP %d", resp.StatusCode))
}
```

### Circuit breaker with observability

```go
breaker := circuit.New(circuit.Config{
    FailureThreshold: 5,
    RecoveryTimeout:  30 * time.Second,
    OnStateChange: func(from, to circuit.State) {
        metrics.RecordCircuitStateChange("acme", from.String(), to.String())
    },
})
```

### Paginated sync with early termination

```go
// opts.Limit stops after N total records.
sent := 0
for cursor != "" && (opts.Limit == 0 || sent < opts.Limit) {
    page, _ := fetchPage(ctx, cursor)
    for _, item := range page.Items {
        ch <- toRecord(item)
        sent++
        if opts.Limit > 0 && sent >= opts.Limit { return }
    }
    cursor = page.NextCursor
}
```

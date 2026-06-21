# AIAUDITOR Connector SDK

The Connector SDK is a Go library for building third-party integrations with the
AIAUDITOR platform. It provides a clean interface contract, ready-made auth
abstractions, built-in resilience primitives, and a test harness so you can write
well-behaved, production-grade connectors quickly.

## Quick Start

```go
import (
    sdk    "github.com/davejduke/obvious/sdk/connector"
    "github.com/davejduke/obvious/sdk/connector/auth"
    "github.com/davejduke/obvious/sdk/connector/examples/rest"
)

func main() {
    c, err := rest.New(rest.Config{
        Name:       "my-api",
        BaseURL:    "https://api.example.com/v1",
        DataPath:   "/findings",
        HealthPath: "/health",
        Auth:       auth.NewAPIKeyAuth(auth.APIKeyConfig{Key: os.Getenv("API_KEY")}),
        RecordType: "finding",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Register with the platform.
    sdk.DefaultRegistry.MustRegister("my-api", c)

    ctx := context.Background()
    if err := c.Connect(ctx); err != nil {
        log.Fatal(err)
    }

    ch, _ := c.Sync(ctx, sdk.SyncOptions{Limit: 100})
    for record := range ch {
        result, _ := c.Transform(ctx, sdk.TransformRequest{
            RecordType: record.Type,
            Raw:        record.Payload,
        })
        fmt.Println(result.Normalised)
    }
}
```

## Package Layout

```
sdk/connector/
├── connector.go          — Connector interface + core types
├── registry.go           — Connector registry (Register, Get, List)
├── auth/
│   └── auth.go           — API key, OAuth 2.0 client credentials, certificate auth
├── ratelimit/
│   └── token_bucket.go   — Token bucket rate limiter
├── retry/
│   └── retry.go          — Exponential backoff retry
├── circuit/
│   └── breaker.go        — Circuit breaker (closed/open/half-open)
├── harness/
│   └── harness.go        — Test harness: mock server + request recording
├── examples/
│   └── rest/
│       └── connector.go  — Generic REST connector (copy-paste starting point)
└── docs/
    ├── README.md         — Getting started (this file)
    ├── INTERFACE.md      — Interface reference
    └── EXAMPLE.md        — Example connector walkthrough
```

## Authentication

Three auth providers are included. All implement `auth.AuthProvider` and can be
swapped without changing connector code.

### API Key

```go
// Injects "Authorization: Bearer <key>" header.
auth.NewAPIKeyAuth(auth.APIKeyConfig{Key: "mysecret"})

// Custom header.
auth.NewAPIKeyAuth(auth.APIKeyConfig{
    Key:        "mysecret",
    HeaderName: "X-API-Key",
})

// Query parameter.
auth.NewAPIKeyAuth(auth.APIKeyConfig{
    Key:        "mysecret",
    Location:   auth.APIKeyQuery,
    QueryParam: "api_key",
})
```

### OAuth 2.0 Client Credentials

```go
auth.NewOAuth2ClientCredentials(auth.OAuth2Config{
    TokenURL:     "https://auth.example.com/token",
    ClientID:     os.Getenv("CLIENT_ID"),
    ClientSecret: os.Getenv("CLIENT_SECRET"),
    Scopes:       []string{"read:findings"},
})
```

Tokens are cached and refreshed automatically 30 seconds before expiry.

### Certificate (mTLS)

```go
certPEM, _ := os.ReadFile("client.crt")
keyPEM,  _ := os.ReadFile("client.key")

certAuth, err := auth.NewCertificateAuth(auth.CertificateConfig{
    CertPEM: certPEM,
    KeyPEM:  keyPEM,
})
// certAuth.ConfigureClient(httpClient) — called automatically by rest.New.
```

## Rate Limiting

The token bucket rate limiter controls outbound request volume.

```go
bucket, _ := ratelimit.New(ratelimit.Config{
    Rate:  10,  // tokens/second
    Burst: 20,  // max burst
})
bucket.Acquire(ctx) // blocks until a token is available
```

## Circuit Breaker

```go
b := circuit.New(circuit.Config{
    FailureThreshold: 5,
    RecoveryTimeout:  30 * time.Second,
    OnStateChange: func(from, to circuit.State) {
        log.Printf("circuit %s → %s", from, to)
    },
})

err := b.Do(func() error {
    return doRemoteCall()
})
if errors.Is(err, circuit.ErrOpen) {
    // Circuit is open — back off.
}
```

## Retry

```go
err := retry.Do(ctx, retry.Config{
    MaxAttempts: 3,
    BaseDelay:   500 * time.Millisecond,
    MaxDelay:    30 * time.Second,
    Multiplier:  2.0,
}, func() error {
    return doRemoteCall()
})

// Mark an error as permanent (no retry):
return retry.Permanent(err)
```

## Test Harness

```go
func TestMyConnector(t *testing.T) {
    h := harness.New(t)
    h.AddJSONResponse("/findings", http.StatusOK, map[string]any{
        "items": []map[string]any{{"id": "v-1", "title": "XSS"}},
    })

    c, _ := myconnector.New(myconnector.Config{BaseURL: h.URL()})
    c.Connect(context.Background())

    ch, _ := c.Sync(context.Background(), sdk.SyncOptions{})
    records := collect(ch)

    // Assert the request was well-formed.
    req := h.Requests("/findings")[0]
    assert.Equal(t, "Bearer mytoken", req.Headers.Get("Authorization"))
}
```

The harness queues responses FIFO and sticks the last response when the queue
empties, so a single `AddJSONResponse` covers unlimited calls.

## Writing a Custom Connector

1. Copy `sdk/connector/examples/rest/` to your own package.
2. Implement the four interface methods: `Connect`, `Sync`, `Transform`, `Healthcheck`.
3. Use `auth.AuthProvider` for credentials — do not hardcode them.
4. Wrap outbound calls with `bucket.Acquire` + `breaker.Do` + `retry.Do`.
5. Register with `sdk.DefaultRegistry.Register("my-connector", c)`.
6. Test with `harness.New(t)`.

See `docs/INTERFACE.md` for the full interface contract and `docs/EXAMPLE.md`
for a step-by-step walkthrough.

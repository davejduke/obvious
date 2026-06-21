# Connector Interface Reference

## Package `github.com/davejduke/obvious/sdk/connector`

### `Connector` Interface

Every AIAUDITOR integration must implement this interface.

```go
type Connector interface {
    Connect(ctx context.Context) error
    Sync(ctx context.Context, opts SyncOptions) (<-chan SyncRecord, error)
    Transform(ctx context.Context, req TransformRequest) (TransformResult, error)
    Healthcheck(ctx context.Context) HealthStatus
}
```

#### `Connect(ctx context.Context) error`

Called once at startup. Establishes the session, validates credentials, and surfaces
connectivity errors early. Must honour `ctx` deadline/cancellation.

**Contract:**
- Return `nil` when the connector is ready to serve `Sync` and `Healthcheck` calls.
- Return an error when credentials are invalid, the endpoint is unreachable, or any
  pre-flight check fails.
- Must not cache state that becomes stale (prefer re-fetching on `Sync`).

#### `Sync(ctx context.Context, opts SyncOptions) (<-chan SyncRecord, error)`

Returns a channel of `SyncRecord` values. The channel is closed when all records have
been sent or `ctx` is cancelled.

```go
type SyncOptions struct {
    Since  time.Time // zero = return all records
    Limit  int       // 0 = no limit
    Cursor string    // pagination token from previous Sync
    Filter string    // provider-specific query expression
}
```

**Contract:**
- Return the channel and `nil` error immediately (do not block the caller).
- Send records on the channel from a goroutine.
- Close the channel when done — never leave it open.
- Honour `ctx`; stop sending when `ctx.Done()` fires.
- Set `SyncRecord.NextCursor` for pagination; empty string means no more pages.

#### `Transform(ctx context.Context, req TransformRequest) (TransformResult, error)`

Converts a raw provider record to the AIAUDITOR canonical schema. Called per record,
must be stateless and safe for concurrent use.

```go
type TransformRequest struct {
    RecordType string         // target AIAUDITOR schema name
    Raw        map[string]any // provider-native payload
}

type TransformResult struct {
    RecordType string         // mirrors TransformRequest.RecordType
    Normalised map[string]any // canonical representation
    Warnings   []string       // non-fatal issues (missing optional fields, etc.)
}
```

**Contract:**
- Never return an error for missing optional fields — use `Warnings` instead.
- Return an error only when transformation is fundamentally impossible (e.g. nil input).
- Must not make network calls.

#### `Healthcheck(ctx context.Context) HealthStatus`

Performs a lightweight probe (e.g. a ping endpoint call). Called by the platform
health monitoring system.

```go
type HealthStatus struct {
    Healthy     bool
    Connector   string
    LastChecked time.Time
    Latency     time.Duration
    Message     string // optional detail
}
```

**Contract:**
- Must complete within the `ctx` deadline (platform default: 5 seconds).
- `Healthy=false` should carry a descriptive `Message`.
- Do not perform a full `Sync` inside `Healthcheck`.

---

### `Registry`

```go
type Registry struct { ... }

func NewRegistry() *Registry
func (r *Registry) Register(name string, c Connector) error
func (r *Registry) MustRegister(name string, c Connector)
func (r *Registry) Get(name string) (Connector, bool)
func (r *Registry) List() []string         // alphabetically sorted
func (r *Registry) Unregister(name string)
func (r *Registry) Len() int

var DefaultRegistry = NewRegistry()        // process-level singleton
```

---

## Package `github.com/davejduke/obvious/sdk/connector/auth`

### `AuthProvider` Interface

```go
type AuthProvider interface {
    AddAuth(ctx context.Context, req *http.Request) error
    Scheme() string // "apikey" | "oauth2_client_credentials" | "certificate"
}
```

### Implementations

| Type | Scheme | Description |
|------|--------|-------------|
| `*APIKeyAuth` | `apikey` | Static token; header or query param |
| `*OAuth2ClientCredentials` | `oauth2_client_credentials` | RFC 6749 §4.4 with token caching |
| `*CertificateAuth` | `certificate` | mTLS; configures `http.Client.Transport` |

---

## Package `github.com/davejduke/obvious/sdk/connector/ratelimit`

```go
type Config struct {
    Rate  float64 // tokens/second
    Burst float64 // max bucket capacity
}

func New(cfg Config) (*TokenBucket, error)
func DefaultConfig() Config  // Rate:10, Burst:20

func (tb *TokenBucket) Acquire(ctx context.Context) error  // blocking
func (tb *TokenBucket) TryAcquire() bool                  // non-blocking
func (tb *TokenBucket) Tokens() float64
func (tb *TokenBucket) WaitTime(n float64) time.Duration
```

---

## Package `github.com/davejduke/obvious/sdk/connector/circuit`

```go
type State int   // StateClosed | StateOpen | StateHalfOpen

type Config struct {
    FailureThreshold int
    RecoveryTimeout  time.Duration
    OnStateChange    func(from, to State)  // optional; async callback
}

func New(cfg Config) *Breaker
func DefaultConfig() Config  // FailureThreshold:5, RecoveryTimeout:30s

var ErrOpen = errors.New("circuit breaker is open")

func (b *Breaker) Allow() error
func (b *Breaker) Record(err error)
func (b *Breaker) Do(fn func() error) error  // Allow + fn + Record
func (b *Breaker) State() State
func (b *Breaker) ConsecutiveFailures() int
```

---

## Package `github.com/davejduke/obvious/sdk/connector/retry`

```go
type Config struct {
    MaxAttempts int
    BaseDelay   time.Duration
    MaxDelay    time.Duration
    Multiplier  float64
    Jitter      float64  // 0–1; fraction of delay to add as random jitter
}

func DefaultConfig() Config  // MaxAttempts:3, BaseDelay:500ms, MaxDelay:30s, ×2

func Do(ctx context.Context, cfg Config, fn func() error) error
func DoWithAttempts(ctx context.Context, cfg Config, fn func() error) ([]Attempt, error)

func Permanent(err error) error    // wrap to stop retrying
func IsPermanent(err error) bool

type Attempt struct {
    Number int
    Err    error
    Delay  time.Duration // delay applied after this attempt
}
```

---

## Package `github.com/davejduke/obvious/sdk/connector/harness`

```go
func New(t testing.TB) *Harness
func NewTLS(t testing.TB) *Harness  // uses httptest.NewTLSServer

func (h *Harness) URL() string
func (h *Harness) Client() *http.Client

// Registering fixtures
func (h *Harness) AddResponse(path string, status int, body string)
func (h *Harness) AddJSONResponse(path string, status int, v any)
func (h *Harness) AddResponseRaw(path string, resp Response)
func (h *Harness) AddErrorResponse(path string, status int, message string)

// Assertions
func (h *Harness) Requests(path string) []RecordedRequest
func (h *Harness) AllRequests() []RecordedRequest
func (h *Harness) RequestCount(path string) int
func (h *Harness) Reset()

type RecordedRequest struct {
    Method  string
    Path    string
    Query   string
    Headers http.Header
    Body    []byte
}
```

Responses are queued FIFO. When the queue empties the last response is
reused (sticky), so a single `AddResponse` covers unlimited requests.

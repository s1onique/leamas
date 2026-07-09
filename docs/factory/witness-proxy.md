# Witness Proxy

## Overview

The witness proxy is a local-only HTTP proxy that captures bounded request/response metadata as witness evidence. It is designed to provide review evidence for AI-assisted development loops without capturing sensitive content.

## Package Location

```
internal/witness/proxy
```

## What It Is

- **Local-only by default**: Binds to loopback (127.0.0.1) addresses
- **Single upstream**: Proxies to exactly one configured upstream target
- **Bounded in-memory capture**: Stores up to 100 records by default (configurable)
- **Metadata-only**: Does not capture body content by default
- **Thread-safe**: Uses mutex protection for concurrent access

## What It Is Not

- **Not a provider router**: Does not select between multiple providers
- **Not a model gateway**: Does not route to different models
- **Not a LiteLLM replacement**: Does not implement LiteLLM-compatible APIs
- **Not an auth/RBAC system**: Does not enforce permissions
- **Not persistent storage**: Records are in-memory only
- **Not a run-bundle generator**: Does not create Hulk run bundles

## Security Design

### Default Bind Address

The proxy defaults to loopback-only binding (`127.0.0.1:0`). This prevents external access to the proxy.

### Header Capture

Headers are **not captured by default**. When header capture is enabled via `CaptureHeaders: true`, sensitive headers are redacted:

- `Authorization`
- `Proxy-Authorization`
- `Cookie`
- `Set-Cookie`
- `X-Api-Key`
- `Api-Key`

Redacted values are replaced with `[REDACTED]`.

### Body Content

Request and response bodies are never captured.

## Usage

```go
import "github.com/s1onique/leamas/internal/witness/proxy"

// Create a proxy with a single upstream target.
p, err := proxy.New(proxy.Config{
    UpstreamURL:    "http://localhost:8080",
    ListenAddr:     "127.0.0.1:0",
    MaxRecords:    100,
    CaptureHeaders: false, // default
})
if err != nil {
    // handle error
}

// Use the handler in your HTTP server.
http.ListenAndServe(":8081", p.Handler())

// Access captured records.
records := p.Records()
for _, rec := range records {
    fmt.Printf("Method: %s, Path: %s, Status: %d\n",
        rec.Method, rec.Path, rec.StatusCode)
}
```

## Configuration

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `ListenAddr` | string | No | `127.0.0.1:0` | Address to listen on |
| `UpstreamURL` | string | Yes | - | Single target upstream URL |
| `MaxRecords` | int | No | 100 | Maximum records to retain |
| `CaptureHeaders` | bool | No | false | Enable header capture |

## API

### `New(config Config) (*WitnessProxy, error)`

Creates a new witness proxy. Returns an error if `UpstreamURL` is empty or invalid.

### `Handler() http.Handler`

Returns an `http.Handler` that proxies requests to the configured upstream.

### `Records() []WitnessRecord`

Returns a defensive copy of all captured records.

### `Reset()`

Clears all captured records.

## Record Structure

```go
type WitnessRecord struct {
    ID              string             // Unique record identifier
    Method          string             // HTTP method (GET, POST, etc.)
    Path            string             // Request path including query string
    UpstreamURL     string             // Configured upstream target
    RequestHeaders  map[string][]string // Request headers (if CaptureHeaders enabled)
    ResponseHeaders map[string][]string // Response headers (if CaptureHeaders enabled)
    StatusCode      int                // Upstream response status code
    Error           string             // Error message if request failed
    StartedAt       string             // RFC3339Nano timestamp
    CompletedAt     string             // RFC3339Nano timestamp
}
```

## Bounded Behavior

When `MaxRecords` is exceeded, the oldest record is dropped to make room for new records. This ensures bounded memory usage.

## Testing

The proxy is designed to be testable with `httptest`:

```go
func TestProxyExample(t *testing.T) {
    upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    defer upstream.Close()

    p, err := proxy.New(proxy.Config{UpstreamURL: upstream.URL})
    if err != nil {
        t.Fatal(err)
    }

    proxyServer := httptest.NewServer(p.Handler())
    defer proxyServer.Close()

    // Make requests through the proxy.
    resp, err := http.Get(proxyServer.URL)
    // ...

    // Check captured records.
    records := p.Records()
    // ...
}
```

## CLI Usage

The witness proxy can be started via the `leamas witness proxy` command:

```bash
# Basic usage
leamas witness proxy --upstream http://127.0.0.1:8080

# Custom listen address
leamas witness proxy --listen 127.0.0.1:8766 --upstream http://127.0.0.1:8080

# With header capture enabled
leamas witness proxy --upstream http://localhost:8080 --capture-headers --max-records 250
```

### CLI Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--upstream` | Yes | - | Single target upstream URL (must start with http:// or https://) |
| `--listen` | No | `127.0.0.1:0` | Listen address (loopback only) |
| `--max-records` | No | `100` | Maximum records to retain (0 = package default) |
| `--capture-headers` | No | `false` | Enable header capture with sanitization |
| `--help`, `-h` | No | - | Show help |

### Security Constraints

- **Listen addresses**: Only loopback addresses are allowed (`127.0.0.1:*`, `localhost:*`)
- **Rejected addresses**: `0.0.0.0:*`, `[::]:*`, `:*`, private networks (`192.168.*`, `10.*`, `172.16.*`)
- **Upstream**: Must be http:// or https:// URL (no routing tables, single upstream only)

### Example Output

```
Leamas witness proxy listening on http://127.0.0.1:54322
Upstream: http://127.0.0.1:8080
Capture headers: false
Press Ctrl-C to stop.
```

## Limitations

- No integration with Hulk cores (`runbundle`, `claimevidence`) in this seed.
- No external service discovery or health checks.
- No TLS support (use a reverse proxy in front if needed).
- IPv6 loopback `[::1]` not yet supported in CLI.

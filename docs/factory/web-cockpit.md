# Web Cockpit

A local-only, read-only web cockpit for reviewing Leamas status and static/demo evidence.

## Overview

The cockpit provides a minimal web interface and JSON API for inspecting Leamas component status. It is designed as a local-first, dependency-light seed that can be run alongside the main Leamas binary.

## Security Properties

The cockpit is explicitly **local-only** and **read-only**:

- Default listen address: `127.0.0.1:0` (loopback only, dynamic port)
- No authentication or session system
- No database or filesystem persistence
- No network client behavior
- No Set-Cookie headers emitted

**Warning:** Do not expose the cockpit beyond loopback (127.0.0.1). It has no authentication.

## What the Cockpit Is NOT

The cockpit is explicitly **not**:

- An enterprise admin UI
- An auth/OIDC/OAuth/RBAC system
- A database application
- A live witness-proxy runtime
- A provider gateway
- A model control plane
- A LiteLLM-compatible API
- A general-purpose web application framework

## Package Location

```
internal/web/cockpit
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Embedded HTML status page |
| GET | `/api/status` | System status as JSON |
| GET | `/api/components` | Component status as JSON |
| GET | `/assets/*` | Static assets (CSS, etc.) |

## API Examples

### GET /api/status

```json
{
  "name": "Leamas Cockpit",
  "mode": "local-only",
  "read_only": true,
  "storage": "none",
  "auth": "none"
}
```

### GET /api/components

```json
{
  "components": [
    {
      "name": "factory",
      "status": "available",
      "summary": "Factory gates and digest workflow"
    },
    {
      "name": "hulk",
      "status": "seeded",
      "summary": "Typed run-bundle and claim/evidence cores"
    },
    {
      "name": "witness",
      "status": "seeded",
      "summary": "Local witness proxy package; runtime not started by cockpit"
    }
  ]
}
```

## CLI Usage

The cockpit is wired into the `leamas` binary as a local-only CLI command:

```bash
# Start cockpit with random port on loopback (default)
leamas cockpit serve

# Start cockpit with specific port
leamas cockpit serve --listen 127.0.0.1:8080

# Show help
leamas cockpit serve --help
```

### CLI Constraints

The CLI enforces loopback-only binding for security:

- **Default listen address**: `127.0.0.1:0` (dynamic port allocation)
- **Allowed addresses**: `127.0.0.1:*`, `localhost:*`
- **Rejected addresses**: `0.0.0.0:*`, `[::]:*`, `:*` (any non-loopback)

### CLI Behavior

1. Prints the actual URL with the chosen port when using `:0`
2. Shuts down gracefully on SIGINT/SIGTERM
3. Returns non-zero on listener/server startup errors

Example output:
```
Leamas cockpit listening on http://127.0.0.1:54321
Press Ctrl-C to stop.
```

### Programmatic Usage

```go
package main

import (
    "log"
    "net/http"

    "github.com/s1onique/leamas/internal/web/cockpit"
)

func main() {
    c, err := cockpit.New(cockpit.Config{
        ListenAddr: "127.0.0.1:8080",
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Starting cockpit at http://%s", c.ListenAddr())
    log.Fatal(http.ListenAndServe(c.ListenAddr(), c.Handler()))
}
```

## Design Constraints

The cockpit seed adheres to strict constraints:

- **Go-only**: No Python, Node, or other runtimes
- **Embedded static**: HTML/CSS served from Go `embed`, no filesystem at runtime
- **No frontend build chain**: No Vite, Webpack, React, Tailwind, npm
- **No database**: In-memory only, no persistence
- **No auth system**: No OIDC, OAuth, sessions, cookies
- **No network clients**: Does not call external APIs
- **No provider/model routing**: Not a gateway or control plane
- **No witness runtime**: Does not start the witness proxy

## Static Assets

Static files are embedded using Go's `embed` package:

```
internal/web/cockpit/static/
├── index.html
└── style.css
```

## Testing

The cockpit includes comprehensive `httptest` coverage:

```bash
go test ./internal/web/cockpit/... -v
```

## Deferred Work

- Boundary verifier to ensure no forbidden imports
- Browser auto-open (`ACT-LEAMAS-WEB-COCKPIT-BROWSER-OPEN01`)
- Witness proxy CLI wiring (`ACT-LEAMAS-WITNESS-PROXY-CLI01`)

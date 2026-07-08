# ADR-0003: Web-first local cockpit

## Status

Accepted

## Context

Developer tools need good user interfaces. The choice of UI paradigm affects:
- Platform coverage (macOS, Linux, Windows, WSL)
- Installation complexity
- User expectations
- Development effort

Traditional CLI tools have limited interactivity. Native GUIs require platform-specific code. Web interfaces offer rich interactivity with universal availability.

## Decision

Leamas will use a **web-first local cockpit** as its primary interface:

1. Default: Browser-based UI served on localhost
2. Binary serves its own web assets (embedded)
3. Fallback: CLI for scripting and headless environments
4. No external dependencies for the UI

### Architecture

```
┌─────────────────────────────────────┐
│           Developer Browser          │
│         (http://localhost:PORT)      │
└─────────────────┬───────────────────┘
                  │ HTTP
┌─────────────────┴───────────────────┐
│         Leamas Binary               │
│  ┌─────────────┬─────────────────┐  │
│  │   CLI/API   │   Web Server    │  │
│  │  (stdin/out)│  (localhost)    │  │
│  └─────────────┴─────────────────┘  │
└─────────────────────────────────────┘
```

### Port Assignment

- Default port: 8080 (configurable via `--port` flag)
- Leamas claims "localhost only" - no binding to external interfaces
- Port conflict resolution: automatic increment or error with suggestion

### Static Assets

- Embedded using Go's `embed` package
- No external CDN or network requests for UI
- Assets bundled at compile time

## Rationale

1. **Ubiquity**: Every developer has a browser
2. **Rich interactions**: CSS/JS enables good UX without native code
3. **Cross-platform**: Works identically on all platforms with a browser
4. **Single binary**: No separate web server process
5. **Offline capable**: No internet required for the UI
6. **Familiar paradigm**: Developers already live in browsers

## Consequences

### Positive

- Rich, interactive UI without platform-specific code
- Same UI works on macOS, Linux, Windows, WSL
- Can be accessed from any device on the same machine
- Can be proxied behind a reverse proxy if needed (optional)

### Negative

- Requires a browser (acceptable for modern developer workflows)
- Security considerations for localhost server
- More complex than pure CLI (but worth it for UX)

### Neutral

- CLI remains available for scripting
- Headless mode can disable web server

## Implementation Notes

### Web Server Requirements

- Listen on `127.0.0.1` or `localhost` only
- No TLS (localhost doesn't need it)
- Graceful shutdown on signal
- Configurable port

### Static Asset Requirements

- Single-page application structure
- No build step required (pre-compiled assets)
- Fallback for `curl`-based interactions

### Security Considerations

- Localhost-only binding prevents remote access
- No authentication for localhost (by default)
- Future: optional authentication for multi-user scenarios (see ADR-0004)

## Alternatives Considered

| Alternative | Rejected Reason |
|-------------|-----------------|
| Terminal UI (tview, bubbletea) | Platform-specific terminal behaviors |
| Native GUI (Qt, GTK, Electron) | Heavy dependencies, cross-platform complexity |
| CLI-only | Limited interactivity for complex workflows |
| SaaS web interface | Contradicts local-first principle |

## Revisit Criteria

- If browsers become obsolete
- If a compelling native UI use case emerges
- If embedded assets become unwieldy

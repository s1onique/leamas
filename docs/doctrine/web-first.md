# Doctrine: Web First

The primary human interface for Leamas is a local web application.

## Core Principle

Developers interact with Leamas through a web browser running on localhost. This is the default UX, not an optional feature.

## Rationale

- **Ubiquity**: Every developer has a browser
- **No native UI**: Avoids platform-specific UI toolkits
- **Familiar paradigm**: Developers already live in browsers
- **Rich interactions**: CSS/JS enables good UX without native code
- **Local scope**: No cloud infrastructure required for the UI

## Architecture

```
┌─────────────────────────────────────┐
│           Developer Browser          │
│         (http://localhost)          │
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

## What This Is NOT

- This is NOT a SaaS web application
- This is NOT requiring internet connectivity
- This is NOT a SPA that needs a build step
- This is NOT a Electron/TAURI desktop app

## Implementation Notes

- Embedded static assets (no external CDN)
- Single binary serves its own web assets
- No Java compilation or bundling required
- Works with `curl` as fallback for headless environments

## Agent Contract

### Always

- Ensure the web interface is embedded in the single binary.
- Test the web UI locally before claiming it works.
- Use localhost (127.0.0.1) for the web server.

### Never

- Add external CDN dependencies for web assets.
- Require internet connectivity for the web UI.
- Add Electron, Tauri, or other desktop wrapper frameworks.

### Ask / Escalate

- If a feature requires a build step for the web UI.
- If headless operation is unclear for a feature.

### Verification Hooks

- `scripts/verify_static_binary_intent.sh` (checks embedded assets)

## References

- ADR-0003: Web-first local cockpit

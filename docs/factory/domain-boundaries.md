# Domain Boundary Import Policies

This document describes the domain boundary import policies enforced by the Factory boundary verifier.

## Overview

Leamas has several intentionally constrained internal packages that were deliberately introduced with narrow boundaries. The domain boundary verifier protects those boundaries from future drift by statically checking import declarations.

## Protected Packages

### Hulk Pure Domain Packages

- `internal/hulk/runbundle`: Pure domain logic for run bundles
- `internal/hulk/claimevidence`: Pure domain logic for claims and evidence

These packages must remain pure domain logic with no side effects.

**Allowed standard-library imports:**
- `sort` - for deterministic ordering
- `strings` - for string manipulation

**Forbidden standard-library imports:**
- `context` - domain cores must not import context
- `time` - domain cores must not import time
- `os` - domain cores must not import os
- `io`, `io/fs` - domain cores must not import io
- `path/filepath` - domain cores must not import path
- `net`, `net/http`, `net/url` - domain cores must not import network
- `database/sql` - domain cores must not import database
- `os/exec` - domain cores must not import process execution
- `sync` - domain cores must not import sync primitives
- `embed` - domain cores must not import embed
- `encoding/json` - domain cores must not import encoding

**Forbidden provider/control-plane imports by substring:**
- `openai`, `anthropic`, `litellm`, `ollama`, `gemini`, `bedrock`, `azure`
- `oauth`, `oidc`, `jwt`
- `session`, `cookie`
- `sqlite`, `postgres`, `mysql`

### Witness Proxy Seed

- `internal/witness/proxy`: Local HTTP witness proxy seed

This package is a local single-upstream metadata capture only.

**Allowed standard-library imports:**
- `errors` - for error handling
- `net/http` - for HTTP proxy functionality
- `net/http/httputil` - for reverse proxy utilities
- `net/url` - for URL parsing
- `strings` - for string manipulation
- `sync` - for mutex operations
- `time` - for timestamps

**Forbidden standard-library imports:**
- `database/sql` - proxy must not import database
- `os` - proxy must not import os
- `io/fs` - proxy must not import filesystem
- `path/filepath` - proxy must not import path
- `os/exec` - proxy must not import process execution
- `embed` - proxy must not import embed
- `encoding/json` - proxy must not import encoding
- `html/template`, `text/template` - proxy must not import templates

**Forbidden provider/control-plane imports by substring:**
- `openai`, `anthropic`, `litellm`, `ollama`, `gemini`, `bedrock`, `azure`
- `oauth`, `oidc`, `jwt`
- `session`, `cookie`
- `sqlite`, `postgres`, `mysql`

### Web Cockpit Seed

- `internal/web/cockpit`: Local read-only embedded static UI only

**Allowed standard-library imports:**
- `embed` - for static asset embedding
- `encoding/json` - for JSON responses
- `fmt` - for formatted output
- `net/http` - for HTTP handlers
- `strings` - for string manipulation

**Forbidden standard-library imports:**
- `database/sql` - cockpit must not import database
- `os` - cockpit must not import os
- `io/fs` - cockpit must not import filesystem
- `path/filepath` - cockpit must not import path
- `os/exec` - cockpit must not import process execution
- `net/http/httputil` - cockpit must not import reverse proxy utilities
- `net/url` - cockpit must not import URL packages
- `sync` - cockpit must not import sync primitives
- `time` - cockpit must not import time
- `html/template`, `text/template` - cockpit must not import templates

**Forbidden auth/provider/control-plane imports by substring:**
- `openai`, `anthropic`, `litellm`, `ollama`, `gemini`, `bedrock`, `azure`
- `oauth`, `oidc`, `jwt`
- `session`, `cookie`
- `sqlite`, `postgres`, `mysql`

## Verifier Implementation

The verifier is implemented in `internal/factory/boundary/` and:

- Uses Go AST parsing (not grep or shell commands)
- Scans only production files (skips `*_test.go`)
- Skips `testdata/`, `vendor/`, `.git/` directories
- Produces deterministic findings ordered by file, import, reason
- Requires no network access

## CLI Command

```bash
leamas factory verify domain-boundaries
```

## Make Target

```bash
make verify-domain-boundaries
```

## Integration

The verifier is wired into:
- CLI command: `leamas factory verify domain-boundaries`
- Make target: `make verify-domain-boundaries`
- Default gate: `make gate` and `make factorize`

## Notes

- Test files (`*_test.go`) are intentionally ignored by this verifier
- The verifier enforces static import boundaries only; it does not enforce runtime behavior
- Protected packages are expected to evolve within their declared scope

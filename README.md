# Leamas

A Golang-only, local-first tool distributed as a single static binary. Leamas makes test/verification harnesses accountable. Future iterations may intercept/proxy LLM interactions so harness behavior and LLM evaluation can become part of a repeatable feedback loop.

## Status

**v0.0.0** — Initial seed documentation only. No implementation yet.

## Quick Start

TBD once implementation is ready.

## Constraints

- Go only (v0)
- Single static binary
- Local-first developer tool
- No OAuth/OIDC
- No enterprise governance complexity
- No database unless proven necessary
- No Kubernetes-first assumptions
- Must be useful on a local developer machine first

## Documentation

- [docs/README.md](docs/README.md) — Documentation overview
- [docs/doctrine/README.md](docs/doctrine/README.md) — Core principles
- [docs/adr/README.md](docs/adr/README.md) — Architecture Decision Records
- [docs/playbooks/README.md](docs/playbooks/README.md) — Playbooks index

## Quality Gates

```bash
chmod +x scripts/quality_gate.sh
./scripts/quality_gate.sh
make gate
make test
```

# Leamas

A Go-only, local-first verification witness distributed as a single static
binary. Leamas makes AI-assisted test and verification harnesses accountable
by recording evidence, checking repository contracts, and exposing practical
Factory quality gates.

## Status

The Leamas implementation is available. The first release target is `v0.1.0`
for Linux amd64, distributed as a Debian package through GitHub Releases.

## Quick Start

Build and inspect a local development binary:

```bash
make build
./bin/leamas version
./bin/leamas doctor
```

Run the repository quality gates:

```bash
make factorize
make gate
```

For the released Linux amd64 package, see
[Debian installation](docs/install/debian.md).

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
- [docs/install/debian.md](docs/install/debian.md) — Debian package installation

## Quality Gates

```bash
make factorize
make gate
go test ./...
go vet ./...
```

## Schema introspection

The installed binary is self-describing for the Gate Summary wire format. Use

```bash
./bin/leamas gate-summary schema list
./bin/leamas gate-summary schema show v1
./bin/leamas gate-summary schema show v2
```

to obtain the exact embedded JSON Schema for v1 and v2 without
cloning the repository, reading Go source, or accessing the network.

See [docs/contracts/gate-summary-schema-introspection.md](docs/contracts/gate-summary-schema-introspection.md)
for the full contract.

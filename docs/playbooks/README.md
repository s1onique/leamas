# Playbooks

Operational runbooks and procedures for Leamas.

## Planned Playbooks

The following playbooks are planned for v0. Implementation will follow.

### Core Operations

| Playbook | Status | Description |
|----------|--------|-------------|
| run-local-gate | Planned | Run quality gate locally before committing |
| create-targeted-digest | Planned | Generate a reviewable digest from local working tree changes |
| capture-harness-interaction | Planned | Capture harness behavior for accountability review |
| review-harness-accountability | Planned | Review evidence for harness accountability |

### Development

| Playbook | Status | Description |
|----------|--------|-------------|
| local-setup | TBD | Setting up local development environment |
| test-running | TBD | Running tests locally |

### Operations

| Playbook | Status | Description |
|----------|--------|-------------|
| cli-usage | TBD | Basic command-line interface usage |
| troubleshooting | TBD | Common issues and solutions |

### Release (Future)

| Playbook | Status | Description |
|----------|--------|-------------|
| release-process | TBD | Creating and publishing releases |
| cross-compile | TBD | Building for multiple platforms |

## Format

Playbooks should include:
- **Prerequisites**: What must be in place before starting
- **Steps**: Numbered, executable instructions
- **Verification**: How to confirm success
- **Rollback**: How to undo if something goes wrong

## Contributing

When adding a new playbook:
1. Create a new `.md` file in this directory
2. Follow the format above
3. Update this index
4. Include the playbook in code review / testing workflow

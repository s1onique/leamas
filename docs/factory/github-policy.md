# GitHub Policy

**ACT**: ACT-LEAMAS-FACTORY-REMOTE-BRANCH-PROTECTION01-R1

## Overview

This document describes the Leamas GitHub policy for repository configuration. It is verified by `leamas factory verify github` which uses hardcoded Leamas-specific settings (owner: `s1onique`, repo: `leamas`).

## Repository

| Setting | Value |
|---------|-------|
| Owner | `s1onique` |
| Name | `leamas` |
| Default Branch | `main` |

## Branch Protection Policy

### `main` Branch Settings

The `main` branch should have these settings:

| Field | Value | Description |
|-------|-------|-------------|
| `pattern` | `main` | Branch pattern to protect |
| `required_status_checks` | `["Factory Gates"]` | Required CI checks |
| `require_strict_status_checks` | `true` | Branch must be up-to-date before merge |
| `enforce_for_admins` | `false` | Allow admin bypass for direct pushes |
| `allow_force_pushes` | `false` | Disable force-push |
| `allow_deletions` | `false` | Prevent branch deletion |

### Verification

Verify remote state:

```bash
leamas factory verify github
```

Or using gh CLI directly:

```bash
gh api graphql -f owner="s1onique" -f name="leamas" -f query='
query($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    branchProtectionRules(first: 50) {
      nodes {
        id
        pattern
        isAdminEnforced
        allowsForcePushes
        allowsDeletions
        requiresStatusChecks
        requiresStrictStatusChecks
        requiredStatusCheckContexts
        matchingRefs(first: 20) { nodes { name } }
      }
    }
  }
}' | jq '.data.repository.branchProtectionRules.nodes[]
  | select(.pattern == "main")'
```

Expected output:

```json
{
  "pattern": "main",
  "isAdminEnforced": false,
  "allowsForcePushes": false,
  "allowsDeletions": false,
  "requiresStatusChecks": true,
  "requiresStrictStatusChecks": true,
  "requiredStatusCheckContexts": ["Factory Gates"]
}
```

### Apply Command

To update branch protection if drift is detected:

```bash
gh api graphql \
  -f ruleId="BPR_kwDOSk9xFs4Ew057" \
  -f query='
mutation($ruleId: ID!) {
  updateBranchProtectionRule(input: {
    branchProtectionRuleId: $ruleId
    isAdminEnforced: false
    allowsForcePushes: false
    allowsDeletions: false
    requiresStatusChecks: true
    requiresStrictStatusChecks: true
  }) {
    branchProtectionRule {
      id
      pattern
      isAdminEnforced
    }
  }
}'
```

## References

- [Branch Protection Proof](./branch-protection-proof.md)
- [Git Safety](./git-safety.md)

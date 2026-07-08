# Branch Protection Proof

**Repository:** s1onique/leamas  
**Default Branch:** main  
**Timestamp:** 2026-07-08T19:51:00Z

## Protection Status

| Setting | Value |
|---------|-------|
| `protected` | `true` |
| `protection_url` | `https://api.github.com/repos/s1onique/leamas/branches/main/protection` |

## Required Status Checks

| Setting | Value |
|---------|-------|
| `strict` | `true` (branch must be up to date before merge) |
| `contexts` | `["Factory Gates"]` |
| `app_id` | `15368` |

## Enforcement Settings

| Setting | Value |
|---------|-------|
| `enforce_admins` | `true` (admin bypass disabled) |
| `allow_force_pushes` | `false` (force-push disabled) |
| `allow_deletions` | `false` (branch deletion disabled) |
| `required_pull_request_reviews` | `null` (not required for v0) |

## Verification Commands

```bash
gh api repos/s1onique/leamas/branches/main --jq '{name,protected,protection_url}'
# Output: {"name":"main","protected":true,"protection_url":"https://api.github.com/repos/s1onique/leamas/branches/main/protection"}

gh api repos/s1onique/leamas/branches/main/protection --jq '{required_status_checks,enforce_admins,allow_force_pushes,allow_deletions}'
```

## Policy Compliance

- [x] Force-push disabled
- [x] Branch deletion disabled
- [x] Factory CI required as status check
- [x] Branch must be up to date before merge
- [x] Admin bypass disabled
- [x] No PR reviews required (v0 exception)

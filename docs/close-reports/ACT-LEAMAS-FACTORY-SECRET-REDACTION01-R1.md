# Close Report: ACT-LEAMAS-FACTORY-SECRET-REDACTION01-R1

## ACT Reference

**ACT-LEAMAS-FACTORY-SECRET-REDACTION01-R1**: Wire secret redaction into digest output and correct over-claimed closure

## Summary

Original ACT added the redaction package. R1 wired it into digest output and corrected the behavior claim by:

1. **Wiring redaction into `digest.Write()`** - The digest output boundary now applies redaction before writing
2. **Removing generic long-hex redaction** - Changed from Option B (restrict to high-confidence contexts) to **Option A** (remove entirely) to prevent Git commit hashes from being redacted
3. **Expanding AWS pattern coverage** - Added ASIA pattern support for temporary credentials
4. **Adding comprehensive tests** - Both redact package tests and digest-level integration tests

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/redact/redact.go` | MODIFIED - Removed generic long-hex redaction, expanded AWS patterns |
| `internal/factory/redact/redact_test.go` | MODIFIED - Comprehensive tests for all patterns |
| `internal/factory/digest/digest.go` | MODIFIED - Added redact import, wired redaction into Write() |
| `internal/factory/digest/redact_integration_test.go` | NEW - Digest-level integration test |
| `docs/close-reports/ACT-LEAMAS-FACTORY-SECRET-REDACTION01.md` | MODIFIED - Added R1 note |
| `docs/close-reports/ACT-LEAMAS-FACTORY-SECRET-REDACTION01-R1.md` | NEW - This close report |

## Behavior Changed

### Redaction Boundary

Redaction is now applied at the **digest Write() boundary** in `internal/factory/digest/digest.go`:

```go
func Write(opts Options) error {
    content, err := Generate(opts)
    if err != nil {
        return err
    }
    // Redact secrets from digest output before writing
    content = redact.RedactDigest(content)
    // ... write content
}
```

### Patterns Covered

| Pattern Type | Example | Redacted As |
|--------------|---------|-------------|
| Bearer tokens | `Bearer eyJhbGciOi...` | `Bearer [REDACTED]` |
| OpenAI keys | `sk-1234567890abcdef...` | `sk-[REDACTED]` |
| Anthropic keys | `sk-ant-1234567890...` | `sk-ant-[REDACTED]` |
| GitHub tokens | `ghp_1234567890abcdef...` | `ghp_[REDACTED]` |
| AWS access keys | `AKIAIOSFODNN7EXAMPLE` | `[REDACTED_AWS_KEY]` |
| AWS temp keys | `ASIAIOSFODNN7EXAMPLE` | `[REDACTED_AWS_KEY]` |
| Password vars | `password=secret123` | `password=[REDACTED]` |
| Secret vars | `secret=myvalue` | `secret=[REDACTED]` |
| Token vars | `token=myvalue` | `token=[REDACTED]` |
| API key vars | `api_key=myvalue` | `api_key=[REDACTED]` |
| Private keys | `-----BEGIN RSA PRIVATE KEY-----` | `-----BEGIN [REDACTED] PRIVATE KEY-----` |

### Patterns Intentionally NOT Covered

- **Generic long hex strings (40+ chars)** - Option A chosen: removed entirely because:
  - Git commit hashes are 40-char hex strings that are important evidence
  - Generic entropy detection is out of scope for this ACT
  - High-confidence patterns are sufficient

- **Broad entropy-based detection** - Out of scope per requirements

### Git Commit Hash Preservation

Git commit hashes are **preserved by default** because:

1. Removed the `[a-f0-9]{40,}` pattern that would have matched them
2. Added explicit test `TestRedactPreservesGitCommitHash` to verify behavior
3. 40-char hex strings that appear in digest output are not redacted

## Verification

### Commands Run

```bash
go test ./internal/factory/redact/... -v
go test ./internal/factory/digest/... -v
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
./bin/leamas factory digest --output /tmp/leamas-redaction-check.md
make factorize
make gate
go test ./...
go vet ./...
```

### Results

- [x] All redact package tests pass (11 test cases)
- [x] All digest integration tests pass (3 test cases)
- [x] Binary builds successfully
- [x] `make factorize` passes
- [x] `make gate` passes
- [x] `go test ./...` passes
- [x] `go vet ./...` passes

## Decisions Made

1. **Option A for long-hex redaction**: Removed generic 40+ hex redaction entirely rather than trying to restrict it
2. **Single boundary approach**: Applied redaction in `Write()` rather than scattering calls across helpers
3. **Comprehensive tests**: Added both package-level and integration-level tests

## Agent Doctrine Impact

None. This is Factory tooling improvement.

## Open Questions

None.

## Follow-up ACTs

| ACT | Description | Priority |
|-----|-------------|----------|
| | | |

## Skipped/Deferred Checks

None.

## Notes

- Redaction is now wired into the actual digest output path
- Tests prove the behavior is wired, not merely package-local
- Git commit hashes are preserved as required
- Ordinary non-secret text remains unchanged

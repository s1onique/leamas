# Close Report: ACT-LEAMAS-WITNESS-CLAIM-EVIDENCE-CLI01

## Summary

Added a minimal CLI surface for creating, listing, showing, and linking
claim/evidence artifacts inside existing run bundles. Commands:
`leamas witness claim create/list/show/attach-evidence` and
`leamas witness evidence create/list/show`. All support `--root`, `--run-id`,
and `--json` flags.

## Files Changed

- `cmd/leamas/claim_cli.go` - New file with claim/evidence CLI dispatchers
- `cmd/leamas/claim_commands.go` - New file with claim create/list/show handlers
- `cmd/leamas/evidence_commands.go` - New file with evidence create/list/show handlers
- `cmd/leamas/claim_attach_evidence.go` - New file with attach-evidence handler
- `cmd/leamas/claim_cli_test.go` - New file with claim CLI dispatch tests
- `cmd/leamas/claim_cli_create_test.go` - New file with claim create/list/show tests
- `cmd/leamas/evidence_cli_test.go` - New file with evidence CLI dispatch tests
- `cmd/leamas/evidence_cli_create_test.go` - New file with evidence create/list/show tests
- `cmd/leamas/claim_evidence_attach_test.go` - New file with attach-evidence tests
- `cmd/leamas/claim_evidence_attach_idempotent_test.go` - New file with attach idempotent tests
- `cmd/leamas/witness.go` - Updated to wire new claim/evidence handlers
- `internal/witness/claim/id_test.go` - Updated to assert expected errors in validation tests
- `docs/factory/claims-evidence.md` - Updated with CLI documentation

## Cleanup Applied

Claim/evidence ID validation tests now assert expected errors instead of only checking that some error occurred:
- `TestValidateClaimIDRejectsUnsafeIDs` now uses `errors.Is(err, tc.expected)`
- `TestValidateEvidenceIDRejectsUnsafeIDs` now uses `errors.Is(err, tc.expected)`
- `TestValidateRelativePathRejectsUnsafePaths` now uses `errors.Is(err, tc.expected)`
- `claim-` returns `ErrIDMissingSuffix`
- `evidence-` returns `ErrIDMissingSuffix`

## CLI Added

| Command | Description |
|---------|-------------|
| `leamas witness claim create` | Create a claim in an existing run bundle |
| `leamas witness claim list` | List claims in a run bundle |
| `leamas witness claim show` | Show a claim's details |
| `leamas witness claim attach-evidence` | Attach evidence to a claim |
| `leamas witness evidence create` | Create evidence in an existing run bundle |
| `leamas witness evidence list` | List evidence in a run bundle |
| `leamas witness evidence show` | Show evidence's details |

All commands support:
- `--root <path>` - Root directory (default: .leamas/runs)
- `--run-id <run-id>` - Required run bundle ID
- `--json` - JSON output format

## Behavior Proved

- Claims and evidence are created as JSON files in run bundle subdirectories
- `claim.NewClaim` and `store.WriteClaim` used for claim creation
- `claim.NewEvidence` and `store.WriteEvidence` used for evidence creation
- Attach verifies both claim and evidence exist before linking
- `store.AddEvidenceToClaim` used for idempotent attachment
- Attach is idempotent (re-attaching returns "already attached")
- JSON success output uses `ok: true`
- Text output provides human-readable summaries

## Tests Added

### Claim CLI Tests (claim_cli_test.go)
- `TestWitnessClaimHelp`
- `TestWitnessClaimUnknownSubcommand`
- `TestWitnessClaimMissingSubcommand`
- `TestWitnessClaimCreateRequiresRunID`
- `TestWitnessClaimCreateRequiresID`
- `TestWitnessClaimCreateRequiresStatement`
- `TestWitnessClaimCreateCreatesClaim`
- `TestWitnessClaimCreateJSONOutput`
- `TestWitnessClaimCreateRejectsInvalidClaimID`
- `TestWitnessClaimCreateRejectsInvalidRunID`
- `TestWitnessClaimCreateRejectsMissingRunBundle`
- `TestWitnessClaimListEmpty`
- `TestWitnessClaimListShowsCreatedClaims`
- `TestWitnessClaimListJSONOutput`
- `TestWitnessClaimShowDisplaysClaim`
- `TestWitnessClaimShowJSONOutput`
- `TestWitnessClaimShowRejectsMissingClaim`
- `TestWitnessClaimShowRejectsInvalidClaimID`
- `TestWitnessClaimShowRequiresClaimID`
- `TestClaimEvidenceCLIDoesNotImportRuntimePackages`

### Evidence CLI Tests (evidence_cli_test.go)
- `TestWitnessEvidenceHelp`
- `TestWitnessEvidenceUnknownSubcommand`
- `TestWitnessEvidenceMissingSubcommand`
- `TestWitnessEvidenceCreateRequiresRunID`
- `TestWitnessEvidenceCreateRequiresID`
- `TestWitnessEvidenceCreateRequiresKind`
- `TestWitnessEvidenceCreateRequiresRole`
- `TestWitnessEvidenceCreateRequiresTitle`
- `TestWitnessEvidenceCreateRejectsBadKind`
- `TestWitnessEvidenceCreateRejectsBadRole`
- `TestWitnessEvidenceCreateRejectsUnsafeRelativePath`
- `TestWitnessEvidenceCreateCreatesEvidence`
- `TestWitnessEvidenceCreateJSONOutput`
- `TestWitnessEvidenceListEmpty`
- `TestWitnessEvidenceListShowsCreatedEvidence`
- `TestWitnessEvidenceListJSONOutput`
- `TestWitnessEvidenceShowDisplaysEvidence`
- `TestWitnessEvidenceShowJSONOutput`
- `TestWitnessEvidenceShowRejectsMissingEvidence`
- `TestWitnessEvidenceShowRejectsInvalidEvidenceID`

### Attach Tests (claim_evidence_attach_test.go)
- `TestWitnessClaimAttachEvidenceRequiresRunID`
- `TestWitnessClaimAttachEvidenceRequiresClaimID`
- `TestWitnessClaimAttachEvidenceRequiresEvidenceID`
- `TestWitnessClaimAttachEvidenceLinksEvidence`
- `TestWitnessClaimAttachEvidenceIsIdempotent`
- `TestWitnessClaimAttachEvidenceRejectsMissingClaim`
- `TestWitnessClaimAttachEvidenceRejectsMissingEvidence`
- `TestWitnessClaimAttachEvidenceJSONOutput`
- `TestWitnessClaimAttachEvidenceJSONIdempotentOutput`
- `TestWitnessClaimAttachEvidenceRejectsInvalidClaimID`
- `TestWitnessClaimAttachEvidenceRejectsInvalidEvidenceID`

## Verification Commands and Results

```bash
# ID validation tests
go test ./internal/witness/claim/... -count=1 -v

# Focused claim/evidence CLI tests
go test ./cmd/leamas/... -run 'WitnessClaim|WitnessEvidence|ClaimEvidence|ValidateClaimID|ValidateEvidenceID|ValidateRelativePath' -count=1 -v

# All cmd/leamas tests
go test ./cmd/leamas/... -v

# All tests
go test ./...

# Vet
go vet ./...

# Build
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas

# Factory gates
make factorize
make gate
```

## Skipped / Deferred

- None

## Hard Stops Honored

- No claim evaluation engine added
- No LLM scoring added
- No witness proxy persistence wiring added
- No cockpit UI added
- No network behavior added
- No database/sql imports added
- No Python added
- No shell verifier logic added
- No Node/Vite/React/npm/yarn/pnpm added

## Follow-up Candidates

1. **ACT-LEAMAS-WITNESS-PROXY-INSPECT-CLI01** (Recommended next)
   - After run bundles, claims, evidence, and CLI creation/linking exist, the next useful operator surface is inspecting witness proxy captures and turning them into evidence later.

2. ACT-LEAMAS-WEB-RUN-BUNDLE-LIST01
   - Web surface for listing run bundles.

3. ACT-LEAMAS-WEB-CLAIM-EVIDENCE-VIEW01
   - Web surface for viewing claims and evidence.

4. ACT-LEAMAS-HULK-CLAIM-EVAL-CORE01
   - Claim evaluation core engine.

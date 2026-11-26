// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_warning_codes.go holds the WARNING
// CODE constants the bounded evidence renderer publishes.
//
// The warning codes are the reviewer's diagnostic vocabulary: each one
// names a SPECIFIC reason the evidence is degraded, so a partial or
// unavailable publication is never presented as a complete one.
//
// The companion file evidence_budget.go holds the classification policy.
package digest

// WarningCode values rendered into the digest for bounded files.
const (
	WarningCodeLargeFileBounded     = "LARGE_FILE_EVIDENCE_BOUNDED"
	WarningCodeLineTooLong          = "PHYSICAL_LINE_BOUNDED"
	WarningCodeSelfOutput           = "SELF_OUTPUT_EXCLUDED"
	WarningCodeDerivedDigestBounded = "DERIVED_DIGEST_BODY_BOUNDED"
	WarningCodeDigestRecursion      = "DIGEST_RECURSION"
	// WarningCodeHistoricalEvidenceUnavailable is reported when the
	// historical blob of a file could not be recovered at either
	// endpoint of the range, so no content-driven classification was
	// possible.
	//
	// AMENDMENT-01 FINALIZATION (F1).
	WarningCodeHistoricalEvidenceUnavailable = "HISTORICAL_EVIDENCE_UNAVAILABLE"
	// WarningCodeComparisonEndpointsUnavailable is reported when the
	// historical comparison endpoints of the range could not be
	// resolved, so no patch was executed.
	//
	// FINAL AUTHORITY CORRECTION (G3): a `git diff` WITHOUT comparison
	// endpoints does not fail -- it compares the mutable worktree
	// against the index and succeeds. Such a comparison is against the
	// WRONG authority, so it is refused and the evidence is explicitly
	// unavailable.
	WarningCodeComparisonEndpointsUnavailable = "COMPARISON_ENDPOINTS_UNAVAILABLE"
)

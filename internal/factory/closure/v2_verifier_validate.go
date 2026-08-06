// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_validate.go implements the request-validation
// layer required by ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-FOUNDATION01.
//
// The validator runs before any Git observation. It rejects:
//
//   - unsupported closure protocol / plan contract versions
//   - unsupported version combinations
//   - empty required string fields (repository, S, F, C, P, M)
//   - unsafe path values for P and M (absolute, parent
//     traversal, backslash, NUL, control characters,
//     lexically unclean)
//
// The validator never reads the working tree, never invokes
// Git, and never produces a manifest or verdict. Its only
// output is a list of typed diagnostics on failure, or nil on
// success.

import (
	"strconv"
	"strings"
)

// ValidateV2ClosureVerifyRequest validates every required
// field of the supplied request. It returns nil on success
// and a non-empty V2VerifierDiagnostics slice on failure. The
// returned slice preserves the order of discovery; tests and
// CLI rendering must therefore walk it in order.
//
// Validation order is fixed:
//
//  1. version axes (closure protocol, plan contract, combo);
//  2. repository root non-empty;
//  3. required OID fields (S, F, C) non-empty;
//  4. required path fields (P, M) non-empty then path-policy.
//
// The validator never produces a manifest, never touches the
// repository, and never writes anywhere.
func ValidateV2ClosureVerifyRequest(req V2ClosureVerifyRequest) V2VerifierDiagnostics {
	var diags V2VerifierDiagnostics

	// 1. Closure protocol version.
	if !req.ClosureProtocolVersion.IsSupported() {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierUnsupportedClosureProtocolVersion,
			"closure_protocol_version "+stringOrQuote(string(req.ClosureProtocolVersion))+" is not supported",
		))
	}

	// 2. Plan contract version.
	if !req.PlanContractVersion.IsSupported() {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierUnsupportedPlanContractVersion,
			"plan_contract_version "+strconv.Itoa(int(req.PlanContractVersion))+" is not supported",
		))
	}

	// 3. Version combination (only when both axes are
	//    individually supported so the typed error code
	//    points at the offending axis, not the combo).
	if req.ClosureProtocolVersion.IsSupported() && req.PlanContractVersion.IsSupported() {
		combo := V2VersionCombination{
			PlanContract:    req.PlanContractVersion,
			ClosureProtocol: req.ClosureProtocolVersion,
		}
		if !combo.IsSupported() {
			diags = append(diags, NewV2VerifierDiagnostic(
				V2VerifierInvalidVersionCombination,
				"plan_contract_v"+strconv.Itoa(int(req.PlanContractVersion))+" + closure_protocol "+stringOrQuote(string(req.ClosureProtocolVersion))+" is not a supported combination",
			))
		}
	}

	// 4. Repository root non-empty. Note: the path is
	//    accepted as-is here; the foundation ACT does not
	//    require the path to exist on disk yet because
	//    Phase 0 establishes the public contract. ACT 4
	//    binds the path to a live repository.
	if strings.TrimSpace(req.RepositoryRoot) == "" {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierRepositoryUnavailable,
			"repository_root is required",
		).withProperty("repository_root"))
	}

	// 5. Required OID fields.
	if strings.TrimSpace(req.SubjectCommit) == "" {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierSubjectUnresolved,
			"subject_commit is required",
		).withProperty("subject_commit"))
	}
	if strings.TrimSpace(req.FreezeCommit) == "" {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierFreezeUnresolved,
			"freeze_commit is required",
		).withProperty("freeze_commit"))
	}
	if strings.TrimSpace(req.ClosureCommit) == "" {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierClosureUnresolved,
			"closure_commit is required",
		).withProperty("closure_commit"))
	}

	// 6. Plan path: non-empty + repository-relative path
	//    policy.
	if strings.TrimSpace(req.PlanPath) == "" {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierPlanPathInvalid,
			"plan_path is required",
		).withProperty("plan_path"))
	} else if msg, ok := validateV2VerifierRelativePath(req.PlanPath); !ok {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierPlanPathInvalid,
			"plan_path "+msg,
		).withProperty("plan_path"))
	}

	// 7. Manifest path: same policy.
	if strings.TrimSpace(req.ManifestPath) == "" {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierManifestPathInvalid,
			"manifest_path is required",
		).withProperty("manifest_path"))
	} else if msg, ok := validateV2VerifierRelativePath(req.ManifestPath); !ok {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierManifestPathInvalid,
			"manifest_path "+msg,
		).withProperty("manifest_path"))
	}

	if len(diags) == 0 {
		return nil
	}
	return diags
}

// withProperty returns a copy of the diagnostic with the
// supplied property name attached. The receiver is taken by
// value because V2VerifierDiagnostic is a small struct and the
// helper is invoked at most once per diagnostic.
func (d V2VerifierDiagnostic) withProperty(name string) V2VerifierDiagnostic {
	d.PropertyName = name
	return d
}

// validateV2VerifierRelativePath enforces the v1 path policy
// (see plan_path_policy.go's portablePathValidate) for both P
// and M. It returns the human-readable rejection message and
// ok=false on failure; the message is intentionally short
// because callers prepend the field name.
//
// The policy rejects:
//
//   - empty paths
//   - NUL bytes and control characters (U+0000-U+001F, U+007F)
//   - backslash
//   - leading "/" (absolute)
//   - Windows-volume prefixes
//   - empty/trailing separators
//   - parent-traversal segments ("..")
//   - lexically unclean paths (e.g. "a//b", "a/./b")
//
// Allowed segments are non-empty, non-dot, non-dotdot, and the
// canonical clean form equals the input.
func validateV2VerifierRelativePath(path string) (string, bool) {
	if path == "" {
		return "must be a non-empty path", false
	}
	if strings.ContainsRune(path, 0) {
		return "must not contain a NUL byte", false
	}
	for _, r := range path {
		if r < 0x20 || r == 0x7f {
			return "must not contain control characters", false
		}
	}
	if strings.ContainsRune(path, '\\') {
		return "must not contain a backslash", false
	}
	if path[0] == '/' {
		return "must not be an absolute path", false
	}
	// Reject Windows-volume prefix "[A-Za-z]:".
	if len(path) >= 2 && isASCIILetterByte(path[0]) && path[1] == ':' {
		return "must not carry a Windows-volume prefix", false
	}
	parts := strings.Split(path, "/")
	for _, seg := range parts {
		if seg == "" {
			return "must not contain empty separators", false
		}
		if seg == "." {
			return "must not contain single-dot segments", false
		}
		if seg == ".." {
			return "must not contain parent-traversal segments", false
		}
	}
	if clean := portablePathClean(path); clean != path {
		return "must be lexically clean", false
	}
	return "", true
}

// isASCIILetterByte reports whether b is an ASCII letter.
func isASCIILetterByte(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

// stringOrQuote renders a string for diagnostic messages.
// Empty strings render as "<empty>"; values containing
// whitespace or quotes are wrapped in double quotes; bare
// identifiers render unquoted.
func stringOrQuote(s string) string {
	if s == "" {
		return "<empty>"
	}
	if strings.ContainsAny(s, " \t\n\"") {
		return "\"" + s + "\""
	}
	return s
}

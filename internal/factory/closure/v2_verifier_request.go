// SPDX-License-Identifier: Apache-2.0

package closure

import "strings"

// v2_verifier_request.go defines the typed verifier request
// for ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-FOUNDATION01.
//
// The request is the only public input shape the v2 closure
// verifier accepts. Every field is explicit; the verifier
// never infers:
//
//   - C from HEAD
//   - M from convention
//   - P from the working tree
//
// Optional fields use `omitempty` JSON semantics so a
// committed manifest that includes the optional assertion or
// expected tag still round-trips byte-for-byte when those
// fields are absent.

// V2ClosureVerifyRequest is the typed input the v2 closure
// verifier accepts. JSON tags follow the repository
// convention (snake_case, omitempty for optional fields).
// Callers may construct the struct programmatically; the JSON
// shape is stable for CLI / API consumers.
//
// All required string fields MUST be non-empty. The verifier
// runs request validation before any Git observation so a
// missing or malformed field never reaches the resolver.
type V2ClosureVerifyRequest struct {
	// ClosureProtocolVersion is the lifecycle version
	// declared by the caller. The foundation ACT accepts
	// ClosureProtocolV2 only.
	ClosureProtocolVersion ClosureProtocolVersion `json:"closure_protocol_version"`

	// PlanContractVersion is the plan contract version
	// declared by the caller. The foundation ACT accepts
	// PlanContractV1 only.
	PlanContractVersion PlanContractVersion `json:"plan_contract_version"`

	// RepositoryRoot is the absolute path to the Git
	// repository root. The verifier binds every Git
	// operation to this path; process CWD is never used
	// as authority.
	RepositoryRoot string `json:"repository_root"`

	// SubjectCommit is the S commit OID. Required. The
	// verifier never derives it from HEAD.
	SubjectCommit string `json:"subject_commit"`

	// FreezeCommit is the F commit OID. Required. The
	// verifier never derives it from the subject.
	FreezeCommit string `json:"freeze_commit"`

	// ClosureCommit is the C commit OID. Required. The
	// verifier never derives it from HEAD or from any
	// manifest field.
	ClosureCommit string `json:"closure_commit"`

	// PlanPath is the repository-relative path to the
	// frozen plan blob at F. Required. The verifier loads
	// the plan blob exclusively from F:PlanPath; it
	// never reads the working tree.
	PlanPath string `json:"plan_path"`

	// ManifestPath is the repository-relative path to
	// the committed manifest blob at C. Required. The
	// verifier loads the manifest blob exclusively from
	// C:ManifestPath; it never reads the working tree.
	ManifestPath string `json:"manifest_path"`

	// OptionalManifestAssertion, when non-empty, is the
	// SHA-256-equivalent of the bytes the caller expects
	// the C:ManifestPath blob to contain. The verifier
	// asserts byte-equality when this field is present
	// but the assertion is never authoritative: a
	// mismatch produces a typed diagnostic, never a
	// change to the C:ManifestPath binding.
	OptionalManifestAssertion []byte `json:"optional_manifest_assertion,omitempty"`

	// ExpectedTagName, when non-empty, asserts that an
	// annotated tag with this exact name exists in the
	// target repository and targets C. ACT 4 wires the
	// tag assertion path; the foundation ACT defines
	// the field so later ACTs do not need to amend the
	// request shape.
	ExpectedTagName string `json:"expected_tag_name,omitempty"`
}

// HasOptionalManifestAssertion reports whether the request
// supplies an optional manifest byte assertion. The verifier
// treats the assertion as a non-authoritative comparison
// against C:ManifestPath bytes.
func (r V2ClosureVerifyRequest) HasOptionalManifestAssertion() bool {
	return len(r.OptionalManifestAssertion) > 0
}

// HasExpectedTag reports whether the request supplies an
// expected annotated-tag name. ACT 4 wires the tag
// assertion; the foundation ACT only carries the field
// forward.
func (r V2ClosureVerifyRequest) HasExpectedTag() bool {
	return strings.TrimSpace(r.ExpectedTagName) != ""
}

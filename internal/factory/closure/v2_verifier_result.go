// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_result.go implements Phase 8 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MANIFEST-RESULTS01:
// the deterministic V2ClosureVerification result model.
//
// The result carries the verified topology, the bound plan /
// manifest authorities, and the binary verdict booleans for
// topology / manifest identity / result-set / run success.
// The struct marshals deterministically (sorted field order
// via canonical struct field order, no map iteration) and
// computes a single Valid flag from the canonical validity
// predicate defined in the ACT 3 specification:
//
//	Valid == supported versions
//	      AND topology valid
//	      AND frozen-plan authority valid
//	      AND committed-manifest authority valid
//	      AND manifest identity valid
//	      AND result bijection valid
//	      AND successful-run integrity valid
//	      AND diagnostics empty

import (
	"encoding/json"
	"sort"
)

// V2ClosureVerification is the deterministic verification
// result published by the v2 closure verifier ACT 3 entry
// point. The struct carries:
//
//   - the version axes declared by the caller
//   - the verified S/F/C topology (OIDs + trees)
//   - the bound plan authority (path / blob / sha256)
//   - the bound manifest authority (path / blob / sha256)
//   - the four binary verdict booleans
//   - the ordered diagnostics
//
// Identical inputs always produce identical results. The
// zero value is not valid; callers must construct a result
// from a successful verification sequence.
type V2ClosureVerification struct {
	ClosureProtocolVersion ClosureProtocolVersion `json:"closure_protocol_version"`
	PlanContractVersion    PlanContractVersion    `json:"plan_contract_version"`

	RepositoryRoot string `json:"repository_root"`

	SubjectCommit string `json:"subject_commit"`
	SubjectTree   string `json:"subject_tree"`

	FreezeCommit string `json:"freeze_commit"`
	FreezeTree   string `json:"freeze_tree"`

	ClosureCommit string `json:"closure_commit"`
	ClosureTree   string `json:"closure_tree"`

	PlanPath   string `json:"plan_path"`
	PlanBlob   string `json:"plan_blob"`
	PlanSHA256 string `json:"plan_sha256"`

	ManifestPath   string `json:"manifest_path"`
	ManifestBlob   string `json:"manifest_blob"`
	ManifestSHA256 string `json:"manifest_sha256"`

	TopologyValid  bool `json:"topology_valid"`
	ManifestValid  bool `json:"manifest_valid"`
	ResultSetValid bool `json:"result_set_valid"`

	Diagnostics V2VerifierDiagnostics `json:"diagnostics,omitempty"`
	Valid       bool                  `json:"valid"`
}

// V2VerificationBuild is the validated input for
// NewV2ClosureVerification. The constructor enforces the
// canonical validity predicate and copies the inputs into a
// deterministic V2ClosureVerification.
type V2VerificationBuild struct {
	ClosureProtocolVersion ClosureProtocolVersion
	PlanContractVersion    PlanContractVersion
	RepositoryRoot         string

	SubjectCommit string
	SubjectTree   string

	FreezeCommit string
	FreezeTree   string

	ClosureCommit string
	ClosureTree   string

	PlanPath   string
	PlanBlob   string
	PlanSHA256 string

	ManifestPath   string
	ManifestBlob   string
	ManifestSHA256 string

	TopologyValid  bool
	ManifestValid  bool
	ResultSetValid bool

	Diagnostics V2VerifierDiagnostics
}

// NewV2ClosureVerification constructs the deterministic
// V2ClosureVerification. The Valid flag is computed from the
// canonical predicate:
//
//	Valid == topologyValid
//	      AND manifestValid
//	      AND resultSetValid
//	      AND diagnostics empty
//	      AND required identity fields present
//
// The constructor never coerces or fabricates fields; an
// empty required string leaves the verdict Invalid but
// non-nil so the CLI can still surface the diagnostics.
func NewV2ClosureVerification(b V2VerificationBuild) V2ClosureVerification {
	sorted := sortedDiagnostics(b.Diagnostics)

	required := b.SubjectCommit != "" &&
		b.SubjectTree != "" &&
		b.FreezeCommit != "" &&
		b.FreezeTree != "" &&
		b.ClosureCommit != "" &&
		b.ClosureTree != "" &&
		b.PlanPath != "" &&
		b.PlanBlob != "" &&
		b.PlanSHA256 != "" &&
		b.ManifestPath != "" &&
		b.ManifestBlob != "" &&
		b.ManifestSHA256 != ""

	valid := required &&
		b.TopologyValid &&
		b.ManifestValid &&
		b.ResultSetValid &&
		len(sorted) == 0

	return V2ClosureVerification{
		ClosureProtocolVersion: b.ClosureProtocolVersion,
		PlanContractVersion:    b.PlanContractVersion,
		RepositoryRoot:         b.RepositoryRoot,
		SubjectCommit:          b.SubjectCommit,
		SubjectTree:            b.SubjectTree,
		FreezeCommit:           b.FreezeCommit,
		FreezeTree:             b.FreezeTree,
		ClosureCommit:          b.ClosureCommit,
		ClosureTree:            b.ClosureTree,
		PlanPath:               b.PlanPath,
		PlanBlob:               b.PlanBlob,
		PlanSHA256:             b.PlanSHA256,
		ManifestPath:           b.ManifestPath,
		ManifestBlob:           b.ManifestBlob,
		ManifestSHA256:         b.ManifestSHA256,
		TopologyValid:          b.TopologyValid,
		ManifestValid:          b.ManifestValid,
		ResultSetValid:         b.ResultSetValid,
		Diagnostics:            sorted,
		Valid:                  valid,
	}
}

// sortedDiagnostics returns a deterministic copy of the
// supplied diagnostics. The Verifier publishes diagnostics
// in order of discovery (preserved by the V2ManifestIdentityFacts
// pipeline), but the canonical V2ClosureVerification sorts
// by (Code, PropertyName, Message) so the marshalled JSON
// is stable across re-runs even when discovery order
// varies.
//
// Sorting by Code first matches the deterministic contract
// required by the close-plan / close-manifest / CLI
// surfaces. Within a code, secondary sort keys prevent
// non-determinism when the same code appears with
// different property anchors.
func sortedDiagnostics(in V2VerifierDiagnostics) V2VerifierDiagnostics {
	out := make(V2VerifierDiagnostics, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Code != out[j].Code {
			return string(out[i].Code) < string(out[j].Code)
		}
		if out[i].PropertyName != out[j].PropertyName {
			return out[i].PropertyName < out[j].PropertyName
		}
		return out[i].Message < out[j].Message
	})
	return out
}

// MarshalJSON renders the V2ClosureVerification as
// deterministic JSON. The struct field tags already enforce
// a stable field order; the only source of non-determinism
// was the diagnostics slice, which sortedDiagnostics
// resolves.
func (v V2ClosureVerification) MarshalJSON() ([]byte, error) {
	type alias V2ClosureVerification
	return json.Marshal(alias(v))
}

package closure

import "encoding/json"

// plan_contract_v2_lifecycle.go centralises the Closure Protocol
// v2 lifecycle constants, topology validators, and version
// dispatch logic. Splitting this from the v1 chain keeps every
// file under the LLM-friendly 400-line threshold while preserving
// the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.
//
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-SUBJECT-THEN-FREEZE01
// introduces the opposite-direction topology where the subject
// commit S is a strict ancestor of the freeze commit F, the
// closure C is a strict descendant of F, and checks execute
// against S^{tree} with frozen plan bytes loaded only from F.
//
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 is preserved
// unchanged: the v1 direction is still freeze F ancestor of
// subject S, with checks executing against F^{tree}.

// ClosureProtocolVersion is the explicit lifecycle version
// dispatched before topology validation. v1 and v2 are the
// only supported values.
type ClosureProtocolVersion string

const (
	// ClosureProtocolV1 is the original "freeze ancestor of
	// subject" direction (ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01).
	ClosureProtocolV1 ClosureProtocolVersion = "1"
	// ClosureProtocolV2 is the new "subject ancestor of freeze"
	// direction introduced by
	// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-SUBJECT-THEN-FREEZE01.
	ClosureProtocolV2 ClosureProtocolVersion = "2"
)

// SupportedClosureProtocolVersions is the closed set of lifecycle
// versions the runner accepts. Any other value produces
// unsupported_closure_protocol_version.
func SupportedClosureProtocolVersions() []ClosureProtocolVersion {
	return []ClosureProtocolVersion{ClosureProtocolV1, ClosureProtocolV2}
}

// IsSupported reports whether v is a known lifecycle version.
func (v ClosureProtocolVersion) IsSupported() bool {
	for _, s := range SupportedClosureProtocolVersions() {
		if s == v {
			return true
		}
	}
	return false
}

// V2Topology captures the three lifecycle anchors of
// Closure Protocol v2. Subject, freeze, and closure are
// pairwise distinct; S is a strict ancestor of F; F is a
// strict ancestor of C.
type V2Topology struct {
	SubjectCommit string
	SubjectTree   string
	FreezeCommit  string
	FreezeTree    string
	ClosureCommit string
	ClosureTree   string
}

// V2DispatchResult reports the outcome of v1/v2 topology
// dispatch. The dispatch decision is made from the explicit
// ClosureProtocolVersion field; it is never inferred from
// ancestry direction or from plan contract version.
type V2DispatchResult struct {
	Version   ClosureProtocolVersion
	Accepted  bool
	Reason    string
	SubjectOK bool
	FreezeOK  bool
}

// V2DispatchTopology chooses the topology rule from the explicit
// lifecycle version. The function never reads the plan contract
// version and never infers the lifecycle version from ancestry
// direction.
//
//	v1: isAncestor(F, S) and F != S
//	v2: isAncestor(S, F) and S != F
func V2DispatchTopology(v ClosureProtocolVersion, isAncestor bool, equal bool) V2DispatchResult {
	if !v.IsSupported() {
		return V2DispatchResult{
			Version:  v,
			Accepted: false,
			Reason:   "unsupported_closure_protocol_version",
		}
	}
	if equal {
		return V2DispatchResult{
			Version:   v,
			Accepted:  false,
			SubjectOK: false,
			FreezeOK:  false,
			Reason:    "subject_equals_freeze",
		}
	}
	switch v {
	case ClosureProtocolV1:
		if !isAncestor {
			return V2DispatchResult{
				Version:   v,
				Accepted:  false,
				SubjectOK: false,
				FreezeOK:  true,
				Reason:    "freeze_not_ancestor_of_subject",
			}
		}
		return V2DispatchResult{
			Version:   v,
			Accepted:  true,
			SubjectOK: true,
			FreezeOK:  true,
		}
	case ClosureProtocolV2:
		if !isAncestor {
			return V2DispatchResult{
				Version:   v,
				Accepted:  false,
				SubjectOK: true,
				FreezeOK:  false,
				Reason:    "subject_not_ancestor_of_freeze",
			}
		}
		return V2DispatchResult{
			Version:   v,
			Accepted:  true,
			SubjectOK: true,
			FreezeOK:  true,
		}
	}
	return V2DispatchResult{
		Version:  v,
		Accepted: false,
		Reason:   "unsupported_closure_protocol_version",
	}
}

// V2Request is the normalised v2 lifecycle request. The
// plan contract version is intentionally separate from
// closure_protocol_version so a Plan Contract v1 document
// can be used with Closure Protocol v2.
type V2Request struct {
	ClosureProtocolVersion ClosureProtocolVersion `json:"closure_protocol_version"`
	PlanContractVersion    int                    `json:"plan_contract_version"`
	RepositoryRoot         string                 `json:"repository_root"`
	SubjectCommit          string                 `json:"subject_commit"`
	FreezeCommit           string                 `json:"freeze_commit"`
	PlanPath               string                 `json:"plan_path"`
	// OptionalWorkingPlanAssertion, when non-empty, must
	// match the disk plan bytes. A mismatch produces
	// working_plan_mismatch.
	OptionalWorkingPlanAssertion string `json:"optional_working_plan_assertion,omitempty"`
	// EvidenceDirectory is where the runner records
	// captured evidence bundles.
	EvidenceDirectory string `json:"evidence_directory,omitempty"`
	// ManifestOutput is the path the runner writes the v2
	// manifest to.
	ManifestOutput string `json:"manifest_output,omitempty"`
}

// V2Manifest is the Closure Protocol v2 manifest. Both version
// axes are bound at the top level so a v1 consumer cannot
// silently consume a v2 manifest.
//
// The manifest makes subject, freeze, caller HEAD, and later
// closure commit unambiguous. Plan bytes and SHA-256 are
// bound to the frozen F:PLAN_PATH, never the disk.
type V2Manifest struct {
	ClosureProtocolVersion ClosureProtocolVersion `json:"closure_protocol_version"`
	PlanContractVersion    int                    `json:"plan_contract_version"`
	SubjectCommit          string                 `json:"subject_commit"`
	SubjectTree            string                 `json:"subject_tree"`
	FreezeCommit           string                 `json:"freeze_commit"`
	FreezeTree             string                 `json:"freeze_tree"`
	PlanPath               string                 `json:"plan_path"`
	PlanBlob               string                 `json:"plan_blob"`
	PlanSHA256             string                 `json:"plan_sha256"`
	ExecutionTree          string                 `json:"execution_tree"`
	CheckResults           []V2CheckResult        `json:"check_results,omitempty"`
	LeamasBinaryIdentity   V2BinaryIdentity       `json:"leamas_binary_identity"`
	CallerHead             string                 `json:"caller_head"`
}

// V2CheckResult is the per-check result bound into the v2
// manifest. Results are emitted in frozen-plan order, and the
// fields retain the execution authority needed to distinguish a
// completed run from an exclusion or a skipped check.
type V2CheckResult struct {
	ID                      string                `json:"id"`
	Mode                    string                `json:"mode"`
	Outcome                 string                `json:"outcome"`
	ExitCode                *int                  `json:"exit_code,omitempty"`
	DurationMS              int64                 `json:"duration_ms"`
	ExecutionClassification string                `json:"execution_classification"`
	CleanupStatus           string                `json:"cleanup_status"`
	Evidence                []V2EvidenceReference `json:"evidence,omitempty"`
	Detail                  string                `json:"detail,omitempty"`
}

// V2EvidenceReference binds a manifest result to one detached
// stdout/stderr record without copying the record payload into the
// core manifest.
type V2EvidenceReference struct {
	LogicalName  string `json:"logical_name"`
	MediaType    string `json:"media_type"`
	SHA256       string `json:"sha256"`
	ByteCount    int64  `json:"byte_count"`
	Availability string `json:"availability"`
}

// V2BinaryIdentity records the Leamas binary that produced
// the v2 manifest. VCSRevision is the full commit OID; the
// runner derives it via the same identity the v1 manifest
// already records. LeamasVersion is the stamped semantic
// version of the binary, matching `leamas version`.
type V2BinaryIdentity struct {
	Path          string `json:"path"`
	SHA256        string `json:"sha256"`
	VCSRevision   string `json:"vcs_revision"`
	VCSModified   bool   `json:"vcs_modified"`
	LeamasVersion string `json:"leamas_version"`
}

// RunningLeamasVersion returns the stamped Leamas version for
// the running binary. The implementation lives in the version
// package and is wired here via a thin shim to avoid an
// import cycle.
var RunningLeamasVersion = func() string { return "" }

// RunningLeamasVCSRevision returns the full commit OID for the
// running binary. See RunningLeamasVersion for the rationale.
var RunningLeamasVCSRevision = func() string { return "" }

// RunningLeamasVCSModified returns the dirty-flag for the
// running binary. See RunningLeamasVersion for the rationale.
var RunningLeamasVCSModified = func() bool { return false }

// V2ManifestJSON renders the v2 manifest as a deterministic
// JSON byte string. The helper exists so callers that need a
// stable text rendering (evidence bundles, audit trails) do
// not accidentally pick up a non-deterministic field order.
func (m V2Manifest) V2ManifestJSON() ([]byte, error) {
	return json.Marshal(m)
}

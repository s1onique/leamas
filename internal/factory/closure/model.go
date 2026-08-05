// Package closure implements Closure Protocol v1.
package closure

import (
	"bytes"
	"encoding/json"
)

const (
	ContractVersionV1           = 1
	CheckModeRun                = "run"
	CheckModeExclude            = "exclude"
	VerdictPass                 = "pass"
	VerdictFail                 = "fail"
	CheckStatusPass             = "pass"
	CheckStatusFail             = "fail"
	CheckStatusNotRun           = "not_run_due_to_prior_failure"
	ArtifactStatusPass          = "pass"
	ArtifactStatusMissing       = "missing"
	ArtifactStatusFail          = "fail"
	CleanupPass                 = "pass"
	CleanupFailed               = "failed"
	CleanupNotRequired          = "not_required"
	LifecycleImplemented        = "IMPLEMENTED"
	LifecycleVerified           = "VERIFIED"
	LifecycleClosedLocal        = "CLOSED_LOCAL"
	LifecyclePublished          = "PUBLISHED"
	LifecycleDownstreamAccepted = "DOWNSTREAM_ACCEPTED"
)

// ExecutionSerialFailFast is retained as a deprecated alias of
// ExecutionModeSerialFailFast. New code MUST use the typed
// ExecutionMode value and ParseExecutionMode; this constant remains
// only so existing string-typed call sites keep compiling while the
// canonical contract is migrated.
const ExecutionSerialFailFast = string(ExecutionModeSerialFailFast)

type Plan struct {
	ContractVersion int              `json:"contract_version"`
	ActID           string           `json:"act_id"`
	Baseline        Baseline         `json:"baseline"`
	Execution       PlanExecution    `json:"execution"`
	Checks          []PlanCheck      `json:"checks"`
	Artifacts       []PlanArtifact   `json:"artifacts"`
	Policy          PlanPolicy       `json:"policy"`
	PolicyProfile   string           `json:"policy_profile,omitempty"`
	RunnerBinding   string           `json:"runner_binding,omitempty"`
	RunnerAuthority *RunnerAuthority `json:"runner_authority,omitempty"`
}

// RunnerAuthorityMode represents the authority mode for runner identity binding.
type RunnerAuthorityMode string

const (
	// RunnerAuthoritySubjectExact uses the subject_exact mode where
	// runner vcs.revision must equal target subject S.
	RunnerAuthoritySubjectExact RunnerAuthorityMode = "subject_exact"
	// RunnerAuthorityToolReleaseExact uses the tool_release_exact mode where
	// runner vcs.revision must equal the pinned tool revision (independent of target).
	RunnerAuthorityToolReleaseExact RunnerAuthorityMode = "tool_release_exact"
)

// ToolAuthority declares the exact tool identity for tool_release_exact mode.
type ToolAuthority struct {
	// Revision is the full lowercase 40-character Git OID of the Leamas
	// source revision from which the runner was built.
	Revision string `json:"revision"`
	// TreeOID is the full lowercase Git OID of the Leamas source tree.
	TreeOID string `json:"tree_oid,omitempty"`
	// BinarySHA256 is the lowercase SHA-256 hex digest of the runner binary.
	BinarySHA256 string `json:"binary_sha256"`
	// Version is the declared Leamas version string.
	Version string `json:"version,omitempty"`
	// TagName is the annotated release tag name.
	TagName string `json:"tag_name,omitempty"`
	// TagObjectOID is the annotated tag object OID.
	TagObjectOID string `json:"tag_object_oid,omitempty"`
}

// RunnerAuthority declares the runner authority mode and tool identity.
// For subject_exact mode, the tool block is optional (but not allowed in
// strict mode). For tool_release_exact mode, the tool block is required.
type RunnerAuthority struct {
	Mode RunnerAuthorityMode `json:"mode"`
	Tool *ToolAuthority      `json:"tool,omitempty"`
}

type Baseline struct {
	CommitOID string `json:"commit_oid"`
	TreeOID   string `json:"tree_oid"`
}

// PlanExecution is the canonical Closure Protocol v1 execution
// descriptor. The Mode field is a pointer so the strict decoder can
// distinguish:
//
//   - the property being absent (Mode == nil) — reported as
//     ExecutionModeMissing;
//   - the property being present with an empty string
//     (Mode != nil, *Mode == "") — reported as
//     ExecutionModePresentEmpty;
//   - the property being present with a non-canonical value
//     (Mode != nil, *Mode != ExecutionModeSerialFailFast) — reported
//     as ExecutionModePresentUnknown.
//
// Tests and producers that build a PlanExecution struct literal MUST
// pass an explicit pointer (see PlanExecutionMode helper below).
type PlanExecution struct {
	Mode *ExecutionMode `json:"mode"`
}

// NewPlanExecution constructs a PlanExecution whose Mode is set to the
// supplied ExecutionMode. Pass an empty ExecutionMode to represent
// "field omitted"; callers that want to distinguish absent from empty
// must use the pointer directly.
func NewPlanExecution(mode ExecutionMode) PlanExecution {
	if mode == "" {
		return PlanExecution{}
	}
	m := mode
	return PlanExecution{Mode: &m}
}

// PlanExecutionFromString builds a PlanExecution from a raw string,
// wrapping every case — including the empty string — in a non-nil
// pointer. This is the helper intended for producer code paths that
// read a textual mode and want to defer parsing to the validator.
func PlanExecutionFromString(raw string) PlanExecution {
	m := ExecutionMode(raw)
	return PlanExecution{Mode: &m}
}

type PlanCheck struct {
	ID               string            `json:"id"`
	Mode             string            `json:"mode"`
	Argv             []string          `json:"argv,omitempty"`
	WorkingDirectory string            `json:"working_directory,omitempty"`
	TimeoutSeconds   int               `json:"timeout_seconds,omitempty"`
	Environment      map[string]string `json:"environment"`
	Reason           string            `json:"reason,omitempty"`
}

// UnmarshalJSON decodes a PlanCheck from the public wire shape.
//
// CORRECTION08+09: the runtime timeout decoder uses the
// production exact-number authority (see plan_exact_number.go)
// and emits typed TimeoutDecodeError failures. The decoder
// accepts 1, 1.0, 1e0, 600, 600.0, 6e2, and any other
// mathematically integral JSON number in the inclusive
// [1, 600] range. Non-integral numbers, numbers outside
// the range, and non-numbers are rejected with a typed
// error. Genuine JSON syntax errors are propagated from
// the decoder and not fabricated.
func (c *PlanCheck) UnmarshalJSON(data []byte) error {
	type alias PlanCheck
	var raw struct {
		alias
		TimeoutSecondsRaw json.RawMessage `json:"timeout_seconds,omitempty"`
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		return err
	}
	c2 := PlanCheck(raw.alias)
	c2.TimeoutSeconds = 0
	if len(raw.TimeoutSecondsRaw) > 0 {
		// CORRECTION09: timeout_seconds must be a JSON number.
		// json.RawMessage preserves the original token; we
		// validate that the first non-whitespace byte is a
		// digit or minus, the structural marker for a JSON
		// number. A quoted string begins with a double quote.
		// Anything else (true, false, null, [, {) fails here.
		trimmed := bytes.TrimSpace(raw.TimeoutSecondsRaw)
		if len(trimmed) == 0 || (trimmed[0] != '-' && (trimmed[0] < '0' || trimmed[0] > '9')) {
			return &TimeoutDecodeError{Kind: "non_number", Raw: string(trimmed)}
		}
		authority := NewExactNumberAuthority()
		v, derr := authority.DecodeTimeout(string(trimmed))
		if derr != nil {
			return derr
		}
		c2.TimeoutSeconds = int(v)
	}
	*c = c2
	return nil
}

type PlanArtifact struct {
	ID        string       `json:"id"`
	Path      string       `json:"path"`
	Required  *bool        `json:"required"`
	MaxBytes  int64        `json:"max_bytes"`
	MediaType string       `json:"media_type"`
	Role      ArtifactRole `json:"role,omitempty"`
}

type PlanPolicy struct {
	RequireCleanBefore       *bool `json:"require_clean_before"`
	RequireCleanAfter        *bool `json:"require_clean_after"`
	ForbidTrackedFullDigests *bool `json:"forbid_tracked_full_digests"`
	RequireDiffCheck         *bool `json:"require_diff_check"`
}

type Manifest struct {
	ContractVersion  int                 `json:"contract_version"`
	ActID            string              `json:"act_id"`
	Plan             ManifestPlanRef     `json:"plan"`
	PlanFreeze       ManifestPlanFreeze  `json:"plan_freeze"`
	Subject          ManifestSubject     `json:"subject"`
	Runner           RunnerIdentity      `json:"runner"`
	Repository       RepositoryIdentity  `json:"repository"`
	Checks           []CheckResult       `json:"checks"`
	Artifacts        []ArtifactResult    `json:"artifacts"`
	DetachedEvidence []EvidenceRecord    `json:"detached_evidence"`
	PatchHygiene     PatchHygiene        `json:"patch_hygiene"`
	ClosurePolicy    ClosurePolicyResult `json:"closure_policy"`
	ExcludedChecks   []ExcludedCheck     `json:"excluded_checks"`
	Verdict          string              `json:"verdict"`
	Tag              string              `json:"tag,omitempty"`
}

type ManifestPlanFreeze struct {
	FreezeCommit  string `json:"freeze_commit"`
	FreezeTree    string `json:"freeze_tree,omitempty"`
	PlanPath      string `json:"plan_path"`
	PlanBlobOID   string `json:"plan_blob_oid"`
	PlanSHA256    string `json:"plan_sha256"`
	SubjectCommit string `json:"subject_commit"`
}

type ManifestPlanRef struct {
	SHA256 string `json:"sha256"`
	Path   string `json:"path"`
	Bytes  string `json:"bytes,omitempty"`
}

type ManifestSubject struct {
	CommitOID string `json:"commit_oid"`
	TreeOID   string `json:"tree_oid"`
}

type RunnerIdentity struct {
	LeamasVersion string `json:"leamas_version"`
	BinarySHA256  string `json:"binary_sha256"`
	VCSRevision   string `json:"vcs_revision"`
	VCSModified   bool   `json:"vcs_modified"`
}

type ManifestRunner struct {
	Binding string `json:"binding"`
}
type _manifestRunnerMarker struct{}
type _planRunnerBindingMarker struct{}
type _planPolicyProfileMarker struct{}

type RepositoryIdentity struct {
	Root                   string `json:"root"`
	RemoteURL              string `json:"remote_url,omitempty"`
	Branch                 string `json:"branch"`
	HeadCommitOID          string `json:"head_commit_oid"`
	HeadTreeOID            string `json:"head_tree_oid"`
	OriginMainCommitOID    string `json:"origin_main_commit_oid,omitempty"`
	AheadBy                *int   `json:"ahead_by,omitempty"`
	BehindBy               *int   `json:"behind_by,omitempty"`
	WorkingTreeCleanBefore bool   `json:"working_tree_clean_before"`
	WorkingTreeCleanAfter  bool   `json:"working_tree_clean_after"`
}

type CheckResult struct {
	CheckID               string   `json:"check_id"`
	SubjectTreeOID        string   `json:"subject_tree_oid"`
	Argv                  []string `json:"argv"`
	WorkingDirectory      string   `json:"working_directory"`
	OverriddenEnvironment []string `json:"overridden_environment"`
	StartedAtUTC          string   `json:"started_at_utc,omitempty"`
	FinishedAtUTC         string   `json:"finished_at_utc,omitempty"`
	DurationMS            int64    `json:"duration_ms"`
	ExitCode              *int     `json:"exit_code"`
	Status                string   `json:"status"`
	StdoutSHA256          string   `json:"stdout_sha256,omitempty"`
	StdoutByteCount       int64    `json:"stdout_byte_count"`
	StderrSHA256          string   `json:"stderr_sha256,omitempty"`
	StderrByteCount       int64    `json:"stderr_byte_count"`
	OutputTruncated       bool     `json:"output_truncated"`
	OutputIncomplete      bool     `json:"output_incomplete"`
	OutputBytesObserved   int64    `json:"output_bytes_observed"`
	CleanupStatus         string   `json:"cleanup_status"`
	ExecutionErrorCode    string   `json:"execution_error_code,omitempty"`
}

type ArtifactResult struct {
	ArtifactID string       `json:"artifact_id"`
	Path       string       `json:"path"`
	Required   bool         `json:"required"`
	MediaType  string       `json:"media_type"`
	Role       ArtifactRole `json:"role,omitempty"`
	Status     string       `json:"status"`
	SHA256     string       `json:"sha256,omitempty"`
	ByteCount  int64        `json:"byte_count"`
	Diagnostic string       `json:"diagnostic,omitempty"`
}

type EvidenceRecord struct {
	LogicalName  string `json:"logical_name"`
	MediaType    string `json:"media_type"`
	SHA256       string `json:"sha256"`
	ByteCount    int64  `json:"byte_count"`
	Availability string `json:"availability"`
}

// DetachedEvidenceIndex binds a sidecar file in the canonical evidence
// directory to the core manifest. The core manifest MUST reference every
// detached sidecar through this index; the sidecar MUST validate against
// the bound identity.
type DetachedEvidenceIndex struct {
	Path          string `json:"path"`
	SchemaVersion int    `json:"schema_version"`
	MediaType     string `json:"media_type"`
	SHA256        string `json:"sha256"`
	ByteCount     int64  `json:"byte_count"`
	ItemCount     int    `json:"item_count"`
}

// CheckSummary replaces per-record detached evidence in the core manifest.
// It carries only aggregate identity and the bound sidecar index.
type CheckSummary struct {
	CheckID             string `json:"check_id"`
	SubjectTreeOID      string `json:"subject_tree_oid"`
	Status              string `json:"status"`
	ExitCode            *int   `json:"exit_code,omitempty"`
	DurationMS          int64  `json:"duration_ms"`
	OutputBytesObserved int64  `json:"output_bytes_observed"`
	OutputTruncated     bool   `json:"output_truncated"`
	OutputIncomplete    bool   `json:"output_incomplete"`
	CleanupStatus       string `json:"cleanup_status"`
}

type PatchHygiene struct {
	Status          string `json:"status"`
	DiagnosticCount int    `json:"diagnostic_count"`
}

type ClosurePolicyResult struct {
	TrackedFullDigestStatus string `json:"tracked_full_digest_status"`
	DiagnosticCount         int    `json:"diagnostic_count"`
}

type ExcludedCheck struct {
	CheckID        string `json:"check_id"`
	SubjectTreeOID string `json:"subject_tree_oid"`
	Reason         string `json:"reason"`
}

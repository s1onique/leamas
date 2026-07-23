// Package closure implements Closure Protocol v1.
package closure

const (
	ContractVersionV1           = 1
	ExecutionSerialFailFast     = "serial_fail_fast"
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

type Plan struct {
	ContractVersion int            `json:"contract_version"`
	ActID           string         `json:"act_id"`
	Baseline        Baseline       `json:"baseline"`
	Execution       PlanExecution  `json:"execution"`
	Checks          []PlanCheck    `json:"checks"`
	Artifacts       []PlanArtifact `json:"artifacts"`
	Policy          PlanPolicy     `json:"policy"`
}

type Baseline struct {
	CommitOID string `json:"commit_oid"`
	TreeOID   string `json:"tree_oid"`
}

type PlanExecution struct {
	Mode string `json:"mode"`
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

type PlanArtifact struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	Required  *bool  `json:"required"`
	MaxBytes  int64  `json:"max_bytes"`
	MediaType string `json:"media_type"`
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
}

type ManifestPlanRef struct {
	SHA256 string `json:"sha256"`
	Path   string `json:"path"`
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
	ArtifactID string `json:"artifact_id"`
	Path       string `json:"path"`
	Required   bool   `json:"required"`
	MediaType  string `json:"media_type"`
	Status     string `json:"status"`
	SHA256     string `json:"sha256,omitempty"`
	ByteCount  int64  `json:"byte_count"`
	Diagnostic string `json:"diagnostic,omitempty"`
}

type EvidenceRecord struct {
	LogicalName  string `json:"logical_name"`
	MediaType    string `json:"media_type"`
	SHA256       string `json:"sha256"`
	ByteCount    int64  `json:"byte_count"`
	Availability string `json:"availability"`
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

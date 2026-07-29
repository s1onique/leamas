// SPDX-License-Identifier: Apache-2.0

package verifierauthority

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/s1onique/leamas/internal/execution"
)

// ExecutionAuthority classifies whether a verifier may execute in a given context.
type ExecutionAuthority string

const (
	// AuthorityLocalSafe permits execution in any local context.
	AuthorityLocalSafe ExecutionAuthority = "local_safe"

	// AuthorityCIExactCheckout permits execution only in GitHub Actions with
	// an exact-checkout of the verified commit. This requires:
	//   - CI == "true"
	//   - GITHUB_ACTIONS == "true"
	//   - authority marker matches configured value
	//   - GITHUB_SHA is a valid full commit OID
	//   - git HEAD == GITHUB_SHA
	//   - worktree is clean
	//   - GITHUB_WORKSPACE Git top-level == repository Git top-level
	AuthorityCIExactCheckout ExecutionAuthority = "ci_exact_checkout"
)

// VerifierOperation classifies the type of operation being performed.
type VerifierOperation string

const (
	// OperationVerify is a read-only verification operation.
	OperationVerify VerifierOperation = "verify"

	// OperationUpdateBaseline is a mutation operation that updates the baseline.
	OperationUpdateBaseline VerifierOperation = "update_baseline"
)

// AuthorityError is the canonical error when authority is denied.
type AuthorityError struct {
	VerifierID        string
	RequiredAuthority ExecutionAuthority
	Operation         VerifierOperation
	ReasonCode        string
	Message           string
}

func (e *AuthorityError) Error() string {
	return e.Message
}

// IsAuthorityError reports whether err originated from this package.
func IsAuthorityError(err error) bool {
	var ae *AuthorityError
	return errors.As(err, &ae)
}

// Reason codes for structured denial diagnostics.
const (
	ReasonCodeMissingCI              = "missing_ci"
	ReasonCodeMissingGitHubActions   = "missing_github_actions"
	ReasonCodeMissingAuthorityMarker = "missing_authority_marker"
	ReasonCodeWrongAuthorityMarker   = "wrong_authority_marker"
	ReasonCodeMissingSHA             = "missing_sha"
	ReasonCodeMalformedSHA           = "malformed_sha"
	ReasonCodeHeadMismatch           = "head_mismatch"
	ReasonCodeDirtyTree              = "dirty_tree"
	ReasonCodeWorkspaceMismatch      = "workspace_mismatch"
	ReasonCodeGitHeadFailed          = "git_head_failed"
	ReasonCodeGitStatusFailed        = "git_status_failed"
	ReasonCodeGitRepoRootFailed      = "git_repo_root_failed"
	ReasonCodeGitWorkspaceRootFailed = "git_workspace_root_failed"
	ReasonCodeOperationDenied        = "operation_denied"
	ReasonCodeUnknownAuthority       = "unknown_authority"
	ReasonCodeEmptyAuthority         = "empty_authority"
)

// ExecutionContext captures all observable signals for authority decisions.
//
// The unexported `observation` field carries provenance: only the
// authority-package observation functions can set it, so external
// callers cannot manufacture a local classification by populating
// exported fields.
type ExecutionContext struct {
	CI              string
	GitHubActions   string
	AuthorityMarker string
	GitHubSHA       string
	GitHubWorkspace string
	RepositoryRoot  string

	HeadCommit     string
	WorktreeStatus string
	WorkspaceRoot  string

	HeadErr           error
	StatusErr         error
	RepositoryRootErr error
	WorkspaceRootErr  error

	observation executionObservation
}

// GitObservation captures the result of Git status reads.
type GitObservation struct {
	Head      string
	Status    string
	HeadErr   error
	StatusErr error
}

// WorkspaceObservation captures the result of Git workspace resolution.
type WorkspaceObservation struct {
	Root    string
	RootErr error
}

// fullCommitOIDRegex matches a 40-character SHA-1 or 64-character SHA-256 OID.
var fullCommitOIDRegex = regexp.MustCompile(`^[0-9a-f]{40}$|^[0-9a-f]{64}$`)

// LookupEnv is the function type for environment variable lookup.
// os.Getenv is the default; tests inject a fake.
type LookupEnv func(string) string

// DefaultLookupEnv is the production environment lookup.
func DefaultLookupEnv(key string) string { return os.Getenv(key) }

// EnvVars for authority checks.
var (
	EnvCI              = "CI"
	EnvGitHubActions   = "GITHUB_ACTIONS"
	EnvAuthorityMarker = "LEAMAS_DUPCODE_AUTHORITY"
	EnvGitHubSHA       = "GITHUB_SHA"
	EnvGitHubWorkspace = "GITHUB_WORKSPACE"
)

// AuthorityMarker is the required value of LEAMAS_DUPCODE_AUTHORITY.
const AuthorityMarker = "github-actions"

// positiveTruthy returns true only when value is the canonical truthy
// string. Empty, "false", or any other value is not positive.
func positiveTruthy(value string) bool {
	return value == "true"
}

// hasPositiveCISignal returns true when the context carries a positive
// CI or GitHub Actions signal. This is the source of truth for
// "this is a CI environment".
func (ec ExecutionContext) hasPositiveCISignal() bool {
	return positiveTruthy(ec.CI) || positiveTruthy(ec.GitHubActions)
}

// isWellFormedEnv returns true when the marker / SHA / workspace fields
// are either empty (absent) or syntactically valid. A malformed value
// disqualifies a local classification.
func (ec ExecutionContext) isWellFormedEnv() bool {
	if ec.AuthorityMarker != "" && ec.AuthorityMarker != AuthorityMarker {
		return false
	}
	if ec.GitHubSHA != "" && !fullCommitOIDRegex.MatchString(ec.GitHubSHA) {
		return false
	}
	return true
}

// observationSummary returns a structured view of the context's
// observation. It is package-private so only the verifierauthority
// package can read it.
func (ec ExecutionContext) observationSummary() executionObservation {
	return ec.observation
}

// ClassifyExecutionEnvironment returns the classified environment
// kind for the observed context. The classification rules are:
//
//	GITHUB_ACTIONS=true                    -> EnvironmentGitHubActions
//	CI=true without GITHUB_ACTIONS=true    -> EnvironmentCI
//	successful local observation + no positive CI signal + well-formed env
//	                                        -> EnvironmentLocal
//	observation failure, malformed, contradictory, or incomplete
//	                                        -> EnvironmentUnknown
//
// External callers cannot manufacture EnvironmentLocal by populating
// exported fields: only the authority-package observation functions
// can set the unexported observation provenance.
func ClassifyExecutionEnvironment(ec ExecutionContext) ExecutionEnvironmentKind {
	// GitHub Actions detection is intentionally independent of CI
	// because GitHub documents CI as overridable while the GITHUB_*
	// variables are reserved defaults.
	if positiveTruthy(ec.GitHubActions) {
		return EnvironmentGitHubActions
	}

	// Generic CI signal.
	if positiveTruthy(ec.CI) {
		return EnvironmentCI
	}

	// Local observation: only the authority package can mark an
	// observation as completed-and-local.
	if ec.observation.completed && ec.observation.local && ec.isWellFormedEnv() {
		return EnvironmentLocal
	}

	return EnvironmentUnknown
}

// ObservationCompleted reports whether the unexported observation
// provenance records a completed observation. It is package-private.
func (ec ExecutionContext) ObservationCompleted() bool {
	return ec.observation.completed
}

// DetectExecutionContext reads the production environment and repository state
// to assemble an ExecutionContext. All Git observations use execution.RunGit
// for bounded execution with fail-closed behavior.
//
// The returned context is classified as EnvironmentLocal when the
// observation completes successfully, no positive CI / GitHub Actions
// signal is present, and the environment fields are well-formed. A
// dirty worktree is allowed; only failed Git subprocesses disqualify
// the local classification.
func DetectExecutionContext(ctx context.Context, repoRoot string) ExecutionContext {
	ec := ExecutionContext{
		CI:              DefaultLookupEnv(EnvCI),
		GitHubActions:   DefaultLookupEnv(EnvGitHubActions),
		AuthorityMarker: DefaultLookupEnv(EnvAuthorityMarker),
		GitHubSHA:       DefaultLookupEnv(EnvGitHubSHA),
		GitHubWorkspace: DefaultLookupEnv(EnvGitHubWorkspace),
		RepositoryRoot:  repoRoot,
	}

	headResult, headErr := execution.RunGit(ctx, repoRoot, "rev-parse", "HEAD^{commit}")
	headOK := headErr == nil
	if headErr != nil {
		ec.HeadErr = fmt.Errorf("git rev-parse HEAD: %w", headErr)
	} else {
		ec.HeadCommit = strings.TrimSpace(string(headResult.Stdout))
	}

	statusResult, statusErr := execution.RunGit(ctx, repoRoot, "status", "--porcelain=v1")
	statusOK := statusErr == nil
	if statusErr != nil {
		ec.StatusErr = fmt.Errorf("git status: %w", statusErr)
	} else {
		ec.WorktreeStatus = strings.TrimSpace(string(statusResult.Stdout))
	}

	repoRootResult, repoRootErr := execution.RunGit(ctx, repoRoot, "rev-parse", "--show-toplevel")
	repoOK := repoRootErr == nil
	if repoRootErr != nil {
		ec.RepositoryRootErr = fmt.Errorf("git rev-parse --show-toplevel: %w", repoRootErr)
	} else {
		ec.RepositoryRoot = strings.TrimSpace(string(repoRootResult.Stdout))
	}

	workspaceLookup := ec.GitHubWorkspace != ""
	workspaceOK := true
	if workspaceLookup {
		workspaceResult, workspaceErr := execution.RunGit(ctx, ec.GitHubWorkspace, "rev-parse", "--show-toplevel")
		if workspaceErr != nil {
			ec.WorkspaceRootErr = fmt.Errorf("git rev-parse --show-toplevel: %w", workspaceErr)
			workspaceOK = false
		} else {
			ec.WorkspaceRoot = strings.TrimSpace(string(workspaceResult.Stdout))
		}
	}

	ec.observation = recordLocalObservation(headOK, statusOK, repoOK, workspaceLookup, workspaceOK)

	return ec
}

// NewLocalOnlyContext returns an ExecutionContext classified as
// EnvironmentLocal by ClassifyExecutionEnvironment. The classification
// is set via the package-private observation provenance, so callers
// outside the authority package cannot reach EnvironmentLocal by
// writing exported fields.
//
// NewLocalOnlyContext is the only authority-package entry point that
// produces a local classification without Git observation; it exists
// for callers that already know the context is local and want to skip
// the Git observation round-trip. The canonical production path uses
// DetectExecutionContext, which records the observation through
// recordLocalObservation.
func NewLocalOnlyContext() *ExecutionContext {
	return &ExecutionContext{
		observation: recordLocalObservation(true, true, true, false, true),
	}
}

// newLocalContextFromObservedEnv is the package-internal helper used
// by tests that need a local-classified context built from a known
// observation status. Production callers must use DetectExecutionContext
// or NewLocalOnlyContext.
func newLocalContextFromObservedEnv(headOK, statusOK, repoOK, workspaceLookup, workspaceOK bool) *ExecutionContext {
	return &ExecutionContext{
		observation: recordLocalObservation(headOK, statusOK, repoOK, workspaceLookup, workspaceOK),
	}
}

// ValidateAuthority checks whether the execution context permits the given
// authority and operation. It returns nil only when all required conditions are met.
func ValidateAuthority(ec ExecutionContext, authority ExecutionAuthority, operation VerifierOperation) error {
	switch authority {
	case AuthorityLocalSafe:
		return validateLocalSafe(ec, operation)
	case AuthorityCIExactCheckout:
		return validateCIExactCheckout(ec, operation)
	default:
		return &AuthorityError{
			RequiredAuthority: authority,
			Operation:         operation,
			ReasonCode:        ReasonCodeUnknownAuthority,
			Message:           fmt.Sprintf("unknown authority: %q", authority),
		}
	}
}

// validateLocalSafe validates local safe authority. The detailed CI
// exact-checkout validator lives in ci_validation.go to keep this file
// below the LLM-friendly line threshold.
func validateLocalSafe(ec ExecutionContext, operation VerifierOperation) error {
	return nil
}

// ReasonCode returns the reason code for a denied authority error.
func ReasonCode(err error) string {
	var ae *AuthorityError
	if errors.As(err, &ae) {
		return ae.ReasonCode
	}
	return ""
}

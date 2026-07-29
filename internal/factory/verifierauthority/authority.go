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
// LocalTrust is set only by the trusted observer
// (DetectExecutionContext, NewLocalOnlyContext, or a test observer that
// mirrors them). It is never derived from environment variables. The only
// valid value is LocalTrustSentinel, which permits
// ClassifyExecutionEnvironment to return EnvironmentLocal. An empty
// LocalTrust makes the context fail closed as EnvironmentUnknown.
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

	// LocalTrust is the explicit observer classification sentinel.
	// Empty means "not observed as local" and the context is classified
	// as EnvironmentUnknown, never as EnvironmentLocal.
	LocalTrust string
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

// DetectExecutionContext reads the production environment and repository state
// to assemble an ExecutionContext. All Git observations use execution.RunGit
// for bounded execution with fail-closed behavior.
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
	if headErr != nil {
		ec.HeadErr = fmt.Errorf("git rev-parse HEAD: %w", headErr)
	} else {
		ec.HeadCommit = strings.TrimSpace(string(headResult.Stdout))
	}

	statusResult, statusErr := execution.RunGit(ctx, repoRoot, "status", "--porcelain=v1")
	if statusErr != nil {
		ec.StatusErr = fmt.Errorf("git status: %w", statusErr)
	} else {
		ec.WorktreeStatus = strings.TrimSpace(string(statusResult.Stdout))
	}

	repoRootResult, repoRootErr := execution.RunGit(ctx, repoRoot, "rev-parse", "--show-toplevel")
	if repoRootErr != nil {
		ec.RepositoryRootErr = fmt.Errorf("git rev-parse --show-toplevel: %w", repoRootErr)
	} else {
		ec.RepositoryRoot = strings.TrimSpace(string(repoRootResult.Stdout))
	}

	if ec.GitHubWorkspace != "" {
		workspaceResult, workspaceErr := execution.RunGit(ctx, ec.GitHubWorkspace, "rev-parse", "--show-toplevel")
		if workspaceErr != nil {
			ec.WorkspaceRootErr = fmt.Errorf("git rev-parse --show-toplevel: %w", workspaceErr)
		} else {
			ec.WorkspaceRoot = strings.TrimSpace(string(workspaceResult.Stdout))
		}
	}

	return ec
}

// NewLocalOnlyContext returns an ExecutionContext classified as
// EnvironmentLocal by ClassifyExecutionEnvironment. The LocalTrust
// sentinel is the only signal that allows a fully empty context to be
// classified as local; without it, ClassifyExecutionEnvironment would
// return EnvironmentUnknown. The sentinel is set only by this function
// and by trusted test observers; it is never derived from environment
// variables.
func NewLocalOnlyContext() *ExecutionContext {
	return &ExecutionContext{
		LocalTrust: LocalTrustSentinel,
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

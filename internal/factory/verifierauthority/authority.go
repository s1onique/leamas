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

	// Observe HEAD commit via bounded Git
	headResult, headErr := execution.RunGit(ctx, repoRoot, "rev-parse", "HEAD^{commit}")
	if headErr != nil {
		ec.HeadErr = fmt.Errorf("git rev-parse HEAD: %w", headErr)
	} else {
		ec.HeadCommit = strings.TrimSpace(string(headResult.Stdout))
	}

	// Observe worktree status via bounded Git
	statusResult, statusErr := execution.RunGit(ctx, repoRoot, "status", "--porcelain=v1")
	if statusErr != nil {
		ec.StatusErr = fmt.Errorf("git status: %w", statusErr)
	} else {
		ec.WorktreeStatus = strings.TrimSpace(string(statusResult.Stdout))
	}

	// Observe repository root via bounded Git
	repoRootResult, repoRootErr := execution.RunGit(ctx, repoRoot, "rev-parse", "--show-toplevel")
	if repoRootErr != nil {
		ec.RepositoryRootErr = fmt.Errorf("git rev-parse --show-toplevel: %w", repoRootErr)
	} else {
		ec.RepositoryRoot = strings.TrimSpace(string(repoRootResult.Stdout))
	}

	// Observe workspace root via bounded Git
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

// NewLocalOnlyContext returns an ExecutionContext suitable for local_safe verifiers.
// This skips expensive Git observation since local_safe verifiers are permitted
// in any local context without further verification.
func NewLocalOnlyContext() *ExecutionContext {
	return &ExecutionContext{
		// CI and GitHubActions are intentionally left empty for local context.
		// AuthorityLocalSafe validation does not require these fields.
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

// validateLocalSafe validates local safe authority.
func validateLocalSafe(ec ExecutionContext, operation VerifierOperation) error {
	// All operations are allowed locally
	return nil
}

// validateCIExactCheckout validates CI exact checkout authority.
// AuthorityCIExactCheckout may allow read-only verification only when all are true:
//   - CI == "true"
//   - GITHUB_ACTIONS == "true"
//   - authority marker matches configured value
//   - GITHUB_SHA is a valid full object ID
//   - observed HEAD == GITHUB_SHA
//   - worktree status is empty
//   - GITHUB_WORKSPACE Git top-level equals repository Git top-level
//   - operation == verify
func validateCIExactCheckout(ec ExecutionContext, operation VerifierOperation) error {
	// Fail-closed on Git errors first
	if ec.HeadErr != nil {
		return &AuthorityError{
			RequiredAuthority: AuthorityCIExactCheckout,
			Operation:         operation,
			ReasonCode:        ReasonCodeGitHeadFailed,
			Message:           fmt.Sprintf("git rev-parse HEAD failed: %v", ec.HeadErr),
		}
	}

	if ec.StatusErr != nil {
		return &AuthorityError{
			RequiredAuthority: AuthorityCIExactCheckout,
			Operation:         operation,
			ReasonCode:        ReasonCodeGitStatusFailed,
			Message:           fmt.Sprintf("git status failed: %v", ec.StatusErr),
		}
	}

	if ec.RepositoryRootErr != nil {
		return &AuthorityError{
			RequiredAuthority: AuthorityCIExactCheckout,
			Operation:         operation,
			ReasonCode:        ReasonCodeGitRepoRootFailed,
			Message:           fmt.Sprintf("git rev-parse --show-toplevel failed: %v", ec.RepositoryRootErr),
		}
	}

	if ec.GitHubWorkspace != "" && ec.WorkspaceRootErr != nil {
		return &AuthorityError{
			RequiredAuthority: AuthorityCIExactCheckout,
			Operation:         operation,
			ReasonCode:        ReasonCodeGitWorkspaceRootFailed,
			Message:           fmt.Sprintf("git rev-parse --show-toplevel for GITHUB_WORKSPACE failed: %v", ec.WorkspaceRootErr),
		}
	}

	// Check CI environment markers
	if ec.CI != "true" {
		return &AuthorityError{
			RequiredAuthority: AuthorityCIExactCheckout,
			Operation:         operation,
			ReasonCode:        ReasonCodeMissingCI,
			Message:           `CI must be set to "true"`,
		}
	}

	if ec.GitHubActions != "true" {
		return &AuthorityError{
			RequiredAuthority: AuthorityCIExactCheckout,
			Operation:         operation,
			ReasonCode:        ReasonCodeMissingGitHubActions,
			Message:           `GITHUB_ACTIONS must be set to "true"`,
		}
	}

	if ec.AuthorityMarker == "" {
		return &AuthorityError{
			RequiredAuthority: AuthorityCIExactCheckout,
			Operation:         operation,
			ReasonCode:        ReasonCodeMissingAuthorityMarker,
			Message:           fmt.Sprintf("%s must be set to %q", EnvAuthorityMarker, AuthorityMarker),
		}
	}

	if ec.AuthorityMarker != AuthorityMarker {
		return &AuthorityError{
			RequiredAuthority: AuthorityCIExactCheckout,
			Operation:         operation,
			ReasonCode:        ReasonCodeWrongAuthorityMarker,
			Message:           fmt.Sprintf("%s must be set to %q (got %q)", EnvAuthorityMarker, AuthorityMarker, ec.AuthorityMarker),
		}
	}

	// Check SHA validity
	if ec.GitHubSHA == "" {
		return &AuthorityError{
			RequiredAuthority: AuthorityCIExactCheckout,
			Operation:         operation,
			ReasonCode:        ReasonCodeMissingSHA,
			Message:           fmt.Sprintf("%s must be set", EnvGitHubSHA),
		}
	}

	if !fullCommitOIDRegex.MatchString(ec.GitHubSHA) {
		return &AuthorityError{
			RequiredAuthority: AuthorityCIExactCheckout,
			Operation:         operation,
			ReasonCode:        ReasonCodeMalformedSHA,
			Message:           fmt.Sprintf("%s must be a valid full Git commit OID", EnvGitHubSHA),
		}
	}

	// Check HEAD matches GITHUB_SHA
	if ec.HeadCommit != ec.GitHubSHA {
		return &AuthorityError{
			RequiredAuthority: AuthorityCIExactCheckout,
			Operation:         operation,
			ReasonCode:        ReasonCodeHeadMismatch,
			Message:           fmt.Sprintf("HEAD (%s) does not match %s (%s)", ec.HeadCommit, EnvGitHubSHA, ec.GitHubSHA),
		}
	}

	// Check worktree is clean
	if ec.WorktreeStatus != "" {
		return &AuthorityError{
			RequiredAuthority: AuthorityCIExactCheckout,
			Operation:         operation,
			ReasonCode:        ReasonCodeDirtyTree,
			Message:           "repository worktree must be clean",
		}
	}

	// Check workspace is set and matches repository root
	if ec.GitHubWorkspace == "" {
		return &AuthorityError{
			RequiredAuthority: AuthorityCIExactCheckout,
			Operation:         operation,
			ReasonCode:        ReasonCodeWorkspaceMismatch,
			Message:           fmt.Sprintf("%s must be set", EnvGitHubWorkspace),
		}
	}
	if ec.WorkspaceRoot != ec.RepositoryRoot {
		return &AuthorityError{
			RequiredAuthority: AuthorityCIExactCheckout,
			Operation:         operation,
			ReasonCode:        ReasonCodeWorkspaceMismatch,
			Message:           fmt.Sprintf("%s does not match the executing checkout", EnvGitHubWorkspace),
		}
	}

	// Check operation is permitted
	if operation == OperationUpdateBaseline {
		return &AuthorityError{
			RequiredAuthority: AuthorityCIExactCheckout,
			Operation:         operation,
			ReasonCode:        ReasonCodeOperationDenied,
			Message:           "update_baseline operation is not permitted under ci_exact_checkout authority",
		}
	}

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

// SPDX-License-Identifier: Apache-2.0

package verifierauthority

import (
	"fmt"
)

// validateCIExactCheckout validates ci_exact_checkout authority.
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

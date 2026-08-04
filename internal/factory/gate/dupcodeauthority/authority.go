// SPDX-License-Identifier: Apache-2.0

// Package dupcodeauthority is the central production authority that
// decides whether the dupcode verifier lane is permitted to execute.
//
// Dupcode is a CI-only verifier lane. Local development, editor
// sessions, agent runs, and ordinary shell invocations cannot start
// the analyzer. Permission is granted only when a central validator
// proves that every required marker is present, well-formed, and
// mutually consistent.
//
// This package is a compatibility shim that delegates to verifierauthority.
// The verifierauthority package provides the canonical authority model.
package dupcodeauthority

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// Authority marker constants. The job that runs the dupcode lane must
// set LEAMAS_DUPCODE_AUTHORITY to exactly this value.
const (
	// AuthorityMarker is the required value of LEAMAS_DUPCODE_AUTHORITY.
	AuthorityMarker = "github-actions"

	// envCI is the canonical CI marker GitHub Actions exports.
	envCI = "CI"
	// envGitHubActions is GitHub Actions' boolean CI flag.
	envGitHubActions = "GITHUB_ACTIONS"
	// envDupcodeAuthority is the dupcode-specific authority switch.
	envDupcodeAuthority = "LEAMAS_DUPCODE_AUTHORITY"
	// envGitHubSHA is the exact commit SHA GitHub Actions checks out.
	envGitHubSHA = "GITHUB_SHA"
	// envGitHubWorkspace is the workspace root GitHub Actions checks out.
	envGitHubWorkspace = "GITHUB_WORKSPACE"
)

// fullCommitOIDRegex matches a 40-character lowercase hex SHA-1 and a
// 64-character lowercase hex SHA-256. The verifier accepts both.
var fullCommitOIDRegex = regexp.MustCompile(`^[0-9a-f]{40}$|^[0-9a-f]{64}$`)

// DupcodeExecutionContext captures every observable signal that the
// central authority consults. This is a compatibility shim that wraps
// verifierauthority.ExecutionContext.
type DupcodeExecutionContext struct {
	CI              string
	GitHubActions   string
	Authority       string
	GitHubSHA       string
	GitHubWorkspace string
	RepositoryRoot  string
	HeadCommit      string
	HeadErr         error
	WorktreeStatus  string
	StatusErr       error
	WorktreeClean   bool
	WorkspaceRoot   string
}

// GitObservation captures the result of Git status reads.
// Failures are encoded in the Err field; any non-nil error means
// the observation must be treated as a denial of authority.
type GitObservation struct {
	Head      string
	Status    string
	HeadErr   error
	StatusErr error
}

// HeadReader returns the result of `git rev-parse HEAD^{commit}` for
// the supplied root, along with any execution error.
type HeadReader func(root string) (string, error)

// WorktreeReader returns the porcelain-v1 status of the supplied root,
// along with any execution error.
type WorktreeReader func(root string) (string, error)

// CheapAnalyzer is the function type the dupcode verifier uses to scan
// the repository. Tests inject a counting fake so the policy tests
// can prove the analyzer never runs when the authority validator
// denies.
type CheapAnalyzer func(root string) error

// DetectContext reads the production environment and the supplied
// repository root to assemble a DupcodeExecutionContext. This delegates
// to verifierauthority.DetectExecutionContext for the canonical Git
// observation using execution.RunGit.
func DetectContext(ec verifierauthority.ExecutionContext) DupcodeExecutionContext {
	worktreeClean := ec.StatusErr == nil && ec.WorktreeStatus == ""
	return DupcodeExecutionContext{
		CI:              ec.CI,
		GitHubActions:   ec.GitHubActions,
		Authority:       ec.AuthorityMarker,
		GitHubSHA:       ec.GitHubSHA,
		GitHubWorkspace: ec.GitHubWorkspace,
		RepositoryRoot:  ec.RepositoryRoot,
		HeadCommit:      ec.HeadCommit,
		HeadErr:         ec.HeadErr,
		WorktreeStatus:  ec.WorktreeStatus,
		StatusErr:       ec.StatusErr,
		WorktreeClean:   worktreeClean,
		WorkspaceRoot:   ec.WorkspaceRoot,
	}
}

// ErrDupcodeDenied is the canonical denial diagnostic. Callers wrap or
// extend it with the specific failure reason. The error message is
// stable: it tells the operator exactly what to do.
var ErrDupcodeDenied = errors.New(
	"dupcode is a CI-only verifier lane; " +
		"local execution is prohibited; " +
		"push a branch or open a PR and use the Factory Dupcode status check",
)

// newDupcodeDenied wraps ErrDupcodeDenied with the specific reason.
func newDupcodeDenied(reason string) error {
	return fmt.Errorf("%w: %s", ErrDupcodeDenied, reason)
}

// IsDupcodeDenied reports whether err originated from this package.
func IsDupcodeDenied(err error) bool {
	return errors.Is(err, ErrDupcodeDenied)
}

// ValidateDupcodeExecutionAuthority is the central gate. It returns
// nil only when every required marker is present, well-formed, and
// mutually consistent. This delegates to verifierauthority.ValidateAuthority
// with the CIExactCheckout authority and verify operation.
func ValidateDupcodeExecutionAuthority(ctx DupcodeExecutionContext, operation verifierauthority.VerifierOperation) error {
	// Convert to the generic context
	ec := verifierauthority.ExecutionContext{
		CI:              ctx.CI,
		GitHubActions:   ctx.GitHubActions,
		AuthorityMarker: ctx.Authority,
		GitHubSHA:       ctx.GitHubSHA,
		GitHubWorkspace: ctx.GitHubWorkspace,
		RepositoryRoot:  ctx.RepositoryRoot,
		HeadCommit:      ctx.HeadCommit,
		WorktreeStatus:  ctx.WorktreeStatus,
		HeadErr:         ctx.HeadErr,
		StatusErr:       ctx.StatusErr,
		WorkspaceRoot:   ctx.WorkspaceRoot,
	}

	// Delegate to the generic validator
	err := verifierauthority.ValidateAuthority(ec, verifierauthority.AuthorityCIExactCheckout, operation)
	if err == nil {
		return nil
	}

	// Convert back to the dupcode-specific error format
	return newDupcodeDenied(err.Error())
}

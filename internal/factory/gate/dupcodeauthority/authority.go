// SPDX-License-Identifier: Apache-2.0

// Package dupcodeauthority is the central production authority that
// decides whether the dupcode verifier lane is permitted to execute.
//
// Dupcode is a CI-only verifier lane. Local development, editor
// sessions, agent runs, and ordinary shell invocations cannot start
// the analyzer. Permission is granted only when a central validator
// proves that every required marker is present, well-formed, and
// mutually consistent:
//
//   - CI == "true"
//   - GITHUB_ACTIONS == "true"
//   - LEAMAS_DUPCODE_AUTHORITY == "github-actions"
//   - GITHUB_SHA is present and is a valid full Git commit OID
//   - git rev-parse HEAD == GITHUB_SHA
//   - the repository worktree is clean
//   - GITHUB_WORKSPACE resolves to the same Git top-level as the
//     executing checkout
//
// Treat a missing, malformed, contradictory, or mismatched value as
// denial. Do not allow overrides; do not install a backdoor.
package dupcodeauthority

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
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
// central authority consults. The fields are exported so the
// production seam and the cheap test fake can supply them
// deterministically.
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
}

// EnvLookup is the function the authority uses to inspect the
// environment. os.Getenv is the default; tests inject a fake.
type EnvLookup func(string) string

// LookupFromOS returns os.Getenv. The default EnvLookup for production.
func LookupFromOS(key string) string { return os.Getenv(key) }

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

// LookupFromGit uses the supplied root to compute HeadCommit via
// `git rev-parse HEAD^{commit}` and the porcelain status of the
// working tree. Any Git execution error is recorded in the observation
// and MUST cause authority denial.
func LookupFromGit(root string) (HeadReader, WorktreeReader) {
	head := func(r string) (string, error) {
		cmd := exec.Command("git", "-C", r, "rev-parse", "HEAD^{commit}")
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	}
	status := func(r string) (string, error) {
		cmd := exec.Command("git", "-C", r, "status", "--porcelain=v1")
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	}
	return head, status
}

// DetectContext reads the production environment and the supplied
// repository root to assemble a DupcodeExecutionContext. The HeadCommit
// is computed via `git rev-parse HEAD^{commit}`; the worktree cleanliness
// is computed via `git status --porcelain=v1`. Any Git error is recorded
// in the context's GitError field; callers MUST check this and deny if set.
func DetectContext(root string) DupcodeExecutionContext {
	head, status := LookupFromGit(root)
	headCommit, headErr := head(root)
	worktreeStatus, statusErr := status(root)
	return DupcodeExecutionContext{
		CI:              LookupFromOS(envCI),
		GitHubActions:   LookupFromOS(envGitHubActions),
		Authority:       LookupFromOS(envDupcodeAuthority),
		GitHubSHA:       LookupFromOS(envGitHubSHA),
		GitHubWorkspace: LookupFromOS(envGitHubWorkspace),
		RepositoryRoot:  root,
		HeadCommit:      headCommit,
		HeadErr:         headErr,
		WorktreeStatus:  worktreeStatus,
		StatusErr:       statusErr,
		WorktreeClean:   statusErr == nil && worktreeStatus == "",
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

// ErrDupcodeDenied wraps ErrDupcodeDenied with the specific reason.
func newDupcodeDenied(reason string) error {
	return fmt.Errorf("%w: %s", ErrDupcodeDenied, reason)
}

// IsDupcodeDenied reports whether err originated from this package.
func IsDupcodeDenied(err error) bool {
	return errors.Is(err, ErrDupcodeDenied)
}

// ValidateDupcodeExecutionAuthority is the central gate. It returns
// nil only when every required marker is present, well-formed, and
// mutually consistent. Otherwise it returns a wrapped ErrDupcodeDenied
// carrying the specific failure reason.
//
// The conditions are evaluated in the order GitHub Actions publishes them.
// The first failure short-circuits. Any Git execution error causes denial.
func ValidateDupcodeExecutionAuthority(ctx DupcodeExecutionContext) error {
	// Git errors fail-closed: any failure to read HEAD or worktree status
	// must deny authorization. This is not recoverable.
	if ctx.HeadErr != nil {
		return newDupcodeDenied(fmt.Sprintf("git rev-parse HEAD failed: %v", ctx.HeadErr))
	}
	if ctx.StatusErr != nil {
		return newDupcodeDenied(fmt.Sprintf("git status failed: %v", ctx.StatusErr))
	}
	if ctx.CI != "true" {
		return newDupcodeDenied("CI must be set to \"true\"")
	}
	if ctx.GitHubActions != "true" {
		return newDupcodeDenied("GITHUB_ACTIONS must be set to \"true\"")
	}
	if ctx.Authority != AuthorityMarker {
		return newDupcodeDenied(fmt.Sprintf(
			"LEAMAS_DUPCODE_AUTHORITY must be set to %q (got %q)",
			AuthorityMarker, ctx.Authority))
	}
	if !fullCommitOIDRegex.MatchString(ctx.GitHubSHA) {
		return newDupcodeDenied("GITHUB_SHA must be a valid full Git commit OID")
	}
	if ctx.HeadCommit != ctx.GitHubSHA {
		return newDupcodeDenied(
			fmt.Sprintf("HEAD (%s) does not match GITHUB_SHA", ctx.HeadCommit))
	}
	if !ctx.WorktreeClean {
		return newDupcodeDenied("repository worktree must be clean")
	}
	if resolveRepoRoot(ctx.GitHubWorkspace) != resolveRepoRoot(ctx.RepositoryRoot) {
		return newDupcodeDenied("GITHUB_WORKSPACE does not match the executing checkout")
	}
	return nil
}

// resolveRepoRoot is the helper invoked by the validator. It applies
// the canonical git top-level resolution so the workspace and the
// executing checkout can be compared even when one is a symlink or
// is supplied as a sub-path.
func resolveRepoRoot(path string) string {
	if path == "" {
		return ""
	}
	cmd := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return strings.TrimSpace(path)
	}
	return strings.TrimSpace(string(out))
}

// CheapAnalyzer is the function type the dupcode verifier uses to scan
// the repository. Tests inject a counting fake so the policy tests
// can prove the analyzer never runs when the authority validator
// denies.
type CheapAnalyzer func(root string) error

// SPDX-License-Identifier: Apache-2.0

// Package authority: authority.go models the executable-authority
// check that distinguishes a binary bound to the repository from an
// obsolete installation.
//
// The check is read-only and uses git to compare the binary's
// embedded VCS commit against the repository HEAD. It never invokes
// git commands that mutate state and never modifies the filesystem.
package authority

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Relationship describes how the running binary's commit relates to
// the repository HEAD.
type Relationship string

const (
	// RelationshipUnknown is returned when git cannot reach the
	// binary's commit, typically in a shallow clone that prunes
	// it from history.
	RelationshipUnknown Relationship = "unknown"
	// RelationshipEqual means the binary's commit equals HEAD.
	RelationshipEqual Relationship = "equal"
	// RelationshipAncestor means the binary's commit is reachable
	// from HEAD (HEAD is descended from the binary).
	RelationshipAncestor Relationship = "ancestor"
	// RelationshipDescendant means HEAD is reachable from the
	// binary's commit (the binary is newer than HEAD).
	RelationshipDescendant Relationship = "descendant"
	// RelationshipUnrelated means the binary's commit and HEAD
	// share no common ancestor in the local repository.
	RelationshipUnrelated Relationship = "unrelated"
)

// Verdict is the combined authority verdict rendered by the doctor.
type Verdict string

const (
	VerdictAuthoritative Verdict = "authoritative"
	VerdictAncestor      Verdict = "ancestor_capability_acceptable"
	VerdictStaleAncestor Verdict = "ancestor_capability_insufficient"
	VerdictStale         Verdict = "stale"
	VerdictUnrelated     Verdict = "unrelated"
	VerdictUnverifiable  Verdict = "unverifiable"
)

// Check is the aggregated executable-authority state computed from
// the binary, the repository, and the capability manifest.
type Check struct {
	ExecutablePath    string
	ResolvedSymlink   string
	RepositoryRoot    string
	RepositoryHead    string
	BinaryCommit      string
	WorkingTreeClean  bool
	EmbeddedCaps      *EmbeddedCapabilities
	RequiredCaps      *RequiredCapabilities
	CapabilityGap     string
	Relationship      Relationship
	Verdict           Verdict
	VerdictReason     string
	BootstrapStrategy string
}

// PathResolver returns the absolute path of the current executable.
// It is a variable so tests can pin the resolver to a temp dir.
var PathResolver = currentExecutablePath

// GitRunner abstracts git invocations so tests can substitute a
// fake. The closure must invoke git with the supplied args and return
// the trimmed stdout; non-zero exit codes must be reported as errors.
type GitRunner func(repoRoot string, args ...string) (string, error)

// DefaultGitRunner invokes `git` via os/exec.
var DefaultGitRunner GitRunner = defaultGitRunner

func defaultGitRunner(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return strings.TrimSpace(string(out)), fmt.Errorf("git %s: exit %d: %s",
				strings.Join(args, " "), exitErr.ExitCode(), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return strings.TrimSpace(string(out)), fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// CheckResult is the document returned from CheckExecutable. It is
// returned to the launcher and rendered by the doctor command.
type CheckResult = Check

// CheckExecutable computes the authority state for the currently
// running executable against repoRoot.
//
// repoRoot may be empty; in that case CheckExecutable refuses to
// produce a verdict and returns VerdictUnverifiable.
//
// embeddedCommit, repoHead, and cleanTree allow tests to inject
// authoritative values without consulting the version package or
// running git. Pass empty strings to read from the running binary /
// git.
func CheckExecutable(repoRoot, embeddedCommit, repoHead string, cleanTree bool) (*Check, error) {
	exe, err := PathResolver()
	if err != nil {
		return nil, fmt.Errorf("locate executable: %w", err)
	}

	resolved, _ := filepath.EvalSymlinks(exe)

	if embeddedCommit == "" {
		embeddedCommit = defaultEmbeddedCommit()
	}

	check := &Check{
		ExecutablePath:   exe,
		ResolvedSymlink:  resolved,
		EmbeddedCaps:     SnapshotEmbedded(),
		BinaryCommit:     embeddedCommit,
		WorkingTreeClean: cleanTree,
	}

	if repoRoot == "" {
		check.Verdict = VerdictUnverifiable
		check.VerdictReason = "repository root unknown"
		return check, nil
	}
	check.RepositoryRoot = repoRoot

	// Resolve repository HEAD when not injected.
	if repoHead == "" {
		head, err := DefaultGitRunner(repoRoot, "rev-parse", "--verify", "HEAD")
		if err != nil {
			check.Verdict = VerdictUnverifiable
			check.VerdictReason = fmt.Sprintf("cannot read HEAD: %v", err)
			return check, nil
		}
		repoHead = head
	}
	check.RepositoryHead = repoHead

	// Determine relationship between binary commit and HEAD.
	check.Relationship = classify(repoRoot, embeddedCommit, repoHead)

	// Load required capabilities from the repository metadata file.
	required, err := LoadRequired(DefaultPath(repoRoot))
	if err != nil {
		check.Verdict = VerdictUnverifiable
		check.VerdictReason = fmt.Sprintf("cannot load required capabilities: %v", err)
		return check, nil
	}
	check.RequiredCaps = required

	// Capability check: older binaries that are technically
	// ancestors of HEAD must still satisfy the capability floor.
	if gap := required.SatisfiedBy(check.EmbeddedCaps); gap != nil {
		check.CapabilityGap = gap.Error()
		switch check.Relationship {
		case RelationshipEqual, RelationshipAncestor:
			check.Verdict = VerdictStaleAncestor
			check.VerdictReason = "binary predates a required capability bump; rebuild"
		default:
			check.Verdict = VerdictStale
			check.VerdictReason = "binary lacks required capabilities"
		}
		check.BootstrapStrategy = "leamas bootstrap self"
		return check, nil
	}

	switch check.Relationship {
	case RelationshipEqual:
		check.Verdict = VerdictAuthoritative
		check.VerdictReason = "binary equals HEAD"
	case RelationshipAncestor:
		check.Verdict = VerdictAncestor
		check.VerdictReason = "binary is an ancestor of HEAD; capability check passed"
	case RelationshipDescendant:
		check.Verdict = VerdictStale
		check.VerdictReason = "binary is descended from HEAD; HEAD is not authoritative for the binary"
		check.BootstrapStrategy = "leamas bootstrap self"
	case RelationshipUnrelated:
		check.Verdict = VerdictUnrelated
		check.VerdictReason = "binary and HEAD share no common ancestor"
		check.BootstrapStrategy = "leamas bootstrap self"
	default:
		check.Verdict = VerdictUnverifiable
		check.VerdictReason = "relationship unknown"
		check.BootstrapStrategy = "leamas bootstrap self"
	}

	return check, nil
}

// classify returns the relationship between embeddedCommit and
// repoHead. Either may be empty.
func classify(repoRoot, embeddedCommit, repoHead string) Relationship {
	if embeddedCommit == "" || embeddedCommit == "unknown" || repoHead == "" {
		return RelationshipUnknown
	}
	if strings.EqualFold(embeddedCommit, repoHead) {
		return RelationshipEqual
	}
	// Distinguish "commit not in this repository" (typical of a
	// shallow clone that prunes history) from "commit exists but
	// shares no ancestor with HEAD". `git cat-file -e` returns
	// non-zero for missing objects.
	_, catErr := DefaultGitRunner(repoRoot, "cat-file", "-e", embeddedCommit)
	if catErr != nil {
		return RelationshipUnknown
	}
	_, catErr = DefaultGitRunner(repoRoot, "cat-file", "-e", repoHead)
	if catErr != nil {
		return RelationshipUnknown
	}
	_, err := DefaultGitRunner(repoRoot, "merge-base", "--is-ancestor", embeddedCommit, repoHead)
	if err == nil {
		return RelationshipAncestor
	}
	_, err = DefaultGitRunner(repoRoot, "merge-base", "--is-ancestor", repoHead, embeddedCommit)
	if err == nil {
		return RelationshipDescendant
	}
	return RelationshipUnrelated
}

// currentExecutablePath returns the absolute path of the running
// executable, dereferenced via /proc/self/exe on Linux.
func currentExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return exe, nil
}

// defaultEmbeddedCommit returns the binary's commit as known to the
// version package, falling back to the value embedded by the
// linker when the package globals are unavailable.
func defaultEmbeddedCommit() string {
	// version.Get is imported transitively via factory.authority
	// only by build-time injection; we read directly via the
	// package-level symbol to avoid pulling the package here.
	return readVersionCommit()
}

// Package digest provides targeted digest generation for Git repositories.
// It creates reviewable artifacts of repository changes suitable for
// agent-assisted review workflows.
package digest

import (
	"fmt"
	"strings"
)

// Mode represents the digest generation mode.
type Mode string

const (
	// ModeAuto automatically selects dirty or range mode based on working tree state.
	ModeAuto Mode = "auto"
	// ModeDirty includes all unstaged, staged, and untracked changes.
	ModeDirty Mode = "dirty"
	// ModeStaged includes only staged changes.
	ModeStaged Mode = "staged"
	// ModeRange includes changes between two commits/refs.
	ModeRange Mode = "range"
)

// ResolvedMode represents the auto-resolved mode with context.
//
// Lifecycle* fields capture the authoritative ACT identities when the
// resolver can identify the current ACT from closure artifacts. They
// are zero when no ACT is in scope (for example, the single-commit
// fallback when HEAD is not part of any ACT).
type ResolvedMode struct {
	Mode       Mode
	Range      string
	Reason     string
	IsClean    bool
	BaseCommit string
	HeadCommit string

	// Lifecycle metadata (populated by the auto-range resolver).
	AutoRangeStrategy   string
	ActID               string
	LifecycleFreeze     string
	LifecycleSubject    string
	LifecycleClosure    string
	IncludedCommits     []string
	GeneratorCommit     string
	GeneratorIsAncestor bool
	GeneratorStale      bool
	StaleReason         string
}

// RangeStrategy returns the strategy label used to authoritatively
// pick the range, or empty when no lifecycle metadata is available.
func (r *ResolvedMode) RangeStrategy() string {
	if r == nil {
		return ""
	}
	return r.AutoRangeStrategy
}

// ResolveAutoMode determines whether to use dirty or range mode based
// on working tree state, and returns the authoritative ACT range when
// the working tree is clean.
//
// The returned ResolvedMode carries the lifecycle metadata required
// by ACT-LEAMAS-FACTORY-DIGEST-AUTO-ACT-RANGE01: ActID, freeze /
// subject / closure OIDs, the strategy that selected the range, the
// list of included commits, and the generator binary freshness
// fields.
func ResolveAutoMode(repoRoot string) (*ResolvedMode, error) {
	result := &ResolvedMode{}

	head, err := runGitValueTrimmed(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	result.HeadCommit = mustResolveOID(repoRoot, head)

	// Check for staged changes - git diff --cached --quiet returns error if there are staged changes
	_, stagedErr := runGitBytes(repoRoot, "diff", "--cached", "--quiet")
	hasStagedChanges := stagedErr != nil

	// Check for unstaged changes - git diff --quiet returns error if there are unstaged changes
	_, unstagedErr := runGitBytes(repoRoot, "diff", "--quiet")
	hasUnstagedChanges := unstagedErr != nil

	// Check for untracked files
	untrackedOutput, err := runGitBytes(repoRoot, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, fmt.Errorf("failed to check untracked files: %w", err)
	}
	hasUntrackedFiles := strings.TrimSpace(string(untrackedOutput)) != ""

	isDirty := hasStagedChanges || hasUnstagedChanges || hasUntrackedFiles

	if isDirty {
		result.Mode = ModeDirty
		result.Reason = "working tree has changes"
		result.IsClean = false
		return result, nil
	}

	// Working tree is clean: defer to the lifecycle resolver. The
	// resolver inspects closure artifacts and falls back to the
	// empty-tree / single-commit paths when no ACT is in scope.
	result.IsClean = true
	resolution, err := resolveLifecycleAtHEAD(repoRoot)
	if err != nil {
		return nil, err
	}
	applyLifecycleToResolved(result, resolution)
	return result, nil
}

// applyLifecycleToResolved copies lifecycle metadata from a
// LifecycleResolution into the legacy ResolvedMode used by the rest
// of the digest pipeline. The function is intentionally local to
// resolve.go so the two structs evolve together.
func applyLifecycleToResolved(out *ResolvedMode, r *LifecycleResolution) {
	out.Mode = ModeRange
	out.Range = r.Range()
	out.Reason = r.Reason
	out.BaseCommit = r.RangeBase
	out.HeadCommit = r.RangeHead
	out.AutoRangeStrategy = r.Strategy
	out.ActID = r.ActID
	out.LifecycleFreeze = r.LifecycleFreeze
	out.LifecycleSubject = r.LifecycleSubject
	out.LifecycleClosure = r.LifecycleClosure
	out.IncludedCommits = append([]string(nil), r.IncludedCommits...)
	out.GeneratorCommit = r.GeneratorCommit
	out.GeneratorIsAncestor = r.GeneratorIsAncestor
	out.GeneratorStale = r.GeneratorStale
	out.StaleReason = r.StaleReason
}

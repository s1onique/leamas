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
type ResolvedMode struct {
	Mode       Mode
	Range      string
	Reason     string
	IsClean    bool
	BaseCommit string
	HeadCommit string
}

// ResolveAutoMode determines whether to use dirty or range mode based on working tree state.
func ResolveAutoMode(repoRoot string) (*ResolvedMode, error) {
	result := &ResolvedMode{}

	// Get HEAD commit
	head, err := RunGit(repoRoot, []string{"rev-parse", "HEAD"})
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	result.HeadCommit = strings.TrimSpace(head)

	// Check for staged changes - git diff --cached --quiet returns error if there are staged changes
	_, stagedErr := RunGit(repoRoot, []string{"diff", "--cached", "--quiet"})
	hasStagedChanges := stagedErr != nil

	// Check for unstaged changes - git diff --quiet returns error if there are unstaged changes
	_, unstagedErr := RunGit(repoRoot, []string{"diff", "--quiet"})
	hasUnstagedChanges := unstagedErr != nil

	// Check for untracked files
	untrackedOutput, err := RunGit(repoRoot, []string{"ls-files", "--others", "--exclude-standard"})
	if err != nil {
		return nil, fmt.Errorf("failed to check untracked files: %w", err)
	}
	hasUntrackedFiles := strings.TrimSpace(untrackedOutput) != ""

	isDirty := hasStagedChanges || hasUnstagedChanges || hasUntrackedFiles

	if isDirty {
		result.Mode = ModeDirty
		result.Reason = "working tree has changes"
		result.IsClean = false
		return result, nil
	}

	// Working tree is clean, use previous commit range
	result.IsClean = true

	// Check if HEAD has a parent
	parentCheck, err := RunGit(repoRoot, []string{"rev-parse", "--verify", "HEAD~1"})
	if err != nil {
		// No parent - initial commit
		// Use empty tree baseline (4b825dc642cb6eb9a060e54bf8d69288fbee4904) for initial commit
		result.Range = "4b825dc642cb6eb9a060e54bf8d69288fbee4904..HEAD"
		result.Mode = ModeRange
		result.Reason = "initial commit; diffing against empty tree baseline"
		return result, nil
	}

	parentCommit := strings.TrimSpace(parentCheck)
	result.BaseCommit = parentCommit

	// Use HEAD~1..HEAD for the range
	result.Range = "HEAD~1..HEAD"
	result.Mode = ModeRange
	result.Reason = "working tree clean; showing previous commit"

	return result, nil
}

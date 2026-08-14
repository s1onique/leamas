// Package digest provides targeted digest generation for Git repositories.
// It creates reviewable artifacts of repository changes suitable for
// agent-assisted review workflows.
package digest

import (
	"fmt"
	"strings"

	"github.com/s1onique/leamas/internal/factory/authority"
	"github.com/s1onique/leamas/internal/version"
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
	AutoRangeStrategy string
	ActID             string
	LifecycleFreeze   string
	LifecycleSubject  string
	// LifecycleSubjectRange (CORRECTION01) is the resolved
	// right endpoint of an explicit --range. When non-empty
	// the renderer uses this value (not HeadCommit) as the
	// digest subject for binding purposes. Empty for clean
	// committed auto-mode resolutions where LifecycleSubject
	// already records the same value.
	LifecycleSubjectRange string
	LifecycleClosure      string
	IncludedCommits       []string
	GeneratorCommit       string
	GeneratorIsAncestor   bool
	GeneratorStale        bool
	StaleReason           string

	// AuthorityStatus is the typed classification from the shared
	// authority resolver. It is the single source of truth for
	// whether the resolved range is authoritative.
	AuthorityStatus authority.AuthorityStatus

	// ToolIdentity records the exact executable that produced
	// the digest. Equality of the binary bytes, declared commit,
	// and VCS revision are recorded separately.
	ToolIdentity authority.ToolIdentity

	// ResolutionSource is the strategy label that resolved the
	// range (closure_manifest, explicit_cli, etc.).
	ResolutionSource string
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
// The function now delegates lifecycle classification to the shared
// authority resolver in internal/factory/authority. It no longer
// implements any heuristic fallback such as `HEAD~1..HEAD`; the
// authoritative auto path must derive its range from validated
// lifecycle artifacts only.
//
// Tool identity is recorded on every resolution so downstream
// consumers can detect incompatible stale binaries.
func ResolveAutoMode(repoRoot string) (*ResolvedMode, error) {
	return resolveAutoModeWith(repoRoot, "", "")
}

// ResolveAutoModeWithTool is the entry point used by the CLI to
// bind a specific tool binary path. The shared authority resolver
// records the path, SHA-256, declared version, and embedded VCS
// revision on every resolution.
func ResolveAutoModeWithTool(repoRoot, toolPath string) (*ResolvedMode, error) {
	return resolveAutoModeWith(repoRoot, toolPath, "")
}

// ResolveAutoModeExplicitRange is the entry point used when the
// caller supplied an explicit --range. The authority resolver
// classifies the result as AuthorityExplicitRange and never as
// authoritative.
func ResolveAutoModeExplicitRange(repoRoot, toolPath, rangeSpec string) (*ResolvedMode, error) {
	return resolveAutoModeWith(repoRoot, toolPath, rangeSpec)
}

// resolveAutoModeWith is the test seam for ResolveAutoMode. The
// optional toolPath and explicitRange allow callers and tests to
// bind a specific binary and to bypass lifecycle authority for
// diagnostic review.
func resolveAutoModeWith(repoRoot, toolPath, explicitRange string) (*ResolvedMode, error) {
	result := &ResolvedMode{}

	head, err := runGitValueTrimmed(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	result.HeadCommit = mustResolveOID(repoRoot, head)

	// Generator identity. Recorded from the binary's embedded
	// VCS commit (linker + runtime/debug.ReadBuildInfo
	// fallback). Populated before the legacy
	// GENERATOR_STALE boolean is computed so the new
	// EvaluateGeneratorBinding classifier (in the renderer)
	// has the inputs it needs. The legacy boolean is the
	// commit-vs-repository-HEAD signal; the new authority
	// signal is rendered from the same value plus the digest
	// subject (see resolveGeneratorBindingForRender).
	result.GeneratorCommit = strings.TrimSpace(version.Get().Commit)
	result.GeneratorStale = computeLegacyGeneratorStale(result.GeneratorCommit, result.HeadCommit)
	if result.GeneratorStale {
		result.StaleReason = "embedded leamas commit does not match repository HEAD"
	}

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

	// If an explicit range was provided, it takes precedence over the
	// dirty-worktree auto-detection. The caller explicitly requested a
	// specific commit range; unrelated worktree dirt must not override
	// that intent.
	//
	// CORRECTION01: even the explicit-range path delegates to
	// the shared authority resolver so the resolver populates
	// RangeSubjectEnd (the resolved right endpoint) and the
	// downstream renderer can bind the digest subject against
	// the correct endpoint rather than falling back to ambient
	// HEAD.
	if explicitRange != "" {
		result.IsClean = !isDirty
		resolved, err := authority.Resolve(authority.ResolverOptions{
			RepoRoot:       repoRoot,
			ToolBinaryPath: toolPath,
			ExplicitRange:  explicitRange,
		})
		if err != nil {
			return nil, err
		}
		applyAuthorityToResolved(result, resolved)
		return result, nil
	}

	// No explicit range: auto-detect based on worktree state.
	if isDirty {
		result.Mode = ModeDirty
		result.Reason = "working tree has changes"
		result.IsClean = false
		result.AuthorityStatus = authority.AuthorityDirtyWorktree
		return result, nil
	}

	// Working tree is clean: delegate to the shared authority
	// resolver. No heuristic fallback is consulted; the resolver
	// either returns an authoritative range from validated
	// lifecycle artifacts or fails closed with a typed status.
	result.IsClean = true
	resolved, err := authority.Resolve(authority.ResolverOptions{
		RepoRoot:       repoRoot,
		ToolBinaryPath: toolPath,
		ExplicitRange:  explicitRange,
	})
	if err != nil {
		// Surface typed authority errors verbatim so callers can
		// render a precise diagnostic without parsing prose.
		var authErr *authority.AuthorityResolutionError
		if e := err; e != nil {
			if v, ok := e.(*authority.AuthorityResolutionError); ok {
				authErr = v
				_ = authErr
			}
		}
		return nil, err
	}

	applyAuthorityToResolved(result, resolved)
	return result, nil
}

// applyAuthorityToResolved copies the shared authority resolution
// into the legacy ResolvedMode shape used by the digest pipeline.
// The function is intentionally local to resolve.go so the two
// structs evolve together.
func applyAuthorityToResolved(out *ResolvedMode, r *authority.ResolvedAuthority) {
	out.AuthorityStatus = r.AuthorityStatus
	out.ToolIdentity = r.ToolIdentity
	out.ResolutionSource = r.ResolutionSrc
	out.ActID = r.ActID
	out.LifecycleFreeze = r.FreezeCommit
	out.LifecycleSubject = r.SubjectEnd
	// CORRECTION01: propagate the resolved explicit-range right
	// endpoint so the renderer can use it for SUBJECT_BINDING.
	out.LifecycleSubjectRange = r.RangeSubjectEnd
	out.LifecycleClosure = r.ClosureCommit
	out.HeadCommit = r.ToolIdentity.RepositoryHead
	out.GeneratorCommit = r.ToolIdentity.ToolCommit
	if r.SubjectEnd != "" && r.FreezeCommit != "" {
		out.AutoRangeStrategy = authorityStatusStrategy(r.AuthorityStatus)
	}
	if r.AuthorityStatus == authority.AuthorityAuthoritativeClosed ||
		r.AuthorityStatus == authority.AuthorityAuthoritativeClosedLocal {
		out.Mode = ModeRange
		out.Range = r.DigestRange
		out.Reason = authorityStatusReason(r.AuthorityStatus, r)
		out.BaseCommit = r.FreezeCommit
		out.HeadCommit = r.SubjectEnd
	} else if r.AuthorityStatus == authority.AuthorityExplicitRange {
		out.Mode = ModeRange
		out.Range = r.DigestRange
		out.Reason = "explicit --range; non-authoritative"
		out.BaseCommit = ""
		out.HeadCommit = r.ToolIdentity.RepositoryHead
	}
}

// authorityStatusStrategy returns the legacy strategy label
// associated with the typed authority status.
func authorityStatusStrategy(s authority.AuthorityStatus) string {
	switch s {
	case authority.AuthorityAuthoritativeClosed,
		authority.AuthorityAuthoritativeClosedLocal:
		return "closure_manifest"
	case authority.AuthorityExplicitRange:
		return "explicit_cli"
	default:
		return string(s)
	}
}

// authorityStatusReason produces a one-line diagnostic for the
// authority status. Callers render this in legacy "Reason" fields
// without losing the typed classification.
func authorityStatusReason(s authority.AuthorityStatus, r *authority.ResolvedAuthority) string {
	switch s {
	case authority.AuthorityAuthoritativeClosed:
		return "manifest + attestation + annotated tag pin " + r.DigestRange
	case authority.AuthorityAuthoritativeClosedLocal:
		return "manifest pins " + r.DigestRange + " (tag or attestation missing)"
	default:
		return string(s)
	}
}

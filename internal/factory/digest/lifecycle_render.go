// SPDX-License-Identifier: Apache-2.0

// Package digest provides targeted digest generation for Git repositories.
//
// lifecycle_render.go renders the lifecycle metadata required by
// ACT-LEAMAS-FACTORY-DIGEST-AUTO-ACT-RANGE01 into the digest body.
// The fields are surfaced under a dedicated "## LIFECYCLE" section
// so reviewers and downstream tooling can verify how the range was
// selected without having to cross-reference the source code.
//
// ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01 extends
// the section with the generator<->digest-subject binding fields.
// The legacy GENERATOR_STALE flag is preserved at its existing
// position; new keys are appended so existing parsers continue
// to work and the additive contract is documented.
package digest

import (
	"fmt"
	"strings"

	"github.com/s1onique/leamas/internal/factory/authority"
)

// LifecycleField names rendered into the digest. These are part of
// the digest's documented surface area and must remain stable.
const (
	LifecycleFieldAutoRangeStrategy   = "AUTO_RANGE_STRATEGY"
	LifecycleFieldActID               = "ACT_ID"
	LifecycleFieldRangeBase           = "RANGE_BASE"
	LifecycleFieldRangeHead           = "RANGE_HEAD"
	LifecycleFieldRangeReason         = "RANGE_REASON"
	LifecycleFieldFreeze              = "LIFECYCLE_FREEZE"
	LifecycleFieldSubject             = "LIFECYCLE_SUBJECT"
	LifecycleFieldClosure             = "LIFECYCLE_CLOSURE"
	LifecycleFieldIncludedCommits     = "INCLUDED_COMMITS"
	LifecycleFieldGeneratorCommit     = "GENERATOR_COMMIT"
	LifecycleFieldRepositoryHead      = "REPOSITORY_HEAD"
	LifecycleFieldGeneratorStale      = "GENERATOR_STALE"
	LifecycleFieldGeneratorStaleBasis = "GENERATOR_STALE_BASIS"
	LifecycleFieldAuthorityStatus     = "AUTHORITY_STATUS"
	LifecycleFieldResolutionSource    = "RESOLUTION_SOURCE"
	// Generator binding fields (ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01).
	LifecycleFieldGeneratorCommitMatchesHead = "GENERATOR_COMMIT_MATCHES_HEAD"
	LifecycleFieldGeneratorBindingStatus     = "GENERATOR_BINDING_STATUS"
	LifecycleFieldGeneratorCommitBinding     = "GENERATOR_COMMIT_BINDING"
	LifecycleFieldGeneratorSubjectBinding    = "GENERATOR_SUBJECT_BINDING"
	LifecycleFieldGeneratorAuthoritative     = "GENERATOR_AUTHORITATIVE_FOR_DIGEST"
	LifecycleFieldGeneratorWarningCode       = "GENERATOR_WARNING_CODE"
)

// LifecycleSectionHeader is the section heading under which the
// lifecycle metadata appears.
const LifecycleSectionHeader = "## LIFECYCLE"

// generatorStaleBasisLabel documents the legacy GENERATOR_STALE
// semantics. The field is the commit-vs-repository-HEAD signal,
// NOT a digest-subject authority signal. Renderers MUST surface
// this label so reviewers do not silently conflate the two.
const generatorStaleBasisLabel = "commit_vs_repository_head"

// RenderLifecycle renders the lifecycle metadata section. The output
// is deterministic and uses short (12-char) OIDs for readability while
// keeping the strategy label explicit.
func RenderLifecycle(r *ResolvedMode) string {
	if r == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(LifecycleSectionHeader)
	sb.WriteString("\n\n")

	appendKV := func(key, value string) {
		sb.WriteString(fmt.Sprintf("%s: %s\n", key, value))
	}

	appendKV(LifecycleFieldAutoRangeStrategy, renderOrUnset(r.RangeStrategy()))
	appendKV(LifecycleFieldActID, renderOrUnset(r.ActID))
	appendKV(LifecycleFieldRangeBase, renderOrUnset(r.BaseCommit))
	appendKV(LifecycleFieldRangeHead, renderOrUnset(r.HeadCommit))
	appendKV(LifecycleFieldRangeReason, renderOrUnset(r.Reason))
	appendKV(LifecycleFieldFreeze, renderOrUnset(r.LifecycleFreeze))
	appendKV(LifecycleFieldSubject, renderOrUnset(r.LifecycleSubject))
	appendKV(LifecycleFieldClosure, renderOrUnset(r.LifecycleClosure))
	appendKV(LifecycleFieldIncludedCommits, renderCommitsList(r.IncludedCommits))
	appendKV(LifecycleFieldGeneratorCommit, renderOrUnset(r.GeneratorCommit))
	appendKV(LifecycleFieldRepositoryHead, renderOrUnset(r.HeadCommit))
	appendKV(LifecycleFieldGeneratorStale, renderStale(r))
	appendKV(LifecycleFieldGeneratorStaleBasis, generatorStaleBasisLabel)
	// Generator binding fields. Rendered adjacent to the
	// legacy GENERATOR_STALE block so reviewers can verify
	// both claims without scrolling. The values come from the
	// pure EvaluateGeneratorBinding classifier; the renderer
	// MUST NOT recompute the verdict.
	binding := resolveGeneratorBindingForRender(r)
	appendKV(LifecycleFieldGeneratorCommitMatchesHead, renderBool(binding.CommitMatchesHead))
	appendKV(LifecycleFieldGeneratorBindingStatus, renderOrUnset(string(binding.Status)))
	appendKV(LifecycleFieldGeneratorCommitBinding, renderOrUnset(string(binding.CommitBinding)))
	appendKV(LifecycleFieldGeneratorSubjectBinding, renderOrUnset(string(binding.SubjectBinding)))
	appendKV(LifecycleFieldGeneratorAuthoritative, renderBool(binding.AuthoritativeForDigest))
	appendKV(LifecycleFieldGeneratorWarningCode, renderOrUnset(binding.WarningCode))
	appendKV(LifecycleFieldAuthorityStatus, renderOrUnset(string(r.AuthorityStatus)))
	appendKV(LifecycleFieldResolutionSource, renderOrUnset(r.ResolutionSource))
	return sb.String()
}

// resolveGeneratorBindingForRender translates ResolvedMode into
// the typed binding inputs and invokes the pure classifier. The
// function performs no I/O; the caller has already resolved all
// required identities.
//
// Subject resolution is authority-sensitive:
//
//   - AuthorityExplicitRange: the digest subject is the resolved
//     right endpoint of the explicit range (LifecycleSubjectRange).
//     NO fallback to HEAD is permitted. When the right endpoint
//     could not be resolved (RangeSubjectEnd empty), the subject
//     stays empty and the classifier reports IDENTITY_UNBOUND with
//     AUTHORITATIVE=false. This is the CORRECTION02 fail-closed
//     contract: ambiguous explicit-range subjects cannot silently
//     become HEAD authority.
//   - All other authorities (auto, manifest-derived, dirty, etc.):
//     fallback chain is LifecycleSubject -> HeadCommit. The ambient
//     HEAD is the documented subject for the single-commit fallback
//     in clean auto-mode.
//
// CORRECTION01 history: the renderer previously had a flat
// fallback chain LifecycleSubjectRange -> LifecycleSubject ->
// HeadCommit, which silently substituted ambient HEAD for an
// unresolved explicit-range right endpoint. That fallback is now
// suppressed for AuthorityExplicitRange.
func resolveGeneratorBindingForRender(r *ResolvedMode) GeneratorBinding {
	if r == nil {
		return GeneratorBinding{
			Status:                 GeneratorBindingIdentityUnbound,
			CommitBinding:          GeneratorStateUnbound,
			SubjectBinding:         GeneratorStateUnbound,
			CommitMatchesHead:      false,
			AuthoritativeForDigest: false,
			WarningCode:            GeneratorWarningCodeIdentityUnbound,
		}
	}
	generatorCommit := strings.TrimSpace(r.GeneratorCommit)
	repoHead := strings.TrimSpace(r.HeadCommit)
	subjectCommit := resolveSubjectForBinding(r)
	dirty := !r.IsClean
	return ResolveGeneratorBinding(generatorCommit, repoHead, subjectCommit, dirty)
}

// resolveSubjectForBinding returns the digest subject used by the
// generator binding classifier. For AuthorityExplicitRange the
// subject is exclusively LifecycleSubjectRange; if that is empty
// the function returns "" and the classifier reports
// IDENTITY_UNBOUND. For all other authorities the documented
// fallback chain LifecycleSubject -> HeadCommit applies.
//
// CORRECTION02: this helper encapsulates the authority-sensitive
// fallback policy so the renderer does not silently embed a
// classifier policy branch.
func resolveSubjectForBinding(r *ResolvedMode) string {
	if r == nil {
		return ""
	}
	if r.AuthorityStatus == authority.AuthorityExplicitRange {
		// AuthorityExplicitRange: subject is the resolved
		// right endpoint of the explicit range only. No
		// fallback to HEAD is permitted: an unresolved
		// endpoint is definitionally ambiguous and the
		// classifier must report IDENTITY_UNBOUND.
		return strings.TrimSpace(r.LifecycleSubjectRange)
	}
	// All other authorities: documented fallback chain.
	if v := strings.TrimSpace(r.LifecycleSubject); v != "" {
		return v
	}
	return strings.TrimSpace(r.HeadCommit)
}

// renderBool renders a strict boolean as the stable lowercase
// string. We deliberately do NOT render "unset" for false so
// the new fields are machine-distinguishable from missing data.
func renderBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// renderOrUnset renders the value or a sentinel when the field is empty.
func renderOrUnset(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "unset"
	}
	return v
}

// renderCommitsList renders the list of OIDs as space-separated
// short SHAs. "unset" is returned when no commits were recorded.
func renderCommitsList(commits []string) string {
	if len(commits) == 0 {
		return "unset"
	}
	parts := make([]string, 0, len(commits))
	for _, c := range commits {
		parts = append(parts, shortSHA(c))
	}
	return strings.Join(parts, " ")
}

// renderStale renders the GENERATOR_STALE flag plus an explanatory
// reason when stale.
func renderStale(r *ResolvedMode) string {
	if r == nil {
		return "false"
	}
	if !r.GeneratorStale {
		return "false"
	}
	if strings.TrimSpace(r.StaleReason) == "" {
		return "true"
	}
	return "true: " + r.StaleReason
}

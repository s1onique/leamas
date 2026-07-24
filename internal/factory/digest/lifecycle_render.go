// SPDX-License-Identifier: Apache-2.0

// Package digest provides targeted digest generation for Git repositories.
//
// lifecycle_render.go renders the lifecycle metadata required by
// ACT-LEAMAS-FACTORY-DIGEST-AUTO-ACT-RANGE01 into the digest body.
// The fields are surfaced under a dedicated "## LIFECYCLE" section
// so reviewers and downstream tooling can verify how the range was
// selected without having to cross-reference the source code.
package digest

import (
	"fmt"
	"strings"
)

// LifecycleField names rendered into the digest. These are part of
// the digest's documented surface area and must remain stable.
const (
	LifecycleFieldAutoRangeStrategy = "AUTO_RANGE_STRATEGY"
	LifecycleFieldActID             = "ACT_ID"
	LifecycleFieldRangeBase         = "RANGE_BASE"
	LifecycleFieldRangeHead         = "RANGE_HEAD"
	LifecycleFieldRangeReason       = "RANGE_REASON"
	LifecycleFieldFreeze            = "LIFECYCLE_FREEZE"
	LifecycleFieldSubject           = "LIFECYCLE_SUBJECT"
	LifecycleFieldClosure           = "LIFECYCLE_CLOSURE"
	LifecycleFieldIncludedCommits   = "INCLUDED_COMMITS"
	LifecycleFieldGeneratorCommit   = "GENERATOR_COMMIT"
	LifecycleFieldRepositoryHead    = "REPOSITORY_HEAD"
	LifecycleFieldGeneratorStale    = "GENERATOR_STALE"
)

// LifecycleSectionHeader is the section heading under which the
// lifecycle metadata appears.
const LifecycleSectionHeader = "## LIFECYCLE"

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
	return sb.String()
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

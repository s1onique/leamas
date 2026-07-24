// SPDX-License-Identifier: Apache-2.0

// Package digest provides targeted digest generation for Git repositories.
//
// auto_range.go implements the Closure-Protocol-aware auto-range
// selection hierarchy required by ACT-LEAMAS-FACTORY-DIGEST-AUTO-ACT-RANGE01.
//
// The selection hierarchy is:
//
//  1. Explicit --range (handled upstream, before this file is consulted).
//
//  2. Dirty working tree (handled in resolve.go).
//
//  3. Closure-aware clean-tree selection, in this priority order:
//
//     closure_manifest         docs/closure-manifests/<ACT-ID>.json
//     post_closure_attestation docs/closure-manifests/<ACT-ID>.attestation.json
//     annotated_act_tag        refs/tags/act/<ACT-ID> pointing at HEAD
//     act_commit_trailers      docs/close-reports/<ACT-ID>.md structured table
//
//  4. Single-commit fallback HEAD~1..HEAD (verified_single_commit).
//
//  5. Empty-tree fallback 4b825dc..HEAD (initial commit).
//
// When multiple ACTs are candidates at HEAD, the resolver fails closed
// with ErrAmbiguousRange. When no authority can identify the current
// ACT, the resolver fails closed with ErrNoACTAuthority. A stale
// generator binary (binary commit is not an ancestor of repository
// HEAD) is reported via ErrStaleGenerator and refuses auto-mode
// unless explicitly overridden.
package digest

import (
	"errors"
	"regexp"
)

// Range strategy labels rendered into the digest and surfaced in errors.
const (
	StrategyClosureManifest        = "closure_manifest"
	StrategyPostClosureAttestation = "post_closure_attestation"
	StrategyAnnotatedActTag        = "annotated_act_tag"
	StrategyActCommitTrailers      = "act_commit_trailers"
	StrategyVerifiedSingleCommit   = "verified_single_commit"
)

// LifecycleResolution describes the authoritative ACT range selected
// during auto-mode resolution.
//
// Every field is populated deterministically: Strategy is one of the
// Strategy* constants, RangeBase and RangeHead are full OIDs, the
// Lifecycle* fields hold the F/S/C commit OIDs when the strategy
// resolves them, and Generator* describes whether the running Leamas
// binary is current with respect to the repository HEAD.
//
// RangeExpression is the literal Git range expression used for the
// "Range" header in the digest body. The default form is
// "<RangeBase>..<RangeHead>" (full OIDs); the verified_single_commit
// fallback overrides this with "HEAD~1..HEAD" to preserve the
// canonical single-commit surface.
type LifecycleResolution struct {
	Strategy            string
	ActID               string
	RangeBase           string
	RangeHead           string
	RangeExpression     string
	Reason              string
	LifecycleFreeze     string
	LifecycleSubject    string
	LifecycleClosure    string
	IncludedCommits     []string
	GeneratorCommit     string
	RepositoryHead      string
	GeneratorIsAncestor bool
	GeneratorStale      bool
	StaleReason         string
}

// Range returns the Git range expression used by the resolver.
// When RangeExpression is set (for example, the verified_single_commit
// fallback uses "HEAD~1..HEAD") it is returned verbatim. Otherwise
// the expression is built from RangeBase and RangeHead.
func (l *LifecycleResolution) Range() string {
	if l == nil {
		return ""
	}
	if l.RangeExpression != "" {
		return l.RangeExpression
	}
	return l.RangeBase + ".." + l.RangeHead
}

// emptyTreeBaseline is the canonical empty-tree OID used as the
// baseline for the very first commit in a repository.
const emptyTreeBaseline = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// Closure-artifact paths under docs/.
const (
	closureManifestsDir = "docs/closure-manifests"
	closureReportsDir   = "docs/close-reports"
)

// Errors surfaced by the lifecycle resolver. They are typed so the CLI
// can render precise diagnostics.
var (
	// ErrAmbiguousRange is returned when more than one ACT is a
	// plausible candidate for the current HEAD.
	ErrAmbiguousRange = errors.New("digest: unable to resolve one authoritative ACT range")

	// ErrNoACTAuthority is returned when the working tree is
	// clean but no closure artifact identifies the current ACT.
	ErrNoACTAuthority = errors.New("digest: no authoritative ACT for clean tree")

	// ErrStaleGenerator is returned when the embedded LEAMAS_COMMIT
	// is not an ancestor of the repository HEAD.
	ErrStaleGenerator = errors.New("digest: generator binary is stale")

	// ErrShallowBaseline is returned when the resolved range base
	// cannot be reached in the repository (typically a shallow clone).
	ErrShallowBaseline = errors.New("digest: range base not reachable in repository")
)

// actIDPattern matches ACT identifiers like
// "ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01-CORRECTION09".
var actIDPattern = regexp.MustCompile(`^ACT-[A-Z0-9][A-Z0-9-]{2,199}$`)

// fullOIDPattern matches 40+ hex chars used as a full Git SHA.
var fullOIDPattern = regexp.MustCompile(`^[0-9a-f]{40,64}$`)

// shortOIDPattern matches 7+ hex chars used as a short Git SHA. Used
// by extractFirstOID to find OIDs embedded in commit messages or
// close-report prose.
var shortOIDPattern = regexp.MustCompile(`[0-9a-f]{7,64}`)

// manifestLoose is the subset of the closure manifest schema the
// resolver needs. Unknown fields are tolerated.
type manifestLoose struct {
	ActID      string `json:"act_id"`
	PlanFreeze struct {
		FreezeCommit string `json:"freeze_commit"`
	} `json:"plan_freeze"`
	Subject struct {
		CommitOID string `json:"commit_oid"`
	} `json:"subject"`
}

// attestationLoose is the subset of the closure attestation schema the
// resolver needs. Unknown fields are tolerated.
type attestationLoose struct {
	ActID           string `json:"act_id"`
	FreezeReference struct {
		FreezeCommit string `json:"freeze_commit"`
	} `json:"freeze_reference"`
	SubjectReference struct {
		SubjectCommit string `json:"subject_commit"`
	} `json:"subject_reference"`
	ClosureReference struct {
		ClosureCommit string `json:"closure_commit"`
	} `json:"closure_reference"`
}

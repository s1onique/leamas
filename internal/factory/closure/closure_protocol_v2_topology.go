// SPDX-License-Identifier: Apache-2.0

package closure

// closure_protocol_v2_topology.go provides the repository-bound
// topology facts that replace the boolean dispatch in
// closure_protocol_v2.go. Topology facts are derived from real
// git operations in the target repository and never from
// caller-supplied booleans.
//
// Splitting this from closure_protocol_v2.go keeps the file
// under the LLM-friendly 400-line threshold while preserving
// the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import (
	"context"
	"fmt"
	"strings"
)

// V2TopologyFacts captures the resolved topology between the
// subject S and the freeze F in the target repository. The
// booleans are mutually exclusive facts derived from bounded
// git operations; SubjectResolved / FreezeResolved report
// whether the supplied revisions identified a real commit.
//
// The fields are deliberately distinct from the old
// V2DispatchResult so machine handling never confuses
// "resolved" with "passed topology".
type V2TopologyFacts struct {
	SubjectResolved       bool
	FreezeResolved        bool
	Equal                 bool
	SubjectAncestorFreeze bool
	FreezeAncestorSubject bool
	// SubjectCommit holds the resolved subject commit OID when
	// SubjectResolved is true; it is empty otherwise.
	SubjectCommit string
	// FreezeCommit holds the resolved freeze commit OID when
	// FreezeResolved is true; it is empty otherwise.
	FreezeCommit string
}

// V2Relation enumerates the seven distinguished topology
// relations. Exactly one relation applies to every
// V2TopologyFacts value.
type V2Relation string

const (
	V2RelationMissingSubject         V2Relation = "missing_subject"
	V2RelationMissingFreeze          V2Relation = "missing_freeze"
	V2RelationEqual                  V2Relation = "equal"
	V2RelationSubjectBeforeFreeze    V2Relation = "subject_before_freeze"
	V2RelationFreezeBeforeSubject    V2Relation = "freeze_before_subject"
	V2RelationSubjectFreezeUnrelated V2Relation = "subject_freeze_unrelated"
	V2RelationGitFailure             V2Relation = "git_failure"
)

// Classify maps a V2TopologyFacts value to its single V2Relation.
// The classification is deterministic and total; tests assert
// on the returned token rather than internal booleans.
func (f V2TopologyFacts) Classify() V2Relation {
	if !f.SubjectResolved {
		return V2RelationMissingSubject
	}
	if !f.FreezeResolved {
		return V2RelationMissingFreeze
	}
	if f.Equal {
		return V2RelationEqual
	}
	if f.SubjectAncestorFreeze && !f.FreezeAncestorSubject {
		return V2RelationSubjectBeforeFreeze
	}
	if f.FreezeAncestorSubject && !f.SubjectAncestorFreeze {
		return V2RelationFreezeBeforeSubject
	}
	if !f.SubjectAncestorFreeze && !f.FreezeAncestorSubject {
		return V2RelationSubjectFreezeUnrelated
	}
	return V2RelationGitFailure
}

// V2TopologyResolver resolves topology facts in the target
// repository. Implementations MUST use bounded git operations
// and never accept caller-supplied ancestry booleans.
type V2TopologyResolver interface {
	ResolveTopology(ctx context.Context, repoRoot, subject, freeze string) (V2TopologyFacts, error)
}

// GitV2TopologyResolver resolves V2TopologyFacts against a real
// git repository using the supplied git client.
type GitV2TopologyResolver struct {
	Git gitClient
}

// NewGitV2TopologyResolver builds a GitV2TopologyResolver that
// uses the provided git client. A nil client defaults to the
// RealGit implementation.
func NewGitV2TopologyResolver(g gitClient) *GitV2TopologyResolver {
	if g == nil {
		g = RealGit{}
	}
	return &GitV2TopologyResolver{Git: g}
}

// ResolveTopology resolves both supplied revisions into commit
// OIDs and evaluates the directed ancestor predicates via
// `git merge-base --is-ancestor`.
//
// The function never falls back to caller-supplied facts. If
// either revision fails to resolve to a commit, the
// corresponding resolved flag is false and Classify returns
// the missing relation. If git itself fails (binary missing,
// permission error, repository corruption) the function returns
// an error wrapping V2CodeGitOperationFailed so the caller can
// distinguish git failure from missing inputs.
func (r *GitV2TopologyResolver) ResolveTopology(ctx context.Context, repoRoot, subject, freeze string) (V2TopologyFacts, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return V2TopologyFacts{}, NewV2Error(V2CodeGitOperationFailed, "repository root is empty")
	}
	if strings.TrimSpace(subject) == "" {
		return V2TopologyFacts{}, NewV2ErrorWith(V2CodeSubjectCommitNotFound, "subject commit is empty", "subject_commit", "")
	}
	if strings.TrimSpace(freeze) == "" {
		return V2TopologyFacts{}, NewV2ErrorWith(V2CodeFreezeCommitNotFound, "freeze commit is empty", "freeze_commit", "")
	}
	subjectCommit, err := resolveTopologyCommit(ctx, r.Git, repoRoot, subject)
	if err != nil {
		// Missing revisions are not errors: report as facts with
		// SubjectResolved=false so Classify emits missing_subject.
		return V2TopologyFacts{SubjectResolved: false}, nil
	}
	freezeCommit, err := resolveTopologyCommit(ctx, r.Git, repoRoot, freeze)
	if err != nil {
		return V2TopologyFacts{
			SubjectResolved: true,
			SubjectCommit:   subjectCommit,
			FreezeResolved:  false,
		}, nil
	}
	facts := V2TopologyFacts{
		SubjectResolved: true,
		SubjectCommit:   subjectCommit,
		FreezeResolved:  true,
		FreezeCommit:    freezeCommit,
	}
	if subjectCommit == freezeCommit {
		facts.Equal = true
		return facts, nil
	}
	subjectAncestorFreeze, err := gitIsAncestor(ctx, r.Git, repoRoot, subjectCommit, freezeCommit)
	if err != nil {
		return V2TopologyFacts{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("git merge-base --is-ancestor %s %s failed", subjectCommit, freezeCommit),
			"subject_commit", err.Error())
	}
	freezeAncestorSubject, err := gitIsAncestor(ctx, r.Git, repoRoot, freezeCommit, subjectCommit)
	if err != nil {
		return V2TopologyFacts{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("git merge-base --is-ancestor %s %s failed", freezeCommit, subjectCommit),
			"freeze_commit", err.Error())
	}
	facts.SubjectAncestorFreeze = subjectAncestorFreeze
	facts.FreezeAncestorSubject = freezeAncestorSubject
	return facts, nil
}

// SubjectCommit returns the resolved subject commit OID. The
// string is empty when SubjectResolved is false.
func (f V2TopologyFacts) SubjectCommitValue() string { return f.SubjectCommit }

// FreezeCommit returns the resolved freeze commit OID. The
// string is empty when FreezeResolved is false.
func (f V2TopologyFacts) FreezeCommitValue() string { return f.FreezeCommit }

// resolveTopologyCommit resolves an arbitrary revision
// expression to its full commit OID. It returns the typed
// V2Error so callers can distinguish revision failures from
// downstream errors.
func resolveTopologyCommit(ctx context.Context, git gitClient, repoRoot, rev string) (string, error) {
	value, err := runGitValue(ctx, git, repoRoot, "rev-parse", "--verify", "--end-of-options", rev+"^{commit}")
	if err != nil {
		return "", err
	}
	if !oidPattern.MatchString(value) {
		return "", fmt.Errorf("resolved %q is not a valid commit OID", value)
	}
	return value, nil
}

// gitIsAncestor returns true iff the first revision is an
// ancestor of the second according to git merge-base.
//
// It returns an error only when git itself fails so callers
// can distinguish git failure (binary missing, permission
// error) from a non-ancestor verdict. Exit code 1 means
// "not an ancestor" and is NOT treated as a failure.
func gitIsAncestor(ctx context.Context, git gitClient, repoRoot, ancestor, descendant string) (bool, error) {
	result := git.Run(ctx, repoRoot, "merge-base", "--is-ancestor", ancestor, descendant)
	if result.ExitCode == 0 {
		return true, nil
	}
	if result.ExitCode == 1 {
		return false, nil
	}
	detail := strings.TrimSpace(string(result.Stderr))
	if detail == "" && result.Err != nil {
		detail = result.Err.Error()
	}
	return false, fmt.Errorf("git merge-base --is-ancestor %s %s exit=%d stderr=%q",
		ancestor, descendant, result.ExitCode, detail)
}

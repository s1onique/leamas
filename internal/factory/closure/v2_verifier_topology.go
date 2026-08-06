// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_topology.go implements the immutable
// Git-object topology required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-TOPOLOGY-OBJECTS01.
//
// The verifier MUST classify S, F, C as a directed triple:
//
//	S < F < C
//
// and reject every other configuration with a typed
// diagnostic. The classifier probes both directed ancestor
// relations per pair and a third "equal" relation per pair so
// reverse, equal, and unrelated topologies never collapse into
// a single "invalid_topology" bucket.
//
// Every observation runs through the repository-bound
// V2ClosureGitAuthority published by the foundation ACT.
// Process CWD is never used as authority.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// V2ClosureTopology captures the resolved immutable Git-object
// topology between the subject S, the freeze F, and the closure
// C. The bools are the two directed ancestor relations that
// must hold for the accepted triple; reverse and equal
// relations are tracked internally by the resolver (see
// topologyTriple) because exposing them publicly would invite
// callers to smuggle in reverse-topology facts.
//
// The accepted triple is S < F < C, requiring all three
// distinct and the two directed ancestor relations true.
type V2ClosureTopology struct {
	SubjectCommit string
	SubjectTree   string

	FreezeCommit string
	FreezeTree   string

	ClosureCommit string
	ClosureTree   string

	SubjectAncestorFreeze bool
	FreezeAncestorClosure bool
}

// V2ClosureRelation enumerates the distinguished triples the
// verifier may observe. The taxonomy is stable: ACT 3 / ACT 4
// branch on these tokens rather than re-deriving facts from
// booleans.
//
// Exactly one relation applies to every resolved triple. The
// unresolved / failure relations are reported before topology
// classification so the CLI can render distinct messages.
type V2ClosureRelation string

const (
	// V2ClosureRelationSubjectBeforeFreezeBeforeClosure is the
	// only accepted relation: S < F < C with all three
	// distinct.
	V2ClosureRelationSubjectBeforeFreezeBeforeClosure V2ClosureRelation = "subject_before_freeze_before_closure"

	// V2ClosureRelationSubjectFreezeEqual reports S == F.
	// Rejected.
	V2ClosureRelationSubjectFreezeEqual V2ClosureRelation = "subject_freeze_equal"

	// V2ClosureRelationFreezeClosureEqual reports F == C.
	// Rejected.
	V2ClosureRelationFreezeClosureEqual V2ClosureRelation = "freeze_closure_equal"

	// V2ClosureRelationSubjectClosureEqual reports S == C.
	// Rejected.
	V2ClosureRelationSubjectClosureEqual V2ClosureRelation = "subject_closure_equal"

	// V2ClosureRelationFreezeBeforeSubject reports F < S.
	// Rejected (reverse subject-freeze topology).
	V2ClosureRelationFreezeBeforeSubject V2ClosureRelation = "freeze_before_subject"

	// V2ClosureRelationClosureBeforeFreeze reports C < F.
	// Rejected (reverse freeze-closure topology).
	V2ClosureRelationClosureBeforeFreeze V2ClosureRelation = "closure_before_freeze"

	// V2ClosureRelationSubjectFreezeUnrelated reports S
	// and F unrelated. Rejected.
	V2ClosureRelationSubjectFreezeUnrelated V2ClosureRelation = "subject_freeze_unrelated"

	// V2ClosureRelationFreezeClosureUnrelated reports F
	// and C unrelated. Rejected.
	V2ClosureRelationFreezeClosureUnrelated V2ClosureRelation = "freeze_closure_unrelated"

	// V2ClosureRelationSubjectUnresolved reports S failed
	// to resolve to a commit.
	V2ClosureRelationSubjectUnresolved V2ClosureRelation = "subject_unresolved"

	// V2ClosureRelationFreezeUnresolved reports F failed
	// to resolve to a commit.
	V2ClosureRelationFreezeUnresolved V2ClosureRelation = "freeze_unresolved"

	// V2ClosureRelationClosureUnresolved reports C failed
	// to resolve to a commit.
	V2ClosureRelationClosureUnresolved V2ClosureRelation = "closure_unresolved"

	// V2ClosureRelationGitFailure reports a downstream
	// observation failure (timeout, cancellation, spawn
	// error, output overflow, or non {0,1} exit from
	// merge-base --is-ancestor).
	V2ClosureRelationGitFailure V2ClosureRelation = "git_failure"
)

// IsAccepted reports whether the relation is the single
// accepted triple. The verifier MUST reject every other
// relation regardless of supporting diagnostics.
func (r V2ClosureRelation) IsAccepted() bool {
	return r == V2ClosureRelationSubjectBeforeFreezeBeforeClosure
}

// topologyTriple is the resolver-internal triple carrying
// both directed ancestor relations per pair. The reverse
// facts are package-private so callers cannot smuggle in a
// reverse-topology fact; only the resolver can construct a
// topologyTriple.
//
// The embedded V2ClosureTopology exposes the public fields
// to ACT 3 / ACT 4 without leaking the resolver-internal
// flags.
type topologyTriple struct {
	V2ClosureTopology
	FreezeAncestorSubject bool
	ClosureAncestorFreeze bool
}

// classifyTriple maps a fully-probed topologyTriple to its
// single V2ClosureRelation. The classifier is total and
// deterministic; every reachable triple produces exactly one
// token.
//
// Classification precedence:
//
//  1. missing S / F / C
//  2. S == F / F == C / S == C
//  3. reverse relations (F < S, C < F)
//  4. unrelated pairs (S vs F, F vs C)
//  5. accepted triple S < F < C
func classifyTriple(t topologyTriple) V2ClosureRelation {
	if t.SubjectCommit == "" {
		return V2ClosureRelationSubjectUnresolved
	}
	if t.FreezeCommit == "" {
		return V2ClosureRelationFreezeUnresolved
	}
	if t.ClosureCommit == "" {
		return V2ClosureRelationClosureUnresolved
	}
	if t.SubjectCommit == t.FreezeCommit {
		return V2ClosureRelationSubjectFreezeEqual
	}
	if t.FreezeCommit == t.ClosureCommit {
		return V2ClosureRelationFreezeClosureEqual
	}
	if t.SubjectCommit == t.ClosureCommit {
		return V2ClosureRelationSubjectClosureEqual
	}
	if t.FreezeAncestorSubject {
		return V2ClosureRelationFreezeBeforeSubject
	}
	if t.ClosureAncestorFreeze {
		return V2ClosureRelationClosureBeforeFreeze
	}
	if !t.SubjectAncestorFreeze {
		return V2ClosureRelationSubjectFreezeUnrelated
	}
	if !t.FreezeAncestorClosure {
		return V2ClosureRelationFreezeClosureUnrelated
	}
	return V2ClosureRelationSubjectBeforeFreezeBeforeClosure
}

// V2ClosureTopologyFacts is the structured verdict returned
// by the resolver. The fields are populated by the resolver
// after the underlying Git observations complete; tests assert
// on Relation plus the verbatim OID fields rather than on the
// internal triple struct.
type V2ClosureTopologyFacts struct {
	Topology    V2ClosureTopology
	Relation    V2ClosureRelation
	Diagnostics V2VerifierDiagnostics
}

// V2ClosureTopologyResolver resolves the S/F/C triple against
// the repository bound to the supplied Git authority. The
// resolver probes both directed relations per pair (S -> F,
// F -> S, F -> C, C -> F) so reverse, equal, and unrelated
// topologies never collapse into a single bucket.
//
// The resolver never accepts caller-supplied ancestry facts;
// every fact is derived from bounded git operations through
// the bound authority.
type V2ClosureTopologyResolver interface {
	ResolveTopology(
		ctx context.Context,
		authority V2ClosureGitAuthority,
		request V2ClosureVerifyRequest,
	) (V2ClosureTopologyFacts, error)
}

// gitV2ClosureTopologyResolver is the production
// V2ClosureTopologyResolver. The struct has no fields; the
// authority passed to ResolveTopology carries the
// repository-binding contract.
type gitV2ClosureTopologyResolver struct{}

// NewV2ClosureTopologyResolver builds a production resolver.
// The resolver has no fields; the repository-bound authority
// is supplied per-call so the resolver is safe to share
// across multiple invocations against different repositories.
func NewV2ClosureTopologyResolver() V2ClosureTopologyResolver {
	return gitV2ClosureTopologyResolver{}
}

// ResolveTopology classifies the S/F/C triple derived from
// the supplied request.
//
// Resolution order:
//
//  1. Resolve S, F, C to canonical 40-char OIDs via the bound
//     authority. A failure emits the typed diagnostic and the
//     corresponding unresolved relation.
//  2. Resolve the tree of each commit. A failure emits
//     V2VerifierTopologyObservationFailed and the git_failure
//     relation (topology cannot be classified without trees).
//  3. Probe S -> F (merge-base --is-ancestor S F).
//  4. Probe F -> S.
//  5. Probe F -> C.
//  6. Probe C -> F.
//
// On the accepted relation, the resolver returns a non-empty
// V2ClosureTopologyFacts with Diagnostics empty. Every other
// relation emits at least one typed diagnostic so the CLI can
// surface the precise rejection reason.
func (gitV2ClosureTopologyResolver) ResolveTopology(
	ctx context.Context,
	authority V2ClosureGitAuthority,
	request V2ClosureVerifyRequest,
) (V2ClosureTopologyFacts, error) {
	if authority == nil {
		return V2ClosureTopologyFacts{}, NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierTopologyObservationFailed,
			"git authority is required",
		))
	}

	subjectCommit, err := authority.ResolveCommit(ctx, request.SubjectCommit)
	if err != nil {
		return V2ClosureTopologyFacts{
			Relation:    V2ClosureRelationSubjectUnresolved,
			Diagnostics: relationDiagnostics(V2ClosureRelationSubjectUnresolved, request.SubjectCommit, request.FreezeCommit, request.ClosureCommit),
		}, nil
	}
	subjectTree, err := authority.ResolveTree(ctx, subjectCommit)
	if err != nil {
		return V2ClosureTopologyFacts{
			Relation: V2ClosureRelationGitFailure,
			Topology: V2ClosureTopology{
				SubjectCommit: subjectCommit,
			},
			Diagnostics: V2VerifierDiagnostics{
				NewV2VerifierDiagnostic(
					V2VerifierTopologyObservationFailed,
					"subject tree resolution failed",
				).withObserved(subjectCommit),
			},
		}, nil
	}

	freezeCommit, err := authority.ResolveCommit(ctx, request.FreezeCommit)
	if err != nil {
		return V2ClosureTopologyFacts{
			Relation: V2ClosureRelationFreezeUnresolved,
			Topology: V2ClosureTopology{
				SubjectCommit: subjectCommit,
				SubjectTree:   subjectTree,
			},
			Diagnostics: relationDiagnostics(V2ClosureRelationFreezeUnresolved, subjectCommit, request.FreezeCommit, request.ClosureCommit),
		}, nil
	}
	freezeTree, err := authority.ResolveTree(ctx, freezeCommit)
	if err != nil {
		return V2ClosureTopologyFacts{
			Relation: V2ClosureRelationGitFailure,
			Topology: V2ClosureTopology{
				SubjectCommit: subjectCommit,
				SubjectTree:   subjectTree,
				FreezeCommit:  freezeCommit,
			},
			Diagnostics: V2VerifierDiagnostics{
				NewV2VerifierDiagnostic(
					V2VerifierTopologyObservationFailed,
					"freeze tree resolution failed",
				).withObserved(freezeCommit),
			},
		}, nil
	}

	closureCommit, err := authority.ResolveCommit(ctx, request.ClosureCommit)
	if err != nil {
		return V2ClosureTopologyFacts{
			Relation: V2ClosureRelationClosureUnresolved,
			Topology: V2ClosureTopology{
				SubjectCommit: subjectCommit,
				SubjectTree:   subjectTree,
				FreezeCommit:  freezeCommit,
				FreezeTree:    freezeTree,
			},
			Diagnostics: relationDiagnostics(V2ClosureRelationClosureUnresolved, subjectCommit, freezeCommit, request.ClosureCommit),
		}, nil
	}
	closureTree, err := authority.ResolveTree(ctx, closureCommit)
	if err != nil {
		return V2ClosureTopologyFacts{
			Relation: V2ClosureRelationGitFailure,
			Topology: V2ClosureTopology{
				SubjectCommit: subjectCommit,
				SubjectTree:   subjectTree,
				FreezeCommit:  freezeCommit,
				FreezeTree:    freezeTree,
				ClosureCommit: closureCommit,
			},
			Diagnostics: V2VerifierDiagnostics{
				NewV2VerifierDiagnostic(
					V2VerifierTopologyObservationFailed,
					"closure tree resolution failed",
				).withObserved(closureCommit),
			},
		}, nil
	}

	// Probe both directed relations per pair so the
	// classifier can distinguish reverse, equal, and
	// unrelated topologies.
	subjectAncestorFreeze, err := authority.IsAncestor(ctx, subjectCommit, freezeCommit)
	if err != nil {
		return V2ClosureTopologyFacts{
			Relation: V2ClosureRelationGitFailure,
			Topology: V2ClosureTopology{
				SubjectCommit: subjectCommit,
				SubjectTree:   subjectTree,
				FreezeCommit:  freezeCommit,
				FreezeTree:    freezeTree,
				ClosureCommit: closureCommit,
				ClosureTree:   closureTree,
			},
			Diagnostics: V2VerifierDiagnostics{
				NewV2VerifierDiagnostic(
					V2VerifierTopologyObservationFailed,
					"S->F ancestry observation failed",
				).withObserved(subjectCommit + ">" + freezeCommit),
			},
		}, nil
	}
	freezeAncestorSubject, err := authority.IsAncestor(ctx, freezeCommit, subjectCommit)
	if err != nil {
		return V2ClosureTopologyFacts{
			Relation: V2ClosureRelationGitFailure,
			Topology: V2ClosureTopology{
				SubjectCommit: subjectCommit,
				SubjectTree:   subjectTree,
				FreezeCommit:  freezeCommit,
				FreezeTree:    freezeTree,
				ClosureCommit: closureCommit,
				ClosureTree:   closureTree,
			},
			Diagnostics: V2VerifierDiagnostics{
				NewV2VerifierDiagnostic(
					V2VerifierTopologyObservationFailed,
					"F->S ancestry observation failed",
				).withObserved(freezeCommit + ">" + subjectCommit),
			},
		}, nil
	}
	freezeAncestorClosure, err := authority.IsAncestor(ctx, freezeCommit, closureCommit)
	if err != nil {
		return V2ClosureTopologyFacts{
			Relation: V2ClosureRelationGitFailure,
			Topology: V2ClosureTopology{
				SubjectCommit: subjectCommit,
				SubjectTree:   subjectTree,
				FreezeCommit:  freezeCommit,
				FreezeTree:    freezeTree,
				ClosureCommit: closureCommit,
				ClosureTree:   closureTree,
			},
			Diagnostics: V2VerifierDiagnostics{
				NewV2VerifierDiagnostic(
					V2VerifierTopologyObservationFailed,
					"F->C ancestry observation failed",
				).withObserved(freezeCommit + ">" + closureCommit),
			},
		}, nil
	}
	closureAncestorFreeze, err := authority.IsAncestor(ctx, closureCommit, freezeCommit)
	if err != nil {
		return V2ClosureTopologyFacts{
			Relation: V2ClosureRelationGitFailure,
			Topology: V2ClosureTopology{
				SubjectCommit: subjectCommit,
				SubjectTree:   subjectTree,
				FreezeCommit:  freezeCommit,
				FreezeTree:    freezeTree,
				ClosureCommit: closureCommit,
				ClosureTree:   closureTree,
			},
			Diagnostics: V2VerifierDiagnostics{
				NewV2VerifierDiagnostic(
					V2VerifierTopologyObservationFailed,
					"C->F ancestry observation failed",
				).withObserved(closureCommit + ">" + freezeCommit),
			},
		}, nil
	}

	topology := V2ClosureTopology{
		SubjectCommit:         subjectCommit,
		SubjectTree:           subjectTree,
		FreezeCommit:          freezeCommit,
		FreezeTree:            freezeTree,
		ClosureCommit:         closureCommit,
		ClosureTree:           closureTree,
		SubjectAncestorFreeze: subjectAncestorFreeze,
		FreezeAncestorClosure: freezeAncestorClosure,
	}
	triple := topologyTriple{
		V2ClosureTopology:     topology,
		FreezeAncestorSubject: freezeAncestorSubject,
		ClosureAncestorFreeze: closureAncestorFreeze,
	}
	relation := classifyTriple(triple)

	facts := V2ClosureTopologyFacts{
		Topology: topology,
		Relation: relation,
	}
	if !relation.IsAccepted() {
		facts.Diagnostics = relationDiagnostics(relation, subjectCommit, freezeCommit, closureCommit)
	}
	return facts, nil
}

// relationDiagnostics maps a rejected relation to the typed
// diagnostic code emitted by the foundation ACT. The codes
// match V2VerifierCode exactly so the CLI does not need to
// inspect the relation token twice.
func relationDiagnostics(
	relation V2ClosureRelation,
	subject, freeze, closure string,
) V2VerifierDiagnostics {
	switch relation {
	case V2ClosureRelationSubjectFreezeEqual:
		return V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierSubjectFreezeEqual,
			"subject and freeze commits are equal",
		).withObserved(subject + "==" + freeze)}
	case V2ClosureRelationFreezeClosureEqual:
		return V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierFreezeClosureEqual,
			"freeze and closure commits are equal",
		).withObserved(freeze + "==" + closure)}
	case V2ClosureRelationSubjectClosureEqual:
		return V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierSubjectClosureEqual,
			"subject and closure commits are equal",
		).withObserved(subject + "==" + closure)}
	case V2ClosureRelationFreezeBeforeSubject:
		return V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierReverseSubjectFreezeTopology,
			"freeze is an ancestor of subject (reverse topology)",
		).withObserved(freeze + ">" + subject)}
	case V2ClosureRelationClosureBeforeFreeze:
		return V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierReverseFreezeClosureTopology,
			"closure is an ancestor of freeze (reverse topology)",
		).withObserved(closure + ">" + freeze)}
	case V2ClosureRelationSubjectFreezeUnrelated:
		return V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierSubjectFreezeUnrelated,
			"subject and freeze are unrelated",
		).withObserved(subject + "?" + freeze)}
	case V2ClosureRelationFreezeClosureUnrelated:
		return V2VerifierDiagnostics{
			NewV2VerifierDiagnostic(
				V2VerifierFreezeClosureUnrelated,
				"freeze and closure are unrelated",
			).withObserved(freeze + "?" + closure),
			NewV2VerifierDiagnostic(
				V2VerifierFreezeNotAncestorClosure,
				"freeze is not an ancestor of closure",
			).withObserved(freeze + "?" + closure),
		}
	case V2ClosureRelationSubjectUnresolved:
		return V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierSubjectUnresolved,
			"subject commit does not resolve",
		).withObserved(subject)}
	case V2ClosureRelationFreezeUnresolved:
		return V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierFreezeUnresolved,
			"freeze commit does not resolve",
		).withObserved(freeze)}
	case V2ClosureRelationClosureUnresolved:
		return V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierClosureUnresolved,
			"closure commit does not resolve",
		).withObserved(closure)}
	case V2ClosureRelationGitFailure:
		return V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierTopologyObservationFailed,
			"topology observation failed",
		)}
	}
	return nil
}

// V2FrozenPlanAuthority captures the byte authority of the
// frozen plan at F:P. The verifier loads the plan bytes
// exclusively from F:PlanPath; the working tree, HEAD:P, and
// C:P are never queried.
//
// All fields are required when the verifier reports a valid
// resolution. The diagnostic slice is non-empty when the
// authority could not be resolved; otherwise it is nil.
type V2FrozenPlanAuthority struct {
	Path        string
	BlobOID     string
	BlobSHA256  string
	RawBytes    []byte
	Diagnostics V2VerifierDiagnostics
}

// V2CommittedManifestAuthority captures the byte authority of
// the committed manifest at C:M. The verifier loads the
// manifest bytes exclusively from C:ManifestPath; the working
// tree, HEAD:M, F:M, and any optional assertion bytes are
// never queried.
//
// All fields are required when the verifier reports a valid
// resolution. The diagnostic slice is non-empty when the
// authority could not be resolved; otherwise it is nil.
type V2CommittedManifestAuthority struct {
	Path        string
	BlobOID     string
	BlobSHA256  string
	RawBytes    []byte
	Diagnostics V2VerifierDiagnostics
}

// ResolveV2FrozenPlanAuthority resolves the frozen plan
// authority at F:PlanPath. The function never reads the
// working tree, HEAD:P, or C:P.
//
// On success the returned V2FrozenPlanAuthority carries the
// exact raw bytes read via `git cat-file blob <oid>`. On
// failure the diagnostic slice names the rejection reason;
// RawBytes is nil.
//
// The byte contract is preserved end-to-end: trailing
// newlines, leading whitespace, and trailing spaces are never
// trimmed so SHA-256(raw) equals the binding in the committed
// manifest at C.
func ResolveV2FrozenPlanAuthority(
	ctx context.Context,
	authority V2ClosureGitAuthority,
	freezeCommit string,
	planPath string,
) (V2FrozenPlanAuthority, error) {
	if authority == nil {
		return V2FrozenPlanAuthority{}, NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierFrozenPlanMissing,
			"git authority is required",
		))
	}
	if strings.TrimSpace(freezeCommit) == "" {
		return V2FrozenPlanAuthority{}, NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierFrozenPlanMissing,
			"freeze commit is empty",
		))
	}
	if strings.TrimSpace(planPath) == "" {
		return V2FrozenPlanAuthority{}, NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierFrozenPlanMissing,
			"plan path is empty",
		))
	}

	oid, objectType, err := authority.ResolvePathObject(ctx, freezeCommit, planPath)
	if err != nil {
		verr, ok := err.(*V2VerifierError)
		if ok && len(verr.Diags) > 0 {
			return V2FrozenPlanAuthority{
				Path:        planPath,
				Diagnostics: V2VerifierDiagnostics{verr.Diags[0]},
			}, nil
		}
		return V2FrozenPlanAuthority{}, WrapV2VerifierError(
			NewV2VerifierDiagnostic(V2VerifierFrozenPlanMissing, "frozen plan path resolution failed"),
			err,
		)
	}
	if objectType != "blob" {
		return V2FrozenPlanAuthority{
			Path:    planPath,
			BlobOID: oid,
			Diagnostics: V2VerifierDiagnostics{NewV2VerifierDiagnostic(
				V2VerifierFrozenPlanNotBlob,
				"frozen plan path is not a blob at F",
			).withObjectPath(planPath).withObjectOID(oid)},
		}, nil
	}

	raw, err := authority.ReadBlob(ctx, oid)
	if err != nil {
		verr, ok := err.(*V2VerifierError)
		if ok && len(verr.Diags) > 0 {
			return V2FrozenPlanAuthority{
				Path:        planPath,
				BlobOID:     oid,
				Diagnostics: V2VerifierDiagnostics{verr.Diags[0]},
			}, nil
		}
		return V2FrozenPlanAuthority{}, WrapV2VerifierError(
			NewV2VerifierDiagnostic(V2VerifierFrozenPlanReadFailed, "frozen plan blob read failed"),
			err,
		)
	}
	if len(raw) == 0 {
		return V2FrozenPlanAuthority{
			Path:     planPath,
			BlobOID:  oid,
			RawBytes: raw,
			Diagnostics: V2VerifierDiagnostics{NewV2VerifierDiagnostic(
				V2VerifierFrozenPlanReadFailed,
				"frozen plan blob is empty",
			).withObjectPath(planPath).withObjectOID(oid)},
		}, nil
	}

	sum := sha256.Sum256(raw)
	return V2FrozenPlanAuthority{
		Path:       planPath,
		BlobOID:    oid,
		BlobSHA256: hex.EncodeToString(sum[:]),
		RawBytes:   append([]byte(nil), raw...),
	}, nil
}

// ResolveV2CommittedManifestAuthority resolves the committed
// manifest authority at C:ManifestPath. The function never
// reads the working tree, HEAD:M, F:M, or the optional disk
// assertion. ACT 3 consumes the raw bytes; ACT 2 only needs
// to prove the bytes come from C:M and are preserved
// verbatim.
//
// On success the returned V2CommittedManifestAuthority
// carries the exact raw bytes. On failure the diagnostic
// slice names the rejection reason.
func ResolveV2CommittedManifestAuthority(
	ctx context.Context,
	authority V2ClosureGitAuthority,
	closureCommit string,
	manifestPath string,
) (V2CommittedManifestAuthority, error) {
	if authority == nil {
		return V2CommittedManifestAuthority{}, NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierClosureManifestMissing,
			"git authority is required",
		))
	}
	if strings.TrimSpace(closureCommit) == "" {
		return V2CommittedManifestAuthority{}, NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierClosureManifestMissing,
			"closure commit is empty",
		))
	}
	if strings.TrimSpace(manifestPath) == "" {
		return V2CommittedManifestAuthority{}, NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierClosureManifestMissing,
			"manifest path is empty",
		))
	}

	oid, objectType, err := authority.ResolvePathObject(ctx, closureCommit, manifestPath)
	if err != nil {
		// Translate the underlying diagnostic code into
		// the closure_manifest_missing namespace; the
		// shared V2ClosureGitAuthority resolver emits
		// frozen_plan_missing for any path-resolution
		// failure, but the manifest authority needs to
		// surface closure_manifest_missing so the CLI
		// can route the rejection correctly.
		verr, ok := err.(*V2VerifierError)
		if ok && len(verr.Diags) > 0 {
			translated := V2VerifierDiagnostic{
				Code:         V2VerifierClosureManifestMissing,
				Message:      "committed manifest path does not resolve at C: " + verr.Diags[0].Message,
				PropertyName: verr.Diags[0].PropertyName,
				ObjectOID:    verr.Diags[0].ObjectOID,
				ObjectPath:   manifestPath,
				Detail:       verr.Diags[0].Detail,
			}
			return V2CommittedManifestAuthority{
				Path:        manifestPath,
				Diagnostics: V2VerifierDiagnostics{translated},
			}, nil
		}
		return V2CommittedManifestAuthority{}, WrapV2VerifierError(
			NewV2VerifierDiagnostic(V2VerifierClosureManifestMissing, "committed manifest path resolution failed"),
			err,
		)
	}
	if objectType != "blob" {
		return V2CommittedManifestAuthority{
			Path:    manifestPath,
			BlobOID: oid,
			Diagnostics: V2VerifierDiagnostics{NewV2VerifierDiagnostic(
				V2VerifierClosureManifestNotBlob,
				"committed manifest path is not a blob at C",
			).withObjectPath(manifestPath).withObjectOID(oid)},
		}, nil
	}

	raw, err := authority.ReadBlob(ctx, oid)
	if err != nil {
		verr, ok := err.(*V2VerifierError)
		if ok && len(verr.Diags) > 0 {
			return V2CommittedManifestAuthority{
				Path:        manifestPath,
				BlobOID:     oid,
				Diagnostics: V2VerifierDiagnostics{verr.Diags[0]},
			}, nil
		}
		return V2CommittedManifestAuthority{}, WrapV2VerifierError(
			NewV2VerifierDiagnostic(V2VerifierClosureManifestReadFailed, "committed manifest blob read failed"),
			err,
		)
	}
	if len(raw) == 0 {
		return V2CommittedManifestAuthority{
			Path:     manifestPath,
			BlobOID:  oid,
			RawBytes: raw,
			Diagnostics: V2VerifierDiagnostics{NewV2VerifierDiagnostic(
				V2VerifierClosureManifestReadFailed,
				"committed manifest blob is empty",
			).withObjectPath(manifestPath).withObjectOID(oid)},
		}, nil
	}

	sum := sha256.Sum256(raw)
	return V2CommittedManifestAuthority{
		Path:       manifestPath,
		BlobOID:    oid,
		BlobSHA256: hex.EncodeToString(sum[:]),
		RawBytes:   append([]byte(nil), raw...),
	}, nil
}

// V2OptionalManifestAssertion captures the result of an
// optional disk-bytes assertion against the committed
// manifest authority. The assertion is non-authoritative: a
// mismatch produces a typed diagnostic but never changes the
// C:ManifestPath binding.
type V2OptionalManifestAssertion struct {
	Supplied    bool
	Matches     bool
	Diagnostics V2VerifierDiagnostics
}

// AssertV2OptionalManifestAssertion compares the optional
// caller-supplied bytes against the committed manifest
// authority bytes. When no assertion bytes are supplied the
// result is Supplied=false, Matches=false, Diagnostics=nil
// (the assertion is optional and the verifier must not
// require it).
//
// When supplied, the function returns Supplied=true and:
//
//   - Matches=true when SHA-256(supplied) == SHA-256(authority)
//   - Matches=false when the digests differ, with a typed
//     closure_manifest_assertion_mismatch diagnostic
//
// The function never modifies the authority bytes and never
// derives authority from the optional bytes.
func AssertV2OptionalManifestAssertion(
	authority V2CommittedManifestAuthority,
	optional []byte,
) V2OptionalManifestAssertion {
	if len(optional) == 0 {
		return V2OptionalManifestAssertion{Supplied: false}
	}
	if len(authority.RawBytes) == 0 {
		return V2OptionalManifestAssertion{
			Supplied: true,
			Matches:  false,
			Diagnostics: V2VerifierDiagnostics{NewV2VerifierDiagnostic(
				V2VerifierClosureManifestAssertionMismatch,
				"optional manifest assertion supplied but committed manifest authority is empty",
			).withObjectPath(authority.Path)},
		}
	}
	suppliedSum := sha256.Sum256(optional)
	authoritySum := sha256.Sum256(authority.RawBytes)
	if suppliedSum == authoritySum {
		return V2OptionalManifestAssertion{
			Supplied: true,
			Matches:  true,
		}
	}
	return V2OptionalManifestAssertion{
		Supplied: true,
		Matches:  false,
		Diagnostics: V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierClosureManifestAssertionMismatch,
			"SHA-256 of optional manifest assertion differs from C:M bytes",
		).withObjectPath(authority.Path).
			withExpected(hex.EncodeToString(suppliedSum[:])).
			withObserved(hex.EncodeToString(authoritySum[:]))},
	}
}

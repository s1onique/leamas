// SPDX-License-Identifier: Apache-2.0

package closure

// closure_protocol_v2_dispatch.go replaces V2DispatchTopology
// from closure_protocol_v2.go with a repository-bound dispatch
// that consumes V2TopologyFacts. The legacy boolean function
// remains exported for the version-axis isolation tests but
// every production call site must route through
// DispatchClosureTopology.
//
// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION02-A
// adds the internal executionTopology axis. The dispatch
// accepts three parameters: the public closure protocol version
// (V1, V2), the internal execution topology mode (default,
// freeze_before_subject), and the resolved topology facts. The
// execution topology is held inside an unexported type so
// external schema consumers and external Go callers cannot
// smuggle the runtime-context mode into the request surface.
//
// Splitting this from closure_protocol_v2.go keeps the file
// under the LLM-friendly 400-line threshold while preserving
// the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import "fmt"

// executionTopology is the unexported internal
// topology selector. The two runner entry points are
// the only authority that selects this mode. The
// type is unexported so external Go callers cannot
// smuggle the runtime-context mode into the request
// surface; the V2Request struct has no field of this
// type.
type executionTopology string

const (
	// executionTopologyDefault keeps the legacy V2
	// (S < F) dispatch behaviour.
	executionTopologyDefault executionTopology = ""
	// executionTopologyFreezeBeforeSubject selects the
	// immutable-subject execution direction: F < S.
	executionTopologyFreezeBeforeSubject executionTopology = "freeze_before_subject"
)

// V2DispatchOutcome is the typed result of dispatching
// V2TopologyFacts against a Closure Protocol version. The
// function never infers a version from topology.
type V2DispatchOutcome struct {
	Version    ClosureProtocolVersion
	Accepted   bool
	Code       V2DiagnosticCode
	Message    string
	Relation   V2Relation
	HasSubject bool
	HasFreeze  bool
}

// DispatchClosureTopology applies the topology rule of the
// declared closure protocol version and the internal
// execution topology mode to the supplied facts:
//
//	v1:                            freeze_ancestor_subject && !subject_ancestor_freeze && !equal
//	v2 (default):                  subject_ancestor_freeze && !freeze_ancestor_subject && !equal
//	v2 (freeze_before_subject):    freeze_ancestor_subject && !subject_ancestor_freeze && !equal
//
// The returned outcome is one of:
//
//	accepted (relation = expected)
//	rejected with a typed code (relation = observed)
//
// Equality, missing revisions, reverse ancestry, unrelated
// commits, and git failure all map to distinct codes so the
// runner can emit diagnostics without parsing message text.
func DispatchClosureTopology(version ClosureProtocolVersion, facts V2TopologyFacts) V2DispatchOutcome {
	return dispatchClosureTopology(version, executionTopologyDefault, facts)
}

// dispatchClosureTopology is the unexported sealed dispatch.
// The two public entry points select the topology via this
// helper. External Go callers cannot reach it through the
// exported DispatchClosureTopology.
func dispatchClosureTopology(version ClosureProtocolVersion, execTopology executionTopology, facts V2TopologyFacts) V2DispatchOutcome {
	if !version.IsSupported() {
		return V2DispatchOutcome{
			Version:  version,
			Accepted: false,
			Code:     V2CodeUnsupportedClosureProtocolVersion,
			Message:  fmt.Sprintf("closure protocol version %q is not supported", string(version)),
		}
	}
	relation := facts.Classify()
	switch relation {
	case V2RelationMissingSubject:
		return V2DispatchOutcome{
			Version:  version,
			Accepted: false,
			Code:     V2CodeSubjectCommitNotFound,
			Relation: relation,
			Message:  "subject commit did not resolve in repository",
		}
	case V2RelationMissingFreeze:
		return V2DispatchOutcome{
			Version:  version,
			Accepted: false,
			Code:     V2CodeFreezeCommitNotFound,
			Relation: relation,
			Message:  "freeze commit did not resolve in repository",
		}
	case V2RelationEqual:
		return V2DispatchOutcome{
			Version:  version,
			Accepted: false,
			Code:     V2CodeSubjectEqualsFreeze,
			Relation: relation,
			Message:  "subject and freeze resolve to the same commit",
		}
	}
	// V2 with the freeze_before_subject execution topology
	// is the immutable-subject direction: F < S.
	if version == ClosureProtocolV2 && execTopology == executionTopologyFreezeBeforeSubject {
		if relation == V2RelationFreezeBeforeSubject {
			return V2DispatchOutcome{
				Version:    version,
				Accepted:   true,
				Relation:   relation,
				HasSubject: true,
				HasFreeze:  true,
			}
		}
		if relation == V2RelationSubjectBeforeFreeze {
			return V2DispatchOutcome{
				Version:  version,
				Accepted: false,
				Code:     V2CodeSubjectNotAncestorOfFreeze,
				Relation: relation,
				Message:  "v2 (freeze_before_subject) requires freeze as ancestor of subject; observed the reverse",
			}
		}
		if relation == V2RelationSubjectFreezeUnrelated {
			return V2DispatchOutcome{
				Version:  version,
				Accepted: false,
				Code:     V2CodeSubjectFreezeUnrelated,
				Relation: relation,
				Message:  "v2 (freeze_before_subject) requires freeze as ancestor of subject; observed unrelated commits",
			}
		}
	}
	switch version {
	case ClosureProtocolV1:
		if relation == V2RelationFreezeBeforeSubject {
			return V2DispatchOutcome{
				Version:    version,
				Accepted:   true,
				Relation:   relation,
				HasSubject: true,
				HasFreeze:  true,
			}
		}
		if relation == V2RelationSubjectBeforeFreeze {
			return V2DispatchOutcome{
				Version:  version,
				Accepted: false,
				Code:     V2CodeSubjectNotAncestorOfFreeze,
				Relation: relation,
				Message:  "v1 requires freeze as ancestor of subject; observed the reverse",
			}
		}
		if relation == V2RelationSubjectFreezeUnrelated {
			return V2DispatchOutcome{
				Version:  version,
				Accepted: false,
				Code:     V2CodeSubjectFreezeUnrelated,
				Relation: relation,
				Message:  "v1 requires freeze as ancestor of subject; observed unrelated commits",
			}
		}
	case ClosureProtocolV2:
		if relation == V2RelationSubjectBeforeFreeze {
			return V2DispatchOutcome{
				Version:    version,
				Accepted:   true,
				Relation:   relation,
				HasSubject: true,
				HasFreeze:  true,
			}
		}
		if relation == V2RelationFreezeBeforeSubject {
			return V2DispatchOutcome{
				Version:  version,
				Accepted: false,
				Code:     V2CodeFreezeAncestorOfSubject,
				Relation: relation,
				Message:  "v2 requires subject as ancestor of freeze; observed the reverse",
			}
		}
		if relation == V2RelationSubjectFreezeUnrelated {
			return V2DispatchOutcome{
				Version:  version,
				Accepted: false,
				Code:     V2CodeSubjectFreezeUnrelated,
				Relation: relation,
				Message:  "v2 requires subject as ancestor of freeze; observed unrelated commits",
			}
		}
	}
	return V2DispatchOutcome{
		Version:  version,
		Accepted: false,
		Code:     V2CodeGitOperationFailed,
		Relation: relation,
		Message:  fmt.Sprintf("unhandled topology relation %q", string(relation)),
	}
}

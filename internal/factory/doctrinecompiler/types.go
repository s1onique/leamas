// Package doctrinecompiler projects a versioned Factory doctrine pack
// into a bounded, deterministic tree inside a target repository.
//
// The compiler is local-first, repo-scoped, and fail-closed. It owns no
// network or registry behaviour and never reaches outside the target
// root. Canonical doctrine definitions live inside Leamas; target
// repositories select and extend but do not redefine them.
package doctrinecompiler

import (
	"fmt"
	"sort"
	"strings"
)

// PackId is a typed pack identifier.
type PackId string

// PackVersion is a semantic version string ("MAJOR.MINOR.PATCH").
type PackVersion string

// PackSchemaVersion identifies the schema dialect of a pack blob.
type PackSchemaVersion int

// ProfileId is a typed profile identifier.
type ProfileId string

// DoctrineId is a typed doctrine identifier.
type DoctrineId string

// TargetPath is a normalized, repo-relative POSIX path.
type TargetPath string

// ContentDigest is a hex-encoded SHA-256 digest.
type ContentDigest string

// SupportedPackSchemaVersion is the only schema version accepted by this
// compiler. Newer schemas must be rejected explicitly.
const SupportedPackSchemaVersion PackSchemaVersion = 1

// LockSchemaVersion is the only lock-file schema version emitted.
const LockSchemaVersion int = 1

// OwnershipMode is a closed enumeration. Use the constructors
// OwnershipManaged, OwnershipSeeded, or OwnershipObserved to construct.
type OwnershipMode int

const (
	OwnershipInvalid OwnershipMode = iota
	OwnershipManaged
	OwnershipSeeded
	OwnershipObserved
)

// String returns the canonical lower-case name.
func (o OwnershipMode) String() string {
	switch o {
	case OwnershipManaged:
		return "managed"
	case OwnershipSeeded:
		return "seeded"
	case OwnershipObserved:
		return "observed"
	default:
		return "invalid"
	}
}

// ParseOwnership decodes a serialized ownership label.
func ParseOwnership(s string) (OwnershipMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "managed":
		return OwnershipManaged, nil
	case "seeded":
		return OwnershipSeeded, nil
	case "observed":
		return OwnershipObserved, nil
	default:
		return OwnershipInvalid, fmt.Errorf("unknown ownership mode %q", s)
	}
}

// ActionClass enumerates the closed set of projection action classifications.
// The numeric values are not persisted; only String() values are stable.
type ActionClass int

const (
	ActionInvalid ActionClass = iota
	ActionCreateManaged
	ActionUpdateManaged
	ActionCreateSeeded
	ActionUnchanged
	ActionPreserveSeeded
	ActionRemoveObsoleteManaged
	ActionReject
)

// String returns the canonical classification label.
func (a ActionClass) String() string {
	switch a {
	case ActionCreateManaged:
		return "create-managed"
	case ActionUpdateManaged:
		return "update-managed"
	case ActionCreateSeeded:
		return "create-seeded"
	case ActionUnchanged:
		return "unchanged"
	case ActionPreserveSeeded:
		return "preserve-seeded"
	case ActionRemoveObsoleteManaged:
		return "remove-obsolete-managed"
	case ActionReject:
		return "reject"
	default:
		return "invalid"
	}
}

// ParseAction decodes a serialized action label.
func ParseAction(s string) (ActionClass, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "create-managed":
		return ActionCreateManaged, nil
	case "update-managed":
		return ActionUpdateManaged, nil
	case "create-seeded":
		return ActionCreateSeeded, nil
	case "unchanged":
		return ActionUnchanged, nil
	case "preserve-seeded":
		return ActionPreserveSeeded, nil
	case "remove-obsolete-managed":
		return ActionRemoveObsoleteManaged, nil
	case "reject":
		return ActionReject, nil
	default:
		return ActionInvalid, fmt.Errorf("unknown action %q", s)
	}
}

// ProjectionEntry is a single file in the desired projection.
//
// Content is the canonical bytes that will be written. The digest is
// recomputed from Content by the planner and so is not trusted from
// callers. For seeded files Content is nil at decode time and is
// resolved from the pack at compile time.
type ProjectionEntry struct {
	Path      TargetPath
	Ownership OwnershipMode
	Content   []byte
	Digest    ContentDigest
	Origin    string // declaration id, for diagnostics
}

// ProjectionPlan is the deterministic ordered list of desired actions.
type ProjectionPlan struct {
	PackId          PackId
	PackVersion     PackVersion
	ProfileId       ProfileId
	CompilerVersion string
	PackDigest      ContentDigest
	Actions         []ProjectionAction
}

// ProjectionAction describes what the compiler intends to do to one path.
//
// Reject entries are produced by safety checks and are always terminal:
// they abort the plan.
type ProjectionAction struct {
	Path      TargetPath
	Ownership OwnershipMode
	Class     ActionClass
	Origin    string
	Reason    string
}

// SortedActions returns a deterministic ordering of actions.
func SortedActions(actions []ProjectionAction) []ProjectionAction {
	out := make([]ProjectionAction, len(actions))
	copy(out, actions)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	return out
}

// ObservedContract describes a runtime invariant the verifier will inspect.
// Observed contracts are not written; they are asserted against existing
// target files.
type ObservedContract struct {
	Id   string
	Kind ObservedContractKind
	Path TargetPath
	// Expected contains a small structured payload used by the verifier.
	// For "makefile-include" the payload is a single string: the include path.
	// For "makefile-target-dep" the payload is a {target, dependency} pair
	// encoded via MakeTargetDep.
	Target string
	Dep    string
}

// ObservedContractKind enumerates the supported observed contracts.
type ObservedContractKind string

const (
	ObservedMakefileInclude   ObservedContractKind = "makefile-include"
	ObservedMakefileTargetDep ObservedContractKind = "makefile-target-dep"
	ObservedFileExists        ObservedContractKind = "file-exists"
	ObservedFileDigestEquals  ObservedContractKind = "file-digest-equals"
)

// FactorizeCheck declares one entry of the factorize chain. The chain is
// rendered into the generated factory.mk.
type FactorizeCheck struct {
	Id      string
	Command []string
}

// ExtensionPoint declares a named hook the target repository owns.
type ExtensionPoint struct {
	Id          string
	Kind        string
	Name        string
	Description string
}

// CompileError is a typed failure from any phase of the compiler.
type CompileError struct {
	Phase   string // "decode", "validate", "plan", "compile", "verify"
	Subject string // path, doctrine id, profile id, etc.
	Reason  string
}

// Error implements the error interface.
func (e *CompileError) Error() string {
	return fmt.Sprintf("%s error: %s: %s", e.Phase, e.Subject, e.Reason)
}

// newError constructs a CompileError.
func newError(phase, subject, reason string) *CompileError {
	return &CompileError{Phase: phase, Subject: subject, Reason: reason}
}

// isPathSegmentSafe reports whether a single path segment is safe.
// It rejects empty, ".", "..", and any segment containing a path
// separator or NUL byte.
func isPathSegmentSafe(seg string) bool {
	if seg == "" || seg == "." || seg == ".." {
		return false
	}
	if strings.ContainsAny(seg, "/\\\x00") {
		return false
	}
	return true
}

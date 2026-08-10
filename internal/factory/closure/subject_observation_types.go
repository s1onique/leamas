// SPDX-License-Identifier: Apache-2.0

package closure

// subject_observation_types.go defines the typed observation
// results required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-SUBJECT-OBSERVATION-AUTHORITY01
// (R6-A). The production GitV2SubjectExecutor is the single
// authority for every fact that can only be observed while the
// detached subject worktree exists, so the result carries:
//
//   - identity facts (SubjectWorktreePath, SubjectHead,
//     SubjectTree, SubjectDetached)
//   - typed byte observations (status, refs) that distinguish
//     "unavailable" from "empty bytes"
//   - three worktree inventory snapshots (Before, AtSubject,
//     After) plus the canonical registration that binds
//     (SubjectWorktreePath, SubjectHead)
//   - transported topology facts (V2TopologyFacts) so the
//     producer can carry the facts the runtime-context
//     resolver already established
//   - cleanup observation (Observed, Error)
//
// Phase 2 requires that every byte observation distinguish
// unavailable (Available=false) from empty bytes. The
// SubjectByteObservation struct is the canonical carrier.
//
// Phase 16 requires canonical equality on (Path, HEAD OID)
// pairs. SubjectWorktreeInventory.Equal is the canonical
// comparator; it ignores order so a reordered set compares
// equal but a Path/HEAD swap does not.
//
// Splitting this from closure_protocol_v2_executor.go keeps the
// executor under the LLM-friendly 400-line threshold while
// preserving the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

// SubjectByteObservation is the canonical carrier for an
// observation whose result is raw bytes. The contract is:
//
//	Available=false  -> observation was not attempted, or the
//	                    underlying Git command failed. Error
//	                    carries a typed diagnostic.
//	Available=true   -> observation succeeded. Bytes is the
//	                    exact observed payload, possibly empty.
//	                    Empty bytes are a legitimate result
//	                    (e.g. clean porcelain status) and MUST
//	                    NOT be encoded as "unavailable".
type SubjectByteObservation struct {
	Available bool
	Bytes     string
	// Error carries the typed diagnostic message when
	// Available is false. Empty when Available is true.
	Error string
}

// SubjectLiveIdentity is the canonical carrier for facts
// observed via `git rev-parse` and `git symbolic-ref` against
// the live subject worktree. The struct is populated only
// after the worktree has been created and the S worktree
// path is known. Every field is fail-closed: an observation
// failure leaves the corresponding field at its zero value
// and Diagnostics carries the typed reason.
//
// SubjectDetached is derived from `git symbolic-ref -q HEAD`:
// a non-zero exit means the HEAD is detached.
type SubjectLiveIdentity struct {
	WorktreePath string
	Head         string
	Tree         string
	Toplevel     string
	Detached     bool
	Diagnostics  V2Diagnostics
	Available    bool
}

// SubjectWorktreeRegistration is one entry from a
// `git worktree list --porcelain -z` snapshot. The pair
// (Path, Head) is the canonical identity required by Phase 8
// and Phase 16. Path is the canonical absolute path of the
// linked worktree; Head is the resolved HEAD commit OID.
type SubjectWorktreeRegistration struct {
	Path string
	Head string
}

// SubjectWorktreeInventory is the canonical result of a
// `git worktree list --porcelain -z` observation. The struct
// is fail-closed: an observation failure leaves Available
// false and Diagnostics carries the typed reason. On success,
// Registrations preserves the canonical order returned by
// the parser.
//
// Equal compares two inventories by (Path, Head) pair
// equality, ignoring order. Two inventories are equal iff
// they contain the same set of (Path, Head) pairs.
type SubjectWorktreeInventory struct {
	Available     bool
	Registrations []SubjectWorktreeRegistration
	Diagnostics   V2Diagnostics
}

// Equal reports whether two inventories contain the same set
// of (Path, Head) pairs, regardless of order. Equal returns
// true only when both inventories are available; an
// unavailable inventory is never equal to any inventory
// (including another unavailable one) so production code
// cannot accidentally equate an observation failure with a
// missing entry.
//
// Phase 16 requires:
//
//	same set / different order -> equal
//	same path / different HEAD -> not equal
//	different path / same HEAD -> not equal
func (inv SubjectWorktreeInventory) Equal(other SubjectWorktreeInventory) bool {
	if !inv.Available || !other.Available {
		return false
	}
	if len(inv.Registrations) != len(other.Registrations) {
		return false
	}
	seen := make(map[SubjectWorktreeRegistration]int, len(inv.Registrations))
	for _, r := range inv.Registrations {
		seen[r]++
	}
	for _, r := range other.Registrations {
		count, ok := seen[r]
		if !ok || count == 0 {
			return false
		}
		seen[r] = count - 1
	}
	return true
}

// FindByPath returns the registration whose Path equals the
// supplied canonical path. The boolean reports whether a
// matching registration was found. Equal returns ok=false
// when the inventory is unavailable.
func (inv SubjectWorktreeInventory) FindByPath(path string) (SubjectWorktreeRegistration, bool) {
	if !inv.Available {
		return SubjectWorktreeRegistration{}, false
	}
	for _, r := range inv.Registrations {
		if r.Path == path {
			return r, true
		}
	}
	return SubjectWorktreeRegistration{}, false
}

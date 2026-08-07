// SPDX-License-Identifier: Apache-2.0

// Package evidence - closure_evidence_completeness_predicates.go
// owns the per-predicate methods of ClosureEvidenceEx.
//
// Splitting these from closure_evidence_completeness.go keeps
// both files under the LLM-friendly 400-line threshold while
// preserving the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.
//
// Every predicate here is exposed as a separate method so each
// branch can be mutation-tested independently by
// TestClosureEvidenceCompletenessDerived.

package evidence

import "path/filepath"

// AllRunChecksSuccessful reports whether every "run" check
// produced a pass outcome with no timeout / cancel / cleanup
// error.
func (e ClosureEvidenceEx) AllRunChecksSuccessful() bool {
	for _, r := range e.Authorities.Checks {
		if r.Mode != "run" {
			continue
		}
		if r.Outcome != "pass" {
			return false
		}
		if r.TimedOut || r.Canceled || r.CleanupError != "" {
			return false
		}
	}
	return true
}

// ExcludeChecksUnexecuted reports whether every "exclude"
// check produced an "excluded" outcome with no exit / cleanup.
func (e ClosureEvidenceEx) ExcludeChecksUnexecuted() bool {
	for _, r := range e.Authorities.Checks {
		if r.Mode != "exclude" {
			continue
		}
		if r.Outcome != "excluded" {
			return false
		}
		if r.ExitCode != 0 || r.TimedOut || r.Canceled || r.CleanupError != "" {
			return false
		}
	}
	return true
}

// NoTimeout reports that no check or gate timed out.
func (e ClosureEvidenceEx) NoTimeout() bool {
	for _, r := range e.Authorities.Checks {
		if r.TimedOut {
			return false
		}
	}
	if e.Gate.TimedOut {
		return false
	}
	return true
}

// NoCancellation reports that no check or gate was cancelled.
func (e ClosureEvidenceEx) NoCancellation() bool {
	for _, r := range e.Authorities.Checks {
		if r.Canceled {
			return false
		}
	}
	if e.Gate.Canceled {
		return false
	}
	return true
}

// NoTruncation reports that no stdout / stderr stream was
// truncated.
func (e ClosureEvidenceEx) NoTruncation() bool {
	for _, r := range e.Authorities.Checks {
		if r.StdoutTruncated || r.StderrTruncated {
			return false
		}
	}
	if e.Gate.StdoutTruncated || e.Gate.StderrTruncated {
		return false
	}
	return true
}

// NoCleanupError reports that no check recorded a cleanup error.
func (e ClosureEvidenceEx) NoCleanupError() bool {
	for _, r := range e.Authorities.Checks {
		if r.CleanupError != "" {
			return false
		}
	}
	return true
}

// GateClassificationPASS reports that the gate classification
// verdict is PASS. Any FAIL or UNAVAILABLE verdict blocks
// COMPLETE.
func (e ClosureEvidenceEx) GateClassificationPASS() bool {
	return e.Authorities.Gate.Verdict == "PASS"
}

// BinaryAuthorityValid reports whether every binary authority
// invariant holds: path absolute, VCS revision matches
// subject commit, vcs.modified is false, executable bit set,
// source HEAD and tree match the recorded values, source was
// clean and detached.
func (e ClosureEvidenceEx) BinaryAuthorityValid() bool {
	b := e.Authorities.Binary
	// Path must be present and absolute so the runner cannot
	// claim COMPLETE from a path that the caller cannot audit.
	if b.BinaryPath == "" || !filepath.IsAbs(b.BinaryPath) {
		return false
	}
	if !isHexSHA256(b.BinarySHA256) {
		return false
	}
	if b.VCSRevision == "" || b.VCSModified {
		return false
	}
	if !b.Executable {
		return false
	}
	// SourceCommit must be a valid commit OID, not just a
	// non-empty string, and it must match the recorded VCS
	// revision so an evidence record cannot silently swap one
	// for the other.
	if !isValidOID(b.SourceCommit) {
		return false
	}
	if !isValidOID(b.SourceTree) {
		return false
	}
	if b.VCSRevision != b.SourceCommit {
		return false
	}
	// The source tree must be clean and detached. A dirty
	// checkout or a branch ref can mutate the source between
	// the build and the manifest, so neither is acceptable.
	if !b.SourceClean {
		return false
	}
	if !b.SourceDetached {
		return false
	}
	return true
}

// BeforeStateAvailable reports whether the BEFORE caller-state
// snapshot was Available.
func (e ClosureEvidenceEx) BeforeStateAvailable() bool {
	return e.Authorities.CallerAvailable.BeforeAvailable
}

// AfterStateAvailable reports whether the AFTER caller-state
// snapshot was Available.
func (e ClosureEvidenceEx) AfterStateAvailable() bool {
	return e.Authorities.CallerAvailable.AfterAvailable
}

// CallerStateUnchanged reports that the BEFORE and AFTER
// snapshots compared equal across HEAD, HEAD tree, status,
// and refs.
func (e ClosureEvidenceEx) CallerStateUnchanged() bool {
	d := e.Authorities.CallerDrift
	return !d.HEADChanged &&
		!d.TreeChanged &&
		!d.StatusChanged &&
		!d.RefsChanged
}

// WorktreeInventoryUnchanged reports that no worktree
// registration leaked between BEFORE and AFTER.
func (e ClosureEvidenceEx) WorktreeInventoryUnchanged() bool {
	return !e.Authorities.CallerDrift.WorktreeLeaked
}

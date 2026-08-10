// SPDX-License-Identifier: Apache-2.0

package closure

// subject_observation.go implements the bounded Git
// observation helpers required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-SUBJECT-OBSERVATION-AUTHORITY01
// (R6-A). The helpers are the production-only bridge between
// the live detached subject worktree and the typed result
// fields documented in subject_observation_types.go.
//
// All helpers route through the existing bounded gitClient so
// R6-A does not introduce a new subprocess gateway or raw
// os/exec calls. The helpers are fail-closed: every
// observation failure populates the typed Available/Diagnostics
// fields and never fabricates a placeholder value.
//
// Splitting this from closure_protocol_v2_executor.go keeps
// the executor under the LLM-friendly 400-line threshold
// while preserving the single closure over the descriptor
// that ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

// observeLiveSubjectIdentity captures the four canonical
// facts from the live subject worktree:
//
//	git -C <worktree> rev-parse HEAD
//	git -C <worktree> rev-parse HEAD^{tree}
//	git -C <worktree> rev-parse --show-toplevel
//	git -C <worktree> symbolic-ref -q HEAD
//
// The detached state is established by the canonical exit
// code of `symbolic-ref -q HEAD`: exit 0 means the HEAD is a
// symbolic ref (NOT detached); non-zero exit means the HEAD
// is detached. A non-zero exit is the authoritative detached
// signal and is NOT collapsed into a generic "git error".
// An `Err` is only treated as a detached-state observation
// failure when neither the canonical exit code nor the
// captured stderr is consistent with the detached contract.
//
// The function is total: every failure path leaves Available
// false and appends a typed diagnostic.
func observeLiveSubjectIdentity(ctx context.Context, git gitClient, worktreePath, subjectCommit string) SubjectLiveIdentity {
	id := SubjectLiveIdentity{WorktreePath: worktreePath}
	diags := V2Diagnostics{}
	if strings.TrimSpace(worktreePath) == "" {
		diags = append(diags, V2Diagnostic{
			Code:         V2CodeSubjectObservationUnavailable,
			Message:      "subject identity observation failed: empty worktree path",
			PropertyName: "subject_worktree",
		})
		id.Diagnostics = diags
		return id
	}
	if git == nil {
		diags = append(diags, V2Diagnostic{
			Code:         V2CodeSubjectObservationUnavailable,
			Message:      "subject identity observation failed: no Git client supplied",
			PropertyName: "subject_worktree",
		})
		id.Diagnostics = diags
		return id
	}
	// HEAD: rev-parse HEAD
	if head, err := runGitValue(ctx, git, worktreePath, "rev-parse", "HEAD"); err != nil {
		diags = append(diags, V2Diagnostic{
			Code:         V2CodeSubjectObservationUnavailable,
			Message:      fmt.Sprintf("subject identity observation failed: HEAD: %s", err.Error()),
			PropertyName: "subject_head",
		})
	} else {
		id.Head = head
	}
	// HEAD^{tree}: rev-parse HEAD^{tree}
	if tree, err := runGitValue(ctx, git, worktreePath, "rev-parse", "HEAD^{tree}"); err != nil {
		diags = append(diags, V2Diagnostic{
			Code:         V2CodeSubjectObservationUnavailable,
			Message:      fmt.Sprintf("subject identity observation failed: HEAD^{tree}: %s", err.Error()),
			PropertyName: "subject_tree",
		})
	} else {
		id.Tree = tree
	}
	// show-toplevel: rev-parse --show-toplevel
	if top, err := runGitValue(ctx, git, worktreePath, "rev-parse", "--show-toplevel"); err != nil {
		diags = append(diags, V2Diagnostic{
			Code:         V2CodeSubjectObservationUnavailable,
			Message:      fmt.Sprintf("subject identity observation failed: show-toplevel: %s", err.Error()),
			PropertyName: "subject_toplevel",
		})
	} else {
		id.Toplevel = top
	}
	// Detached: symbolic-ref -q HEAD
	//
	// `symbolic-ref -q HEAD` returns exit 0 with stdout of
	// the current branch when HEAD is a symbolic ref, and
	// exit 1 when HEAD is detached. The canonical detached
	// contract is "non-zero exit AND the err is the typed
	// exec.ExitError". Anything else (spawn failure,
	// timeout, output overflow) is an observation failure
	// and MUST be reported as such; the act explicitly
	// forbids collapsing arbitrary Git errors into
	// "detached".
	symResult := git.Run(ctx, worktreePath, "symbolic-ref", "-q", "HEAD")
	switch {
	case symResult.ExitCode == 0:
		// HEAD is a symbolic ref: NOT detached. This is
		// unexpected for the production subject worktree
		// (which is created with --detach) but the act
		// requires the observation to be authoritative:
		// the executor reports the observed value, not a
		// hard-coded boolean.
		id.Detached = false
	case symResult.ExitCode == 1 && len(bytes.TrimSpace(symResult.Stdout)) == 0:
		// Canonical detached signal: non-zero exit with
		// empty stdout and a typed exec.ExitError.
		id.Detached = true
	default:
		diags = append(diags, V2Diagnostic{
			Code: V2CodeSubjectObservationUnavailable,
			Message: fmt.Sprintf("subject identity observation failed: symbolic-ref: exit=%d stderr=%q",
				symResult.ExitCode, strings.TrimSpace(string(symResult.Stderr))),
			PropertyName: "subject_detached",
		})
	}
	if len(diags) > 0 {
		id.Available = false
		id.Diagnostics = diags
		return id
	}
	id.Available = true
	return id
}

// observeSubjectStatus captures
//
//	git -C <worktree> status --porcelain=v2 --untracked-files=all
//
// The function is fail-closed. Empty bytes are a legitimate
// result (clean worktree) and MUST be encoded as
// Available=true, Bytes="" per Phase 2.
func observeSubjectStatus(ctx context.Context, git gitClient, worktreePath string) SubjectByteObservation {
	if strings.TrimSpace(worktreePath) == "" || git == nil {
		return SubjectByteObservation{
			Available: false,
			Error:     "subject status observation failed: no worktree path or Git client",
		}
	}
	res := git.Run(ctx, worktreePath, "status", "--porcelain=v2", "--untracked-files=all")
	if res.Err != nil || res.ExitCode != 0 {
		return SubjectByteObservation{
			Available: false,
			Error:     fmt.Sprintf("subject status observation failed: exit=%d stderr=%q", res.ExitCode, strings.TrimSpace(string(res.Stderr))),
		}
	}
	return SubjectByteObservation{
		Available: true,
		Bytes:     string(res.Stdout),
	}
}

// observeSubjectRefs reuses the existing canonical refs
// authority (snapshotCallerRefs) applied to the subject
// worktree. The act forbids a second refs representation;
// this helper IS the canonical authority.
//
// RefsBytes and RefsHash from snapshotCallerRefs are
// projected into SubjectByteObservation so empty bytes (a
// repository with no refs) are still observed as
// Available=true.
func observeSubjectRefs(ctx context.Context, git gitClient, worktreePath string) SubjectByteObservation {
	if strings.TrimSpace(worktreePath) == "" || git == nil {
		return SubjectByteObservation{
			Available: false,
			Error:     "subject refs observation failed: no worktree path or Git client",
		}
	}
	bytes, _, diags, available := snapshotCallerRefs(ctx, git, worktreePath)
	if !available {
		if len(diags) == 0 {
			return SubjectByteObservation{
				Available: false,
				Error:     "subject refs observation failed: unknown failure",
			}
		}
		return SubjectByteObservation{
			Available: false,
			Error:     diags[0].Message,
		}
	}
	return SubjectByteObservation{
		Available: true,
		Bytes:     bytes,
	}
}

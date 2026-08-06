// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_state.go implements Phase 5 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-CLI-TAG-STATE01:
//
// the optional read-only caller-state capture used to prove
// that the verifier does not mutate the target repository
// during verification.
//
// The capture snapshots four orthogonal facets of caller
// state, all under the bound repository root:
//
//   - HEAD commit OID (rev-parse --verify HEAD^{commit})
//   - HEAD tree OID  (rev-parse --verify HEAD^{tree})
//   - porcelain-v2  (status --porcelain=v2 --untracked-files=normal)
//   - worktree list (worktree list --porcelain)
//   - refs snapshot (for-each-ref --format=%(refname)
//                   %(objectname:short=0)
//
// Every observation is purely read-only. The verifier MUST
// NEVER checkout, create worktrees, mutate refs, or write
// inside the target repository. CaptureV2CallerState
// returns a V2CallerStateSnapshot whose pre- and post-run
// hashes must be byte-equal to satisfy ACT 4's read-only
// guarantee.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// V2CallerStateSnapshot captures the read-only caller-state
// projections used to prove the verifier is non-mutating.
// All fields are populated from the bound authority at
// invocation time; the orchestrator captures one snapshot
// before and one after the verification sequence, then
// compares Hash() values for byte equality.
//
// Status is the raw porcelain-v2 output bytes. Worktrees is
// the parsed worktree list output. Refs is a deterministic
// "refname -> oid" mapping sorted by refname.
type V2CallerStateSnapshot struct {
	RepositoryRoot string
	HeadCommit     string
	HeadTree       string
	Status         string
	Worktrees      string
	Refs           string

	Diagnostics V2VerifierDiagnostics
}

// CaptureV2CallerState snapshots the bound repository's
// caller state. The function calls the bound authority for
// every observation and never writes back. Each observation
// failure produces a non-fatal typed diagnostic on the
// returned snapshot; a capture failure does NOT cause a
// verification rejection because the verifier itself
// remains read-only regardless of the capture result.
//
// Capture failures are surfaced through the Diagnostics
// slice and the corresponding field is left empty so the
// diagnostics are the authoritative signal.
func CaptureV2CallerState(
	ctx context.Context,
	authority V2ClosureGitAuthority,
) V2CallerStateSnapshot {
	snap := V2CallerStateSnapshot{}

	rootValue := authorityRunGit(ctx, authority, "rev-parse", "--show-toplevel")
	if rootValue.exitCode == 0 {
		snap.RepositoryRoot = strings.TrimSpace(string(rootValue.stdout))
	}

	headCommit := authorityRunGit(ctx, authority, "rev-parse", "--verify", "--end-of-options", "HEAD^{commit}")
	if headCommit.exitCode == 0 {
		snap.HeadCommit = strings.TrimSpace(string(headCommit.stdout))
	} else {
		snap.Diagnostics = append(snap.Diagnostics, NewV2VerifierDiagnostic(
			V2VerifierStateCaptureHeadFailed,
			"git rev-parse --verify HEAD^{commit} failed during state capture",
		))
	}

	headTree := authorityRunGit(ctx, authority, "rev-parse", "--verify", "--end-of-options", "HEAD^{tree}")
	if headTree.exitCode == 0 {
		snap.HeadTree = strings.TrimSpace(string(headTree.stdout))
	} else {
		snap.Diagnostics = append(snap.Diagnostics, NewV2VerifierDiagnostic(
			V2VerifierStateCaptureHeadFailed,
			"git rev-parse --verify HEAD^{tree} failed during state capture",
		))
	}

	statusValue := authorityRunGit(ctx, authority, "status", "--porcelain=v2", "--untracked-files=normal")
	if statusValue.exitCode == 0 {
		snap.Status = string(statusValue.stdout)
	} else {
		snap.Diagnostics = append(snap.Diagnostics, NewV2VerifierDiagnostic(
			V2VerifierStateCaptureStatusFailed,
			"git status --porcelain=v2 failed during state capture",
		))
	}

	worktreeValue := authorityRunGit(ctx, authority, "worktree", "list", "--porcelain")
	if worktreeValue.exitCode == 0 {
		snap.Worktrees = string(worktreeValue.stdout)
	} else {
		// worktree list is permitted to fail in linked
		// worktrees, but the bound repo is always the
		// bare or primary worktree. A failure here is
		// treated as a non-fatal capture warning.
		snap.Diagnostics = append(snap.Diagnostics, NewV2VerifierDiagnostic(
			V2VerifierStateCaptureWorktreeFailed,
			"git worktree list --porcelain failed during state capture",
		))
	}

	refsValue := authorityRunGit(
		ctx,
		authority,
		"for-each-ref",
		"--format=%(HEAD)%(refname)%00%(objectname)",
	)
	if refsValue.exitCode == 0 {
		snap.Refs = canonicalizeRefsOutput(refsValue.stdout)
	} else {
		snap.Diagnostics = append(snap.Diagnostics, NewV2VerifierDiagnostic(
			V2VerifierStateCaptureRefsFailed,
			"git for-each-ref failed during state capture",
		))
	}
	return snap
}

// canonicalizeRefsOutput normalises the raw for-each-ref
// output into a deterministic "refname -> oid" mapping
// sorted by refname. The same set of refs presented in a
// different discovery order produces identical strings,
// which keeps before/after comparisons stable across
// concurrent runs that race against ref creation.
func canonicalizeRefsOutput(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	lines := strings.Split(string(raw), "\n")
	pairs := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		idx := strings.Index(line, "\x00")
		if idx < 0 {
			pairs = append(pairs, line)
			continue
		}
		ref := line[:idx]
		oid := strings.TrimSpace(line[idx+1:])
		if ref == "" || oid == "" {
			continue
		}
		pairs = append(pairs, ref+"\x00"+oid)
	}
	sort.Strings(pairs)
	return strings.Join(pairs, "\n")
}

// Hash returns a deterministic SHA-256 hex digest over the
// five read-only projections of the caller state. The
// function ignores the Diagnostics slice so a capture
// failure is not obscured by the hash: the caller should
// check Diagnostics separately and treat the hash as the
// authoritative equality signal.
//
// Captures from the same repository at the same logical
// moment produce identical hashes; captures that differ by
// a single byte differ by a full hash.
func (s V2CallerStateSnapshot) Hash() string {
	parts := []string{
		s.RepositoryRoot,
		s.HeadCommit,
		s.HeadTree,
		s.Status,
		s.Worktrees,
		s.Refs,
	}
	h := sha256.Sum256([]byte(strings.Join(parts, "\x00\x00\x00")))
	return hex.EncodeToString(h[:])
}

// CheckReadOnly returns nil when the supplied pre- and
// post-verification snapshots are byte-identical for every
// read-only projection. A single difference produces a
// state_mutation_detected diagnostic. The check is cheap
// (one comparison per field plus one hash comparison for
// the convenience helper) and always surfaces Diagnostics
// rather than panicking on a Nil receiver.
func CheckReadOnly(before, after V2CallerStateSnapshot) V2VerifierDiagnostics {
	var diags V2VerifierDiagnostics
	if before.RepositoryRoot != after.RepositoryRoot {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierStateMutationDetected,
			"repository_root changed during verification",
		).withProperty("repository_root").
			withExpected(before.RepositoryRoot).
			withObserved(after.RepositoryRoot))
	}
	if before.HeadCommit != after.HeadCommit {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierStateMutationDetected,
			"HEAD commit changed during verification",
		).withProperty("head_commit").
			withExpected(before.HeadCommit).
			withObserved(after.HeadCommit))
	}
	if before.HeadTree != after.HeadTree {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierStateMutationDetected,
			"HEAD tree changed during verification",
		).withProperty("head_tree").
			withExpected(before.HeadTree).
			withObserved(after.HeadTree))
	}
	if before.Status != after.Status {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierStateMutationDetected,
			"porcelain-v2 status changed during verification",
		).withProperty("status"))
	}
	if before.Worktrees != after.Worktrees {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierStateMutationDetected,
			"worktree list changed during verification",
		).withProperty("worktrees"))
	}
	if before.Refs != after.Refs {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierStateMutationDetected,
			"refs snapshot changed during verification",
		).withProperty("refs"))
	}
	return diags
}

// SPDX-License-Identifier: Apache-2.0

// Package digest: gate_evidence_adapter.go adapts the
// normalized Gate Summary and the digest's ResolvedMode into
// the binding-relevant input structs consumed by
// EvaluateGateEvidenceBinding. The adapter is the only path
// that translates wire/resolved fields into the pure
// classifier's input contract.
//
// The adapter is pure: it never reads the filesystem, never
// runs Git, and never consults the clock. It is the narrowest
// possible mapping function.
package digest

import "github.com/s1onique/leamas/internal/gatesummary"

// summarizeGateIdentity extracts the binding-relevant fields
// from a normalized gatesummary.Summary. The boolean flags
// HasExecutionIdentity and HasScopeIdentity are populated
// explicitly so the classifier can short-circuit legacy
// shapes without consulting each field.
func summarizeGateIdentity(summary gatesummary.Summary) GateSummaryIdentity {
	out := GateSummaryIdentity{
		SchemaVersion: int(summary.SchemaVersion),
		GeneratedAt:   summary.GeneratedAt,
		OverallStatus: string(summary.Overall.Status),
	}
	if summary.Execution != nil {
		out.ExecutionHeadOID = summary.Execution.HeadOID
		out.ExecutionTreeOID = summary.Execution.TreeOID
		out.SubjectTreeOID = summary.Execution.SubjectOID
		out.HasExecutionIdentity = summary.Execution.HeadOID != ""
	}
	if summary.Scope != nil {
		out.ScopeID = summary.Scope.ID
	}
	if summary.Parent != nil {
		out.ParentAct = summary.Parent.Act
	}
	if summary.Worktree != nil {
		out.WorktreeCleanBefore = summary.Worktree.CleanBefore
		out.WorktreeCleanBeforeSet = true
		out.WorktreeCleanAfter = summary.Worktree.CleanAfter
		out.WorktreeCleanAfterSet = true
	}
	out.HasScopeIdentity = out.ScopeID != "" || out.ParentAct != ""
	return out
}

// digestAuthorityFromResolved converts the digest's
// ResolvedMode into the binding-relevant DigestAuthority. The
// mapping is intentionally conservative:
//
//   - Dirty mode forces Dirty=true and an empty subject.
//   - Clean committed modes use the resolved right endpoint
//     as SubjectCommitOID. The tree OID is recorded when
//     tooling identity already captured it (no extra Git
//     calls are introduced here).
//   - ActID is the resolved lifecycle ActID.
//
// The returned struct is the single source of truth for the
// digest side of the binding comparison.
func digestAuthorityFromResolved(resolved *ResolvedMode) DigestAuthority {
	out := DigestAuthority{}
	if resolved == nil {
		return out
	}
	out.Mode = string(resolved.Mode)
	out.Range = resolved.Range
	out.SubjectCommitOID = resolved.HeadCommit
	out.ActID = resolved.ActID
	if resolved.Mode == ModeDirty {
		out.Dirty = true
		out.SubjectCommitOID = ""
		out.SubjectTreeOID = ""
	}
	// Tree OID is only meaningful when the resolved tool
	// identity already captured it. The adapter deliberately
	// does not invoke OID lookup machinery here — that would
	// add Git subprocesses and is the responsibility of the
	// resolver callers.
	if resolved.ToolIdentity.RepositoryTree != "" && !out.Dirty {
		out.SubjectTreeOID = resolved.ToolIdentity.RepositoryTree
	}
	return out
}

// classifyGateEvidence is the shared adapter entry point. It
// produces the typed GateEvidenceBinding record used by the
// renderer. The function is total: every combination of
// inputs (including nil resolved) returns a valid record.
//
// sourceValidity is the gate-summary probe result. The
// classifier distinguishes the three validity states
// ABSENT, INVALID, and VALID. The ACT explicitly requires
// ABSENT and INVALID to remain distinct.
func classifyGateEvidence(summary gatesummary.Summary, sourceValidity GateSourceValidity, resolved *ResolvedMode) GateEvidenceBinding {
	gate := summarizeGateIdentity(summary)
	digest := digestAuthorityFromResolved(resolved)
	return EvaluateGateEvidenceBinding(gate, digest, sourceValidity)
}

// bindingFieldKV renders the binding block as a sequence of
// key=value lines terminated by a trailing newline. The
// block is rendered BEFORE the historical verdict
// (overall_status) so that authoritative qualification is
// adjacent to (and above) the verdict it qualifies. The
// rendering is deterministic and stable.
func bindingFieldKV(b GateEvidenceBinding, gate GateSummaryIdentity) string {
	// Direct string concatenation avoids fmt overhead for
	// the hot path.
	var out string
	out += "binding_status=" + string(b.Status) + "\n"
	out += "source_validity=" + b.SourceValidity.String() + "\n"
	out += "authoritative_for_digest=" + boolStr(b.AuthoritativeForDigest) + "\n"
	out += "state_binding=" + string(b.StateBinding) + "\n"
	out += "scope_binding=" + string(b.ScopeBinding) + "\n"
	if b.TreeBinding != StateUnbound {
		out += "tree_binding=" + string(b.TreeBinding) + "\n"
	}
	out += "warning_code=" + b.WarningCode + "\n"
	// Canonical OID fields: included only when the gate
	// summary actually exposes them. This keeps legacy
	// summaries visually compact while still revealing the
	// raw provenance for new summaries.
	if gate.ExecutionHeadOID != "" {
		out += "gate_execution_head_oid=" + sanitizeLine(gate.ExecutionHeadOID) + "\n"
	}
	if gate.ScopeID != "" {
		out += "gate_scope_id=" + sanitizeLine(gate.ScopeID) + "\n"
	}
	if gate.ParentAct != "" {
		out += "gate_parent_act=" + sanitizeLine(gate.ParentAct) + "\n"
	}
	return out
}

// boolStr converts a Go bool to its canonical digest wire
// form. The form is "true" / "false" with no capitalization.
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

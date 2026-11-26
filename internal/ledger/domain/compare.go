package domain

// This file holds the content-equality helpers of the canonical records.
//
// They are the ORTHOGONAL comparison used by the mutation no-op detection
// and by the optimistic revision guard: a record whose semantic content is
// unchanged is not a change, whatever its revision says.

// SameWorkItemContent reports whether two work items carry the same
// semantic content.
//
// The revision is deliberately excluded: a revision bump is a bookkeeping
// event, not a content change. It is the comparison that decides whether a
// mutation is a semantic no-op, which must not rewrite canonical state.
func SameWorkItemContent(a, b WorkItem) bool {
	if a.Title != b.Title || a.Summary != b.Summary || a.Kind != b.Kind ||
		a.Lifecycle != b.Lifecycle || a.Acceptance != b.Acceptance ||
		a.Priority != b.Priority || a.Quality != b.Quality ||
		a.Owner != b.Owner || a.Archived != b.Archived ||
		a.RequiresLiveGate != b.RequiresLiveGate ||
		a.TerminalDisposition != b.TerminalDisposition ||
		a.AcceptedBy != b.AcceptedBy ||
		a.AcceptanceBasis != b.AcceptanceBasis ||
		a.AcceptanceSourceHead != b.AcceptanceSourceHead ||
		a.SourceStatus != b.SourceStatus ||
		a.SourceRow != b.SourceRow ||
		a.SourceLine != b.SourceLine ||
		a.SourceDetail != b.SourceDetail ||
		a.EpicID != b.EpicID || a.ItemID != b.ItemID {
		return false
	}
	if !equalStringSlices(a.AcceptanceCriteria, b.AcceptanceCriteria) {
		return false
	}
	if !equalStringSlices(a.ExitCriteria, b.ExitCriteria) {
		return false
	}
	if len(a.Dependencies) != len(b.Dependencies) {
		return false
	}
	for i := range a.Dependencies {
		if a.Dependencies[i] != b.Dependencies[i] {
			return false
		}
	}
	if len(a.Evidence) != len(b.Evidence) {
		return false
	}
	for i := range a.Evidence {
		if a.Evidence[i] != b.Evidence[i] {
			return false
		}
	}
	if len(a.Successors) != len(b.Successors) {
		return false
	}
	for i := range a.Successors {
		if a.Successors[i] != b.Successors[i] {
			return false
		}
	}
	if len(a.Reopen) != len(b.Reopen) {
		return false
	}
	for i := range a.Reopen {
		if a.Reopen[i] != b.Reopen[i] {
			return false
		}
	}
	return true
}

// equalStringSlices reports whether two string slices are identical.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// SameEvidenceContent reports whether two evidence records carry the same
// semantic content. The revision is excluded: it is bookkeeping, not
// content.
func SameEvidenceContent(a, b EvidenceRefRecord) bool {
	if a.EvidenceID != b.EvidenceID || a.Kind != b.Kind || a.Title != b.Title ||
		a.RelativePath != b.RelativePath || a.SHA256 != b.SHA256 ||
		a.ItemID != b.ItemID || a.Quality != b.Quality || a.Archived != b.Archived ||
		a.Class != b.Class || a.Criterion != b.Criterion ||
		a.ByteSize != b.ByteSize || a.SourceHead != b.SourceHead ||
		a.Verified != b.Verified || a.VerifiedAt != b.VerifiedAt {
		return false
	}
	if (a.Provenance == nil) != (b.Provenance == nil) {
		return false
	}
	if a.Provenance != nil && *a.Provenance != *b.Provenance {
		return false
	}
	return true
}

// SameActContent reports whether two ACT execution records carry the same
// semantic content. The revision is excluded: it is bookkeeping, not
// content.
//
// EVERY work-item identity the ACT addresses participates in the
// comparison. An ACT that addresses a different set of work items is a
// different record, even when every other field is identical: comparing
// only the primary ItemID would let two materially different ACT records
// compare as identical.
func SameActContent(a, b ActExecution) bool {
	if a.ActID != b.ActID || a.ItemID != b.ItemID || a.Title != b.Title ||
		a.Verdict != b.Verdict || a.Archived != b.Archived ||
		a.TerminalDisposition != b.TerminalDisposition ||
		a.RetainedObligation != b.RetainedObligation || a.Note != b.Note {
		return false
	}
	if len(a.ItemIDs) != len(b.ItemIDs) {
		return false
	}
	for i := range a.ItemIDs {
		if a.ItemIDs[i] != b.ItemIDs[i] {
			return false
		}
	}
	if len(a.Evidence) != len(b.Evidence) {
		return false
	}
	for i := range a.Evidence {
		if a.Evidence[i] != b.Evidence[i] {
			return false
		}
	}
	if (a.Provenance == nil) != (b.Provenance == nil) {
		return false
	}
	if a.Provenance != nil && *a.Provenance != *b.Provenance {
		return false
	}
	return true
}

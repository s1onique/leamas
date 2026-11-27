package domain

import (
	"strings"
	"testing"
)

// TestValidateRejectsInvalidSourceRefs proves the canonical validator
// rejects a source reference that does not bind an exact byte range.
//
// A source reference without a one-based range or without a digest does not
// bind anything: the provenance of the record would be an assertion.
func TestValidateRejectsInvalidSourceRefs(t *testing.T) {
	cases := []struct {
		name string
		ref  SourceRef
	}{
		{
			name: "no path",
			ref:  SourceRef{StartLine: 1, EndLine: 2, SHA256: strings.Repeat("a", 64)},
		},
		{
			name: "zero start line",
			ref:  SourceRef{Path: "a.md", EndLine: 2, SHA256: strings.Repeat("a", 64)},
		},
		{
			name: "end before start",
			ref:  SourceRef{Path: "a.md", StartLine: 5, EndLine: 2, SHA256: strings.Repeat("a", 64)},
		},
		{
			name: "malformed digest",
			ref:  SourceRef{Path: "a.md", StartLine: 1, EndLine: 2, SHA256: "not-a-digest"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := Ledger{
				SchemaVersion: LedgerSchemaVersion,
				Epics: []Epic{{
					SchemaVersion: LedgerSchemaVersion,
					EpicID:        "EPIC-A",
					Title:         "A",
					Revision:      1,
				}},
				Items: []WorkItem{{
					SchemaVersion: LedgerSchemaVersion,
					EpicID:        "EPIC-A",
					ItemID:        "ITEM-1",
					Title:         "One",
					Kind:          ItemKindTask,
					Lifecycle:     LifecycleOpen,
					Acceptance:    AcceptancePending,
					Revision:      1,
					SourceRefs:    []SourceRef{tc.ref},
				}},
			}
			res := ValidateLedger(l)
			if res.OK() {
				t.Fatalf("the invalid source reference was accepted: %+v", tc.ref)
			}
		})
	}
}

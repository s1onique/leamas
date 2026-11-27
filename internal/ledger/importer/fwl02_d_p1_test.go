package importer

import (
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// This file qualifies the bounded P1 corrections of the FWL02-C consumer.
//
// P1-A — the unresolved relationship metadata is preserved.
// P1-B — the incomplete prerequisite coverage is reported distinctly from a
//        verified absence of prerequisites.

// TestP1AUnresolvedRelationshipMetadataIsPreserved witnesses P1-A.
//
// The contract: an unresolved relationship is never reduced to a bare target
// identity. Its kind, the outcome it required and its byte-exact source
// locations are conserved in the explicit report.
func TestP1AUnresolvedRelationshipMetadataIsPreserved(t *testing.T) {
	source := SourceBoard{
		Source: "p1a-source",
		Items: []SourceItem{{
			SourceID: "W-01", EpicID: "EPIC-A", Title: "first", Status: SourceStatusOpen,
			Relations: []SourceRelation{{
				Kind:            "CROSS_LINK",
				Target:          "EPIC-B/X",
				RequiredOutcome: "the cross-linked obligation is qualified",
				SourceRefs: []domain.SourceRef{{
					Path: "board.md", StartLine: 10, EndLine: 12,
					SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				}},
			}},
		}},
	}
	plan := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	found := false
	for _, ref := range plan.Unresolved {
		if ref.Kind != "relation" {
			continue
		}
		found = true
		if ref.RelationshipKind != "CROSS_LINK" {
			t.Errorf("relationship kind = %q", ref.RelationshipKind)
		}
		if ref.RequiredOutcome != "the cross-linked obligation is qualified" {
			t.Errorf("required outcome = %q", ref.RequiredOutcome)
		}
		if len(ref.SourceRefs) != 1 {
			t.Errorf("source refs = %v", ref.SourceRefs)
		}
	}
	if !found {
		t.Fatalf("the unresolved relationship was not reported: %+v", plan.Unresolved)
	}
}

// TestP1BIncompletePrerequisiteCoverageIsReported witnesses P1-B.
//
// The contract: a source that declares an INCOMPLETE dependency coverage has
// not recovered the prerequisites of the record. The importer reports it and
// never treats it as a verified absence.
func TestP1BIncompletePrerequisiteCoverageIsReported(t *testing.T) {
	cases := []struct {
		name     string
		coverage string
		reported bool
	}{
		{"complete", "COMPLETE", false},
		{"unknown", "UNKNOWN", true},
		{"partial", "PARTIAL", true},
		{"undeclared", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := SourceBoard{
				Source: "p1b-source",
				Items: []SourceItem{{
					SourceID: "W-01", EpicID: "EPIC-A", Title: "first",
					Status:             SourceStatusOpen,
					DependencyCoverage: tc.coverage,
				}},
			}
			plan := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
			reported := false
			for _, ref := range plan.Unresolved {
				if ref.Kind == "incomplete_dependency_coverage" {
					reported = true
					if ref.Ref != tc.coverage {
						t.Errorf("reported coverage = %q, want %q", ref.Ref, tc.coverage)
					}
				}
			}
			if reported != tc.reported {
				t.Fatalf("the incomplete coverage report = %t, want %t: %+v",
					reported, tc.reported, plan.Unresolved)
			}
			// The coverage itself is conserved on the record either way.
			if plan.Items[0].DependencyCoverage != tc.coverage {
				t.Errorf("the conserved coverage = %q, want %q",
					plan.Items[0].DependencyCoverage, tc.coverage)
			}
		})
	}
}

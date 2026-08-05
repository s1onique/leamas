package closure

import (
	"testing"
)

// TestV2TopologyFactsClassify covers every classified
// relation without any git calls.
func TestV2TopologyFactsClassify(t *testing.T) {
	cases := []struct {
		name   string
		facts  V2TopologyFacts
		expect V2Relation
	}{
		{"missing_subject", V2TopologyFacts{SubjectResolved: false}, V2RelationMissingSubject},
		{"missing_freeze", V2TopologyFacts{SubjectResolved: true, SubjectCommit: "s", FreezeResolved: false}, V2RelationMissingFreeze},
		{"equal", V2TopologyFacts{
			SubjectResolved: true, SubjectCommit: "s",
			FreezeResolved: true, FreezeCommit: "s", Equal: true,
		}, V2RelationEqual},
		{"subject_before_freeze", V2TopologyFacts{
			SubjectResolved: true, SubjectCommit: "s",
			FreezeResolved: true, FreezeCommit: "f",
			SubjectAncestorFreeze: true, FreezeAncestorSubject: false,
		}, V2RelationSubjectBeforeFreeze},
		{"freeze_before_subject", V2TopologyFacts{
			SubjectResolved: true, SubjectCommit: "s",
			FreezeResolved: true, FreezeCommit: "f",
			SubjectAncestorFreeze: false, FreezeAncestorSubject: true,
		}, V2RelationFreezeBeforeSubject},
		{"unrelated", V2TopologyFacts{
			SubjectResolved: true, SubjectCommit: "s",
			FreezeResolved: true, FreezeCommit: "f",
			SubjectAncestorFreeze: false, FreezeAncestorSubject: false,
		}, V2RelationSubjectFreezeUnrelated},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := c.facts.Classify()
			if got != c.expect {
				t.Fatalf("got %s want %s", got, c.expect)
			}
		})
	}
}

// TestDispatchClosureTopologyRepositoryBound covers every
// topology verdict that the legacy boolean dispatch could not
// distinguish. Facts come from V2TopologyFacts, never from
// caller-supplied booleans.
func TestDispatchClosureTopologyRepositoryBound(t *testing.T) {
	cases := []struct {
		name           string
		version        ClosureProtocolVersion
		facts          V2TopologyFacts
		wantAccepted   bool
		wantCode       V2DiagnosticCode
		wantHasSubject bool
		wantHasFreeze  bool
	}{
		{
			name:    "v2_subject_before_freeze_accepts",
			version: ClosureProtocolV2,
			facts: V2TopologyFacts{
				SubjectResolved: true, FreezeResolved: true,
				SubjectAncestorFreeze: true, FreezeAncestorSubject: false,
			},
			wantAccepted:   true,
			wantHasSubject: true,
			wantHasFreeze:  true,
		},
		{
			name:    "v2_freeze_before_subject_rejects_with_freeze_ancestor",
			version: ClosureProtocolV2,
			facts: V2TopologyFacts{
				SubjectResolved: true, FreezeResolved: true,
				SubjectAncestorFreeze: false, FreezeAncestorSubject: true,
			},
			wantAccepted: false,
			wantCode:     V2CodeFreezeAncestorOfSubject,
		},
		{
			name:    "v2_unrelated_rejects",
			version: ClosureProtocolV2,
			facts: V2TopologyFacts{
				SubjectResolved: true, FreezeResolved: true,
				SubjectAncestorFreeze: false, FreezeAncestorSubject: false,
			},
			wantAccepted: false,
			wantCode:     V2CodeSubjectFreezeUnrelated,
		},
		{
			name:    "v2_equal_rejects",
			version: ClosureProtocolV2,
			facts: V2TopologyFacts{
				SubjectResolved: true, FreezeResolved: true, Equal: true,
			},
			wantAccepted: false,
			wantCode:     V2CodeSubjectEqualsFreeze,
		},
		{
			name:    "v1_freeze_before_subject_accepts",
			version: ClosureProtocolV1,
			facts: V2TopologyFacts{
				SubjectResolved: true, FreezeResolved: true,
				SubjectAncestorFreeze: false, FreezeAncestorSubject: true,
			},
			wantAccepted:   true,
			wantHasSubject: true,
			wantHasFreeze:  true,
		},
		{
			name:    "v1_subject_before_freeze_rejects_with_subject_not_ancestor",
			version: ClosureProtocolV1,
			facts: V2TopologyFacts{
				SubjectResolved: true, FreezeResolved: true,
				SubjectAncestorFreeze: true, FreezeAncestorSubject: false,
			},
			wantAccepted: false,
			wantCode:     V2CodeSubjectNotAncestorOfFreeze,
		},
		{
			name:         "missing_subject_rejects",
			version:      ClosureProtocolV2,
			facts:        V2TopologyFacts{SubjectResolved: false},
			wantAccepted: false,
			wantCode:     V2CodeSubjectCommitNotFound,
		},
		{
			name:    "missing_freeze_rejects",
			version: ClosureProtocolV2,
			facts: V2TopologyFacts{
				SubjectResolved: true, SubjectCommit: "s", FreezeResolved: false,
			},
			wantAccepted: false,
			wantCode:     V2CodeFreezeCommitNotFound,
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := DispatchClosureTopology(c.version, c.facts)
			if got.Accepted != c.wantAccepted {
				t.Fatalf("Accepted=%v want %v (code=%s)", got.Accepted, c.wantAccepted, got.Code)
			}
			if !c.wantAccepted && got.Code != c.wantCode {
				t.Fatalf("Code=%s want %s", got.Code, c.wantCode)
			}
			if got.HasSubject != c.wantHasSubject {
				t.Fatalf("HasSubject=%v want %v", got.HasSubject, c.wantHasSubject)
			}
			if got.HasFreeze != c.wantHasFreeze {
				t.Fatalf("HasFreeze=%v want %v", got.HasFreeze, c.wantHasFreeze)
			}
		})
	}
}

// TestV2VersionCombinationMatrix covers the closed version
// matrix end-to-end. Every unsupported combination must emit
// a typed V2Error; only v1+v1 and v1+v2 must be accepted.
func TestV2VersionCombinationMatrix(t *testing.T) {
	cases := []struct {
		name      string
		plan      PlanContractVersion
		closure   ClosureProtocolVersion
		supported bool
	}{
		{"v1+v1", PlanContractV1, ClosureProtocolV1, true},
		{"v1+v2", PlanContractV1, ClosureProtocolV2, true},
		{"v0+v2", PlanContractVersion(0), ClosureProtocolV2, false},
		{"v2+v2", PlanContractVersion(2), ClosureProtocolV2, false},
		{"v9+v2", PlanContractVersion(9), ClosureProtocolV2, false},
		{"v1+v9", PlanContractV1, ClosureProtocolVersion("9"), false},
		{"neg+v1", PlanContractVersion(-1), ClosureProtocolV1, false},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			err := ValidateV2VersionCombination(c.plan, c.closure)
			if c.supported && err != nil {
				t.Fatalf("expected supported, got %v", err)
			}
			if !c.supported && err == nil {
				t.Fatalf("expected rejection for %s", c.name)
			}
		})
	}
}

// TestV2RequestValidation rejects incomplete and inconsistent
// requests so the runner never reaches topology with
// zero-value fields.
func TestV2RequestValidation(t *testing.T) {
	t.Run("missing_subject_reports_incomplete", func(t *testing.T) {
		req := V2Request{
			ClosureProtocolVersion: ClosureProtocolV2,
			PlanContractVersion:    1,
			RepositoryRoot:         t.TempDir(),
			FreezeCommit:           "f",
			PlanPath:               "plan.json",
			EvidenceDirectory:      t.TempDir(),
			ManifestOutput:         t.TempDir() + "/manifest.json",
		}
		err := ValidateV2Request(req)
		if err == nil {
			t.Fatalf("expected validation failure")
		}
		v2err, ok := err.(*V2Error)
		if !ok {
			t.Fatalf("expected *V2Error, got %T", err)
		}
		if !v2err.Diags.HasCode(V2CodeRequestIncomplete) {
			t.Fatalf("expected request_incomplete, got %v", v2err.Diags.Codes())
		}
	})
}

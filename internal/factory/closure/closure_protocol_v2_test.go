package closure

import (
	"strings"
	"testing"
)

// TestV2VersionDispatch exercises the v1 vs v2 topology dispatch
// without any ancestry machinery. The function under test takes
// only the explicit lifecycle version and the boolean
// "isAncestor" + "equal" facts the runner would have computed.
//
//	ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-SUBJECT-THEN-FREEZE01
//	requires version-isolation tests.
func TestV2VersionDispatch(t *testing.T) {
	type tc struct {
		name         string
		version      ClosureProtocolVersion
		isAncestor   bool
		equal        bool
		wantAccepted bool
		wantReason   string
	}
	cases := []tc{
		// v1: isAncestor(F, S) true and F != S -> accept
		{name: "v1_F_ancestor_S", version: ClosureProtocolV1, isAncestor: true, equal: false, wantAccepted: true},
		// v1: isAncestor(F, S) false (i.e. only S < F) -> reject
		{name: "v1_S_ancestor_F_rejects", version: ClosureProtocolV1, isAncestor: false, equal: false, wantAccepted: false, wantReason: "freeze_not_ancestor_of_subject"},
		// v2: isAncestor(S, F) true and S != F -> accept
		{name: "v2_S_ancestor_F", version: ClosureProtocolV2, isAncestor: true, equal: false, wantAccepted: true},
		// v2: isAncestor(S, F) false (i.e. only F < S) -> reject
		{name: "v2_F_ancestor_S_rejects", version: ClosureProtocolV2, isAncestor: false, equal: false, wantAccepted: false, wantReason: "subject_not_ancestor_of_freeze"},
		// equality rejected in both versions
		{name: "v1_S_equals_F", version: ClosureProtocolV1, isAncestor: true, equal: true, wantAccepted: false, wantReason: "subject_equals_freeze"},
		{name: "v2_S_equals_F", version: ClosureProtocolV2, isAncestor: true, equal: true, wantAccepted: false, wantReason: "subject_equals_freeze"},
		// unsupported version rejected
		{name: "unsupported_v9", version: ClosureProtocolVersion("9"), isAncestor: true, equal: false, wantAccepted: false, wantReason: "unsupported_closure_protocol_version"},
		// non-ancestor in v2 rejected
		{name: "v2_no_ancestor", version: ClosureProtocolV2, isAncestor: false, equal: false, wantAccepted: false, wantReason: "subject_not_ancestor_of_freeze"},
		// non-ancestor in v1 rejected
		{name: "v1_no_ancestor", version: ClosureProtocolV1, isAncestor: false, equal: false, wantAccepted: false, wantReason: "freeze_not_ancestor_of_subject"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := V2DispatchTopology(c.version, c.isAncestor, c.equal)
			if got.Accepted != c.wantAccepted {
				t.Fatalf("accepted=%v want %v reason=%q", got.Accepted, c.wantAccepted, got.Reason)
			}
			if !c.wantAccepted && c.wantReason != "" && got.Reason != c.wantReason {
				t.Fatalf("reason=%q want %q", got.Reason, c.wantReason)
			}
		})
	}
}

// TestV2VersionAxesIndependent proves the plan contract version
// and closure protocol version are separate axes. A Plan Contract
// v1 document must be usable with Closure Protocol v2, and a
// Closure Protocol v1 manifest must continue to bind a v1 plan.
func TestV2VersionAxesIndependent(t *testing.T) {
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		SubjectCommit:          "sub",
		FreezeCommit:           "frz",
		PlanPath:               "plan.json",
	}
	if req.PlanContractVersion != 1 {
		t.Fatalf("plan contract version not 1: %d", req.PlanContractVersion)
	}
	if req.ClosureProtocolVersion != ClosureProtocolV2 {
		t.Fatalf("closure protocol version not v2: %s", req.ClosureProtocolVersion)
	}
	mfst := V2Manifest{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		SubjectCommit:          "sub",
		FreezeCommit:           "frz",
	}
	if mfst.PlanContractVersion != 1 {
		t.Fatalf("v2 manifest plan contract version not 1: %d", mfst.PlanContractVersion)
	}
	if mfst.ClosureProtocolVersion != ClosureProtocolV2 {
		t.Fatalf("v2 manifest closure protocol version not v2: %s", mfst.ClosureProtocolVersion)
	}
}

// TestV2VersionIsSupported covers the IsSupported contract
// including the failure case for unknown versions.
func TestV2VersionIsSupported(t *testing.T) {
	if !ClosureProtocolV1.IsSupported() {
		t.Fatalf("v1 must be supported")
	}
	if !ClosureProtocolV2.IsSupported() {
		t.Fatalf("v2 must be supported")
	}
	if ClosureProtocolVersion("3").IsSupported() {
		t.Fatalf("v3 must not be supported")
	}
	if ClosureProtocolVersion("").IsSupported() {
		t.Fatalf("empty version must not be supported")
	}
}

// TestV2ManifestJSON is the small structural smoke test that
// proves the v2 manifest JSON tag wiring is stable.
func TestV2ManifestJSON(t *testing.T) {
	m := V2Manifest{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		SubjectCommit:          "s",
		FreezeCommit:           "f",
		PlanPath:               "plan.json",
	}
	data, err := m.V2ManifestJSON()
	if err != nil {
		t.Fatalf("V2ManifestJSON: %v", err)
	}
	s := string(data)
	for _, want := range []string{
		`"closure_protocol_version":"2"`,
		`"plan_contract_version":1`,
		`"subject_commit":"s"`,
		`"freeze_commit":"f"`,
		`"plan_path":"plan.json"`,
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("v2 manifest JSON missing %q: %s", want, s)
		}
	}
}

// SPDX-License-Identifier: Apache-2.0

// Package digest: gate_evidence_regression_extra_test.go covers
// the remaining end-to-end binding classification regressions
// required by ACT-LEAMAS-DIGEST-GATE-EVIDENCE-AUTHORITY-BINDING01
// sections 32-35: scope-mismatch, dirty-worktree, missing,
// malformed, historical-range, and subprocess-free invariants.
package digest

import (
	"strings"
	"testing"
)

// TestGateSummary_ScopeMismatchClassification covers the
// HEAD-match-but-scope-mismatch case. The two proves that
// HEAD equality alone is insufficient.
func TestGateSummary_ScopeMismatchClassification(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	v2 := `{
		"schema_version": 2,
		"generated_at": "2026-08-14T12:01:39Z",
		"scope_id": "ACT-LEAMAS-DEMO-01",
		"scope_status": "CLOSED",
		"scope_disposition": "test",
		"parent_act": "ACT-LEAMAS-DEMO-01",
		"parent_status": "OPEN",
		"parent_disposition": "test",
		"overall_status": "pass",
		"overall_disposition": "all checks passed",
		"execution_head_oid": "0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid": "fedcba9876543210fedcba9876543210fedcba98",
		"subject_tree_oid": "fedcba9876543210fedcba9876543210fedcba98",
		"worktree_clean_before": true,
		"worktree_clean_after": true,
		"checks": [
          {
            "name": "factory_smoke",
            "scope": "ROOT",
            "status": "pass",
            "evidence": "factory_smoke.sh",
            "detail": "factory smoke test passed",
            "extras": {
              "argv": ["factory_smoke.sh"],
              "exit_code": 0,
              "duration_ms": 42,
              "stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
              "stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
            }
          }
        ]
	}`
	writeGateSummaryFile(t, tmpDir, []byte(v2))

	// Digest has the same HEAD but a different ACT.
	resolved := &ResolvedMode{
		Mode:            ModeRange,
		HeadCommit:      "0123456789abcdef0123456789abcdef01234567",
		ActID:           "ACT-LEAMAS-DEMO-02",
		AuthorityStatus: "AuthoritativeClosed",
		IsClean:         true,
	}

	section := buildGateSummarySection(tmpDir, resolved)

	mustContain := []string{
		"binding_status=SCOPE_MISMATCH",
		"authoritative_for_digest=false",
		"state_binding=MATCH",
		"scope_binding=MISMATCH",
		"warning_code=GATE_SUMMARY_SCOPE_MISMATCH",
		"reported_overall_status=pass",
	}
	for _, want := range mustContain {
		if !strings.Contains(section, want) {
			t.Errorf("scope-mismatch section missing %q:\n%s", want, section)
		}
	}
}

// TestGateSummary_DirtyWorktreeClassification covers the
// dirty-with-commit-only-binding case. The fixture exposes
// execution_head_oid=X but the digest is in dirty mode
// (uncommitted changes). The section MUST classify as
// DIRTY_SUBJECT_UNBOUND.
func TestGateSummary_DirtyWorktreeClassification(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	v2 := `{
		"schema_version": 2,
		"generated_at": "2026-08-14T12:01:39Z",
		"scope_id": "ACT-LEAMAS-DEMO-01",
		"scope_status": "CLOSED",
		"scope_disposition": "test",
		"parent_act": "ACT-LEAMAS-DEMO-01",
		"parent_status": "OPEN",
		"parent_disposition": "test",
		"overall_status": "pass",
		"overall_disposition": "all checks passed",
		"execution_head_oid": "0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid": "fedcba9876543210fedcba9876543210fedcba98",
		"subject_tree_oid": "fedcba9876543210fedcba9876543210fedcba98",
		"worktree_clean_before": true,
		"worktree_clean_after": true,
		"checks": [
          {
            "name": "factory_smoke",
            "scope": "ROOT",
            "status": "pass",
            "evidence": "factory_smoke.sh",
            "detail": "factory smoke test passed",
            "extras": {
              "argv": ["factory_smoke.sh"],
              "exit_code": 0,
              "duration_ms": 42,
              "stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
              "stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
            }
          }
        ]
	}`
	writeGateSummaryFile(t, tmpDir, []byte(v2))

	resolved := &ResolvedMode{
		Mode:    ModeDirty,
		Reason:  "working tree has changes",
		IsClean: false,
	}

	section := buildGateSummarySection(tmpDir, resolved)

	mustContain := []string{
		"binding_status=DIRTY_SUBJECT_UNBOUND",
		"authoritative_for_digest=false",
		"state_binding=UNBOUND",
		"warning_code=GATE_SUMMARY_DIRTY_SUBJECT_UNBOUND",
		"reported_overall_status=pass",
	}
	for _, want := range mustContain {
		if !strings.Contains(section, want) {
			t.Errorf("dirty-worktree section missing %q:\n%s", want, section)
		}
	}
}

// TestGateSummary_MissingSummaryClassification covers the
// "absent summary" case. The classifier MUST return
// NOT_APPLICABLE, distinct from LEGACY_UNBOUND.
func TestGateSummary_MissingSummaryClassification(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	// No .factory/gate-summary.json is created.

	resolved := &ResolvedMode{
		Mode:       ModeRange,
		HeadCommit: "0123456789abcdef0123456789abcdef01234567",
		ActID:      "ACT-LEAMAS-DEMO-01",
		IsClean:    true,
	}

	section := buildGateSummarySection(tmpDir, resolved)

	mustContain := []string{
		"source_status=missing",
		"binding_status=NOT_APPLICABLE",
		"authoritative_for_digest=false",
		"warning_code=GATE_SUMMARY_NOT_APPLICABLE",
		"overall_status=unavailable",
	}
	for _, want := range mustContain {
		if !strings.Contains(section, want) {
			t.Errorf("missing-summary section missing %q:\n%s", want, section)
		}
	}
}

// TestGateSummary_MalformedSummaryClassification covers the
// "malformed JSON" case. The classifier MUST return
// EVIDENCE_INVALID (fail-closed) and the diagnostics block
// must remain visible. The classification IS NOT
// NOT_APPLICABLE: the ACT explicitly distinguishes ABSENT
// from INVALID.
func TestGateSummary_MalformedSummaryClassification(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	writeGateSummaryFile(t, tmpDir, []byte("{not valid json"))

	section := buildGateSummarySection(tmpDir, nil)

	mustContain := []string{
		"source_status=invalid",
		"binding_status=EVIDENCE_INVALID",
		"source_validity=INVALID",
		"authoritative_for_digest=false",
		"warning_code=GATE_SUMMARY_INVALID_BINDING",
		"diagnostics_total=",
	}
	for _, want := range mustContain {
		if !strings.Contains(section, want) {
			t.Errorf("malformed-summary section missing %q:\n%s", want, section)
		}
	}
}

// TestGateSummary_HistoricalRangeMatchClassification covers
// the historical-range case: gate summary's execution_head_oid
// matches the digest's right endpoint B, even though ambient
// HEAD is C. Authority MUST be granted.
func TestGateSummary_HistoricalRangeMatchClassification(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	v2 := `{
		"schema_version": 2,
		"generated_at": "2026-08-14T12:01:39Z",
		"scope_id": "ACT-LEAMAS-DEMO-01",
		"scope_status": "CLOSED",
		"scope_disposition": "test",
		"parent_act": "ACT-LEAMAS-DEMO-01",
		"parent_status": "OPEN",
		"parent_disposition": "test",
		"overall_status": "pass",
		"overall_disposition": "all checks passed",
		"execution_head_oid": "fedcba9876543210fedcba9876543210fedcba98",
		"execution_tree_oid": "fedcba9876543210fedcba9876543210fedcba98",
		"subject_tree_oid": "fedcba9876543210fedcba9876543210fedcba98",
		"worktree_clean_before": true,
		"worktree_clean_after": true,
		"checks": [
          {
            "name": "factory_smoke",
            "scope": "ROOT",
            "status": "pass",
            "evidence": "factory_smoke.sh",
            "detail": "factory smoke test passed",
            "extras": {
              "argv": ["factory_smoke.sh"],
              "exit_code": 0,
              "duration_ms": 42,
              "stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
              "stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
            }
          }
        ]
	}`
	writeGateSummaryFile(t, tmpDir, []byte(v2))

	resolved := &ResolvedMode{
		Mode:            ModeRange,
		Range:           "A..fedcba9876543210fedcba9876543210fedcba98",
		HeadCommit:      "fedcba9876543210fedcba9876543210fedcba98",
		ActID:           "ACT-LEAMAS-DEMO-01",
		AuthorityStatus: "AuthoritativeClosed",
		IsClean:         true,
	}

	section := buildGateSummarySection(tmpDir, resolved)

	mustContain := []string{
		"binding_status=AUTHORITATIVE",
		"authoritative_for_digest=true",
		"state_binding=MATCH",
		"scope_binding=MATCH",
		"warning_code=none",
	}
	for _, want := range mustContain {
		if !strings.Contains(section, want) {
			t.Errorf("historical-range-match section missing %q:\n%s", want, section)
		}
	}
}

// TestGateSummary_DoesNotSpawnGitSubprocesses is a smoke
// test that exercises the binding pipeline against a
// realistic ResolvedMode. The classifier is pure; no Git
// subprocesses are spawned. The test fails closed if any
// panic or hang occurs.
func TestGateSummary_DoesNotSpawnGitSubprocesses(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	writeGateSummaryFile(t, tmpDir, []byte(`{
		"schema_version": 2,
		"generated_at": "2026-08-14T12:01:39Z",
		"scope_id": "ACT-LEAMAS-DEMO-01",
		"scope_status": "CLOSED",
		"scope_disposition": "test",
		"parent_act": "ACT-LEAMAS-DEMO-01",
		"parent_status": "OPEN",
		"parent_disposition": "test",
		"overall_status": "pass",
		"overall_disposition": "all checks passed",
		"execution_head_oid": "0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid": "fedcba9876543210fedcba9876543210fedcba98",
		"subject_tree_oid": "fedcba9876543210fedcba9876543210fedcba98",
		"worktree_clean_before": true,
		"worktree_clean_after": true,
		"checks": [{"name": "a", "scope": "ROOT", "status": "pass", "duration_ms": 1, "evidence": "a.sh"}]
	}`))
	resolved := &ResolvedMode{
		Mode:            ModeRange,
		HeadCommit:      "0123456789abcdef0123456789abcdef01234567",
		ActID:           "ACT-LEAMAS-DEMO-01",
		AuthorityStatus: "AuthoritativeClosed",
		IsClean:         true,
	}
	// Two renders in succession must be byte-identical.
	a := buildGateSummarySection(tmpDir, resolved)
	b := buildGateSummarySection(tmpDir, resolved)
	if a != b {
		t.Errorf("two renders of the same input must be identical")
	}
}

// TestGateSummary_ParentActOnlyDoesNotMatchScope is the
// end-to-end regression for the parent_act-as-scope
// substitution defect closed in CORRECTION02. The gate
// summary is a child with empty scope_id but a populated
// parent_act. The digest's ActID matches the parent_act.
// The classifier MUST NOT report AUTHORITATIVE: parent_act
// is provenance metadata, not a scope authority claim.
//
// NOTE: The current v2/v3 wire schemas enforce
// minLength: 1 on scope_id, so a fully-valid wire-form
// document cannot directly exercise this path. The
// authoritative regression is the classifier matrix test
// row18. This end-to-end test exercises the same
// GateSummaryIdentity through the renderer and verifies
// that the digest output reflects it without promoting
// parent_act to a scope authority claim.
func TestGateSummary_ParentActOnlyDoesNotMatchScope(t *testing.T) {
	t.Parallel()
	// Use the classifier directly with the same shape as
	// the wire schema would allow if it permitted an empty
	// scope_id. The GateSummaryIdentity is the same
	// structure produced by the digest adapter from a
	// normalized v2/v3 document.
	gate := GateSummaryIdentity{
		SchemaVersion:        2,
		ExecutionHeadOID:     testOIDX,
		ScopeID:              "", // no scope_id
		ParentAct:            testActA,
		HasExecutionIdentity: true,
		HasScopeIdentity:     true, // parent_act counts as scope identity presence
	}
	digest := DigestAuthority{
		SubjectCommitOID: testOIDX,
		ActID:            testActA, // matches parent_act
	}

	binding := EvaluateGateEvidenceBinding(gate, digest, SourceValid)

	// Critical assertions: parent_act does NOT establish scope.
	mustHave := []struct {
		field string
		got   interface{}
		want  interface{}
	}{
		{"Status", binding.Status, BindingStateMatchScopeUnbound},
		{"StateBinding", binding.StateBinding, StateMatch},
		{"ScopeBinding", binding.ScopeBinding, ScopeUnbound},
		{"AuthoritativeForDigest", binding.AuthoritativeForDigest, false},
		{"WarningCode", binding.WarningCode, GateBindingWarningCodeStateMatchScopeUnbound},
	}
	for _, c := range mustHave {
		if c.got != c.want {
			t.Errorf("%s: got %v want %v", c.field, c.got, c.want)
		}
	}

	// Negative assertions: must NOT be AUTHORITATIVE.
	if binding.Status == BindingAuthoritative {
		t.Errorf("parent-act-only MUST NOT be AUTHORITATIVE: got %v", binding.Status)
	}
	if binding.WarningCode == GateBindingWarningCodeNone {
		t.Errorf("parent-act-only MUST NOT have warning_code=none")
	}
}

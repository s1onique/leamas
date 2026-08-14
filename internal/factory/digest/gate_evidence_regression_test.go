// SPDX-License-Identifier: Apache-2.0

// Package digest: gate_evidence_regression_test.go covers the
// end-to-end rendering of the GATE_SUMMARY section against the
// fixture shapes called out in ACT-LEAMAS-DIGEST-GATE-EVIDENCE-
// AUTHORITY-BINDING01 sections 29-35. Each test asserts the
// binding classification, the warning code, and the rendering
// contract.
package digest

import (
	"strings"
	"testing"
)

// TestGateSummary_LegacyV1Reproduction reproduces the file
// observed in the motivating defect:
//
//	.factory/gate-summary.json
//	schema_version=1
//	generated_at=2026-07-18T18:29:21Z
//	overall_status=pass
//
// The digest MUST render:
//
//	source_status=present
//	binding_status=LEGACY_UNBOUND
//	authoritative_for_digest=false
//	reported_overall_status=pass
//
// (the historical PASS remains visible; it is no longer
// labelled as the current verdict).
func TestGateSummary_LegacyV1Reproduction(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	legacy := `{
		"schema_version": 1,
		"generated_at": "2026-07-18T18:29:21Z",
		"tool": "leamas factory gate",
		"overall_status": "pass",
		"checks": [
			{"name": "gate", "status": "pass", "duration_ms": 268587, "evidence": "leamas factory gate"}
		]
	}`
	writeGateSummaryFile(t, tmpDir, []byte(legacy))

	section := buildGateSummarySection(tmpDir, nil)

	mustContain := []string{
		"source_status=present",
		"binding_status=LEGACY_UNBOUND",
		"authoritative_for_digest=false",
		"state_binding=UNBOUND",
		"scope_binding=UNBOUND",
		"warning_code=GATE_SUMMARY_LEGACY_UNBOUND",
		"reported_overall_status=pass",
	}
	for _, want := range mustContain {
		if !strings.Contains(section, want) {
			t.Errorf("legacy v1 section missing %q:\n%s", want, section)
		}
	}

	// The original overall_status= field IS preserved for
	// backwards compatibility with consumers that read it
	// (per the digest contract compatibility correction).
	// HOWEVER, the authoritative qualifier MUST be false
	// and the warning_code MUST be present. The presence
	// of overall_status=pass without an authoritative
	// qualifier must NOT be the only signal.
	// We require the qualifier to be adjacent to the
	// verdict line.
	if !strings.Contains(section, "\noverall_status_authoritative=false\n") {
		t.Errorf("legacy v1 section must declare overall_status_authoritative=false:\n%s", section)
	}
	if !strings.Contains(section, "\nreported_overall_status=pass\n") {
		t.Errorf("legacy v1 section must emit reported_overall_status=pass alias:\n%s", section)
	}
}

// TestGateSummary_AuthoritativeV2Rendering covers the
// fully-bound case: gate summary v2 with execution_head_oid and
// scope_id, against a digest that resolves to the same OID and
// ACT. The unqualified overall_status line is permitted.
func TestGateSummary_AuthoritativeV2Rendering(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	v2 := `{
		"schema_version": 2,
		"generated_at": "2026-08-14T12:01:39Z",
		"scope_id": "ACT-LEAMAS-DIGEST-GATE-EVIDENCE-AUTHORITY-BINDING01",
		"scope_status": "CLOSED",
		"scope_disposition": "test",
		"parent_act": "ACT-LEAMAS-DIGEST-GATE-EVIDENCE-AUTHORITY-BINDING01",
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
              "stdout_sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
              "stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
            }
          }
        ]
	}`
	writeGateSummaryFile(t, tmpDir, []byte(v2))

	resolved := &ResolvedMode{
		Mode:            ModeRange,
		Range:           "0123456789abcdef0123456789abcdef01234567^..0123456789abcdef0123456789abcdef01234567",
		HeadCommit:      "0123456789abcdef0123456789abcdef01234567",
		ActID:           "ACT-LEAMAS-DIGEST-GATE-EVIDENCE-AUTHORITY-BINDING01",
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
		"overall_status=pass",
		"gate_execution_head_oid=0123456789abcdef0123456789abcdef01234567",
		"gate_scope_id=ACT-LEAMAS-DIGEST-GATE-EVIDENCE-AUTHORITY-BINDING01",
	}
	for _, want := range mustContain {
		if !strings.Contains(section, want) {
			t.Errorf("authoritative v2 section missing %q:\n%s", want, section)
		}
	}
}

// TestGateSummary_StateMismatchClassification covers the
// stale-state case: gate summary points to OID X, digest
// subject is Y. The section MUST classify as STATE_MISMATCH
// while preserving the historical PASS as reported_overall_status.
func TestGateSummary_StateMismatchClassification(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	v2 := `{
		"schema_version": 2,
		"generated_at": "2026-07-18T18:29:21Z",
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
              "stdout_sha256": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
              "stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
            }
          }
        ]
	}`
	writeGateSummaryFile(t, tmpDir, []byte(v2))

	// Digest subject is Y, not X.
	resolved := &ResolvedMode{
		Mode:            ModeRange,
		HeadCommit:      "89abcdef0123456789abcdef0123456789abcdef01",
		ActID:           "ACT-LEAMAS-DEMO-01",
		AuthorityStatus: "AuthoritativeClosed",
		IsClean:         true,
	}

	section := buildGateSummarySection(tmpDir, resolved)

	mustContain := []string{
		"binding_status=STATE_MISMATCH",
		"authoritative_for_digest=false",
		"state_binding=MISMATCH",
		"warning_code=GATE_SUMMARY_STATE_MISMATCH",
		"reported_overall_status=pass",
	}
	for _, want := range mustContain {
		if !strings.Contains(section, want) {
			t.Errorf("state-mismatch section missing %q:\n%s", want, section)
		}
	}

	// The original overall_status= field IS preserved for
	// backwards compatibility with consumers that read it
	// (per the digest contract compatibility correction).
	// HOWEVER, the authoritative qualifier MUST be false.
	// The presence of overall_status=pass without an
	// authoritative qualifier must NOT be the only signal.
	if !strings.Contains(section, "\noverall_status_authoritative=false\n") {
		t.Errorf("state-mismatch section must declare overall_status_authoritative=false:\n%s", section)
	}
}

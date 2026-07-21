// Package gate provides tests for factorize metrics v3 invariants.
package gate

import (
	"os"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// TestMetricsSchema_IsExactlyV3 verifies schema is v3.
func TestMetricsSchema_IsExactlyV3(t *testing.T) {
	if MetricsSchema != "factorize-performance-v3" {
		t.Errorf("expected schema v3, got %s", MetricsSchema)
	}
}

func TestNewMetricsCollectionV3_RequiresScenario(t *testing.T) {
	_, err := NewMetricsCollectionV3("/tmp/metrics.json", "", "1")
	if err == nil {
		t.Fatalf("expected error for missing scenario")
	}
}

func TestNewMetricsCollectionV3_RequiresSequence(t *testing.T) {
	_, err := NewMetricsCollectionV3("/tmp/metrics.json", "controlled-warm", "")
	if err == nil {
		t.Fatalf("expected error for missing sequence")
	}
}

func TestNewMetricsCollectionV3_RejectsUnknownScenario(t *testing.T) {
	_, err := NewMetricsCollectionV3("/tmp/metrics.json", "invalid-scenario", "1")
	if err == nil {
		t.Fatalf("expected error for unknown scenario")
	}
}

func TestNewMetricsCollectionV3_ValidatesPositiveSequence(t *testing.T) {
	_, err := NewMetricsCollectionV3("/tmp/metrics.json", "controlled-warm", "0")
	if err == nil {
		t.Fatalf("expected error for sequence 0")
	}
}

func TestNewMetricsCollectionV3_CreatesValidCollection(t *testing.T) {
	mc, err := NewMetricsCollectionV3("/tmp/metrics.json", "controlled-warm", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mc == nil {
		t.Fatalf("expected non-nil collection")
	}
	if mc.Path != "/tmp/metrics.json" {
		t.Errorf("Path = %q, want %q", mc.Path, "/tmp/metrics.json")
	}
	if mc.Scenario != "controlled-warm" {
		t.Errorf("Scenario = %q, want %q", mc.Scenario, "controlled-warm")
	}
	if mc.Sequence != 1 {
		t.Errorf("Sequence = %d, want %d", mc.Sequence, 1)
	}
}

func TestValidateSubjectIdentity_RejectsNil(t *testing.T) {
	err := ValidateSubjectIdentity(nil)
	if err == nil {
		t.Fatalf("expected error for nil identity")
	}
}

func TestValidateSubjectIdentity_RejectsEmptyHeadOID(t *testing.T) {
	id := &SubjectIdentity{
		HeadOID:            "",
		TreeOID:            "abc123",
		WorktreeState:      "clean",
		SubjectInputDigest: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	err := ValidateSubjectIdentity(id)
	if err == nil {
		t.Fatalf("expected error for empty head OID")
	}
}

func TestValidateSubjectIdentity_RejectsEmptyDigest(t *testing.T) {
	id := &SubjectIdentity{
		HeadOID:            "abc123def456",
		TreeOID:            "abc123",
		WorktreeState:      "clean",
		SubjectInputDigest: "",
	}
	err := ValidateSubjectIdentity(id)
	if err == nil {
		t.Fatalf("expected error for empty digest")
	}
}

func TestValidateSubjectIdentity_RejectsShortDigest(t *testing.T) {
	id := &SubjectIdentity{
		HeadOID:            "abc123def456",
		TreeOID:            "abc123",
		WorktreeState:      "clean",
		SubjectInputDigest: "abc123",
	}
	err := ValidateSubjectIdentity(id)
	if err == nil {
		t.Fatalf("expected error for short digest")
	}
}

func TestValidateSubjectIdentity_AcceptsValidIdentity(t *testing.T) {
	id := &SubjectIdentity{
		HeadOID:            "abc123def456",
		TreeOID:            "abc123",
		WorktreeState:      "clean",
		SubjectInputDigest: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	err := ValidateSubjectIdentity(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMetricsCollectionV3_ValidateReconciliation_RequiresChecks(t *testing.T) {
	mc := &MetricsCollectionV3{}
	err := mc.validateReconciliation()
	if err == nil {
		t.Fatalf("expected error for no checks")
	}
}

func TestMetricsCollectionV3_ValidateReconciliation_RejectsDuplicateOrdinals(t *testing.T) {
	mc := &MetricsCollectionV3{
		Checks: []MetricsCheckV3{
			{Ordinal: 1, ID: "alpha", Status: "pass"},
			{Ordinal: 1, ID: "beta", Status: "pass"}, // duplicate ordinal
		},
	}
	err := mc.validateReconciliation()
	if err == nil {
		t.Fatalf("expected error for duplicate ordinals")
	}
}

func TestMetricsCollectionV3_ValidateReconciliation_RejectsMissingOrdinals(t *testing.T) {
	mc := &MetricsCollectionV3{
		Checks: []MetricsCheckV3{
			{Ordinal: 1, ID: "alpha", Status: "pass"},
			{Ordinal: 3, ID: "gamma", Status: "pass"}, // missing ordinal 2
		},
	}
	err := mc.validateReconciliation()
	if err == nil {
		t.Fatalf("expected error for missing ordinals")
	}
}

func TestMetricsCollectionV3_ValidateReconciliation_AcceptsValidChecks(t *testing.T) {
	mc := &MetricsCollectionV3{
		Checks: []MetricsCheckV3{
			{Ordinal: 1, ID: "alpha", Status: "pass"},
			{Ordinal: 2, ID: "beta", Status: "pass"},
		},
	}
	err := mc.validateReconciliation()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAddCheckWithResources_RejectsNegativeCPU(t *testing.T) {
	mc := &MetricsCollectionV3{}
	v := testVerifier("test", func(string) []checks.Finding { return nil })

	err := mc.AddCheckWithResources(
		v,
		1,
		nil,
		100*time.Millisecond,
		-1*time.Nanosecond, // negative
		0,
		0,
		".",
		nil,
	)
	if err == nil {
		t.Fatalf("expected error for negative CPU")
	}
}

func TestPlatformSampler_Sample(t *testing.T) {
	sampler := NewPlatformSampler()
	snap, err := sampler.Sample()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// On Linux, Maxrss should be > 0 for a running process
	if snap.ProcessMaxRSSKB <= 0 {
		t.Logf("note: ProcessMaxRSSKB = %d (may be 0 in container)", snap.ProcessMaxRSSKB)
	}
}

func TestFinalize_RejectsEmptyChecks(t *testing.T) {
	mc := &MetricsCollectionV3{}
	err := mc.Finalize(false)
	if err == nil {
		t.Fatalf("expected error for empty checks")
	}
}

func TestFinalize_AcceptsValidCollection(t *testing.T) {
	mc := &MetricsCollectionV3{
		Path:               "/tmp/test-metrics.json",
		Scenario:           "controlled-warm",
		Sequence:           1,
		HeadOID:            "abc123",
		TreeOID:            "def456",
		WorktreeState:      "content-bound",
		SubjectInputDigest: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		RunID:              "test:controlled-warm:1",
		Host: HostIdentity{
			GoVersion: "go1.21",
			GOOS:      "linux",
			GOARCH:    "amd64",
		},
		Checks: []MetricsCheckV3{
			{Ordinal: 1, ID: "alpha", Status: "pass"},
		},
	}

	// Create a temp directory for the test
	tmpDir, err := os.MkdirTemp("", "metrics-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	mc.Path = tmpDir + "/metrics.json"

	err = mc.Finalize(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReconciliation_RejectsMissingExpectedVerifier(t *testing.T) {
	mc := &MetricsCollectionV3{
		ExpectedVerifierIDs: []string{"alpha", "beta", "gamma"},
		Checks: []MetricsCheckV3{
			{Ordinal: 1, ID: "alpha", Status: "pass"},
			{Ordinal: 2, ID: "beta", Status: "pass"},
			// gamma is missing
		},
	}
	err := mc.validateReconciliation()
	if err == nil {
		t.Fatalf("expected error for missing expected verifier")
	}
}

func TestValidateReconciliation_RejectsUnexpectedVerifier(t *testing.T) {
	mc := &MetricsCollectionV3{
		ExpectedVerifierIDs: []string{"alpha", "beta"},
		Checks: []MetricsCheckV3{
			{Ordinal: 1, ID: "alpha", Status: "pass"},
			{Ordinal: 2, ID: "gamma", Status: "pass"}, // gamma unexpected
		},
	}
	err := mc.validateReconciliation()
	if err == nil {
		t.Fatalf("expected error for unexpected verifier")
	}
}

func TestValidateReconciliation_AcceptsMatchingExpectedAndRecorded(t *testing.T) {
	mc := &MetricsCollectionV3{
		ExpectedVerifierIDs: []string{"alpha", "beta"},
		Checks: []MetricsCheckV3{
			{Ordinal: 1, ID: "alpha", Status: "pass"},
			{Ordinal: 2, ID: "beta", Status: "pass"},
		},
	}
	err := mc.validateReconciliation()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContentBoundDigest_DifferentContentProducesDifferentDigest(t *testing.T) {
	head := "abc123"
	tree := "def456"

	d1 := ComputeSubjectDigestForTest(head, tree, map[string]string{
		"file.txt": "content-a",
	})
	d2 := ComputeSubjectDigestForTest(head, tree, map[string]string{
		"file.txt": "content-b", // different content
	})

	if d1 == d2 {
		t.Fatalf("different content should produce different digest")
	}
}

func TestContentBoundDigest_DifferentPathProducesDifferentDigest(t *testing.T) {
	head := "abc123"
	tree := "def456"

	d1 := ComputeSubjectDigestForTest(head, tree, map[string]string{
		"file-a.txt": "same-content",
	})
	d2 := ComputeSubjectDigestForTest(head, tree, map[string]string{
		"file-b.txt": "same-content", // different path
	})

	if d1 == d2 {
		t.Fatalf("different paths should produce different digest")
	}
}

func TestContentBoundDigest_SameContentSameDigest(t *testing.T) {
	head := "abc123"
	tree := "def456"

	d1 := ComputeSubjectDigestForTest(head, tree, map[string]string{
		"file.txt": "content",
	})
	d2 := ComputeSubjectDigestForTest(head, tree, map[string]string{
		"file.txt": "content",
	})

	if d1 != d2 {
		t.Fatalf("same content should produce same digest")
	}
}

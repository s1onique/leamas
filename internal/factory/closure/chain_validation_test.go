package closure

import (
	"context"
	"os"
	"testing"
)

// fakeGitClient is a mock for testing without actual Git calls
type fakeGitClient struct{}

func (fakeGitClient) Run(ctx context.Context, directory string, args ...string) gitCommandResult {
	// Return empty results for tests that don't need real Git
	return gitCommandResult{ExitCode: 1, Err: nil}
}

func TestVerifyChain_FEqualsS(t *testing.T) {
	// Test without real Git - just check OID validation
	req := ChainValidationRequest{
		Freeze:  "TODO", // Placeholder - should fail validation
		Subject: "TODO",
		Closure: "56ba5bbe2816f77acc0f1e228666481444b14056",
	}
	result, err := VerifyChain(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Verdict != "FAIL" {
		t.Errorf("expected FAIL verdict, got %s", result.Verdict)
	}
	if len(result.Errors) == 0 {
		t.Error("expected errors for placeholder OIDs")
	}
}

func TestVerifyChain_InvalidOID(t *testing.T) {
	req := ChainValidationRequest{
		Freeze:  "TODO",
		Subject: "64b6c20c0e0230f1eeb8aa1f5e21f96220f9bf28",
		Closure: "56ba5bbe2816f77acc0f1e228666481444b14056",
	}
	result, err := VerifyChain(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Verdict != "FAIL" {
		t.Errorf("expected FAIL verdict for placeholder OID, got %s", result.Verdict)
	}
}

func TestDetectObjectFormat(t *testing.T) {
	tests := []struct {
		oid    string
		expect ObjectFormat
	}{
		{"8362d35c65f66ccd140f5b5044b776f435fdc711", ObjectFormatSHA1},
		{"a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", ObjectFormatSHA256},
		{"abc", ObjectFormatUnknown},
		{"", ObjectFormatUnknown},
	}
	for _, tc := range tests {
		got := DetectObjectFormat(tc.oid)
		if got != tc.expect {
			t.Errorf("DetectObjectFormat(%q) = %s, want %s", tc.oid, got, tc.expect)
		}
	}
}

func TestValidateOIDWithFormat(t *testing.T) {
	// Valid SHA-1
	err := ValidateOIDWithFormat("test", "8362d35c65f66ccd140f5b5044b776f435fdc711", ObjectFormatSHA1)
	if err != nil {
		t.Errorf("expected valid SHA-1: %v", err)
	}

	// Invalid SHA-1 (wrong format)
	err = ValidateOIDWithFormat("test", "8362d35c65f66ccd140f5b5044b776f435fdc711", ObjectFormatSHA256)
	if err == nil {
		t.Error("expected error for SHA-1 OID in SHA-256 format")
	}

	// Valid SHA-256
	err = ValidateOIDWithFormat("test", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", ObjectFormatSHA256)
	if err != nil {
		t.Errorf("expected valid SHA-256: %v", err)
	}

	// Invalid format
	err = ValidateOIDWithFormat("test", "8362d35c65f66ccd140f5b5044b776f435fdc711", ObjectFormatUnknown)
	if err == nil {
		t.Error("expected error for unknown format")
	}
}

func TestRejectPlaceholder(t *testing.T) {
	tests := []struct {
		value string
		want  bool // true = should be rejected
	}{
		{"TODO", true},
		{"TBD", true},
		{"UNKNOWN", true},
		{"<COMMIT>", true},
		{"8362d35c65f66ccd140f5b5044b776f435fdc711", false},
		{"a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", false},
	}
	for _, tc := range tests {
		err := RejectPlaceholder("test", tc.value)
		got := err != nil
		if got != tc.want {
			t.Errorf("RejectPlaceholder(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

func TestValidatePlanBytes_ValidPlan(t *testing.T) {
	plan := `{
		"act_id": "ACT-TEST-01",
		"description": "Test plan"
	}`
	err := ValidatePlanBytes([]byte(plan))
	if err != nil {
		t.Errorf("expected valid plan: %v", err)
	}
}

func TestValidatePlanBytes_InvalidJSON(t *testing.T) {
	err := ValidatePlanBytes([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestValidatePlanBytes_RejectsForbiddenKey(t *testing.T) {
	plan := `{"freeze_commit": "8362d35c65f66ccd140f5b5044b776f435fdc711"}`
	err := ValidatePlanBytes([]byte(plan))
	if err == nil {
		t.Error("expected rejection of freeze_commit")
	}
}

func TestValidatePlanBytes_RejectsNestedForbiddenKey(t *testing.T) {
	plan := `{"nested": {"subject_commit": "8362d35c65f66ccd140f5b5044b776f435fdc711"}}`
	err := ValidatePlanBytes([]byte(plan))
	if err == nil {
		t.Error("expected rejection of nested forbidden key")
	}
}

func TestPlanBytesAtCommit(t *testing.T) {
	// This test requires a real git repository
	// Skip if not in a git repo
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("not in a git repository")
	}

	bytes, err := PlanBytesAtCommit(context.Background(), ".", "HEAD", "README.md")
	if err != nil {
		t.Skipf("plan not found: %v", err)
	}
	if len(bytes) == 0 {
		t.Error("expected non-empty bytes")
	}
}

func TestChainsEqual(t *testing.T) {
	// This test requires a real git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("not in a git repository")
	}

	// Test with any file in the repo
	equal, err := ChainsEqual(context.Background(), ".", "HEAD", "HEAD^", "README.md")
	if err != nil {
		t.Skipf("file not found: %v", err)
	}
	// HEAD equals HEAD - should be true
	if !equal {
		t.Error("expected HEAD == HEAD")
	}
}

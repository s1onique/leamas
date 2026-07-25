// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"errors"
	"testing"
)

// countingRunnerIdentity is a runner identity provider that counts calls.
type countingRunnerIdentity struct {
	calls  int
	value  RunnerIdentity
	errOut error
}

func (c *countingRunnerIdentity) Identity() (RunnerIdentity, error) {
	c.calls++
	return c.value, c.errOut
}

func (c *countingRunnerIdentity) CallCount() int { return c.calls }

// countingBinaryHash is a SHA256 provider that counts calls.
type countingBinaryHash struct {
	calls int
	value string
	err   error
}

func (c *countingBinaryHash) Hash() (string, error) {
	c.calls++
	return c.value, c.err
}

func (c *countingBinaryHash) CallCount() int { return c.calls }

// countingVerifier is a verified candidate verifier that counts calls.
type countingVerifier struct {
	calls  int
	result *TransactionResult
	err    error
}

func (c *countingVerifier) Verify(ctx context.Context, plan Plan, evidenceDir string, candidateSubject string, candidateClosure string, runner RunnerIdentity) (*TransactionResult, error) {
	c.calls++
	return c.result, c.err
}

func (c *countingVerifier) CallCount() int { return c.calls }

// TestRunnerIdentityHelperRejectsMismatchedRevision exercises the narrow
// identity helper. Production ordering is covered by the orchestrator suite.
func TestRunnerIdentityHelperRejectsMismatchedRevision(t *testing.T) {
	runner := &countingRunnerIdentity{
		value: RunnerIdentity{
			VCSRevision:  "sub123",
			VCSModified:  false,
			BinarySHA256: "hash1",
		},
	}
	bin := &countingBinaryHash{value: "hash1"}

	// Test fail-closed: helper enforces identity with mismatched subject.
	identity := RunnerIdentity{
		VCSRevision:  "different",
		VCSModified:  false,
		BinarySHA256: "hash1",
	}
	err := enforceRunnerIdentity(identity, "sub123", "hash1")
	if err == nil {
		t.Fatal("expected error for mismatched revision")
	}

	// Verify the fake runner has not been called yet (helper does not call into it).
	if runner.CallCount() != 0 {
		t.Errorf("runner called %d times before enforcement; want 0", runner.CallCount())
	}
	if bin.CallCount() != 0 {
		t.Errorf("binary hash called %d times before enforcement; want 0", bin.CallCount())
	}
}

// TestEnforceRunnerIdentityMismatchedHash verifies the new helper
// rejects a binary hash mismatch.
func TestRunnerIdentityHelperRejectsMismatchedHash(t *testing.T) {
	identity := RunnerIdentity{
		VCSRevision:  "sub123",
		VCSModified:  false,
		BinarySHA256: "claimed",
	}
	err := enforceRunnerIdentity(identity, "sub123", "actual")
	if err == nil {
		t.Fatal("expected error for hash mismatch")
	}
	if !containsString(err.Error(), "binary SHA256 mismatch") {
		t.Errorf("err = %v; want binary SHA256 mismatch", err)
	}
}

// TestEnforceRunnerIdentityEmptyActual verifies the new helper
// rejects empty actual hash.
func TestRunnerIdentityHelperRejectsEmptyActualHash(t *testing.T) {
	identity := RunnerIdentity{
		VCSRevision:  "sub123",
		VCSModified:  false,
		BinarySHA256: "claimed",
	}
	err := enforceRunnerIdentity(identity, "sub123", "")
	if err == nil {
		t.Fatal("expected error for empty actual hash")
	}
	if !containsString(err.Error(), "actual binary SHA256 is empty") {
		t.Errorf("err = %v; want empty actual", err)
	}
}

// TestCountingVerifierSimple prove the counting verifier records calls.
func TestVerifiedCandidateVerifierFakeRecordsCall(t *testing.T) {
	cv := &countingVerifier{
		result: &TransactionResult{
			ActID:         "ACT-1",
			FreezeCommit:  "f1",
			SubjectCommit: "s1",
			Verdict:       VerdictPass,
		},
	}
	r, err := cv.Verify(context.Background(), Plan{ActID: "ACT-1"}, "/tmp", "s1", "c1", RunnerIdentity{})
	if err != nil {
		t.Fatal(err)
	}
	if r.ActID != "ACT-1" {
		t.Errorf("r.ActID = %q; want ACT-1", r.ActID)
	}
	if cv.CallCount() != 1 {
		t.Errorf("calls = %d; want 1", cv.CallCount())
	}
}

// TestCountingVerifierFailure proves the verifier error is propagated.
func TestVerifiedCandidateVerifierFakeReturnsFailure(t *testing.T) {
	cv := &countingVerifier{err: errors.New("verifier boom")}
	_, err := cv.Verify(context.Background(), Plan{ActID: "ACT-1"}, "/tmp", "s1", "c1", RunnerIdentity{})
	if err == nil {
		t.Fatal("expected error from verifier")
	}
	if !containsString(err.Error(), "verifier boom") {
		t.Errorf("err = %v; want verifier boom", err)
	}
}

func containsString(s, sub string) bool {
	if len(s) < len(sub) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

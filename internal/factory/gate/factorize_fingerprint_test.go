// Package gate provides tests for factorize execution fingerprint v3.
package gate

import (
	"encoding/hex"
	"testing"
)

// TestExecutionFingerprintV3_ReturnsErrorForEmptyName verifies fail-closed.
func TestExecutionFingerprintV3_ReturnsErrorForEmptyName(t *testing.T) {
	exec := ExecutionDefinition{
		Kind:             ExecutionInProcess,
		ImplementationID: "internal/factory/test.CheckRepo",
		EnvVars:          []string{"GOFLAGS"},
	}
	_, err := executionFingerprintV3("", exec, nil)
	if err == nil {
		t.Fatalf("empty name must return error")
	}
}

// TestExecutionFingerprintV3_ReturnsErrorForEmptyImplID verifies fail-closed.
func TestExecutionFingerprintV3_ReturnsErrorForEmptyImplID(t *testing.T) {
	exec := ExecutionDefinition{
		Kind:             ExecutionInProcess,
		ImplementationID: "",
		EnvVars:          []string{"GOFLAGS"},
	}
	_, err := executionFingerprintV3("verifier", exec, nil)
	if err == nil {
		t.Fatalf("empty implementation ID must return error")
	}
}

// TestExecutionFingerprintV3_IdenticalExecProduceIdenticalFingerprints verifies.
func TestExecutionFingerprintV3_IdenticalExecProduceIdenticalFingerprints(t *testing.T) {
	exec := ExecutionDefinition{
		Kind:             ExecutionInProcess,
		ImplementationID: "internal/factory/test.CheckRepo",
		EnvVars:          []string{"GOFLAGS"},
	}

	fp1, err1 := executionFingerprintV3("test-verifier", exec, nil)
	fp2, err2 := executionFingerprintV3("test-verifier", exec, nil)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 != fp2 {
		t.Fatalf("identical execution definitions must produce identical fingerprints")
	}
}

// TestExecutionFingerprintV3_DifferentVerifierNamesAlterFingerprint verifies.
func TestExecutionFingerprintV3_DifferentVerifierNamesAlterFingerprint(t *testing.T) {
	exec := ExecutionDefinition{
		Kind:             ExecutionInProcess,
		ImplementationID: "internal/factory/test.CheckRepo",
		EnvVars:          []string{"GOFLAGS"},
	}

	fp1, err1 := executionFingerprintV3("verifier-alpha", exec, nil)
	fp2, err2 := executionFingerprintV3("verifier-beta", exec, nil)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 == fp2 {
		t.Fatalf("different verifier names must produce different fingerprints")
	}
}

// TestExecutionFingerprintV3_ImplIDChangesAlterFingerprint verifies.
func TestExecutionFingerprintV3_ImplIDChangesAlterFingerprint(t *testing.T) {
	exec1 := ExecutionDefinition{
		Kind:             ExecutionInProcess,
		ImplementationID: "internal/factory/test.CheckRepo",
		EnvVars:          []string{"GOFLAGS"},
	}
	exec2 := ExecutionDefinition{
		Kind:             ExecutionInProcess,
		ImplementationID: "internal/factory/test.CheckAlt",
		EnvVars:          []string{"GOFLAGS"},
	}

	fp1, err1 := executionFingerprintV3("test-verifier", exec1, nil)
	fp2, err2 := executionFingerprintV3("test-verifier", exec2, nil)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 == fp2 {
		t.Fatalf("implementation ID changes must alter the fingerprint")
	}
}

// TestExecutionFingerprintV3_GoEnvChangesAlterFingerprint verifies.
func TestExecutionFingerprintV3_GoEnvChangesAlterFingerprint(t *testing.T) {
	exec := ExecutionDefinition{
		Kind:             ExecutionInProcess,
		ImplementationID: "internal/factory/test.CheckRepo",
		EnvVars:          []string{"GOFLAGS"},
	}

	env1 := []string{"GOFLAGS=-v"}
	env2 := []string{"GOFLAGS=-vv"}

	fp1, err1 := executionFingerprintV3("test-verifier", exec, env1)
	fp2, err2 := executionFingerprintV3("test-verifier", exec, env2)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 == fp2 {
		t.Fatalf("GOFLAGS changes must alter the fingerprint")
	}
}

// TestExecutionFingerprintV3_EnvOrderingDoesNotMatter verifies.
func TestExecutionFingerprintV3_EnvOrderingDoesNotMatter(t *testing.T) {
	exec := ExecutionDefinition{
		Kind:             ExecutionInProcess,
		ImplementationID: "internal/factory/test.CheckRepo",
		EnvVars:          []string{"GOFLAGS", "GOCACHE"},
	}

	env1 := []string{"GOFLAGS=-v", "GOCACHE=/tmp/cache"}
	env2 := []string{"GOCACHE=/tmp/cache", "GOFLAGS=-v"}

	fp1, err1 := executionFingerprintV3("test-verifier", exec, env1)
	fp2, err2 := executionFingerprintV3("test-verifier", exec, env2)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 != fp2 {
		t.Fatalf("environment ordering must NOT affect the fingerprint")
	}
}

// TestExecutionFingerprintV3_FullDigestLength verifies SHA-256.
func TestExecutionFingerprintV3_FullDigestLength(t *testing.T) {
	exec := ExecutionDefinition{
		Kind:             ExecutionInProcess,
		ImplementationID: "internal/factory/test.CheckRepo",
		EnvVars:          []string{"GOFLAGS"},
	}

	fp, err := executionFingerprintV3("test-verifier", exec, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(fp) != 64 {
		t.Fatalf("fingerprint must use full SHA-256 digest (64 hex chars), got %d", len(fp))
	}

	_, err = hex.DecodeString(fp)
	if err != nil {
		t.Fatalf("fingerprint must be valid hex: %v", err)
	}
}

// TestExecutionFingerprintV3_KindIncluded verifies execution kind is in hash.
func TestExecutionFingerprintV3_KindIncluded(t *testing.T) {
	execInProcess := ExecutionDefinition{
		Kind:             ExecutionInProcess,
		ImplementationID: "internal/factory/test.CheckRepo",
		EnvVars:          []string{},
	}
	execChild := ExecutionDefinition{
		Kind:             ExecutionChild,
		ImplementationID: "internal/factory/test.CheckRepo",
		EnvVars:          []string{},
	}

	fp1, err1 := executionFingerprintV3("test-verifier", execInProcess, nil)
	fp2, err2 := executionFingerprintV3("test-verifier", execChild, nil)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 == fp2 {
		t.Fatalf("execution kind must alter the fingerprint")
	}
}

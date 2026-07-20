// Package gate provides tests for factorize execution fingerprint.
package gate

import (
	"encoding/hex"
	"testing"
)

// TestExecutionFingerprint_ReturnsErrorForEmptyName verifies fail-closed.
func TestExecutionFingerprint_ReturnsErrorForEmptyName(t *testing.T) {
	exec := ExecutionDefinition{
		Kind:        ExecutionInProcess,
		LogicalArgv: []string{"alpha"},
		EnvVars:     []string{"GOFLAGS"},
	}
	_, err := executionFingerprint("", exec, nil)
	if err == nil {
		t.Fatalf("empty name must return error")
	}
}

// TestExecutionFingerprint_ReturnsErrorForEmptyArgv verifies fail-closed.
func TestExecutionFingerprint_ReturnsErrorForEmptyArgv(t *testing.T) {
	exec := ExecutionDefinition{
		Kind:        ExecutionInProcess,
		LogicalArgv: []string{},
		EnvVars:     []string{"GOFLAGS"},
	}
	_, err := executionFingerprint("verifier", exec, nil)
	if err == nil {
		t.Fatalf("empty argv must return error")
	}
}

// TestExecutionFingerprint_IdenticalExecProduceIdenticalFingerprints verifies.
func TestExecutionFingerprint_IdenticalExecProduceIdenticalFingerprints(t *testing.T) {
	exec := ExecutionDefinition{
		Kind:        ExecutionInProcess,
		LogicalArgv: []string{"alpha", "--verbose"},
		EnvVars:     []string{"GOFLAGS"},
	}

	fp1, err1 := executionFingerprint("test-verifier", exec, nil)
	fp2, err2 := executionFingerprint("test-verifier", exec, nil)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 != fp2 {
		t.Fatalf("identical execution definitions must produce identical fingerprints")
	}
}

// TestExecutionFingerprint_DifferentVerifierNamesAlterFingerprint verifies.
func TestExecutionFingerprint_DifferentVerifierNamesAlterFingerprint(t *testing.T) {
	exec := ExecutionDefinition{
		Kind:        ExecutionInProcess,
		LogicalArgv: []string{"alpha"},
		EnvVars:     []string{"GOFLAGS"},
	}

	fp1, err1 := executionFingerprint("verifier-alpha", exec, nil)
	fp2, err2 := executionFingerprint("verifier-beta", exec, nil)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 == fp2 {
		t.Fatalf("different verifier names must produce different fingerprints")
	}
}

// TestExecutionFingerprint_ArgvChangesAlterFingerprint verifies.
func TestExecutionFingerprint_ArgvChangesAlterFingerprint(t *testing.T) {
	exec1 := ExecutionDefinition{
		Kind:        ExecutionInProcess,
		LogicalArgv: []string{"alpha", "--verbose"},
		EnvVars:     []string{"GOFLAGS"},
	}
	exec2 := ExecutionDefinition{
		Kind:        ExecutionInProcess,
		LogicalArgv: []string{"alpha", "--debug"},
		EnvVars:     []string{"GOFLAGS"},
	}

	fp1, err1 := executionFingerprint("test-verifier", exec1, nil)
	fp2, err2 := executionFingerprint("test-verifier", exec2, nil)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 == fp2 {
		t.Fatalf("argv changes must alter the fingerprint")
	}
}

// TestExecutionFingerprint_GoEnvChangesAlterFingerprint verifies.
func TestExecutionFingerprint_GoEnvChangesAlterFingerprint(t *testing.T) {
	exec := ExecutionDefinition{
		Kind:        ExecutionInProcess,
		LogicalArgv: []string{"alpha"},
		EnvVars:     []string{"GOFLAGS"},
	}

	env1 := []string{"GOFLAGS=-v"}
	env2 := []string{"GOFLAGS=-vv"}

	fp1, err1 := executionFingerprint("test-verifier", exec, env1)
	fp2, err2 := executionFingerprint("test-verifier", exec, env2)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 == fp2 {
		t.Fatalf("GOFLAGS changes must alter the fingerprint")
	}
}

// TestExecutionFingerprint_EnvOrderingDoesNotMatter verifies.
func TestExecutionFingerprint_EnvOrderingDoesNotMatter(t *testing.T) {
	exec := ExecutionDefinition{
		Kind:        ExecutionInProcess,
		LogicalArgv: []string{"alpha"},
		EnvVars:     []string{"GOFLAGS", "GOCACHE"},
	}

	env1 := []string{"GOFLAGS=-v", "GOCACHE=/tmp/cache"}
	env2 := []string{"GOCACHE=/tmp/cache", "GOFLAGS=-v"}

	fp1, err1 := executionFingerprint("test-verifier", exec, env1)
	fp2, err2 := executionFingerprint("test-verifier", exec, env2)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 != fp2 {
		t.Fatalf("environment ordering must NOT affect the fingerprint")
	}
}

// TestExecutionFingerprint_FullDigestLength verifies SHA-256.
func TestExecutionFingerprint_FullDigestLength(t *testing.T) {
	exec := ExecutionDefinition{
		Kind:        ExecutionInProcess,
		LogicalArgv: []string{"alpha"},
		EnvVars:     []string{"GOFLAGS"},
	}

	fp, err := executionFingerprint("test-verifier", exec, nil)
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

// TestExecutionFingerprint_KindIncluded verifies execution kind is in hash.
func TestExecutionFingerprint_KindIncluded(t *testing.T) {
	execInProcess := ExecutionDefinition{
		Kind:        ExecutionInProcess,
		LogicalArgv: []string{"alpha"},
		EnvVars:     []string{},
	}
	execChild := ExecutionDefinition{
		Kind:        ExecutionChild,
		LogicalArgv: []string{"alpha"},
		EnvVars:     []string{},
	}

	fp1, err1 := executionFingerprint("test-verifier", execInProcess, nil)
	fp2, err2 := executionFingerprint("test-verifier", execChild, nil)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 == fp2 {
		t.Fatalf("execution kind must alter the fingerprint")
	}
}

// TestExecutionFingerprint_ChildProcessVerifier verifies child-process argv.
func TestExecutionFingerprint_ChildProcessVerifier(t *testing.T) {
	exec := ExecutionDefinition{
		Kind:        ExecutionChild,
		LogicalArgv: []string{"leamas", "factory", "verify", "dupcode"},
		EnvVars:     []string{"GOFLAGS", "CGO_ENABLED", "GOCACHE"},
	}

	fp, err := executionFingerprint("dupcode", exec, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(fp) != 64 {
		t.Fatalf("expected 64-char fingerprint, got %d", len(fp))
	}
}

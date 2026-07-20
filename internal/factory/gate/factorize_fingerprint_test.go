// Package gate provides tests for factorize command fingerprint.
package gate

import (
	"encoding/hex"
	"testing"
)

// TestCommandFingerprint_ReturnsErrorForEmptyName verifies fail-closed behavior.
func TestCommandFingerprint_ReturnsErrorForEmptyName(t *testing.T) {
	argv := []string{"alpha"}
	env := []string{"GOFLAGS=-v"}
	execPath := "/usr/local/bin/leamas"

	_, err := commandFingerprint("", "/checkout", argv, env, execPath)
	if err == nil {
		t.Fatalf("empty name must return error")
	}
}

// TestCommandFingerprint_ReturnsErrorForEmptyArgv verifies fail-closed behavior.
func TestCommandFingerprint_ReturnsErrorForEmptyArgv(t *testing.T) {
	env := []string{"GOFLAGS=-v"}
	execPath := "/usr/local/bin/leamas"

	_, err := commandFingerprint("verifier", "/checkout", nil, env, execPath)
	if err == nil {
		t.Fatalf("empty argv must return error")
	}
}

// TestCommandFingerprint_IdenticalExecProduceIdenticalFingerprints verifies
// identical execution definitions produce identical fingerprints.
func TestCommandFingerprint_IdenticalExecProduceIdenticalFingerprints(t *testing.T) {
	argv := []string{"alpha", "--verbose"}
	env := []string{"GOFLAGS=-v"}
	execPath := "/usr/local/bin/leamas"

	fp1, err1 := commandFingerprint("test-verifier", "/checkout", argv, env, execPath)
	fp2, err2 := commandFingerprint("test-verifier", "/checkout", argv, env, execPath)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 != fp2 {
		t.Fatalf("identical execution definitions must produce identical fingerprints")
	}
}

// TestCommandFingerprint_DifferentVerifierNamesAlterFingerprint verifies
// different verifier names produce different fingerprints.
func TestCommandFingerprint_DifferentVerifierNamesAlterFingerprint(t *testing.T) {
	argv := []string{"alpha", "--verbose"}
	env := []string{"GOFLAGS=-v"}
	execPath := "/usr/local/bin/leamas"

	fp1, err1 := commandFingerprint("verifier-alpha", "/checkout", argv, env, execPath)
	fp2, err2 := commandFingerprint("verifier-beta", "/checkout", argv, env, execPath)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 == fp2 {
		t.Fatalf("different verifier names must produce different fingerprints")
	}
}

// TestCommandFingerprint_ArgvChangesAlterFingerprint verifies changes to argv.
func TestCommandFingerprint_ArgvChangesAlterFingerprint(t *testing.T) {
	env := []string{"GOFLAGS=-v"}
	execPath := "/usr/local/bin/leamas"

	argv1 := []string{"alpha", "--verbose"}
	argv2 := []string{"alpha", "--debug"}

	fp1, err1 := commandFingerprint("test-verifier", "/checkout", argv1, env, execPath)
	fp2, err2 := commandFingerprint("test-verifier", "/checkout", argv2, env, execPath)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 == fp2 {
		t.Fatalf("argv changes must alter the fingerprint")
	}
}

// TestCommandFingerprint_GoEnvChangesAlterFingerprint verifies Go env affects.
func TestCommandFingerprint_GoEnvChangesAlterFingerprint(t *testing.T) {
	argv := []string{"alpha"}
	execPath := "/usr/local/bin/leamas"

	env1 := []string{"GOFLAGS=-v"}
	env2 := []string{"GOFLAGS=-vv"}

	fp1, err1 := commandFingerprint("test-verifier", "/checkout", argv, env1, execPath)
	fp2, err2 := commandFingerprint("test-verifier", "/checkout", argv, env2, execPath)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 == fp2 {
		t.Fatalf("GOFLAGS changes must alter the fingerprint")
	}
}

// TestCommandFingerprint_ScenarioDoesNotAlterFingerprint verifies LEAMAS_*
// observation metadata is excluded.
func TestCommandFingerprint_ScenarioDoesNotAlterFingerprint(t *testing.T) {
	argv := []string{"alpha"}
	execPath := "/usr/local/bin/leamas"

	env1 := []string{"LEAMAS_FACTORIZE_SCENARIO=controlled-warm"}
	env2 := []string{"LEAMAS_FACTORIZE_SCENARIO=controlled-cold"}

	fp1, err1 := commandFingerprint("test-verifier", "/checkout", argv, env1, execPath)
	fp2, err2 := commandFingerprint("test-verifier", "/checkout", argv, env2, execPath)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 != fp2 {
		t.Fatalf("scenario must NOT alter the fingerprint")
	}
}

// TestCommandFingerprint_SequenceDoesNotAlterFingerprint verifies LEAMAS_*.
func TestCommandFingerprint_SequenceDoesNotAlterFingerprint(t *testing.T) {
	argv := []string{"alpha"}
	execPath := "/usr/local/bin/leamas"

	env1 := []string{"LEAMAS_FACTORIZE_SEQUENCE=1"}
	env2 := []string{"LEAMAS_FACTORIZE_SEQUENCE=3"}

	fp1, err1 := commandFingerprint("test-verifier", "/checkout", argv, env1, execPath)
	fp2, err2 := commandFingerprint("test-verifier", "/checkout", argv, env2, execPath)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 != fp2 {
		t.Fatalf("sequence must NOT alter the fingerprint")
	}
}

// TestCommandFingerprint_MetricsFileDoesNotAlterFingerprint verifies evidence.
func TestCommandFingerprint_MetricsFileDoesNotAlterFingerprint(t *testing.T) {
	argv := []string{"alpha"}
	execPath := "/usr/local/bin/leamas"

	env1 := []string{"LEAMAS_FACTORIZE_METRICS_FILE=/tmp/m1.json"}
	env2 := []string{"LEAMAS_FACTORIZE_METRICS_FILE=/tmp/m2.json"}

	fp1, err1 := commandFingerprint("test-verifier", "/checkout", argv, env1, execPath)
	fp2, err2 := commandFingerprint("test-verifier", "/checkout", argv, env2, execPath)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 != fp2 {
		t.Fatalf("metrics file path must NOT alter the fingerprint")
	}
}

// TestCommandFingerprint_NonGoEnvDoesNotAlterFingerprint verifies non-GO vars.
func TestCommandFingerprint_NonGoEnvDoesNotAlterFingerprint(t *testing.T) {
	argv := []string{"alpha"}
	execPath := "/usr/local/bin/leamas"

	env1 := []string{"HOME=/root"}
	env2 := []string{"HOME=/home/user"}

	fp1, err1 := commandFingerprint("test-verifier", "/checkout", argv, env1, execPath)
	fp2, err2 := commandFingerprint("test-verifier", "/checkout", argv, env2, execPath)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 != fp2 {
		t.Fatalf("non-execution-relevant env must NOT alter the fingerprint")
	}
}

// TestCommandFingerprint_EnvOrderingDoesNotMatter verifies deterministic order.
func TestCommandFingerprint_EnvOrderingDoesNotMatter(t *testing.T) {
	argv := []string{"alpha"}
	execPath := "/usr/local/bin/leamas"

	env1 := []string{"GOFLAGS=-v", "GOCACHE=/tmp/cache"}
	env2 := []string{"GOCACHE=/tmp/cache", "GOFLAGS=-v"}

	fp1, err1 := commandFingerprint("test-verifier", "/checkout", argv, env1, execPath)
	fp2, err2 := commandFingerprint("test-verifier", "/checkout", argv, env2, execPath)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 != fp2 {
		t.Fatalf("environment ordering must NOT affect the fingerprint")
	}
}

// TestCommandFingerprint_FullDigestLength verifies SHA-256 is used.
func TestCommandFingerprint_FullDigestLength(t *testing.T) {
	argv := []string{"alpha"}
	env := []string{"GOFLAGS=-v"}
	execPath := "/usr/local/bin/leamas"

	fp, err := commandFingerprint("test-verifier", "/checkout", argv, env, execPath)
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

// TestCommandFingerprint_RelocationInvariant verifies checkout reloc doesn't.
func TestCommandFingerprint_RelocationInvariant(t *testing.T) {
	argv := []string{"alpha", "--verbose"}
	env := []string{"GOFLAGS=-v"}
	execPath := "/usr/local/bin/leamas"

	fp1, err1 := commandFingerprint("test-verifier", "/checkout/original", argv, env, execPath)
	fp2, err2 := commandFingerprint("test-verifier", "/checkout/relocated", argv, env, execPath)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	if fp1 != fp2 {
		t.Fatalf("relocation must NOT change fingerprint")
	}
}

// TestCommandFingerprint_ExecPathIgnored verifies exec path is not included in hash.
// Per the review: "An absolute path may be diagnostic metadata, but should not
// be the primary semantic identity."
func TestCommandFingerprint_ExecPathIgnored(t *testing.T) {
	argv := []string{"alpha"}

	fp1, err1 := commandFingerprint("test-verifier", "/checkout", argv, nil, "/bin/leamas-v1")
	fp2, err2 := commandFingerprint("test-verifier", "/checkout", argv, nil, "/bin/leamas-v2")
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v or %v", err1, err2)
	}
	// execPath is diagnostic metadata, not semantic identity
	if fp1 != fp2 {
		t.Fatalf("exec path must NOT alter the fingerprint (path is diagnostic metadata)")
	}
}

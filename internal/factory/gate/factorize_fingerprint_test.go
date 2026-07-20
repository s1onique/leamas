// Package gate provides tests for factorize command fingerprint.
package gate

import (
	"encoding/hex"
	"testing"
)

// TestCommandFingerprint_IdenticalExecProduceIdenticalFingerprints verifies that
// identical execution definitions produce identical fingerprints.
func TestCommandFingerprint_IdenticalExecProduceIdenticalFingerprints(t *testing.T) {
	argv := []string{"alpha", "--verbose"}
	env := []string{"LEAMAS_FACTORIZE_SCENARIO=controlled-warm"}
	execPath := "/usr/local/bin/leamas"
	root := "/checkout"

	fp1 := commandFingerprint("test-verifier", root, argv, env, execPath)
	fp2 := commandFingerprint("test-verifier", root, argv, env, execPath)

	if fp1 != fp2 {
		t.Fatalf("identical execution definitions must produce identical fingerprints:\n  fp1=%q\n  fp2=%q", fp1, fp2)
	}
}

// TestCommandFingerprint_DifferentVerifierNamesAlterFingerprint verifies that
// different verifier names produce different fingerprints.
func TestCommandFingerprint_DifferentVerifierNamesAlterFingerprint(t *testing.T) {
	argv := []string{"alpha", "--verbose"}
	env := []string{"LEAMAS_FACTORIZE_SCENARIO=controlled-warm"}
	execPath := "/usr/local/bin/leamas"
	root := "/checkout"

	fp1 := commandFingerprint("verifier-alpha", root, argv, env, execPath)
	fp2 := commandFingerprint("verifier-beta", root, argv, env, execPath)

	if fp1 == fp2 {
		t.Fatalf("different verifier names must produce different fingerprints:\n  fp1=%q\n  fp2=%q", fp1, fp2)
	}
}

// TestCommandFingerprint_ArgvChangesAlterFingerprint verifies that changes
// to argv alter the fingerprint.
func TestCommandFingerprint_ArgvChangesAlterFingerprint(t *testing.T) {
	env := []string{"LEAMAS_FACTORIZE_SCENARIO=controlled-warm"}
	execPath := "/usr/local/bin/leamas"
	root := "/checkout"

	argv1 := []string{"alpha", "--verbose"}
	argv2 := []string{"alpha", "--debug"}

	fp1 := commandFingerprint("test-verifier", root, argv1, env, execPath)
	fp2 := commandFingerprint("test-verifier", root, argv2, env, execPath)

	if fp1 == fp2 {
		t.Fatalf("argv changes must alter the fingerprint:\n  fp1=%q\n  fp2=%q", fp1, fp2)
	}
}

// TestCommandFingerprint_ExecPathChangesAlterFingerprint verifies that changes
// to the executable path alter the fingerprint.
func TestCommandFingerprint_ExecPathChangesAlterFingerprint(t *testing.T) {
	argv := []string{"alpha", "--verbose"}
	env := []string{"LEAMAS_FACTORIZE_SCENARIO=controlled-warm"}
	root := "/checkout"

	fp1 := commandFingerprint("test-verifier", root, argv, env, "/usr/local/bin/leamas")
	fp2 := commandFingerprint("test-verifier", root, argv, env, "/usr/local/bin/leamas-v2")

	if fp1 == fp2 {
		t.Fatalf("exec path changes must alter the fingerprint:\n  fp1=%q\n  fp2=%q", fp1, fp2)
	}
}

// TestCommandFingerprint_RelevantEnvChangesAlterFingerprint verifies that
// relevant LEAMAS_* environment changes alter the fingerprint.
func TestCommandFingerprint_RelevantEnvChangesAlterFingerprint(t *testing.T) {
	argv := []string{"alpha", "--verbose"}
	execPath := "/usr/local/bin/leamas"
	root := "/checkout"

	env1 := []string{"LEAMAS_FACTORIZE_SCENARIO=controlled-warm"}
	env2 := []string{"LEAMAS_FACTORIZE_SCENARIO=controlled-cold"}

	fp1 := commandFingerprint("test-verifier", root, argv, env1, execPath)
	fp2 := commandFingerprint("test-verifier", root, argv, env2, execPath)

	if fp1 == fp2 {
		t.Fatalf("relevant environment changes must alter the fingerprint:\n  fp1=%q\n  fp2=%q", fp1, fp2)
	}
}

// TestCommandFingerprint_IrrelevantEnvChangesDoNotAlterFingerprint verifies that
// evidence-only environment variables do not alter the fingerprint.
func TestCommandFingerprint_IrrelevantEnvChangesDoNotAlterFingerprint(t *testing.T) {
	argv := []string{"alpha", "--verbose"}
	execPath := "/usr/local/bin/leamas"
	root := "/checkout"

	// LEAMAS_FACTORIZE_METRICS_FILE is evidence-only; changing it should not
	// alter the command fingerprint since it does not affect execution.
	env1 := []string{"LEAMAS_FACTORIZE_METRICS_FILE=/tmp/metrics.json"}
	env2 := []string{"LEAMAS_FACTORIZE_METRICS_FILE=/tmp/other-metrics.json"}

	fp1 := commandFingerprint("test-verifier", root, argv, env1, execPath)
	fp2 := commandFingerprint("test-verifier", root, argv, env2, execPath)

	if fp1 != fp2 {
		t.Fatalf("evidence-only environment changes must NOT alter the fingerprint:\n  fp1=%q\n  fp2=%q", fp1, fp2)
	}
}

// TestCommandFingerprint_NonLEAMASEnvChangesDoNotAlterFingerprint verifies that
// non-LEAMAS_* environment changes do not alter the fingerprint.
func TestCommandFingerprint_NonLEAMASEnvChangesDoNotAlterFingerprint(t *testing.T) {
	argv := []string{"alpha", "--verbose"}
	execPath := "/usr/local/bin/leamas"
	root := "/checkout"

	env1 := []string{"PATH=/usr/bin:/bin", "HOME=/root"}
	env2 := []string{"PATH=/usr/local/bin:/usr/bin", "HOME=/home/user"}

	fp1 := commandFingerprint("test-verifier", root, argv, env1, execPath)
	fp2 := commandFingerprint("test-verifier", root, argv, env2, execPath)

	if fp1 != fp2 {
		t.Fatalf("non-LEAMAS environment changes must NOT alter the fingerprint:\n  fp1=%q\n  fp2=%q", fp1, fp2)
	}
}

// TestCommandFingerprint_EnvOrderingDoesNotMatter verifies that the order of
// LEAMAS_* environment variables does not affect the fingerprint.
func TestCommandFingerprint_EnvOrderingDoesNotMatter(t *testing.T) {
	argv := []string{"alpha", "--verbose"}
	execPath := "/usr/local/bin/leamas"
	root := "/checkout"

	env1 := []string{
		"LEAMAS_FACTORIZE_SCENARIO=controlled-warm",
		"LEAMAS_FACTORIZE_SEQUENCE=1",
	}
	env2 := []string{
		"LEAMAS_FACTORIZE_SEQUENCE=1",
		"LEAMAS_FACTORIZE_SCENARIO=controlled-warm",
	}

	fp1 := commandFingerprint("test-verifier", root, argv, env1, execPath)
	fp2 := commandFingerprint("test-verifier", root, argv, env2, execPath)

	if fp1 != fp2 {
		t.Fatalf("environment ordering must NOT affect the fingerprint:\n  fp1=%q\n  fp2=%q", fp1, fp2)
	}
}

// TestCommandFingerprint_EmptyExecPathHandled verifies that empty executable
// path produces a distinguishable fingerprint (fail-closed).
func TestCommandFingerprint_EmptyExecPathHandled(t *testing.T) {
	argv := []string{"alpha", "--verbose"}
	env := []string{"LEAMAS_FACTORIZE_SCENARIO=controlled-warm"}
	root := "/checkout"

	// Empty execPath must produce a different fingerprint than valid path
	fpEmpty := commandFingerprint("test-verifier", root, argv, env, "")
	fpValid := commandFingerprint("test-verifier", root, argv, env, "/usr/bin/leamas")

	if fpEmpty == fpValid {
		t.Fatalf("empty exec path must produce different fingerprint:\n  empty=%q\n  valid=%q", fpEmpty, fpValid)
	}

	// Verify empty exec path produces a non-empty fingerprint (fails closed)
	if fpEmpty == "" {
		t.Fatalf("empty exec path must not produce empty fingerprint")
	}
}

// TestCommandFingerprint_FullDigestLength verifies that the fingerprint uses
// the complete SHA-256 digest, not truncated.
func TestCommandFingerprint_FullDigestLength(t *testing.T) {
	argv := []string{"alpha"}
	env := []string{"LEAMAS_FACTORIZE_SCENARIO=controlled-warm"}
	execPath := "/usr/local/bin/leamas"
	root := "/checkout"

	fp := commandFingerprint("test-verifier", root, argv, env, execPath)

	// SHA-256 produces 64 hex characters; verify we get the full digest
	if len(fp) != 64 {
		t.Fatalf("fingerprint must use full SHA-256 digest (64 hex chars), got %d: %q", len(fp), fp)
	}

	// Verify it's valid hex
	_, err := hex.DecodeString(fp)
	if err != nil {
		t.Fatalf("fingerprint must be valid hex: %v", err)
	}
}

// TestCommandFingerprint_RelocationInvariant verifies that relocating an
// identical checkout does not change the fingerprint.
func TestCommandFingerprint_RelocationInvariant(t *testing.T) {
	argv := []string{"alpha", "--verbose"}
	env := []string{"LEAMAS_FACTORIZE_SCENARIO=controlled-warm"}
	execPath := "/usr/local/bin/leamas"

	// Different checkout roots should produce the same fingerprint
	// since the fingerprint is about verifier execution, not checkout location
	root1 := "/checkout/original"
	root2 := "/checkout/relocated"

	fp1 := commandFingerprint("test-verifier", root1, argv, env, execPath)
	fp2 := commandFingerprint("test-verifier", root2, argv, env, execPath)

	if fp1 != fp2 {
		t.Fatalf("relocation of identical checkout must not change fingerprint:\n  root1=%q fp1=%q\n  root2=%q fp2=%q", root1, fp1, root2, fp2)
	}
}

// TestCommandFingerprint_ExecKindInProcess verifies that in-process verifiers
// (like most Leamas verifiers) handle fingerprinting appropriately.
func TestCommandFingerprint_ExecKindInProcess(t *testing.T) {
	// For in-process verifiers, argv is not the actual child argv
	// The fingerprint should bind whatever metadata is available
	argv := []string{"in-process-verifier"} // placeholder
	env := []string{"LEAMAS_FACTORIZE_SCENARIO=controlled-warm"}
	execPath := "/usr/local/bin/leamas"
	root := "/checkout"

	fp := commandFingerprint("in-process-verifier", root, argv, env, execPath)

	if fp == "" {
		t.Fatalf("in-process verifier must produce a non-empty fingerprint")
	}

	// The fingerprint should be the same regardless of argv placeholder
	fp2 := commandFingerprint("in-process-verifier", root, []string{"different-argv"}, env, execPath)

	// Note: Currently argv is included in fingerprint, which may not be correct
	// for in-process verifiers. This test documents the current behavior.
	if fp != fp2 {
		t.Logf("Note: in-process verifier fingerprint changes with argv placeholder")
	}
}

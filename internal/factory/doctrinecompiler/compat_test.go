package doctrinecompiler

import (
	"os"
	"strings"
	"testing"
)

// TestCheckCompilerCompatibility_EmptyRejected verifies an empty
// compiler version is rejected for a non-empty constraint.
func TestCheckCompilerCompatibility_EmptyRejected(t *testing.T) {
	if err := CheckCompilerCompatibility(">=0.1.0", ""); err == nil {
		t.Errorf("expected empty compiler version to be rejected")
	}
}

// TestCheckCompilerCompatibility_DevRejected verifies that a
// "dev" placeholder compiler version is rejected for a non-empty
// constraint.
func TestCheckCompilerCompatibility_DevRejected(t *testing.T) {
	if err := CheckCompilerCompatibility(">=0.1.0", "dev"); err == nil {
		t.Errorf("expected dev compiler version to be rejected")
	}
}

// TestCheckCompilerCompatibility_UnknownRejected verifies an
// "unknown" compiler version is rejected for a non-empty constraint.
func TestCheckCompilerCompatibility_UnknownRejected(t *testing.T) {
	if err := CheckCompilerCompatibility(">=0.1.0", "unknown"); err == nil {
		t.Errorf("expected unknown compiler version to be rejected")
	}
}

// TestCheckCompilerCompatibility_BelowVersionRejected verifies a
// version below the canonical constraint is rejected.
func TestCheckCompilerCompatibility_BelowVersionRejected(t *testing.T) {
	if err := CheckCompilerCompatibility(">=0.1.0", "0.0.9"); err == nil {
		t.Errorf("expected 0.0.9 to be rejected by >=0.1.0")
	}
}

// TestCheckCompilerCompatibility_AtVersionAccepted verifies that
// the exact floor version is accepted.
func TestCheckCompilerCompatibility_AtVersionAccepted(t *testing.T) {
	if err := CheckCompilerCompatibility(">=0.1.0", "0.1.0"); err != nil {
		t.Errorf("expected 0.1.0 to satisfy >=0.1.0: %v", err)
	}
}

// TestCheckCompilerCompatibility_LaterVersionAccepted verifies a
// later version is accepted.
func TestCheckCompilerCompatibility_LaterVersionAccepted(t *testing.T) {
	if err := CheckCompilerCompatibility(">=0.1.0", "0.2.0"); err != nil {
		t.Errorf("expected 0.2.0 to satisfy >=0.1.0: %v", err)
	}
	if err := CheckCompilerCompatibility(">=0.1.0", "1.5.0"); err != nil {
		t.Errorf("expected 1.5.0 to satisfy >=0.1.0: %v", err)
	}
}

// TestCompileRefusesIncompatibleVersion ensures the Compile path
// rejects an incompatible compiler version BEFORE any target
// mutation occurs.
func TestCompileRefusesIncompatibleVersion(t *testing.T) {
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	_, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.0.9"})
	if err == nil {
		t.Fatalf("expected compile to refuse incompatible version")
	}
	if !strings.Contains(err.Error(), "compiler_version") {
		t.Errorf("error does not name compiler_version subject: %v", err)
	}
	// The target must remain empty: no mutations occurred.
	entries, _ := os.ReadDir(target)
	for _, e := range entries {
		t.Errorf("target mutation despite rejection: %s", e.Name())
	}
}

// TestVerifyRejectsIncompatibleCompilerUnderLock verifies that an
// already-compiled lock is rejected by Verify under an incompatible
// runtime compiler.
func TestVerifyRejectsIncompatibleCompilerUnderLock(t *testing.T) {
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	withCompilerVersion(t, "0.0.9")
	result, err := Verify(pack, prof, target)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.OK {
		t.Errorf("expected Verify to fail under 0.0.9 runtime")
	}
	found := false
	for _, f := range result.Findings {
		if f.Kind == "compiler_incompatible" {
			found = true
		}
	}
	if !found {
		t.Errorf("missing compiler_incompatible finding: %v", result.Findings)
	}
}

// TestCompileSucceedsUnderCompatibleReleaseVersion proves a
// compatible release-built binary can compile and verify a fresh
// target.
func TestCompileSucceedsUnderCompatibleReleaseVersion(t *testing.T) {
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	withCompilerVersion(t, "0.1.0")
	result, err := Verify(pack, prof, target)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !result.OK {
		t.Errorf("verify failed: %v", result.Findings)
	}
}

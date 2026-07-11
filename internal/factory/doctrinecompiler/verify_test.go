package doctrinecompiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// verifyFresh compiles a fresh target, sets a real compiler version
// that satisfies the pack constraint, and returns the prepared bundle.
func verifyFresh(t *testing.T) (*Pack, string) {
	t.Helper()
	pack, err := LoadCorePack()
	if err != nil {
		t.Fatalf("LoadCorePack: %v", err)
	}
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	withCompilerVersion(t, "0.1.0")
	return pack, target
}

// TestVerifyFreshlyCompiledTarget ensures a clean compile is verified.
func TestVerifyFreshlyCompiledTarget(t *testing.T) {
	pack, target := verifyFresh(t)
	prof, _ := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	result, err := Verify(pack, prof, target)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !result.OK {
		t.Errorf("verify should pass: %v", result.Findings)
	}
}

// TestVerifyDetectsManagedModification verifies drift in a managed
// file causes verify to fail.
func TestVerifyDetectsManagedModification(t *testing.T) {
	pack, target := verifyFresh(t)
	prof, _ := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	mk := filepath.Join(target, ".factory/generated/factory.mk")
	if err := os.WriteFile(mk, []byte("tampered"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	result, err := Verify(pack, prof, target)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.OK {
		t.Errorf("expected verify to fail")
	}
	found := false
	for _, f := range result.Findings {
		if f.Kind == "managed_drift" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected managed_drift finding: %v", result.Findings)
	}
}

// TestVerifyDetectsMissingManaged verifies a missing managed file
// causes verify to fail.
func TestVerifyDetectsMissingManaged(t *testing.T) {
	pack, target := verifyFresh(t)
	prof, _ := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err := os.Remove(filepath.Join(target, ".factory/generated/factory.mk")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	result, err := Verify(pack, prof, target)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.OK {
		t.Errorf("expected verify to fail on missing managed file")
	}
}

// TestVerifyDetectsLockModification verifies tampering with the lock
// causes verify to fail.
func TestVerifyDetectsLockModification(t *testing.T) {
	pack, target := verifyFresh(t)
	prof, _ := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	lockPath := filepath.Join(target, ".factory/doctrine.lock.json")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	patched := []byte(strings.ReplaceAll(string(data), `"factory-core-v1"`, `"bogus-pack"`))
	if err := os.WriteFile(lockPath, patched, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	result, err := Verify(pack, prof, target)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.OK {
		t.Errorf("expected verify to fail on lock modification")
	}
}

// TestVerifyDetectsPackDigestMismatch verifies a corrupted pack
// produces a digest mismatch finding.
func TestVerifyDetectsPackDigestMismatch(t *testing.T) {
	pack, target := verifyFresh(t)
	prof, _ := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	lockPath := filepath.Join(target, ".factory/doctrine.lock.json")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	patched := []byte(strings.ReplaceAll(string(data),
		`"pack_digest": "`+string(pack.PackDigest())+`"`,
		`"pack_digest": "0000000000000000000000000000000000000000000000000000000000000000"`))
	if err := os.WriteFile(lockPath, patched, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	result, err := Verify(pack, prof, target)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.OK {
		t.Errorf("expected verify to fail on digest mismatch")
	}
}

// TestVerifyDetectsProfileMismatch verifies that an unexpected
// profile_id in the lock is detected.
func TestVerifyDetectsProfileMismatch(t *testing.T) {
	pack, target := verifyFresh(t)
	prof, _ := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	lockPath := filepath.Join(target, ".factory/doctrine.lock.json")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	patched := []byte(strings.ReplaceAll(string(data),
		`"fsharp-elm-service-v1"`, `"some-other-profile"`))
	if err := os.WriteFile(lockPath, patched, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	result, err := Verify(pack, prof, target)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.OK {
		t.Errorf("expected verify to fail on profile mismatch")
	}
}

// TestVerifyDetectsMissingMakefileInclude verifies the
// makefile-include observed contract.
func TestVerifyDetectsMissingMakefileInclude(t *testing.T) {
	pack, target := verifyFresh(t)
	prof, _ := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	mk := filepath.Join(target, "Makefile")
	if err := os.WriteFile(mk, []byte("gate: factorize\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	result, err := Verify(pack, prof, target)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.OK {
		t.Errorf("expected verify to fail on missing include")
	}
}

// TestVerifyDetectsGateWithoutFactorizeDep verifies the
// makefile-target-dep observed contract.
func TestVerifyDetectsGateWithoutFactorizeDep(t *testing.T) {
	pack, target := verifyFresh(t)
	prof, _ := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	mk := filepath.Join(target, "Makefile")
	bad := "# Generated by Leamas from factory-core-v1.\n# Do not edit this file directly.\ninclude .factory/generated/factory.mk\n.PHONY: gate\ngate: other-step\n"
	if err := os.WriteFile(mk, []byte(bad), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	result, err := Verify(pack, prof, target)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.OK {
		t.Errorf("expected verify to fail on missing dep")
	}
}

// TestVerifyIgnoresUnrelatedChanges verifies a target with extra
// unrelated files still verifies.
func TestVerifyIgnoresUnrelatedChanges(t *testing.T) {
	pack, target := verifyFresh(t)
	prof, _ := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err := os.WriteFile(filepath.Join(target, "user-notes.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	result, err := Verify(pack, prof, target)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !result.OK {
		t.Errorf("verify should pass with unrelated changes: %v", result.Findings)
	}
}

// TestVerifyPerformsNoWrites asserts verify does not modify the target.
func TestVerifyPerformsNoWrites(t *testing.T) {
	pack, target := verifyFresh(t)
	prof, _ := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	before := snapshotTree(t, target)
	if _, err := Verify(pack, prof, target); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	after := snapshotTree(t, target)
	if !equalStringSet(before, after) {
		t.Errorf("verify wrote to target: before=%v after=%v", before, after)
	}
}

// TestVerifyDetectsCompilerIncompatibility verifies that a compiler
// version violating the pack's constraint is reported. The test is
// skipped when the pack declares no constraint (empty
// compiler_version) since no check is meaningful in that case.
func TestVerifyDetectsCompilerIncompatibility(t *testing.T) {
	pack, _ := LoadCorePack()
	if pack == nil {
		t.Skip("LoadCorePack failed")
	}
	if pack.CompilerVersion == "" {
		t.Skip("pack declares no compiler_version constraint")
	}
	withCompilerVersion(t, "0.0.5")
	target := newEmptyTarget(t)
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	if _, err := Compile(pack, prof, target, CompilerOptions{}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	result, err := Verify(pack, prof, target)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.OK {
		t.Errorf("expected verify to fail on incompatible compiler")
	}
	found := false
	for _, f := range result.Findings {
		if f.Kind == "compiler_incompatible" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected compiler_incompatible finding: %v", result.Findings)
	}
}

// TestVerifyDetectsUnexpectedSeededEntry verifies that an unexpected
// seeded file in the lock is detected. The lock must record exactly
// the canonical seeded set.
func TestVerifyDetectsUnexpectedSeededEntry(t *testing.T) {
	pack, target := verifyFresh(t)
	prof, _ := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	lockPath := filepath.Join(target, ".factory/doctrine.lock.json")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	patched := []byte(strings.Replace(string(data),
		`"seeded_files": [`,
		`"seeded_files": [{"path": "docs/stray-extra.md"},`,
		1))
	if err := os.WriteFile(lockPath, patched, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	result, err := Verify(pack, prof, target)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.OK {
		t.Errorf("expected verify to fail on unexpected seeded entry")
	}
	found := false
	for _, f := range result.Findings {
		if f.Kind == "lock_unexpected_seeded" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected lock_unexpected_seeded finding: %v", result.Findings)
	}
}

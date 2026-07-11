package doctrinecompiler

import (
	"path/filepath"
	"strings"
	"testing"
)

// writeSelectorForTest plants a selector with the given pack/profile
// pair into a target repository. The selector schema is the same
// canonical one used by the compiler.
func writeSelectorForTest(t *testing.T, target, pack, profile string) {
	t.Helper()
	factory := filepath.Join(target, ".factory")
	if err := osMkdirAllImpl(factory, 0o755); err != nil {
		t.Fatalf("mkdir .factory: %v", err)
	}
	body := `{
  "schema_version": 1,
  "pack": "` + pack + `",
  "profile": "` + profile + `"
}`
	if err := osWriteFileImpl(filepath.Join(factory, "project.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write selector: %v", err)
	}
}

// TestVerifySelectorPackCanonical verifies the canonical selector is
// accepted.
func TestVerifySelectorPackCanonical(t *testing.T) {
	pack, target := verifyFresh(t)
	prof, _ := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	// verifyFresh() compiles a valid target; do not modify the
	// committed selector.
	result, err := Verify(pack, prof, target)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !result.OK {
		t.Errorf("verify should pass: %v", result.Findings)
	}
}

// TestVerifySelectorPackForeignRejected verifies that a selector
// naming a different pack is rejected by Verify, before any target
// inspection produces canonical-pack findings.
func TestVerifySelectorPackForeignRejected(t *testing.T) {
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	// Replace the committed selector with a foreign pack.
	writeSelectorForTest(t, target, "other-pack", "fsharp-elm-service-v1")
	result, err := Verify(pack, prof, target)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.OK {
		t.Fatalf("expected verify to fail on foreign pack selector")
	}
	var f *VerifyFinding
	for i := range result.Findings {
		if result.Findings[i].Kind == "selector_pack_mismatch" {
			f = &result.Findings[i]
			break
		}
	}
	if f == nil {
		t.Fatalf("missing selector_pack_mismatch finding: %+v", result.Findings)
	}
	if !strings.Contains(f.Message, "other-pack") || !strings.Contains(f.Message, "factory-core-v1") {
		t.Errorf("finding does not name both packs: %s", f.Message)
	}
}

// TestExplainSelectorPackForeignRejected verifies that Explain also
// rejects a foreign pack selector and returns an error mentioning
// both packs.
func TestExplainSelectorPackForeignRejected(t *testing.T) {
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	writeSelectorForTest(t, target, "other-pack", "fsharp-elm-service-v1")
	_, err := Explain(pack, prof, target, "0.1.0", "unknown", "unknown")
	if err == nil {
		t.Fatalf("expected Explain to reject foreign pack selector")
	}
	if !strings.Contains(err.Error(), "other-pack") ||
		!strings.Contains(err.Error(), "factory-core-v1") {
		t.Errorf("error does not name both packs: %v", err)
	}
}

// TestVerifySelectorInferenceWorks ensures selector inference still
// works for the canonical pack when --profile is omitted.
func TestVerifySelectorInferenceWorks(t *testing.T) {
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	withCompilerVersion(t, "0.1.0")
	// ResolveSelection should return factory-core-v1 and the
	// canonical profile from the committed selector.
	selPack, selProf, err := ResolveSelection("", "", target, true)
	if err != nil {
		t.Fatalf("ResolveSelection: %v", err)
	}
	if selPack != "factory-core-v1" {
		t.Errorf("selector pack = %q, want factory-core-v1", selPack)
	}
	if string(selProf) != "fsharp-elm-service-v1" {
		t.Errorf("selector profile = %q, want fsharp-elm-service-v1", selProf)
	}
}

// TestSelectorExplicitProfileRetained ensures that passing an
// explicit --profile (or analogous library flag) bypasses selector
// inference without affecting the result.
func TestSelectorExplicitProfileRetained(t *testing.T) {
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	// Plant a foreign selector. With an explicit profile flag the
	// selector is irrelevant; ResolveSelection with empty explicit
	// profile and fallback=false must not load the selector at all.
	writeSelectorForTest(t, target, "other-pack", "fsharp-elm-service-v1")
	_, selProf, err := ResolveSelection("", "fsharp-elm-service-v1", target, false)
	if err != nil {
		t.Fatalf("ResolveSelection with explicit profile: %v", err)
	}
	if string(selProf) != "fsharp-elm-service-v1" {
		t.Errorf("explicit profile not retained: %q", selProf)
	}
}

// TestCompileSelectorUsesCanonicalPack verifies that compile writes
// the canonical pack id into the selector it produces.
func TestCompileSelectorUsesCanonicalPack(t *testing.T) {
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	sel, err := readSelector(filepath.Join(target, ".factory/project.json"))
	if err != nil {
		t.Fatalf("readSelector: %v", err)
	}
	if sel.Pack != "factory-core-v1" {
		t.Errorf("selector pack = %q, want factory-core-v1", sel.Pack)
	}
	if string(sel.Profile) != "fsharp-elm-service-v1" {
		t.Errorf("selector profile = %q, want fsharp-elm-service-v1", sel.Profile)
	}
}

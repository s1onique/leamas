package doctrinecompiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// newEmptyTarget creates a fresh empty target root.
func newEmptyTarget(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// copyTree copies the contents of src into dst, recursively.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy: %v", err)
	}
}

// expectedFixtureDir returns the path to the golden expected tree.
func expectedFixtureDir(t *testing.T) string {
	t.Helper()
	return "testdata/fsharp-elm-empty/expected"
}

// TestPlanEmptyTargetProducesCreateActions verifies the empty-target
// plan classifies every desired file as create-* with no reject
// actions.
func TestPlanEmptyTargetProducesCreateActions(t *testing.T) {
	pack, err := LoadCorePack()
	if err != nil {
		t.Fatalf("LoadCorePack: %v", err)
	}
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	target := newEmptyTarget(t)
	plan, err := Plan(pack, prof, target)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	classes := map[ActionClass]int{}
	for _, a := range plan.Actions {
		classes[a.Class]++
	}
	if classes[ActionCreateManaged]+classes[ActionCreateSeeded] != len(plan.Actions) {
		t.Errorf("plan should be all create-*, got %v", classes)
	}
	if classes[ActionReject] != 0 {
		t.Errorf("plan should not reject, got %d rejects", classes[ActionReject])
	}
}

// TestPlanIdempotent verifies a second plan against a freshly compiled
// target reports all unchanged / preserve-seeded.
func TestPlanIdempotent(t *testing.T) {
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
		t.Fatalf("first Compile: %v", err)
	}
	plan, err := Plan(pack, prof, target)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	for _, a := range plan.Actions {
		if a.Class == ActionUpdateManaged {
			t.Errorf("plan reports update-managed after compile: %s", a.Path)
		}
		if a.Class == ActionReject {
			t.Errorf("plan reports reject after compile: %s", a.Path)
		}
	}
}

// TestPlanDetectsUpdateManaged verifies that a tampered managed file
// triggers an update-managed classification.
func TestPlanDetectsUpdateManaged(t *testing.T) {
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
	// Tamper with a managed file.
	path := filepath.Join(target, ".factory/generated/factory.mk")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_, _ = f.WriteString("\n# tamper\n")
	_ = f.Close()
	plan, err := Plan(pack, prof, target)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	found := false
	for _, a := range plan.Actions {
		if a.Class == ActionUpdateManaged && a.Path == ".factory/generated/factory.mk" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected update-managed for factory.mk; got %v", plan.Actions)
	}
}

// TestPlanDetectsMissingManaged verifies that removing a managed file
// yields create-managed on the next plan.
func TestPlanDetectsMissingManaged(t *testing.T) {
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
	if err := os.Remove(filepath.Join(target, ".factory/generated/factory.mk")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	plan, err := Plan(pack, prof, target)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	found := false
	for _, a := range plan.Actions {
		if a.Class == ActionCreateManaged && a.Path == ".factory/generated/factory.mk" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected create-managed for factory.mk after removal")
	}
}

// TestPlanPreserveSeeded verifies an existing seeded file stays.
func TestPlanPreserveSeeded(t *testing.T) {
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
	// Modify the seeded Makefile with custom content. The next plan
	// must still classify it as preserve-seeded, not update.
	mk := filepath.Join(target, "Makefile")
	custom := "# custom content\n.PHONY: gate\ngate: factorize custom-step\n"
	if err := os.WriteFile(mk, []byte(custom), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	plan, err := Plan(pack, prof, target)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	for _, a := range plan.Actions {
		if a.Path == "Makefile" && a.Class != ActionPreserveSeeded {
			t.Errorf("Makefile class = %s, want preserve-seeded", a.Class)
		}
	}
}

// TestPlanObsoleteManaged verifies a recorded managed file that is no
// longer in the projection is flagged for removal.
func TestPlanObsoleteManaged(t *testing.T) {
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
	// Manually plant an obsolete managed file by writing to the lock.
	// The simplest path is to copy a managed file under a different
	// path and extend the lock; here we instead patch the lock to
	// claim a path that does not exist in the projection.
	lockPath := filepath.Join(target, ".factory/doctrine.lock.json")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	// Plant a fake obsolete file.
	plant := filepath.Join(target, ".factory/generated/obsolete.txt")
	if err := os.WriteFile(plant, []byte("obsolete"), 0o644); err != nil {
		t.Fatalf("plant: %v", err)
	}
	patched, err2 := patchLockAddManaged(string(data),
		".factory/generated/obsolete.txt",
		"0000000000000000000000000000000000000000000000000000000000000000")
	if err2 != nil {
		t.Fatalf("patch lock: %v", err2)
	}
	if err := os.WriteFile(lockPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write patched lock: %v", err)
	}
	plan, err := Plan(pack, prof, target)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	found := false
	for _, a := range plan.Actions {
		if a.Class == ActionRemoveObsoleteManaged && a.Path == ".factory/generated/obsolete.txt" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected remove-obsolete-managed for obsolete.txt; got %v", plan.Actions)
	}
}

// TestPlanIgnoresUnrelatedFiles ensures unrelated files do not appear
// in the plan output.
func TestPlanIgnoresUnrelatedFiles(t *testing.T) {
	pack, err := LoadCorePack()
	if err != nil {
		t.Fatalf("LoadCorePack: %v", err)
	}
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	target := newEmptyTarget(t)
	if err := os.WriteFile(filepath.Join(target, "README.md"), []byte("user"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Compile(pack, prof, target, CompilerOptions{}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	plan, err := Plan(pack, prof, target)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	for _, a := range plan.Actions {
		if a.Path == "README.md" {
			t.Errorf("plan references unrelated README.md: %v", a)
		}
	}
}

// TestPlanRejectsSymlink verifies that a target containing a symlink at
// the destination of a desired managed file produces a reject action.
func TestPlanRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on windows")
	}
	pack, err := LoadCorePack()
	if err != nil {
		t.Fatalf("LoadCorePack: %v", err)
	}
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	target := newEmptyTarget(t)
	if err := os.MkdirAll(filepath.Join(target, ".factory/generated"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Symlink("/etc/hostname", filepath.Join(target, ".factory/generated/factory.mk")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	_, err = Plan(pack, prof, target)
	if err == nil {
		t.Errorf("expected plan to fail on symlink")
	}
	if err != nil && !strings.Contains(err.Error(), "unsafe") {
		t.Errorf("expected unsafe error, got %v", err)
	}
}

// TestPlanPerformsNoWrites is a defensive test that re-running Plan
// twice against the same target yields identical output and never
// touches the filesystem between calls.
func TestPlanPerformsNoWrites(t *testing.T) {
	pack, err := LoadCorePack()
	if err != nil {
		t.Fatalf("LoadCorePack: %v", err)
	}
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	target := newEmptyTarget(t)
	a, err := Plan(pack, prof, target)
	if err != nil {
		t.Fatalf("first plan: %v", err)
	}
	// Target must be untouched: no files created.
	entries, _ := os.ReadDir(target)
	if len(entries) != 0 {
		t.Errorf("Plan wrote to target: %d entries", len(entries))
	}
	b, err := Plan(pack, prof, target)
	if err != nil {
		t.Fatalf("second plan: %v", err)
	}
	if string(FormatPlan(a)) != string(FormatPlan(b)) {
		t.Errorf("plan output differs across two calls")
	}
}

// TestPlanAcceptsManagedSeededCollision makes sure output and seed
// paths are deduplicated by the decoder/validator.
func TestPlanOutputSeedCollisionIsRejected(t *testing.T) {
	// Build a pack with output path colliding with seed path.
	raw := []byte(`{
        "schema_version": 1,
        "pack_id": "x",
        "pack_version": "1.0.0",
        "compiler_version": ">=0",
        "doctrines": [{"id":"a","summary":""}],
        "target_profiles": [{
            "id":"p","summary":"",
            "outputs":[{"id":"o","path":"dup","ownership":"managed","content_id":"c"}],
            "seeds":[{"id":"s","path":"dup","ownership":"seeded","content_id":"c"}],
            "observed_contracts":[],"factorize_checks":[],"extension_points":[]
        }]
    }`)
	_, _, err := DecodePack(raw)
	if err == nil {
		t.Errorf("expected output/seed collision rejection")
	}
}

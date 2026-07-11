package doctrinecompiler

import (
	"strings"
	"testing"
)

// TestCorePackValid verifies the canonical pack loads.
func TestCorePackValid(t *testing.T) {
	pack, err := LoadCorePack()
	if err != nil {
		t.Fatalf("LoadCorePack: %v", err)
	}
	if pack.PackID != PackId("factory-core-v1") {
		t.Errorf("PackID = %q, want factory-core-v1", pack.PackID)
	}
	if pack.SchemaVersion != SupportedPackSchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", pack.SchemaVersion, SupportedPackSchemaVersion)
	}
	if len(pack.Doctrines) == 0 {
		t.Errorf("Doctrines empty")
	}
	if len(pack.Profiles) == 0 {
		t.Errorf("Profiles empty")
	}
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	if len(prof.Outputs) == 0 || len(prof.Seeds) == 0 {
		t.Errorf("profile empty: outputs=%d seeds=%d", len(prof.Outputs), len(prof.Seeds))
	}
}

// TestPackDigestStable ensures two decodes yield identical digests.
func TestPackDigestStable(t *testing.T) {
	a, err := LoadCorePack()
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	b, err := LoadCorePack()
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if a.PackDigest() != b.PackDigest() {
		t.Errorf("digests differ: %s vs %s", a.PackDigest(), b.PackDigest())
	}
}

// TestPackSchemaVersionEnforced verifies schema mismatch is rejected.
func TestPackSchemaVersionEnforced(t *testing.T) {
	data := []byte(`{
        "schema_version": 99,
        "pack_id": "x",
        "pack_version": "1.0.0",
        "compiler_version": ">=0",
        "doctrines": [{"id": "a", "summary": ""}],
        "target_profiles": [{"id": "p", "summary": "",
            "outputs": [{"id": "o", "path": "Makefile",
                "ownership": "managed", "content_id": "x"}],
            "seeds": [], "observed_contracts": [],
            "factorize_checks": [], "extension_points": []
        }]
    }`)
	_, _, err := DecodePack(data)
	if err == nil {
		t.Fatal("expected schema-version rejection")
	}
	if !strings.Contains(err.Error(), "schema_version") {
		t.Errorf("error did not mention schema_version: %v", err)
	}
}

// TestPackUnknownField verifies strict decoding.
func TestPackUnknownField(t *testing.T) {
	raw := corePackBytes()
	// Inject an unknown field.
	patched := append([]byte{}, raw...)
	patched = append(patched[:len(patched)-1], []byte(", \"rogue\": true}")...)
	if _, _, err := DecodePack(patched); err == nil {
		t.Fatal("expected unknown-field rejection")
	}
}

// TestPackDuplicateDoctrineFails verifies duplicate ids are rejected.
func TestPackDuplicateDoctrineFails(t *testing.T) {
	data := []byte(`{
        "schema_version": 1,
        "pack_id": "x",
        "pack_version": "1.0.0",
        "compiler_version": ">=0",
        "doctrines": [
            {"id": "dup", "summary": ""},
            {"id": "dup", "summary": ""}
        ],
        "target_profiles": [{
            "id": "p", "summary": "",
            "outputs": [{"id":"o","path":"a","ownership":"managed","content_id":"c"}],
            "seeds": [], "observed_contracts": [], "factorize_checks": [], "extension_points": []
        }]
    }`)
	_, _, err := DecodePack(data)
	if err == nil {
		t.Fatal("expected duplicate-doctrine rejection")
	}
}

// TestPackDuplicateProfileFails verifies duplicate profile ids fail.
func TestPackDuplicateProfileFails(t *testing.T) {
	data := []byte(`{
        "schema_version": 1,
        "pack_id": "x",
        "pack_version": "1.0.0",
        "compiler_version": ">=0",
        "doctrines": [{"id":"a","summary":""}],
        "target_profiles": [
            {"id":"p","summary":"","outputs":[{"id":"o","path":"a","ownership":"managed","content_id":"c"}],"seeds":[],"observed_contracts":[],"factorize_checks":[],"extension_points":[]},
            {"id":"p","summary":"","outputs":[{"id":"o","path":"a","ownership":"managed","content_id":"c"}],"seeds":[],"observed_contracts":[],"factorize_checks":[],"extension_points":[]}
        ]
    }`)
	_, _, err := DecodePack(data)
	if err == nil {
		t.Fatal("expected duplicate-profile rejection")
	}
}

// TestPackDuplicateNormalizedPathFails verifies duplicate output paths fail.
func TestPackDuplicateNormalizedPathFails(t *testing.T) {
	data := []byte(`{
        "schema_version": 1,
        "pack_id": "x",
        "pack_version": "1.0.0",
        "compiler_version": ">=0",
        "doctrines": [{"id":"a","summary":""}],
        "target_profiles": [{
            "id":"p","summary":"",
            "outputs":[
                {"id":"o1","path":"a/b","ownership":"managed","content_id":"c"},
                {"id":"o2","path":"a/./b","ownership":"managed","content_id":"c"}
            ],
            "seeds":[],"observed_contracts":[],"factorize_checks":[],"extension_points":[]
        }]
    }`)
	_, _, err := DecodePack(data)
	if err == nil {
		t.Fatal("expected duplicate-path rejection")
	}
}

// TestPackAbsolutePathFails verifies absolute output paths fail.
func TestPackAbsolutePathFails(t *testing.T) {
	data := []byte(`{
        "schema_version": 1,
        "pack_id": "x",
        "pack_version": "1.0.0",
        "compiler_version": ">=0",
        "doctrines": [{"id":"a","summary":""}],
        "target_profiles": [{
            "id":"p","summary":"",
            "outputs":[{"id":"o","path":"/abs/path","ownership":"managed","content_id":"c"}],
            "seeds":[],"observed_contracts":[],"factorize_checks":[],"extension_points":[]
        }]
    }`)
	_, _, err := DecodePack(data)
	if err == nil {
		t.Fatal("expected absolute-path rejection")
	}
}

// TestPackTraversalPathFails verifies traversal segments fail.
func TestPackTraversalPathFails(t *testing.T) {
	data := []byte(`{
        "schema_version": 1,
        "pack_id": "x",
        "pack_version": "1.0.0",
        "compiler_version": ">=0",
        "doctrines": [{"id":"a","summary":""}],
        "target_profiles": [{
            "id":"p","summary":"",
            "outputs":[{"id":"o","path":"a/../escape","ownership":"managed","content_id":"c"}],
            "seeds":[],"observed_contracts":[],"factorize_checks":[],"extension_points":[]
        }]
    }`)
	_, _, err := DecodePack(data)
	if err == nil {
		t.Fatal("expected traversal rejection")
	}
}

// TestPackInvalidOwnershipFails verifies ownership label validation.
func TestPackInvalidOwnershipFails(t *testing.T) {
	data := []byte(`{
        "schema_version": 1,
        "pack_id": "x",
        "pack_version": "1.0.0",
        "compiler_version": ">=0",
        "doctrines": [{"id":"a","summary":""}],
        "target_profiles": [{
            "id":"p","summary":"",
            "outputs":[{"id":"o","path":"a","ownership":"banana","content_id":"c"}],
            "seeds":[],"observed_contracts":[],"factorize_checks":[],"extension_points":[]
        }]
    }`)
	_, _, err := DecodePack(data)
	if err == nil {
		t.Fatal("expected ownership rejection")
	}
}

// TestPackEmptyIdFails verifies empty ids fail.
func TestPackEmptyIdFails(t *testing.T) {
	data := []byte(`{
        "schema_version": 1,
        "pack_id": "",
        "pack_version": "1.0.0",
        "compiler_version": ">=0",
        "doctrines": [{"id":"a","summary":""}],
        "target_profiles": [{"id":"p","summary":"","outputs":[{"id":"o","path":"a","ownership":"managed","content_id":"c"}],"seeds":[],"observed_contracts":[],"factorize_checks":[],"extension_points":[]}]
    }`)
	_, _, err := DecodePack(data)
	if err == nil {
		t.Fatal("expected empty pack_id rejection")
	}
}

// TestPackUnknownDoctrineRef verifies the unknown-doctrine reference
// rejection path. We construct a profile with an observed contract
// referencing a non-existent kind and expect failure.
func TestPackUnknownObservedKind(t *testing.T) {
	data := []byte(`{
        "schema_version": 1,
        "pack_id": "x",
        "pack_version": "1.0.0",
        "compiler_version": ">=0",
        "doctrines": [{"id":"a","summary":""}],
        "target_profiles": [{
            "id":"p","summary":"",
            "outputs":[{"id":"o","path":"a","ownership":"managed","content_id":"c"}],
            "seeds":[],
            "observed_contracts":[{"id":"c","kind":"bogus-kind","path":"a"}],
            "factorize_checks":[],"extension_points":[]
        }]
    }`)
	_, _, err := DecodePack(data)
	if err == nil {
		t.Fatal("expected observed_contract kind rejection")
	}
}

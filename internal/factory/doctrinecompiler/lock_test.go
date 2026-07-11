package doctrinecompiler

import (
	"path/filepath"
	"strings"
	"testing"
)

// writeLockForTest writes a synthetic lock file to disk and returns
// its absolute path. The fixture is rebuilt per test so the cases can
// vary paths and IDs freely.
func writeLockForTest(t *testing.T, body string) string {
	t.Helper()
	tmp := t.TempDir()
	p := filepath.Join(tmp, ".factory")
	if err := writeFileMkdirAll(p, "doctrine.lock.json", body); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	return filepath.Join(p, "doctrine.lock.json")
}

// writeFileMkdirAll is a tiny test helper that ensures the parent
// directory exists before writing a file. It does not depend on the
// transaction package.
func writeFileMkdirAll(dir, name, body string) error {
	if err := osMkdirAllImpl(dir, 0o755); err != nil {
		return err
	}
	return osWriteFileImpl(filepath.Join(dir, name), []byte(body), 0o644)
}

// TestReadLockFileRejectsDuplicates table-drives the duplicate
// detection in ReadLockFile.
func TestReadLockFileRejectsDuplicates(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		wantError string
	}{
		{
			name: "exact_duplicate_managed",
			body: `{
				"schema_version": 1,
				"pack_id": "factory-core-v1",
				"pack_version": "1.0.0",
				"pack_digest": "x",
				"profile_id": "fsharp-elm-service-v1",
				"compiler_version": "0.1.0",
				"compiler_commit": "unknown",
				"managed_files": [
					{"path": ".factory/generated/factory.mk", "digest": "a"},
					{"path": ".factory/generated/factory.mk", "digest": "b"}
				],
				"seeded_files": [],
				"observed_contracts": []
			}`,
			wantError: "duplicate normalized managed path",
		},
		{
			name: "managed_normalize_to_same",
			body: `{
				"schema_version": 1,
				"pack_id": "factory-core-v1",
				"pack_version": "1.0.0",
				"pack_digest": "x",
				"profile_id": "fsharp-elm-service-v1",
				"compiler_version": "0.1.0",
				"compiler_commit": "unknown",
				"managed_files": [
					{"path": "a/b", "digest": "a"},
					{"path": "./a/./b", "digest": "b"}
				],
				"seeded_files": [],
				"observed_contracts": []
			}`,
			wantError: "duplicate normalized managed path",
		},
		{
			name: "exact_duplicate_seeded",
			body: `{
				"schema_version": 1,
				"pack_id": "factory-core-v1",
				"pack_version": "1.0.0",
				"pack_digest": "x",
				"profile_id": "fsharp-elm-service-v1",
				"compiler_version": "0.1.0",
				"compiler_commit": "unknown",
				"managed_files": [],
				"seeded_files": [
					{"path": "Makefile"},
					{"path": "Makefile"}
				],
				"observed_contracts": []
			}`,
			wantError: "duplicate normalized seeded path",
		},
		{
			name: "seeded_normalize_to_same",
			body: `{
				"schema_version": 1,
				"pack_id": "factory-core-v1",
				"pack_version": "1.0.0",
				"pack_digest": "x",
				"profile_id": "fsharp-elm-service-v1",
				"compiler_version": "0.1.0",
				"compiler_commit": "unknown",
				"managed_files": [],
				"seeded_files": [
					{"path": "a/b"},
					{"path": "./a/./b"}
				],
				"observed_contracts": []
			}`,
			wantError: "duplicate normalized seeded path",
		},
		{
			name: "managed_seeded_collision",
			body: `{
				"schema_version": 1,
				"pack_id": "factory-core-v1",
				"pack_version": "1.0.0",
				"pack_digest": "x",
				"profile_id": "fsharp-elm-service-v1",
				"compiler_version": "0.1.0",
				"compiler_commit": "unknown",
				"managed_files": [
					{"path": "Makefile", "digest": "a"}
				],
				"seeded_files": [
					{"path": "Makefile"}
				],
				"observed_contracts": []
			}`,
			wantError: "cross-ownership collision",
		},
		{
			name: "duplicate_observed_id",
			body: `{
				"schema_version": 1,
				"pack_id": "factory-core-v1",
				"pack_version": "1.0.0",
				"pack_digest": "x",
				"profile_id": "fsharp-elm-service-v1",
				"compiler_version": "0.1.0",
				"compiler_commit": "unknown",
				"managed_files": [],
				"seeded_files": [],
				"observed_contracts": [
					{"id": "x", "kind": "makefile-include", "path": "Makefile"},
					{"id": "x", "kind": "makefile-include", "path": "Makefile"}
				]
			}`,
			wantError: "duplicate observed-contract id",
		},
		{
			name: "empty_managed_path",
			body: `{
				"schema_version": 1,
				"pack_id": "factory-core-v1",
				"pack_version": "1.0.0",
				"pack_digest": "x",
				"profile_id": "fsharp-elm-service-v1",
				"compiler_version": "0.1.0",
				"compiler_commit": "unknown",
				"managed_files": [
					{"path": "", "digest": "a"}
				],
				"seeded_files": [],
				"observed_contracts": []
			}`,
			wantError: "managed_files[0].path is empty",
		},
		{
			name: "empty_observed_id",
			body: `{
				"schema_version": 1,
				"pack_id": "factory-core-v1",
				"pack_version": "1.0.0",
				"pack_digest": "x",
				"profile_id": "fsharp-elm-service-v1",
				"compiler_version": "0.1.0",
				"compiler_commit": "unknown",
				"managed_files": [],
				"seeded_files": [],
				"observed_contracts": [
					{"id": "", "kind": "makefile-include", "path": "Makefile"}
				]
			}`,
			wantError: "observed_contracts[0].id is empty",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := writeLockForTest(t, c.body)
			_, err := ReadLockFile(path)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", c.wantError)
			}
			if !strings.Contains(err.Error(), c.wantError) {
				t.Errorf("error %q does not contain %q", err.Error(), c.wantError)
			}
		})
	}
}

// TestReadLockFileAcceptsValid verifies a valid distinct lock loads.
func TestReadLockFileAcceptsValid(t *testing.T) {
	body := `{
		"schema_version": 1,
		"pack_id": "factory-core-v1",
		"pack_version": "1.0.0",
		"pack_digest": "x",
		"profile_id": "fsharp-elm-service-v1",
		"compiler_version": "0.1.0",
		"compiler_commit": "unknown",
		"managed_files": [
			{"path": ".factory/generated/factory.mk", "digest": "a"},
			{"path": "docs/factory/README.md", "digest": "b"}
		],
		"seeded_files": [
			{"path": "Makefile"},
			{"path": ".factory/project.json"}
		],
		"observed_contracts": [
			{"id": "alpha", "kind": "makefile-include", "path": "Makefile"},
			{"id": "beta", "kind": "makefile-target-dep", "path": "Makefile", "target": "gate", "dependency": "factorize"}
		]
	}`
	path := writeLockForTest(t, body)
	lf, err := ReadLockFile(path)
	if err != nil {
		t.Fatalf("ReadLockFile: %v", err)
	}
	if len(lf.ManagedFiles) != 2 {
		t.Errorf("managed count = %d", len(lf.ManagedFiles))
	}
}

// TestVerifyFailsBeforeTargetWhenLockAmbiguous proves that an
// ambiguous lock is rejected before Verify inspects any target files.
func TestVerifyFailsBeforeTargetWhenLockAmbiguous(t *testing.T) {
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	// Replace the lock with an ambiguous duplicate-managed entry.
	lockPath := filepath.Join(target, ".factory/doctrine.lock.json")
	body := `{
		"schema_version": 1,
		"pack_id": "factory-core-v1",
		"pack_version": "1.0.0",
		"pack_digest": "x",
		"profile_id": "fsharp-elm-service-v1",
		"compiler_version": "0.1.0",
		"compiler_commit": "unknown",
		"managed_files": [
			{"path": "Makefile", "digest": "a"},
			{"path": "Makefile", "digest": "b"}
		],
		"seeded_files": [],
		"observed_contracts": []
	}`
	if err := osWriteFileImpl(lockPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write ambiguous lock: %v", err)
	}
	_, err := Verify(pack, prof, target)
	if err == nil {
		t.Fatalf("expected Verify to refuse ambiguous lock")
	}
	if !strings.Contains(err.Error(), "duplicate") &&
		!strings.Contains(err.Error(), "lock") {
		t.Errorf("expected lock-duplicate error, got: %v", err)
	}
}

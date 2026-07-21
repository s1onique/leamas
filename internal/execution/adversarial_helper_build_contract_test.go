//go:build unix || darwin || linux

package execution

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestHelperBuildDiscoversPackageSources(t *testing.T) {
	sourceDir := copyHelperPackage(t)

	before, err := loadHelperSourceSnapshot(sourceDir)
	if err != nil {
		t.Fatalf("load original source snapshot: %v", err)
	}

	const addedName = "freshness_discovery.go"
	addedPath := filepath.Join(sourceDir, addedName)
	if err := os.WriteFile(addedPath, []byte(
		"package main\n\nconst freshnessDiscovery = true\n"), 0o644); err != nil {
		t.Fatalf("add package source: %v", err)
	}

	after, err := loadHelperSourceSnapshot(sourceDir)
	if err != nil {
		t.Fatalf("load changed source snapshot: %v", err)
	}
	if !slices.Contains(after.Files, addedName) {
		t.Fatalf("package-discovered sources omit %q: %v", addedName, after.Files)
	}
	if after.Digest == before.Digest {
		t.Fatalf("source digest did not change after adding %q", addedName)
	}
}

func TestHelperBuildIdentityChangesForEveryPackageSource(t *testing.T) {
	sourceDir := copyHelperPackage(t)
	outputDir := t.TempDir()

	baseline, built, err := buildContentAddressedHelper(sourceDir, outputDir)
	if err != nil {
		t.Fatalf("build baseline helper: %v", err)
	}
	if !built {
		t.Fatal("first content-addressed build unexpectedly reused output")
	}
	reused, built, err := buildContentAddressedHelper(sourceDir, outputDir)
	if err != nil {
		t.Fatalf("reuse baseline helper: %v", err)
	}
	if built {
		t.Fatal("unchanged content unexpectedly forced a rebuild")
	}
	if reused != baseline {
		t.Fatalf("unchanged build identity changed: before=%+v after=%+v", baseline, reused)
	}

	snapshot, err := loadHelperSourceSnapshot(sourceDir)
	if err != nil {
		t.Fatalf("load baseline source snapshot: %v", err)
	}
	for _, name := range snapshot.Files {
		name := name
		t.Run(name, func(t *testing.T) {
			mutatedDir := copyHelperPackageFrom(t, sourceDir)
			path := filepath.Join(mutatedDir, filepath.FromSlash(name))
			f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
			if err != nil {
				t.Fatalf("open %s for mutation: %v", name, err)
			}
			_, writeErr := f.WriteString("\n// source identity mutation\n")
			closeErr := f.Close()
			if writeErr != nil || closeErr != nil {
				t.Fatalf("mutate %s: write=%v close=%v", name, writeErr, closeErr)
			}

			identity, built, err := buildContentAddressedHelper(mutatedDir, outputDir)
			if err != nil {
				t.Fatalf("build after mutating %s: %v", name, err)
			}
			if !built {
				t.Fatalf("changing %s did not force a rebuild", name)
			}
			if identity.SourceDigest == baseline.SourceDigest {
				t.Fatalf("changing %s did not change source digest", name)
			}
			if identity.Path == baseline.Path {
				t.Fatalf("changing %s did not change output path", name)
			}
			assertHelperIdentity(t, identity)
		})
	}
}

func TestHelperRuntimeIdentityMatchesPreBuildDigest(t *testing.T) {
	sourceDir := copyHelperPackage(t)
	identity, built, err := buildContentAddressedHelper(sourceDir, t.TempDir())
	if err != nil {
		t.Fatalf("build helper: %v", err)
	}
	if !built {
		t.Fatal("first helper build unexpectedly reused output")
	}
	assertHelperIdentity(t, identity)
}

func assertHelperIdentity(t *testing.T, want helperBuildIdentity) {
	t.Helper()
	got, err := readHelperIdentity(want.Path)
	if err != nil {
		t.Fatalf("read helper identity: %v", err)
	}
	if got.SourceDigest != want.SourceDigest {
		t.Fatalf("runtime source digest=%q, pre-build digest=%q",
			got.SourceDigest, want.SourceDigest)
	}
	if got.GoVersion == "" || !strings.HasPrefix(got.GoVersion, "go") {
		t.Fatalf("invalid helper build Go version %q", got.GoVersion)
	}
}

func copyHelperPackage(t *testing.T) string {
	t.Helper()
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("find repository root: %v", err)
	}
	return copyHelperPackageFrom(t,
		filepath.Join(repoRoot, "internal/execution/testdata/testhelper"))
}

func copyHelperPackageFrom(t *testing.T, sourceDir string) string {
	t.Helper()
	snapshot, err := loadHelperSourceSnapshot(sourceDir)
	if err != nil {
		t.Fatalf("discover helper package sources: %v", err)
	}
	destination := t.TempDir()
	for _, name := range snapshot.Files {
		sourcePath := filepath.Join(sourceDir, filepath.FromSlash(name))
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatalf("read %s: %v", sourcePath, err)
		}
		destinationPath := filepath.Join(destination, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
			t.Fatalf("create source directory: %v", err)
		}
		if err := os.WriteFile(destinationPath, data, 0o644); err != nil {
			t.Fatalf("copy %s: %v", name, err)
		}
	}
	return destination
}

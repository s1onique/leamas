package closure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func artifactPlan(path string, maximum int64) PlanArtifact {
	return PlanArtifact{ID: "artifact", Path: path, Required: boolPtr(true), MaxBytes: maximum, MediaType: "application/octet-stream"}
}

func TestClosureArtifactRequiresRegularFile(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "directory"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := hashRepositoryArtifact(root, artifactPlan("directory", 100), nil)
	assertArtifactError(t, err, "regular")
}

func TestClosureArtifactRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "real"), []byte("value"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	_, err := hashRepositoryArtifact(root, artifactPlan("link", 100), nil)
	assertArtifactError(t, err, "symlink")
}

func TestClosureArtifactRejectsSymlinkParent(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "value"), []byte("value"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "parent")); err != nil {
		t.Fatal(err)
	}
	_, err := hashRepositoryArtifact(root, artifactPlan("parent/value", 100), nil)
	assertArtifactError(t, err, "symlink")
}

func TestClosureArtifactRejectsPathEscape(t *testing.T) {
	root := t.TempDir()
	_, err := hashRepositoryArtifact(root, artifactPlan("../outside", 100), nil)
	assertArtifactError(t, err, "repository")
}

func TestClosureArtifactEnforcesMaximumBytes(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "large"), []byte("too large"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := hashRepositoryArtifact(root, artifactPlan("large", 2), nil)
	assertArtifactError(t, err, "maximum")
}

func TestClosureArtifactHashesOpenedFile(t *testing.T) {
	root := t.TempDir()
	content := []byte("stable")
	if err := os.WriteFile(filepath.Join(root, "value"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := hashRepositoryArtifact(root, artifactPlan("value", 100), nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.SHA256 != SHA256Hex(content) || result.ByteCount != int64(len(content)) {
		t.Fatalf("result = %+v", result)
	}
}

func TestClosureArtifactRejectsMutationDuringHash(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "value")
	if err := os.WriteFile(path, []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := hashRepositoryArtifact(root, artifactPlan("value", 100), func(stage string) {
		if stage == "between_hashes" {
			if writeErr := os.WriteFile(path, []byte("after!"), 0o644); writeErr != nil {
				t.Fatalf("mutate artifact: %v", writeErr)
			}
		}
	})
	assertArtifactError(t, err, "changed")
}

func TestClosureEvidenceDirectoryMustBeDetached(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "evidence")
	if _, err := prepareEvidenceDirectory(root, inside); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("inside evidence error = %v", err)
	}
	outside := filepath.Join(t.TempDir(), "evidence")
	resolved, err := prepareEvidenceDirectory(root, outside)
	if err != nil || resolved == "" {
		t.Fatalf("outside evidence = %q, %v", resolved, err)
	}
}

func assertArtifactError(t *testing.T, err error, contains string) {
	t.Helper()
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(contains)) {
		t.Fatalf("error = %v, want containing %q", err, contains)
	}
}

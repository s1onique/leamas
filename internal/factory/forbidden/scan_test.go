package forbidden

import (
	"testing"
)

// TestScanBoundaryContract verifies the explicit scan boundary contract.
// SCAN: cmd/, internal/ (except internal/factory/), scripts/, githooks/
// ALLOW: internal/factory/, docs/doctrine/, docs/adr/, docs/factory/, docs/close-reports/, *_test.go, testdata/
func TestScanBoundaryContract(t *testing.T) {
	tests := []struct {
		name        string
		relPath     string
		shouldScan  bool
		description string
	}{
		// SCAN: cmd/
		{"cmd/main.go", "cmd/main.go", true, "cmd/ is in scan scope"},
		// SCAN: internal/ (except internal/factory/)
		{"internal/app/foo.go", "internal/app/foo.go", true, "internal/app/ is in scan scope"},
		{"internal/ui/bar.go", "internal/ui/bar.go", true, "internal/ui/ is in scan scope"},
		{"internal/factory/auth.go", "internal/factory/auth.go", false, "internal/factory/ is EXCLUDED"},
		{"internal/factory/forbidden/check.go", "internal/factory/forbidden/check.go", false, "internal/factory/ is EXCLUDED"},
		// SCAN: scripts/
		{"scripts/build.sh", "scripts/build.sh", true, "scripts/ is in scan scope"},
		// SCAN: githooks/
		{"githooks/pre-push", "githooks/pre-push", true, "githooks/ is in scan scope"},
		// ALLOW: docs/doctrine/
		{"docs/doctrine/test.md", "docs/doctrine/test.md", false, "docs/doctrine/ is ALLOWED"},
		// ALLOW: docs/adr/
		{"docs/adr/0001-test.md", "docs/adr/0001-test.md", false, "docs/adr/ is ALLOWED"},
		// ALLOW: docs/factory/
		{"docs/factory/test.md", "docs/factory/test.md", false, "docs/factory/ is ALLOWED"},
		// ALLOW: docs/close-reports/
		{"docs/close-reports/test.md", "docs/close-reports/test.md", false, "docs/close-reports/ is ALLOWED"},
		// ALLOW: *_test.go files
		{"internal/app/foo_test.go", "internal/app/foo_test.go", false, "_test.go files are EXCLUDED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inAllowed := isInAllowedDir(tt.relPath)
			isInternalFactory := len(tt.relPath) >= len("internal/factory") &&
				(tt.relPath[:len("internal/factory")] == "internal/factory" ||
					tt.relPath[:len("internal\\factory")] == "internal\\factory")
			isTestFile := len(tt.relPath) >= 8 && tt.relPath[len(tt.relPath)-8:] == "_test.go"
			shouldScan := !inAllowed && !isInternalFactory && !isTestFile

			if shouldScan != tt.shouldScan {
				t.Errorf("scan boundary contract: %s: got shouldScan=%v, want %v (%s)",
					tt.relPath, shouldScan, tt.shouldScan, tt.description)
			}
		})
	}
}

func TestScanDirsAreExported(t *testing.T) {
	expectedDirs := []string{"cmd", "internal", "scripts", "githooks"}
	for _, expected := range expectedDirs {
		found := false
		for _, actual := range ScanDirs {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ScanDirs missing: %s", expected)
		}
	}
}

func TestAllowedDirsAreExported(t *testing.T) {
	expectedDirs := []string{"internal/factory", "docs/doctrine", "docs/adr", "docs/factory", "docs/close-reports", "testdata"}
	for _, expected := range expectedDirs {
		found := false
		for _, actual := range AllowedDirs {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AllowedDirs missing: %s", expected)
		}
	}
}

func TestScanFilesAreExported(t *testing.T) {
	expectedFiles := []string{"AGENTS.md", ".clinerules/leamas.md"}
	for _, expected := range expectedFiles {
		found := false
		for _, actual := range ScanFiles {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ScanFiles missing: %s", expected)
		}
	}
}

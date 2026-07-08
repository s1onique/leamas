package forbidden

import (
	"testing"
)

// TestScanBoundaryContract verifies the explicit scan boundary contract.
// SCAN: cmd/ (.go non-test), internal/ (.go non-test except factory), scripts/**, githooks/**
// ALLOW: internal/factory/, docs/doctrine/, docs/adr/, docs/factory/, docs/close-reports/, *_test.go, testdata/, AGENTS.md, .clinerules/
func TestScanBoundaryContract(t *testing.T) {
	tests := []struct {
		name        string
		relPath     string
		shouldScan  bool
		description string
	}{
		// SCAN: cmd/ - Go production files only
		{"cmd/main.go", "cmd/main.go", true, "cmd/ .go files are in scan scope"},
		{"cmd/test.sh", "cmd/test.sh", false, "cmd/ non-go files are NOT scanned"},
		// SCAN: internal/ (except internal/factory/) - Go production files only
		{"internal/app/foo.go", "internal/app/foo.go", true, "internal/app/ .go files are in scan scope"},
		{"internal/ui/bar.go", "internal/ui/bar.go", true, "internal/ui/ .go files are in scan scope"},
		{"internal/factory/auth.go", "internal/factory/auth.go", false, "internal/factory/ is EXCLUDED"},
		{"internal/factory/forbidden/check.go", "internal/factory/forbidden/check.go", false, "internal/factory/ is EXCLUDED"},
		{"internal/app/readme.txt", "internal/app/readme.txt", false, "internal/ non-go files are NOT scanned"},
		// SCAN: scripts/** - all text files
		{"scripts/build.sh", "scripts/build.sh", true, "scripts/ is in scan scope"},
		{"scripts/test.sh", "scripts/test.sh", true, "scripts/ all files are scanned"},
		// SCAN: githooks/** - all text files
		{"githooks/pre-push", "githooks/pre-push", true, "githooks/ is in scan scope"},
		{"githooks/pre-commit", "githooks/pre-commit", true, "githooks/ all files are scanned"},
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
		// ALLOW: AGENTS.md - policy document
		{"AGENTS.md", "AGENTS.md", false, "AGENTS.md is ALLOWED (policy document)"},
		// ALLOW: .clinerules/ - policy documents
		{".clinerules/leamas.md", ".clinerules/leamas.md", false, ".clinerules/ is ALLOWED (policy documents)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldScan := shouldScanFile(tt.relPath)
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
	// ScanFiles is empty because AGENTS.md and .clinerules/ are in AllowedDirs (policy documents)
	if len(ScanFiles) != 0 {
		t.Errorf("ScanFiles should be empty, got %v", ScanFiles)
	}
}

package llmfriendly

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxBytes != 64*1024 {
		t.Errorf("expected MaxBytes 64*1024, got %d", cfg.MaxBytes)
	}
	if cfg.MaxLines != 400 {
		t.Errorf("expected MaxLines 400, got %d", cfg.MaxLines)
	}
	if cfg.MaxLineLength != 240 {
		t.Errorf("expected MaxLineLength 240, got %d", cfg.MaxLineLength)
	}
	if cfg.MinifiedLineLength != 1000 {
		t.Errorf("expected MinifiedLineLength 1000, got %d", cfg.MinifiedLineLength)
	}
}

func TestIsBinary(t *testing.T) {
	tmpDir := t.TempDir()

	binaryPath := filepath.Join(tmpDir, "binary.bin")
	if err := os.WriteFile(binaryPath, []byte{0x00, 0x01, 0x02}, 0644); err != nil {
		t.Fatal(err)
	}
	if !isBinary(binaryPath) {
		t.Error("expected file with NUL byte to be detected as binary")
	}

	textPath := filepath.Join(tmpDir, "text.txt")
	if err := os.WriteFile(textPath, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	if isBinary(textPath) {
		t.Error("expected text file to not be detected as binary")
	}
}

func TestIsMinifiableFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"file.js", true},
		{"file.css", true},
		{"file.html", true},
		{"file.json", true},
		{"file.xml", true},
		{"file.svg", true},
		{"file.min.js", true},
		{"file.min.css", true},
		{"file.go", false},
		{"file.md", false},
		{"file.sh", false},
		{"file.txt", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			result := isMinifiableFile(tc.path)
			if result != tc.expected {
				t.Errorf("isMinifiableFile(%q) = %v, want %v", tc.path, result, tc.expected)
			}
		})
	}
}

func TestIsCanonicalClosurePlan(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		// Accept: canonical closure-plan JSON files
		{"docs/closure-plans/plan.json", true},
		{"docs/closure-plans/subdir/plan.json", true},
		{"/absolute/path/docs/closure-plans/plan.json", true},
		{"DOCS/CLOSURE-PLANS/PLAN.JSON", true}, // case-insensitive extension

		// Reject: not under docs/closure-plans/
		{"docs/other/plan.json", false},
		{"docs/closure-manifests/plan.json", false},
		{"file.json", false},
		{"tmp/closure-plans/plan.json", false},

		// Reject: not .json extension
		{"docs/closure-plans/plan.txt", false},
		{"docs/closure-plans/plan.js", false},
		{"docs/closure-plans/readme.md", false},

		// Reject: similar but wrong paths
		{"docs/not-closure-plans/file.json", false},
		{"docs/closure-plans-archive/file.json", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			result := isCanonicalClosurePlan(tc.path)
			if result != tc.expected {
				t.Errorf("isCanonicalClosurePlan(%q) = %v, want %v", tc.path, result, tc.expected)
			}
		})
	}
}

func TestCheckRepo_SmallFiles(t *testing.T) {
	tmpDir := t.TempDir()

	textPath := filepath.Join(tmpDir, "small.txt")
	if err := os.WriteFile(textPath, []byte("hello world\n"), 0644); err != nil {
		t.Fatal(err)
	}

	runGitCommand(tmpDir, "init")
	runGitCommand(tmpDir, "config", "user.email", "test@test.com")
	runGitCommand(tmpDir, "config", "user.name", "Test")
	runGitCommand(tmpDir, "add", ".")
	runGitCommand(tmpDir, "commit", "-m", "add small file")

	cfg := DefaultConfig()
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo error: %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("expected no findings for small file, got %d", len(findings))
	}
}

func TestCheckRepo_LongLineAllowedInCanonicalClosurePlan(t *testing.T) {
	tmpDir := t.TempDir()

	planPath := filepath.Join(tmpDir, "docs", "closure-plans", "test-plan.json")
	if err := os.MkdirAll(filepath.Dir(planPath), 0755); err != nil {
		t.Fatal(err)
	}

	longLine := make([]byte, 300)
	for i := range longLine {
		longLine[i] = 'a'
	}
	content := `{"command": "` + string(longLine) + `"}\n`
	if err := os.WriteFile(planPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	runGitCommand(tmpDir, "init")
	runGitCommand(tmpDir, "config", "user.email", "test@test.com")
	runGitCommand(tmpDir, "config", "user.name", "Test")
	runGitCommand(tmpDir, "add", ".")
	runGitCommand(tmpDir, "commit", "-m", "add closure plan")

	cfg := DefaultConfig()
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo error: %v", err)
	}

	for _, f := range findings {
		if f.Kind == "long_line" && strings.Contains(f.Path, "test-plan.json") {
			t.Errorf("long_line finding in canonical closure plan should be allowed, got: %v", f)
		}
	}
}

func TestCheckRepo_LongLineNotAllowedInOtherDocs(t *testing.T) {
	tmpDir := t.TempDir()

	docPath := filepath.Join(tmpDir, "docs", "readme.txt")
	if err := os.MkdirAll(filepath.Dir(docPath), 0755); err != nil {
		t.Fatal(err)
	}

	longLine := make([]byte, 300)
	for i := range longLine {
		longLine[i] = 'a'
	}
	longLine = append(longLine, '\n')
	if err := os.WriteFile(docPath, longLine, 0644); err != nil {
		t.Fatal(err)
	}

	runGitCommand(tmpDir, "init")
	runGitCommand(tmpDir, "config", "user.email", "test@test.com")
	runGitCommand(tmpDir, "config", "user.name", "Test")
	runGitCommand(tmpDir, "add", ".")
	runGitCommand(tmpDir, "commit", "-m", "add doc")

	cfg := DefaultConfig()
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Kind == "long_line" && strings.Contains(f.Path, "docs/readme.txt") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected long_line finding for regular doc")
	}
}

func TestCheckRepo_ClosurePlanTooLargeStillFails(t *testing.T) {
	tmpDir := t.TempDir()

	planPath := filepath.Join(tmpDir, "docs", "closure-plans", "large-plan.json")
	if err := os.MkdirAll(filepath.Dir(planPath), 0755); err != nil {
		t.Fatal(err)
	}

	largeContent := make([]byte, 70*1024)
	for i := range largeContent {
		largeContent[i] = 'a'
	}
	if err := os.WriteFile(planPath, largeContent, 0644); err != nil {
		t.Fatal(err)
	}

	runGitCommand(tmpDir, "init")
	runGitCommand(tmpDir, "config", "user.email", "test@test.com")
	runGitCommand(tmpDir, "config", "user.name", "Test")
	runGitCommand(tmpDir, "add", ".")
	runGitCommand(tmpDir, "commit", "-m", "add large closure plan")

	cfg := DefaultConfig()
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Kind == "too_large" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected too_large finding for oversized closure plan")
	}
}

func TestCheckRepo_ClosurePlanTooManyLinesStillFails(t *testing.T) {
	tmpDir := t.TempDir()

	planPath := filepath.Join(tmpDir, "docs", "closure-plans", "many-lines.json")
	if err := os.MkdirAll(filepath.Dir(planPath), 0755); err != nil {
		t.Fatal(err)
	}

	// Create content with more than 400 lines
	lines := make([]byte, 0, 450*10)
	for i := 0; i < 450; i++ {
		lines = append(lines, []byte("line content\n")...)
	}
	if err := os.WriteFile(planPath, lines, 0644); err != nil {
		t.Fatal(err)
	}

	runGitCommand(tmpDir, "init")
	runGitCommand(tmpDir, "config", "user.email", "test@test.com")
	runGitCommand(tmpDir, "config", "user.name", "Test")
	runGitCommand(tmpDir, "add", ".")
	runGitCommand(tmpDir, "commit", "-m", "add plan")

	cfg := DefaultConfig()
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Kind == "too_many_lines" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected too_many_lines finding for plan with many lines")
	}
}

func TestCheckRepo_NonCanonicalClosurePlanStillFails(t *testing.T) {
	tmpDir := t.TempDir()

	planPath := filepath.Join(tmpDir, "docs", "closure-plans", "plan.txt")
	if err := os.MkdirAll(filepath.Dir(planPath), 0755); err != nil {
		t.Fatal(err)
	}

	longLine := make([]byte, 300)
	for i := range longLine {
		longLine[i] = 'a'
	}
	longLine = append(longLine, '\n')
	if err := os.WriteFile(planPath, longLine, 0644); err != nil {
		t.Fatal(err)
	}

	runGitCommand(tmpDir, "init")
	runGitCommand(tmpDir, "config", "user.email", "test@test.com")
	runGitCommand(tmpDir, "config", "user.name", "Test")
	runGitCommand(tmpDir, "add", ".")
	runGitCommand(tmpDir, "commit", "-m", "add plan")

	cfg := DefaultConfig()
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Kind == "long_line" && strings.Contains(f.Path, "plan.txt") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected long_line finding for non-canonical closure plan file")
	}
}

func TestCheckRepo_BinaryFilesSkipped(t *testing.T) {
	tmpDir := t.TempDir()

	binaryPath := filepath.Join(tmpDir, "binary.dat")
	if err := os.WriteFile(binaryPath, []byte{0x00, 0x01, 0x02, 0xFF}, 0644); err != nil {
		t.Fatal(err)
	}

	runGitCommand(tmpDir, "init")
	runGitCommand(tmpDir, "config", "user.email", "test@test.com")
	runGitCommand(tmpDir, "config", "user.name", "Test")
	runGitCommand(tmpDir, "add", ".")
	runGitCommand(tmpDir, "commit", "-m", "add binary")

	cfg := DefaultConfig()
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo error: %v", err)
	}

	for _, f := range findings {
		if strings.Contains(f.Path, "binary.dat") {
			t.Errorf("binary file should be skipped, got finding: %v", f)
		}
	}
}

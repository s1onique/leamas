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
	// Create temp dir
	tmpDir := t.TempDir()

	// Test binary file with NUL byte
	binaryPath := filepath.Join(tmpDir, "binary.bin")
	if err := os.WriteFile(binaryPath, []byte{0x00, 0x01, 0x02}, 0644); err != nil {
		t.Fatal(err)
	}
	if !isBinary(binaryPath) {
		t.Error("expected file with NUL byte to be detected as binary")
	}

	// Test text file
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

func TestIsClosurePlanCommandString(t *testing.T) {
	tests := []struct {
		path     string
		lineLen  int
		expected bool
	}{
		// Closure plans with long lines are allowed
		{"docs/closure-plans/plan.json", 300, true},
		{"docs/closure-plans/subdir/plan.json", 500, true},
		{"docs/closure-plans/plan.json", 240, false},  // exactly 240 is not > 240
		{"docs/closure-plans/plan.json", 200, false},  // under 240

		// Non-closure-plan files are not allowed
		{"docs/other/plan.json", 300, false},
		{"docs/closure-manifests/plan.json", 300, false},
		{"file.json", 300, false},
		{"file.txt", 300, false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			result := isClosurePlanCommandString(tc.path, tc.lineLen)
			if result != tc.expected {
				t.Errorf("isClosurePlanCommandString(%q, %d) = %v, want %v",
					tc.path, tc.lineLen, result, tc.expected)
			}
		})
	}
}

func TestCheckRepo_SmallFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a small text file
	textPath := filepath.Join(tmpDir, "small.txt")
	if err := os.WriteFile(textPath, []byte("hello world\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
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

func TestCheckRepo_LongLineAllowedInClosurePlan(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a closure plan with a long line (> 240 chars)
	planPath := filepath.Join(tmpDir, "docs", "closure-plans", "test-plan.json")
	if err := os.MkdirAll(filepath.Dir(planPath), 0755); err != nil {
		t.Fatal(err)
	}

	// Create a line that's 300 chars long
	longLine := make([]byte, 300)
	for i := range longLine {
		longLine[i] = 'a'
	}
	content := `{"command": "` + string(longLine) + `"}\n`
	if err := os.WriteFile(planPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
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

	// Long line in closure plan should be allowed
	for _, f := range findings {
		if f.Kind == "long_line" {
			t.Errorf("long_line finding in closure plan should be allowed, got: %v", f)
		}
	}
}

func TestCheckRepo_LongLineNotAllowedInOtherDocs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a regular doc with a long line (> 240 chars)
	docPath := filepath.Join(tmpDir, "docs", "readme.txt")
	if err := os.MkdirAll(filepath.Dir(docPath), 0755); err != nil {
		t.Fatal(err)
	}

	// Create a line that's 300 chars long
	longLine := make([]byte, 300)
	for i := range longLine {
		longLine[i] = 'a'
	}
	longLine = append(longLine, '\n')
	if err := os.WriteFile(docPath, longLine, 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
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

	// Long line in regular doc should be flagged
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

func TestCheckRepo_ClosurePlanOtherFindingsStillFail(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a closure plan that's too large (> 64 KiB)
	planPath := filepath.Join(tmpDir, "docs", "closure-plans", "large-plan.json")
	if err := os.MkdirAll(filepath.Dir(planPath), 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file larger than MaxBytes (64 KiB)
	largeContent := make([]byte, 70*1024) // 70 KiB
	for i := range largeContent {
		largeContent[i] = 'a'
	}
	if err := os.WriteFile(planPath, largeContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
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

	// Too-large finding should still be flagged
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

func TestCheckRepo_SortedFindings(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple files
	files := map[string]string{
		"zzz_file.txt": "content\n",
		"aaa_file.txt": "content\n",
		"mmm_file.txt": "content\n",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Initialize git repo
	runGitCommand(tmpDir, "init")
	runGitCommand(tmpDir, "config", "user.email", "test@test.com")
	runGitCommand(tmpDir, "config", "user.name", "Test")
	runGitCommand(tmpDir, "add", ".")
	runGitCommand(tmpDir, "commit", "-m", "add files")

	// Modify files to create findings (long lines)
	for name := range files {
		path := filepath.Join(tmpDir, name)
		longContent := make([]byte, 300)
		for i := range longContent {
			longContent[i] = 'a'
		}
		longContent = append(longContent, '\n')
		if err := os.WriteFile(path, longContent, 0644); err != nil {
			t.Fatal(err)
		}
		runGitCommand(tmpDir, "add", name)
		runGitCommand(tmpDir, "commit", "-m", "update "+name)
	}

	cfg := DefaultConfig()
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo error: %v", err)
	}

	// Verify sorted order
	for i := 1; i < len(findings); i++ {
		if findings[i].Path < findings[i-1].Path {
			t.Errorf("findings not sorted: %q before %q", findings[i].Path, findings[i-1].Path)
		}
	}
}

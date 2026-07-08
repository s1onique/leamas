package llmfriendly

import (
	"os"
	"path/filepath"
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

func TestCheckRepo_SmallFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a small text file
	smallFile := filepath.Join(tmpDir, "small.txt")
	if err := os.WriteFile(smallFile, []byte("hello\nworld\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	runGitCommand(tmpDir, "init")
	runGitCommand(tmpDir, "config", "user.email", "test@test.com")
	runGitCommand(tmpDir, "config", "user.name", "Test")
	runGitCommand(tmpDir, "add", "small.txt")
	runGitCommand(tmpDir, "commit", "-m", "add small file")

	cfg := DefaultConfig()
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo error: %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("expected no findings for small file, got %v", findings)
	}
}

func TestCheckRepo_TooManyLines(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with too many lines
	manyLines := filepath.Join(tmpDir, "many_lines.txt")
	var content []byte
	for i := 0; i < 450; i++ {
		content = append(content, []byte("line\n")...)
	}
	if err := os.WriteFile(manyLines, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	runGitCommand(tmpDir, "init")
	runGitCommand(tmpDir, "config", "user.email", "test@test.com")
	runGitCommand(tmpDir, "config", "user.name", "Test")
	runGitCommand(tmpDir, "add", "many_lines.txt")
	runGitCommand(tmpDir, "commit", "-m", "add many lines")

	cfg := DefaultConfig()
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo error: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "too_many_lines" {
		t.Errorf("expected kind 'too_many_lines', got %q", findings[0].Kind)
	}
}

func TestCheckRepo_LongLine(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with a long line
	longLine := filepath.Join(tmpDir, "long.txt")
	longContent := make([]byte, 300)
	for i := range longContent {
		longContent[i] = 'a'
	}
	longContent = append(longContent, '\n')
	if err := os.WriteFile(longLine, longContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	runGitCommand(tmpDir, "init")
	runGitCommand(tmpDir, "config", "user.email", "test@test.com")
	runGitCommand(tmpDir, "config", "user.name", "Test")
	runGitCommand(tmpDir, "add", "long.txt")
	runGitCommand(tmpDir, "commit", "-m", "add long line")

	cfg := DefaultConfig()
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Kind == "long_line" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find long_line finding")
	}
}

func TestCheckRepo_MinifiedLine(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a JSON file with minified-looking line
	jsonFile := filepath.Join(tmpDir, "data.json")
	// Create a line longer than MinifiedLineLength
	minifiedContent := make([]byte, 1100)
	for i := range minifiedContent {
		minifiedContent[i] = '"'
	}
	minifiedContent = append(minifiedContent, '\n')
	if err := os.WriteFile(jsonFile, minifiedContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	runGitCommand(tmpDir, "init")
	runGitCommand(tmpDir, "config", "user.email", "test@test.com")
	runGitCommand(tmpDir, "config", "user.name", "Test")
	runGitCommand(tmpDir, "add", "data.json")
	runGitCommand(tmpDir, "commit", "-m", "add json")

	cfg := DefaultConfig()
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Kind == "minified_line" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find minified_line finding")
	}
}

func TestCheckRepo_SkipsBinary(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a binary file
	binaryFile := filepath.Join(tmpDir, "binary.bin")
	if err := os.WriteFile(binaryFile, []byte{0x00, 0x01, 0x02}, 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	runGitCommand(tmpDir, "init")
	runGitCommand(tmpDir, "config", "user.email", "test@test.com")
	runGitCommand(tmpDir, "config", "user.name", "Test")
	runGitCommand(tmpDir, "add", "binary.bin")
	runGitCommand(tmpDir, "commit", "-m", "add binary")

	cfg := DefaultConfig()
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo error: %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("expected no findings for binary file, got %v", findings)
	}
}

func TestCheckRepo_SkipsIgnoredDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .gitignore to ignore the test directories
	gitignore := filepath.Join(tmpDir, ".gitignore")
	gitignoreContent := ".git\nbuild\nbin\nvendor\n"
	if err := os.WriteFile(gitignore, []byte(gitignoreContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create files in ignored directories
	ignoredPaths := []string{
		filepath.Join(tmpDir, ".git", "config"),
		filepath.Join(tmpDir, "build", "output"),
		filepath.Join(tmpDir, "bin", "leamas"),
		filepath.Join(tmpDir, "vendor", "module.go"),
	}

	for _, p := range ignoredPaths {
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		// Create non-trivial content so size check would fail if processed
		content := make([]byte, 100000) // 100KB - well over limit
		for i := range content {
			content[i] = 'x'
		}
		if err := os.WriteFile(p, content, 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Initialize git repo
	runGitCommand(tmpDir, "init")
	runGitCommand(tmpDir, "config", "user.email", "test@test.com")
	runGitCommand(tmpDir, "config", "user.name", "Test")
	runGitCommand(tmpDir, "add", ".")
	runGitCommand(tmpDir, "commit", "-m", "add ignored dirs")

	cfg := DefaultConfig()
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo error: %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("expected no findings for ignored dirs, got %v", findings)
	}
}

func TestCheckRepo_TooLarge(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file larger than MaxBytes (64KB)
	largeFile := filepath.Join(tmpDir, "large.bin")
	largeContent := make([]byte, 70*1024) // 70KB
	for i := range largeContent {
		largeContent[i] = 'x'
	}
	if err := os.WriteFile(largeFile, largeContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize git repo
	runGitCommand(tmpDir, "init")
	runGitCommand(tmpDir, "config", "user.email", "test@test.com")
	runGitCommand(tmpDir, "config", "user.name", "Test")
	runGitCommand(tmpDir, "add", "large.bin")
	runGitCommand(tmpDir, "commit", "-m", "add large file")

	cfg := DefaultConfig()
	findings, err := CheckRepo(tmpDir, cfg)
	if err != nil {
		t.Fatalf("CheckRepo error: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Kind != "too_large" {
		t.Errorf("expected kind 'too_large', got %q", findings[0].Kind)
	}
}

func TestFindingsSorted(t *testing.T) {
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

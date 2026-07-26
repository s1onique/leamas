package llmfriendly

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxBytes != 64*1024 || cfg.MaxLines != 400 || cfg.MaxLineLength != 240 || cfg.MinifiedLineLength != 1000 {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestIsBinary(t *testing.T) {
	tmpDir := t.TempDir()
	if !isBinary(writeFile(tmpDir, "b.bin", []byte{0x00, 0x01})) {
		t.Error("NUL byte not detected")
	}
	if isBinary(writeFile(tmpDir, "t.txt", []byte("hello"))) {
		t.Error("text detected as binary")
	}
}

func TestIsMinifiableFile(t *testing.T) {
	for p, want := range map[string]bool{"f.js": true, "f.css": true, "f.json": true, "f.go": false, "f.md": false} {
		if got := isMinifiableFile(p); got != want {
			t.Errorf("isMinifiableFile(%s)=%v, want %v", p, got, want)
		}
	}
}

func TestIsCanonicalClosurePlan(t *testing.T) {
	for p, want := range map[string]bool{
		"docs/closure-plans/plan.json":     true,
		"docs/closure-plans/sub/plan.json": true,
		"DOCS/CLOSURE-PLANS/PLAN.JSON":     true,
		"docs/other/plan.json":             false,
		"docs/closure-manifests/plan.json": false,
		"tmp/closure-plans/plan.json":      false,
		"vendor/docs/closure-plans/x.json": false,
		"docs/closure-plans/plan.txt":      false,
		"docs/not-closure-plans/file.json": false,
	} {
		if got := isCanonicalClosurePlan(p); got != want {
			t.Errorf("isCanonicalClosurePlan(%q)=%v, want %v", p, got, want)
		}
	}
}

func TestCheckRepo_SmallFiles(t *testing.T) {
	tmpDir := t.TempDir()
	writeFile(tmpDir, "small.txt", []byte("hello\n"))
	gitInit(tmpDir)
	if findings, err := CheckRepo(tmpDir, DefaultConfig()); err != nil || len(findings) != 0 {
		t.Fatalf("expected no findings, got %d: %v", len(findings), err)
	}
}

func TestCheckRepo_LongLineAllowedInCanonicalClosurePlan(t *testing.T) {
	tmpDir := t.TempDir()
	writePlan(tmpDir, "docs/closure-plans/p.json", `{"cmd":"`+strings.Repeat("x", 300)+`"}`)
	gitInit(tmpDir)
	findings, _ := CheckRepo(tmpDir, DefaultConfig())
	for _, f := range findings {
		if f.Kind == "long_line" && strings.Contains(f.Path, "p.json") {
			t.Errorf("closure plan long_line should be exempt: %v", f)
		}
	}
}

func TestCheckRepo_LongLineNotAllowedInOtherDocs(t *testing.T) {
	tmpDir := t.TempDir()
	writeLongFile(tmpDir, "docs/readme.txt", 300)
	gitInit(tmpDir)
	findings, _ := CheckRepo(tmpDir, DefaultConfig())
	if !hasFinding(findings, "long_line", "docs/readme.txt") {
		t.Error("expected long_line for regular doc")
	}
}

func TestCheckRepo_ClosurePlanTooLargeStillFails(t *testing.T) {
	tmpDir := t.TempDir()
	writeLargeFile(tmpDir, "docs/closure-plans/large.json")
	gitInit(tmpDir)
	findings, _ := CheckRepo(tmpDir, DefaultConfig())
	if !hasFinding(findings, "too_large", "") {
		t.Error("expected too_large for large closure plan")
	}
}

func TestCheckRepo_ClosurePlanTooManyLinesStillFails(t *testing.T) {
	tmpDir := t.TempDir()
	writePlan(tmpDir, "docs/closure-plans/many.json", "")
	path := filepath.Join(tmpDir, "docs", "closure-plans", "many.json")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 450; i++ {
		f.WriteString("line\n")
	}
	f.Close()
	gitInit(tmpDir)
	findings, _ := CheckRepo(tmpDir, DefaultConfig())
	if !hasFinding(findings, "too_many_lines", "") {
		t.Error("expected too_many_lines for plan")
	}
}

func TestCheckRepo_NonCanonicalClosurePlanStillFails(t *testing.T) {
	tmpDir := t.TempDir()
	writeLongFile(tmpDir, "docs/closure-plans/plan.txt", 300)
	gitInit(tmpDir)
	findings, _ := CheckRepo(tmpDir, DefaultConfig())
	if !hasFinding(findings, "long_line", "plan.txt") {
		t.Error("expected long_line for non-canonical closure plan")
	}
}

func TestCheckRepo_MinifiedLineInOrdinaryJSONFails(t *testing.T) {
	tmpDir := t.TempDir()
	writeFile(tmpDir, "data.json", []byte(`{"d":"`+strings.Repeat("x", 1200)+`"}`+"\n"))
	gitInit(tmpDir)
	findings, _ := CheckRepo(tmpDir, DefaultConfig())
	if !hasFinding(findings, "minified_line", "") {
		t.Error("expected minified_line for ordinary JSON")
	}
}

func TestCheckRepo_MinifiedLineInClosurePlanAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	cmd := `set -euo pipefail; ` + strings.Repeat("x", 1200)
	writePlan(tmpDir, "docs/closure-plans/cmd.json", `{"argv":["bash","-c","`+cmd+`"]}`)
	gitInit(tmpDir)
	findings, _ := CheckRepo(tmpDir, DefaultConfig())
	for _, f := range findings {
		if f.Kind == "minified_line" && strings.Contains(f.Path, "cmd.json") {
			t.Errorf("closure plan minified_line should be exempt: %v", f)
		}
	}
}

func TestCheckRepo_BinaryFilesSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	writeFile(tmpDir, "bin.dat", []byte{0x00, 0x01})
	gitInit(tmpDir)
	findings, _ := CheckRepo(tmpDir, DefaultConfig())
	for _, f := range findings {
		if strings.Contains(f.Path, "bin.dat") {
			t.Errorf("binary should be skipped: %v", f)
		}
	}
}

func TestCheckRepo_SortedFindings(t *testing.T) {
	tmpDir := t.TempDir()
	for _, name := range []string{"zzz.txt", "aaa.txt", "mmm.txt"} {
		writeFile(tmpDir, name, []byte("c\n"))
	}
	gitInit(tmpDir)
	for _, name := range []string{"zzz.txt", "aaa.txt", "mmm.txt"} {
		writeLongFile(tmpDir, name, 300)
		gitAddCommit(tmpDir, name)
	}
	findings, _ := CheckRepo(tmpDir, DefaultConfig())
	for i := 1; i < len(findings); i++ {
		if findings[i].Path < findings[i-1].Path {
			t.Errorf("unsorted: %q < %q", findings[i].Path, findings[i-1].Path)
		}
	}
}

// Helpers

func writeFile(dir, name string, data []byte) string {
	p := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(p), 0755)
	os.WriteFile(p, data, 0644)
	return p
}

func writeLongFile(dir, name string, length int) {
	p := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(p), 0755)
	data := make([]byte, length+1)
	for i := 0; i < length; i++ {
		data[i] = 'a'
	}
	data[length] = '\n'
	os.WriteFile(p, data, 0644)
}

func writeLargeFile(dir, name string) {
	p := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(p), 0755)
	data := make([]byte, 70*1024)
	for i := 0; i < len(data); i++ {
		data[i] = 'a'
	}
	os.WriteFile(p, data, 0644)
}

func writePlan(dir, name, content string) string {
	p := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(p), 0755)
	if content == "" {
		content = `{"checks":[]}`
	}
	os.WriteFile(p, []byte(content), 0644)
	return p
}

func gitInit(dir string) {
	runGitCommand(dir, "init")
	runGitCommand(dir, "config", "user.email", "t@t.t")
	runGitCommand(dir, "config", "user.name", "T")
	runGitCommand(dir, "add", ".")
	runGitCommand(dir, "commit", "-m", "init")
}

func gitAddCommit(dir, name string) {
	runGitCommand(dir, "add", name)
	runGitCommand(dir, "commit", "-m", "upd")
}

func hasFinding(findings []Finding, kind, path string) bool {
	for _, f := range findings {
		if f.Kind == kind && (path == "" || strings.Contains(f.Path, path)) {
			return true
		}
	}
	return false
}

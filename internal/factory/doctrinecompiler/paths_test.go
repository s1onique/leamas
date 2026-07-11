package doctrinecompiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestNormalizeTargetPathAcceptsCanonical verifies the happy path.
func TestNormalizeTargetPathAcceptsCanonical(t *testing.T) {
	cases := []struct {
		in   string
		want TargetPath
	}{
		{"Makefile", "Makefile"},
		{"a/b/c.txt", "a/b/c.txt"},
		{".factory/doctrine.lock.json", ".factory/doctrine.lock.json"},
		{"./Makefile", "Makefile"},
	}
	for _, c := range cases {
		got, err := NormalizeTargetPath(c.in)
		if err != nil {
			t.Fatalf("NormalizeTargetPath(%q) error: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("NormalizeTargetPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestNormalizeTargetPathRejects verifies the rejection cases.
func TestNormalizeTargetPathRejects(t *testing.T) {
	cases := []string{
		"",
		"/abs/path",
		"../escape",
		"foo/../bar",
		"foo/..",
		"foo/./bar/..",
		"a//b",
		"foo\x00bar",
		"foo\\bar",
		".",
	}
	for _, c := range cases {
		_, err := NormalizeTargetPath(c)
		if err == nil {
			t.Errorf("NormalizeTargetPath(%q) expected error, got nil", c)
		}
	}
}

// TestValidatePathUniqueness verifies duplicate detection.
func TestValidatePathUniqueness(t *testing.T) {
	if err := ValidatePathUniqueness(nil); err != nil {
		t.Fatalf("nil set: unexpected error %v", err)
	}
	if err := ValidatePathUniqueness([]TargetPath{"a", "b"}); err != nil {
		t.Fatalf("unique set: unexpected error %v", err)
	}
	err := ValidatePathUniqueness([]TargetPath{"a", "a"})
	if err == nil {
		t.Errorf("duplicate normalized: expected error")
	}
}

// TestResolverContains verifies the containment check.
func TestResolverContains(t *testing.T) {
	tmp := t.TempDir()
	resolver, err := NewResolver(tmp)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	abs := resolver.Resolve("a/b.txt")
	if !resolver.Contains(abs) {
		t.Errorf("Contains(%s) = false, want true", abs)
	}
	if resolver.Contains("/etc/passwd") {
		t.Errorf("Contains(/etc/passwd) = true, want false")
	}
}

// TestResolverHasSymlinkEscapeRejects verifies the symlink-escape
// detector refuses to descend into symlinked parents.
func TestResolverHasSymlinkEscapeRejects(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on windows")
	}
	tmp := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(tmp, "link")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	resolver, err := NewResolver(tmp)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	if sym, ok := resolver.HasSymlinkEscape(TargetPath("link/foo")); !ok || sym == "" {
		t.Errorf("HasSymlinkEscape = (%q,%v), want (path,true)", sym, ok)
	}
}

// TestInspectPathClassifies verifies classification of common shapes.
func TestInspectPathClassifies(t *testing.T) {
	tmp := t.TempDir()
	resolver, err := NewResolver(tmp)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	cases := []struct {
		setup func() error
		path  TargetPath
		want  PathKind
	}{
		{func() error { return nil }, "missing.txt", PathMissing},
		{func() error { return os.WriteFile(filepath.Join(tmp, "file.txt"), []byte("x"), 0o644) }, "file.txt", PathRegularFile},
		{func() error { return os.Mkdir(filepath.Join(tmp, "dir"), 0o755) }, "dir", PathDirectory},
	}
	for _, c := range cases {
		if err := c.setup(); err != nil {
			t.Fatalf("setup: %v", err)
		}
		got, _, err := resolver.InspectPath(c.path)
		if err != nil {
			t.Fatalf("InspectPath(%q): %v", c.path, err)
		}
		if got != c.want {
			t.Errorf("InspectPath(%q) = %d, want %d", c.path, got, c.want)
		}
	}
}

// TestWriteAtomicFileRoundTrip exercises create + update + error paths.
func TestWriteAtomicFileRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "a", "b.txt")
	if _, err := writeAtomicFile(target, []byte("v1"), 0o644); err != nil {
		t.Fatalf("first write: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "v1" {
		t.Fatalf("first write content: %q err=%v", data, err)
	}
	if _, err := writeAtomicFile(target, []byte("v2"), 0o644); err != nil {
		t.Fatalf("update write: %v", err)
	}
	data, _ = os.ReadFile(target)
	if string(data) != "v2" {
		t.Fatalf("update content: %q", data)
	}
	// Refuses to overwrite a directory.
	if err := os.Mkdir(filepath.Join(tmp, "c"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := writeAtomicFile(filepath.Join(tmp, "c"), []byte("v3"), 0o644); err == nil {
		t.Fatalf("expected error overwriting directory")
	}
}

// TestRemoveFileIfExistsRefusesSymlinks ensures the compiler refuses to
// remove a symlink even if it is the recorded managed path.
func TestRemoveFileIfExistsRefusesSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on windows")
	}
	tmp := t.TempDir()
	target := filepath.Join(tmp, "lnk")
	if err := os.Symlink("/etc/hostname", target); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := removeFileIfExists(target); err == nil {
		t.Fatalf("expected remove of symlink to fail")
	}
	if _, err := os.Lstat(target); err != nil {
		t.Fatalf("symlink vanished: %v", err)
	}
}

// TestSameFilesystem verifies the device-id check behaves on the
// current filesystem.
func TestSameFilesystem(t *testing.T) {
	tmp := t.TempDir()
	child := filepath.Join(tmp, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ok, err := SameFilesystem(tmp, child)
	if err != nil {
		t.Fatalf("SameFilesystem: %v", err)
	}
	if !ok {
		t.Errorf("SameFilesystem returned false for tmpdir")
	}
}

// TestEnsureDeterministicNewline documents the canonical newline.
func TestEnsureDeterministicNewline(t *testing.T) {
	if Newline != "\n" {
		t.Errorf("Newline = %q, want \"\\n\"", Newline)
	}
	if strings.Contains(string(Newline), "\r") {
		t.Errorf("Newline contains CR")
	}
}

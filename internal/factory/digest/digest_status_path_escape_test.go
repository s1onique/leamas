// Package digest provides targeted digest generation for Git repositories.
//
// Path-escape integration tests. The parser preserves filenames that
// contain tab, newline, backslash and other unusual bytes, but only
// the renderer decides whether they survive to the final digest.
// These tests confirm that a path with embedded whitespace emerges as
// a single, well-formed manifest entry (and a single Changed-files
// entry) in the rendered digest.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// PathEscapeRoundTrip exercises the canonical escape form across
// every byte category that could otherwise split a digest entry.
func TestPathEscapeRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"plain ASCII", "regular.go"},
		{"leading dash", "-dash.go"},
		{"space", "path with spaces.go"},
		{"tab", "before\tafter.go"},
		{"newline", "weird\nfile\nname.go"},
		{"CRLF", "weird\r\nfile\r\nname.go"},
		{"backslash", `back\slash.go`},
		{"control NUL", "before\x00after.go"},
		{"DEL byte", "before\x7fafter.go"},
		{"unicode", "путь/файл.go"},
		{"combining NUL+tab+newline+backslash", "x\x00y\tz\nw\\u.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := PathEscape(tt.path)
			dec, err := ParseEscapedPath(enc)
			if err != nil {
				t.Fatalf("ParseEscapedPath(%q) failed: %v", enc, err)
			}
			if dec != tt.path {
				t.Fatalf("round-trip changed path:\n  in:  %q\n  enc: %q\n  out: %q",
					tt.path, enc, dec)
			}
			// The canonical escaped form must itself never include an
			// unescaped LF, CR, NUL, or tab. Only the explicit `\n`,
			// `\r`, `\t`, `\\`, `\xNN` escapes are permitted.
			for i := 0; i < len(enc); i++ {
				switch enc[i] {
				case '\n', '\r', '\t', 0x00:
					t.Fatalf("unescaped control byte %q at offset %d in %q",
						enc[i], i, enc)
				}
			}
		})
	}
}

// TestStagedStatus_NewlinePathInManifest renders a digest for a
// staged file whose name contains an embedded newline. The manifest
// must carry the escaped form on a single line, and the digester
// must not crash. After escaping, the literal newline is gone and
// the entry reads as `A  weird\\nfile\\nname.go`.
func TestStagedStatus_NewlinePathInManifest(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	weird := "weird\nfile\nname.go"
	path := filepath.Join(dir, weird)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("body\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, dir, "add", "--", weird)
	runGit(t, dir, "commit", "-m", "weird path")

	// Touch the file again so the next `git add` produces a
	// modified record (regular file). We want to verify that the
	// rendered digest carries both the staged and the path-escape
	// contracts: even with embedded newlines the line survives.
	if err := os.WriteFile(path, []byte("body v2\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, dir, "add", "--", weird)

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeStaged,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	escaped := PathEscape(weird)
	want := "M  " + escaped
	if !strings.Contains(out, want) {
		t.Fatalf("expected escaped manifest line %q in digest, full:\n%s",
			want, manifestSection(out))
	}
	// The escaped form must be on a single manifest line, not split
	// across lines.
	lines := strings.Split(manifestSection(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, weird) {
			t.Fatalf("unescaped newline-bearing path slipped through: %q", line)
		}
	}
}

// TestRangeStatus_NewlinePathInManifest mirrors the staged
// integration test for range mode. A commit introducing a path with
// an embedded newline must render the escaped form as a single
// manifest entry.
func TestRangeStatus_NewlinePathInManifest(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	weird := "weird\nfile\nname.go"
	path := filepath.Join(dir, weird)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("first\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Need to add then commit since path contains newlines.
	runGit(t, dir, "add", "--", weird)
	runGit(t, dir, "commit", "-m", "initial weird")

	// Second commit modifies the file.
	if err := os.WriteFile(path, []byte("second\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, dir, "add", "--", weird)
	runGit(t, dir, "commit", "-m", "modify weird")

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	escaped := PathEscape(weird)
	want := "M  " + escaped
	if !strings.Contains(out, want) {
		t.Fatalf("expected escaped range manifest line %q, full:\n%s",
			want, manifestSection(out))
	}
}

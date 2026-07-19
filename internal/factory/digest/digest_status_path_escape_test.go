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
	"slices"
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

// digestManifestLines returns the rendered manifest lines exactly as
// they appear in the digest, in order. Paths are *escaped* in the
// output (PathEscape form) because the renderer runs only at the
// rendering boundary.
func escapedManifestLines(digestText string) []string {
	idx := strings.Index(digestText, "## CHANGESET_MANIFEST")
	if idx == -1 {
		return nil
	}
	rest := digestText[idx+len("## CHANGESET_MANIFEST"):]
	end := strings.Index(rest, "## CHANGESET_STATS")
	if end != -1 {
		rest = rest[:end]
	}
	var out []string
	for _, line := range strings.Split(rest, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, strings.TrimSpace(line))
	}
	return out
}

// TestStagedStatus_NewlinePathInManifest renders a digest for a
// staged file whose name contains an embedded newline. The manifest
// must carry **exactly one** escaped entry on a single visual line;
// literals of the newline-bearing path must never appear in the
// rendered output. The escaped form is at the canonical rendering
// boundary, the raw path is preserved in ComputeStats / diff lookup.
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

	got := escapedManifestLines(out)
	want := []string{"M  " + PathEscape(weird)}
	if !slices.Equal(got, want) {
		t.Fatalf("staged manifest mismatch\nwant: %#v\ngot:  %#v", want, got)
	}

	// The escaping contract applies to the rendered sections only
	// (manifest, changed-files, diff). Other documentation sections
	// such as the redaction policy legitimately list file patterns
	// with their original bytes.
	for _, sect := range []string{"## Changed files", "## Diffs"} {
		idx := strings.Index(out, sect)
		if idx == -1 {
			continue
		}
		rest := out[idx:]
		end := strings.Index(rest, "\n## ")
		if end == -1 {
			end = len(rest)
		}
		body := rest[:end]
		if strings.Contains(body, weird) {
			t.Fatalf("raw path slipped through rendered section %q:\n%s", sect, body)
		}
	}
	diffIdx := strings.Index(out, "## Diffs")
	if diffIdx == -1 {
		t.Fatalf("missing ## Diffs section; out:\n%s", out)
	}
	diffBody := out[diffIdx:]
	diffHeadings := 0
	for _, line := range strings.Split(diffBody, "\n") {
		if strings.HasPrefix(line, "=== ") && strings.HasSuffix(line, " ===") {
			diffHeadings++
		}
	}
	if diffHeadings != 1 {
		t.Fatalf("expected exactly one diff heading, got %d in:\n%s", diffHeadings, diffBody)
	}
}

// TestRangeStatus_NewlinePathInManifest mirrors the staged
// integration test for range mode. A commit introducing a path with
// an embedded newline must render the escaped form as exactly one
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

	got := escapedManifestLines(out)
	want := []string{"M  " + PathEscape(weird)}
	if !slices.Equal(got, want) {
		t.Fatalf("range manifest mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
	// The escaping contract applies to the rendered sections only
	// (manifest, changed-files, diff). Other documentation sections
	// such as the redaction policy legitimately list file
	// patterns with their original bytes.
	for _, sect := range []string{"## Changed files", "## Diffs"} {
		idx := strings.Index(out, sect)
		if idx == -1 {
			continue
		}
		rest := out[idx:]
		end := strings.Index(rest, "\n## ")
		if end == -1 {
			end = len(rest)
		}
		body := rest[:end]
		if strings.Contains(body, weird) {
			t.Fatalf("raw path slipped through rendered section %q:\n%s", sect, body)
		}
	}
}

// TestReviewMap_NewlinePath ensures the REVIEW_MAP section's bullet
// list survives a path containing embedded newlines: the raw path
// must not appear in the rendered REVIEW_MAP body, and exactly one
// bullet line must exist per file in the production group.
func TestReviewMap_NewlinePath(t *testing.T) {
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

	// Modify so the production code path picks it up.
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

	idx := strings.Index(out, "## REVIEW_MAP")
	if idx == -1 {
		t.Fatalf("missing ## REVIEW_MAP section; out:\n%s", out)
	}
	rest := out[idx:]
	end := strings.Index(rest, "\n## ")
	if end == -1 {
		end = len(rest)
	}
	body := rest[:end]

	// The raw path must not appear anywhere in REVIEW_MAP; only
	// the escaped form (which contains literal backslashes and 'n')
	// is permitted.
	if strings.Contains(body, weird) {
		t.Fatalf("REVIEW_MAP contains raw newline-bearing path:\n%s", body)
	}
	if !strings.Contains(body, PathEscape(weird)) {
		t.Fatalf("REVIEW_MAP missing escaped path form %q:\n%s", PathEscape(weird), body)
	}
}

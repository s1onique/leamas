// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"strings"
	"testing"
)

// nul is the field separator emitted by `git diff --name-status -z`.
const nul = "\x00"

// joinFields concatenates ordered fields with NUL delimiters so tests can
// build parser inputs without scattering hex literals.
func joinFields(parts ...string) string {
	return strings.Join(parts, nul)
}

func TestParseGitStatusRecords_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []GitChange
		wantErr string // substring
	}{
		{
			name:  "1. modified file",
			input: joinFields("M", "foo.go"),
			want: []GitChange{
				{Kind: KindModified, Path: "foo.go"},
			},
		},
		{
			name:  "2. added file",
			input: joinFields("A", "new.go"),
			want: []GitChange{
				{Kind: KindAdded, Path: "new.go"},
			},
		},
		{
			name:  "3. deleted file",
			input: joinFields("D", "old.go"),
			want: []GitChange{
				{Kind: KindDeleted, Path: "old.go"},
			},
		},
		{
			name:  "4. unmerged file",
			input: joinFields("U", "merged.go"),
			want: []GitChange{
				{Kind: KindUnmerged, Path: "merged.go"},
			},
		},
		{
			name:  "5. rename with R100",
			input: joinFields("R100", "old/path.go", "new/path.go"),
			want: []GitChange{
				{Kind: KindRenamed, OldPath: "old/path.go", Path: "new/path.go"},
			},
		},
		{
			name:  "6. rename with non-100 score",
			input: joinFields("R087", "old.go", "new.go"),
			want: []GitChange{
				{Kind: KindRenamed, OldPath: "old.go", Path: "new.go"},
			},
		},
		{
			name:  "7. copy with C100",
			input: joinFields("C100", "source.go", "copy.go"),
			want: []GitChange{
				{Kind: KindCopied, OldPath: "source.go", Path: "copy.go"},
			},
		},
		{
			name:  "8. copy with non-100 score",
			input: joinFields("C075", "src.go", "dest.go"),
			want: []GitChange{
				{Kind: KindCopied, OldPath: "src.go", Path: "dest.go"},
			},
		},
		{
			name:  "9. paths with spaces",
			input: joinFields("M", "path with spaces.go"),
			want: []GitChange{
				{Kind: KindModified, Path: "path with spaces.go"},
			},
		},
		{
			name:  "10. paths with tabs",
			input: joinFields("A", "tab\tin\tpath.go"),
			want: []GitChange{
				{Kind: KindAdded, Path: "tab\tin\tpath.go"},
			},
		},
		{
			name:  "11. paths with newlines",
			input: joinFields("D", "weird\nnewline\npath.go"),
			want: []GitChange{
				{Kind: KindDeleted, Path: "weird\nnewline\npath.go"},
			},
		},
		{
			name:  "12. unicode paths",
			input: joinFields("A", "путь/файл.go"),
			want: []GitChange{
				{Kind: KindAdded, Path: "путь/файл.go"},
			},
		},
		{
			name:  "13. leading-dash paths",
			input: joinFields("M", "-dashed-flag.go"),
			want: []GitChange{
				{Kind: KindModified, Path: "-dashed-flag.go"},
			},
		},
		{
			name: "14. multiple adjacent records",
			input: joinFields(
				"A", "a.go",
				"M", "b.go",
				"D", "c.go",
				"R100", "d.go", "d2.go",
				"C075", "e.go", "e2.go",
			),
			want: []GitChange{
				{Kind: KindAdded, Path: "a.go"},
				{Kind: KindModified, Path: "b.go"},
				{Kind: KindDeleted, Path: "c.go"},
				{Kind: KindRenamed, OldPath: "d.go", Path: "d2.go"},
				{Kind: KindCopied, OldPath: "e.go", Path: "e2.go"},
			},
		},
		{
			name:  "15. empty input",
			input: "",
			want:  nil,
		},
		{
			name:    "16. truncated normal record (M with no path)",
			input:   "M",
			wantErr: "missing path for M record",
		},
		{
			name:    "17. truncated rename (R100 with no paths)",
			input:   "R100",
			wantErr: "truncated R record",
		},
		{
			name:    "18. truncated copy (C075 missing new path)",
			input:   joinFields("C075", "src.go"),
			wantErr: "truncated C record",
		},
		{
			name:    "19. unknown status",
			input:   joinFields("X", "weird.go"),
			wantErr: "unsupported status token",
		},
		{
			name:    "20. empty destination path (A with NUL followed by NUL)",
			input:   "A" + nul + nul,
			wantErr: "empty destination path for A record",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGitStatusRecords(tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parsed %d records, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].Kind != tt.want[i].Kind {
					t.Errorf("[%d] Kind = %q, want %q", i, got[i].Kind, tt.want[i].Kind)
				}
				if got[i].Path != tt.want[i].Path {
					t.Errorf("[%d] Path = %q, want %q", i, got[i].Path, tt.want[i].Path)
				}
				if got[i].OldPath != tt.want[i].OldPath {
					t.Errorf("[%d] OldPath = %q, want %q", i, got[i].OldPath, tt.want[i].OldPath)
				}
			}
		})
	}
}

func TestParseGitStatusRecords_DoesNotPanic(t *testing.T) {
	// A handful of additional abuse cases; the parser must surface
	// errors rather than panic.
	bad := []string{
		"M\x00",                    // explicit empty destination
		"R100\x00old.go\x00",       // rename missing destination
		"R12x\x00old.go\x00new.go", // non-numeric similarity
		" \x00foo.go",              // unsupported token (space)
		"AA\x00foo.go",             // unsupported token prefix
	}
	for _, in := range bad {
		_, _ = ParseGitStatusRecords(in) // must not panic
	}
}

func TestParseGitStatusRecords_OrderPreserved(t *testing.T) {
	// Multiple records arriving in arbitrary order should come out
	// in the same order they were written, never re-ordered.
	in := joinFields(
		"A", "z.go",
		"M", "a.go",
		"D", "m.go",
	)
	got, err := ParseGitStatusRecords(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"z.go", "a.go", "m.go"}
	for i := range got {
		if got[i].Path != want[i] {
			t.Errorf("[%d] Path = %q, want %q", i, got[i].Path, want[i])
		}
	}
}

func TestNormalizeGitStatusToken(t *testing.T) {
	tests := []struct {
		in     string
		want   ChangeKind
		wantOK bool
	}{
		{"A", KindAdded, true},
		{"M", KindModified, true},
		{"D", KindDeleted, true},
		{"U", KindUnmerged, true},
		{"R100", KindRenamed, true},
		{"R087", KindRenamed, true},
		{"C100", KindCopied, true},
		{"C075", KindCopied, true},
		{"", "", false},
		{"X", "", false},
		{"AA", "", false},
	}
	for _, tt := range tests {
		got, ok := NormalizeGitStatusToken(tt.in)
		if ok != tt.wantOK || got != tt.want {
			t.Errorf("NormalizeGitStatusToken(%q) = (%q,%v), want (%q,%v)", tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
}

// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"reflect"
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

// TestParseGitStatusRecords_TableDriven covers every status letter the
// parser accepts (A, M, D, T, U, X, B, R*, C*), example paths with
// spaces / tabs / newlines / Unicode / leading dashes, multiple
// records, empty input, and every malformed form the parser must
// reject. The data is intentionally conservative — only stable
// asserted behavior is encoded.
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
			name:  "5. type-change file (regular -> symlink/submodule)",
			input: joinFields("T", "linked.go"),
			want: []GitChange{
				{Kind: KindTypeChanged, Path: "linked.go"},
			},
		},
		{
			name:  "6. unknown (X) status",
			input: joinFields("X", "mystery.go"),
			want: []GitChange{
				{Kind: KindUnknown, Path: "mystery.go"},
			},
		},
		{
			name:  "7. broken-pair (B) status",
			input: joinFields("B", "broken.go"),
			want: []GitChange{
				{Kind: KindBrokenPair, Path: "broken.go"},
			},
		},
		{
			name:  "8. rename with R100",
			input: joinFields("R100", "old/path.go", "new/path.go"),
			want: []GitChange{
				{Kind: KindRenamed, OldPath: "old/path.go", Path: "new/path.go"},
			},
		},
		{
			name:  "9. rename with non-100 score",
			input: joinFields("R087", "old.go", "new.go"),
			want: []GitChange{
				{Kind: KindRenamed, OldPath: "old.go", Path: "new.go"},
			},
		},
		{
			name:  "10. copy with C100",
			input: joinFields("C100", "source.go", "copy.go"),
			want: []GitChange{
				{Kind: KindCopied, OldPath: "source.go", Path: "copy.go"},
			},
		},
		{
			name:  "11. copy with non-100 score",
			input: joinFields("C075", "src.go", "dest.go"),
			want: []GitChange{
				{Kind: KindCopied, OldPath: "src.go", Path: "dest.go"},
			},
		},
		{
			name:  "12. paths with spaces",
			input: joinFields("M", "path with spaces.go"),
			want: []GitChange{
				{Kind: KindModified, Path: "path with spaces.go"},
			},
		},
		{
			name:  "13. paths with tabs",
			input: joinFields("A", "tab\tin\tpath.go"),
			want: []GitChange{
				{Kind: KindAdded, Path: "tab\tin\tpath.go"},
			},
		},
		{
			name:  "14. paths with newlines",
			input: joinFields("D", "weird\nnewline\npath.go"),
			want: []GitChange{
				{Kind: KindDeleted, Path: "weird\nnewline\npath.go"},
			},
		},
		{
			name:  "15. unicode paths",
			input: joinFields("A", "путь/файл.go"),
			want: []GitChange{
				{Kind: KindAdded, Path: "путь/файл.go"},
			},
		},
		{
			name:  "16. leading-dash paths",
			input: joinFields("M", "-dashed-flag.go"),
			want: []GitChange{
				{Kind: KindModified, Path: "-dashed-flag.go"},
			},
		},
		{
			name: "17. multiple adjacent records",
			input: joinFields(
				"A", "a.go",
				"M", "b.go",
				"D", "c.go",
				"T", "d.go",
				"U", "e.go",
				"R100", "f.go", "f2.go",
				"C075", "g.go", "g2.go",
				"X", "h.go",
				"B", "i.go",
			),
			want: []GitChange{
				{Kind: KindAdded, Path: "a.go"},
				{Kind: KindModified, Path: "b.go"},
				{Kind: KindDeleted, Path: "c.go"},
				{Kind: KindTypeChanged, Path: "d.go"},
				{Kind: KindUnmerged, Path: "e.go"},
				{Kind: KindRenamed, OldPath: "f.go", Path: "f2.go"},
				{Kind: KindCopied, OldPath: "g.go", Path: "g2.go"},
				{Kind: KindUnknown, Path: "h.go"},
				{Kind: KindBrokenPair, Path: "i.go"},
			},
		},
		{
			name:  "18. empty input",
			input: "",
			want:  nil,
		},
		{
			name:    "19. truncated normal record (M with no path)",
			input:   "M",
			wantErr: "missing path for M record",
		},
		{
			name:    "20. truncated rename (R100 with no paths)",
			input:   "R100",
			wantErr: "truncated R record",
		},
		{
			name:    "21. truncated copy (C075 missing new path)",
			input:   joinFields("C075", "src.go"),
			wantErr: "truncated C record",
		},
		{
			name:    "22. unknown status",
			input:   joinFields("X1", "weird.go"),
			wantErr: "unsupported status token",
		},
		{
			name:    "23. empty destination path",
			input:   "A" + nul + nul,
			wantErr: "empty destination path for A record",
		},
		{
			name:    "24. type-change with no path",
			input:   "T",
			wantErr: "missing path for T record",
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
	bad := []string{
		"M\x00",                    // ordinary record with empty destination
		"R100\x00old.go\x00",       // rename missing destination
		"R12x\x00old.go\x00new.go", // non-numeric similarity
		" \x00foo.go",              // unsupported token (space)
		"AA\x00foo.go",             // unsupported token form
		"t\x00foo.go",              // lowercase rewrite is not a -z token
	}
	for _, in := range bad {
		_, _ = ParseGitStatusRecords(in) // must not panic
	}
}

func TestParseGitStatusRecords_OrderPreserved(t *testing.T) {
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
		// Plain uppercase status letters are accepted.
		{"A", KindAdded, true},
		{"M", KindModified, true},
		{"D", KindDeleted, true},
		{"T", KindTypeChanged, true},
		{"U", KindUnmerged, true},
		{"X", KindUnknown, true},
		{"B", KindBrokenPair, true},
		// Rename/copy with a numeric similarity score.
		{"R100", KindRenamed, true},
		{"R087", KindRenamed, true},
		{"C100", KindCopied, true},
		{"C075", KindCopied, true},
		// Empty input is rejected.
		{"", "", false},
		// Bare R/C without a numeric score are rejected; they would
		// correspond to a malformed `git diff --name-status -z`
		// record and the parser already rejects them at the structured
		// layer.
		{"R", "", false},
		{"C", "", false},
		// Bare letters outside the supported set are rejected.
		{"X1", "", false},
		{"AA", "", false},
		{"Y", "", false},
		// Lowercase rewrites are rejected: Git's `-z` form emits only
		// uppercase letters and we refuse to guess.
		{"a", "", false},
		{"m", "", false},
		{"r100", "", false},
	}
	for _, tt := range tests {
		got, ok := NormalizeGitStatusToken(tt.in)
		if ok != tt.wantOK || got != tt.want {
			t.Errorf("NormalizeGitStatusToken(%q) = (%q,%v), want (%q,%v)",
				tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
}

// TestSplitNULRecords locks the helper's documented behaviour:
// trailing NUL yields no trailing empty field, interior empty fields
// are preserved.
func TestSplitNULRecords(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "empty input",
			in:   "",
			want: nil,
		},
		{
			name: "single field without trailing NUL",
			in:   "M",
			want: []string{"M"},
		},
		{
			name: "single field with trailing NUL drops empty",
			in:   "M\x00",
			want: []string{"M"},
		},
		{
			name: "two fields with trailing NUL",
			in:   "M\x00file.go\x00",
			want: []string{"M", "file.go"},
		},
		{
			name: "two fields without trailing NUL",
			in:   "M\x00file.go",
			want: []string{"M", "file.go"},
		},
		{
			name: "interior empty field preserved",
			in:   "M\x00\x00file.go\x00",
			want: []string{"M", "", "file.go"},
		},
		{
			name: "rename-style layout",
			in:   "R100\x00old.go\x00new.go\x00",
			want: []string{"R100", "old.go", "new.go"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitNULRecords(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("SplitNULRecords(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}

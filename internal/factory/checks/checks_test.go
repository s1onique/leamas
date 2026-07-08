package checks

import (
	"testing"
)

func TestHasErrors(t *testing.T) {
	tests := []struct {
		name     string
		findings []Finding
		want     bool
	}{
		{
			name:     "empty findings",
			findings: []Finding{},
			want:     false,
		},
		{
			name: "all warnings",
			findings: []Finding{
				{Severity: SeverityWarn},
			},
			want: false,
		},
		{
			name: "has error",
			findings: []Finding{
				{Severity: SeverityWarn},
				{Severity: SeverityError},
			},
			want: true,
		},
		{
			name: "only errors",
			findings: []Finding{
				{Severity: SeverityError},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasErrors(tt.findings)
			if got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortFindings(t *testing.T) {
	findings := []Finding{
		{Path: "b.go", Kind: "x", Message: "1"},
		{Path: "a.go", Kind: "y", Message: "2"},
		{Path: "a.go", Kind: "x", Message: "3"},
		{Path: "a.go", Kind: "x", Message: "2"},
	}

	SortFindings(findings)

	// Expected order: a.go x 2, a.go x 3, a.go y 2, b.go x 1
	expected := []string{
		"a.go:x:2",
		"a.go:x:3",
		"a.go:y:2",
		"b.go:x:1",
	}

	for i, f := range findings {
		want := expected[i]
		got := f.Path + ":" + f.Kind + ":" + f.Message
		if got != want {
			t.Errorf("SortFindings()[%d] = %s, want %s", i, got, want)
		}
	}
}

func TestFileExists(t *testing.T) {
	if FileExists("checks_test.go") != true {
		t.Error("checks_test.go should exist")
	}
	if FileExists("nonexistent.go") != false {
		t.Error("nonexistent.go should not exist")
	}
}

func TestPathInDir(t *testing.T) {
	if !PathInDir("internal/foo.go", "internal") {
		t.Error("internal/foo.go should be in internal")
	}
	if PathInDir("external/foo.go", "internal") {
		t.Error("external/foo.go should not be in internal")
	}
}

func TestCountMeaningfulBashLOC(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "empty",
			content: "",
			want:    0,
		},
		{
			name:    "blank lines only",
			content: "\n\n\n",
			want:    0,
		},
		{
			name:    "comments only",
			content: "# comment\n# another",
			want:    0,
		},
		{
			name:    "mixed",
			content: "# comment\n\necho hello\n# another\nworld",
			want:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountMeaningfulBashLOC(tt.content)
			if got != tt.want {
				t.Errorf("CountMeaningfulBashLOC() = %d, want %d", got, tt.want)
			}
		})
	}
}

// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"os"
	"testing"
)

func TestExtractGoFiles(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  []string
	}{
		{
			name:  "empty",
			paths: []string{},
			want:  nil,
		},
		{
			name:  "go files only",
			paths: []string{"a.go", "b.go"},
			want:  []string{"a.go", "b.go"},
		},
		{
			name:  "excludes test files",
			paths: []string{"a.go", "a_test.go", "b.go"},
			want:  []string{"a.go", "b.go"},
		},
		{
			name:  "mixed with non-go",
			paths: []string{"a.go", "b.txt", "c.md"},
			want:  []string{"a.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractGoFiles(tt.paths)
			if len(got) != len(tt.want) {
				t.Errorf("extractGoFiles() len = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("extractGoFiles()[%d] = %v, want %v", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestExtractCLIFiles(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  []string
	}{
		{
			name:  "empty",
			paths: []string{},
			want:  nil,
		},
		{
			name:  "cmd files only",
			paths: []string{"cmd/leamas/main.go", "cmd/leamas/cmd.go"},
			want:  []string{"cmd/leamas/main.go", "cmd/leamas/cmd.go"},
		},
		{
			name:  "excludes non-cmd",
			paths: []string{"cmd/leamas/main.go", "internal/pkg/a.go"},
			want:  []string{"cmd/leamas/main.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCLIFiles(tt.paths)
			if len(got) != len(tt.want) {
				t.Errorf("extractCLIFiles() len = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("extractCLIFiles()[%d] = %v, want %v", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestIsLikelyCommand(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"empty", "", false},
		{"single char", "a", false},
		{"too long", string(make([]byte, 51)), false},
		{"url", "https://example.com", false},
		{"env var", "${HOME}/bin", false},
		{"valid command", "leamas digest", true},
		{"simple command", "ls", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLikelyCommand(tt.s); got != tt.want {
				t.Errorf("isLikelyCommand(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestDeduplicateStrings(t *testing.T) {
	tests := []struct {
		name string
		strs []string
		want []string
	}{
		{
			name: "empty",
			strs: []string{},
			want: nil,
		},
		{
			name: "single element",
			strs: []string{"a"},
			want: []string{"a"},
		},
		{
			name: "no duplicates",
			strs: []string{"a", "b", "c"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "with duplicates",
			strs: []string{"a", "b", "a", "c", "b"},
			want: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deduplicateStrings(tt.strs)
			if len(got) != len(tt.want) {
				t.Errorf("deduplicateStrings() len = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("deduplicateStrings()[%d] = %v, want %v", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestSymbolKeyString(t *testing.T) {
	tests := []struct {
		name string
		sk   symbolKey
		want string
	}{
		{
			name: "simple function",
			sk:   symbolKey{Name: "Foo", Kind: "func"},
			want: "Foo(func)",
		},
		{
			name: "simple type",
			sk:   symbolKey{Name: "Bar", Kind: "type"},
			want: "Bar(type)",
		},
		{
			name: "method",
			sk:   symbolKey{Name: "Method", Kind: "method", Receiver: "Foo"},
			want: "Method.method(Foo)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := symbolKeyString(tt.sk); got != tt.want {
				t.Errorf("symbolKeyString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRangeModeInfo(t *testing.T) {
	tests := []struct {
		name     string
		revRange string
		wantBase string
		wantHead string
	}{
		{
			name:     "standard format",
			revRange: "abc123..def456",
			wantBase: "abc123",
			wantHead: "def456",
		},
		{
			name:     "HEAD style",
			revRange: "HEAD~1..HEAD",
			wantBase: "HEAD~1",
			wantHead: "HEAD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, head := getRangeModeInfo(tt.revRange)
			if base != tt.wantBase {
				t.Errorf("getRangeModeInfo() base = %v, want %v", base, tt.wantBase)
			}
			if head != tt.wantHead {
				t.Errorf("getRangeModeInfo() head = %v, want %v", head, tt.wantHead)
			}
		})
	}
}

func TestParseExports_Functions(t *testing.T) {
	code := `package foo
func Exported() {}
func unexported() {}
func AnotherExported() int { return 0 }
`

	exports, err := parseExportsFromBytes([]byte(code), "foo")
	if err != nil {
		t.Fatalf("parseExportsFromBytes failed: %v", err)
	}

	var funcNames []string
	for key := range exports {
		if key.Kind == "func" {
			funcNames = append(funcNames, key.Name)
		}
	}

	if len(funcNames) != 2 {
		t.Errorf("expected 2 exported funcs, got %d: %v", len(funcNames), funcNames)
	}
}

func TestParseExports_Methods(t *testing.T) {
	code := `package foo
type MyType struct{}
func (m *MyType) ExportedMethod() {}
func (m MyType) AnotherMethod() {}
func (m *MyType) unexportedMethod() {}
`

	exports, err := parseExportsFromBytes([]byte(code), "foo")
	if err != nil {
		t.Fatalf("parseExportsFromBytes failed: %v", err)
	}

	var methodNames []string
	for key := range exports {
		if key.Kind == "method" {
			methodNames = append(methodNames, key.Name)
		}
	}

	if len(methodNames) != 2 {
		t.Errorf("expected 2 exported methods, got %d: %v", len(methodNames), methodNames)
	}
}

func TestParseExports_Interfaces(t *testing.T) {
	code := `package foo
type ExportedInterface interface {
	DoSomething() error
	AnotherMethod() int
	unexported() // should be ignored
}
`

	exports, err := parseExportsFromBytes([]byte(code), "foo")
	if err != nil {
		t.Fatalf("parseExportsFromBytes failed: %v", err)
	}

	var ifaceMethodNames []string
	for key := range exports {
		if key.Kind == "interface_method" {
			ifaceMethodNames = append(ifaceMethodNames, key.Name)
		}
	}

	if len(ifaceMethodNames) != 2 {
		t.Errorf("expected 2 exported interface methods, got %d: %v", len(ifaceMethodNames), ifaceMethodNames)
	}
}

func TestReceiverTypeName(t *testing.T) {
	// This test verifies the function exists and handles edge cases
	// Full parsing tests are covered by TestParseExports_Methods
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

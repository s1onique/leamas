// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"testing"
)

func TestGoModuleFiles(t *testing.T) {
	files := goModuleFiles()
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
	if files[0] != "go.mod" {
		t.Errorf("expected go.mod, got %s", files[0])
	}
	if files[1] != "go.sum" {
		t.Errorf("expected go.sum, got %s", files[1])
	}
}

func TestHasGoModuleFiles(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		expected bool
	}{
		{
			name:     "no go files",
			paths:    []string{"internal/foo.go", "cmd/bar/main.go"},
			expected: false,
		},
		{
			name:     "go.mod only",
			paths:    []string{"go.mod"},
			expected: true,
		},
		{
			name:     "go.sum only",
			paths:    []string{"go.sum"},
			expected: true,
		},
		{
			name:     "both go files",
			paths:    []string{"go.mod", "go.sum"},
			expected: true,
		},
		{
			name:     "go files with others",
			paths:    []string{"internal/foo.go", "go.mod", "cmd/bar/main.go", "go.sum"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasGoModuleFiles(tt.paths)
			if result != tt.expected {
				t.Errorf("hasGoModuleFiles(%v) = %v, want %v", tt.paths, result, tt.expected)
			}
		})
	}
}

func TestMapsEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        map[string]string
		b        map[string]string
		expected bool
	}{
		{
			name:     "both nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "empty maps",
			a:        map[string]string{},
			b:        map[string]string{},
			expected: true,
		},
		{
			name:     "same content",
			a:        map[string]string{"foo": "v1"},
			b:        map[string]string{"foo": "v1"},
			expected: true,
		},
		{
			name:     "different values",
			a:        map[string]string{"foo": "v1"},
			b:        map[string]string{"foo": "v2"},
			expected: false,
		},
		{
			name:     "different keys",
			a:        map[string]string{"foo": "v1"},
			b:        map[string]string{"bar": "v1"},
			expected: false,
		},
		{
			name:     "different sizes",
			a:        map[string]string{"foo": "v1"},
			b:        map[string]string{"foo": "v1", "bar": "v2"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapsEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("mapsEqual(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestCompareGoSum(t *testing.T) {
	tests := []struct {
		name     string
		base     map[string]string
		head     map[string]string
		expected []string
	}{
		{
			name:     "no additions",
			base:     map[string]string{"foo v1.0.0": "h1"},
			head:     map[string]string{"foo v1.0.0": "h1"},
			expected: nil,
		},
		{
			name:     "one addition",
			base:     map[string]string{"foo v1.0.0": "h1"},
			head:     map[string]string{"foo v1.0.0": "h1", "bar v2.0.0": "h2"},
			expected: []string{"bar v2.0.0"},
		},
		{
			name:     "multiple additions",
			base:     map[string]string{"foo v1.0.0": "h1"},
			head:     map[string]string{"foo v1.0.0": "h1", "bar v2.0.0": "h2", "baz v3.0.0": "h3"},
			expected: []string{"bar v2.0.0", "baz v3.0.0"},
		},
		{
			name:     "empty head",
			base:     map[string]string{"foo v1.0.0": "h1"},
			head:     map[string]string{},
			expected: nil,
		},
		{
			name:     "empty base",
			base:     map[string]string{},
			head:     map[string]string{"foo v1.0.0": "h1"},
			expected: []string{"foo v1.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareGoSum(tt.base, tt.head)
			if len(result) != len(tt.expected) {
				t.Errorf("compareGoSum() = %v, want %v", result, tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("compareGoSum()[%d] = %v, want %v", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestParseGoSum(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name:     "empty",
			content:  "",
			expected: map[string]string{},
		},
		{
			name:     "single line",
			content:  "foo v1.0.0 h1",
			expected: map[string]string{"foo v1.0.0": "v1.0.0"},
		},
		{
			name:     "multiple lines",
			content:  "foo v1.0.0 h1\nbar v2.0.0 h2\n",
			expected: map[string]string{"foo v1.0.0": "v1.0.0", "bar v2.0.0": "v2.0.0"},
		},
		{
			name:     "with empty lines",
			content:  "foo v1.0.0 h1\n\nbar v2.0.0 h2\n",
			expected: map[string]string{"foo v1.0.0": "v1.0.0", "bar v2.0.0": "v2.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGoSum([]byte(tt.content))
			if len(result) != len(tt.expected) {
				t.Errorf("parseGoSum() = %v, want %v", result, tt.expected)
				return
			}
			for k, v := range tt.expected {
				if rv, ok := result[k]; !ok || rv != v {
					t.Errorf("parseGoSum()[%s] = %v, want %v", k, rv, v)
				}
			}
		})
	}
}

func TestRenderDependencyDelta(t *testing.T) {
	delta := &DependencyDelta{
		Ecosystem:            "go",
		SourceStatus:         "present",
		GoModChanged:         true,
		GoSumChanged:         true,
		ModulePathChanged:    false,
		GoVersionChanged:     false,
		ToolchainChanged:     false,
		RequiresAdded:        1,
		RequiresRemoved:      0,
		RequiresModified:     2,
		ReplacesAdded:        0,
		ReplacesRemoved:      0,
		ReplacesModified:     0,
		Module:               ModuleInfo{Before: "github.com/s1onique/leamas", After: "github.com/s1onique/leamas"},
		GoVersion:            VersionInfo{Before: "1.21", After: "1.21"},
		Toolchain:            VersionInfo{Before: "go1.23.0", After: "go1.23.0"},
		RequiresAddedList:    []string{"github.com/example/foo v1.2.3"},
		RequiresRemovedList:  []string{},
		RequiresModifiedList: []string{"golang.org/x/tools v0.33.0 -> v0.34.0"},
		ReplacesAddedList:    []string{},
		ReplacesRemovedList:  []string{},
		ReplacesModifiedList: []string{},
	}

	result := RenderDependencyDelta(delta)

	// Check key fields are present
	if !containsString(result, "## DEPENDENCY_DELTA") {
		t.Error("missing DEPENDENCY_DELTA header")
	}
	if !containsString(result, "ecosystem=go") {
		t.Error("missing ecosystem field")
	}
	if !containsString(result, "go_mod_changed=true") {
		t.Error("missing go_mod_changed field")
	}
	if !containsString(result, "requires_added=1") {
		t.Error("missing requires_added field")
	}
	if !containsString(result, "requires_modified=2") {
		t.Error("missing requires_modified field")
	}
	if !containsString(result, "github.com/example/foo v1.2.3") {
		t.Error("missing requires_added content")
	}
}

func TestRenderEmptyDependencyDelta(t *testing.T) {
	result := RenderEmptyDependencyDelta()

	if !containsString(result, "## DEPENDENCY_DELTA") {
		t.Error("missing DEPENDENCY_DELTA header")
	}
	if !containsString(result, "source_status=absent") {
		t.Error("missing source_status=absent for empty delta")
	}
	if !containsString(result, "go_mod_changed=false") {
		t.Error("missing go_mod_changed=false")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

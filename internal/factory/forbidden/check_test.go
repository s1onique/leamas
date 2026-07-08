package forbidden

import (
	"testing"
)

func TestContainsForbidden(t *testing.T) {
	tests := []struct {
		pattern string
		content string
		want    bool
	}{
		{"OIDC|oidc", "using OIDC for auth", true},
		{"OIDC|oidc", "no auth here", false},
		{"RBAC|rbac", "has RBAC permissions", true},
		{"database/sql", `import "database/sql"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := containsForbidden(tt.content, tt.pattern)
			if got != tt.want {
				t.Errorf("containsForbidden(%q, %q) = %v, want %v", tt.content, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestIsInAllowedDir(t *testing.T) {
	tests := []struct {
		path  string
		allow bool
	}{
		{"docs/doctrine/test.md", true},
		{"./docs/doctrine/test.md", true},
		{"docs/adr/0001-test.md", true},
		{"docs/factory/test.md", true},
		{"docs/close-reports/test.md", true},
		{"internal/factory/auth.go", true},
		{"testdata/test.go", true},
		{"internal/app/foo.go", false},
		{"cmd/main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isInAllowedDir(tt.path)
			if got != tt.allow {
				t.Errorf("isInAllowedDir(%q) = %v, want %v", tt.path, got, tt.allow)
			}
		})
	}
}

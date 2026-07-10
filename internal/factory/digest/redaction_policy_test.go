// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"testing"
)

func TestDecideRedactionPolicy_SourceFiles(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		tracked bool
		want    RedactionClass
	}{
		// Python - the regression-critical extension
		{"Python file tracked", "cmd/app/main.py", true, RedactionClassSource},
		{"Python file untracked", "new_file.py", false, RedactionClassSource},
		{"Python test file", "tests/test_auth.py", true, RedactionClassSource},

		// Go
		{"Go file", "internal/pkg/foo.go", true, RedactionClassSource},

		// TypeScript
		{"TypeScript file", "src/app.ts", true, RedactionClassSource},
		{"TSX file", "src/Component.tsx", true, RedactionClassSource},

		// JavaScript
		{"JavaScript file", "src/index.js", true, RedactionClassSource},
		{"JSX file", "src/App.jsx", true, RedactionClassSource},

		// Rust
		{"Rust file", "src/main.rs", true, RedactionClassSource},

		// Zig (correct .zig extension, not "zig" without dot)
		{"Zig file", "src/main.zig", true, RedactionClassSource},

		// Java and Kotlin
		{"Java file", "src/Main.java", true, RedactionClassSource},
		{"Kotlin file", "src/Main.kt", true, RedactionClassSource},
		{"Kotlin script file", "src/script.kts", true, RedactionClassSource},

		// C/C++
		{"C file", "src/main.c", true, RedactionClassSource},
		{"C header", "include/main.h", true, RedactionClassSource},
		{"C++ file", "src/main.cpp", true, RedactionClassSource},
		{"C++ header", "include/main.hpp", true, RedactionClassSource},

		// Shell scripts
		{"Shell script", "scripts/deploy.sh", true, RedactionClassSource},
		{"Bash script", "scripts/cleanup.bash", true, RedactionClassSource},
		{"Zsh script", "scripts/setup.zsh", true, RedactionClassSource},
		{"Fish script", "scripts/test.fish", true, RedactionClassSource},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecideRedactionPolicy(tt.path, tt.tracked)
			if got.Class != tt.want {
				t.Errorf("DecideRedactionPolicy(%q, %v) class = %v, want %v", tt.path, tt.tracked, got.Class, tt.want)
			}
		})
	}
}

func TestDecideRedactionPolicy_NonSourceFiles(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		tracked bool
	}{
		// Environment files
		{".env file", ".env", true},
		{".env.local file", ".env.local", true},
		{"api.env file", "config/api.env", true},

		// Log files
		{"Log file", "logs/app.log", true},
		{"Debug log", "debug.log", true},

		// Config files
		{"JSON config", "config.json", true},
		{"YAML config", "config.yaml", true},
		{"YML config", "config.yml", true},
		{"TOML config", "config.toml", true},
		{"INI file", "settings.ini", true},
		{"Properties file", "app.properties", true},
		{"XML file", "config.xml", true},

		// Certificate/key files
		{"PEM file", "certs/server.pem", true},
		{"Key file", "keys/private.key", true},
		{"CRT file", "certs/server.crt", true},

		// Markdown and docs (not source by default)
		{"Markdown file", "README.md", true},
		{"Documentation", "docs/guide.md", true},

		// Other
		{"Plain text", "notes.txt", true},
		{"Dockerfile", "Dockerfile", true},
		{"Makefile", "Makefile", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecideRedactionPolicy(tt.path, tt.tracked)
			if got.Class != RedactionClassNonSource {
				t.Errorf("DecideRedactionPolicy(%q, %v) class = %v, want %v",
					tt.path, tt.tracked, got.Class, RedactionClassNonSource)
			}
		})
	}
}

func TestDecideRedactionPolicy_Decisions(t *testing.T) {
	// Source files should get preserve_and_warn
	sourceFile := DecideRedactionPolicy("main.py", true)
	if sourceFile.Decision != RedactionDecisionPreserveAndWarn {
		t.Errorf("source file decision = %v, want %v", sourceFile.Decision, RedactionDecisionPreserveAndWarn)
	}

	// Non-source files should get redact
	nonSourceFile := DecideRedactionPolicy(".env", true)
	if nonSourceFile.Decision != RedactionDecisionRedact {
		t.Errorf("non-source file decision = %v, want %v", nonSourceFile.Decision, RedactionDecisionRedact)
	}
}

func TestDecideRedactionPolicy_Reasons(t *testing.T) {
	// Source files have review_fidelity reason
	sourceResult := DecideRedactionPolicy("main.py", true)
	if sourceResult.Reason != "review_fidelity" {
		t.Errorf("source reason = %q, want %q", sourceResult.Reason, "review_fidelity")
	}

	// Non-source files have operational_secret_risk reason
	nonSourceResult := DecideRedactionPolicy(".env", true)
	if nonSourceResult.Reason != "operational_secret_risk" {
		t.Errorf("non-source reason = %q, want %q", nonSourceResult.Reason, "operational_secret_risk")
	}
}

func TestIsSourceExtension(t *testing.T) {
	tests := []struct {
		path  string
		isSrc bool
	}{
		// Source extensions - should return true
		{"main.py", true},
		{"main.go", true},
		{"main.ts", true},
		{"main.tsx", true},
		{"main.js", true},
		{"main.jsx", true},
		{"main.rs", true},
		{"main.zig", true},
		{"Main.java", true},
		{"Main.kt", true},
		{"Main.kts", true},
		{"main.c", true},
		{"main.h", true},
		{"main.cpp", true},
		{"main.hpp", true},
		{"deploy.sh", true},
		{"test.bash", true},
		{"config.zsh", true},
		{"script.fish", true},

		// Case insensitive
		{"main.PY", true},
		{"main.Go", true},
		{"main.TS", true},

		// Non-source extensions - should return false
		{"main.pyc", false},
		{"main.pyi", false},
		{"main.hpp.gch", false},
		{"Makefile", false},
		{"config.json", false},
		{"config.yaml", false},
		{"config.yml", false},
		{"config.toml", false},
		{".env", false},
		{"app.log", false},
		{"README.md", false},
		{"server.pem", false},
		{"private.key", false},
		{"main.html", false},
		{"main.css", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsSourceExtension(tt.path)
			if got != tt.isSrc {
				t.Errorf("IsSourceExtension(%q) = %v, want %v", tt.path, got, tt.isSrc)
			}
		})
	}
}

func TestClassifyDigestEntry(t *testing.T) {
	tests := []struct {
		path string
		kind DigestEntryKind
	}{
		// Source files
		{"main.py", DigestEntryKindSource},
		{"pkg/foo.go", DigestEntryKindSource},
		{"src/App.tsx", DigestEntryKindSource},
		{"lib/util.rs", DigestEntryKindSource},

		// Environment files
		{".env", DigestEntryKindEnv},
		{".env.local", DigestEntryKindEnv},
		{"config/.env", DigestEntryKindEnv},
		{"api.env", DigestEntryKindEnv},

		// Log files
		{"app.log", DigestEntryKindLog},
		{"logs/debug.log", DigestEntryKindLog},
		{"error.log", DigestEntryKindLog},

		// Config files
		{"config.json", DigestEntryKindConfig},
		{"settings.yaml", DigestEntryKindConfig},
		{"config.yml", DigestEntryKindConfig},
		{"meta.toml", DigestEntryKindConfig},
		{"app.ini", DigestEntryKindConfig},
		{"settings.conf", DigestEntryKindConfig},
		{"server.pem", DigestEntryKindConfig},
		{"private.key", DigestEntryKindConfig},
		{"cert.crt", DigestEntryKindConfig},
		{"Makefile", DigestEntryKindConfig},
		{"rules.mk", DigestEntryKindConfig},
		{"Dockerfile", DigestEntryKindConfig},
		{".gitlab-ci.yml", DigestEntryKindConfig},

		// Generated directories - .go files in generated dirs are still classified as source
		// because source extension check takes precedence
		{"generated/foo.pb.go", DigestEntryKindSource},
		{"src/__pycache__/foo.pyc", DigestEntryKindGenerated},

		// Other
		{"README.md", DigestEntryKindOther},
		{"notes.txt", DigestEntryKindOther},
		{"output.csv", DigestEntryKindOther},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := ClassifyDigestEntry(tt.path)
			if got != tt.kind {
				t.Errorf("ClassifyDigestEntry(%q) = %v, want %v", tt.path, got, tt.kind)
			}
		})
	}
}

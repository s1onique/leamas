// Package digest provides targeted digest generation for Git repositories.
// It creates reviewable artifacts of repository changes suitable for
// agent-assisted review workflows.
package digest

import (
	"path/filepath"
	"strings"
)

// RedactionClass classifies a file for redaction policy purposes.
type RedactionClass string

const (
	// RedactionClassSource indicates a tracked source file that should be
	// preserved for review fidelity in default mode.
	RedactionClassSource RedactionClass = "source"

	// RedactionClassNonSource indicates a non-source file (config, log, env, etc.)
	// that should be redacted.
	RedactionClassNonSource RedactionClass = "non_source"
)

// RedactionDecision describes what action to take for redaction.
type RedactionDecision string

const (
	// RedactionDecisionPreserveAndWarn means preserve source content exactly
	// and emit warnings for detected secret-like patterns.
	RedactionDecisionPreserveAndWarn RedactionDecision = "preserve_and_warn"

	// RedactionDecisionRedact means apply standard redaction.
	RedactionDecisionRedact RedactionDecision = "redact"
)

// RedactionPolicyResult contains the policy decision for a file.
type RedactionPolicyResult struct {
	Class    RedactionClass
	Decision RedactionDecision
	Reason   string
}

// sourceExtensions is the allowlist of source file extensions.
// Files with these extensions are treated as source for redaction purposes.
var sourceExtensions = map[string]bool{
	".py":   true, // Python
	".go":   true, // Go
	".ts":   true, // TypeScript
	".tsx":  true, // TypeScript React
	".js":   true, // JavaScript
	".jsx":  true, // JavaScript React
	".rs":   true, // Rust
	".zig":  true, // Zig
	".java": true, // Java
	".kt":   true, // Kotlin
	".kts":  true, // Kotlin Script
	".c":    true, // C
	".h":    true, // C header
	".cpp":  true, // C++
	".hpp":  true, // C++ header
	".sh":   true, // Shell
	".bash": true, // Bash
	".zsh":  true, // Zsh
	".fish": true, // Fish
}

// IsSourceExtension returns true if the extension is a known source extension.
func IsSourceExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return sourceExtensions[ext]
}

// DecideRedactionPolicy determines the redaction policy for a file.
//
// For default review-fidelity mode:
//   - Tracked source files: preserve and warn (for review fidelity)
//   - Non-source files: redact (logs, config, env, etc.)
//
// The tracked parameter indicates whether the file is tracked in Git.
// In practice, we treat untracked files that match source extensions
// the same as tracked source files, since they represent source content
// being added to the project.
func DecideRedactionPolicy(path string, tracked bool) RedactionPolicyResult {
	if IsSourceExtension(path) {
		return RedactionPolicyResult{
			Class:    RedactionClassSource,
			Decision: RedactionDecisionPreserveAndWarn,
			Reason:   "review_fidelity",
		}
	}

	// All other files (config, logs, env, generated, etc.) are redacted
	return RedactionPolicyResult{
		Class:    RedactionClassNonSource,
		Decision: RedactionDecisionRedact,
		Reason:   "operational_secret_risk",
	}
}

// DigestEntryKind classifies a digest entry for redaction purposes.
// This is used when we have more context about the entry type.
type DigestEntryKind string

const (
	DigestEntryKindSource    DigestEntryKind = "source"
	DigestEntryKindConfig    DigestEntryKind = "config"
	DigestEntryKindLog       DigestEntryKind = "log"
	DigestEntryKindEnv       DigestEntryKind = "env"
	DigestEntryKindGenerated DigestEntryKind = "generated"
	DigestEntryKindOther     DigestEntryKind = "other"
)

// ClassifyDigestEntry determines the kind of a digest entry based on path.
func ClassifyDigestEntry(path string) DigestEntryKind {
	ext := strings.ToLower(filepath.Ext(path))
	base := filepath.Base(path)
	dir := filepath.Dir(path)

	// Check for source files first
	if IsSourceExtension(path) {
		return DigestEntryKindSource
	}

	// Environment files
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return DigestEntryKindEnv
	}
	if strings.HasSuffix(base, ".env") {
		return DigestEntryKindEnv
	}

	// Log files
	if ext == ".log" {
		return DigestEntryKindLog
	}

	// Known config extensions
	configExts := map[string]bool{
		".json":       true,
		".yaml":       true,
		".yml":        true,
		".toml":       true,
		".ini":        true,
		".conf":       true,
		".pem":        true,
		".key":        true,
		".crt":        true,
		".cfg":        true,
		".xml":        true,
		".properties": true,
	}
	if configExts[ext] {
		return DigestEntryKindConfig
	}

	// Makefile and friends
	if base == "Makefile" || strings.HasSuffix(base, ".mk") {
		return DigestEntryKindConfig
	}

	// Dockerfile and scripts without extension
	if base == "Dockerfile" || base == "Dockerfile.dev" || base == "Dockerfile.prod" {
		return DigestEntryKindConfig
	}

	// CI config files
	if base == ".gitlab-ci.yml" || base == ".travis.yml" || base == "Jenkinsfile" {
		return DigestEntryKindConfig
	}

	// Check for generated markers in path
	if strings.Contains(dir, "generated") || strings.Contains(dir, "__pycache__") {
		return DigestEntryKindGenerated
	}

	return DigestEntryKindOther
}

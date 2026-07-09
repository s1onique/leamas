// Package digest provides targeted digest generation for Git repositories.
// Contract constants and header rendering for versioned digest output.
package digest

import (
	"fmt"
	"strings"
)

// ContractVersion is the current digest contract version.
// This version governs the header format and field names.
// Breaking changes to the digest output shape require a version bump.
const ContractVersion = 1

// Contract header field names - these must remain stable.
const (
	ContractFieldVersion   = "LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION"
	ContractFieldAppVer    = "LEAMAS_VERSION"
	ContractFieldCommit    = "LEAMAS_COMMIT"
	ContractFieldBuildTime = "LEAMAS_BUILD_TIME"
	ContractFieldMode      = "DIGEST_MODE"
	ContractFieldCreatedAt = "DIGEST_CREATED_AT"
)

// ContractHeaderFields defines the expected field names in order.
// Used by tests to verify header stability.
var ContractHeaderFields = []string{
	ContractFieldVersion,
	ContractFieldAppVer,
	ContractFieldCommit,
	ContractFieldBuildTime,
	ContractFieldMode,
	ContractFieldCreatedAt,
}

// HeaderInfo holds metadata for rendering the contract header.
// All fields must be provided - RenderContractHeader is a pure formatter.
type HeaderInfo struct {
	Version   string // Leamas application version (e.g., "dev", "0.2.0")
	Commit    string // Git commit of Leamas binary (e.g., "unknown", git SHA)
	BuildTime string // Build time of Leamas binary (e.g., "unknown", RFC3339)
	Mode      Mode   // Effective digest mode
	CreatedAt string // RFC3339 timestamp when digest was generated
}

// RenderContractHeader renders the versioned contract header.
// This is a pure formatter - all values must be provided in info.
func RenderContractHeader(info HeaderInfo) string {
	var sb strings.Builder

	// Contract version (integer)
	sb.WriteString(fmt.Sprintf("%s: %d\n", ContractFieldVersion, ContractVersion))
	// Application version
	sb.WriteString(fmt.Sprintf("%s: %s\n", ContractFieldAppVer, info.Version))
	// Git commit
	sb.WriteString(fmt.Sprintf("%s: %s\n", ContractFieldCommit, info.Commit))
	// Build time
	sb.WriteString(fmt.Sprintf("%s: %s\n", ContractFieldBuildTime, info.BuildTime))
	// Effective digest mode
	sb.WriteString(fmt.Sprintf("%s: %s\n", ContractFieldMode, info.Mode))
	// Timestamp when digest was created
	sb.WriteString(fmt.Sprintf("%s: %s\n", ContractFieldCreatedAt, info.CreatedAt))

	// Blank line to separate header from body
	sb.WriteString("\n")

	return sb.String()
}

// ParseContractHeader parses the contract header from digest content.
// Returns the header lines (without trailing blank line) and the remaining body.
// Returns empty header if content doesn't start with a valid contract header.
func ParseContractHeader(content string) (header string, body string) {
	// Find the first line that starts with our contract version marker
	lines := strings.Split(content, "\n")
	if len(lines) < 7 {
		return "", content
	}

	// Verify it starts with our contract version marker
	if !strings.HasPrefix(lines[0], ContractFieldVersion+": ") {
		return "", content
	}

	// Extract header (first 6 lines)
	header = strings.Join(lines[:6], "\n") + "\n"

	// Body is everything after line 6 (skip the blank line separator at index 6)
	if len(lines) > 7 {
		body = strings.Join(lines[7:], "\n")
	} else if len(lines) == 7 {
		body = lines[6]
	}

	return header, body
}

// ValidateContractHeader checks that a header has expected fields in correct order.
// Returns nil if valid, error describing the mismatch.
func ValidateContractHeader(header string) error {
	lines := strings.Split(header, "\n")
	if len(lines) < 6 {
		return fmt.Errorf("header has %d lines, expected 6", len(lines))
	}

	expected := ContractHeaderFields
	for i, field := range expected {
		if !strings.HasPrefix(lines[i], field+":") {
			return fmt.Errorf("line %d: expected field %q, got %q", i+1, field, lines[i])
		}
	}

	return nil
}

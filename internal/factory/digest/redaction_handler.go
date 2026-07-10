// Package digest provides targeted digest generation for Git repositories.
// It creates reviewable artifacts of repository changes suitable for
// agent-assisted review workflows.
package digest

import (
	"strings"

	"github.com/s1onique/leamas/internal/factory/redact"
)

// RedactionPolicyMetadata contains digest-level redaction policy info.
type RedactionPolicyMetadata struct {
	Version            string
	SourceRedaction    string // "warn_only" or "disabled"
	NonSourceRedaction string // "redact"
}

// DefaultPolicyMetadata returns the default redaction policy metadata.
func DefaultPolicyMetadata() RedactionPolicyMetadata {
	return RedactionPolicyMetadata{
		Version:            "v2",
		SourceRedaction:    "warn_only",
		NonSourceRedaction: "redact",
	}
}

// RenderRedactionPolicyMetadata renders the digest-level redaction policy header.
func RenderRedactionPolicyMetadata(meta RedactionPolicyMetadata) string {
	var sb strings.Builder
	sb.WriteString("REDACTION_POLICY:\n")
	sb.WriteString("  version=")
	sb.WriteString(meta.Version)
	sb.WriteString("\n")
	sb.WriteString("  source_redaction=")
	sb.WriteString(meta.SourceRedaction)
	sb.WriteString("\n")
	sb.WriteString("  source_secret_scan=warn_only\n")
	sb.WriteString("  non_source_redaction=")
	sb.WriteString(meta.NonSourceRedaction)
	sb.WriteString("\n")
	sb.WriteString("  reason=review_fidelity\n")
	return sb.String()
}

// FileRedactionMetadata contains per-file redaction policy info.
type FileRedactionMetadata struct {
	Path             string
	Class            RedactionClass
	Decision         RedactionDecision
	RedactionApplied bool
	WarningCount     int
}

// RenderFileRedactionMetadata renders the per-file redaction metadata.
func RenderFileRedactionMetadata(meta FileRedactionMetadata) string {
	var sb strings.Builder
	sb.WriteString("REDACTION_POLICY:\n")
	sb.WriteString("  class=")
	sb.WriteString(string(meta.Class))
	sb.WriteString("\n")
	sb.WriteString("  decision=")
	sb.WriteString(string(meta.Decision))
	sb.WriteString("\n")
	sb.WriteString("  redaction_applied=")
	if meta.RedactionApplied {
		sb.WriteString("true\n")
	} else {
		sb.WriteString("false\n")
	}
	if meta.Class == RedactionClassSource && meta.WarningCount > 0 {
		sb.WriteString("  source_secret_scan=warn_only\n")
	}
	return sb.String()
}

// RedactDigestWithPolicy applies selective redaction based on content type.
// For source files: preserves content, emits warnings for secret-like patterns.
// For non-source files: applies standard redaction.
func RedactDigestWithPolicy(digestContent string) (string, []SourceSecretWarning) {
	var warnings []SourceSecretWarning

	// Split into sections based on file diff markers
	sections := splitDigestSections(digestContent)

	var result strings.Builder
	for _, section := range sections {
		if section.IsFileSection && section.Path != "" {
			policy := DecideRedactionPolicy(section.Path, section.Tracked)

			if policy.Class == RedactionClassSource {
				// Source file: preserve content, scan for warnings
				// Output order: header (file marker + Metadata) + REDACTION_POLICY + SOURCE_SECRET_WARNINGS + content marker + content
				headerParts := splitSectionHeader(section.Header)
				result.WriteString(headerParts.BeforeMarker) // file marker + Metadata + blank line

				result.WriteString(RenderFileRedactionMetadata(FileRedactionMetadata{
					Path:             section.Path,
					Class:            RedactionClassSource,
					Decision:         RedactionDecisionPreserveAndWarn,
					RedactionApplied: false,
					WarningCount:     countSecretFindings(section.Content),
				}))
				result.WriteString("\n")

				// Scan source content for secrets
				warning := ScanSourceForSecrets(section.Path, section.Content)
				if warning.HasFindings() {
					result.WriteString(RenderSourceSecretWarnings(warning))
					result.WriteString("\n")
					warnings = append(warnings, warning)
				}

				result.WriteString(headerParts.ContentMarker) // content marker (e.g., "--- untracked file content ---")
				result.WriteString(section.Content)           // actual source content
			} else {
				// Non-source file: apply standard redaction
				// Output order: header (file marker + Metadata) + REDACTION_POLICY + content marker + redacted content
				headerParts := splitSectionHeader(section.Header)
				result.WriteString(headerParts.BeforeMarker)

				result.WriteString(RenderFileRedactionMetadata(FileRedactionMetadata{
					Path:             section.Path,
					Class:            RedactionClassNonSource,
					Decision:         RedactionDecisionRedact,
					RedactionApplied: true,
				}))
				result.WriteString("\n")

				result.WriteString(headerParts.ContentMarker)
				redacted := redact.RedactDigest(section.Content)
				result.WriteString(redacted)
			}
		} else {
			// Non-file sections: apply standard redaction
			result.WriteString(redact.RedactDigest(section.Header))
			result.WriteString(redact.RedactDigest(section.Content))
		}
	}

	return result.String(), warnings
}

// sectionHeaderParts holds the parsed header sections.
type sectionHeaderParts struct {
	BeforeMarker  string // file marker, Metadata, REDACTION_POLICY, blank lines
	ContentMarker string // "--- untracked file content ---" or "--- staged diff ---" etc.
}

// splitSectionHeader splits the header into parts before and after the content marker.
func splitSectionHeader(header string) sectionHeaderParts {
	lines := strings.Split(header, "\n")
	var before []string
	var after []string
	markerFound := false

	for _, line := range lines {
		if !markerFound && (strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ")) {
			markerFound = true
		}
		if !markerFound {
			before = append(before, line)
		} else {
			after = append(after, line)
		}
	}

	return sectionHeaderParts{
		BeforeMarker:  strings.Join(before, "\n") + "\n",
		ContentMarker: strings.Join(after, "\n") + "\n",
	}
}

// parseState represents the state of the digest parser.
type parseState int

const (
	stateBeforeFile parseState = iota
	stateFileHeader
	stateFileContent
)

// isContentMarker checks if a line is a digest content marker (not source content).
// These markers have a space after --- or +++ and describe the content type.
func isContentMarker(line string) bool {
	// Real content markers have format: "--- <description> ---" or "+++ <description>"
	// Examples: "--- untracked file content ---", "--- staged diff ---", "+++ b/path/file.go"
	return (strings.HasPrefix(line, "--- ") && strings.HasSuffix(line, " ---")) ||
		strings.HasPrefix(line, "+++ ")
}

// digestSection represents a parsed section of the digest.
type digestSection struct {
	Header        string
	Content       string
	Path          string
	Tracked       bool
	IsFileSection bool
}

// splitDigestSections parses the digest into file sections and other content.
// It uses state tracking: before_file -> file_header -> file_content.
// Once content starts, all lines (including those starting with ---, +++, REDACTION_POLICY:)
// are treated as content until the next file section marker.
func splitDigestSections(digest string) []digestSection {
	var sections []digestSection
	var currentSection digestSection
	lines := strings.Split(digest, "\n")

	state := stateBeforeFile

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Check for file section marker - this always transitions to a new section
		if strings.HasPrefix(line, "=== ") && strings.HasSuffix(line, " ===") {
			// Save previous section
			if currentSection.Header != "" || currentSection.Content != "" {
				sections = append(sections, currentSection)
			}

			// Start new section
			currentSection = digestSection{
				Header:        line + "\n",
				IsFileSection: true,
			}
			state = stateFileHeader

			// Extract path from marker
			path := strings.TrimPrefix(line, "=== ")
			path = strings.TrimSuffix(path, " ===")
			currentSection.Path = path

			i++
			continue
		}

		// Once in file content state, everything is content
		if state == stateFileContent {
			currentSection.Content += line + "\n"
			i++
			continue
		}

		// In file_header state: check for content marker transitions
		if state == stateFileHeader {
			// Check for untracked file content marker
			if strings.HasPrefix(line, "--- untracked file content ---") {
				currentSection.Header += line + "\n"
				currentSection.Tracked = false
				state = stateFileContent
				i++
				continue
			}

			// Check for staged/unstaged diff marker
			if strings.HasPrefix(line, "--- staged diff ---") ||
				strings.HasPrefix(line, "--- unstaged diff ---") ||
				strings.HasPrefix(line, "--- dirty diff ---") {
				currentSection.Header += line + "\n"
				if strings.Contains(line, "staged") {
					currentSection.Tracked = true
				}
				state = stateFileContent
				i++
				continue
			}

			// Check for staged/unstaged file marker (git diff format)
			if strings.HasPrefix(line, "--- a/") || strings.HasPrefix(line, "+++ a/") {
				currentSection.Header += line + "\n"
				state = stateFileContent
				i++
				continue
			}

			// Check for Metadata line
			if strings.HasPrefix(line, "Metadata:") {
				currentSection.Header += line + "\n"
				if strings.Contains(line, "tracked") && !strings.Contains(line, "untracked") {
					currentSection.Tracked = true
				}
				i++
				continue
			}

			// Empty line in file header - stay in header state
			if line == "" {
				currentSection.Header += line + "\n"
				i++
				continue
			}

			// If we see a non-header line, transition to content state
			// This handles formats where content appears directly after Metadata
			currentSection.Content += line + "\n"
			state = stateFileContent
			i++
			continue
		}

		// In before_file state: everything goes to header
		if state == stateBeforeFile {
			if currentSection.Header != "" || currentSection.Content != "" {
				currentSection.Header += line + "\n"
			} else {
				currentSection.Header += line + "\n"
			}
			i++
			continue
		}
		i++
	}

	// Don't forget the last section
	if currentSection.Header != "" || currentSection.Content != "" {
		sections = append(sections, currentSection)
	}

	return sections
}

// countSecretFindings counts how many secret-like patterns are in the content.
func countSecretFindings(content string) int {
	// Create a temporary path to scan
	warning := ScanSourceForSecrets("temp", content)
	return warning.FindingCount()
}

// SimpleRedactDigest applies standard redaction to the entire digest.
// This is the original behavior for backward compatibility.
func SimpleRedactDigest(digest string) string {
	return redact.RedactDigest(digest)
}

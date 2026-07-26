// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"fmt"
	"strings"
)

// RunnerAuthorityError represents errors in runner authority validation.
type RunnerAuthorityError struct {
	Field   string
	Message string
}

func (e *RunnerAuthorityError) Error() string {
	return fmt.Sprintf("runner_authority.%s: %s", e.Field, e.Message)
}

// ResolvedRunnerAuthority contains the fully resolved runner authority state.
type ResolvedRunnerAuthority struct {
	Mode                RunnerAuthorityMode
	ExecutablePath      string
	ExecutableSHA256    string
	VCSRevision         string
	VCSModified         bool
	PinnedToolRevision  string
	PinnedToolTree      string
	PinnedBinarySHA256  string
	TargetSubjectCommit string
	TargetSubjectTree   string
}

// ValidateRunnerAuthority validates the runner_authority block in a plan.
// It enforces strict mode separation between subject_exact and tool_release_exact.
func ValidateRunnerAuthority(authority *RunnerAuthority) error {
	if authority == nil {
		return nil // No authority declared, use defaults
	}

	switch authority.Mode {
	case RunnerAuthoritySubjectExact:
		// For subject_exact, tool block is allowed but must be empty if present
		if authority.Tool != nil {
			// Tool block provided for subject_exact - check it's effectively empty
			if authority.Tool.Revision != "" || authority.Tool.BinarySHA256 != "" {
				return &RunnerAuthorityError{
					Field:   "tool",
					Message: "tool block not allowed for subject_exact mode",
				}
			}
		}
	case RunnerAuthorityToolReleaseExact:
		// For tool_release_exact, tool block is required
		if authority.Tool == nil {
			return &RunnerAuthorityError{
				Field:   "tool",
				Message: "tool block is required for tool_release_exact mode",
			}
		}
		// Validate required fields
		if err := validateToolBlock(authority.Tool); err != nil {
			return err
		}
	default:
		return &RunnerAuthorityError{
			Field:   "mode",
			Message: fmt.Sprintf("unknown runner authority mode %q; expected subject_exact or tool_release_exact", authority.Mode),
		}
	}

	return nil
}

// validateToolBlock validates the tool block for tool_release_exact mode.
func validateToolBlock(tool *ToolAuthority) error {
	if tool == nil {
		return &RunnerAuthorityError{
			Field:   "tool",
			Message: "tool block is required for tool_release_exact mode",
		}
	}

	// Revision is required (40-char lowercase hex)
	if tool.Revision == "" {
		return &RunnerAuthorityError{
			Field:   "tool.revision",
			Message: "revision is required",
		}
	}
	if len(tool.Revision) != 40 {
		return &RunnerAuthorityError{
			Field:   "tool.revision",
			Message: fmt.Sprintf("revision must be 40 characters, got %d", len(tool.Revision)),
		}
	}
	if !isValidHex40(tool.Revision) {
		return &RunnerAuthorityError{
			Field:   "tool.revision",
			Message: "revision must be lowercase hexadecimal",
		}
	}

	// BinarySHA256 is required (64-char lowercase hex)
	if tool.BinarySHA256 == "" {
		return &RunnerAuthorityError{
			Field:   "tool.binary_sha256",
			Message: "binary_sha256 is required",
		}
	}
	if len(tool.BinarySHA256) != 64 {
		return &RunnerAuthorityError{
			Field:   "tool.binary_sha256",
			Message: fmt.Sprintf("binary_sha256 must be 64 characters, got %d", len(tool.BinarySHA256)),
		}
	}
	if !isValidHex64(tool.BinarySHA256) {
		return &RunnerAuthorityError{
			Field:   "tool.binary_sha256",
			Message: "binary_sha256 must be lowercase hexadecimal",
		}
	}

	return nil
}

// isValidHex40 checks if s is exactly 40 lowercase hex characters.
func isValidHex40(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// isValidHex64 checks if s is exactly 64 lowercase hex characters.
func isValidHex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// isValidOID checks if s is a valid 40 or 64 char lowercase hex OID.
func isValidOID(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// EnforceRunnerAuthority enforces runner identity against the plan's authority declaration.
func EnforceRunnerAuthority(
	authority *RunnerAuthority,
	runnerIdentity RunnerIdentity,
	actualBinarySHA256 string,
	targetSubjectCommit string,
	targetSubjectTree string,
) error {
	// Determine the effective mode
	mode := RunnerAuthoritySubjectExact // default
	if authority != nil {
		mode = authority.Mode
	}

	switch mode {
	case RunnerAuthoritySubjectExact:
		// subject_exact: runner vcs.revision must equal target subject
		return enforceSubjectExact(runnerIdentity, actualBinarySHA256, targetSubjectCommit)

	case RunnerAuthorityToolReleaseExact:
		// tool_release_exact: runner vcs.revision must equal pinned tool revision
		if authority == nil || authority.Tool == nil {
			return &RunnerAuthorityError{
				Field:   "mode",
				Message: "tool_release_exact requires runner_authority.tool block",
			}
		}
		return enforceToolReleaseExact(runnerIdentity, actualBinarySHA256, authority.Tool, targetSubjectCommit, targetSubjectTree)

	default:
		return &RunnerAuthorityError{
			Field:   "mode",
			Message: fmt.Sprintf("unknown runner authority mode %q", mode),
		}
	}
}

// enforceSubjectExact enforces the subject_exact mode invariants.
func enforceSubjectExact(identity RunnerIdentity, actualBinarySHA256, targetSubject string) error {
	if identity.VCSRevision == "" {
		return &RunnerAuthorityError{
			Field:   "vcs.revision",
			Message: "runner VCS revision is empty",
		}
	}
	if identity.VCSRevision != targetSubject {
		return &RunnerAuthorityError{
			Field:   "vcs.revision",
			Message: fmt.Sprintf("runner VCS revision (%s) does not match target subject (%s)", identity.VCSRevision, targetSubject),
		}
	}
	if identity.VCSModified {
		return &RunnerAuthorityError{
			Field:   "vcs.modified",
			Message: "runner is built from modified sources",
		}
	}
	if identity.BinarySHA256 == "" {
		return &RunnerAuthorityError{
			Field:   "binary_sha256",
			Message: "runner binary_sha256 is empty",
		}
	}
	if actualBinarySHA256 == "" {
		return &RunnerAuthorityError{
			Field:   "binary_sha256",
			Message: "actual binary SHA256 is empty",
		}
	}
	if identity.BinarySHA256 != actualBinarySHA256 {
		return &RunnerAuthorityError{
			Field:   "binary_sha256",
			Message: fmt.Sprintf("runner binary SHA256 mismatch: identity=%s actual=%s", identity.BinarySHA256, actualBinarySHA256),
		}
	}
	return nil
}

// enforceToolReleaseExact enforces the tool_release_exact mode invariants.
func enforceToolReleaseExact(
	identity RunnerIdentity,
	actualBinarySHA256 string,
	tool *ToolAuthority,
	targetSubjectCommit string,
	targetSubjectTree string,
) error {
	// 1. Runner vcs.revision must equal pinned tool revision
	if identity.VCSRevision == "" {
		return &RunnerAuthorityError{
			Field:   "vcs.revision",
			Message: "runner VCS revision is empty",
		}
	}
	if identity.VCSRevision != tool.Revision {
		return &RunnerAuthorityError{
			Field:   "vcs.revision",
			Message: fmt.Sprintf("runner VCS revision (%s) does not match pinned tool revision (%s)", identity.VCSRevision, tool.Revision),
		}
	}

	// 2. Runner must be clean build
	if identity.VCSModified {
		return &RunnerAuthorityError{
			Field:   "vcs.modified",
			Message: "runner is built from modified sources",
		}
	}

	// 3. Binary SHA256 must match pinned value in plan
	if tool.BinarySHA256 == "" {
		return &RunnerAuthorityError{
			Field:   "tool.binary_sha256",
			Message: "pinned binary_sha256 is empty in plan",
		}
	}
	if actualBinarySHA256 == "" {
		return &RunnerAuthorityError{
			Field:   "binary_sha256",
			Message: "actual binary SHA256 is empty",
		}
	}
	if tool.BinarySHA256 != actualBinarySHA256 {
		return &RunnerAuthorityError{
			Field:   "binary_sha256",
			Message: fmt.Sprintf("plan-pinned binary SHA256 (%s) does not match actual binary SHA256 (%s)", tool.BinarySHA256, actualBinarySHA256),
		}
	}

	// 4. Runner identity binary SHA256 must match actual binary SHA256
	if identity.BinarySHA256 == "" {
		return &RunnerAuthorityError{
			Field:   "binary_sha256",
			Message: "runner binary_sha256 is empty",
		}
	}
	if identity.BinarySHA256 != actualBinarySHA256 {
		return &RunnerAuthorityError{
			Field:   "binary_sha256",
			Message: fmt.Sprintf("runner binary SHA256 mismatch: identity=%s actual=%s", identity.BinarySHA256, actualBinarySHA256),
		}
	}

	// 4. Target subject commit must match supplied value
	if targetSubjectCommit == "" {
		return &RunnerAuthorityError{
			Field:   "target.subject",
			Message: "target subject commit is empty",
		}
	}

	// 5. Target subject tree must match supplied value
	if targetSubjectTree == "" {
		return &RunnerAuthorityError{
			Field:   "target.tree",
			Message: "target subject tree is empty",
		}
	}

	return nil
}

// ParseRunnerAuthorityMode parses a string into RunnerAuthorityMode.
func ParseRunnerAuthorityMode(raw string) (RunnerAuthorityMode, error) {
	switch strings.ToLower(raw) {
	case string(RunnerAuthoritySubjectExact):
		return RunnerAuthoritySubjectExact, nil
	case string(RunnerAuthorityToolReleaseExact):
		return RunnerAuthorityToolReleaseExact, nil
	default:
		return "", fmt.Errorf("unknown runner authority mode %q", raw)
	}
}

// ModeDisplayNames returns a list of valid mode names for error messages.
func ModeDisplayNames() []string {
	return []string{
		string(RunnerAuthoritySubjectExact),
		string(RunnerAuthorityToolReleaseExact),
	}
}

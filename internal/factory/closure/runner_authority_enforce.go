// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"fmt"
	"strings"
)

// EnforceRunnerAuthority enforces runner identity against the plan's authority declaration.
func EnforceRunnerAuthority(
	authority *RunnerAuthority,
	runnerIdentity RunnerIdentity,
	actualBinarySHA256 string,
	targetSubjectCommit string,
	targetSubjectTree string,
) error {
	mode := RunnerAuthoritySubjectExact
	if authority != nil {
		mode = authority.Mode
	}

	switch mode {
	case RunnerAuthoritySubjectExact:
		return enforceSubjectExact(runnerIdentity, actualBinarySHA256, targetSubjectCommit)
	case RunnerAuthorityToolReleaseExact:
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

func enforceSubjectExact(identity RunnerIdentity, actualBinarySHA256, targetSubject string) error {
	if identity.VCSRevision == "" {
		return &RunnerAuthorityError{Field: "vcs.revision", Message: "runner VCS revision is empty"}
	}
	if identity.VCSRevision != targetSubject {
		return &RunnerAuthorityError{
			Field:   "vcs.revision",
			Message: fmt.Sprintf("runner VCS revision (%s) does not match target subject (%s)", identity.VCSRevision, targetSubject),
		}
	}
	if identity.VCSModified {
		return &RunnerAuthorityError{Field: "vcs.modified", Message: "runner is built from modified sources"}
	}
	if identity.BinarySHA256 == "" {
		return &RunnerAuthorityError{Field: "binary_sha256", Message: "runner binary_sha256 is empty"}
	}
	if actualBinarySHA256 == "" {
		return &RunnerAuthorityError{Field: "binary_sha256", Message: "actual binary SHA256 is empty"}
	}
	if identity.BinarySHA256 != actualBinarySHA256 {
		return &RunnerAuthorityError{
			Field:   "binary_sha256",
			Message: fmt.Sprintf("runner binary SHA256 mismatch: identity=%s actual=%s", identity.BinarySHA256, actualBinarySHA256),
		}
	}
	return nil
}

func enforceToolReleaseExact(identity RunnerIdentity, actualBinarySHA256 string, tool *ToolAuthority, targetSubjectCommit, targetSubjectTree string) error {
	if identity.VCSRevision == "" {
		return &RunnerAuthorityError{Field: "vcs.revision", Message: "runner VCS revision is empty"}
	}
	if identity.VCSRevision != tool.Revision {
		return &RunnerAuthorityError{
			Field:   "vcs.revision",
			Message: fmt.Sprintf("runner VCS revision (%s) does not match pinned tool revision (%s)", identity.VCSRevision, tool.Revision),
		}
	}
	if identity.VCSModified {
		return &RunnerAuthorityError{Field: "vcs.modified", Message: "runner is built from modified sources"}
	}
	if identity.BinarySHA256 == "" {
		return &RunnerAuthorityError{Field: "binary_sha256", Message: "runner binary_sha256 is empty"}
	}
	if actualBinarySHA256 == "" {
		return &RunnerAuthorityError{Field: "binary_sha256", Message: "actual binary SHA256 is empty"}
	}
	if identity.BinarySHA256 != actualBinarySHA256 {
		return &RunnerAuthorityError{
			Field:   "binary_sha256",
			Message: fmt.Sprintf("runner binary SHA256 mismatch: identity=%s actual=%s", identity.BinarySHA256, actualBinarySHA256),
		}
	}
	if targetSubjectCommit == "" {
		return &RunnerAuthorityError{Field: "target.subject", Message: "target subject commit is empty"}
	}
	if targetSubjectTree == "" {
		return &RunnerAuthorityError{Field: "target.tree", Message: "target subject tree is empty"}
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

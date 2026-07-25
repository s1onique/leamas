// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"strings"
	"testing"
)

func TestValidateRunnerAuthority_NilAuthority(t *testing.T) {
	if err := ValidateRunnerAuthority(nil); err != nil {
		t.Errorf("ValidateRunnerAuthority(nil) = %v, want nil", err)
	}
}

func TestValidateRunnerAuthority_SubjectExact(t *testing.T) {
	auth := &RunnerAuthority{Mode: RunnerAuthoritySubjectExact}
	if err := ValidateRunnerAuthority(auth); err != nil {
		t.Errorf("ValidateRunnerAuthority(subject_exact) = %v, want nil", err)
	}
	authWithEmptyTool := &RunnerAuthority{Mode: RunnerAuthoritySubjectExact, Tool: &ToolAuthority{}}
	if err := ValidateRunnerAuthority(authWithEmptyTool); err != nil {
		t.Errorf("ValidateRunnerAuthority(subject_exact with empty tool) = %v, want nil", err)
	}
}

func TestValidateRunnerAuthority_SubjectExactWithToolRejects(t *testing.T) {
	auth := &RunnerAuthority{
		Mode: RunnerAuthoritySubjectExact,
		Tool: &ToolAuthority{
			Revision:     "0123456789abcdef0123456789abcdef01234567",
			BinarySHA256: strings.Repeat("a", 64),
		},
	}
	if err := ValidateRunnerAuthority(auth); err == nil {
		t.Error("ValidateRunnerAuthority(subject_exact with tool) should reject")
	}
}

func TestValidateRunnerAuthority_ToolReleaseExactRequiresTool(t *testing.T) {
	auth := &RunnerAuthority{Mode: RunnerAuthorityToolReleaseExact}
	if err := ValidateRunnerAuthority(auth); err == nil {
		t.Error("ValidateRunnerAuthority(tool_release_exact without tool) should fail")
	}
}

func TestValidateRunnerAuthority_ToolReleaseExactValid(t *testing.T) {
	auth := &RunnerAuthority{
		Mode: RunnerAuthorityToolReleaseExact,
		Tool: &ToolAuthority{
			Revision:     "0123456789abcdef0123456789abcdef01234567",
			BinarySHA256: strings.Repeat("a", 64),
		},
	}
	if err := ValidateRunnerAuthority(auth); err != nil {
		t.Errorf("ValidateRunnerAuthority(tool_release_exact) = %v, want nil", err)
	}
}

func TestValidateRunnerAuthority_ToolReleaseExactWithOptionalFields(t *testing.T) {
	auth := &RunnerAuthority{
		Mode: RunnerAuthorityToolReleaseExact,
		Tool: &ToolAuthority{
			Revision:     "0123456789abcdef0123456789abcdef01234567",
			TreeOID:      "fedcba9876543210fedcba9876543210fedcba98",
			BinarySHA256: strings.Repeat("b", 64),
			Version:      "v1.0.0",
			TagName:      "v1.0.0",
			TagObjectOID: "1234567890abcdef1234567890abcdef12345678",
		},
	}
	if err := ValidateRunnerAuthority(auth); err != nil {
		t.Errorf("ValidateRunnerAuthority(tool_release_exact with optional fields) = %v, want nil", err)
	}
}

func TestValidateRunnerAuthority_UnknownMode(t *testing.T) {
	auth := &RunnerAuthority{Mode: "unknown_mode"}
	if err := ValidateRunnerAuthority(auth); err == nil {
		t.Error("ValidateRunnerAuthority(unknown mode) should fail")
	}
}

func TestValidateToolBlock_InvalidRevision(t *testing.T) {
	cases := []struct {
		name      string
		revision  string
		wantError string
	}{
		{"empty", "", "revision is required"},
		{"too short", "abc123", "must be 40 characters"},
		{"too long", strings.Repeat("a", 41), "must be 40 characters"},
		{"uppercase", strings.ToUpper("0123456789abcdef0123456789abcdef01234567"), "lowercase"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			auth := &RunnerAuthority{
				Mode: RunnerAuthorityToolReleaseExact,
				Tool: &ToolAuthority{Revision: tt.revision, BinarySHA256: strings.Repeat("a", 64)},
			}
			err := ValidateRunnerAuthority(auth)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("ValidateRunnerAuthority() with revision=%q error = %v, want containing %q", tt.revision, err, tt.wantError)
			}
		})
	}
}

func TestValidateToolBlock_InvalidBinarySHA256(t *testing.T) {
	cases := []struct {
		name      string
		binarySHA string
		wantError string
	}{
		{"empty", "", "binary_sha256 is required"},
		{"too short", "abc123", "must be 64 characters"},
		{"too long", strings.Repeat("a", 65), "must be 64 characters"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			auth := &RunnerAuthority{
				Mode: RunnerAuthorityToolReleaseExact,
				Tool: &ToolAuthority{Revision: "0123456789abcdef0123456789abcdef01234567", BinarySHA256: tt.binarySHA},
			}
			err := ValidateRunnerAuthority(auth)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("ValidateRunnerAuthority() with binary_sha256=%q error = %v, want containing %q", tt.binarySHA, err, tt.wantError)
			}
		})
	}
}

func TestEnforceRunnerAuthority_SubjectExact(t *testing.T) {
	subject := "abc123def456abc789def123abc456def789abc"
	identity := RunnerIdentity{VCSRevision: subject, VCSModified: false, BinarySHA256: "deadbeef"}
	err := EnforceRunnerAuthority(&RunnerAuthority{Mode: RunnerAuthoritySubjectExact}, identity, "deadbeef", subject, "sometree")
	if err != nil {
		t.Errorf("EnforceRunnerAuthority(subject_exact) = %v, want nil", err)
	}
}

func TestEnforceRunnerAuthority_SubjectExactMismatch(t *testing.T) {
	identity := RunnerIdentity{VCSRevision: "abc123", VCSModified: false, BinarySHA256: "deadbeef"}
	err := EnforceRunnerAuthority(&RunnerAuthority{Mode: RunnerAuthoritySubjectExact}, identity, "deadbeef", "different_subject", "sometree")
	if err == nil {
		t.Error("EnforceRunnerAuthority(subject_exact) with mismatch should fail")
	}
}

func TestEnforceRunnerAuthority_SubjectExactModified(t *testing.T) {
	subject := "abc123"
	identity := RunnerIdentity{VCSRevision: subject, VCSModified: true, BinarySHA256: "deadbeef"}
	err := EnforceRunnerAuthority(&RunnerAuthority{Mode: RunnerAuthoritySubjectExact}, identity, "deadbeef", subject, "sometree")
	if err == nil {
		t.Error("EnforceRunnerAuthority(subject_exact) with modified sources should fail")
	}
}

func TestEnforceRunnerAuthority_ToolReleaseExact(t *testing.T) {
	toolRev := "abc123def456abc789def123abc456def789abc"
	tool := &ToolAuthority{Revision: toolRev, BinarySHA256: "cafebabe"}
	identity := RunnerIdentity{VCSRevision: toolRev, VCSModified: false, BinarySHA256: "cafebabe"}
	err := EnforceRunnerAuthority(&RunnerAuthority{Mode: RunnerAuthorityToolReleaseExact, Tool: tool}, identity, "cafebabe", "different_target", "target_tree")
	if err != nil {
		t.Errorf("EnforceRunnerAuthority(tool_release_exact) = %v, want nil", err)
	}
}

func TestEnforceRunnerAuthority_ToolReleaseExactToolRevisionMismatch(t *testing.T) {
	tool := &ToolAuthority{Revision: "abc123def456abc789def123abc456def789abc", BinarySHA256: "cafebabe"}
	identity := RunnerIdentity{VCSRevision: "different_revision", VCSModified: false, BinarySHA256: "cafebabe"}
	err := EnforceRunnerAuthority(&RunnerAuthority{Mode: RunnerAuthorityToolReleaseExact, Tool: tool}, identity, "cafebabe", "target", "tree")
	if err == nil {
		t.Error("EnforceRunnerAuthority(tool_release_exact) with tool revision mismatch should fail")
	}
}

func TestEnforceRunnerAuthority_ToolReleaseExactBinaryMismatch(t *testing.T) {
	tool := &ToolAuthority{Revision: "abc123def456abc789def123abc456def789abc", BinarySHA256: "cafebabe"}
	identity := RunnerIdentity{VCSRevision: "abc123def456abc789def123abc456def789abc", VCSModified: false, BinarySHA256: "cafebabe"}
	err := EnforceRunnerAuthority(&RunnerAuthority{Mode: RunnerAuthorityToolReleaseExact, Tool: tool}, identity, "deadbeef", "target", "tree")
	if err == nil {
		t.Error("EnforceRunnerAuthority(tool_release_exact) with actual binary mismatch should fail")
	}
}

func TestEnforceRunnerAuthority_ToolReleaseExactModified(t *testing.T) {
	tool := &ToolAuthority{Revision: "abc123def456abc789def123abc456def789abc", BinarySHA256: "cafebabe"}
	identity := RunnerIdentity{VCSRevision: "abc123def456abc789def123abc456def789abc", VCSModified: true, BinarySHA256: "cafebabe"}
	err := EnforceRunnerAuthority(&RunnerAuthority{Mode: RunnerAuthorityToolReleaseExact, Tool: tool}, identity, "cafebabe", "target", "tree")
	if err == nil {
		t.Error("EnforceRunnerAuthority(tool_release_exact) with modified sources should fail")
	}
}

func TestEnforceRunnerAuthority_ToolReleaseExactEmptySubject(t *testing.T) {
	tool := &ToolAuthority{Revision: "abc123def456abc789def123abc456def789abc", BinarySHA256: "cafebabe"}
	identity := RunnerIdentity{VCSRevision: "abc123def456abc789def123abc456def789abc", VCSModified: false, BinarySHA256: "cafebabe"}
	err := EnforceRunnerAuthority(&RunnerAuthority{Mode: RunnerAuthorityToolReleaseExact, Tool: tool}, identity, "cafebabe", "", "tree")
	if err == nil {
		t.Error("EnforceRunnerAuthority(tool_release_exact) with empty target subject should fail")
	}
}

func TestParseRunnerAuthorityMode(t *testing.T) {
	cases := []struct {
		input    string
		wantMode RunnerAuthorityMode
		wantErr  bool
	}{
		{"subject_exact", RunnerAuthoritySubjectExact, false},
		{"SUBJECT_EXACT", RunnerAuthoritySubjectExact, false},
		{"tool_release_exact", RunnerAuthorityToolReleaseExact, false},
		{"TOOL_RELEASE_EXACT", RunnerAuthorityToolReleaseExact, false},
		{"unknown", "", true},
	}
	for _, tt := range cases {
		t.Run(tt.input, func(t *testing.T) {
			mode, err := ParseRunnerAuthorityMode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRunnerAuthorityMode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if mode != tt.wantMode {
				t.Errorf("ParseRunnerAuthorityMode(%q) = %v, want %v", tt.input, mode, tt.wantMode)
			}
		})
	}
}

func TestModeDisplayNames(t *testing.T) {
	names := ModeDisplayNames()
	if len(names) != 2 {
		t.Errorf("ModeDisplayNames() = %v, want 2 modes", names)
	}
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["subject_exact"] || !found["tool_release_exact"] {
		t.Errorf("ModeDisplayNames() missing modes")
	}
}

func TestIsValidHex40(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"0123456789abcdef0123456789abcdef01234567", true},
		{"0123456789ABCDEF0123456789ABCDEF01234567", false},
		{"abc", false},
		{strings.Repeat("a", 40), true},
	}
	for _, tt := range cases {
		if got := isValidHex40(tt.input); got != tt.want {
			t.Errorf("isValidHex40(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIsValidHex64(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{strings.Repeat("a", 64), true},
		{strings.Repeat("A", 64), false},
		{"abc", false},
		{strings.Repeat("a", 63), false},
	}
	for _, tt := range cases {
		if got := isValidHex64(tt.input); got != tt.want {
			t.Errorf("isValidHex64(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestRunnerAuthorityError(t *testing.T) {
	err := &RunnerAuthorityError{Field: "mode", Message: "unknown mode"}
	if err.Error() != "runner_authority.mode: unknown mode" {
		t.Errorf("RunnerAuthorityError.Error() = %q, want %q", err.Error(), "runner_authority.mode: unknown mode")
	}
}

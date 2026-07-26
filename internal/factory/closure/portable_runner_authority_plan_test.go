// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"strings"
	"testing"
)

// TestToolReleaseExactRejectsPlanPinnedBinaryMismatch proves the isolated
// regression for the tool_release_exact mode: when the runner identity
// hash equals the actual hash but the plan-pinned hash differs, the
// runner must reject execution.
//
// This is the critical pinned binary mismatch test: the plan itself is
// authoritative, not the observed hash.
func TestToolReleaseExactRejectsPlanPinnedBinaryMismatch(t *testing.T) {
	toolRev := "abcd1234abcd1234abcd1234abcd1234abcd1234"
	planPinnedBinary := strings.Repeat("ab", 32) // 64-char hex
	actualBinary := strings.Repeat("cd", 32)     // 64-char hex, different from plan

	identity := RunnerIdentity{
		VCSRevision:  toolRev,
		VCSModified:  false,
		BinarySHA256: actualBinary, // Runner sees its own actual hash
	}

	// Plan says binary should be planPinnedBinary, but runner has actualBinary
	tool := &ToolAuthority{
		Revision:     toolRev,
		BinarySHA256: planPinnedBinary,
	}
	authority := &RunnerAuthority{
		Mode: RunnerAuthorityToolReleaseExact,
		Tool: tool,
	}

	// The critical test: identity hash == actual hash, but plan-pinned != actual.
	// This must REJECT because the plan pin is authoritative.
	err := EnforceRunnerAuthority(authority, identity, actualBinary, "target", "tree")
	if err == nil {
		t.Fatal("expected rejection when plan-pinned binary hash != actual runner hash")
	}
	if !strings.Contains(err.Error(), "binary_sha256") {
		t.Fatalf("expected binary_sha256 error, got: %v", err)
	}
}

// TestPlanValidationRunnerAuthorityContract asserts that plan validation
// correctly accepts valid runner_authority blocks and rejects invalid ones
// according to the mode-specific rules.
func TestPlanValidationRunnerAuthorityContract(t *testing.T) {
	validToolRev := "abcd1234abcd1234abcd1234abcd1234abcd1234"
	validBinary := strings.Repeat("a", 64)

	tests := []struct {
		name        string
		authority   *RunnerAuthority
		wantErr     bool
		errContains string
	}{
		{
			name:      "nil authority is valid",
			authority: nil,
			wantErr:   false,
		},
		{
			name:      "subject_exact without tool is valid",
			authority: &RunnerAuthority{Mode: RunnerAuthoritySubjectExact},
			wantErr:   false,
		},
		{
			name: "subject_exact with empty tool is valid",
			authority: &RunnerAuthority{
				Mode: RunnerAuthoritySubjectExact,
				Tool: &ToolAuthority{},
			},
			wantErr: false,
		},
		{
			name: "subject_exact with populated tool is invalid",
			authority: &RunnerAuthority{
				Mode: RunnerAuthoritySubjectExact,
				Tool: &ToolAuthority{
					Revision:     validToolRev,
					BinarySHA256: validBinary,
				},
			},
			wantErr:     true,
			errContains: "tool block not allowed",
		},
		{
			name:        "tool_release_exact without tool is invalid",
			authority:   &RunnerAuthority{Mode: RunnerAuthorityToolReleaseExact},
			wantErr:     true,
			errContains: "tool block is required",
		},
		{
			name: "tool_release_exact with valid tool is valid",
			authority: &RunnerAuthority{
				Mode: RunnerAuthorityToolReleaseExact,
				Tool: &ToolAuthority{
					Revision:     validToolRev,
					BinarySHA256: validBinary,
				},
			},
			wantErr: false,
		},
		{
			name: "tool_release_exact with missing revision is invalid",
			authority: &RunnerAuthority{
				Mode: RunnerAuthorityToolReleaseExact,
				Tool: &ToolAuthority{
					BinarySHA256: validBinary,
				},
			},
			wantErr:     true,
			errContains: "revision is required",
		},
		{
			name: "tool_release_exact with wrong revision length is invalid",
			authority: &RunnerAuthority{
				Mode: RunnerAuthorityToolReleaseExact,
				Tool: &ToolAuthority{
					Revision:     "too-short",
					BinarySHA256: validBinary,
				},
			},
			wantErr:     true,
			errContains: "40 characters",
		},
		{
			name: "tool_release_exact with uppercase revision is invalid",
			authority: &RunnerAuthority{
				Mode: RunnerAuthorityToolReleaseExact,
				Tool: &ToolAuthority{
					Revision:     strings.ToUpper(validToolRev),
					BinarySHA256: validBinary,
				},
			},
			wantErr:     true,
			errContains: "lowercase hexadecimal",
		},
		{
			name: "tool_release_exact with missing binary_sha256 is invalid",
			authority: &RunnerAuthority{
				Mode: RunnerAuthorityToolReleaseExact,
				Tool: &ToolAuthority{
					Revision: validToolRev,
				},
			},
			wantErr:     true,
			errContains: "binary_sha256 is required",
		},
		{
			name: "tool_release_exact with wrong binary_sha256 length is invalid",
			authority: &RunnerAuthority{
				Mode: RunnerAuthorityToolReleaseExact,
				Tool: &ToolAuthority{
					Revision:     validToolRev,
					BinarySHA256: "too-short",
				},
			},
			wantErr:     true,
			errContains: "64 characters",
		},
		{
			name: "unknown mode is invalid",
			authority: &RunnerAuthority{
				Mode: "unknown_mode",
			},
			wantErr:     true,
			errContains: "unknown runner authority mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRunnerAuthority(tt.authority)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ValidateRunnerAuthority() = nil, want error containing %q", tt.errContains)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("ValidateRunnerAuthority() error = %q, want containing %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Fatalf("ValidateRunnerAuthority() = %v, want nil", err)
				}
			}
		})
	}
}

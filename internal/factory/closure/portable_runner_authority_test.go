// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"strings"
	"testing"
)

// TestToolReleaseExactRejectsPlanPinnedBinaryMismatch verifies that
// enforceToolReleaseExact rejects when the plan-pinned binary SHA256
// does not match the actual binary SHA256.
func TestToolReleaseExactRejectsPlanPinnedBinaryMismatch(t *testing.T) {
	planPinnedHash := "2222222222222222222222222222222222222222222222222222222222222222"
	actualHash := "1111111111111111111111111111111111111111111111111111111111111111"

	tool := &ToolAuthority{
		Revision:     "0000000000000000000000000000000000000000",
		BinarySHA256: planPinnedHash, // Plan says this hash
	}

	identity := RunnerIdentity{
		VCSRevision:  "0000000000000000000000000000000000000000",
		VCSModified:  false,
		BinarySHA256: planPinnedHash, // Identity matches plan
	}

	// Actual binary hash differs from plan pin - should fail
	err := enforceToolReleaseExact(identity, actualHash, tool, "target-commit", "target-tree")
	if err == nil {
		t.Fatal("expected error for plan pin != actual hash, got nil")
	}

	if !strings.Contains(err.Error(), "binary_sha256") {
		t.Fatalf("expected binary_sha256 mismatch error, got: %v", err)
	}
}

// TestPlanValidationRunnerAuthorityContract verifies that ValidateRunnerAuthority
// correctly validates the runner_authority contract in a plan.
func TestPlanValidationRunnerAuthorityContract(t *testing.T) {
	tests := []struct {
		name    string
		auth    *RunnerAuthority
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil authority is valid",
			auth:    nil,
			wantErr: false,
		},
		{
			name: "subject_exact with empty tool is valid",
			auth: &RunnerAuthority{
				Mode: RunnerAuthoritySubjectExact,
				Tool: &ToolAuthority{},
			},
			wantErr: false,
		},
		{
			name: "subject_exact with non-empty revision fails",
			auth: &RunnerAuthority{
				Mode: RunnerAuthoritySubjectExact,
				Tool: &ToolAuthority{
					Revision: "0000000000000000000000000000000000000001",
				},
			},
			wantErr: true,
			errMsg:  "tool block not allowed for subject_exact mode",
		},
		{
			name: "subject_exact with non-empty binary_sha256 fails",
			auth: &RunnerAuthority{
				Mode: RunnerAuthoritySubjectExact,
				Tool: &ToolAuthority{
					BinarySHA256: "0000000000000000000000000000000000000000000000000000000000000000",
				},
			},
			wantErr: true,
			errMsg:  "tool block not allowed for subject_exact mode",
		},
		{
			name: "tool_release_exact without tool fails",
			auth: &RunnerAuthority{
				Mode: RunnerAuthorityToolReleaseExact,
			},
			wantErr: true,
			errMsg:  "tool block is required for tool_release_exact mode",
		},
		{
			name: "tool_release_exact with empty revision fails",
			auth: &RunnerAuthority{
				Mode: RunnerAuthorityToolReleaseExact,
				Tool: &ToolAuthority{
					BinarySHA256: "1111111111111111111111111111111111111111111111111111111111111111",
				},
			},
			wantErr: true,
			errMsg:  "revision is required",
		},
		{
			name: "tool_release_exact with wrong revision length fails",
			auth: &RunnerAuthority{
				Mode: RunnerAuthorityToolReleaseExact,
				Tool: &ToolAuthority{
					Revision:     "0000000001",
					BinarySHA256: "1111111111111111111111111111111111111111111111111111111111111111",
				},
			},
			wantErr: true,
			errMsg:  "revision must be 40 characters",
		},
		{
			name: "tool_release_exact with invalid revision chars fails",
			auth: &RunnerAuthority{
				Mode: RunnerAuthorityToolReleaseExact,
				Tool: &ToolAuthority{
					Revision:     "gggggggggggggggggggggggggggggggggggggggg",
					BinarySHA256: "1111111111111111111111111111111111111111111111111111111111111111",
				},
			},
			wantErr: true,
			errMsg:  "revision must be lowercase hexadecimal",
		},
		{
			name: "tool_release_exact with empty binary_sha256 fails",
			auth: &RunnerAuthority{
				Mode: RunnerAuthorityToolReleaseExact,
				Tool: &ToolAuthority{
					Revision: "0000000000000000000000000000000000000000",
				},
			},
			wantErr: true,
			errMsg:  "binary_sha256 is required",
		},
		{
			name: "tool_release_exact with wrong binary_sha256 length fails",
			auth: &RunnerAuthority{
				Mode: RunnerAuthorityToolReleaseExact,
				Tool: &ToolAuthority{
					Revision:     "0000000000000000000000000000000000000000",
					BinarySHA256: "1111111111",
				},
			},
			wantErr: true,
			errMsg:  "binary_sha256 must be 64 characters",
		},
		{
			name: "tool_release_exact with invalid binary_sha256 chars fails",
			auth: &RunnerAuthority{
				Mode: RunnerAuthorityToolReleaseExact,
				Tool: &ToolAuthority{
					Revision:     "0000000000000000000000000000000000000000",
					BinarySHA256: "gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg",
				},
			},
			wantErr: true,
			errMsg:  "binary_sha256 must be lowercase hexadecimal",
		},
		{
			name: "tool_release_exact with valid tool is valid",
			auth: &RunnerAuthority{
				Mode: RunnerAuthorityToolReleaseExact,
				Tool: &ToolAuthority{
					Revision:     "0000000000000000000000000000000000000000",
					BinarySHA256: "1111111111111111111111111111111111111111111111111111111111111111",
				},
			},
			wantErr: false,
		},
		{
			name: "unknown mode fails",
			auth: &RunnerAuthority{
				Mode: "unknown_mode",
			},
			wantErr: true,
			errMsg:  "unknown runner authority mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRunnerAuthority(tt.auth)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Fatalf("expected error containing %q, got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

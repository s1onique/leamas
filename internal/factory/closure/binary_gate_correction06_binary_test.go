// SPDX-License-Identifier: Apache-2.0

// binary_gate_correction06_binary_test.go owns the focused
// production-side tests for the R6-B B1 binary authority
// fail-closed surface added by ACT-CORRECTION06. The test
// proves that validateExactSubjectBinaryResult emits a
// typed V2Error with V2CodeR6BBinaryAuthorityInvalid for
// every field-level disagreement with the resolved S /
// S^{tree} authority.
//
// The test is split from binary_gate_correction06_test.go
// so each ACT-owned file stays under the LLM-friendly
// 400-line threshold while preserving the production
// assertion contract.

package closure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestR6BBinaryAuthorityResultValidation proves the
// validateExactSubjectBinaryResult contract. Each row
// mutates one field of the canonical r6BStubBuildFn result
// and asserts the validator emits
// V2CodeR6BBinaryAuthorityInvalid via the typed V2Error
// code authority (no substring fallback).
func TestR6BBinaryAuthorityResultValidation(t *testing.T) {
	t.Parallel()
	dir := r6BInitRepo(t)
	freeze := r6BMakeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/X.json": string(r6BValidPlanBytes()),
	})
	subject := r6BMakeCommit(t, dir, "subject", map[string]string{"f.txt": "x"})
	// Use a fresh directory for the binary path so the
	// stub build cannot collide with the file produced
	// by an earlier sub-test.
	binaryDir := t.TempDir()
	binaryPath := filepath.Join(binaryDir, "leamas")
	if err := os.WriteFile(binaryPath, []byte("fake\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	_ = r6BRequestFor(t, dir, freeze, subject)
	rows := []struct {
		name string
		mut  func(ExactSubjectBinaryResult) ExactSubjectBinaryResult
	}{
		{name: "valid", mut: func(r ExactSubjectBinaryResult) ExactSubjectBinaryResult { return r }},
		{
			name: "BinaryCommit mismatch",
			mut: func(r ExactSubjectBinaryResult) ExactSubjectBinaryResult {
				r.BinaryCommit = strings.Repeat("0", 40)
				return r
			},
		},
		{
			name: "SourceCommit mismatch",
			mut: func(r ExactSubjectBinaryResult) ExactSubjectBinaryResult {
				r.SourceCommit = strings.Repeat("0", 40)
				return r
			},
		},
		{
			name: "BinaryModified=true",
			mut: func(r ExactSubjectBinaryResult) ExactSubjectBinaryResult {
				r.BinaryModified = true
				return r
			},
		},
		{
			name: "SourceClean=false",
			mut: func(r ExactSubjectBinaryResult) ExactSubjectBinaryResult {
				r.SourceClean = false
				return r
			},
		},
		{
			name: "SourceDetached=false",
			mut: func(r ExactSubjectBinaryResult) ExactSubjectBinaryResult {
				r.SourceDetached = false
				return r
			},
		},
		{
			name: "OutputOutsideAllWorktrees=false",
			mut: func(r ExactSubjectBinaryResult) ExactSubjectBinaryResult {
				r.OutputOutsideAllWorktrees = false
				return r
			},
		},
		{
			name: "Executable=false",
			mut: func(r ExactSubjectBinaryResult) ExactSubjectBinaryResult {
				r.Executable = false
				return r
			},
		},
		{
			name: "empty BinaryPath",
			mut: func(r ExactSubjectBinaryResult) ExactSubjectBinaryResult {
				r.BinaryPath = ""
				return r
			},
		},
		{
			name: "invalid BinarySHA256",
			mut: func(r ExactSubjectBinaryResult) ExactSubjectBinaryResult {
				r.BinarySHA256 = "short"
				return r
			},
		},
		{
			name: "empty BinaryCommit",
			mut: func(r ExactSubjectBinaryResult) ExactSubjectBinaryResult {
				r.BinaryCommit = ""
				return r
			},
		},
		{
			name: "empty SourceCommit",
			mut: func(r ExactSubjectBinaryResult) ExactSubjectBinaryResult {
				r.SourceCommit = ""
				return r
			},
		},
		{
			name: "empty SourceTree",
			mut: func(r ExactSubjectBinaryResult) ExactSubjectBinaryResult {
				r.SourceTree = ""
				return r
			},
		},
		{
			name: "wrong SourceTree",
			mut: func(r ExactSubjectBinaryResult) ExactSubjectBinaryResult {
				r.SourceTree = strings.Repeat("0", 40)
				return r
			},
		},
	}
	expectedCommit := strings.Repeat("a", 40)
	expectedTree := strings.Repeat("b", 40)
	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			original := ExactSubjectBinaryResult{
				BinaryPath:                binaryPath,
				BinarySHA256:              strings.Repeat("a", 64),
				BinaryCommit:              expectedCommit,
				SourceCommit:              expectedCommit,
				SourceTree:                expectedTree,
				SourceClean:               true,
				SourceDetached:            true,
				OutputOutsideAllWorktrees: true,
				Executable:                true,
			}
			mutated := row.mut(original)
			vErr := validateExactSubjectBinaryResult(
				expectedCommit, expectedTree, mutated,
			)
			if row.name == "valid" {
				if vErr != nil {
					t.Fatalf("valid result must validate, got %v", vErr)
				}
				return
			}
			if vErr == nil {
				t.Fatalf("row %q must surface a typed binary authority failure", row.name)
			}
			// Typed V2Error authority: the first diagnostic
			// code MUST be V2CodeR6BBinaryAuthorityInvalid.
			// Substring matching the message is no longer
			// accepted as the authority.
			requireV2Code(t, vErr, V2CodeR6BBinaryAuthorityInvalid)
		})
	}
}

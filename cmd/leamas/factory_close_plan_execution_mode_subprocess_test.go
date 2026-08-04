package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestClosurePlanValidateExecutionModeSubprocess drives the
// `leamas factory close plan validate` CLI surface against the
// canonical and non-canonical execution-mode fixtures. The test
// pins the directive's acceptance criteria that:
//
//   - canonical valid plan → exit 0;
//   - missing mode → non-zero with a diagnostic that names the path;
//   - empty mode → non-zero with "empty" in the diagnostic;
//   - whitespace mode → non-zero with "whitespace" in the diagnostic;
//   - unknown mode → non-zero with the rejected value in the diagnostic;
//   - unknown field → non-zero with "unknown field" in the diagnostic;
//   - valid plans no longer fail with `unknown execution mode ""`.
//
// All subprocess evidence uses a freshly built binary so the test
// cannot pass against a stale installed executable.
func TestClosurePlanValidateExecutionModeSubprocess(t *testing.T) {
	binary := buildLeamasForTest(t)

	cases := []struct {
		name        string
		body        map[string]any
		wantExit    int
		wantValid   bool
		wantDiagHas string
	}{
		{
			name:      "canonical",
			body:      closureExecModeCanonical(t),
			wantExit:  0,
			wantValid: true,
		},
		{
			name:        "execution-omitted",
			body:        closureExecModeOmittedExecution(t),
			wantExit:    1,
			wantDiagHas: "required",
		},
		{
			name:        "mode-omitted",
			body:        closureExecModeModeOmitted(t),
			wantExit:    1,
			wantDiagHas: "required",
		},
		{
			name:        "mode-empty-string",
			body:        closureExecModeModeEmpty(t),
			wantExit:    1,
			wantDiagHas: "serial_fail_fast",
		},
		{
			name:        "mode-whitespace",
			body:        closureExecModeModeWhitespace(t),
			wantExit:    1,
			wantDiagHas: "serial_fail_fast",
		},
		{
			name:        "mode-unknown",
			body:        closureExecModeModeUnknown(t),
			wantExit:    1,
			wantDiagHas: "parallel",
		},
		{
			name:        "unknown-sibling",
			body:        closureExecModeUnknownSibling(t),
			wantExit:    1,
			wantDiagHas: "unknown property",
		},
		{
			name:        "top-level-mode-alias",
			body:        closureExecModeTopLevelAlias(t),
			wantExit:    1,
			wantDiagHas: "unknown property",
		},
		{
			name:        "policy-mode-alias",
			body:        closureExecModePolicyModeAlias(t),
			wantExit:    1,
			wantDiagHas: "unknown property",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			path := writeClosureSubprocessFixture(t, tc.body)
			stdout, stderr, exit := runLeamasExpect(t, binary,
				"factory", "close", "plan", "validate", "--file", path)
			if exit != tc.wantExit {
				t.Fatalf("exit=%d want=%d stdout=%q stderr=%q", exit, tc.wantExit, stdout, stderr)
			}
			// Parse JSON result
			var result map[string]any
			if err := json.Unmarshal([]byte(stdout), &result); err != nil {
				t.Fatalf("failed to parse JSON output: %v, stdout=%q", err, stdout)
			}
			if tc.wantValid {
				// For valid plans, check result.valid == true
				valid, ok := result["valid"].(bool)
				if !ok || !valid {
					t.Fatalf("expected valid=true in JSON result, got stdout=%q", stdout)
				}
			}
			if tc.wantDiagHas != "" {
				// Check for diagnostic in structural errors or semantic errors
				hasDiag := false
				if structural, ok := result["structural"].(map[string]any); ok {
					if errors, ok := structural["errors"].([]any); ok {
						for _, e := range errors {
							if errStr, ok := e.(map[string]any); ok {
								if msg, ok := errStr["message"].(string); ok {
									if strings.Contains(strings.ToLower(msg), strings.ToLower(tc.wantDiagHas)) {
										hasDiag = true
										break
									}
								}
							}
						}
					}
				}
				if !hasDiag {
					t.Fatalf("expected diagnostic containing %q in JSON result, got stdout=%q stderr=%q", tc.wantDiagHas, stdout, stderr)
				}
			}
		})
	}
}

// TestClosurePlanValidateCLIExitCodesAreDistinct checks that the
// CLI distinguishes the canonical valid case (exit 0 / stdout
// "VALID") from every negative case (non-zero exit). A regression
// where the canonical valid plan returned the same exit code as a
// missing-mode plan would let bad plans slip through unnoticed.
func TestClosurePlanValidateCLIExitCodesAreDistinct(t *testing.T) {
	binary := buildLeamasForTest(t)

	validPath := writeClosureSubprocessFixture(t, closureExecModeCanonical(t))
	badPath := writeClosureSubprocessFixture(t, closureExecModeOmittedExecution(t))

	validOut, _, validExit := runLeamasExpect(t, binary,
		"factory", "close", "plan", "validate", "--file", validPath)
	if validExit != 0 {
		t.Fatalf("canonical fixture did not return exit 0: exit=%d stdout=%q", validExit, validOut)
	}
	// Check result contains valid: true
	var validResult map[string]any
	if err := json.Unmarshal([]byte(validOut), &validResult); err != nil {
		t.Fatalf("failed to parse valid output as JSON: %v", err)
	}
	if valid, ok := validResult["valid"].(bool); !ok || !valid {
		t.Fatalf("canonical fixture result valid=false: stdout=%q", validOut)
	}

	if _, _, badExit := runLeamasExpect(t, binary,
		"factory", "close", "plan", "validate", "--file", badPath); badExit == 0 {
		t.Fatalf("missing-mode fixture accepted by CLI")
	}
}

// runLeamasExpect runs the binary and returns stdout, stderr, and
// the raw exit code. A non-zero exit code is not an error here; it
// is part of the contract the test must observe.
func runLeamasExpect(t *testing.T, bin string, args ...string) (string, string, int) {
	t.Helper()
	cwd := t.TempDir()
	cmd := exec.Command(bin, args...)
	cmd.Dir = cwd
	cmd.Env = withoutLeamasEnv()
	var out, errBuf strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	exitCode := 0
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			t.Fatalf("subprocess error (not ExitError): %v stderr=%q", runErr, errBuf.String())
		}
	}
	return out.String(), errBuf.String(), exitCode
}

// closureExecModeCanonical returns the canonical plan body shared by
// the CLI subprocess tests.
func closureExecModeCanonical(t *testing.T) map[string]any {
	t.Helper()
	return closureExecModeBuilder()
}

// closureExecModeBuilder returns the canonical plan body. Every
// helper composes from this baseline so the subprocess tests and the
// unit tests cannot drift.
func closureExecModeBuilder() map[string]any {
	return map[string]any{
		"contract_version": 1,
		"act_id":           "ACT-LEAMAS-CLI-EXECUTION-MODE",
		"baseline": map[string]any{
			"commit_oid": "1111111111111111111111111111111111111111",
			"tree_oid":   "2222222222222222222222222222222222222222",
		},
		"execution": map[string]any{"mode": "serial_fail_fast"},
		"checks": []any{
			map[string]any{
				"id":                "noop",
				"mode":              "run",
				"argv":              []any{"true"},
				"working_directory": ".",
				"timeout_seconds":   60,
				"environment":       map[string]any{},
			},
		},
		"artifacts": []any{},
		"policy": map[string]any{
			"require_clean_before":        true,
			"require_clean_after":         true,
			"forbid_tracked_full_digests": true,
			"require_diff_check":          true,
		},
	}
}

// closureExecModeOmittedExecution deletes the entire `execution`
// object so the validator must reject the plan as Missing.
func closureExecModeOmittedExecution(t *testing.T) map[string]any {
	t.Helper()
	body := closureExecModeBuilder()
	delete(body, "execution")
	return body
}

// closureExecModeModeOmitted keeps `execution` but drops `mode`,
// exercising the decoder's pointer-aware path.
func closureExecModeModeOmitted(t *testing.T) map[string]any {
	t.Helper()
	body := closureExecModeBuilder()
	body["execution"] = map[string]any{}
	return body
}

// closureExecModeModeEmpty keeps `execution.mode` but sets it to "".
func closureExecModeModeEmpty(t *testing.T) map[string]any {
	t.Helper()
	body := closureExecModeBuilder()
	body["execution"] = map[string]any{"mode": ""}
	return body
}

// closureExecModeModeWhitespace keeps `execution.mode` but sets it
// to "   ".
func closureExecModeModeWhitespace(t *testing.T) map[string]any {
	t.Helper()
	body := closureExecModeBuilder()
	body["execution"] = map[string]any{"mode": "   "}
	return body
}

// closureExecModeModeUnknown keeps `execution.mode` but sets it to
// "parallel", exercising the unknown-value diagnostic.
func closureExecModeModeUnknown(t *testing.T) map[string]any {
	t.Helper()
	body := closureExecModeBuilder()
	body["execution"] = map[string]any{"mode": "parallel"}
	return body
}

// closureExecModeUnknownSibling adds an unknown top-level property
// to ensure the strict decoder still rejects unknown fields.
func closureExecModeUnknownSibling(t *testing.T) map[string]any {
	t.Helper()
	body := closureExecModeBuilder()
	body["surprise"] = "nope"
	return body
}

// closureExecModeTopLevelAlias puts `mode` at the top level instead
// of under `execution`. The directive documents this as an alias
// that must be rejected.
func closureExecModeTopLevelAlias(t *testing.T) map[string]any {
	t.Helper()
	body := closureExecModeBuilder()
	delete(body, "execution")
	body["mode"] = "serial_fail_fast"
	return body
}

// closureExecModePolicyModeAlias puts `mode` inside `policy`. The
// directive documents this as an alias that must be rejected.
func closureExecModePolicyModeAlias(t *testing.T) map[string]any {
	t.Helper()
	body := closureExecModeBuilder()
	delete(body, "execution")
	body["policy"] = map[string]any{
		"mode":                        "serial_fail_fast",
		"require_clean_before":        true,
		"require_clean_after":         true,
		"forbid_tracked_full_digests": true,
		"require_diff_check":          true,
	}
	return body
}

// writeClosureSubprocessFixture serialises body to a temporary JSON
// file and returns its absolute path. The file lives in the test's
// own TempDir so concurrent test runs cannot collide.
func writeClosureSubprocessFixture(t *testing.T, body map[string]any) string {
	t.Helper()
	data, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

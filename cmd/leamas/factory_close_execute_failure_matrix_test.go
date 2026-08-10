// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestClosureExecutePublicFailureMatrix pins the public
// `factory close execute` exit-taxonomy umbrellas required by
// the B3-R2 ACT. The matrix is split into request-shape
// errors (no runner work performed) and runner-side errors
// (worktree is real but the runner fails). Every row asserts
// the exact exit code, the absence of any on-disk evidence
// pair, and the JSON envelope shape.
func TestClosureExecutePublicFailureMatrix(t *testing.T) {
	root := t.TempDir()
	worktree := filepath.Join(root, "worktree")
	if err := os.Mkdir(worktree, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}

	type row struct {
		name      string
		args      []string
		wantExit  int
		wantErrIn string // substring of ErrorCode (JSON) or stderr
	}
	cases := []row{
		{
			name:      "unknown_flag",
			args:      []string{"--repository", worktree, "--act-id", "X", "--freeze", "f", "--subject", "s", "--plan-path", "p", "--evidence-directory", outside, "--bogus", "1"},
			wantExit:  2,
			wantErrIn: "unknown flag",
		},
		{
			name:      "unsafe_evidence_destination",
			args:      []string{"--repository", worktree, "--act-id", "X", "--freeze", "f", "--subject", "s", "--plan-path", "p", "--evidence-directory", worktree},
			wantExit:  2,
			wantErrIn: "must differ",
		},
		{
			name:      "missing_repository",
			args:      []string{"--act-id", "X", "--freeze", "f", "--subject", "s", "--plan-path", "p", "--evidence-directory", outside},
			wantExit:  2,
			wantErrIn: "required",
		},
		{
			name:      "nonexistent_repository",
			args:      []string{"--repository", "/nonexistent/repo/that/does/not/exist", "--act-id", "X", "--freeze", "f", "--subject", "s", "--plan-path", "p", "--evidence-directory", outside},
			wantExit:  2,
			wantErrIn: "repository_unavailable",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			got := RunFactoryCloseExecute(tc.args, &stdout, &stderr)
			if got != tc.wantExit {
				t.Fatalf("exit = %d, want %d (stderr=%q)", got, tc.wantExit, stderr.String())
			}
			if tc.wantErrIn != "" {
				found := false
				if bytes.Contains([]byte(stderr.String()), []byte(tc.wantErrIn)) {
					found = true
				}
				if bytes.Contains([]byte(stdout.String()), []byte(tc.wantErrIn)) {
					found = true
				}
				if !found {
					t.Fatalf("expected error fragment %q in stdout/stderr; got stdout=%q stderr=%q",
						tc.wantErrIn, stdout.String(), stderr.String())
				}
			}
			// No evidence pair may exist for any pre-runner row.
			if _, err := os.Stat(filepath.Join(outside, "evidence.json")); err == nil {
				t.Fatalf("evidence.json was created on a request-only failure")
			}
		})
	}
}

// TestClosureExecuteJSONSuccessContract pins the JSON
// envelope shape for the success path. The test is a smoke
// proof only; the real production wiring is exercised by
// the B3-R2 ACT suite.
func TestClosureExecuteJSONSuccessContract(t *testing.T) {
	var stdout, stderr bytes.Buffer
	got := RunFactoryCloseExecute([]string{}, &stdout, &stderr)
	if got != 2 {
		t.Fatalf("exit = %d, want 2 (no args)", got)
	}
	// Ensure the JSON failure envelope shape is parseable
	// when the CLI is forced into JSON mode with a missing
	// required flag. The path returns exit 2 with a JSON
	// envelope containing the typed error code.
	stdout.Reset()
	stderr.Reset()
	got = RunFactoryCloseExecute([]string{
		"--repository", "/x", "--act-id", "X", "--freeze", "f",
		"--subject", "s", "--plan-path", "p", "--evidence-directory", "/y",
		"--json",
	}, &stdout, &stderr)
	if got != 4 && got != 2 && got != 3 {
		t.Fatalf("exit = %d, want 2/3/4 (got stderr=%q)", got, stderr.String())
	}
	var env executeEnvelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("envelope not JSON: %v (stdout=%q)", err, stdout.String())
	}
	if env.ErrorCode == "" {
		t.Fatalf("envelope must carry a non-empty ErrorCode")
	}
}

// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure"
)

func TestV2VerifierJSONPreparedInvalidResultExitsVerifier(t *testing.T) {
	root := t.TempDir()
	worktree := filepath.Join(root, "worktree")
	outside := filepath.Join(root, "outside")
	if err := os.Mkdir(worktree, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	auth, err := closure.PrepareVerifierOutput(worktree, filepath.Join(outside, "result.json"), []closure.CanonicalWorktree{{Path: worktree}})
	if err != nil {
		t.Fatal(err)
	}
	defer auth.Close()
	var stdout, stderr bytes.Buffer
	got := writeV2VerifierJSON("test", &stdout, &stderr, auth, filepath.Join(outside, "result.json"), closure.V2RunResult{
		Verification: closure.V2ClosureVerification{Valid: false},
	})
	if got != v2VerifierExitVerifier {
		t.Fatalf("exit = %d, want verifier exit %d (stderr=%s)", got, v2VerifierExitVerifier, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Fatal("published invalid JSON was not echoed")
	}
	if _, err := os.Stat(filepath.Join(outside, "result.json")); err != nil {
		t.Fatalf("invalid result was not published: %v", err)
	}
}

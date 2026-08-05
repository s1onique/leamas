package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runLeamasInDir runs the leamas binary from the supplied
// working directory with the supplied arguments and returns
// the exit code, stdout, stderr, and SHA-256 of stdout. The
// function builds the binary once per test and reuses it
// across invocations so the test is bounded.
//
// The test framework's parent Leamas execution sets the
// LEAMAS_EXEC_ROOT_ID / LEAMAS_EXEC_PARENT_PID /
// LEAMAS_EXEC_GENERATION environment variables. These cause
// the child invocation to fail closed (Leamas cannot be
// nested). The test clears them so the child runs as a
// fresh root execution.
func runLeamasInDir(t *testing.T, binDir, workDir string, args ...string) (int, []byte, []byte, string) {
	t.Helper()
	cmd := exec.Command(filepath.Join(binDir, "leamas"), args...)
	cmd.Dir = workDir
	cmd.Env = stripLeamasExecEnv(os.Environ())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exit := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exit = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("leamas %s: %v", strings.Join(args, " "), err)
	}
	sum := sha256.Sum256(stdout.Bytes())
	return exit, stdout.Bytes(), stderr.Bytes(), hex.EncodeToString(sum[:])
}

// stripLeamasExecEnv removes the LEAMAS_EXEC_* environment
// variables that mark a process as nested under an outer
// Leamas execution. Without the strip the spawned binary
// refuses to start.
func stripLeamasExecEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		if strings.HasPrefix(e, "LEAMAS_EXEC_") {
			continue
		}
		out = append(out, e)
	}
	return out
}

// TestSchemaExampleDeterminismTwoDirs runs the schema and
// example commands from two distinct clean temporary
// directories and proves the stdout bytes are identical and
// the SHA-256 digests match. CORRECTION16 binds the public
// CLI to the deterministic closure authority: a transient
// host-path, environment variable, or timestamp must never
// leak into the command output.
func TestSchemaExampleDeterminismTwoDirs(t *testing.T) {
	binDir := buildLeamasBinary(t)

	dirA := t.TempDir()
	dirB := t.TempDir()

	type cmdCase struct {
		name string
		args []string
	}
	cases := []cmdCase{
		{"schema", []string{"factory", "close", "plan", "schema"}},
		{"example", []string{"factory", "close", "plan", "example"}},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			exitA, stdoutA, stderrA, shaA := runLeamasInDir(t, binDir, dirA, c.args...)
			exitB, stdoutB, stderrB, shaB := runLeamasInDir(t, binDir, dirB, c.args...)

			if exitA != 0 {
				t.Fatalf("%s: dir A exit = %d, want 0 (stderr=%q)", c.name, exitA, stderrA)
			}
			if exitB != 0 {
				t.Fatalf("%s: dir B exit = %d, want 0 (stderr=%q)", c.name, exitB, stderrB)
			}
			if !bytes.Equal(stdoutA, stdoutB) {
				t.Fatalf("%s: stdout bytes differ across directories (A=%d bytes, B=%d bytes)",
					c.name, len(stdoutA), len(stdoutB))
			}
			if shaA != shaB {
				t.Fatalf("%s: SHA-256 mismatch A=%s B=%s", c.name, shaA, shaB)
			}
			// Two runs from the SAME directory must also
			// produce identical bytes; this is the
			// single-run determinism clause of the ACT.
			exitA2, stdoutA2, stderrA2, shaA2 := runLeamasInDir(t, binDir, dirA, c.args...)
			if exitA2 != 0 {
				t.Fatalf("%s: second run exit = %d, want 0 (stderr=%q)",
					c.name, exitA2, stderrA2)
			}
			if !bytes.Equal(stdoutA, stdoutA2) || shaA != shaA2 {
				t.Fatalf("%s: second run differs from first (len=%d vs %d, sha=%s vs %s)",
					c.name, len(stdoutA), len(stdoutA2), shaA, shaA2)
			}
		})
	}
}

// buildLeamasBinary compiles the leamas binary into a
// per-test temporary directory so the test never touches
// the repository's bin/ output. The binary is removed
// automatically when the test finishes.
func buildLeamasBinary(t *testing.T) string {
	t.Helper()
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "leamas")
	// Walk up from cwd until we find go.mod; that directory
	// is the repository root and the right place to invoke
	// `go build ./cmd/leamas`. The walk is bounded so a
	// transient cwd outside the repo still resolves.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repoRoot := cwd
	for i := 0; i < 16; i++ {
		if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			t.Fatalf("could not locate go.mod from cwd=%s", cwd)
		}
		repoRoot = parent
	}
	cmd := exec.Command("go", "build", "-trimpath", "-o", binPath, "./cmd/leamas")
	cmd.Dir = repoRoot
	cmd.Env = stripLeamasExecEnv(append(os.Environ(), "CGO_ENABLED=0"))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build: %v (stderr=%q)", err, stderr.String())
	}
	return binDir
}

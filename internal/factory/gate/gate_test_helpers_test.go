// Package gate provides test helpers for gate tests.
package gate

import (
	"os"
	"path/filepath"
	"testing"
)

func buildTestEnv(t *testing.T, overrides map[string]string) []string {
	t.Helper()
	// Explicitly clear all ambient gate-related variables
	// to ensure tests are deterministic regardless of shell environment
	env := []string{
		"HOME=" + os.Getenv("HOME"),
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"TMPDIR=" + os.Getenv("TMPDIR"),
		// Clear gate control variables
		"LEAMAS_GATE_CALLER=",
		"LEAMAS_ALLOW_FULL_GATE=",
		// Clear editor detection variables
		"TERM_PROGRAM=",
		"VSCODE_PID=",
	}
	for k, v := range overrides {
		env = append(env, k+"="+v)
	}
	return env
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repository root")
		}
		dir = parent
	}
}

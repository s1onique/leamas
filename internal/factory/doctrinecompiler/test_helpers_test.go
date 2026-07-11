package doctrinecompiler

import (
	"os"
	"testing"
)

// TestMain installs a default compiler-version source that satisfies
// the canonical pack's compatibility constraint, so library tests
// that compile with empty CompilerOptions still pass. Individual tests
// may override via withCompilerVersion(t, ...).
func TestMain(m *testing.M) {
	compilerVersionSource = func() string { return "0.1.0" }
	os.Exit(m.Run())
}

// withCompilerVersion sets the current compiler version source for the
// duration of the test. It restores the original source on cleanup.
func withCompilerVersion(t *testing.T, v string) {
	t.Helper()
	prev := compilerVersionSource
	compilerVersionSource = func() string { return v }
	t.Cleanup(func() { compilerVersionSource = prev })
}

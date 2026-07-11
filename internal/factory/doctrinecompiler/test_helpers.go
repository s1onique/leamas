package doctrinecompiler

import "testing"

// withCompilerVersion sets the current compiler version source for the
// duration of the test. It restores the original source on cleanup.
func withCompilerVersion(t *testing.T, v string) {
	t.Helper()
	prev := compilerVersionSource
	compilerVersionSource = func() string { return v }
	t.Cleanup(func() { compilerVersionSource = prev })
}

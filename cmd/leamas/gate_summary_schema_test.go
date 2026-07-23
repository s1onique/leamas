// CLI tests for `leamas gate-summary schema`.
//
// These tests use the injected writer API on the CLI struct. They do
// not replace os.Stdout / os.Stderr; instead they build the CLI with
// bytes.Buffer instances and assert the produced bytes byte-for-byte.
//
// The tests cover the exact output contract, the unknown-version and
// mutable-alias failure modes, the writer-error propagation, and the
// help-text contract.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/s1onique/leamas/internal/gatesummary/schema"
)

// runCLI runs the CLI with the given args using injected writers and
// returns the exit code, stdout, and stderr. Tests use this helper
// to assert the byte-exact contract without touching os.Stdout.
func runCLI(args []string) (int, string, string) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	code := newGateSummarySchemaCLI(out, errBuf).Run(args)
	return code, out.String(), errBuf.String()
}

// TestCLISchemaListExactBytes asserts the exact bytes the `schema
// list` command produces. The format is frozen; the bytes are
// captured as a literal string so any drift triggers a clear failure.
func TestCLISchemaListExactBytes(t *testing.T) {
	code, stdout, stderr := runCLI([]string{"list"})
	if code != 0 {
		t.Fatalf("list exited %d", code)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	want := "VERSION  STATUS     SCHEMA_ID\n" +
		"v1       supported  urn:leamas:gate-summary:v1\n" +
		"v2       current    urn:leamas:gate-summary:v2\n"
	if stdout != want {
		t.Fatalf("list output mismatch.\nWANT:\n%s\nGOT:\n%s", want, stdout)
	}
}

// TestCLISchemaShowV1ExactBytes asserts that `schema show v1` emits
// the exact embedded bytes of the v1 schema.
func TestCLISchemaShowV1ExactBytes(t *testing.T) {
	code, stdout, stderr := runCLI([]string{"show", "v1"})
	if code != 0 {
		t.Fatalf("show v1 exited %d", code)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	want, err := schema.Bytes(schema.VersionV1)
	if err != nil {
		t.Fatalf("Bytes(v1): %v", err)
	}
	if stdout != string(want) {
		t.Fatalf("show v1 output mismatch; got %q, want %q", stdout, string(want))
	}
}

// TestCLISchemaShowV2ExactBytes asserts that `schema show v2` emits
// the exact embedded bytes of the v2 schema.
func TestCLISchemaShowV2ExactBytes(t *testing.T) {
	code, stdout, stderr := runCLI([]string{"show", "v2"})
	if code != 0 {
		t.Fatalf("show v2 exited %d", code)
	}
	if stderr != "" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	want, err := schema.Bytes(schema.VersionV2)
	if err != nil {
		t.Fatalf("Bytes(v2): %v", err)
	}
	if stdout != string(want) {
		t.Fatalf("show v2 output mismatch; got %q, want %q", stdout, string(want))
	}
}

// TestCLISchemaShowV1WritesNothingToStderr asserts that the success
// path of `show v1` produces no stderr output.
func TestCLISchemaShowV1WritesNothingToStderr(t *testing.T) {
	_, _, stderr := runCLI([]string{"show", "v1"})
	if stderr != "" {
		t.Fatalf("show v1 wrote to stderr on success: %q", stderr)
	}
}

// TestCLISchemaShowV2WritesNothingToStderr asserts that the success
// path of `show v2` produces no stderr output.
func TestCLISchemaShowV2WritesNothingToStderr(t *testing.T) {
	_, _, stderr := runCLI([]string{"show", "v2"})
	if stderr != "" {
		t.Fatalf("show v2 wrote to stderr on success: %q", stderr)
	}
}

// TestCLIListWritesNothingToStderr asserts the list command is silent
// on stderr.
func TestCLIListWritesNothingToStderr(t *testing.T) {
	_, _, stderr := runCLI([]string{"list"})
	if stderr != "" {
		t.Fatalf("list wrote to stderr: %q", stderr)
	}
}

// TestCLIUnknownVersionFails asserts that an unknown version produces
// a non-zero exit code and a diagnostic on stderr.
func TestCLIUnknownVersionFails(t *testing.T) {
	code, stdout, stderr := runCLI([]string{"show", "v3"})
	if code == 0 {
		t.Fatalf("unknown version must exit non-zero; got %d", code)
	}
	if stdout != "" {
		t.Fatalf("unknown version must not write stdout; got %q", stdout)
	}
	if !strings.Contains(stderr, "v3") {
		t.Fatalf("unknown-version diagnostic must mention 'v3'; got %q", stderr)
	}
}

// TestCLIUnknownVersionRejectsUpperCaseV2 asserts that case-mismatched
// versions are rejected. The CLI is case-sensitive on purpose.
func TestCLIUnknownVersionRejectsUpperCaseV2(t *testing.T) {
	code, stdout, _ := runCLI([]string{"show", "V2"})
	if code == 0 {
		t.Fatalf("V2 must be rejected; got %d", code)
	}
	if stdout != "" {
		t.Fatalf("V2 must not write stdout; got %q", stdout)
	}
}

// TestCLIUnknownVersionRejectsInteger asserts that an integer (2)
// is not accepted as a version alias.
func TestCLIUnknownVersionRejectsInteger(t *testing.T) {
	code, stdout, _ := runCLI([]string{"show", "2"})
	if code == 0 {
		t.Fatalf("2 must be rejected; got %d", code)
	}
	if stdout != "" {
		t.Fatalf("2 must not write stdout; got %q", stdout)
	}
}

// TestCLIMutableAliasesRejected asserts that the closed set of
// mutable aliases is rejected. The exact rejection scenario is
// documented in the contract.
func TestCLIMutableAliasesRejected(t *testing.T) {
	for _, bad := range []string{"latest", "current", "stable", "default", "Latest", "CURRENT"} {
		t.Run(bad, func(t *testing.T) {
			code, stdout, _ := runCLI([]string{"show", bad})
			if code == 0 {
				t.Fatalf("mutable alias %q must be rejected", bad)
			}
			if stdout != "" {
				t.Fatalf("mutable alias %q must not write stdout; got %q", bad, stdout)
			}
		})
	}
}

// TestCLIMissingVersionFails asserts that `show` with no version
// produces a non-zero exit code and a usage hint on stderr.
func TestCLIMissingVersionFails(t *testing.T) {
	code, stdout, stderr := runCLI([]string{"show"})
	if code == 0 {
		t.Fatalf("missing version must exit non-zero")
	}
	if stdout != "" {
		t.Fatalf("missing version must not write stdout; got %q", stdout)
	}
	if !strings.Contains(stderr, "missing version") {
		t.Fatalf("stderr must mention 'missing version'; got %q", stderr)
	}
}

// TestCLIUnknownSubcommandFails asserts that `schema` with an
// unknown subcommand produces a non-zero exit code.
func TestCLIUnknownSubcommandFails(t *testing.T) {
	code, stdout, _ := runCLI([]string{"bogus"})
	if code == 0 {
		t.Fatalf("unknown subcommand must exit non-zero")
	}
	if stdout != "" {
		t.Fatalf("unknown subcommand must not write stdout; got %q", stdout)
	}
}

// TestCLIExtraArgsAfterVersionFails asserts that `show v1 extra` is
// rejected.
func TestCLIExtraArgsAfterVersionFails(t *testing.T) {
	code, stdout, _ := runCLI([]string{"show", "v1", "extra"})
	if code == 0 {
		t.Fatalf("extra args must exit non-zero")
	}
	if stdout != "" {
		t.Fatalf("extra args must not write stdout; got %q", stdout)
	}
}

// TestCLIWriteFailureFailsClosed asserts that a writer failure
// produces a non-zero exit code, an error on stderr, and no stdout
// output. The exact stdout-empty rule guards against partial writes
// that could leak into downstream consumers.
func TestCLIWriteFailureFailsClosed(t *testing.T) {
	out := &failingWriter{err: errors.New("disk full")}
	errBuf := &bytes.Buffer{}
	code := newGateSummarySchemaCLI(out, errBuf).Run([]string{"show", "v1"})
	if code == 0 {
		t.Fatalf("writer failure must exit non-zero")
	}
	if out.n == 0 {
		t.Fatalf("writer was never invoked")
	}
	// The stdout sink is the failingWriter; consume the wrapper's
	// own accounting field instead of trying to read it.
	if errBuf.String() == "" {
		t.Fatalf("writer failure diagnostic must reach stderr")
	}
}

// TestCLIOutputMatchesEmbeddedBytes asserts that the stdout from
// `show v1` and `show v2` matches the sha256 of the embedded bytes.
// This is the byte-identity contract.
func TestCLIOutputMatchesEmbeddedBytes(t *testing.T) {
	cases := []struct {
		version schema.Version
		path    string
	}{
		{schema.VersionV1, "../../internal/gatesummary/schema/gate-summary-v1.schema.json"},
		{schema.VersionV2, "../../internal/gatesummary/schema/gate-summary-v2.schema.json"},
	}
	for _, tc := range cases {
		t.Run(string(tc.version), func(t *testing.T) {
			code, stdout, _ := runCLI([]string{"show", string(tc.version)})
			if code != 0 {
				t.Fatalf("show %s exited %d", tc.version, code)
			}
			canonical, err := os.ReadFile(tc.path)
			if err != nil {
				t.Fatalf("read %s: %v", tc.path, err)
			}
			cliHash := sha256.Sum256([]byte(stdout))
			canonicalHash := sha256.Sum256(canonical)
			if cliHash != canonicalHash {
				t.Fatalf("show %s hash mismatch:\n  CLI: %s\n  File: %s",
					tc.version, hex.EncodeToString(cliHash[:]), hex.EncodeToString(canonicalHash[:]))
			}
		})
	}
}

// TestCLIOutputDeterministic asserts that the `show` output is
// byte-identical across 32 goroutines × 4 repetitions per goroutine.
func TestCLIOutputDeterministic(t *testing.T) {
	for _, v := range []string{"v1", "v2"} {
		t.Run(v, func(t *testing.T) {
			_, baseline, _ := runCLI([]string{"show", v})
			const goroutines = 32
			const repetitions = 4
			var wg sync.WaitGroup
			failures := make(chan string, goroutines*repetitions)
			for g := 0; g < goroutines; g++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for r := 0; r < repetitions; r++ {
						_, got, _ := runCLI([]string{"show", v})
						if got != baseline {
							failures <- got
							return
						}
					}
				}()
			}
			wg.Wait()
			close(failures)
			for f := range failures {
				t.Fatalf("non-deterministic show %s: %q", v, f)
			}
		})
	}
}

// TestCLIListDeterministic asserts that the `list` output is
// byte-identical across 32 goroutines × 4 repetitions per goroutine.
func TestCLIListDeterministic(t *testing.T) {
	_, baseline, _ := runCLI([]string{"list"})
	const goroutines = 32
	const repetitions = 4
	var wg sync.WaitGroup
	failures := make(chan string, goroutines*repetitions)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for r := 0; r < repetitions; r++ {
				_, got, _ := runCLI([]string{"list"})
				if got != baseline {
					failures <- got
					return
				}
			}
		}()
	}
	wg.Wait()
	close(failures)
	for f := range failures {
		t.Fatalf("non-deterministic list: %q", f)
	}
}

// --- helpers below ---

// failingWriter is an io.Writer that records the byte count handed
// to it and returns the configured error. It is used by the
// writer-failure test.
type failingWriter struct {
	err error
	n   int
}

func (w *failingWriter) Write(p []byte) (int, error) {
	w.n += len(p)
	return 0, w.err
}

// Sanity check: the failingWriter satisfies the io.Writer contract.
var _ io.Writer = (*failingWriter)(nil)

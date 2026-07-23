package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildLeamasForTest builds the leamas binary into t.TempDir() and
// returns the path. The build uses the same module the tested
// package lives in, so the resulting binary reflects the current
// working tree.
func buildLeamasForTest(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "leamas-test")
	cmd := exec.Command("go", "build", "-trimpath", "-o", bin, "github.com/s1onique/leamas/cmd/leamas")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build: %v", err)
	}
	return bin
}

// runLeamas runs the binary in a separate process, working from a
// temporary directory that is not the source tree. The test
// asserts the subprocess end-to-end behavior: main,
// gate-summary, schema, stdout, stderr, exit code.
func runLeamas(t *testing.T, bin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cwd := t.TempDir()
	cmd := exec.Command(bin, args...)
	cmd.Dir = cwd
	// Strip the Leamas re-entry fuse variables so the subprocess is
	// not blocked by the production emergency-reentry check.
	cmd.Env = withoutLeamasEnv()
	var out, errBuf strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	return out.String(), errBuf.String(), runErr
}

// headBytes returns the first 16 bytes of data with non-printable
// bytes escaped.
func headBytes(data string) string {
	if len(data) <= 16 {
		return data
	}
	var b strings.Builder
	for i := 0; i < 16; i++ {
		c := data[i]
		switch {
		case c == '\n':
			b.WriteString("\\n")
		case c == '\r':
			b.WriteString("\\r")
		case c == '\t':
			b.WriteString("\\t")
		case c >= 0x20 && c < 0x7f:
			b.WriteByte(c)
		default:
			b.WriteString(".")
		}
	}
	return b.String()
}

// TestSubprocessSchemaList binds the documented clean-binary smoke
// from a temporary directory. The subprocess must reproduce the
// frozen byte-exact table that the in-process tests assert.
func TestSubprocessSchemaList(t *testing.T) {
	bin := buildLeamasForTest(t)
	stdout, stderr, err := runLeamas(t, bin, "gate-summary", "schema", "list")
	if err != nil {
		t.Fatalf("schema list failed: %v\nstderr=%q", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("schema list must not write stderr; got %q", stderr)
	}
	want := "VERSION  STATUS     SCHEMA_ID\n" +
		"v1       supported  urn:leamas:gate-summary:v1\n" +
		"v2       current    urn:leamas:gate-summary:v2\n"
	if stdout != want {
		t.Fatalf("subprocess list output mismatch.\nWANT:\n%s\nGOT:\n%s", want, stdout)
	}
}

// TestSubprocessSchemaShowV1 binds the subprocess emission for v1.
func TestSubprocessSchemaShowV1(t *testing.T) {
	bin := buildLeamasForTest(t)
	stdout, stderr, err := runLeamas(t, bin, "gate-summary", "schema", "show", "v1")
	if err != nil {
		t.Fatalf("schema show v1 failed: %v\nstderr=%q", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("show v1 must not write stderr; got %q", stderr)
	}
	if !strings.HasPrefix(stdout, "{") {
		t.Fatalf("show v1 did not emit a JSON object; head=%q", headBytes(stdout))
	}
	if !strings.HasSuffix(stdout, "\n") {
		t.Fatalf("show v1 missing trailing LF")
	}
}

// TestSubprocessSchemaShowV2 binds the subprocess emission for v2.
func TestSubprocessSchemaShowV2(t *testing.T) {
	bin := buildLeamasForTest(t)
	stdout, stderr, err := runLeamas(t, bin, "gate-summary", "schema", "show", "v2")
	if err != nil {
		t.Fatalf("schema show v2 failed: %v\nstderr=%q", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("show v2 must not write stderr; got %q", stderr)
	}
	if !strings.HasPrefix(stdout, "{") {
		t.Fatalf("show v2 did not emit a JSON object; head=%q", headBytes(stdout))
	}
	if !strings.HasSuffix(stdout, "\n") {
		t.Fatalf("show v2 missing trailing LF")
	}
}

// TestSubprocessSchemaUnknownVersion asserts the subprocess
// rejects an unknown version with non-zero exit code and a
// diagnostic on stderr.
func TestSubprocessSchemaUnknownVersion(t *testing.T) {
	bin := buildLeamasForTest(t)
	stdout, stderr, err := runLeamas(t, bin, "gate-summary", "schema", "show", "v3")
	if err == nil {
		t.Fatalf("unknown version must exit non-zero")
	}
	if stdout != "" {
		t.Fatalf("unknown version must not write stdout; got %q", stdout)
	}
	if !strings.Contains(stderr, "v3") {
		t.Fatalf("unknown-version diagnostic must mention 'v3'; got %q", stderr)
	}
}

// TestSubprocessSchemaMutableAlias asserts the subprocess rejects
// the closed set of mutable aliases.
func TestSubprocessSchemaMutableAlias(t *testing.T) {
	bin := buildLeamasForTest(t)
	for _, alias := range []string{"latest", "current", "stable", "default"} {
		t.Run(alias, func(t *testing.T) {
			stdout, _, err := runLeamas(t, bin, "gate-summary", "schema", "show", alias)
			if err == nil {
				t.Fatalf("mutable alias %q must exit non-zero", alias)
			}
			if stdout != "" {
				t.Fatalf("mutable alias %q must not write stdout; got %q", alias, stdout)
			}
		})
	}
}

// TestSubprocessSchemaHelp asserts that the subprocess prints help
// text to stdout on a help flag from a temporary directory.
func TestSubprocessSchemaHelp(t *testing.T) {
	bin := buildLeamasForTest(t)
	stdout, stderr, err := runLeamas(t, bin, "gate-summary", "schema", "--help")
	if err != nil {
		t.Fatalf("schema --help failed: %v\nstderr=%q", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("help must not write stderr; got %q", stderr)
	}
	if !strings.Contains(stdout, "JSON Schema") {
		t.Fatalf("help must mention 'JSON Schema'")
	}
}

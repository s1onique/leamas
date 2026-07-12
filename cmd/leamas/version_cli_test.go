package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// TestVersionCLI_Output_FieldSchema asserts the line-oriented
// schema exposes both `version:` (the effective SemVer used by the
// compatibility oracle) and the immutable provenance fields
// `commit:` and `build_time:`. The optional `declared_version:`
// line is emitted only when the stamp was derived (declared value
// differs from effective), and that is exercised in the dedicated
// declared-version test below.
func TestVersionCLI_Output_FieldSchema(t *testing.T) {
	cmd := exec.Command("go", "run", "github.com/s1onique/leamas/cmd/leamas", "version")
	cmd.Env = withoutLeamasEnv()
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("leamas version failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	// Permitted line counts:
	//   3: version, commit, build_time                  (release, clean)
	//   4: version, declared_version, commit, build_time  (dev, clean)
	//   5: + dirty appended when vcs.modified reports true
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d: %q", len(lines), output)
	}
	if len(lines) > 5 {
		t.Fatalf("expected at most 5 lines, got %d: %q", len(lines), output)
	}

	required := []string{"version:", "commit:", "build_time:"}
	for _, prefix := range required {
		if !hasLineWithPrefix(lines, prefix) {
			t.Errorf("missing required line with prefix %q in %q", prefix, lines)
		}
	}
	// The first semantic line must be `version:`.
	if !strings.HasPrefix(lines[0], "version:") {
		t.Errorf("first line: expected prefix %q, got %q", "version:", lines[0])
	}
}

// TestVersionCLI_DeclaredVersionEmitted ensures that when the
// binary is built with the default VERSION=dev the CLI surfaces a
// dedicated `declared_version:` line so reviewers can see the
// stamp was auto-derived from a placeholder. Release builds that
// pre-stamp with a real SemVer must NOT emit `declared_version:`.
//
// When the LEAMAS_TEST_DECLARED_VERSION environment variable is
// set to "release" the test asserts the line is absent (release
// path). Otherwise it must be present and report a known
// placeholder.
func TestVersionCLI_DeclaredVersionEmitted(t *testing.T) {
	cmd := exec.Command("go", "run", "github.com/s1onique/leamas/cmd/leamas", "version")
	cmd.Env = withoutLeamasEnv()
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("leamas version failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	hasDeclared := false
	for _, line := range lines {
		if strings.HasPrefix(line, "declared_version:") {
			hasDeclared = true
		}
	}
	if os.Getenv("LEAMAS_TEST_DECLARED_VERSION") == "release" {
		if hasDeclared {
			t.Errorf("declared_version line was emitted on a release build; expected absent when declared==effective")
		}
		return
	}
	if !hasDeclared {
		t.Fatalf("declared_version line not present (development build); output: %q", output)
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "declared_version:") {
			value := strings.TrimPrefix(line, "declared_version: ")
			if value != "dev" && value != "" && value != "unknown" {
				t.Errorf("declared_version line reports %q; expected a known placeholder", value)
			}
		}
	}
}

func hasLineWithPrefix(lines []string, prefix string) bool {
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func TestVersionCLI_ExitCode(t *testing.T) {
	cmd := exec.Command("go", "run", "github.com/s1onique/leamas/cmd/leamas", "version")
	cmd.Env = withoutLeamasEnv()
	err := cmd.Run()
	if err != nil {
		t.Errorf("leamas version should exit 0, got error: %v", err)
	}
}

func TestVersionCLI_JSON(t *testing.T) {
	cmd := exec.Command("go", "run", "github.com/s1onique/leamas/cmd/leamas", "version", "--json")
	cmd.Env = withoutLeamasEnv()
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("leamas version --json failed: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(output, &data); err != nil {
		t.Fatalf("invalid JSON output: %v\nOutput: %s", err, output)
	}

	if data["version"] == "" {
		t.Error("JSON missing 'version' field")
	}
	if data["commit"] == "" {
		t.Error("JSON missing 'commit' field")
	}
	if data["build_time"] == "" {
		t.Error("JSON missing 'build_time' field")
	}
}

// TestVersionCLI_NoExtraOutput guards against accidental logging
// in the line-oriented output.
func TestVersionCLI_NoExtraOutput(t *testing.T) {
	cmd := exec.Command("go", "run", "github.com/s1onique/leamas/cmd/leamas", "version")
	cmd.Env = withoutLeamasEnv()
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("leamas version failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "leamas") && !strings.Contains(line, "version") {
			t.Errorf("unexpected output line: %q", line)
		}
	}
}

// TestVersionCLI_MalformedLinkerVersionRejected (R5.1 executable
// regression) builds a temporary binary with only a malformed
// Version linker variable injected, and confirms that:
//
//   - `leamas version` reports the malformed value verbatim (no
//     silent laundering into a derived stamp).
//   - the strict-SemVer regex (the same one used by
//     `make stamp-check`) rejects the value.
func TestVersionCLI_MalformedLinkerVersionRejected(t *testing.T) {
	binDir := t.TempDir()
	binPath := binDir + "/leamas-bad"
	build := exec.Command(
		"go", "build", "-trimpath", "-o", binPath,
		"-ldflags",
		"-X 'github.com/s1onique/leamas/internal/version.Version=banana'",
		"github.com/s1onique/leamas/cmd/leamas",
	)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	// The temporary binary inherits the parent process env unless
	// we sanitize it. `make gate` runs `leamas factory gate`, which
	// exports the LEAMAS_EXEC_* re-entry markers; without stripping
	// them the spawned binary exits 1 with only "exit status 1"
	// surfaced (the re-entry message is on stderr). Use
	// CombinedOutput() so the diagnostic is included on failure.
	runBad := exec.Command(binPath, "version")
	runBad.Env = withoutLeamasEnv()
	out, err := runBad.CombinedOutput()
	if err != nil {
		t.Fatalf("release binary exited non-zero: %v\n%s", err, out)
	}
	versionLine := ""
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasPrefix(line, "version:") {
			versionLine = line
			break
		}
	}
	if versionLine != "version: banana" {
		t.Errorf("malformed Version must be preserved verbatim; got %q (full output:\n%s)", versionLine, out)
	}
	// The same strict-SemVer regex used by `make stamp-check` must
	// reject "banana"; if it accepts, the production guard would
	// also accept. We use a portable POSIX-ERE equivalent.
	strictSemVer := regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(\-((0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*))*))?(\+([0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*))?$`)
	if strictSemVer.MatchString("banana") {
		t.Errorf("strict-SemVer regex accepted %q; the Makefile STAMP_REGEX would also accept it", "banana")
	}
}

// TestVersionCLI_ReleaseBinary (R2.5b) builds a real release
// binary with explicit `-ldflags` injecting a known SemVer, then
// runs it to confirm:
//
//   - declared_version line is absent on a release build
//   - version line reports the injected SemVer
//   - JSON wire form omits declared_version
//
// This is the missing acceptance that the LEAMAS_TEST_DECLARED_VERSION
// env var alone could not provide.
func TestVersionCLI_ReleaseBinary(t *testing.T) {
	releaseVer := "0.1.0"
	commit := "abcdef1234"
	buildTime := "2026-08-01T12:00:00Z"

	binDir := t.TempDir()
	binPath := binDir + "/leamas"

	build := exec.Command(
		"go", "build", "-trimpath", "-o", binPath,
		"-ldflags",
		"-X 'github.com/s1onique/leamas/internal/version.Version="+releaseVer+"'"+
			" -X 'github.com/s1onique/leamas/internal/version.DeclaredVersion="+releaseVer+"'"+
			" -X 'github.com/s1onique/leamas/internal/version.Commit="+commit+"'"+
			" -X 'github.com/s1onique/leamas/internal/version.BuildTime="+buildTime+"'",
		"github.com/s1onique/leamas/cmd/leamas",
	)
	build.Env = withoutLeamasEnv()
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("release build failed: %v\n%s", err, out)
	}

	// Line-oriented output. Accept either 3 or 4 lines: a clean
	// build emits version/commit/build_time; a dirty build also
	// appends `dirty:`. Both must omit `declared_version:`.
	// Sanitize env (see comment in TestVersionCLI_MalformedLinkerVersionRejected).
	runLine := exec.Command(binPath, "version")
	runLine.Env = withoutLeamasEnv()
	out, err := runLine.CombinedOutput()
	if err != nil {
		t.Fatalf("release binary exited non-zero: %v\n%s", err, out)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 3 && len(lines) != 4 {
		t.Errorf("release binary should emit 3 or 4 lines, got %d:\n%s", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "version: "+releaseVer) {
		t.Errorf("release binary version line = %q, want prefix %q", lines[0], "version: "+releaseVer)
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "declared_version:") {
			t.Errorf("release binary must NOT emit declared_version line; got %q", line)
		}
	}

	// JSON wire form: declared_version must be omitted because
	// it equals version. Sanitize env (see comment in
	// TestVersionCLI_MalformedLinkerVersionRejected).
	runJSON := exec.Command(binPath, "version", "--json")
	runJSON.Env = withoutLeamasEnv()
	jsonOut, err := runJSON.CombinedOutput()
	if err != nil {
		t.Fatalf("release binary --json failed: %v\n%s", err, jsonOut)
	}
	var data map[string]string
	if err := json.Unmarshal(jsonOut, &data); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, jsonOut)
	}
	if data["version"] != releaseVer {
		t.Errorf("JSON version = %q, want %q", data["version"], releaseVer)
	}
	if _, present := data["declared_version"]; present {
		t.Errorf("JSON declared_version must be omitted on release; got %v", data["declared_version"])
	}
	if data["commit"] != commit {
		t.Errorf("JSON commit = %q, want %q", data["commit"], commit)
	}
	if data["build_time"] != buildTime {
		t.Errorf("JSON build_time = %q, want %q", data["build_time"], buildTime)
	}
}

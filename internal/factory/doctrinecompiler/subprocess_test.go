package doctrinecompiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ldflagVersion injects a real compiler version into the leamas
// binary so the verifier compatibility check has something to
// evaluate against.
const ldflagVersion = "0.1.0"

// TestSubprocessMakeFactorizeAndGate proves that the generated
// `make factorize` and `make gate` targets work end-to-end. The test
// builds the leamas binary, places it on PATH, compiles a fresh
// target, then runs `make factorize` and `make gate` inside it.
//
// This is the headlined end-to-end proof: the generated Makefile
// must be self-sufficient when LEAMAS resolves to a working
// compiler binary.
func TestSubprocessMakeFactorizeAndGate(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make not on PATH")
	}
	_ = LoadCorePack
	target := t.TempDir()
	repoRoot, err := repoRoot()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}

	// Build a leamas binary in the target's temp dir so `make` can
	// resolve `leamas` on its PATH. Inject a real compiler version so
	// the verify compatibility check has a real value to evaluate.
	bindir := filepath.Join(target, "bin")
	if err := os.MkdirAll(bindir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	leamasBin := filepath.Join(bindir, "leamas")
	buildCmd := exec.Command("go", "build",
		"-ldflags", "-X github.com/s1onique/leamas/internal/version.Version="+ldflagVersion,
		"-trimpath", "-o", leamasBin, "./cmd/leamas")
	buildCmd.Dir = repoRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}

	// Step 1: compile via the CLI surface.
	cmd := exec.Command(leamasBin,
		"factory", "doctrine", "compile",
		"--profile", "fsharp-elm-service-v1",
		"--target", target,
	)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("doctrine compile failed: %v\n%s", err, out)
	}

	// Step 2: run `make factorize` with LEAMAS on PATH inside the
	// compiled target.
	envPath := os.Getenv("PATH")
	cmd = exec.Command("make", "factorize")
	cmd.Dir = target
	cmd.Env = append(os.Environ(), "PATH="+bindir+string(os.PathListSeparator)+envPath)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make factorize failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "factorize") {
		t.Errorf("unexpected factorize output: %s", out)
	}

	// Step 3: run `make gate` inside the compiled target.
	cmd = exec.Command("make", "gate")
	cmd.Dir = target
	cmd.Env = append(os.Environ(), "PATH="+bindir+string(os.PathListSeparator)+envPath)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make gate failed: %v\n%s", err, out)
	}
}

// repoRoot walks upward from the current test's working directory to
// locate the Leamas go.mod.
func repoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}

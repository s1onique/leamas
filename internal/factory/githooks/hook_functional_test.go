package githooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrePushFunctional_AllowsBranchCreation(t *testing.T) {
	tmpDir := t.TempDir()
	if err := runGitCommand(tmpDir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if err := runGitCommand(tmpDir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}
	if err := runGitCommand(tmpDir, "config", "user.name", "Test"); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runGitCommand(tmpDir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := runGitCommand(tmpDir, "commit", "-m", "initial"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tmpDir
	shaBytes, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	commitSHA := strings.TrimSpace(string(shaBytes))

	hookPath, cleanup := installTestHook(t, tmpDir)
	defer cleanup()

	input := fmt.Sprintf("refs/heads/main %s refs/heads/main %s\n", commitSHA, "0000000000000000000000000000000000000000")
	cmd = exec.Command("bash", hookPath)
	cmd.Dir = tmpDir
	cmd.Stdin = strings.NewReader(input)
	if _, err := cmd.CombinedOutput(); err != nil && err.Error() != "exit status 0" {
		t.Errorf("hook rejected new branch creation: %v", err)
	}
}

func TestPrePushFunctional_RejectsProtectedBranchDeletion(t *testing.T) {
	tmpDir := t.TempDir()
	if err := runGitCommand(tmpDir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if err := runGitCommand(tmpDir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}
	if err := runGitCommand(tmpDir, "config", "user.name", "Test"); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	hookPath, cleanup := installTestHook(t, tmpDir)
	defer cleanup()

	input := "refs/heads/main 0000000000000000000000000000000000000000 refs/heads/main abc123def0000000000000000000000000000000\n"
	cmd := exec.Command("bash", hookPath)
	cmd.Dir = tmpDir
	cmd.Stdin = strings.NewReader(input)
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Error("hook allowed protected branch deletion, expected rejection")
	}
	if !strings.Contains(string(output), "refusing to delete protected branch") {
		t.Errorf("expected deletion error message, got: %s", output)
	}
}

func TestPrePushFunctional_RejectsNonFastForward(t *testing.T) {
	tmpDir := t.TempDir()
	if err := runGitCommand(tmpDir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if err := runGitCommand(tmpDir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}
	if err := runGitCommand(tmpDir, "config", "user.name", "Test"); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	// Create initial commit
	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runGitCommand(tmpDir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := runGitCommand(tmpDir, "commit", "-m", "initial"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Get initial SHA
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tmpDir
	shaBytes, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	initialSHA := strings.TrimSpace(string(shaBytes))

	// Create divergent commit
	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runGitCommand(tmpDir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := runGitCommand(tmpDir, "commit", "-m", "divergent"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tmpDir
	shaBytes, err = cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	divergentSHA := strings.TrimSpace(string(shaBytes))

	hookPath, cleanup := installTestHook(t, tmpDir)
	defer cleanup()

	// Simulate non-fast-forward: remote has divergent commit, local is behind
	input := fmt.Sprintf("refs/heads/main %s refs/heads/main %s\n", initialSHA, divergentSHA)
	cmd = exec.Command("bash", hookPath)
	cmd.Dir = tmpDir
	cmd.Stdin = strings.NewReader(input)
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Error("hook allowed non-fast-forward push, expected rejection")
	}
	if !strings.Contains(string(output), "refusing non-fast-forward") {
		t.Errorf("expected non-fast-forward error message, got: %s", output)
	}
}

func TestPrePushFunctional_AllowsFastForward(t *testing.T) {
	tmpDir := t.TempDir()
	if err := runGitCommand(tmpDir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if err := runGitCommand(tmpDir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}
	if err := runGitCommand(tmpDir, "config", "user.name", "Test"); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	// Create initial commit
	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runGitCommand(tmpDir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := runGitCommand(tmpDir, "commit", "-m", "initial"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Get first commit SHA
	cmd := exec.Command("git", "rev-parse", "HEAD~0")
	cmd.Dir = tmpDir
	shaBytes, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	oldSHA := strings.TrimSpace(string(shaBytes))

	// Create another commit
	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runGitCommand(tmpDir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := runGitCommand(tmpDir, "commit", "-m", "additional"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tmpDir
	shaBytes, err = cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	newSHA := strings.TrimSpace(string(shaBytes))

	hookPath, cleanup := installTestHook(t, tmpDir)
	defer cleanup()

	// Simulate fast-forward push
	input := fmt.Sprintf("refs/heads/main %s refs/heads/main %s\n", newSHA, oldSHA)
	cmd = exec.Command("bash", hookPath)
	cmd.Dir = tmpDir
	cmd.Stdin = strings.NewReader(input)
	if _, err := cmd.CombinedOutput(); err != nil && err.Error() != "exit status 0" {
		t.Errorf("hook rejected fast-forward push: %v", err)
	}
}

func TestPrePushFunctional_IgnoresUnprotectedBranch(t *testing.T) {
	tmpDir := t.TempDir()
	if err := runGitCommand(tmpDir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if err := runGitCommand(tmpDir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}
	if err := runGitCommand(tmpDir, "config", "user.name", "Test"); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	// Create initial commit
	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runGitCommand(tmpDir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := runGitCommand(tmpDir, "commit", "-m", "initial"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tmpDir
	shaBytes, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	oldSHA := strings.TrimSpace(string(shaBytes))

	// Create divergent commit
	if err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runGitCommand(tmpDir, "add", "."); err != nil {
		t.Fatal(err)
	}
	if err := runGitCommand(tmpDir, "commit", "-m", "divergent"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = tmpDir
	shaBytes, err = cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	newSHA := strings.TrimSpace(string(shaBytes))

	hookPath, cleanup := installTestHook(t, tmpDir)
	defer cleanup()

	// Simulate force-push to unprotected branch (feature-x)
	input := fmt.Sprintf("refs/heads/feature-x %s refs/heads/feature-x %s\n", newSHA, oldSHA)
	cmd = exec.Command("bash", hookPath)
	cmd.Dir = tmpDir
	cmd.Stdin = strings.NewReader(input)
	if _, err := cmd.CombinedOutput(); err != nil && err.Error() != "exit status 0" {
		t.Errorf("hook rejected force-push to unprotected branch: %v", err)
	}
}

package githooks

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runGitCommand runs a git command and returns an error if it fails.
func runGitCommand(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Run()
}

// validHookContent is the standard pre-push hook used in tests.
const validHookContent = `#!/usr/bin/env bash
set -euo pipefail

protected_ref() {
  case "$1" in
    refs/heads/main|refs/heads/master|refs/heads/release/*) return 0 ;;
    *) return 1 ;;
  esac
}

zero=0000000000000000000000000000000000000000
failed=0

while read -r local_ref local_sha remote_ref remote_sha; do
  protected_ref "$remote_ref" || continue

  if [[ "$local_sha" == "$zero" ]]; then
    echo "ERROR: refusing to delete protected branch: $remote_ref" >&2
    failed=1
    continue
  fi

  if [[ "$remote_sha" == "$zero" ]]; then
    continue
  fi

  if git merge-base --is-ancestor "$remote_sha" "$local_sha"; then
    continue
  fi

  echo "ERROR: refusing non-fast-forward push to protected branch: $remote_ref" >&2
  echo "Use pull/rebase/merge and create a forward corrective commit." >&2
  echo "Force-push is disabled for Leamas Factory work." >&2
  failed=1
done

exit "$failed"
`

// installTestHook creates a temporary git repo with the standard hook installed.
func installTestHook(t *testing.T, tmpDir string) (string, func()) {
	if err := os.MkdirAll(filepath.Join(tmpDir, "githooks"), 0755); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(tmpDir, "githooks", "pre-push")
	if err := os.WriteFile(hookPath, []byte(validHookContent), 0755); err != nil {
		t.Fatal(err)
	}
	return hookPath, func() {
		os.RemoveAll(filepath.Join(tmpDir, "githooks"))
	}
}

func TestCheckRepo(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(tmpDir, "githooks"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "githooks", "pre-push"), []byte(validHookContent), 0755); err != nil {
		t.Fatal(err)
	}

	installerContent := `#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

chmod +x githooks/pre-push
git config --local core.hooksPath githooks

echo "Installed Leamas Git hooks:"
echo "  core.hooksPath=$(git config --get core.hooksPath)"
`
	if err := os.MkdirAll(filepath.Join(tmpDir, "scripts"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "scripts", "install_git_hooks.sh"), []byte(installerContent), 0755); err != nil {
		t.Fatal(err)
	}

	if err := runGitCommand(tmpDir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if err := runGitCommand(tmpDir, "config", "core.hooksPath", "githooks"); err != nil {
		t.Fatalf("git config failed: %v", err)
	}

	findings, err := CheckRepo(tmpDir)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d: %+v", len(findings), findings)
	}
}

func TestCheckRepo_MissingHook(t *testing.T) {
	tmpDir := t.TempDir()
	findings, err := CheckRepo(tmpDir)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.Kind == "missing" && f.Path == filepath.Join(tmpDir, "githooks", "pre-push") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finding for missing githooks/pre-push")
	}
}

func TestCheckRepo_NonExecutableHook(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "githooks"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "githooks", "pre-push"), []byte("#!/bin/bash\necho hi"), 0644); err != nil {
		t.Fatal(err)
	}

	findings, err := CheckRepo(tmpDir)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.Kind == "not_executable" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finding for non-executable hook")
	}
}

func TestCheckRepo_MissingInstaller(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "githooks"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "githooks", "pre-push"), []byte("#!/bin/bash\nexit 0"), 0755); err != nil {
		t.Fatal(err)
	}

	findings, err := CheckRepo(tmpDir)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.Kind == "missing" && f.Path == filepath.Join(tmpDir, "scripts", "install_git_hooks.sh") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finding for missing installer")
	}
}

func TestCheckRepo_HookMissingProtectedBranchRef(t *testing.T) {
	tmpDir := t.TempDir()
	hookContent := `#!/usr/bin/env bash
set -euo pipefail
echo "test"
`
	if err := os.MkdirAll(filepath.Join(tmpDir, "githooks"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "githooks", "pre-push"), []byte(hookContent), 0755); err != nil {
		t.Fatal(err)
	}

	findings, err := CheckRepo(tmpDir)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.Kind == "missing_content" && f.Message == "missing required content: protected branch main" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finding for missing protected branch refs")
	}
}

func TestCheckRepo_HookMissingMergeBaseCheck(t *testing.T) {
	tmpDir := t.TempDir()
	hookContent := `#!/usr/bin/env bash
set -euo pipefail
if [[ "$1" == "refs/heads/main" ]]; then
  echo "ERROR" >&2
  exit 1
fi
`
	if err := os.MkdirAll(filepath.Join(tmpDir, "githooks"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "githooks", "pre-push"), []byte(hookContent), 0755); err != nil {
		t.Fatal(err)
	}

	findings, err := CheckRepo(tmpDir)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.Kind == "missing_content" && f.Message == "missing required content: non-fast-forward detection" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finding for missing merge-base check")
	}
}

func TestCheckRepo_BashVerifierForbidden(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "scripts"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "scripts", "verify_git_hooks.sh"), []byte("#!/bin/bash\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	findings, err := CheckRepo(tmpDir)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.Kind == "forbidden" && f.Path == filepath.Join(tmpDir, "scripts", "verify_git_hooks.sh") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finding for forbidden Bash verifier")
	}
}

func TestCheckRepo_HookTooLong(t *testing.T) {
	tmpDir := t.TempDir()
	lines := []string{"#!/usr/bin/env bash", "set -euo pipefail"}
	for i := 0; i < 55; i++ {
		lines = append(lines, "echo 'line content'")
	}
	hookContent := ""
	for _, l := range lines {
		hookContent += l + "\n"
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "githooks"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "githooks", "pre-push"), []byte(hookContent), 0755); err != nil {
		t.Fatal(err)
	}

	findings, err := CheckRepo(tmpDir)
	if err != nil {
		t.Fatalf("CheckRepo failed: %v", err)
	}
	found := false
	for _, f := range findings {
		if f.Kind == "too_long" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finding for hook exceeding LOC limit")
	}
}

func TestCountMeaningfulLOC(t *testing.T) {
	tests := []struct {
		content string
		expect  int
	}{
		{"#!/bin/bash\n# comment\n\necho hi", 1},
		{"#!/bin/bash\n\n\necho hi", 1},
		{"line1\nline2\nline3", 3},
		{"", 0},
		{"# only comments\n# and more", 0},
		{"set -euo pipefail\necho test", 2},
	}
	for _, tt := range tests {
		got := countMeaningfulLOC(tt.content)
		if got != tt.expect {
			t.Errorf("countMeaningfulLOC(%q) = %d, want %d", tt.content, got, tt.expect)
		}
	}
}

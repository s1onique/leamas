// SPDX-License-Identifier: Apache-2.0

package reporoot_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/factory/forbidden/reporoot"
)

func TestRootResolver_RepoRoot(t *testing.T) {
	resolver := reporoot.New()

	cwd, err := os.Getwd()
	if err != nil {
		t.Skip("cannot get cwd")
	}

	root, err := resolver.Resolve(cwd)
	if err != nil {
		t.Fatalf("failed to resolve root: %v", err)
	}

	gitDir := filepath.Join(root, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Error("resolved root should contain .git directory")
	}
}

func TestRootResolver_Subdirectory(t *testing.T) {
	resolver := reporoot.New()

	cwd, err := os.Getwd()
	if err != nil {
		t.Skip("cannot get cwd")
	}

	subdir := filepath.Join(cwd, "internal")
	if _, err := os.Stat(subdir); os.IsNotExist(err) {
		t.Skip("internal directory does not exist")
	}

	root, err := resolver.Resolve(subdir)
	if err != nil {
		t.Fatalf("failed to resolve root from subdir: %v", err)
	}

	cwdRoot, err := resolver.Resolve(cwd)
	if err != nil {
		t.Fatalf("failed to resolve cwd root: %v", err)
	}

	if root != cwdRoot {
		t.Errorf("subdirectory root != cwd root: %s vs %s", root, cwdRoot)
	}
}

func TestRootResolver_Nonexistent(t *testing.T) {
	resolver := reporoot.New()

	_, err := resolver.Resolve("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestRootResolver_EmptyInput(t *testing.T) {
	resolver := reporoot.New()

	_, err := resolver.Resolve("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestRootResolver_IsWithinRepo(t *testing.T) {
	resolver := reporoot.New()

	cwd, err := os.Getwd()
	if err != nil {
		t.Skip("cannot get cwd")
	}

	tests := []struct {
		path     string
		repo     string
		expected bool
	}{
		{cwd, cwd, true},
		{filepath.Join(cwd, "internal"), cwd, true},
		{filepath.Join(cwd, "cmd"), cwd, true},
		{"/tmp", cwd, false},
	}

	for _, tc := range tests {
		result := resolver.IsWithinRepo(tc.path, tc.repo)
		if result != tc.expected {
			t.Errorf("IsWithinRepo(%s, %s) = %v, want %v", tc.path, tc.repo, result, tc.expected)
		}
	}
}

func TestRootResolver_SplitRepoPath(t *testing.T) {
	resolver := reporoot.New()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Skip("cannot create .git dir")
	}
	if err := os.MkdirAll(filepath.Join(dir, "internal", "factory"), 0755); err != nil {
		t.Skip("cannot create internal/factory")
	}

	root, rel, err := resolver.SplitRepoPath(dir)
	if err != nil {
		t.Fatalf("failed to split repo root path: %v", err)
	}

	if rel != "." {
		t.Errorf("relative path for root = %q, want \".\"", rel)
	}

	subdir := filepath.Join(dir, "internal", "factory")
	root2, rel2, err := resolver.SplitRepoPath(subdir)
	if err != nil {
		t.Fatalf("failed to split subdir path: %v", err)
	}

	if root2 != root {
		t.Errorf("root from subdir != root from repo root: %s vs %s", root2, root)
	}

	expected := filepath.Join("internal", "factory")
	if rel2 != expected {
		t.Errorf("relative path = %q, want %q", rel2, expected)
	}
}

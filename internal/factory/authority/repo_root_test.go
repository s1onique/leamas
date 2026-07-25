// SPDX-License-Identifier: Apache-2.0

// Package authority: repo_root_test.go pins the bounded repository
// root resolver contract used by the executable-authority checks
// and by future consumers that must anchor a path to the Leamas
// source tree without assuming a particular package test depth.
package authority

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// materializeRoot creates a temporary directory that satisfies the
// full RepositorySentinels list and returns it.
func materializeRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, sentinel := range RepositorySentinels {
		full := filepath.Join(dir, sentinel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir parent for %s: %v", sentinel, err)
		}
		if err := os.WriteFile(full, []byte("sentinel\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", sentinel, err)
		}
	}
	return dir
}

func TestFindRepositoryRootStartsAtRoot(t *testing.T) {
	root := materializeRoot(t)
	got, err := FindRepositoryRoot(root)
	if err != nil {
		t.Fatalf("FindRepositoryRoot: %v", err)
	}
	if got != root {
		t.Fatalf("got %q want %q", got, root)
	}
}

func TestFindRepositoryRootStartsInSubdirectory(t *testing.T) {
	root := materializeRoot(t)
	sub := filepath.Join(root, "internal", "factory", "authority")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	got, err := FindRepositoryRoot(sub)
	if err != nil {
		t.Fatalf("FindRepositoryRoot: %v", err)
	}
	if got != root {
		t.Fatalf("got %q want %q", got, root)
	}
}

func TestFindRepositoryRootStartsFromNestedTempDir(t *testing.T) {
	root := materializeRoot(t)
	nested := t.TempDir()
	if err := os.Symlink(root, filepath.Join(nested, "link-to-root")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	got, err := FindRepositoryRoot(filepath.Join(nested, "link-to-root", "internal"))
	if err != nil {
		t.Fatalf("FindRepositoryRoot: %v", err)
	}
	resolved, err := filepath.EvalSymlinks(got)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	want, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks(root): %v", err)
	}
	if resolved != want {
		t.Fatalf("resolved=%q want=%q (raw=%q)", resolved, want, got)
	}
}

func TestFindRepositoryRootNoRepository(t *testing.T) {
	dir := t.TempDir()
	if dir == "/" {
		t.Skip("tempdir is filesystem root; cannot exercise no-root case")
	}
	_, err := FindRepositoryRoot(dir)
	if err == nil {
		t.Fatalf("expected error locating repository above %s", dir)
	}
	if !errors.Is(err, ErrNoRepositoryRoot) {
		t.Fatalf("got error %v want ErrNoRepositoryRoot", err)
	}
}

func TestFindRepositoryRootPartialSentinelFailsClosed(t *testing.T) {
	tempRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempRoot, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	_, err := FindRepositoryRoot(tempRoot)
	if !errors.Is(err, ErrNoRepositoryRoot) {
		t.Fatalf("partial sentinel set must fail closed: err=%v", err)
	}
}

func TestFindRepositoryRootAcceptsFileAsStart(t *testing.T) {
	root := materializeRoot(t)
	target := filepath.Join(root, "go.mod")
	got, err := FindRepositoryRoot(target)
	if err != nil {
		t.Fatalf("FindRepositoryRoot(file): %v", err)
	}
	if got != root {
		t.Fatalf("got %q want %q", got, root)
	}
}

func TestFindRepositoryRootResolvesRealRepo(t *testing.T) {
	pkgDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	got, err := FindRepositoryRoot(pkgDir)
	if err != nil {
		t.Fatalf("FindRepositoryRoot: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute path, got %q", got)
	}
	if filepath.Base(got) != "" && filepath.Base(got) == got {
		t.Fatalf("returned filesystem root: %q", got)
	}
	if _, err := os.Stat(filepath.Join(got, ".factory")); err != nil {
		t.Fatalf(".factory sentinel missing in resolved root: %v", err)
	}
	if runtime.GOOS != "windows" {
		if _, err := os.Stat(filepath.Join(got, "githooks", "pre-push")); err != nil {
			t.Fatalf("githooks/pre-push sentinel missing: %v", err)
		}
	}
}

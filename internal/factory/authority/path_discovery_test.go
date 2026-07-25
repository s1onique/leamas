// SPDX-License-Identifier: Apache-2.0

// Package authority: path_discovery_test.go pins the bounded PATH
// scanning contract used by the doctor surface. Each test drives
// the explicit-input seam discoverPATHExecutablesFrom and never
// mutates process-global PATH.
package authority

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeExecutable is a test helper that creates an executable
// file with explicit mode bits so the discovery seam's exec-bit
// predicate has a deterministic signal to evaluate.
func writeExecutable(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
}

func TestDiscoverPATHExecutablesFromMultiple(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
	candA := filepath.Join(dirA, "leamas")
	candB := filepath.Join(dirB, "leamas")
	writeExecutable(t, candA, 0o755)
	writeExecutable(t, candB, 0o755)

	pathValue := strings.Join([]string{dirA, dirB}, string(os.PathListSeparator))
	got := discoverPATHExecutablesFrom("leamas", pathValue, os.Stat)
	want := []string{candA, candB}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d]=%q want %q", i, got[i], want[i])
		}
	}
}

func TestDiscoverPATHExecutablesFromPreservesOrder(t *testing.T) {
	d1 := t.TempDir()
	d2 := t.TempDir()
	c1 := filepath.Join(d1, "leamas")
	c2 := filepath.Join(d2, "leamas")
	writeExecutable(t, c1, 0o755)
	writeExecutable(t, c2, 0o755)
	pathValue := strings.Join([]string{d1, d2, d1}, string(os.PathListSeparator))
	got := discoverPATHExecutablesFrom("leamas", pathValue, os.Stat)
	if len(got) != 2 {
		t.Fatalf("got %v want exactly two entries", got)
	}
	if got[0] != c1 || got[1] != c2 {
		t.Fatalf("order broken: got %v want [%s %s]", got, c1, c2)
	}
}

func TestDiscoverPATHExecutablesFromDeduplicates(t *testing.T) {
	d1 := t.TempDir()
	c := filepath.Join(d1, "leamas")
	writeExecutable(t, c, 0o755)
	pathValue := strings.Join([]string{d1, d1, d1}, string(os.PathListSeparator))
	got := discoverPATHExecutablesFrom("leamas", pathValue, os.Stat)
	if len(got) != 1 {
		t.Fatalf("got %v want exactly one entry", got)
	}
	if got[0] != c {
		t.Fatalf("got %v want %q", got, c)
	}
}

func TestDiscoverPATHExecutablesFromRejectsNonExecutable(t *testing.T) {
	d := t.TempDir()
	c := filepath.Join(d, "leamas")
	if err := os.WriteFile(c, []byte("not executable"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Chmod(c, 0o644); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	got := discoverPATHExecutablesFrom("leamas", d, os.Stat)
	if len(got) != 0 {
		t.Fatalf("non-executable entries leaked: %v", got)
	}
}

func TestDiscoverPATHExecutablesFromIgnoresMissingCandidate(t *testing.T) {
	d1 := t.TempDir()
	d2 := t.TempDir()
	c := filepath.Join(d1, "leamas")
	writeExecutable(t, c, 0o755)
	pathValue := strings.Join([]string{d1, d2}, string(os.PathListSeparator))
	got := discoverPATHExecutablesFrom("leamas", pathValue, os.Stat)
	if len(got) != 1 || got[0] != c {
		t.Fatalf("got %v want [%q]", got, c)
	}
}

func TestDiscoverPATHExecutablesFromEmptyPath(t *testing.T) {
	if got := discoverPATHExecutablesFrom("leamas", "", os.Stat); got != nil {
		t.Fatalf("expected nil on empty PATH, got %v", got)
	}
	pathValue := string(os.PathListSeparator) + string(os.PathListSeparator)
	if got := discoverPATHExecutablesFrom("leamas", pathValue, os.Stat); got != nil {
		t.Fatalf("expected nil when only separators supplied, got %v", got)
	}
}

func TestDiscoverPATHExecutablesFromSkipsDirectories(t *testing.T) {
	d := t.TempDir()
	if err := os.Mkdir(filepath.Join(d, "leamas"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	got := discoverPATHExecutablesFrom("leamas", d, os.Stat)
	if len(got) != 0 {
		t.Fatalf("directory leaked as executable candidate: %v", got)
	}
}

func TestDiscoverPATHExecutablesFromRespectsPlatformMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skipf("Unix executable bits are not the host executable semantic on %s", runtime.GOOS)
	}
	d := t.TempDir()
	c := filepath.Join(d, "leamas")
	writeExecutable(t, c, 0o755)
	info, err := os.Stat(c)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("host does not preserve executable bit: mode=%v", info.Mode())
	}
	got := discoverPATHExecutablesFrom("leamas", d, os.Stat)
	if len(got) != 1 || got[0] != c {
		t.Fatalf("expected executable on %s; got %v", runtime.GOOS, got)
	}
}

func TestDiscoverPATHExecutablesIsThinWrapper(t *testing.T) {
	dir := t.TempDir()
	want := filepath.Join(dir, "leamas")
	writeExecutable(t, want, 0o755)
	prev, hadPrev := os.LookupEnv("PATH")
	t.Cleanup(func() {
		if hadPrev {
			_ = os.Setenv("PATH", prev)
		} else {
			_ = os.Unsetenv("PATH")
		}
	})
	if err := os.Setenv("PATH", dir); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	got := discoverPATHExecutables("leamas")
	if len(got) != 1 || got[0] != want {
		t.Fatalf("wrapper mismatch: got %v want [%q]", got, want)
	}
}

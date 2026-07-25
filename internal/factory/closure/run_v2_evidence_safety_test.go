// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestWriteExclusiveRegularRejectsExistingPaths(t *testing.T) {
	cases := []struct {
		name string
		make func(*testing.T, string)
	}{
		{"regular", func(t *testing.T, path string) { mustWriteFile(t, path, []byte("old")) }},
		{"symlink", func(t *testing.T, path string) { mustSymlink(t, "target", path) }},
		{"directory", func(t *testing.T, path string) {
			if err := os.Mkdir(path, 0o700); err != nil {
				t.Fatal(err)
			}
		}},
		{"fifo", func(t *testing.T, path string) {
			if err := syscall.Mkfifo(path, 0o600); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "reserved")
			tc.make(t, path)
			if err := writeExclusiveRegular(path, []byte("new"), 0o600); err == nil {
				t.Fatal("existing path was accepted")
			}
		})
	}
}

func TestReservedEvidenceSymlinksDoNotTouchOutsideTargets(t *testing.T) {
	for _, name := range []string{v2EvidenceIndexName, "runtime.json"} {
		t.Run(name, func(t *testing.T) {
			outside := filepath.Join(t.TempDir(), "outside")
			original := []byte("outside bytes\n")
			mustWriteFile(t, outside, original)
			staging := makeEvidenceStaging(t, map[string]string{"check.stdout": "ok\n"})
			mustSymlink(t, outside, filepath.Join(staging, name))
			var err error
			if name == v2EvidenceIndexName {
				_, err = buildV2EvidenceIndex(staging)
			} else {
				input := v2FinalizeInput{EvidenceDirectory: staging, Plan: Plan{ActID: objectTransactionActID}}
				err = writeV2RuntimeEvidence(input)
			}
			if err == nil {
				t.Fatal("reserved symlink was accepted")
			}
			got, readErr := os.ReadFile(outside)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Equal(got, original) {
				t.Fatalf("outside target changed: %q", got)
			}
		})
	}
}

func TestRuntimeEvidenceRejectsReservedCheckEvidenceName(t *testing.T) {
	for _, name := range []string{"runtime.json", "patch-policy.log", "closure-policy.log", v2EvidenceIndexName} {
		t.Run(name, func(t *testing.T) {
			input := v2FinalizeInput{
				EvidenceDirectory: makeEvidenceStaging(t, nil),
				Plan:              Plan{ActID: objectTransactionActID},
				Branch:            "main",
				CheckEvidence:     []EvidenceRecord{{LogicalName: name}},
			}
			if err := writeV2RuntimeEvidence(input); err == nil || !strings.Contains(err.Error(), "reserved") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestV2EvidenceIndexRejectsTrailingJSON(t *testing.T) {
	root := makeEvidenceStaging(t, nil)
	index := []byte("{\"contract_version\":1,\"entries\":[]} {}")
	if err := verifyV2EvidenceIndex(root, index); err == nil || !strings.Contains(err.Error(), "trailing") {
		t.Fatalf("error = %v", err)
	}
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func mustSymlink(t *testing.T, target, path string) {
	t.Helper()
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
}

// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestV2EvidenceIndexStableAndSorted(t *testing.T) {
	first := makeEvidenceStaging(t, map[string]string{
		"z/out.log": "z\n", "a/runtime.json": "{}\n", "m.txt": "m\n",
	})
	second := makeEvidenceStaging(t, map[string]string{
		"m.txt": "m\n", "a/runtime.json": "{}\n", "z/out.log": "z\n",
	})
	firstEntries, err := scanV2Evidence(first)
	if err != nil {
		t.Fatal(err)
	}
	secondEntries, err := scanV2Evidence(second)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstEntries, secondEntries) {
		t.Fatalf("filesystem creation order affected index: first=%+v second=%+v", firstEntries, secondEntries)
	}
	for i := 1; i < len(firstEntries); i++ {
		if firstEntries[i-1].RelativePath >= firstEntries[i].RelativePath {
			t.Fatal("evidence entries are not strictly sorted")
		}
	}
	firstBytes, _ := marshalCanonicalJSON(v2EvidenceIndex{ContractVersion: 1, Entries: firstEntries})
	secondBytes, _ := marshalCanonicalJSON(v2EvidenceIndex{ContractVersion: 1, Entries: secondEntries})
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Fatal("canonical evidence index bytes differ")
	}
}

func TestV2EvidenceQualificationVerifiesAllBytes(t *testing.T) {
	parent := t.TempDir()
	staging := makeEvidenceStagingUnder(t, parent, map[string]string{"runtime.json": "{}\n", "check.stdout": "hello\n"})
	final := filepath.Join(parent, strings.Repeat("a", 40))
	qualified, err := buildV2EvidenceIndex(staging)
	if err != nil {
		t.Fatal(err)
	}
	qualified, err = publishV2Evidence(staging, final, qualified)
	if err != nil {
		t.Fatal(err)
	}
	if qualified.IndexSHA256 != SHA256Hex(qualified.IndexBytes) || qualified.FinalPath != final {
		t.Fatalf("qualified evidence = %+v", qualified)
	}
	if _, err := os.Lstat(staging); !os.IsNotExist(err) {
		t.Fatalf("staging remains after publication: %v", err)
	}
	if err := verifyV2EvidenceIndex(final, qualified.IndexBytes); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(final, "check.stdout"), []byte("tampered\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyV2EvidenceIndex(final, qualified.IndexBytes); err == nil {
		t.Fatal("tampered evidence bytes were accepted")
	}
	if err := os.WriteFile(filepath.Join(final, "extra.log"), []byte("extra\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyV2EvidenceIndex(final, qualified.IndexBytes); err == nil {
		t.Fatal("additional unindexed evidence was accepted")
	}
}

func TestV2EvidenceQualificationRejectsUnsafeEntries(t *testing.T) {
	t.Run("symlink", func(t *testing.T) {
		staging := makeEvidenceStaging(t, map[string]string{"target": "x"})
		if err := os.Symlink("target", filepath.Join(staging, "link")); err != nil {
			t.Fatal(err)
		}
		if _, err := scanV2Evidence(staging); err == nil || !strings.Contains(err.Error(), "regular file") {
			t.Fatalf("symlink error = %v", err)
		}
	})
	t.Run("special file", func(t *testing.T) {
		staging := makeEvidenceStaging(t, nil)
		if err := os.Mkdir(filepath.Join(staging, "nested"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("nested", filepath.Join(staging, "special")); err != nil {
			t.Fatal(err)
		}
		if _, err := scanV2Evidence(staging); err == nil {
			t.Fatal("special evidence entry accepted")
		}
	})
	for _, path := range []string{"/absolute", "../escape", "a/../b", ""} {
		if err := validateEvidenceRelativePath(path); err == nil {
			t.Fatalf("unsafe relative path %q accepted", path)
		}
	}
}

func TestV2EvidenceQualificationRejectsWrongOrExistingFinalPath(t *testing.T) {
	parent := t.TempDir()
	staging := makeEvidenceStagingUnder(t, parent, map[string]string{"runtime.json": "{}"})
	final := filepath.Join(parent, "final")
	if err := os.Mkdir(final, 0o700); err != nil {
		t.Fatal(err)
	}
	evidence, err := buildV2EvidenceIndex(staging)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := publishV2Evidence(staging, final, evidence); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("pre-existing final path error = %v", err)
	}
	if _, err := publishV2Evidence(filepath.Join(parent, "not-staging"), filepath.Join(parent, "other"), evidence); err == nil {
		t.Fatal("non-staging path accepted")
	}
}

func TestV2EvidenceIndexRejectsMissingFile(t *testing.T) {
	root := makeEvidenceStaging(t, map[string]string{"a.txt": "a", "b.txt": "b"})
	entries, err := scanV2Evidence(root)
	if err != nil {
		t.Fatal(err)
	}
	indexBytes, _ := marshalCanonicalJSON(v2EvidenceIndex{ContractVersion: 1, Entries: entries})
	if err := os.Remove(filepath.Join(root, "b.txt")); err != nil {
		t.Fatal(err)
	}
	if err := verifyV2EvidenceIndex(root, indexBytes); err == nil {
		t.Fatal("missing indexed evidence was accepted")
	}
}

func makeEvidenceStaging(t *testing.T, files map[string]string) string {
	t.Helper()
	return makeEvidenceStagingUnder(t, t.TempDir(), files)
}

func makeEvidenceStagingUnder(t *testing.T, parent string, files map[string]string) string {
	t.Helper()
	if err := os.MkdirAll(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	staging, err := os.MkdirTemp(parent, ".staging-test-")
	if err != nil {
		t.Fatal(err)
	}
	for relative, content := range files {
		path := filepath.Join(staging, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return staging
}

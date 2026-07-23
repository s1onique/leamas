package schema

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestRegistryListReturnsExpectedVersions asserts the closed version
// set and the descriptor ordering contract. The list is sorted by
// Version lexicographically and is returned as a fresh slice.
func TestRegistryListReturnsExpectedVersions(t *testing.T) {
	descriptors := List()
	if len(descriptors) != 2 {
		t.Fatalf("List() returned %d descriptors, want 2", len(descriptors))
	}
	if descriptors[0].Version != VersionV1 {
		t.Errorf("descriptors[0].Version = %q, want %q", descriptors[0].Version, VersionV1)
	}
	if descriptors[1].Version != VersionV2 {
		t.Errorf("descriptors[1].Version = %q, want %q", descriptors[1].Version, VersionV2)
	}
	if descriptors[0].Status != StatusSupported {
		t.Errorf("descriptors[0].Status = %q, want %q", descriptors[0].Status, StatusSupported)
	}
	if descriptors[1].Status != StatusCurrent {
		t.Errorf("descriptors[1].Status = %q, want %q", descriptors[1].Status, StatusCurrent)
	}
	if descriptors[0].SchemaID != SchemaIDV1 {
		t.Errorf("descriptors[0].SchemaID = %q, want %q", descriptors[0].SchemaID, SchemaIDV1)
	}
	if descriptors[1].SchemaID != SchemaIDV2 {
		t.Errorf("descriptors[1].SchemaID = %q, want %q", descriptors[1].SchemaID, SchemaIDV2)
	}
}

// TestRegistryListReturnsFreshSlice asserts that the slice returned by
// List is owned by the caller. Mutating it must not affect subsequent
// calls.
func TestRegistryListReturnsFreshSlice(t *testing.T) {
	first := List()
	first[0].Version = "MUTATED"
	second := List()
	if second[0].Version == "MUTATED" {
		t.Fatalf("List() returned a shared slice; mutating one call leaked into the next")
	}
}

// TestRegistryBytesReturnsClone asserts that Bytes returns a fresh
// slice for each invocation. Mutating the returned slice must not
// affect the embedded authority.
func TestRegistryBytesReturnsClone(t *testing.T) {
	original, err := Bytes(VersionV1)
	if err != nil {
		t.Fatalf("Bytes(v1): %v", err)
	}
	for i := range original {
		original[i] = 'X'
	}
	second, err := Bytes(VersionV1)
	if err != nil {
		t.Fatalf("Bytes(v1) second call: %v", err)
	}
	for _, b := range second {
		if b == 'X' {
			t.Fatalf("Bytes() returned a shared slice; mutation leaked into the embedded authority")
		}
	}
}

// TestRegistryBytesMatchesCanonicalFile asserts that the embedded
// bytes equal the canonical checked-in file bytes for both versions.
func TestRegistryBytesMatchesCanonicalFile(t *testing.T) {
	cases := []struct {
		version Version
		file    string
		wantID  string
	}{
		{VersionV1, "gate-summary-v1.schema.json", SchemaIDV1},
		{VersionV2, "gate-summary-v2.schema.json", SchemaIDV2},
	}
	for _, tc := range cases {
		t.Run(string(tc.version), func(t *testing.T) {
			embedded, err := Bytes(tc.version)
			if err != nil {
				t.Fatalf("Bytes(%s): %v", tc.version, err)
			}
			fsBytes, err := os.ReadFile(tc.file)
			if err != nil {
				t.Fatalf("ReadFile(%s): %v", tc.file, err)
			}
			if !bytes.Equal(embedded, fsBytes) {
				t.Fatalf("Bytes(%s) does not match checked-in file %s", tc.version, tc.file)
			}
			// Sanity check: the v1 bytes must contain the v1 $id.
			if tc.version == VersionV1 {
				if !bytes.Contains(embedded, []byte(tc.wantID)) {
					t.Errorf("v1 bytes missing $id %q", tc.wantID)
				}
			}
		})
	}
}

// TestRegistryBytesRejectsUnknownVersion asserts that requesting an
// unknown version returns a typed *UnknownVersionError.
func TestRegistryBytesRejectsUnknownVersion(t *testing.T) {
	bogus := Version("v3")
	_, err := Bytes(bogus)
	if err == nil {
		t.Fatalf("Bytes(%q) must reject unknown version", bogus)
	}
	if !IsUnknownVersion(err) {
		t.Fatalf("Bytes(%q) error = %v, want *UnknownVersionError", bogus, err)
	}
}

// TestRegistryWriteExactBytes asserts that Write copies the exact
// embedded bytes to the destination writer.
func TestRegistryWriteExactBytes(t *testing.T) {
	cases := []Version{VersionV1, VersionV2}
	for _, v := range cases {
		t.Run(string(v), func(t *testing.T) {
			var buf bytes.Buffer
			if err := Write(v, &buf); err != nil {
				t.Fatalf("Write(%s): %v", v, err)
			}
			expected, err := Bytes(v)
			if err != nil {
				t.Fatalf("Bytes(%s): %v", v, err)
			}
			if !bytes.Equal(buf.Bytes(), expected) {
				t.Fatalf("Write(%s) output mismatch; got %d bytes, want %d bytes", v, buf.Len(), len(expected))
			}
		})
	}
}

// TestRegistryBytesDeterministic asserts that repeated Bytes calls
// return byte-identical results. The read path is exercised multiple
// times to detect non-deterministic generation.
func TestRegistryBytesDeterministic(t *testing.T) {
	for _, v := range []Version{VersionV1, VersionV2} {
		first := sha256.Sum256(MustBytes(v))
		second := sha256.Sum256(MustBytes(v))
		if first != second {
			t.Fatalf("Bytes(%s) is not deterministic; %x != %x", v, first, second)
		}
	}
}

// TestRegistryBytesConcurrentDeterministic asserts that concurrent
// calls produce byte-identical results. This exercises any hidden
// shared mutable state inside the embed/registry path.
func TestRegistryBytesConcurrentDeterministic(t *testing.T) {
	const goroutines = 32
	const repetitions = 4
	var wg sync.WaitGroup
	failures := make(chan error, goroutines*repetitions)
	for _, v := range []Version{VersionV1, VersionV2} {
		baseline := sha256.Sum256(MustBytes(v))
		for g := 0; g < goroutines; g++ {
			wg.Add(1)
			go func(version Version, want [32]byte) {
				defer wg.Done()
				for r := 0; r < repetitions; r++ {
					got := sha256.Sum256(MustBytes(version))
					if got != want {
						failures <- errors.New("hash mismatch")
					}
				}
			}(v, baseline)
		}
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
}

// TestRegistryWriteDoesNotMutateInput asserts that Write does not
// mutate the caller's slice by retaining the byte buffer.
func TestRegistryWriteDoesNotMutateInput(t *testing.T) {
	original, err := Bytes(VersionV1)
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	before := sha256.Sum256(original)
	var buf bytes.Buffer
	if err := Write(VersionV1, &buf); err != nil {
		t.Fatalf("Write: %v", err)
	}
	after := sha256.Sum256(original)
	if before != after {
		t.Fatalf("Bytes() returned slice was mutated by Write")
	}
}

// TestRegistryTestdataConvention asserts that the canonical schema
// files live in the same directory as the embed declaration.
func TestRegistryTestdataConvention(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	for _, name := range []string{
		"gate-summary-v1.schema.json",
		"gate-summary-v2.schema.json",
	} {
		p := filepath.Join(wd, name)
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("canonical schema missing: %s: %v", name, err)
		}
	}
}

package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"sync"
	"testing"
)

// plan_contract_descriptor_identity_concurrency_test.go contains
// the concurrent-determinism proof for the descriptor-identity
// validator. The tests run eight goroutines in parallel with a
// start barrier and assert that every goroutine receives an
// identical diagnostic stream. Each goroutine also receives an
// independent result slice — mutating one result must not affect
// a later invocation.

// concurrentDiagnosticsHash returns a deterministic hash over a
// diagnostic stream, used to compare goroutine outputs.
func concurrentDiagnosticsHash(diagnostics []PlanValidationError) string {
	return sha256HexHash(diagnostics)
}

// TestDescriptorIdentityConcurrencyDeterminism proves the
// descriptor validator is safe to call from many goroutines
// concurrently. Each goroutine receives an independent result
// slice with non-empty diagnostics whose normalised hash matches
// every other goroutine's hash. A start barrier ensures all
// goroutines begin validation simultaneously.
func TestDescriptorIdentityConcurrencyDeterminism(t *testing.T) {
	// Use duplicate-laden descriptor so each goroutine receives
	// non-empty diagnostics.
	contract := buildDuplicateFieldsDescriptor(t, []string{"alpha", "beta"})
	const goroutines = 8
	results := make([][]PlanValidationError, goroutines)
	hashes := make([]string, goroutines)

	// Start barrier: each goroutine signals readiness then waits
	// for the signal before validating.
	ready := sync.WaitGroup{}
	start := make(chan struct{})

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		i := i
		ready.Add(1)
		wg.Add(1)
		go func() {
			defer wg.Done()
			ready.Done()
			<-start // Wait for all goroutines to be ready.
			results[i] = validateDescriptorApplicabilityIdentity(contract)
			// Before hashing, require each goroutine has the expected
			// non-empty diagnostic count.
			if len(results[i]) != 2 {
				t.Errorf("goroutine %d: expected 2 diagnostics, got %d", i, len(results[i]))
				return
			}
			hashes[i] = concurrentDiagnosticsHash(results[i])
		}()
	}
	// Wait for all goroutines to be ready.
	ready.Wait()
	// Signal all goroutines to start simultaneously.
	close(start)
	wg.Wait()

	// All hashes must be identical.
	for i := 1; i < goroutines; i++ {
		if hashes[i] != hashes[0] {
			t.Fatalf("goroutine %d hash %s != goroutine 0 hash %s", i, hashes[i], hashes[0])
		}
	}

	// All result slices must be deeply equal.
	for i := 1; i < goroutines; i++ {
		if !reflect.DeepEqual(results[i], results[0]) {
			t.Fatalf("goroutine %d result differs from goroutine 0", i)
		}
	}

	// Verify expected ordered paths: /alpha, /beta.
	if results[0][0].InstancePath != "/alpha" {
		t.Fatalf("expected first diagnostic path /alpha, got %s", results[0][0].InstancePath)
	}
	if results[0][1].InstancePath != "/beta" {
		t.Fatalf("expected second diagnostic path /beta, got %s", results[0][1].InstancePath)
	}

	// Run a fresh validation after the concurrent group and require
	// the same stream and hash.
	freshResult := validateDescriptorApplicabilityIdentity(contract)
	freshHash := sha256HexHash(freshResult)
	if freshHash != hashes[0] {
		t.Fatalf("fresh validation hash %s != concurrent hash %s", freshHash, hashes[0])
	}
	if !reflect.DeepEqual(freshResult, results[0]) {
		t.Fatalf("fresh result differs from concurrent result")
	}
}

// TestDescriptorIdentityConcurrencyResultIsolation proves
// mutating one goroutine's result does not affect another
// goroutine's view of the same contract. The test uses a
// duplicate-laden descriptor so each goroutine receives a
// non-empty diagnostic stream. The test captures each
// goroutine's complete diagnostics BEFORE mutating goroutine 0;
// after the mutation goroutine 0's hash must change while every
// other goroutine's hash must remain unchanged.
func TestDescriptorIdentityConcurrencyResultIsolation(t *testing.T) {
	contract := buildDuplicateFieldsDescriptor(t, []string{"alpha", "beta"})
	const goroutines = 8
	results := make([][]PlanValidationError, goroutines)
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = validateDescriptorApplicabilityIdentity(contract)
		}()
	}
	wg.Wait()
	if len(results[0]) == 0 {
		t.Fatalf("duplicate descriptor emitted no diagnostics; cannot test mutation isolation")
	}
	// Capture each goroutine's hash and complete diagnostics BEFORE any mutation.
	preHashes := make([]string, goroutines)
	preDiags := make([][]PlanValidationError, goroutines)
	for i := 0; i < goroutines; i++ {
		preHashes[i] = sha256HexHash(results[i])
		// Deep copy the slice so we can verify isolation later.
		preDiags[i] = make([]PlanValidationError, len(results[i]))
		copy(preDiags[i], results[i])
	}
	// Mutate goroutine 0 only.
	for j := range results[0] {
		results[0][j].Message = "G0_MUTATED_" + itoa(j)
	}
	// Goroutine 0's hash must have changed.
	postHash0 := sha256HexHash(results[0])
	if postHash0 == preHashes[0] {
		t.Fatalf("goroutine 0 hash did not change after mutation: %s", postHash0)
	}
	// Every other goroutine's hash must be unchanged.
	for i := 1; i < goroutines; i++ {
		post := sha256HexHash(results[i])
		if post != preHashes[i] {
			t.Fatalf("goroutine %d hash changed: %s -> %s", i, preHashes[i], post)
		}
		if post == postHash0 {
			t.Fatalf("goroutine %d shares mutated hash with goroutine 0", i)
		}
		// Also verify complete diagnostics are unchanged before mutation.
		if !reflect.DeepEqual(results[i], preDiags[i]) {
			t.Fatalf("goroutine %d diagnostics changed unexpectedly", i)
		}
	}
}

// TestDescriptorIdentityRefreshDeterminism runs the validator
// 20 times in sequence and asserts every run produces the same
// normalised hash. Together with the goroutine test this proves
// the validator is both repeatable and concurrency-safe.
// Uses a duplicate-laden descriptor so each iteration emits
// non-empty diagnostics.
func TestDescriptorIdentityRefreshDeterminism(t *testing.T) {
	// Use duplicate-laden descriptor so every iteration produces
	// non-empty diagnostics.
	contract := buildDuplicateFieldsDescriptor(t, []string{"alpha", "beta"})
	var prevHash string
	var firstDiags []PlanValidationError
	for i := 0; i < 20; i++ {
		diags := validateDescriptorApplicabilityIdentity(contract)
		if len(diags) != 2 {
			t.Fatalf("iter %d: expected 2 diagnostics, got %d", i, len(diags))
		}
		hash := sha256HexHash(diags)
		if i == 0 {
			firstDiags = make([]PlanValidationError, len(diags))
			copy(firstDiags, diags)
		}
		if i > 0 && hash != prevHash {
			t.Fatalf("iter %d: hash drifted %s -> %s", i, prevHash, hash)
		}
		// Every iteration's diagnostics must deeply equal the first.
		if i > 0 && !reflect.DeepEqual(diags, firstDiags) {
			t.Fatalf("iter %d: diagnostics differ from first iteration", i)
		}
		prevHash = hash
	}
	// Final hash must match the frozen first-iteration hash.
	finalHash := sha256HexHash(firstDiags)
	if finalHash != prevHash {
		t.Fatalf("final hash %s != first hash %s", finalHash, prevHash)
	}
}

// sha256HexHash returns the hex-encoded SHA-256 over the
// supplied diagnostic stream. It uses the same hashing
// construction as normalizeDiagnosticsHash but is dedicated to
// concurrency helpers so the file's responsibilities stay
// separate.
func sha256HexHash(diagnostics []PlanValidationError) string {
	h := sha256.New()
	for _, d := range diagnostics {
		_, _ = h.Write([]byte(d.InstancePath))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(d.Code))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(d.PropertyName))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(d.Message))
		_, _ = h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))
}

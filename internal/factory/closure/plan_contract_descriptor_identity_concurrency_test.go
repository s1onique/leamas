package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"testing"
)

// plan_contract_descriptor_identity_concurrency_test.go contains
// the concurrent-determinism proof for the descriptor-identity
// validator. The tests run eight goroutines in parallel and
// assert that every goroutine receives an identical diagnostic
// stream. Each goroutine also receives an independent result
// slice — mutating one result must not affect a later invocation.

// concurrentDiagnosticsHash returns a deterministic hash over a
// diagnostic stream, used to compare goroutine outputs.
func concurrentDiagnosticsHash(diagnostics []PlanValidationError) string {
	return normalizeDiagnosticsHash(diagnostics)
}

// TestDescriptorIdentityConcurrencyDeterminism proves the
// descriptor validator is safe to call from many goroutines
// concurrently. Each goroutine receives an independent result
// slice whose normalised hash matches every other goroutine's
// hash.
func TestDescriptorIdentityConcurrencyDeterminism(t *testing.T) {
	contract := planContractV1()
	const goroutines = 8
	results := make([][]PlanValidationError, goroutines)
	hashes := make([]string, goroutines)
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = validateDescriptorApplicabilityIdentity(contract)
			hashes[i] = concurrentDiagnosticsHash(results[i])
		}()
	}
	wg.Wait()
	for i := 1; i < goroutines; i++ {
		if hashes[i] != hashes[0] {
			t.Fatalf("goroutine %d hash %s != goroutine 0 hash %s", i, hashes[i], hashes[0])
		}
	}
}

// TestDescriptorIdentityConcurrencyResultIsolation proves
// mutating one goroutine's result does not affect another
// goroutine's view of the same contract. The test uses a
// duplicate-laden descriptor so each goroutine receives a
// non-empty diagnostic stream. The test captures each
// goroutine's hash before mutating goroutine 0; after the
// mutation goroutine 0's hash must change while every other
// goroutine's hash must remain unchanged.
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
	// Capture each goroutine's hash BEFORE any mutation.
	preHashes := make([]string, goroutines)
	for i := 0; i < goroutines; i++ {
		preHashes[i] = sha256HexHash(results[i])
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
	}
}

// TestDescriptorIdentityConcurrentRefreshDeterminism runs the
// validator 20 times in sequence and asserts every run produces
// the same normalised hash. Together with the goroutine test
// this proves the validator is both repeatable and
// concurrency-safe.
func TestDescriptorIdentityConcurrentRefreshDeterminism(t *testing.T) {
	contract := planContractV1()
	var prevHash string
	for i := 0; i < 20; i++ {
		diags := validateDescriptorApplicabilityIdentity(contract)
		hash := sha256HexHash(diags)
		if i > 0 && hash != prevHash {
			t.Fatalf("iter %d: hash drifted %s -> %s", i, prevHash, hash)
		}
		prevHash = hash
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

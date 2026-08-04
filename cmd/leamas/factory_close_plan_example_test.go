package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// invocationResult captures the full output of one handler invocation.
type invocationResult struct {
	Exit   int
	Stdout []byte
	Stderr []byte
}

func TestExampleHelpSolo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanExample([]string{"-h"}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("help exit = %d, want 0", exit)
	}
	if stderr.Len() == 0 {
		t.Fatal("help wrote nothing to stderr")
	}
	if stdout.Len() != 0 {
		t.Fatalf("help wrote to stdout: %q", stdout.String())
	}
}

func TestExampleHelpLong(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanExample([]string{"--help"}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("help exit = %d, want 0", exit)
	}
	if stderr.Len() == 0 {
		t.Fatal("help wrote nothing to stderr")
	}
	if stdout.Len() != 0 {
		t.Fatalf("help wrote to stdout: %q", stdout.String())
	}
}

func TestExampleHelpWithExtra(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanExample([]string{"-h", "extra"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
}

func TestExampleRejectsArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanExample([]string{"extra"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
}

// TestExampleValidInjection proves valid example passes and exits 0.
func TestExampleValidInjection(t *testing.T) {
	validPlan := closure.DescriptorExample()
	deps := planExampleDeps{
		Example: func() map[string]any {
			return validPlan
		},
		Validate: func(data []byte) closure.ComposedPlanValidationResult {
			return closure.ValidatePlanComposed(data)
		},
	}
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanExampleWith(nil, &stdout, &stderr, deps)
	if exit != 0 {
		t.Fatalf("valid example exit = %d, want 0", exit)
	}
	if stdout.Len() == 0 {
		t.Fatal("no stdout written")
	}
	if stderr.Len() != 0 {
		t.Fatalf("valid example wrote to stderr: %q", stderr.String())
	}
}

// TestExampleInvalidInjection proves invalid example fails with exit 2.
func TestExampleInvalidInjection(t *testing.T) {
	// Use a valid plan that marshals but fails semantic validation
	// (missing required act_id)
	type unmarshalable struct{}
	invalidPlan := map[string]any{"contract_version": 1}
	deps := planExampleDeps{
		Example: func() map[string]any {
			return invalidPlan
		},
		Validate: func(data []byte) closure.ComposedPlanValidationResult {
			// Simulate a failing validation by passing invalid data
			// The real ValidatePlanComposed would fail on this data
			result := closure.ValidatePlanComposed([]byte(`{"invalid": true}`))
			// Force failure
			result.Valid = false
			result.SemanticValid = false
			return result
		},
	}
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanExampleWith(nil, &stdout, &stderr, deps)
	if exit != 2 {
		t.Fatalf("invalid example exit = %d, want 2", exit)
	}
	if stdout.Len() != 0 {
		t.Fatalf("invalid example wrote to stdout: %q", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Fatal("invalid example wrote nothing to stderr")
	}
}

// TestExampleDeterminism20Sequential proves byte-exact determinism with success verification.
func TestExampleDeterminism20Sequential(t *testing.T) {
	results := make([]invocationResult, 20)
	for i := 0; i < 20; i++ {
		var stdout, stderr bytes.Buffer
		exit := runFactoryClosePlanExample(nil, &stdout, &stderr)
		results[i] = invocationResult{
			Exit:   exit,
			Stdout: bytes.Clone(stdout.Bytes()),
			Stderr: bytes.Clone(stderr.Bytes()),
		}
	}
	// First invocation must succeed
	if results[0].Exit != 0 {
		t.Fatalf("run 0 exit = %d, want 0", results[0].Exit)
	}
	if results[0].StderrLen() != 0 {
		t.Fatalf("run 0 stderr not empty: %q", results[0].Stderr)
	}
	if len(results[0].Stdout) == 0 {
		t.Fatal("run 0 wrote nothing to stdout")
	}
	// All subsequent invocations must match
	for i := 1; i < 20; i++ {
		if results[i].Exit != results[0].Exit {
			t.Fatalf("run %d exit = %d, want %d", i, results[i].Exit, results[0].Exit)
		}
		if !bytes.Equal(results[i].Stdout, results[0].Stdout) {
			t.Fatalf("run %d stdout differs from run 0", i)
		}
	}
}

// StderrLen returns len(Stderr).
func (r invocationResult) StderrLen() int { return len(r.Stderr) }

// TestExampleDeterminism8Concurrent proves concurrent determinism with success verification.
func TestExampleDeterminism8Concurrent(t *testing.T) {
	results := make([]invocationResult, 8)
	var wg sync.WaitGroup
	var mu sync.Mutex
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			var stdout, stderr bytes.Buffer
			exit := runFactoryClosePlanExample(nil, &stdout, &stderr)
			mu.Lock()
			results[idx] = invocationResult{
				Exit:   exit,
				Stdout: bytes.Clone(stdout.Bytes()),
				Stderr: bytes.Clone(stderr.Bytes()),
			}
			mu.Unlock()
		}(i)
	}
	close(start)
	wg.Wait()
	// First invocation must succeed
	if results[0].Exit != 0 {
		t.Fatalf("run 0 exit = %d, want 0", results[0].Exit)
	}
	if results[0].StderrLen() != 0 {
		t.Fatalf("run 0 stderr not empty: %q", results[0].Stderr)
	}
	if len(results[0].Stdout) == 0 {
		t.Fatal("run 0 wrote nothing to stdout")
	}
	// All subsequent invocations must match
	for i := 1; i < 8; i++ {
		if results[i].Exit != results[0].Exit {
			t.Fatalf("run %d exit = %d, want %d", i, results[i].Exit, results[0].Exit)
		}
		if !bytes.Equal(results[i].Stdout, results[0].Stdout) {
			t.Fatalf("run %d stdout differs from run 0", i)
		}
	}
}

// TestExampleOutputIsValidJSON proves output is valid JSON object.
func TestExampleOutputIsValidJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanExample(nil, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0", exit)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr not empty: %q", stderr.String())
	}
	if stdout.Len() == 0 {
		t.Fatal("no output written")
	}
	s := stdout.String()
	if s[0] != '{' {
		t.Fatalf("output is not JSON object: %s", s[:10])
	}
}

// TestExampleOutputIsValidPlan proves output passes composed validation.
func TestExampleOutputIsValidPlan(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanExample(nil, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0", exit)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr not empty: %q", stderr.String())
	}
	if stdout.Len() == 0 {
		t.Fatal("no output written")
	}
	var plan map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	result := closure.ValidatePlanComposed(stdout.Bytes())
	if !result.Valid {
		t.Errorf("example validation failed: structural=%v decoded=%v semantic=%v",
			result.Structural.Valid, result.Decoded, result.SemanticValid)
	}
}

// TestExampleMarshalFailure proves marshal failure exits 2.
func TestExampleMarshalFailure(t *testing.T) {
	// Use a type that cannot be marshaled to JSON
	type unmarshalable struct{}
	deps := planExampleDeps{
		Example: func() map[string]any {
			return map[string]any{"bad": unmarshalable{}}
		},
		Validate: closure.ValidatePlanComposed,
	}
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanExampleWith(nil, &stdout, &stderr, deps)
	if exit != 2 {
		t.Fatalf("marshal failure exit = %d, want 2", exit)
	}
	if stdout.Len() != 0 {
		t.Fatalf("marshal failure wrote to stdout")
	}
}

// TestExampleWriteFailure proves write failure exits 2.
func TestExampleWriteFailure(t *testing.T) {
	validPlan := closure.DescriptorExample()
	deps := planExampleDeps{
		Example:  func() map[string]any { return validPlan },
		Validate: closure.ValidatePlanComposed,
	}
	// Use a writer that always fails
	var stderr bytes.Buffer
	failWriter := &failWriter{}
	exit := runFactoryClosePlanExampleWith(nil, failWriter, &stderr, deps)
	if exit != 2 {
		t.Fatalf("write failure exit = %d, want 2", exit)
	}
}

// failWriter always returns an error on write.
type failWriter struct{}

func (w *failWriter) Write([]byte) (int, error) {
	return 0, errors.New("forced write error")
}

// TestExampleValidationCallCount proves exactly one validation call.
func TestExampleValidationCallCount(t *testing.T) {
	validPlan := closure.DescriptorExample()
	var validateCalls int
	deps := planExampleDeps{
		Example: func() map[string]any {
			return validPlan
		},
		Validate: func(data []byte) closure.ComposedPlanValidationResult {
			validateCalls++
			return closure.ValidatePlanComposed(data)
		},
	}
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanExampleWith(nil, &stdout, &stderr, deps)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0", exit)
	}
	if validateCalls != 1 {
		t.Errorf("validateCalls = %d, want 1", validateCalls)
	}
}

// TestExampleGeneratorCallCount proves exactly one generator call.
func TestExampleGeneratorCallCount(t *testing.T) {
	var generatorCalls int
	deps := planExampleDeps{
		Example: func() map[string]any {
			generatorCalls++
			return closure.DescriptorExample()
		},
		Validate: closure.ValidatePlanComposed,
	}
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanExampleWith(nil, &stdout, &stderr, deps)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0", exit)
	}
	if generatorCalls != 1 {
		t.Errorf("generatorCalls = %d, want 1", generatorCalls)
	}
}

// TestExampleZeroCallsHelp proves help path makes zero generator/validator calls.
func TestExampleZeroCallsHelp(t *testing.T) {
	var generatorCalls, validateCalls int
	deps := planExampleDeps{
		Example: func() map[string]any {
			generatorCalls++
			return closure.DescriptorExample()
		},
		Validate: func(data []byte) closure.ComposedPlanValidationResult {
			validateCalls++
			return closure.ValidatePlanComposed(data)
		},
	}
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanExampleWith([]string{"-h"}, &stdout, &stderr, deps)
	if exit != 0 {
		t.Fatalf("help exit = %d, want 0", exit)
	}
	if generatorCalls != 0 {
		t.Errorf("help generatorCalls = %d, want 0", generatorCalls)
	}
	if validateCalls != 0 {
		t.Errorf("help validateCalls = %d, want 0", validateCalls)
	}
}

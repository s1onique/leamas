package main

import (
	"bytes"
	"sync"
	"testing"
)

func TestSchemaHelpSolo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanSchema([]string{"-h"}, &stdout, &stderr)
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

func TestSchemaHelpLong(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanSchema([]string{"--help"}, &stdout, &stderr)
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

func TestSchemaHelpWithExtra(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanSchema([]string{"-h", "extra"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
}

func TestSchemaRejectsArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanSchema([]string{"extra"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
}

// schemaInvocationResult captures the full output of one schema
// handler invocation for non-vacuous determinism checks.
type schemaInvocationResult struct {
	Exit   int
	Stdout []byte
	Stderr []byte
}

func TestSchemaDeterminism20Sequential(t *testing.T) {
	results := make([]schemaInvocationResult, 20)
	for i := 0; i < 20; i++ {
		var stdout, stderr bytes.Buffer
		exit := runFactoryClosePlanSchema(nil, &stdout, &stderr)
		results[i] = schemaInvocationResult{
			Exit:   exit,
			Stdout: bytes.Clone(stdout.Bytes()),
			Stderr: bytes.Clone(stderr.Bytes()),
		}
	}
	// Every invocation must succeed before comparing
	for i, r := range results {
		if r.Exit != 0 {
			t.Fatalf("run %d exit = %d, want 0", i, r.Exit)
		}
		if len(r.Stderr) != 0 {
			t.Fatalf("run %d stderr not empty: %q", i, r.Stderr)
		}
		if len(r.Stdout) == 0 {
			t.Fatalf("run %d wrote nothing to stdout", i)
		}
	}
	// All invocations must be byte-identical
	for i := 1; i < 20; i++ {
		if !bytes.Equal(results[i].Stdout, results[0].Stdout) {
			t.Fatalf("run %d stdout differs from run 0", i)
		}
	}
}

func TestSchemaDeterminism8Concurrent(t *testing.T) {
	results := make([]schemaInvocationResult, 8)
	var wg sync.WaitGroup
	var mu sync.Mutex
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			var stdout, stderr bytes.Buffer
			exit := runFactoryClosePlanSchema(nil, &stdout, &stderr)
			mu.Lock()
			results[idx] = schemaInvocationResult{
				Exit:   exit,
				Stdout: bytes.Clone(stdout.Bytes()),
				Stderr: bytes.Clone(stderr.Bytes()),
			}
			mu.Unlock()
		}(i)
	}
	close(start)
	wg.Wait()
	// Every invocation must succeed before comparing
	for i, r := range results {
		if r.Exit != 0 {
			t.Fatalf("concurrent run %d exit = %d, want 0", i, r.Exit)
		}
		if len(r.Stderr) != 0 {
			t.Fatalf("concurrent run %d stderr not empty: %q", i, r.Stderr)
		}
		if len(r.Stdout) == 0 {
			t.Fatalf("concurrent run %d wrote nothing to stdout", i)
		}
	}
	// All invocations must be byte-identical
	for i := 1; i < 8; i++ {
		if !bytes.Equal(results[i].Stdout, results[0].Stdout) {
			t.Fatalf("concurrent run %d stdout differs from run 0", i)
		}
	}
}

func TestSchemaOutputIsValidJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanSchema(nil, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0", exit)
	}
	if stdout.Len() == 0 {
		t.Fatal("no output written")
	}
	s := stdout.String()
	if s[0] != '{' {
		t.Fatalf("output is not JSON object: %s", s[:10])
	}
}

func TestSchemaContainsRequiredFields(t *testing.T) {
	var stdout, stderr bytes.Buffer
	runFactoryClosePlanSchema(nil, &stdout, &stderr)
	s := stdout.String()
	// Check for required JSON Schema fields
	if !bytes.Contains(stdout.Bytes(), []byte(`"$schema"`)) {
		t.Fatal("schema missing $schema")
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"$id"`)) {
		t.Fatal("schema missing $id")
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"type"`)) {
		t.Fatal("schema missing type")
	}
	_ = s // silence unused
}

func TestSchemaNoAliases(t *testing.T) {
	var stdout, stderr bytes.Buffer
	runFactoryClosePlanSchema(nil, &stdout, &stderr)
	if bytes.Contains(stdout.Bytes(), []byte(`top_level_aliases`)) {
		t.Fatal("schema contains migration aliases")
	}
	if bytes.Contains(stdout.Bytes(), []byte(`alias_subpaths`)) {
		t.Fatal("schema contains alias subpaths")
	}
}

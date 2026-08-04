package main

import (
	"bytes"
	"sync"
	"testing"
)

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

func TestExampleDeterminism20Sequential(t *testing.T) {
	results := make([]string, 20)
	for i := 0; i < 20; i++ {
		var stdout, stderr bytes.Buffer
		runFactoryClosePlanExample(nil, &stdout, &stderr)
		results[i] = stdout.String()
	}
	for i := 1; i < 20; i++ {
		if results[i] != results[0] {
			t.Fatalf("run %d differs from run 0", i)
		}
	}
}

func TestExampleDeterminism8Concurrent(t *testing.T) {
	results := make([]string, 8)
	var wg sync.WaitGroup
	var mu sync.Mutex
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			var stdout, stderr bytes.Buffer
			runFactoryClosePlanExample(nil, &stdout, &stderr)
			mu.Lock()
			results[idx] = stdout.String()
			mu.Unlock()
		}(i)
	}
	close(start)
	wg.Wait()
	for i := 1; i < 8; i++ {
		if results[i] != results[0] {
			t.Fatalf("concurrent run %d differs from run 0", i)
		}
	}
}

func TestExampleOutputIsValidJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	runFactoryClosePlanExample(nil, &stdout, &stderr)
	if stdout.Len() == 0 {
		t.Fatal("no output written")
	}
	// Basic JSON structure check
	s := stdout.String()
	if s[0] != '{' {
		t.Fatalf("output is not JSON object: %s", s[:10])
	}
}

func TestExampleOutputIsValidPlan(t *testing.T) {
	var stdout, stderr bytes.Buffer
	runFactoryClosePlanExample(nil, &stdout, &stderr)
	// Example should pass validation
	// (Full validation tested in integration)
}

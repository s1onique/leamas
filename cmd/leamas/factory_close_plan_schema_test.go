package main

import (
	"bytes"
	"testing"
)

func TestPlanSchemaDeterminism(t *testing.T) {
	var results [28]string
	for i := 0; i < 20; i++ {
		var stdout, stderr bytes.Buffer
		runFactoryClosePlanSchema(nil, &stdout, &stderr)
		results[i] = stdout.String()
	}

	for i := 20; i < 28; i++ {
		var stdout, stderr bytes.Buffer
		runFactoryClosePlanSchema(nil, &stdout, &stderr)
		results[i] = stdout.String()
	}

	for i := 1; i < 28; i++ {
		if results[i] != results[0] {
			t.Fatalf("run %d output differs from run 0", i)
		}
	}
}

func TestPlanSchemaHelpExitsZero(t *testing.T) {
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

func TestPlanSchemaRejectsArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanSchema([]string{"extra"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
}

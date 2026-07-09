package main

import (
	"strings"
	"testing"
)

// ============================================================================
// Evidence dispatcher tests
// ============================================================================

func TestWitnessEvidenceHelp(t *testing.T) {
	args := []string{"--help"}
	_, stderr, code := captureRunBundleOutput(args, runWitnessEvidence)

	if code != 0 {
		t.Errorf("help should exit 0, got %d", code)
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Errorf("help should print usage to stderr, got: %s", stderr)
	}
}

func TestWitnessEvidenceUnknownSubcommand(t *testing.T) {
	args := []string{"unknown"}
	_, stderr, code := captureRunBundleOutput(args, runWitnessEvidence)

	if code == 0 {
		t.Error("unknown subcommand should exit non-zero")
	}
	if !strings.Contains(stderr, "unknown evidence subcommand") {
		t.Errorf("stderr should mention unknown subcommand, got: %s", stderr)
	}
}

func TestWitnessEvidenceMissingSubcommand(t *testing.T) {
	args := []string{}
	_, stderr, code := captureRunBundleOutput(args, runWitnessEvidence)

	if code == 0 {
		t.Error("missing subcommand should exit non-zero")
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Errorf("should print usage for missing subcommand, got: %s", stderr)
	}
}
